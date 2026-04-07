package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	v1 "github.com/ai-agent-os/ai-agent-os/core/control-service/api/v1"
	"github.com/ai-agent-os/ai-agent-os/core/control-service/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/natsx"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

// Server Control Service 服务器
type Server struct {
	// 核心组件
	natsConn      *nats.Conn
	httpServer    *gin.Engine
	subscriptions []*nats.Subscription

	// 服务
	licenseService *service.LicenseService

	// API层
	licenseAPI *v1.License

	// 配置
	cfg             *config.ControlServiceConfig
	licensePath     string
	natsURL         string
	encryptionKey   []byte // AES 加密密钥
	httpPort        int
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

	logger.Infof(ctx, "[Control Service] Creating server instance...")
	logger.Infof(ctx, "[Control Service] Configuration: Port=%d, NATS=%s, LicensePath=%s",
		cfg.GetPort(), cfg.GetNatsURL(), cfg.GetLicensePath())

	s := &Server{
		cfg:             cfg,
		licensePath:     cfg.GetLicensePath(),
		natsURL:         cfg.GetNatsURL(),
		encryptionKey:   cfg.GetEncryptionKey(),
		httpPort:        cfg.GetPort(),
		publishInterval: cfg.GetPublishInterval(),
		ctx:             ctx,
		subscriptions:   make([]*nats.Subscription, 0),
	}

	// 初始化各个组件
	logger.Infof(ctx, "[Control Service] Initializing components...")
	if err := s.initNATS(ctx); err != nil {
		return nil, fmt.Errorf("failed to init NATS: %w", err)
	}

	if err := s.initServices(ctx); err != nil {
		return nil, fmt.Errorf("failed to init services: %w", err)
	}

	if err := s.initAPI(ctx); err != nil {
		return nil, fmt.Errorf("failed to init API: %w", err)
	}

	if err := s.initRouter(ctx); err != nil {
		return nil, fmt.Errorf("failed to init router: %w", err)
	}

	logger.Infof(ctx, "[Control Service] Server instance created successfully")
	return s, nil
}

// initNATS 初始化 NATS 连接
func (s *Server) initNATS(ctx context.Context) error {
	logger.Infof(ctx, "[Control Service] Initializing NATS connection...")

	conn, err := natsx.ConnectNamed(s.natsURL, "control-service")
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

// initAPI 初始化API层
func (s *Server) initAPI(ctx context.Context) error {
	logger.Infof(ctx, "[Control Service] Initializing API layer...")

	// 初始化 License API
	s.licenseAPI = v1.NewLicense(s.licenseService)

	return nil
}

// Start 启动服务器
func (s *Server) Start(ctx context.Context) error {
	logger.Infof(ctx, "[Control Service] Starting control-service...")

	// 1. 加载 License
	logger.Infof(ctx, "[Control Service] Loading license from: %s", s.licensePath)
	if err := s.licenseService.LoadAndPublish(ctx); err != nil {
		logger.Warnf(ctx, "[Control Service] Failed to load license: %v, continuing with community edition", err)
		logger.Infof(ctx, "[Control Service] Running in community edition mode")
	} else {
		lic := s.licenseService.GetLicense()
		if lic != nil {
			logger.Infof(ctx, "[Control Service] License loaded: Edition=%s, Customer=%s, ExpiresAt=%v",
				lic.Edition, lic.Customer, lic.ExpiresAt.Time)
		}
	}

	// 2. 启动定期任务（检查过期，快过期时推送刷新指令）
	// 注意：定期任务不推送 License 内容，只检查过期和推送刷新指令
	// License 推送主要用于激活/更新后的主动通知，启动时主要靠各服务主动请求
	logger.Infof(ctx, "[Control Service] Starting periodic tasks (check interval: %d seconds)", s.publishInterval)
	s.ticker = time.NewTicker(time.Duration(s.publishInterval) * time.Second)
	go s.startPeriodicTasks(ctx)
	logger.Infof(ctx, "[Control Service] Periodic tasks started (checking license expiry)")

	// 3. 注册 NATS 订阅（请求-响应模式）
	if err := s.subscribeNATS(ctx); err != nil {
		return fmt.Errorf("failed to subscribe NATS: %w", err)
	}

	// 4. 启动 HTTP 服务器
	port := fmt.Sprintf(":%d", s.httpPort)
	logger.Infof(ctx, "[Control Service] Starting HTTP server on port %s", port)

	go func() {
		logger.Infof(ctx, "[Control Service] HTTP server listening on http://localhost%s", port)
		if err := s.httpServer.Run(port); err != nil {
			logger.Errorf(ctx, "[Control Service] HTTP server error: %v", err)
		}
	}()

	// 等待一小段时间确保 HTTP 服务器启动
	time.Sleep(100 * time.Millisecond)

	logger.Infof(ctx, "[Control Service] Control-service started successfully")
	logger.Infof(ctx, "[Control Service] API endpoints:")
	logger.Infof(ctx, "[Control Service]   - GET  http://localhost%s/control/api/v1/license/status", port)
	logger.Infof(ctx, "[Control Service]   - POST http://localhost%s/control/api/v1/license/activate", port)
	return nil
}

// startPeriodicTasks 启动定期任务
// 定期任务职责：
//  1. 检查 License 是否过期
//  2. License 快过期时（提前7天）推送刷新指令，提醒用户续费
//
// 注意：不定期推送 License 内容，推送主要用于激活/更新后的主动通知
func (s *Server) startPeriodicTasks(ctx context.Context) {
	for {
		select {
		case <-s.ticker.C:
			// 检查 License 是否过期
			if err := s.licenseService.CheckExpiry(ctx); err != nil {
				logger.Warnf(ctx, "[Control Service] License expired: %v", err)
				continue
			}

			// 检查 License 是否快过期（提前7天推送刷新指令，提醒用户续费）
			lic := s.licenseService.GetLicense()
			if lic != nil && !lic.ExpiresAt.IsZero() {
				daysUntilExpiry := time.Until(lic.ExpiresAt.Time).Hours() / 24
				if daysUntilExpiry <= 7 && daysUntilExpiry > 0 {
					// 快过期了，推送刷新指令（提醒用户续费，用户更新后会自动推送）
					if err := s.licenseService.PublishRefresh(ctx); err != nil {
						logger.Warnf(ctx, "[Control Service] Failed to publish refresh instruction: %v", err)
					} else {
						logger.Infof(ctx, "[Control Service] Published refresh instruction (License expires in %.1f days, please renew)", daysUntilExpiry)
					}
				}
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

	s.unsubscribeNATS(ctx)

	// 关闭 NATS 连接
	if s.natsConn != nil {
		s.natsConn.Close()
		logger.Infof(ctx, "[Control Service] NATS connection closed")
	}

	logger.Infof(ctx, "[Control Service] Control-service stopped")
	return nil
}

// subscribeNATS 注册所有 NATS 订阅
func (s *Server) subscribeNATS(ctx context.Context) error {
	handler := NewLicenseCommandHandler(s.licenseService)
	return RegisterNATS(ctx, s.natsConn, &s.subscriptions, handler)
}

// unsubscribeNATS 取消所有 NATS 订阅
func (s *Server) unsubscribeNATS(ctx context.Context) {
	for _, sub := range s.subscriptions {
		if sub == nil {
			continue
		}
		if err := sub.Unsubscribe(); err != nil {
			logger.Warnf(ctx, "[Control Service] Failed to unsubscribe NATS subject %s: %v", sub.Subject, err)
		}
	}
	s.subscriptions = s.subscriptions[:0]
}
