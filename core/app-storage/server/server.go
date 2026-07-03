package server

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/app-storage/model"
	"github.com/kageos/kageos/core/app-storage/repository"
	"github.com/kageos/kageos/core/app-storage/service"
	"github.com/kageos/kageos/core/app-storage/storage"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/dbx"
	"github.com/kageos/kageos/pkg/logger"
	middleware2 "github.com/kageos/kageos/pkg/middleware"
	"github.com/kageos/kageos/pkg/serverx"
	"gorm.io/gorm"
)

// Server app-storage 服务器
type Server struct {
	// 配置
	cfg *config.AppStorageConfig

	// 核心组件
	db          *gorm.DB
	storage     storage.Storage // ✅ 存储接口（抽象）
	httpServer  *gin.Engine
	httpRuntime *serverx.HTTPServer

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
	addr := net.JoinHostPort(s.cfg.GetListenHost(), strconv.Itoa(s.cfg.GetPort()))
	logger.Infof(ctx, "[Server] HTTP server starting on %s", addr)

	httpRuntime, err := serverx.StartHTTPServer(ctx, addr, s.httpServer)
	if err != nil {
		return fmt.Errorf("failed to start HTTP server on %s: %w", addr, err)
	}
	s.httpRuntime = httpRuntime
	go func() {
		if err := <-httpRuntime.Err(); err != nil {
			logger.Errorf(ctx, "[Server] HTTP server error: %v", err)
		}
	}()

	logger.Infof(ctx, "[Server] App-storage started successfully")
	return nil
}

// Stop 停止服务器（优雅关闭）
func (s *Server) Stop(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Stopping app-storage...")
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

	// 关闭数据库连接
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err == nil {
			sqlDB.Close()
			logger.Infof(ctx, "[Server] Database connection closed")
		}
	}

	logger.Infof(ctx, "[Server] App-storage stopped")
	return stopErr
}

// initDatabase 初始化可选数据库，用于记录文件上传/下载元数据。
func (s *Server) initDatabase(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing database...")

	dbCfg := s.cfg.DB

	// 如果数据库配置为空，跳过元数据记录。
	if dbCfg.Host == "" {
		logger.Infof(ctx, "[Server] Database config not found, skipping file metadata records")
		return nil
	}

	if dbCfg.Type != "mysql" {
		logger.Infof(ctx, "[Server] Database type not specified, skipping")
		return nil
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
	logger.Infof(ctx, "[Server] Tables created: file_uploads, file_downloads")
	return nil
}

// initStorage 初始化存储（抽象层）
func (s *Server) initStorage(ctx context.Context) error {
	storageType := s.cfg.GetStorageType()
	logger.Infof(ctx, "[Server] Initializing storage (%s)...", storageType)

	// 通过工厂创建存储实例
	factory := storage.NewFactory()
	storageConfig := config.NewStorageConfigAdapter(s.cfg)

	storageInstance, err := factory.CreateStorage(storageType, storageConfig)
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

	logger.Infof(ctx, "[Server] Storage initialized successfully (type: %s, bucket: %s)", storageType, bucket)
	return nil
}

// initServices 初始化所有业务服务
func (s *Server) initServices(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing services...")

	// 初始化 Repository 层
	var fileRepo *repository.FileRepository
	if s.db != nil {
		fileRepo = repository.NewFileRepository(s.db)
		logger.Infof(ctx, "[Server] File repository initialized (database connected)")
	} else {
		logger.Warnf(ctx, "[Server] Database not connected, upload/download tracking will be disabled")
		logger.Warnf(ctx, "[Server] Please configure database in app-storage.yaml to enable audit tracking")
	}

	// 初始化 Service 层（依赖抽象接口）
	s.storageService = service.NewStorageService(s.storage, s.cfg, fileRepo)

	// 检查审计配置
	if s.cfg.Audit.UploadTracking.Enabled {
		if fileRepo == nil {
			logger.Warnf(ctx, "[Server] Upload tracking is enabled but database is not connected, records will not be saved")
		} else {
			logger.Infof(ctx, "[Server] Upload tracking enabled")
		}
	}

	if s.cfg.Audit.DownloadTracking.Enabled {
		if fileRepo == nil {
			logger.Warnf(ctx, "[Server] Download tracking is enabled but database is not connected, records will not be saved")
		} else {
			logger.Infof(ctx, "[Server] Download tracking enabled (retention: %d days)", s.cfg.Audit.DownloadTracking.RetentionDays)
		}
	}

	logger.Infof(ctx, "[Server] Services initialized successfully")
	return nil
}

// initRouter 初始化路由
func (s *Server) initRouter(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing router...")

	// 创建 gin 引擎并挂载通用中间件
	// ✅ 移除 WithTraceId 中间件，统一在网关生成 TraceId
	// s.httpServer.Use(middleware2.WithTraceId())
	s.httpServer = serverx.NewGin(
		serverx.WithRecovery(),
		serverx.WithMiddleware(middleware2.Cors()),
		serverx.WithRegisteredMiddlewares(serverx.ServiceAppStorage),
	)

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
