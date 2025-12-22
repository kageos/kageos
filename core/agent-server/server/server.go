package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	middleware2 "github.com/ai-agent-os/ai-agent-os/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// Server agent-server 服务器
type Server struct {
	// 配置
	cfg *config.AgentServerConfig

	// 核心组件
	db         *gorm.DB
	httpServer *gin.Engine
	natsConn   *nats.Conn // NATS 连接，用于 plugin 调用

	// Repository
	agentRepo              *repository.AgentRepository
	pluginRepo             *repository.PluginRepository
	knowledgeRepo          *repository.KnowledgeRepository
	llmRepo                *repository.LLMRepository
	functionGenRepo        *repository.FunctionGenRepository
	functionGroupAgentRepo *repository.FunctionGroupAgentRepository
	sessionRepo            *repository.ChatSessionRepository
	messageRepo            *repository.ChatMessageRepository

	// 服务
	agentService       *service.AgentService
	pluginService      *service.PluginService
	knowledgeService   *service.KnowledgeService
	llmService         *service.LLMService
	agentChatService   *service.AgentChatService
	functionGenService *service.FunctionGenService

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

	if err := s.initNATS(ctx); err != nil {
		return nil, fmt.Errorf("failed to init NATS: %w", err)
	}

	// ⭐ 初始化 License Client（在 NATS 初始化之后）
	if err := s.initLicenseClient(ctx); err != nil {
		// License Client 初始化失败，记录警告但不中断启动（社区版可以继续运行）
		logger.Warnf(ctx, "[Server] Failed to init license client: %v, continuing with community edition", err)
	}

	if err := s.initServices(ctx); err != nil {
		return nil, fmt.Errorf("failed to init services: %w", err)
	}

	if err := s.initRouter(ctx); err != nil {
		return nil, fmt.Errorf("failed to init router: %w", err)
	}

	// NATS 订阅已移除，回调现在通过 HTTP 处理

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

	// 配置 GORM 日志
	gormConfig := &gorm.Config{}

	// 如果启用了数据库日志
	if dbCfg.LogLevel != "silent" {
		var logLevel gormLogger.LogLevel
		switch dbCfg.LogLevel {
		case "error":
			logLevel = gormLogger.Error
		case "warn":
			logLevel = gormLogger.Warn
		case "info":
			logLevel = gormLogger.Info
		default:
			logLevel = gormLogger.Warn
		}

		// 配置慢查询阈值
		slowThreshold := time.Duration(dbCfg.SlowThreshold) * time.Millisecond
		if slowThreshold == 0 {
			slowThreshold = 200 * time.Millisecond // 默认200毫秒
		}

		// 使用 GORM 默认日志配置
		gormConfig.Logger = gormLogger.Default.LogMode(logLevel)
	} else {
		// 禁用日志
		gormConfig.Logger = gormLogger.Default.LogMode(gormLogger.Silent)
	}

	var err error
	switch dbCfg.Type {
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			dbCfg.User, dbCfg.Password, dbCfg.Host, dbCfg.Port, dbCfg.Name)
		s.db, err = gorm.Open(mysql.Open(dsn), gormConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to MySQL: %w", err)
		}
	default:
		return fmt.Errorf("unsupported database type: %s", dbCfg.Type)
	}

	// 自动迁移表结构
	if err := model.InitTables(s.db); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	logger.Infof(ctx, "[Server] Database initialized successfully")
	return nil
}

// initNATS 初始化 NATS 连接
func (s *Server) initNATS(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing NATS connection...")

	natsHost := s.cfg.GetNatsHost()
	if natsHost == "" {
		return fmt.Errorf("NATS host is not configured")
	}

	// 构建 NATS URL（如果配置中已经包含 nats:// 前缀，则直接使用，否则添加前缀）
	natsURL := natsHost
	if !strings.HasPrefix(natsHost, "nats://") {
		natsURL = fmt.Sprintf("nats://%s", natsHost)
	}

	// 连接选项
	opts := []nats.Option{
		nats.Name("agent-server"),
		nats.Timeout(10 * time.Second),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(5),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Warnf(ctx, "[Server] NATS disconnected: %v", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Infof(ctx, "[Server] NATS reconnected to %s", nc.ConnectedUrl())
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			logger.Errorf(ctx, "[Server] NATS error: %v", err)
		}),
	}

	conn, err := nats.Connect(natsURL, opts...)
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
		natsConn, err = nats.Connect(controlCfg.GetNatsURL())
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
	s.agentRepo = repository.NewAgentRepository(s.db)
	s.pluginRepo = repository.NewPluginRepository(s.db)
	s.knowledgeRepo = repository.NewKnowledgeRepository(s.db)
	s.llmRepo = repository.NewLLMRepository(s.db)
	sessionRepo := repository.NewChatSessionRepository(s.db)
	messageRepo := repository.NewChatMessageRepository(s.db)
	s.sessionRepo = sessionRepo
	s.messageRepo = messageRepo
	s.functionGenRepo = repository.NewFunctionGenRepository(s.db)
	s.functionGroupAgentRepo = repository.NewFunctionGroupAgentRepository(s.db)

	// 初始化 Service
	s.agentService = service.NewAgentService(s.agentRepo, s.pluginRepo, s.knowledgeRepo)
	s.pluginService = service.NewPluginService(s.pluginRepo)
	s.knowledgeService = service.NewKnowledgeService(s.knowledgeRepo)
	s.llmService = service.NewLLMService(s.llmRepo)

	// 先初始化函数生成服务（因为 agentChatService 依赖它）
	s.functionGenService = service.NewFunctionGenService(s.natsConn, s.cfg, s.functionGenRepo)

	// 初始化智能体聊天服务（传入 functionGenService）
	s.agentChatService = service.NewAgentChatService(s.agentRepo, s.llmRepo, s.knowledgeRepo, s.functionGenService, sessionRepo, messageRepo, s.functionGenRepo)

	logger.Infof(ctx, "[Server] Services initialized successfully")
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

// HandleFunctionGenCallback 处理函数生成回调（HTTP 接口）
func (s *Server) HandleFunctionGenCallback(c *gin.Context, callback *dto.FunctionGenCallback) {
	ctx := contextx.ToContext(c)

	logger.Infof(ctx, "[Server] 收到回调消息 (HTTP) - RecordID: %d, MessageID: %d, Success: %v, FullGroupCodes: %v, AppCode: %s",
		callback.RecordID, callback.MessageID, callback.Success, callback.FullGroupCodes, callback.AppCode)

	// 调用 Service 层处理
	if err := s.functionGenService.ProcessFunctionGenCallback(ctx, callback); err != nil {
		logger.Errorf(ctx, "[Server] 处理回调失败: %v", err)
	}
}
