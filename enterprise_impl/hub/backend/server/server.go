package server

import (
	"context"
	"fmt"
	"time"

	"github.com/ai-agent-os/hub/backend/model"
	"github.com/ai-agent-os/hub/backend/repository"
	"github.com/ai-agent-os/hub/backend/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// Server Hub 服务器
type Server struct {
	// 配置
	cfg *config.HubConfig

	// 核心组件
	db         *gorm.DB
	httpServer *gin.Engine

	// 服务
	hubDirectoryService *service.HubDirectoryService
	// authService   *service.AuthService
	// paymentService *service.PaymentService

	// 上下文
	ctx context.Context
}

// NewServer 创建新的服务器实例
func NewServer(cfg *config.HubConfig) (*Server, error) {
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

	if err := s.initRouter(ctx); err != nil {
		return nil, fmt.Errorf("failed to init router: %w", err)
	}

	return s, nil
}

// Start 启动服务器
func (s *Server) Start(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Starting hub-server...")

	// 启动 HTTP 服务器
	port := fmt.Sprintf(":%d", s.cfg.GetPort())
	logger.Infof(ctx, "[Server] HTTP server starting on port %s", port)

	go func() {
		if err := s.httpServer.Run(port); err != nil {
			logger.Errorf(ctx, "[Server] HTTP server error: %v", err)
		}
	}()

	logger.Infof(ctx, "[Server] Hub-server started successfully")
	return nil
}

// Stop 停止服务器（优雅关闭）
func (s *Server) Stop(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Stopping hub-server...")

	// 关闭数据库连接
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err == nil {
			sqlDB.Close()
			logger.Infof(ctx, "[Server] Database connection closed")
		}
	}

	logger.Infof(ctx, "[Server] Hub-server stopped")
	return nil
}

// initDatabase 初始化数据库
func (s *Server) initDatabase(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing database...")

	dbCfg := s.cfg.DB

	// 配置 GORM 日志
	gormConfig := &gorm.Config{}
	// 关闭 GORM 控制台日志
	gormConfig.Logger = gormLogger.Default.LogMode(gormLogger.Silent)

	// 连接 MySQL
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbCfg.User, dbCfg.Password, dbCfg.Host, dbCfg.Port, dbCfg.Name)
	
	var err error
	s.db, err = gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	// 自动迁移表结构
	if err := model.InitTables(s.db); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	logger.Infof(ctx, "[Server] Database initialized successfully")
	return nil
}

// initServices 初始化所有业务服务
func (s *Server) initServices(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing services...")

		// 初始化 Repository
		hubDirectoryRepo := repository.NewHubDirectoryRepository(s.db)
		hubServiceTreeRepo := repository.NewHubServiceTreeRepository(s.db)
		hubSnapshotRepo := repository.NewHubSnapshotRepository(s.db)
		hubFileSnapshotRepo := repository.NewHubFileSnapshotRepository(s.db)
		// constructionLogRepo := repository.NewConstructionLogRepository(s.db)
		// paymentRepo := repository.NewPaymentRepository(s.db)

		// 初始化 Service
		s.hubDirectoryService = service.NewHubDirectoryService(
			hubDirectoryRepo,
			hubServiceTreeRepo,
			hubSnapshotRepo,
			hubFileSnapshotRepo,
		)
	// s.authService = service.NewAuthService(s.osClient)
	// s.paymentService = service.NewPaymentService(paymentRepo)

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

	// 添加中间件
	s.httpServer.Use(gin.Logger())
	s.httpServer.Use(gin.Recovery())

	// 设置路由
	s.setupRoutes()

	logger.Infof(ctx, "[Server] Router initialized successfully")
	return nil
}

// healthHandler 健康检查
func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "ok",
		"service": "hub-server",
	})
}

