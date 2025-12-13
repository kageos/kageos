package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/control-service/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

// Server Control Service 服务器
type Server struct {
	// 核心组件
	natsConn   *nats.Conn
	httpServer *gin.Engine

	// 服务
	licenseService *service.LicenseService

	// 配置
	cfg            *config.ControlServiceConfig
	licensePath    string
	natsURL        string
	encryptionKey  []byte // AES 加密密钥
	httpPort       int
	publishInterval int // 发布间隔（秒）

	// 定期任务
	ticker *time.Ticker
	mu     sync.RWMutex

	// 上下文
	ctx context.Context
}

// NewServer 创建 Control Service 服务器
func NewServer(cfg *config.ControlServiceConfig) (*Server, error) {
	ctx := context.Background()

	s := &Server{
		cfg:             cfg,
		licensePath:     cfg.GetLicensePath(),
		natsURL:         cfg.GetNatsURL(),
		encryptionKey:   cfg.GetEncryptionKey(),
		httpPort:        cfg.GetPort(),
		publishInterval: cfg.GetPublishInterval(),
		ctx:             ctx,
	}

	// 初始化各个组件
	if err := s.initNATS(ctx); err != nil {
		return nil, fmt.Errorf("failed to init NATS: %w", err)
	}

	if err := s.initServices(ctx); err != nil {
		return nil, fmt.Errorf("failed to init services: %w", err)
	}

	if err := s.initRouter(ctx); err != nil {
		return nil, fmt.Errorf("failed to init router: %w", err)
	}

	return s, nil
}

// initNATS 初始化 NATS 连接
func (s *Server) initNATS(ctx context.Context) error {
	logger.Infof(ctx, "[Control Service] Initializing NATS connection...")

	conn, err := nats.Connect(s.natsURL,
		nats.Name("control-service"),
		nats.Timeout(10*time.Second),
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(5),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Warnf(ctx, "[Control Service] NATS disconnected: %v", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Infof(ctx, "[Control Service] NATS reconnected to %s", nc.ConnectedUrl())
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	s.natsConn = conn
	logger.Infof(ctx, "[Control Service] NATS connected successfully to %s", conn.ConnectedUrl())
	return nil
}

// initServices 初始化服务
func (s *Server) initServices(ctx context.Context) error {
	logger.Infof(ctx, "[Control Service] Initializing services...")

	// 初始化 License 服务
	s.licenseService = service.NewLicenseService(s.natsConn, s.licensePath, s.encryptionKey)

	return nil
}

// initRouter 初始化路由
func (s *Server) initRouter(ctx context.Context) error {
	logger.Infof(ctx, "[Control Service] Initializing router...")

	gin.SetMode(gin.ReleaseMode)
	s.httpServer = gin.New()
	s.httpServer.Use(gin.Recovery())
	s.httpServer.Use(gin.Logger())

	// 注册 API 路由
	api := s.httpServer.Group("/api/v1")
	{
		// License 相关
		api.GET("/license/status", s.getLicenseStatus)
	}

	return nil
}

// Start 启动服务器
func (s *Server) Start(ctx context.Context) error {
	logger.Infof(ctx, "[Control Service] Starting control-service...")

	// 1. 加载并发布 License 密钥
	if err := s.licenseService.LoadAndPublish(ctx); err != nil {
		logger.Warnf(ctx, "[Control Service] Failed to load license: %v, continuing with community edition", err)
	}

	// 2. 启动定期任务（定期发布 License 密钥）
	s.ticker = time.NewTicker(time.Duration(s.publishInterval) * time.Second)
	go s.startPeriodicTasks(ctx)

	// 3. 启动 HTTP 服务器
	port := fmt.Sprintf(":%d", s.httpPort)
	logger.Infof(ctx, "[Control Service] HTTP server starting on port %s", port)

	go func() {
		if err := s.httpServer.Run(port); err != nil {
			logger.Errorf(ctx, "[Control Service] HTTP server error: %v", err)
		}
	}()

	logger.Infof(ctx, "[Control Service] Control-service started successfully")
	return nil
}

// startPeriodicTasks 启动定期任务
func (s *Server) startPeriodicTasks(ctx context.Context) {
	for {
		select {
		case <-s.ticker.C:
			// 定期发布 License 密钥（确保新实例能获取）
			if err := s.licenseService.PublishKey(ctx); err != nil {
				logger.Warnf(ctx, "[Control Service] Failed to publish license key: %v", err)
			}

			// 检查 License 是否过期
			if err := s.licenseService.CheckExpiry(ctx); err != nil {
				logger.Warnf(ctx, "[Control Service] License expired: %v", err)
			}

		case <-ctx.Done():
			return
		}
	}
}

// Stop 停止服务器（优雅关闭）
func (s *Server) Stop(ctx context.Context) error {
	logger.Infof(ctx, "[Control Service] Stopping control-service...")

	// 停止定期任务
	if s.ticker != nil {
		s.ticker.Stop()
	}

	// 关闭 NATS 连接
	if s.natsConn != nil {
		s.natsConn.Close()
		logger.Infof(ctx, "[Control Service] NATS connection closed")
	}

	logger.Infof(ctx, "[Control Service] Control-service stopped")
	return nil
}

