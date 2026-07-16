package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/core/app-server/service"
	"github.com/kageos/kageos/pkg/appcall"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/dbx"
	"github.com/kageos/kageos/pkg/logger"
	middleware2 "github.com/kageos/kageos/pkg/middleware"
	"github.com/kageos/kageos/pkg/natsx"
	"github.com/kageos/kageos/pkg/openapitoken"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/kageos/kageos/pkg/serverx"
	"github.com/kageos/kageos/pkg/waiter"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

// Server app-server 服务器
type Server struct {
	// 配置
	cfg *config.AppServerConfig

	// 核心组件
	db          *gorm.DB
	natsConn    *nats.Conn
	httpServer  *gin.Engine
	httpRuntime *serverx.HTTPServer

	// 服务
	appService                    *service.AppService
	appCall                       *appcall.Client // 调用 app-runtime 的 SDK 客户端（替代原 AppRuntime）
	serviceTreeService            *service.ServiceTreeService
	functionService               *service.FunctionService
	docService                    *service.DocService
	directoryUpdateHistoryService *service.DirectoryUpdateHistoryService
	operateLogService             *service.OperateLogService
	teamAccessService             *service.TeamAccessService
	functionSensitiveFieldService *service.FunctionSensitiveFieldService
	publicShareService            *service.PublicShareService
	appRepo                       *repository.AppRepository // ⭐ 应用仓储（用于其他服务）
	scheduledFuncWorker           *scheduledsdk.Worker

	// 上游服务
	natsConnPool *service.NATSConnPool

	// 上下文
	ctx context.Context
}

func newBaseServer(cfg *config.AppServerConfig) *Server {
	ctx := context.Background()
	return &Server{
		cfg: cfg,
		ctx: ctx,
	}
}

func (s *Server) initSharedComponents(ctx context.Context) error {
	if err := s.initDatabase(ctx); err != nil {
		return fmt.Errorf("failed to init database: %w", err)
	}

	if err := s.initNATS(ctx); err != nil {
		return fmt.Errorf("failed to init NATS: %w", err)
	}

	return nil
}

// NewServer 创建新的 app-server 实例。
func NewServer(cfg *config.AppServerConfig) (*Server, error) {
	s := newBaseServer(cfg)
	ctx := s.ctx

	if err := s.initSharedComponents(ctx); err != nil {
		return nil, err
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
	logger.Infof(ctx, "[Server] Starting app-server...")

	if s.scheduledFuncWorker != nil {
		if err := s.scheduledFuncWorker.Start(ctx); err != nil {
			return fmt.Errorf("failed to start scheduled function worker: %w", err)
		}
		logger.Infof(ctx, "[Server] Scheduled function worker started")
	}

	if err := s.StartHTTP(ctx); err != nil {
		if s.scheduledFuncWorker != nil {
			_ = s.scheduledFuncWorker.Stop()
		}
		return err
	}

	s.startSystemWorkspaceInit(ctx)

	logger.Infof(ctx, "[Server] App-server started successfully")
	logger.Infof(ctx, "[Server] NATS subscriptions are active")
	return nil
}

func (s *Server) startSystemWorkspaceInit(ctx context.Context) {
	go func() {
		logger.Infof(ctx, "[Server] 系统工作空间后台初始化开始")
		if err := service.InitSystemWorkspace(ctx, s.appService, s.serviceTreeService); err != nil {
			logger.Warnf(ctx, "[Server] 初始化系统工作空间失败: %v", err)
			return
		}
		logger.Infof(ctx, "[Server] 系统工作空间后台初始化完成")
	}()
}

func (s *Server) StartHTTP(ctx context.Context) error {
	if s.httpServer == nil {
		return fmt.Errorf("http server is not initialized")
	}
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
	return nil
}

// Stop 停止服务器（优雅关闭）
func (s *Server) Stop(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Stopping server...")
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

	if s.scheduledFuncWorker != nil {
		if err := s.scheduledFuncWorker.Stop(); err != nil {
			logger.Warnf(ctx, "[Server] Scheduled function worker stop failed: %v", err)
		} else {
			logger.Infof(ctx, "[Server] Scheduled function worker stopped")
		}
	}

	// 关闭 appcall 客户端（取消 NATS 订阅）
	if s.appCall != nil {
		_ = s.appCall.Close()
		logger.Infof(ctx, "[Server] appcall client closed")
	}

	// 关闭 NATS 连接池
	if s.natsConnPool != nil {
		s.natsConnPool.Close()
		logger.Infof(ctx, "[Server] NATS conn pool closed")
	}

	// 关闭 NATS 连接
	if s.natsConn != nil {
		s.natsConn.Close()
		logger.Infof(ctx, "[Server] NATS connection closed")
	}

	// 关闭数据库连接
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err == nil {
			sqlDB.Close()
			logger.Infof(ctx, "[Server] Database connection closed")
		}
	}

	logger.Infof(ctx, "[Server] Server stopped")
	return stopErr
}

// initDatabase 初始化数据库
func (s *Server) initDatabase(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing database...")

	dbCfg := s.cfg.GetDB()
	if dbCfg.Type != "mysql" {
		return fmt.Errorf("unsupported database type: %s", dbCfg.Type)
	}
	db, err := dbx.OpenMySQL(dbCfg, dbx.OpenOptions{
		DisableForeignKeyConstraintWhenMigrating: true,
		DefaultMaxLifetime:                       5 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}
	s.db = db

	// 自动迁移表结构
	if err := model.InitTables(s.db); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}
	if err := openapitoken.SetDB(s.db); err != nil {
		return fmt.Errorf("failed to init openapi token store: %w", err)
	}

	logger.Infof(ctx, "[Server] Database initialized successfully")
	return nil
}

// initNATS 初始化 NATS 连接
func (s *Server) initNATS(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing NATS connection...")

	var err error
	natsConfig := s.cfg.GetNats()
	s.natsConn, err = natsx.Connect(natsConfig.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	logger.Infof(ctx, "[Server] NATS connection initialized successfully")
	return nil
}

// initServices 初始化所有业务服务
func (s *Server) initServices(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing services...")

	if err := model.ReconcileNatsHostFromURL(s.db, s.cfg.GetNats().URL); err != nil {
		return fmt.Errorf("reconcile nats host from config: %w", err)
	}
	if err := model.ReconcileNatsHostFromEnv(s.db); err != nil {
		return fmt.Errorf("reconcile nats host from NATS_SEED_HOST: %w", err)
	}

	// 初始化 NATS 连接池 - 其他服务调用 app-runtime 的基础依赖
	s.natsConnPool = service.NewNATSConnPoolWithDB(s.db)

	// 初始化 appcall 客户端（调用 app-runtime 的 SDK 风格客户端，依赖注入）
	s.appCall = appcall.New(appcall.Options{
		ConnProvider:       s.natsConnPool,
		NatsRequestTimeout: time.Duration(s.cfg.GetNatsRequestTimeout()) * time.Second,
		AppRequestTimeout:  time.Duration(s.cfg.GetAppRequestTimeout()) * time.Second,
		Waiter:             waiter.GetDefaultWaiter(),
	})

	if s.appRepo == nil {
		s.appRepo = repository.NewAppRepository(s.db)
	}
	appRepo := s.appRepo // 局部变量，用于传递给其他服务
	//hostRepo := repository.NewHostRepository(s.db)
	functionRepo := repository.NewFunctionRepository(s.db)
	serviceTreeRepo := repository.NewServiceTreeRepository(s.db)
	operateLogRepo := repository.NewOperateLogRepository(s.db)
	teamAccessRepo := repository.NewTeamAccessRepository(s.db)
	functionSensitiveFieldRepo := repository.NewFunctionSensitiveFieldRepository(s.db)
	publicShareRepo := repository.NewPublicShareRepository(s.db)
	fileSnapshotRepo := repository.NewFileSnapshotRepository(s.db)
	directoryUpdateHistoryRepo := repository.NewDirectoryUpdateHistoryRepository(s.db)
	s.appService = service.NewAppService(s.appCall, appRepo, functionRepo, serviceTreeRepo, operateLogRepo)
	s.operateLogService = service.NewOperateLogService(operateLogRepo)
	s.teamAccessService = service.NewTeamAccessService(teamAccessRepo, operateLogRepo, appRepo)
	s.functionSensitiveFieldService = service.NewFunctionSensitiveFieldService(functionSensitiveFieldRepo)
	if err := s.functionSensitiveFieldService.LoadAll(ctx); err != nil {
		return fmt.Errorf("加载敏感字段缓存失败: %w", err)
	}
	s.appService.SetTeamAccessService(s.teamAccessService)
	s.appService.SetFunctionSensitiveFieldService(s.functionSensitiveFieldService)
	scheduledFuncWorker, err := service.NewScheduledFunctionWorker(s.natsConn, s.appService)
	if err != nil {
		return fmt.Errorf("failed to init scheduled function worker: %w", err)
	}
	s.scheduledFuncWorker = scheduledFuncWorker

	// 初始化文档服务（需要在 ServiceTreeService 之前初始化，因为 ServiceTreeService 依赖它）
	docRepo := repository.NewDocRepository(s.db)
	s.docService = service.NewDocService(docRepo, serviceTreeRepo, appRepo, s.teamAccessService)
	s.appService.SetDocService(s.docService)

	// 初始化服务目录服务（包含目录管理功能：copy、create、remove）
	// ⭐ 函数生成逻辑已移到 ServiceTreeService 中
	s.serviceTreeService = service.NewServiceTreeService(serviceTreeRepo, appRepo, s.appCall, fileSnapshotRepo, s.appService, s.docService, s.teamAccessService)
	if _, err := service.ReconcileAppRootServiceTrees(ctx, appRepo, serviceTreeRepo); err != nil {
		return fmt.Errorf("reconcile app root service trees: %w", err)
	}

	// 初始化函数服务
	s.functionService = service.NewFunctionService(functionRepo, appRepo)

	// 初始化公开分享服务（MVP: Form 匿名提交）
	s.publicShareService = service.NewPublicShareService(publicShareRepo, functionRepo, serviceTreeRepo)

	// 初始化目录更新历史服务
	s.directoryUpdateHistoryService = service.NewDirectoryUpdateHistoryService(directoryUpdateHistoryRepo, serviceTreeRepo)

	logger.Infof(ctx, "[Server] Services initialized successfully")
	return nil
}

// initRouter 初始化路由
func (s *Server) initRouter(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing router...")

	// 创建 gin 引擎并挂载通用中间件
	// ✅ 移除 WithTraceId 中间件，统一在网关生成 TraceId
	// s.httpServer.Use(middleware2.WithTraceId())
	// 注意：gzip 压缩只在服务树接口上使用，在路由层面配置
	s.httpServer = serverx.NewGin(
		serverx.WithRecovery(),
		serverx.WithMiddleware(middleware2.Cors()),
		serverx.WithRegisteredMiddlewares(serverx.ServiceAppServer),
	)

	// 设置路由
	s.setupRoutes()

	// 设置 router 引用

	logger.Infof(ctx, "[Server] Router initialized successfully")
	return nil
}

// healthHandler 健康检查处理器
func (s *Server) healthHandler(c *gin.Context) {
	pingCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := s.pingDatabase(pingCtx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":     "unavailable",
			"timestamp":  time.Now().Format(time.DateTime),
			"service":    "app-server",
			"dependency": "mysql",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Format(time.DateTime),
		"service":   "app-server",
	})
}

func (s *Server) pingDatabase(ctx context.Context) error {
	if s.db == nil {
		return fmt.Errorf("database is not initialized")
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// GetDB 获取数据库连接
func (s *Server) GetDB() *gorm.DB {
	return s.db
}

// GetNATS 获取 NATS 连接
func (s *Server) GetNATS() *nats.Conn {
	return s.natsConn
}

// GetAppService 获取应用服务
func (s *Server) GetAppService() *service.AppService {
	return s.appService
}

// getDirectoryUpdateHistoryRepo 获取目录更新历史Repository（内部方法，用于路由注册）
func (s *Server) getDirectoryUpdateHistoryRepo() *repository.DirectoryUpdateHistoryRepository {
	return repository.NewDirectoryUpdateHistoryRepository(s.db)
}
