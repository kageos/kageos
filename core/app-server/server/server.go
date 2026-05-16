package server

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/ai-agent-os/ai-agent-os/enterprise"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/appcall"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/dbx"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	middleware2 "github.com/ai-agent-os/ai-agent-os/pkg/middleware"
	"github.com/ai-agent-os/ai-agent-os/pkg/natsx"
	"github.com/ai-agent-os/ai-agent-os/pkg/serverx"
	"github.com/ai-agent-os/ai-agent-os/pkg/waiter"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

// Server app-server 服务器
type Server struct {
	// 配置
	cfg *config.AppServerConfig

	// 核心组件
	db         *gorm.DB
	natsConn   *nats.Conn
	httpServer *gin.Engine

	// 服务
	appService                    *service.AppService
	appCall                       *appcall.Client // 调用 app-runtime 的 SDK 客户端（替代原 AppRuntime）
	serviceTreeService            *service.ServiceTreeService
	functionService               *service.FunctionService
	docService                    *service.DocService
	directoryUpdateHistoryService *service.DirectoryUpdateHistoryService
	permissionService             *service.PermissionService // ⭐ 权限管理服务
	appRepo                       *repository.AppRepository  // ⭐ 应用仓储（用于其他服务）

	// 上游服务
	natsConnPool *service.NATSConnPool

	// 上下文
	ctx context.Context

	//企业功能
	operateLogger enterprise.OperateLogger

	// License Client
	licenseClient *license.Client
}

func newBaseServer(cfg *config.AppServerConfig) *Server {
	ctx := context.Background()
	return &Server{
		cfg: cfg,
		ctx: ctx,
	}
}

func (s *Server) initSharedComponents(ctx context.Context) error {
	// ⭐ 1. 首先加载 License（必须在其他初始化之前）
	if err := s.initLicense(ctx); err != nil {
		// License 加载失败，记录警告但不中断启动（社区版可以继续运行）
		logger.Warnf(ctx, "[Server] Failed to load license: %v, continuing with community edition", err)
	}

	if err := s.initDatabase(ctx); err != nil {
		return fmt.Errorf("failed to init database: %w", err)
	}

	if err := s.initNATS(ctx); err != nil {
		return fmt.Errorf("failed to init NATS: %w", err)
	}

	// ⭐ 初始化 License Client（在 NATS 初始化之后）
	if err := s.initLicenseClient(ctx); err != nil {
		// License Client 初始化失败，记录警告但不中断启动（社区版可以继续运行）
		logger.Warnf(ctx, "[Server] Failed to init license client: %v, continuing with community edition", err)
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

	// ⭐ 2. 初始化企业功能（在数据库和 NATS 初始化之后，在服务初始化之前）
	// ⭐ 这样 enterprise.GetPermissionService() 就可以在 initServices 中使用了
	if err := s.initEnterprise(); err != nil {
		return nil, fmt.Errorf("failed to init enterprise features: %w", err)
	}

	if err := s.initServices(ctx); err != nil {
		return nil, fmt.Errorf("failed to init services: %w", err)
	}

	// ⭐ 初始化系统工作空间
	// 注意：system 用户应该在 hr-server 中初始化
	// 在服务初始化之后，路由初始化之前
	if err := service.InitSystemWorkspace(ctx, s.appService, s.serviceTreeService); err != nil {
		logger.Warnf(ctx, "[Server] 初始化系统工作空间失败: %v", err)
		// 不中断启动，记录警告即可
	}

	if err := s.initRouter(ctx); err != nil {
		return nil, fmt.Errorf("failed to init router: %w", err)
	}

	return s, nil
}

// Start 启动服务器
func (s *Server) Start(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Starting app-server...")

	if err := s.StartHTTP(ctx); err != nil {
		return err
	}

	logger.Infof(ctx, "[Server] App-server started successfully")
	logger.Infof(ctx, "[Server] NATS subscriptions are active")
	return nil
}

func (s *Server) StartHTTP(ctx context.Context) error {
	if s.httpServer == nil {
		return fmt.Errorf("http server is not initialized")
	}
	addr := net.JoinHostPort(s.cfg.GetListenHost(), strconv.Itoa(s.cfg.GetPort()))
	logger.Infof(ctx, "[Server] HTTP server starting on %s", addr)

	go func() {
		if err := s.httpServer.Run(addr); err != nil {
			logger.Errorf(ctx, "[Server] HTTP server error: %v", err)
		}
	}()
	return nil
}

// Stop 停止服务器（优雅关闭）
func (s *Server) Stop(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Stopping server...")

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

	// 关闭数据库连接
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err == nil {
			sqlDB.Close()
			logger.Infof(ctx, "[Server] Database connection closed")
		}
	}

	logger.Infof(ctx, "[Server] Server stopped")
	return nil
}

// initLicense 初始化 License（从文件加载，向后兼容）
// 在服务器启动时加载和验证 License 文件
func (s *Server) initLicense(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing license from file...")

	// 获取 License 管理器
	licenseMgr := license.GetManager()

	// 加载 License（如果文件不存在，返回 nil，表示社区版）
	if err := licenseMgr.LoadLicense(""); err != nil {
		// License 加载失败，可能是文件不存在（社区版）或验证失败
		// 如果是验证失败，记录错误但不中断启动（允许降级到社区版）
		logger.Warnf(ctx, "[Server] License loading from file failed: %v", err)
		return err
	}

	// 检查 License 状态
	currentLicense := licenseMgr.GetLicense()
	if currentLicense == nil {
		logger.Infof(ctx, "[Server] Community edition (no license file)")
	} else {
		logger.Infof(ctx, "[Server] License loaded from file: Edition=%s, Customer=%s, ExpiresAt=%v",
			currentLicense.Edition, currentLicense.Customer, currentLicense.ExpiresAt)
	}

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
		natsConn, err = natsx.Connect(controlCfg.GetNatsURL())
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

	// 初始化应用服务（若企业版已创建 appRepo 则复用）
	if s.appRepo == nil {
		s.appRepo = repository.NewAppRepository(s.db)
	}
	appRepo := s.appRepo // 局部变量，用于传递给其他服务
	//hostRepo := repository.NewHostRepository(s.db)
	functionRepo := repository.NewFunctionRepository(s.db)
	serviceTreeRepo := repository.NewServiceTreeRepository(s.db)
	operateLogRepo := repository.NewOperateLogRepository(s.db)
	fileSnapshotRepo := repository.NewFileSnapshotRepository(s.db)
	directoryUpdateHistoryRepo := repository.NewDirectoryUpdateHistoryRepository(s.db)
	s.appService = service.NewAppService(s.appCall, appRepo, functionRepo, serviceTreeRepo, operateLogRepo)

	// ⭐ 邮件服务已迁移到 hr-server，不再需要初始化

	// 初始化权限查询适配器（需要在 initEnterprise 之后，因为需要 enterprise.GetPermissionService()）
	s.permissionService = service.NewPermissionService()

	// 初始化文档服务（需要在 ServiceTreeService 之前初始化，因为 ServiceTreeService 依赖它）
	docRepo := repository.NewDocRepository(s.db)
	s.docService = service.NewDocService(docRepo, serviceTreeRepo, appRepo)

	// 初始化服务目录服务（包含目录管理功能：copy、create、remove）
	// ⭐ 函数生成逻辑已移到 ServiceTreeService 中
	s.serviceTreeService = service.NewServiceTreeService(serviceTreeRepo, appRepo, s.appCall, fileSnapshotRepo, s.appService, s.permissionService, s.docService)

	// 初始化函数服务
	s.functionService = service.NewFunctionService(functionRepo, appRepo)

	// 操作日志服务已迁移到企业版，通过 enterprise.GetOperateLogger() 获取

	// 初始化目录更新历史服务
	s.directoryUpdateHistoryService = service.NewDirectoryUpdateHistoryService(directoryUpdateHistoryRepo, serviceTreeRepo)

	// ⭐ 初始化权限管理服务（需要在 initEnterprise 之后，因为需要 enterprise.GetPermissionService()）
	// 注意：这里先不初始化，等 initEnterprise 之后再初始化
	// 在 initEnterprise 中会初始化 enterprise.GetPermissionService()，然后在这里创建 PermissionService

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
	)

	// 设置路由
	s.setupRoutes()

	// 设置 router 引用

	logger.Infof(ctx, "[Server] Router initialized successfully")
	return nil
}

// healthHandler 健康检查处理器
func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Format(time.DateTime),
		"service":   "app-server",
	})
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
