package server

import (
	"context"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/hr-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/hr-server/repository"
	"github.com/ai-agent-os/ai-agent-os/core/hr-server/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// Server hr-server 服务器
type Server struct {
	// 配置
	cfg *config.HRServerConfig

	// 核心组件
	db         *gorm.DB
	httpServer *gin.Engine

	// 服务
	authService       *service.AuthService
	emailService      *service.EmailService
	jwtService        *service.JWTService
	userService       *service.UserService
	departmentService *service.DepartmentService // ⭐ 新增：部门服务
	natsService       *service.NATSService

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

	// ⭐ 初始化默认组织（根节点和未分配组织）
	if err := s.departmentService.InitDefaultDepartments(ctx); err != nil {
		return nil, fmt.Errorf("failed to init default departments: %w", err)
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

	// 配置 GORM 日志
	gormConfig := &gorm.Config{}

	// 如果启用了数据库日志
	if s.cfg.IsDBLogEnabled() {
		var logLevel gormLogger.LogLevel
		switch s.cfg.GetDBLogLevel() {
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
		slowThreshold := time.Duration(s.cfg.GetDBSlowThreshold()) * time.Millisecond
		if slowThreshold == 0 {
			slowThreshold = 200 * time.Millisecond // 默认200毫秒
		}

		// 使用 GORM 默认日志配置
		gormConfig.Logger = gormLogger.Default.LogMode(logLevel)
	} else {
		// 禁用日志
		gormConfig.Logger = gormLogger.Default.LogMode(gormLogger.Silent)
	}

	// 构建 DSN
	dsn := s.cfg.GetDatabaseDSN()
	if dsn == "" {
		return fmt.Errorf("database DSN is empty")
	}

	// 创建数据库连接
	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// 设置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// 使用配置的连接池参数，如果没有配置则使用默认值
	maxIdleConns := dbCfg.MaxIdleConns
	if maxIdleConns == 0 {
		maxIdleConns = 10
	}
	maxOpenConns := dbCfg.MaxOpenConns
	if maxOpenConns == 0 {
		maxOpenConns = 100
	}
	maxLifetime := time.Duration(dbCfg.MaxLifetime) * time.Second
	if maxLifetime == 0 {
		maxLifetime = time.Hour
	}

	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(maxLifetime)

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
	deptRepo := repository.NewDepartmentRepository(s.db) // ⭐ 新增：部门仓库

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

	// 初始化 JWT 服务
	s.jwtService = service.NewJWTService()

	// 初始化用户服务
	s.userService = service.NewUserService(userRepo, s.natsService, userSessionRepo)

	// ⭐ 新增：初始化部门服务
	s.departmentService = service.NewDepartmentService(deptRepo, userRepo)

	logger.Infof(ctx, "[Server] Services initialized successfully")
	return nil
}

// initRouter 初始化路由
func (s *Server) initRouter(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing router...")

	// 创建 Gin 引擎
	if s.cfg.IsDebug() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	s.httpServer = gin.New()

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
