package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	middleware2 "github.com/ai-agent-os/ai-agent-os/pkg/middleware"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
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

	// 服务
	agentService     *service.AgentService
	pluginService    *service.PluginService
	knowledgeService *service.KnowledgeService
	llmService       *service.LLMService
	agentChatService *service.AgentChatService

	// 上下文
	ctx context.Context
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

	if err := s.initServices(ctx); err != nil {
		return nil, fmt.Errorf("failed to init services: %w", err)
	}

	if err := s.initRouter(ctx); err != nil {
		return nil, fmt.Errorf("failed to init router: %w", err)
	}

	// 初始化 NATS 订阅（包括回调订阅）
	if err := s.initNATSSubscriptions(ctx); err != nil {
		return nil, fmt.Errorf("failed to init NATS subscriptions: %w", err)
	}

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
	s.functionGenRepo = repository.NewFunctionGenRepository(s.db)
	s.functionGroupAgentRepo = repository.NewFunctionGroupAgentRepository(s.db)

	// 初始化 Service
	s.agentService = service.NewAgentService(s.agentRepo, s.pluginRepo, s.knowledgeRepo)
	s.pluginService = service.NewPluginService(s.pluginRepo)
	s.knowledgeService = service.NewKnowledgeService(s.knowledgeRepo)
	s.llmService = service.NewLLMService(s.llmRepo)
	s.agentChatService = service.NewAgentChatService(s.agentRepo, s.llmRepo, s.knowledgeRepo, s.natsConn, s.cfg)
	s.agentChatService.SetRepositories(sessionRepo, messageRepo, s.functionGenRepo)

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

// initNATSSubscriptions 初始化 NATS 订阅
func (s *Server) initNATSSubscriptions(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing NATS subscriptions...")

	// 订阅函数生成回调主题（app-server -> agent-server）
	callbackSubject := subjects.GetAgentServerFunctionGenCallbackSubject()
	_, err := s.natsConn.Subscribe(callbackSubject, func(msg *nats.Msg) {
		s.handleFunctionGenCallback(msg)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", callbackSubject, err)
	}
	logger.Infof(ctx, "[Server] Subscribed to callback subject: %s", callbackSubject)

	return nil
}

// handleFunctionGenCallback 处理函数生成回调消息
func (s *Server) handleFunctionGenCallback(msg *nats.Msg) {
	ctx := context.Background()

	// 从 header 获取 trace_id 和 user
	traceID := msg.Header.Get("X-Trace-Id")
	requestUser := msg.Header.Get("X-Request-User")
	if traceID != "" {
		ctx = context.WithValue(ctx, "trace_id", traceID)
	}
	if requestUser != "" {
		ctx = context.WithValue(ctx, "request_user", requestUser)
	}

	// 解析回调消息
	var callback dto.FunctionGenCallback
	if err := json.Unmarshal(msg.Data, &callback); err != nil {
		logger.Errorf(ctx, "[Server] 解析回调消息失败: %v", err)
		return
	}

	logger.Infof(ctx, "[Server] 收到回调消息 - RecordID: %d, MessageID: %d, Success: %v, FullGroupCodes: %v, AppCode: %s",
		callback.RecordID, callback.MessageID, callback.Success, callback.FullGroupCodes, callback.AppCode)

	// 1. 更新 FunctionGenRecord
	record, err := s.functionGenRepo.GetByID(callback.RecordID)
	if err != nil {
		logger.Errorf(ctx, "[Server] 获取生成记录失败: %v", err)
		return
	}

	// 更新 FullGroupCodes
	if err := record.SetFullGroupCodes(callback.FullGroupCodes); err != nil {
		logger.Errorf(ctx, "[Server] 设置 FullGroupCodes 失败: %v", err)
		return
	}

	// 更新状态
	if callback.Success {
		record.Status = model.FunctionGenStatusCompleted
	} else {
		record.Status = model.FunctionGenStatusFailed
		record.ErrorMsg = callback.Error
	}

	if err := s.functionGenRepo.Update(record); err != nil {
		logger.Errorf(ctx, "[Server] 更新生成记录失败: %v", err)
		return
	}

	// 2. 创建 FunctionGroupAgent 关联记录
	for _, fullGroupCode := range callback.FullGroupCodes {
		fga := &model.FunctionGroupAgent{
			FullGroupCode: fullGroupCode,
			AgentID:       record.AgentID,
			RecordID:     record.ID,
			AppID:        callback.AppID,
			AppCode:      callback.AppCode,
			User:         record.User,
		}
		fga.CreatedBy = record.User
		fga.UpdatedBy = record.User

		if err := s.functionGroupAgentRepo.Create(fga); err != nil {
			logger.Errorf(ctx, "[Server] 创建关联记录失败: fullGroupCode=%s, error=%v", fullGroupCode, err)
			// 继续处理其他记录，不中断
		}
	}

	logger.Infof(ctx, "[Server] 回调处理完成 - RecordID: %d, FullGroupCodesCount: %d",
		callback.RecordID, len(callback.FullGroupCodes))
}

