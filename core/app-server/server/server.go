package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	middleware2 "github.com/ai-agent-os/ai-agent-os/pkg/middleware"
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
	appService         *service.AppService
	authService        *service.AuthService
	emailService       *service.EmailService
	jwtService         *service.JWTService
	appRuntime         *service.AppRuntime
	serviceTreeService *service.ServiceTreeService
	functionService    *service.FunctionService
	userService        *service.UserService

	// 上游服务
	natsService *service.NatsService

	// NATS 订阅
	functionGenSub *nats.Subscription

	// 上下文
	ctx context.Context
}

// NewServer 创建新的服务器实例
func NewServer(cfg *config.AppServerConfig) (*Server, error) {
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

// initDatabase 初始化数据库
func (s *Server) initDatabase(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing database...")

	dbCfg := s.cfg.DB

	// 配置 GORM 日志
	gormConfig := &gorm.Config{}

	// 如果启用了数据库日志
	if dbCfg.LogLevel != "silent" {
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

		// 配置慢查询阈值
		slowThreshold := time.Duration(dbCfg.SlowThreshold) * time.Millisecond
		if slowThreshold == 0 {
			slowThreshold = 200 * time.Millisecond // 默认200毫秒
		}

		// 使用 GORM 默认日志配置
		gormConfig.Logger = gormLogger.Default.LogMode(logLevel)
	} else {
		// 禁用日志
		gormConfig.Logger = gormLogger.Default.LogMode(gormLogger.Silent)
	}

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
	s.natsConn, err = nats.Connect(s.cfg.Nats.URL)
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
	userRepo := repository.NewUserRepository(s.db)
	appRepo := repository.NewAppRepository(s.db)
	hostRepo := repository.NewHostRepository(s.db)
	userSessionRepo := repository.NewUserSessionRepository(s.db)
	functionRepo := repository.NewFunctionRepository(s.db)
	serviceTreeRepo := repository.NewServiceTreeRepository(s.db)
	sourceCodeRepo := repository.NewSourceCodeRepository(s.db)

	s.appService = service.NewAppService(s.appRuntime, userRepo, appRepo, functionRepo, serviceTreeRepo, sourceCodeRepo)

	// 初始化认证服务
	s.authService = service.NewAuthService(userRepo, hostRepo, userSessionRepo)

	// 初始化邮件服务
	emailCodeRepo := repository.NewEmailCodeRepository(s.db)
	s.emailService = service.NewEmailService(emailCodeRepo)

	// 初始化 JWT 服务
	s.jwtService = service.NewJWTService()

	s.serviceTreeService = service.NewServiceTreeService(serviceTreeRepo, appRepo, s.appRuntime)

	// 初始化函数服务（需要更多依赖）
	s.functionService = service.NewFunctionService(functionRepo, sourceCodeRepo, appRepo, serviceTreeRepo, s.appRuntime, s.appService)

	// 初始化用户服务
	s.userService = service.NewUserService(userRepo)

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

// GetAuthService 获取认证服务
func (s *Server) GetAuthService() *service.AuthService {
	return s.authService
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

// handleFunctionGenResult 处理函数生成结果消息
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
		ctx = context.WithValue(ctx, "request_user", requestUser)
	}

	// 解析消息体
	var result dto.FunctionGenResult
	if err := json.Unmarshal(msg.Data, &result); err != nil {
		logger.Errorf(ctx, "[Server] Failed to unmarshal function gen result: %v, Data: %s", err, string(msg.Data))
		return
	}

	// 打印日志（先保证流程通了）
	logger.Infof(ctx, "[Server] Received function generation result:")
	logger.Infof(ctx, "[Server]   RecordID: %d", result.RecordID)
	logger.Infof(ctx, "[Server]   AgentID: %d", result.AgentID)
	logger.Infof(ctx, "[Server]   TreeID: %d", result.TreeID)
	logger.Infof(ctx, "[Server]   User: %s", result.User)
	logger.Infof(ctx, "[Server]   Code length: %d bytes", len(result.Code))
	logger.Infof(ctx, "[Server]   TraceID: %s", traceID)
	logger.Infof(ctx, "[Server]   RequestUser: %s", requestUser)

	// TODO: 后续添加解析代码和创建函数的逻辑
	// 1. 解析生成的代码
	// 2. 在指定的服务目录（TreeID）下创建函数
	// 3. 更新应用状态
}
