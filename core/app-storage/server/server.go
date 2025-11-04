package server

import (
	"context"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-storage/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-storage/repository"
	"github.com/ai-agent-os/ai-agent-os/core/app-storage/service"
	"github.com/ai-agent-os/ai-agent-os/core/app-storage/storage"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	middleware2 "github.com/ai-agent-os/ai-agent-os/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// Server app-storage 服务器
type Server struct {
	// 配置
	cfg *config.AppStorageConfig

	// 核心组件
	db          *gorm.DB
	storage     storage.Storage  // ✅ 存储接口（抽象）
	httpServer  *gin.Engine

	// 服务
	storageService *service.StorageService

	// 上下文
	ctx context.Context
}

// NewServer 创建新的服务器实例
func NewServer(cfg *config.AppStorageConfig) (*Server, error) {
	ctx := context.Background()

	s := &Server{
		cfg: cfg,
		ctx: ctx,
	}

	// 初始化各个组件
	if err := s.initDatabase(ctx); err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}

	if err := s.initStorage(ctx); err != nil {
		return nil, fmt.Errorf("failed to init storage: %w", err)
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
	logger.Infof(ctx, "[Server] Starting app-storage...")

	// 启动 HTTP 服务器
	port := fmt.Sprintf(":%d", s.cfg.GetPort())
	logger.Infof(ctx, "[Server] HTTP server starting on port %s", port)

	go func() {
		if err := s.httpServer.Run(port); err != nil {
			logger.Errorf(ctx, "[Server] HTTP server error: %v", err)
		}
	}()

	logger.Infof(ctx, "[Server] App-storage started successfully")
	return nil
}

// Stop 停止服务器（优雅关闭）
func (s *Server) Stop(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Stopping app-storage...")

	// 关闭数据库连接
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err == nil {
			sqlDB.Close()
			logger.Infof(ctx, "[Server] Database connection closed")
		}
	}

	logger.Infof(ctx, "[Server] App-storage stopped")
	return nil
}

// initDatabase 初始化数据库（可选，用于秒传功能）
func (s *Server) initDatabase(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing database...")

	dbCfg := s.cfg.DB

	// 如果数据库配置为空，跳过（秒传功能未启用）
	if dbCfg.Host == "" {
		logger.Infof(ctx, "[Server] Database config not found, skipping (deduplication disabled)")
		return nil
	}

	// 配置 GORM 日志
	gormConfig := &gorm.Config{}
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
	gormConfig.Logger = gormLogger.Default.LogMode(logLevel)

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
		logger.Infof(ctx, "[Server] Database type not specified, skipping")
		return nil
	}

	// 自动迁移表结构（预留表，用于未来秒传功能）
	if err := model.InitTables(s.db); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	logger.Infof(ctx, "[Server] Database initialized successfully (tables created for future deduplication)")
	return nil
}

// initStorage 初始化存储（抽象层）
func (s *Server) initStorage(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing storage (%s)...", s.cfg.Storage.Type)

	// 通过工厂创建存储实例
	factory := storage.NewFactory()
	storageConfig := config.NewStorageConfigAdapter(s.cfg)
	
	storageInstance, err := factory.CreateStorage(s.cfg.Storage.Type, storageConfig)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}
	s.storage = storageInstance

	// 确保默认 Bucket 存在
	bucket := storageConfig.GetDefaultBucket()
	region := storageConfig.GetRegion()
	if err := s.storage.EnsureBucket(ctx, bucket, region); err != nil {
		return fmt.Errorf("failed to ensure bucket: %w", err)
	}

	logger.Infof(ctx, "[Server] Storage initialized successfully (type: %s, bucket: %s)", s.cfg.Storage.Type, bucket)
	return nil
}

// initServices 初始化所有业务服务
func (s *Server) initServices(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing services...")

	// 初始化 Repository 层
	var fileRepo *repository.FileRepository
	if s.db != nil {
		fileRepo = repository.NewFileRepository(s.db)
	}

	// 初始化 Service 层（依赖抽象接口）
	s.storageService = service.NewStorageService(s.storage, s.cfg, fileRepo)

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
	s.httpServer.Use(middleware2.WithTraceId())

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
		"service":   "app-storage",
	})
}

