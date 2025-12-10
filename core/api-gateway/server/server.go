package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	middleware2 "github.com/ai-agent-os/ai-agent-os/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// Server api-gateway 服务器
type Server struct {
	// 配置
	cfg *config.APIGatewayConfig

	// 核心组件
	httpServer      *gin.Engine
	sharedTransport *http.Transport // 共享 Transport，提高性能

	// 上下文
	ctx context.Context
}

// NewServer 创建新的服务器实例
func NewServer(cfg *config.APIGatewayConfig) (*Server, error) {
	ctx := context.Background()

	s := &Server{
		cfg: cfg,
		ctx: ctx,
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

	// 启动 HTTP 服务器
	port := fmt.Sprintf(":%d", s.cfg.GetPort())
	logger.Infof(ctx, "[Server] HTTP server starting on port %s", port)

	go func() {
		if err := s.httpServer.Run(port); err != nil {
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
				strategy := "round_robin"
				if route.LoadBalance != nil && route.LoadBalance.Strategy != "" {
					strategy = route.LoadBalance.Strategy
				}
				targetStr = fmt.Sprintf("%d targets (%s)", len(route.Targets), strategy)
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

	// 关闭 HTTP 服务器（优雅关闭）
	if s.httpServer != nil {
		// 注意：gin.Engine 没有 Shutdown 方法，需要手动实现
		// 这里先记录日志，实际关闭由 gin 的 Run 方法处理
		logger.Infof(ctx, "[Server] HTTP server shutting down...")
		// TODO: 实现真正的优雅关闭（需要将 http.Server 暴露出来）
	}

	// 关闭共享 Transport
	if s.sharedTransport != nil {
		s.sharedTransport.CloseIdleConnections()
		logger.Infof(ctx, "[Server] Shared transport closed")
	}

	logger.Infof(ctx, "[Server] Api-gateway stopped")
	return nil
}

// initRouter 初始化路由
func (s *Server) initRouter(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing router...")

	// 创建 gin 引擎
	s.httpServer = gin.New()

	// 添加中间件
	s.httpServer.Use(gin.Recovery())
	s.httpServer.Use(middleware2.Cors())
	s.httpServer.Use(middleware2.WithTraceId())
	s.httpServer.Use(middleware2.AccessLog()) // 访问日志中间件，记录所有请求（包括 agent-server）

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
	s.sharedTransport = &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}
}

// getTimeout 获取超时时间（统一处理逻辑）
func (s *Server) getTimeout(timeout int) int {
	if timeout <= 0 {
		timeout = s.cfg.Timeouts.Default
	}
	if timeout == 0 {
		timeout = 30 // 默认 30 秒
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
