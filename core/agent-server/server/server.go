package server

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/core/agent-server/repository"
	"github.com/kageos/kageos/core/agent-server/service"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/dbx"
	"github.com/kageos/kageos/pkg/logger"
	middleware2 "github.com/kageos/kageos/pkg/middleware"
	"github.com/kageos/kageos/pkg/natsx"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/kageos/kageos/pkg/serverx"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

// Server agent-server 服务器
type Server struct {
	// 配置
	cfg *config.AgentServerConfig

	// 核心组件
	db          *gorm.DB
	httpServer  *gin.Engine
	httpRuntime *serverx.HTTPServer
	natsConn    *nats.Conn

	// Repository
	llmRepo     *repository.LLMRepository
	sessionRepo *repository.ChatSessionRepository
	messageRepo *repository.ChatMessageRepository

	// 服务
	llmService           *service.LLMService
	toolRegistry         *service.ToolRegistry
	runtimeStateStore    service.RuntimeStateStore
	workspaceChatService *service.WorkspaceChatService
	scheduledAgentWorker *scheduledsdk.Worker

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

	return s, nil
}

// Start 启动服务器
func (s *Server) Start(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Starting agent-server...")

	if s.scheduledAgentWorker != nil {
		if err := s.scheduledAgentWorker.Start(ctx); err != nil {
			return fmt.Errorf("failed to start scheduled agent worker: %w", err)
		}
		logger.Infof(ctx, "[Server] Scheduled agent session worker started")
	}

	// 启动 HTTP 服务器
	addr := net.JoinHostPort(s.cfg.GetListenHost(), strconv.Itoa(s.cfg.GetPort()))
	logger.Infof(ctx, "[Server] HTTP server starting on %s", addr)

	httpRuntime, err := serverx.StartHTTPServer(ctx, addr, s.httpServer)
	if err != nil {
		if s.scheduledAgentWorker != nil {
			_ = s.scheduledAgentWorker.Stop()
		}
		return fmt.Errorf("failed to start HTTP server on %s: %w", addr, err)
	}
	s.httpRuntime = httpRuntime
	go func() {
		if err := <-httpRuntime.Err(); err != nil {
			logger.Errorf(ctx, "[Server] HTTP server error: %v", err)
		}
	}()

	logger.Infof(ctx, "[Server] Agent-server started successfully")
	return nil
}

// Stop 停止服务器（优雅关闭）
func (s *Server) Stop(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Stopping agent-server...")
	var stopErr error

	if s.httpRuntime != nil {
		if err := s.httpRuntime.Shutdown(ctx); err != nil {
			logger.Warnf(ctx, "[Server] HTTP server shutdown failed: %v", err)
			stopErr = err
		} else {
			logger.Infof(ctx, "[Server] HTTP server stopped")
		}
		s.httpRuntime = nil
	}

	// 先停止定时任务 worker，避免退订过程中继续接新执行。
	if s.scheduledAgentWorker != nil {
		if err := s.scheduledAgentWorker.Stop(); err != nil {
			logger.Warnf(ctx, "[Server] Scheduled agent session worker stop failed: %v", err)
		} else {
			logger.Infof(ctx, "[Server] Scheduled agent session worker stopped")
		}
	}

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
	return stopErr
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

// initServices 初始化所有业务服务
func (s *Server) initServices(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing services...")

	// 初始化 Repository
	s.llmRepo = repository.NewLLMRepository(s.db)
	sessionRepo := repository.NewChatSessionRepository(s.db)
	messageRepo := repository.NewChatMessageRepository(s.db)
	s.sessionRepo = sessionRepo
	s.messageRepo = messageRepo

	// 初始化 Service
	s.llmService = service.NewLLMService(s.llmRepo)
	if err := s.llmService.InitLLMSeeds(ctx, s.cfg.LLMs); err != nil {
		return fmt.Errorf("failed to init LLM seeds: %w", err)
	}
	s.runtimeStateStore = service.NewInMemoryRuntimeStateStore()

	// 智能工作台 ToolRegistry、WorkspaceChatService（只认 LLM，单模式；已移除插件）
	s.toolRegistry = service.NewToolRegistry(service.WithToolMessagePublisher(s.natsConn))
	s.workspaceChatService = service.NewWorkspaceChatService(s.toolRegistry, sessionRepo, messageRepo, s.llmRepo, s.runtimeStateStore)
	scheduledAgentWorker, err := service.NewScheduledAgentSessionWorker(s.natsConn, s.workspaceChatService)
	if err != nil {
		return fmt.Errorf("failed to init scheduled agent worker: %w", err)
	}
	s.scheduledAgentWorker = scheduledAgentWorker

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
		serverx.WithRegisteredMiddlewares(serverx.ServiceAgentServer),
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
