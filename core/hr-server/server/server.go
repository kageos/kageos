package server

import (
	"context"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/hr-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/hr-server/repository"
	"github.com/ai-agent-os/ai-agent-os/core/hr-server/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/dbx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/serverx"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Server hr-server 服务器
type Server struct {
	// 配置
	cfg *config.HRServerConfig

	// 核心组件
	db         *gorm.DB
	httpServer *gin.Engine

	// 服务
	authService            *service.AuthService
	emailService           *service.EmailService
	userService            *service.UserService
	departmentService      *service.DepartmentService
	natsService            *service.NATSService
	messageConsumerService *service.MessageConsumerService

	// 上下文
	ctx context.Context
}

// NewServer 创建新的服务器实例
func NewServer(cfg *config.HRServerConfig) (*Server, error) {
	ctx := context.Background()

	s := &Server{
		cfg: cfg,
		ctx: ctx,
	}

	// 初始化各个组件
	if err := s.initDatabase(ctx); err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}

	if err := s.initServices(ctx); err != nil {
		return nil, fmt.Errorf("failed to init services: %w", err)
	}

	// ⭐ 初始化默认组织（根节点、未分配组织、虚拟组织/测试组）；须在默认用户之前，以便 test_user 归属 /org/virtual/test
	if err := s.departmentService.InitDefaultDepartments(ctx); err != nil {
		return nil, fmt.Errorf("failed to init default departments: %w", err)
	}

	// ⭐ 初始化默认用户：system + test_user（test_user 归属 /org/virtual/test，密码与 system 共用）
	if err := service.InitDefaultUsers(ctx, s.db); err != nil {
		logger.Warnf(ctx, "[Server] 初始化默认用户失败: %v", err)
		// 不中断启动，记录警告即可
	}

	if err := s.initRouter(ctx); err != nil {
		return nil, fmt.Errorf("failed to init router: %w", err)
	}

	return s, nil
}

// Start 启动服务器
func (s *Server) Start(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Starting hr-server...")

	// 启动 HTTP 服务器
	port := fmt.Sprintf(":%d", s.cfg.GetPort())
	logger.Infof(ctx, "[Server] HTTP server starting on port %s", port)

	go func() {
		if err := s.httpServer.Run(port); err != nil {
			logger.Errorf(ctx, "[Server] HTTP server error: %v", err)
		}
	}()

	if s.messageConsumerService != nil {
		if err := s.messageConsumerService.Start(ctx); err != nil {
			logger.Warnf(ctx, "[Server] Message consumer start failed: %v", err)
		}
	}

	logger.Infof(ctx, "[Server] HR-server started successfully")
	return nil
}

// Stop 停止服务器（优雅关闭）
func (s *Server) Stop(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Stopping hr-server...")

	// 关闭数据库连接
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err == nil {
			sqlDB.Close()
			logger.Infof(ctx, "[Server] Database connection closed")
		}
	}

	// 关闭 NATS 连接
	if s.natsService != nil {
		if err := s.natsService.Close(); err != nil {
			logger.Warnf(ctx, "[Server] Failed to close NATS connection: %v", err)
		} else {
			logger.Infof(ctx, "[Server] NATS connection closed")
		}
	}

	logger.Infof(ctx, "[Server] HR-server stopped")
	return nil
}

// initDatabase 初始化数据库
func (s *Server) initDatabase(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing database...")

	dbCfg := s.cfg.GetDB()
	db, err := dbx.OpenMySQL(dbCfg, dbx.OpenOptions{
		DefaultMaxLifetime: time.Hour,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	s.db = db

	// ⭐ 执行数据库迁移（自动创建/更新表结构）
	if err := model.InitModels(db); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	logger.Infof(ctx, "[Server] Database initialized successfully")
	return nil
}

// initServices 初始化所有业务服务
func (s *Server) initServices(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing services...")

	// 初始化仓库
	userRepo := repository.NewUserRepository(s.db)
	userSessionRepo := repository.NewUserSessionRepository(s.db)
	emailCodeRepo := repository.NewEmailCodeRepository(s.db)
	deptRepo := repository.NewDepartmentRepository(s.db)

	// 初始化 NATS 服务
	natsService, err := service.NewNATSService()
	if err != nil {
		logger.Warnf(ctx, "[Server] Failed to initialize NATS service: %v, continuing without NATS", err)
		// 不返回错误，允许服务在没有 NATS 的情况下运行（向后兼容）
	} else {
		s.natsService = natsService
		logger.Infof(ctx, "[Server] NATS service initialized successfully")
	}

	// 初始化认证服务
	s.authService = service.NewAuthService(userRepo, userSessionRepo, s.natsService)

	// 初始化邮件服务
	s.emailService = service.NewEmailService(emailCodeRepo)

	// 初始化用户服务
	s.userService = service.NewUserService(userRepo, s.natsService, userSessionRepo)

	s.departmentService = service.NewDepartmentService(deptRepo, userRepo)

	// 消息消费服务（订阅 NATS 发邮件，仅当 NATS 可用时）
	if s.natsService != nil {
		s.messageConsumerService = service.NewMessageConsumerService(s.natsService, s.emailService, s.userService)
	}

	logger.Infof(ctx, "[Server] Services initialized successfully")
	return nil
}

// initRouter 初始化路由
func (s *Server) initRouter(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing router...")

	// 创建 Gin 引擎
	s.httpServer = serverx.NewGin(serverx.WithDebug(s.cfg.IsDebug()))

	// 设置路由
	s.setupRoutes()

	logger.Infof(ctx, "[Server] Router initialized successfully")
	return nil
}

// healthHandler 健康检查处理器
func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "ok",
		"service": "hr-server",
	})
}
