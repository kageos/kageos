package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/kageos/kageos/pkg/logger"
	middleware2 "github.com/kageos/kageos/pkg/middleware"
	"github.com/kageos/kageos/pkg/natsx"
	"github.com/kageos/kageos/pkg/serverx"
	"github.com/nats-io/nats.go"
)

// Server api-gateway 服务器
type Server struct {
	// 配置
	cfg *config.APIGatewayConfig

	// 核心组件
	httpServer              *gin.Engine
	httpRuntime             *serverx.HTTPServer
	sharedTransport         *http.Transport // 共享 Transport，提高性能
	tokenBlacklist          *TokenBlacklist // ⭐ 新增：Token 黑名单管理器
	workspaceActionVerifier *controlauth.Verifier
	agentBackendSigner      *controlauth.Signer
	agentDelegationVerifier *controlauth.Verifier
	delegatedBackendSigner  *controlauth.Signer
	agentTimerVerifier      *controlauth.Verifier
	timerBackendSigner      *controlauth.Signer
	tokenCommandVerifier    *controlauth.Verifier
	natsConn                *nats.Conn
	subscriptions           []*nats.Subscription

	// 上下文
	ctx context.Context
}

// NewServer 创建新的服务器实例
func NewServer(cfg *config.APIGatewayConfig) (*Server, error) {
	ctx := context.Background()
	controlPlaneSecret, err := config.GetControlPlaneSecret()
	if err != nil {
		return nil, fmt.Errorf("load control-plane secret: %w", err)
	}
	workspaceActionVerifier, err := controlauth.NewVerifier(
		controlPlaneSecret,
		controlauth.HTTPWorkspaceActionScope,
		controlauth.VerifierOptions{
			MaxAge:        internalHTTPMaxAge,
			MaxFutureSkew: internalHTTPMaxFutureSkew,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("initialize workspace-action request verifier: %w", err)
	}
	agentBackendSigner, err := controlauth.NewSigner(
		controlPlaneSecret,
		controlauth.HTTPGatewayAgentBackendScope,
	)
	if err != nil {
		return nil, fmt.Errorf("initialize agent backend request signer: %w", err)
	}
	agentDelegationVerifier, err := controlauth.NewVerifier(
		controlPlaneSecret,
		controlauth.HTTPAgentDelegatedAPIScope,
		controlauth.VerifierOptions{MaxAge: internalHTTPMaxAge, MaxFutureSkew: internalHTTPMaxFutureSkew},
	)
	if err != nil {
		return nil, fmt.Errorf("initialize Agent delegated API verifier: %w", err)
	}
	delegatedBackendSigner, err := controlauth.NewSigner(
		controlPlaneSecret,
		controlauth.HTTPGatewayDelegatedBackendScope,
	)
	if err != nil {
		return nil, fmt.Errorf("initialize delegated backend request signer: %w", err)
	}
	agentTimerVerifier, err := controlauth.NewVerifier(
		controlPlaneSecret,
		controlauth.HTTPAgentDelegatedTimerScope,
		controlauth.VerifierOptions{MaxAge: internalHTTPMaxAge, MaxFutureSkew: internalHTTPMaxFutureSkew},
	)
	if err != nil {
		return nil, fmt.Errorf("initialize Agent delegated timer verifier: %w", err)
	}
	timerBackendSigner, err := controlauth.NewSigner(controlPlaneSecret, controlauth.HTTPGatewayTimerBackendScope)
	if err != nil {
		return nil, fmt.Errorf("initialize timer backend request signer: %w", err)
	}
	tokenCommandVerifier, err := controlauth.NewVerifier(
		controlPlaneSecret,
		controlauth.NATSGatewayTokenCommandScope,
		controlauth.VerifierOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("initialize Gateway token command verifier: %w", err)
	}

	s := &Server{
		cfg:                     cfg,
		ctx:                     ctx,
		tokenBlacklist:          NewTokenBlacklist(), // ⭐ 新增：初始化 Token 黑名单管理器
		workspaceActionVerifier: workspaceActionVerifier,
		agentBackendSigner:      agentBackendSigner,
		agentDelegationVerifier: agentDelegationVerifier,
		delegatedBackendSigner:  delegatedBackendSigner,
		agentTimerVerifier:      agentTimerVerifier,
		timerBackendSigner:      timerBackendSigner,
		tokenCommandVerifier:    tokenCommandVerifier,
		subscriptions:           make([]*nats.Subscription, 0),
	}

	// 初始化共享 Transport
	s.initSharedTransport()

	// 验证配置
	if err := s.validateConfig(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// 初始化路由
	if err := s.initRouter(ctx); err != nil {
		return nil, fmt.Errorf("failed to init router: %w", err)
	}

	return s, nil
}

// Start 启动服务器
func (s *Server) Start(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Starting api-gateway...")

	// 打印代理配置信息
	s.printProxyRoutes(ctx)

	// 启动 NATS 连接和订阅。默认要求 NATS 可用，避免 token 失效链路静默漂移。
	if err := s.initNATS(ctx); err != nil {
		if s.cfg.AllowNATSDegradedStartup() {
			logger.Warnf(ctx, "[Server] Failed to init NATS: %v, allow_nats_degraded_startup=true so HTTP gateway will continue", err)
		} else {
			return fmt.Errorf("failed to init required NATS: %w", err)
		}
	} else if err := s.subscribeNATS(ctx); err != nil {
		s.unsubscribeNATS(ctx)
		s.closeNATS(ctx)
		if s.cfg.AllowNATSDegradedStartup() {
			logger.Warnf(ctx, "[Server] Failed to subscribe NATS: %v, allow_nats_degraded_startup=true so HTTP gateway will continue", err)
		} else {
			return fmt.Errorf("failed to subscribe required NATS: %w", err)
		}
	}

	// 启动 HTTP 服务器
	addr := net.JoinHostPort(s.cfg.GetListenHost(), strconv.Itoa(s.cfg.GetPort()))
	logger.Infof(ctx, "[Server] HTTP server starting on %s", addr)

	httpRuntime, err := serverx.StartHTTPServer(ctx, addr, s.httpServer)
	if err != nil {
		s.unsubscribeNATS(ctx)
		s.closeNATS(ctx)
		return fmt.Errorf("failed to start HTTP server on %s: %w", addr, err)
	}
	s.httpRuntime = httpRuntime
	go func() {
		if err := <-httpRuntime.Err(); err != nil {
			logger.Errorf(ctx, "[Server] HTTP server error: %v", err)
		}
	}()

	logger.Infof(ctx, "[Server] Api-gateway started successfully")
	return nil
}

// printProxyRoutes 打印所有代理路由信息
func (s *Server) printProxyRoutes(ctx context.Context) {
	logger.Infof(ctx, "═══════════════════════════════════════════════════════════")
	logger.Infof(ctx, "🚀 API Gateway Proxy Routes")
	logger.Infof(ctx, "═══════════════════════════════════════════════════════════")

	cfg := s.cfg
	if len(cfg.Routes) == 0 {
		logger.Warnf(ctx, "  ⚠️  No routes configured in config file")
	} else {
		for i, route := range cfg.Routes {
			timeout := s.getTimeout(route.Timeout)
			// 显示目标信息
			targetStr := ""
			if len(route.Targets) == 0 {
				targetStr = "no targets"
			} else if len(route.Targets) == 1 {
				targetStr = route.Targets[0].URL
			} else {
				targetStr = fmt.Sprintf("%d targets (load balancing not implemented; using first target)", len(route.Targets))
			}
			logger.Infof(ctx, "  [%d] %-25s -> %-35s (timeout: %ds)",
				i+1, route.Path+"/*", targetStr, timeout)
		}
	}

	logger.Infof(ctx, "═══════════════════════════════════════════════════════════")
	logger.Infof(ctx, "  Gateway URL: http://localhost:%d", cfg.GetPort())
	logger.Infof(ctx, "═══════════════════════════════════════════════════════════")
}

// Stop 停止服务器（优雅关闭）
func (s *Server) Stop(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Stopping api-gateway...")
	var stopErr error

	// 关闭 HTTP 服务器（优雅关闭）
	if s.httpRuntime != nil {
		if err := s.httpRuntime.Shutdown(ctx); err != nil {
			logger.Warnf(ctx, "[Server] HTTP server shutdown failed: %v", err)
			stopErr = err
		} else {
			logger.Infof(ctx, "[Server] HTTP server stopped")
		}
		s.httpRuntime = nil
	}

	// 关闭共享 Transport
	if s.sharedTransport != nil {
		s.sharedTransport.CloseIdleConnections()
		logger.Infof(ctx, "[Server] Shared transport closed")
	}

	s.unsubscribeNATS(ctx)
	s.closeNATS(ctx)

	logger.Infof(ctx, "[Server] Api-gateway stopped")
	return stopErr
}

// initNATS 初始化 NATS 连接
func (s *Server) initNATS(ctx context.Context) error {
	globalConfig := config.GetGlobalSharedConfig()
	natsURL := globalConfig.Nats.URL
	if natsURL == "" {
		natsURL = "nats://127.0.0.1:4222"
	}

	conn, err := natsx.ConnectNamed(natsURL, "api-gateway")
	if err != nil {
		return fmt.Errorf("connect NATS: %w", err)
	}

	s.natsConn = conn
	logger.Infof(ctx, "[Server] NATS connected: %s", conn.ConnectedUrl())
	return nil
}

// subscribeNATS 注册所有 NATS 订阅
func (s *Server) subscribeNATS(ctx context.Context) error {
	if s.natsConn == nil {
		return nil
	}

	tokenHandler := NewTokenCommandHandler(s.tokenBlacklist, s.tokenCommandVerifier)
	return RegisterNATS(ctx, s.natsConn, &s.subscriptions, tokenHandler)
}

// unsubscribeNATS 取消所有 NATS 订阅
func (s *Server) unsubscribeNATS(ctx context.Context) {
	for _, sub := range s.subscriptions {
		if sub == nil {
			continue
		}
		if err := sub.Unsubscribe(); err != nil {
			logger.Warnf(ctx, "[Server] Failed to unsubscribe NATS subject %s: %v", sub.Subject, err)
		}
	}
	s.subscriptions = s.subscriptions[:0]
}

// closeNATS 关闭 NATS 连接
func (s *Server) closeNATS(ctx context.Context) {
	if s.natsConn == nil {
		return
	}
	s.natsConn.Close()
	s.natsConn = nil
	logger.Infof(ctx, "[Server] NATS connection closed")
}

// initRouter 初始化路由
func (s *Server) initRouter(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing router...")

	// 添加中间件：自定义 Recovery。使用 nil writer 避免 Gin 先往 stderr 打堆栈，
	// 再由我们统一处理：ErrAbortHandler（客户端断开/代理中止）只打 Debug，其它 panic 打 Error+堆栈。
	customRecovery := gin.RecoveryWithWriter(nil, func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(error); ok && (errors.Is(err, http.ErrAbortHandler) || err == http.ErrAbortHandler) {
			logger.Debugf(ctx, "[Recovery] Client connection closed (ErrAbortHandler), path: %s", c.Request.URL.Path)
			c.Abort()
			return
		}
		// 与 Gin 一致：broken pipe / connection reset 不打堆栈
		var brokenPipe bool
		if ne, ok := recovered.(*net.OpError); ok {
			var se *os.SyscallError
			if errors.As(ne, &se) {
				low := strings.ToLower(se.Error())
				if strings.Contains(low, "broken pipe") || strings.Contains(low, "connection reset by peer") {
					brokenPipe = true
				}
			}
		}
		if brokenPipe {
			logger.Debugf(ctx, "[Recovery] Broken connection: %v, path: %s", recovered, c.Request.URL.Path)
			c.Abort()
			return
		}
		logger.Errorf(ctx, "[Recovery] panic recovered: %v\n%s", recovered, debug.Stack())
		c.AbortWithStatus(http.StatusInternalServerError)
	})

	s.httpServer = serverx.NewGin(
		serverx.WithMiddleware(customRecovery),
		serverx.WithMiddleware(middleware2.Cors(), middleware2.WithTraceId(), middleware2.AccessLog()),
		serverx.WithRegisteredMiddlewares(serverx.ServiceAPIGateway),
	) // 访问日志中间件，记录所有请求（包括 agent-server）

	// 设置路由
	s.setupRoutes()

	logger.Infof(ctx, "[Server] Router initialized successfully")
	return nil
}

// healthHandler 健康检查处理器
func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Format(time.DateTime),
		"service":   "api-gateway",
	})
}

// initSharedTransport 初始化共享 Transport（提高性能）
func (s *Server) initSharedTransport() {
	// 获取默认超时时间
	defaultTimeout := time.Duration(s.cfg.Timeouts.Default) * time.Second
	if defaultTimeout == 0 {
		defaultTimeout = 300 * time.Second // 默认 300 秒（5分钟）
	}

	// ⭐ 对于 SSE 流式接口，ResponseHeaderTimeout 需要更长
	// 但为了性能，我们使用共享 Transport，超时由 Context 控制
	// ResponseHeaderTimeout 设置为 30 分钟，足够支持长时间流式响应
	streamingTimeout := 30 * time.Minute
	if defaultTimeout > streamingTimeout {
		streamingTimeout = defaultTimeout
	}

	s.sharedTransport = &http.Transport{
		MaxIdleConns:          200, // ✅ 优化：增加到 200，提高并发处理能力
		MaxIdleConnsPerHost:   50,  // ✅ 优化：增加到 50，支持更高并发
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: streamingTimeout, // ✅ 优化：设置响应头超时为 30 分钟，支持 SSE 流式响应
		ExpectContinueTimeout: 1 * time.Second,  // ✅ 优化：设置 Expect 100-continue 超时
		TLSHandshakeTimeout:   10 * time.Second, // ✅ 优化：设置 TLS 握手超时
	}
}

// getTimeout 获取超时时间（统一处理逻辑）
func (s *Server) getTimeout(timeout int) int {
	if timeout <= 0 {
		timeout = s.cfg.Timeouts.Default
	}
	if timeout == 0 {
		timeout = 300 // 默认 300 秒（5分钟）
	}
	return timeout
}

// validateConfig 验证配置
func (s *Server) validateConfig() error {
	serviceNames := make(map[string]bool)

	for i, route := range s.cfg.Routes {
		// 验证 service_name 重复
		if route.ServiceName != "" {
			if serviceNames[route.ServiceName] {
				return fmt.Errorf("duplicate service_name '%s' in route[%d] (path: %s)", route.ServiceName, i, route.Path)
			}
			serviceNames[route.ServiceName] = true
		}

		// 验证 URL 格式
		for j, target := range route.Targets {
			if target.URL == "" {
				continue
			}
			if _, err := url.Parse(target.URL); err != nil {
				return fmt.Errorf("invalid target URL in route[%d] target[%d]: %s, error: %v", i, j, target.URL, err)
			}
		}
	}

	return nil
}
