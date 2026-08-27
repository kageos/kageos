package server

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/core/hr-server/repository"
	"github.com/kageos/kageos/core/hr-server/service"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/dbx"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/natsx"
	"github.com/kageos/kageos/pkg/openapitoken"
	"github.com/kageos/kageos/pkg/serverx"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

// Server hr-server 服务器
type Server struct {
	// 配置
	cfg *config.HRServerConfig

	// 核心组件
	db          *gorm.DB
	httpServer  *gin.Engine
	httpRuntime *serverx.HTTPServer
	natsConn    *nats.Conn

	// 服务
	authService         *service.AuthService
	authOAuthService    *service.AuthOAuthService
	authWechatService   *service.AuthWechatOfficialService
	authProviderService *service.AuthLoginProviderService
	emailService        *service.EmailService
	settingsService     *service.SystemSettingsService
	resourceService     *service.SystemResourceService
	userService         *service.UserService
	departmentService   *service.DepartmentService
	tokenPublisher      service.TokenPublisher
	openAPITokenStore   *openapitoken.Store
	subscriptions       []*nats.Subscription

	// 上下文
	ctx context.Context
}

// NewServer 创建新的服务器实例
func NewServer(cfg *config.HRServerConfig) (*Server, error) {
	ctx := context.Background()

	s := &Server{
		cfg:           cfg,
		ctx:           ctx,
		subscriptions: make([]*nats.Subscription, 0),
	}

	// 初始化各个组件
	if err := s.initDatabase(ctx); err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}

	if err := s.initNATS(ctx); err != nil {
		if s.cfg.AllowNATSDegradedStartup() {
			logger.Warnf(ctx, "[Server] Failed to initialize NATS: %v, allow_nats_degraded_startup=true so HR server will continue", err)
		} else {
			return nil, fmt.Errorf("failed to initialize required NATS: %w", err)
		}
	}

	if err := s.initServices(ctx); err != nil {
		return nil, fmt.Errorf("failed to init services: %w", err)
	}
	if err := s.authProviderService.SeedDefaults(ctx); err != nil {
		return nil, fmt.Errorf("failed to seed auth login providers: %w", err)
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

	if err := s.subscribeNATS(ctx); err != nil {
		if s.cfg.AllowNATSDegradedStartup() {
			logger.Warnf(ctx, "[Server] NATS subscribe failed: %v, allow_nats_degraded_startup=true so HR server will continue", err)
		} else {
			return fmt.Errorf("failed to subscribe required NATS: %w", err)
		}
	}

	// 启动 HTTP 服务器
	addr := net.JoinHostPort(s.cfg.GetListenHost(), strconv.Itoa(s.cfg.GetPort()))
	logger.Infof(ctx, "[Server] HTTP server starting on %s", addr)

	httpRuntime, err := serverx.StartHTTPServer(ctx, addr, s.httpServer)
	if err != nil {
		s.unsubscribeNATS(ctx)
		s.closeNATS(ctx)
		return fmt.Errorf("failed to start HTTP server on %s: %w", addr, err)
	}
	s.httpRuntime = httpRuntime
	go func() {
		if err := <-httpRuntime.Err(); err != nil {
			logger.Errorf(ctx, "[Server] HTTP server error: %v", err)
		}
	}()

	logger.Infof(ctx, "[Server] HR-server started successfully")
	s.resourceService.Start(ctx)
	return nil
}

// Stop 停止服务器（优雅关闭）
func (s *Server) Stop(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Stopping hr-server...")
	var stopErr error
	s.resourceService.Stop()

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

	s.unsubscribeNATS(ctx)
	s.closeNATS(ctx)

	logger.Infof(ctx, "[Server] HR-server stopped")
	return stopErr
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
	openAPITokenStore, err := openapitoken.NewStore(db)
	if err != nil {
		return fmt.Errorf("failed to init openapi token store: %w", err)
	}
	s.openAPITokenStore = openAPITokenStore

	logger.Infof(ctx, "[Server] Database initialized successfully")
	return nil
}

// initNATS 初始化 NATS 连接
func (s *Server) initNATS(ctx context.Context) error {
	natsURL := config.GetGlobalSharedConfig().Nats.URL
	if natsURL == "" {
		natsURL = "nats://127.0.0.1:4222"
	}

	conn, err := natsx.ConnectNamed(natsURL, "hr-server")
	if err != nil {
		return fmt.Errorf("connect NATS: %w", err)
	}

	s.natsConn = conn
	logger.Infof(ctx, "[Server] NATS initialized successfully")
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
	settingRepo := repository.NewSystemSettingRepository(s.db)
	resourceRepo := repository.NewSystemResourceRepository(s.db)
	authProviderRepo := repository.NewAuthLoginProviderRepository(s.db)
	authOAuthStateRepo := repository.NewAuthOAuthStateRepository(s.db)
	authOAuthRegistrationIntentRepo := repository.NewAuthOAuthRegistrationIntentRepository(s.db)
	authExternalIdentityRepo := repository.NewAuthExternalIdentityRepository(s.db)
	authWechatAttemptRepo := repository.NewAuthWechatLoginAttemptRepository(s.db)

	if s.natsConn != nil {
		s.tokenPublisher = service.NewGatewayTokenPublisher(s.natsConn)
	}

	// 初始化认证服务
	s.authService = service.NewAuthService(userRepo, userSessionRepo, s.tokenPublisher)
	s.authProviderService = service.NewAuthLoginProviderService(authProviderRepo)
	s.authOAuthService = service.NewAuthOAuthService(s.authService, s.authProviderService, authOAuthStateRepo, authOAuthRegistrationIntentRepo, authExternalIdentityRepo, userRepo)
	s.authWechatService = service.NewAuthWechatOfficialService(s.authProviderService, authWechatAttemptRepo, s.authOAuthService)
	s.settingsService = service.NewSystemSettingsService(settingRepo)
	s.resourceService = service.NewSystemResourceService(resourceRepo)

	// 初始化邮件服务
	s.emailService = service.NewEmailService(emailCodeRepo, s.settingsService)

	// 初始化用户服务
	s.userService = service.NewUserService(userRepo, s.tokenPublisher, userSessionRepo, deptRepo)

	s.departmentService = service.NewDepartmentService(deptRepo, userRepo)

	logger.Infof(ctx, "[Server] Services initialized successfully")
	return nil
}

// subscribeNATS 注册所有 NATS 订阅
func (s *Server) subscribeNATS(ctx context.Context) error {
	return RegisterNATS(ctx, s.natsConn, &s.subscriptions)
}

// unsubscribeNATS 取消所有 NATS 订阅
func (s *Server) unsubscribeNATS(ctx context.Context) {
	for _, sub := range s.subscriptions {
		if sub == nil {
			continue
		}
		if err := sub.Unsubscribe(); err != nil {
			logger.Warnf(ctx, "[Server] Failed to unsubscribe NATS subject %s: %v", sub.Subject, err)
		}
	}
	s.subscriptions = s.subscriptions[:0]
}

// closeNATS 关闭 NATS 连接
func (s *Server) closeNATS(ctx context.Context) {
	if s.natsConn == nil {
		return
	}
	s.natsConn.Close()
	s.natsConn = nil
	logger.Infof(ctx, "[Server] NATS connection closed")
}

// initRouter 初始化路由
func (s *Server) initRouter(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing router...")

	// 创建 Gin 引擎
	s.httpServer = serverx.NewGin(
		serverx.WithDebug(s.cfg.IsDebug()),
		serverx.WithRegisteredMiddlewares(serverx.ServiceHRServer),
	)

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
