package server

import (
	"context"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/dbx"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	middleware2 "github.com/ai-agent-os/ai-agent-os/pkg/middleware"
	"github.com/ai-agent-os/ai-agent-os/pkg/natsx"
	"github.com/ai-agent-os/ai-agent-os/pkg/serverx"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

// Server agent-server 服务器
type Server struct {
	// 配置
	cfg *config.AgentServerConfig

	// 核心组件
	db         *gorm.DB
	httpServer *gin.Engine
	natsConn   *nats.Conn // NATS 连接，当前主要供 license client 使用

	// Repository
	llmRepo     *repository.LLMRepository
	sessionRepo *repository.ChatSessionRepository
	messageRepo *repository.ChatMessageRepository

	// 服务
	llmService           *service.LLMService
	toolRegistry         *service.ToolRegistry
	workspaceChatService *service.WorkspaceChatService

	// 上下文
	ctx context.Context

	// License Client
	licenseClient *license.Client
}

// NewServer 创建新的服务器实例
func NewServer(cfg *config.AgentServerConfig) (*Server, error) {
	ctx := context.Background()

	s := &Server{
		cfg: cfg,
		ctx: ctx,
	}

	// 初始化各个组件
	if err := s.initDatabase(ctx); err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}

	controlCfg := s.cfg.GetControlService()
	if controlCfg.IsEnabled() {
		if err := s.initNATS(ctx); err != nil {
			return nil, fmt.Errorf("failed to init NATS: %w", err)
		}

		// ⭐ 初始化 License Client（在 NATS 初始化之后）
		if err := s.initLicenseClient(ctx); err != nil {
			// License Client 初始化失败，记录警告但不中断启动（社区版可以继续运行）
			logger.Warnf(ctx, "[Server] Failed to init license client: %v, continuing with community edition", err)
		}
	} else {
		logger.Infof(ctx, "[Server] Control Service client disabled, skipping NATS initialization")
	}

	if err := s.initServices(ctx); err != nil {
		return nil, fmt.Errorf("failed to init services: %w", err)
	}

	if err := s.initRouter(ctx); err != nil {
		return nil, fmt.Errorf("failed to init router: %w", err)
	}

	// 当前 agent-server 不注册 NATS 订阅，回调通过 HTTP 处理。

	return s, nil
}

// Start 启动服务器
func (s *Server) Start(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Starting agent-server...")

	// 启动 HTTP 服务器
	port := fmt.Sprintf(":%d", s.cfg.GetPort())
	logger.Infof(ctx, "[Server] HTTP server starting on port %s", port)

	// 在 goroutine 中启动 HTTP 服务器
	go func() {
		if err := s.httpServer.Run(port); err != nil {
			logger.Errorf(ctx, "[Server] HTTP server error: %v", err)
		}
	}()

	// 等待一小段时间确保服务器启动
	time.Sleep(100 * time.Millisecond)

	logger.Infof(ctx, "[Server] Agent-server started successfully")
	return nil
}

// Stop 停止服务器（优雅关闭）
func (s *Server) Stop(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Stopping agent-server...")

	// 关闭数据库连接
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err == nil {
			sqlDB.Close()
			logger.Infof(ctx, "[Server] Database connection closed")
		}
	}

	// 关闭 License Client
	if s.licenseClient != nil {
		if err := s.licenseClient.Stop(ctx); err != nil {
			logger.Warnf(ctx, "[Server] Failed to stop license client: %v", err)
		} else {
			logger.Infof(ctx, "[Server] License client stopped")
		}
	}

	// 关闭 NATS 连接
	if s.natsConn != nil {
		s.natsConn.Close()
		logger.Infof(ctx, "[Server] NATS connection closed")
	}

	logger.Infof(ctx, "[Server] Agent-server stopped")
	return nil
}

// initDatabase 初始化数据库
func (s *Server) initDatabase(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing database...")

	dbCfg := s.cfg.DB
	if dbCfg.Type != "mysql" {
		return fmt.Errorf("unsupported database type: %s", dbCfg.Type)
	}
	db, err := dbx.OpenMySQL(dbCfg, dbx.OpenOptions{})
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}
	s.db = db

	// 自动迁移表结构
	if err := model.InitTables(s.db); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}
	// 初始化工作台内置 3 个模式（若不存在则插入）
	if err := model.InitWorkspaceModes(s.db); err != nil {
		return fmt.Errorf("failed to init workspace modes: %w", err)
	}

	logger.Infof(ctx, "[Server] Database initialized successfully")
	return nil
}

// initNATS 初始化 NATS 连接
func (s *Server) initNATS(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing NATS connection...")

	// 从全局配置读取 NATS URL
	globalConfig := config.GetGlobalSharedConfig()
	natsURL := globalConfig.Nats.URL
	if natsURL == "" {
		return fmt.Errorf("NATS URL is not configured in global config")
	}

	conn, err := natsx.ConnectNamed(natsURL, "agent-server")
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	s.natsConn = conn
	logger.Infof(ctx, "[Server] NATS connected successfully to %s", conn.ConnectedUrl())
	return nil
}

// initLicenseClient 初始化 License Client（通过 NATS 获取和刷新 License）
func (s *Server) initLicenseClient(ctx context.Context) error {
	// 检查是否启用 Control Service 客户端
	controlCfg := s.cfg.GetControlService()
	if !controlCfg.IsEnabled() {
		logger.Infof(ctx, "[Server] Control Service client is disabled, skipping license client initialization")
		return nil
	}

	// 检查加密密钥
	encryptionKey := controlCfg.GetEncryptionKey()
	if len(encryptionKey) != 32 {
		return fmt.Errorf("encryption key must be 32 bytes, got %d bytes", len(encryptionKey))
	}

	// 确定使用的 NATS 连接
	// 如果配置了独立的 NATS URL，需要创建新连接；否则使用现有的连接
	natsConn := s.natsConn
	if controlCfg.GetNatsURL() != "" {
		// 使用独立的 NATS 连接
		var err error
		natsConn, err = natsx.ConnectNamed(controlCfg.GetNatsURL(), "agent-server-control-client")
		if err != nil {
			return fmt.Errorf("failed to connect to Control Service NATS: %w", err)
		}
		logger.Infof(ctx, "[Server] Connected to Control Service NATS: %s", controlCfg.GetNatsURL())
	}

	// 创建 License Client
	client, err := license.NewClient(natsConn, encryptionKey, controlCfg.GetKeyPath())
	if err != nil {
		return fmt.Errorf("failed to create license client: %w", err)
	}

	// 启动 License Client
	if err := client.Start(ctx); err != nil {
		return fmt.Errorf("failed to start license client: %w", err)
	}

	s.licenseClient = client
	logger.Infof(ctx, "[Server] License client initialized successfully")
	return nil
}

// initServices 初始化所有业务服务
func (s *Server) initServices(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing services...")

	// 初始化 Repository
	s.llmRepo = repository.NewLLMRepository(s.db)
	sessionRepo := repository.NewChatSessionRepository(s.db)
	messageRepo := repository.NewChatMessageRepository(s.db)
	workspaceEventRepo := repository.NewWorkspaceEventRepository(s.db)
	s.sessionRepo = sessionRepo
	s.messageRepo = messageRepo

	// 初始化 Service
	s.llmService = service.NewLLMService(s.llmRepo)

	// 智能工作台 ToolRegistry、WorkspaceChatService（只认 LLM，单模式；已移除插件）
	s.toolRegistry = service.NewToolRegistry(workspaceEventRepo)
	s.workspaceChatService = service.NewWorkspaceChatService(s.toolRegistry, sessionRepo, messageRepo, s.llmRepo)

	logger.Infof(ctx, "[Server] Services initialized successfully")
	return nil
}

// initRouter 初始化路由
func (s *Server) initRouter(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing router...")

	// 创建 gin 引擎并挂载通用中间件
	s.httpServer = serverx.NewGin(
		serverx.WithRecovery(),
		serverx.WithMiddleware(middleware2.Cors()),
	)
	// 注意：用户信息中间件在路由组中添加

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
		"service":   "agent-server",
	})
}

// GetDB 获取数据库连接
func (s *Server) GetDB() *gorm.DB {
	return s.db
}
