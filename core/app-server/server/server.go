package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/enterprise"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	middleware2 "github.com/ai-agent-os/ai-agent-os/pkg/middleware"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
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
	jwtService                    *service.JWTService
	appRuntime                    *service.AppRuntime
	serviceTreeService            *service.ServiceTreeService
	functionService               *service.FunctionService
	functionGenService            *service.FunctionGenService
	directoryUpdateHistoryService *service.DirectoryUpdateHistoryService
	permissionService             *service.PermissionService // ⭐ 权限管理服务
	appRepo                       *repository.AppRepository  // ⭐ 应用仓储（用于其他服务）

	// 上游服务
	natsService *service.NatsService

	// NATS 订阅
	functionGenSub *nats.Subscription

	// 上下文
	ctx context.Context

	//企业功能
	operateLogger enterprise.OperateLogger

	// License Client
	licenseClient *license.Client
}

// NewServer 创建新的服务器实例
func NewServer(cfg *config.AppServerConfig) (*Server, error) {
	ctx := context.Background()

	s := &Server{
		cfg: cfg,
		ctx: ctx,
	}

	// ⭐ 1. 首先加载 License（必须在其他初始化之前）
	if err := s.initLicense(ctx); err != nil {
		// License 加载失败，记录警告但不中断启动（社区版可以继续运行）
		logger.Warnf(ctx, "[Server] Failed to load license: %v, continuing with community edition", err)
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

	// ⭐ 2. 初始化企业功能（在数据库和 NATS 初始化之后，在服务初始化之前）
	// ⭐ 这样 enterprise.GetPermissionService() 就可以在 initServices 中使用了
	if err := s.initEnterprise(); err != nil {
		return nil, fmt.Errorf("failed to init enterprise features: %w", err)
	}

	if err := s.initServices(ctx); err != nil {
		return nil, fmt.Errorf("failed to init services: %w", err)
	}

	// ⭐ 初始化系统工作空间（只初始化 official 工作空间）
	// 注意：system 用户应该在 hr-server 中初始化
	// 在服务初始化之后，路由初始化之前
	if err := service.InitSystemWorkspace(ctx, s.appService); err != nil {
		logger.Warnf(ctx, "[Server] 初始化系统工作空间失败: %v", err)
		// 不中断启动，记录警告即可
	}

	if err := s.initRouter(ctx); err != nil {
		return nil, fmt.Errorf("failed to init router: %w", err)
	}

	// 初始化 NATS 订阅
	if err := s.initNATSSubscriptions(ctx); err != nil {
		return nil, fmt.Errorf("failed to init NATS subscriptions: %w", err)
	}

	return s, nil
}

// Start 启动服务器
func (s *Server) Start(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Starting app-server...")

	// 启动 HTTP 服务器
	port := fmt.Sprintf(":%d", s.cfg.GetPort())
	logger.Infof(ctx, "[Server] HTTP server starting on port %s", port)

	go func() {
		if err := s.httpServer.Run(port); err != nil {
			logger.Errorf(ctx, "[Server] HTTP server error: %v", err)
		}
	}()

	logger.Infof(ctx, "[Server] App-server started successfully")
	logger.Infof(ctx, "[Server] NATS subscriptions are active")
	return nil
}

// Stop 停止服务器（优雅关闭）
func (s *Server) Stop(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Stopping app-server...")

	// 关闭 NATS 订阅
	if s.functionGenSub != nil {
		if err := s.functionGenSub.Unsubscribe(); err != nil {
			logger.Warnf(ctx, "[Server] Failed to unsubscribe function gen: %v", err)
		} else {
			logger.Infof(ctx, "[Server] Function gen subscription closed")
		}
	}

	// 关闭 AppRuntime 服务（包括 NATS 订阅）
	if s.appRuntime != nil {
		s.appRuntime.Close()
		logger.Infof(ctx, "[Server] AppRuntime service closed")
	}

	// 关闭 NATS 服务
	if s.natsService != nil {
		s.natsService.Close()
		logger.Infof(ctx, "[Server] NATS service closed")
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

	logger.Infof(ctx, "[Server] App-server stopped")
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

// initDatabase 初始化数据库
func (s *Server) initDatabase(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing database...")

	dbCfg := s.cfg.GetDB()

	// 配置 GORM 日志
	gormConfig := &gorm.Config{}
	// 关闭 GORM 控制台日志
	gormConfig.Logger = gormLogger.Default.LogMode(gormLogger.Silent)

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

	// ✅ 配置数据库连接池（解决 "Too many connections" 问题）
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 从配置读取连接池参数，如果没有配置则使用默认值
	maxOpenConns := dbCfg.MaxOpenConns
	if maxOpenConns <= 0 {
		maxOpenConns = 100 // 默认值
	}
	maxIdleConns := dbCfg.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = 10 // 默认值
	}
	maxLifetime := time.Duration(dbCfg.MaxLifetime) * time.Second
	if maxLifetime <= 0 {
		maxLifetime = 300 * time.Second // 默认 5 分钟
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)   // 最大打开连接数
	sqlDB.SetMaxIdleConns(maxIdleConns)   // 最大空闲连接数
	sqlDB.SetConnMaxLifetime(maxLifetime) // 连接最大生命周期

	logger.Infof(ctx, "[Server] Database connection pool configured: MaxOpenConns=%d, MaxIdleConns=%d, MaxLifetime=%v",
		maxOpenConns, maxIdleConns, maxLifetime)

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
	s.natsConn, err = nats.Connect(natsConfig.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	logger.Infof(ctx, "[Server] NATS connection initialized successfully")
	return nil
}

// initServices 初始化所有业务服务
func (s *Server) initServices(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing services...")

	// 初始化 NATS 服务 - 其他服务的基础依赖
	s.natsService = service.NewNatsServiceWithDB(s.db)

	// 初始化应用运行时服务
	s.appRuntime = service.NewAppRuntimeService(s.cfg, s.natsService)

	// 初始化应用服务
	s.appRepo = repository.NewAppRepository(s.db) // ⭐ 保存到 Server 结构体，供权限服务使用
	appRepo := s.appRepo                          // 局部变量，用于传递给其他服务
	//hostRepo := repository.NewHostRepository(s.db)
	functionRepo := repository.NewFunctionRepository(s.db)
	serviceTreeRepo := repository.NewServiceTreeRepository(s.db)
	operateLogRepo := repository.NewOperateLogRepository(s.db)
	fileSnapshotRepo := repository.NewFileSnapshotRepository(s.db)
	directoryUpdateHistoryRepo := repository.NewDirectoryUpdateHistoryRepository(s.db)
	s.appService = service.NewAppService(s.appRuntime, appRepo, functionRepo, serviceTreeRepo, operateLogRepo, fileSnapshotRepo, directoryUpdateHistoryRepo)

	// ⭐ 邮件服务已迁移到 hr-server，不再需要初始化

	// 初始化 JWT 服务
	s.jwtService = service.NewJWTService()

	// 初始化函数生成服务
	s.functionGenService = service.NewFunctionGenService(s.appService, serviceTreeRepo, appRepo)

	// ⭐ 初始化权限申请仓储
	permissionRequestRepo := repository.NewPermissionRequestRepository(s.db)

	// ⭐ 初始化权限管理服务（需要在 initEnterprise 之后，因为需要 enterprise.GetPermissionService()）
	// ⭐ 完全移除 Casbin，使用新的权限系统（不再需要 appRepo，从 resourcePath 解析 user 和 app）
	s.permissionService = service.NewPermissionService(enterprise.GetPermissionService(), serviceTreeRepo, permissionRequestRepo)

	// 初始化服务目录服务（包含目录管理功能：copy、create、remove）
	s.serviceTreeService = service.NewServiceTreeService(serviceTreeRepo, appRepo, s.appRuntime, fileSnapshotRepo, s.appService, s.functionGenService, s.permissionService)

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

	// 创建 gin 引擎
	s.httpServer = gin.New()

	// 添加中间件
	s.httpServer.Use(gin.Recovery())
	s.httpServer.Use(middleware2.Cors())
	// ✅ 移除 WithTraceId 中间件，统一在网关生成 TraceId
	// s.httpServer.Use(middleware2.WithTraceId())
	// 注意：gzip 压缩只在服务树接口上使用，在路由层面配置

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

// initNATSSubscriptions 初始化 NATS 订阅
func (s *Server) initNATSSubscriptions(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing NATS subscriptions...")

	// 订阅 agent-server 的函数生成结果主题
	subject := subjects.GetAgentServerFunctionGenSubject()
	sub, err := s.natsConn.Subscribe(subject, s.handleFunctionGenResult)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", subject, err)
	}
	s.functionGenSub = sub
	logger.Infof(ctx, "[Server] Subscribed to %s", subject)

	return nil
}

// handleFunctionGenResult 处理函数生成结果消息（NATS 订阅）
func (s *Server) handleFunctionGenResult(msg *nats.Msg) {
	ctx := context.Background()

	// 从消息 header 中获取 trace_id 和 user
	traceID := msg.Header.Get("X-Trace-Id")
	requestUser := msg.Header.Get("X-Request-User")

	// 如果有 trace_id，设置到 context 中
	if traceID != "" {
		ctx = context.WithValue(ctx, "trace_id", traceID)
	}
	if requestUser != "" {
		ctx = context.WithValue(ctx, contextx.RequestUserHeader, requestUser)
	}

	// 解析消息体
	var req dto.AddFunctionsReq
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		logger.Errorf(ctx, "[Server] Failed to unmarshal add functions request: %v, Data: %s", err, string(msg.Data))
		return
	}

	// 调用 Service 层处理
	if err := s.functionGenService.ProcessFunctionGenResult(ctx, &req); err != nil {
		logger.Errorf(ctx, "[Server] 处理函数生成结果失败: %v", err)
	}
}

// HandleFunctionGenResult 处理函数生成结果（HTTP 接口，实现 FunctionGenServer 接口）
func (s *Server) HandleFunctionGenResult(c *gin.Context, req *dto.AddFunctionsReq) {
	ctx := contextx.ToContext(c)

	// 打印日志
	logger.Infof(ctx, "[Server] Received add functions request (HTTP):")
	logger.Infof(ctx, "[Server]   RecordID: %d", req.RecordID)
	logger.Infof(ctx, "[Server]   AgentID: %d", req.AgentID)
	logger.Infof(ctx, "[Server]   TreeID: %d", req.TreeID)
	logger.Infof(ctx, "[Server]   User: %s", req.User)
	logger.Infof(ctx, "[Server]   Code length: %d bytes", len(req.Code))

	// 调用 Service 层处理
	if err := s.functionGenService.ProcessFunctionGenResult(ctx, req); err != nil {
		logger.Errorf(ctx, "[Server] 处理添加函数请求失败: %v", err)
	}
}
