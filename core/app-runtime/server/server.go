package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"

	"github.com/ai-agent-os/ai-agent-os/core/app-runtime/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-runtime/repository"
	"github.com/ai-agent-os/ai-agent-os/core/app-runtime/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/builder"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/nats-io/nats.go"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// Server app-runtime 服务器
// 负责管理所有服务的生命周期和依赖关系
type Server struct {
	cfg *config.AppRuntimeConfig

	// 基础设施
	natsConn *nats.Conn
	db       *gorm.DB

	// 业务服务
	containerService    service.ContainerOperator
	appManageService    *service.AppManageService
	appDiscoveryService *service.AppDiscoveryService
	serviceTreeService  *service.ServiceTreeService
	forkService         *service.ForkService

	// HTTP 健康检查服务器
	httpServer *http.Server

	// NATS 订阅
	subscriptions []*nats.Subscription
}

// NewServer 创建服务器实例
func NewServer(cfg *config.AppRuntimeConfig) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	s := &Server{
		cfg:           cfg,
		subscriptions: make([]*nats.Subscription, 0),
	}

	return s, nil
}

// Start 启动服务器
func (s *Server) Start(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Starting app-runtime server...")

	// 按依赖顺序启动各个组件
	if err := s.initDatabase(ctx); err != nil {
		return fmt.Errorf("failed to init database: %w", err)
	}

	if err := s.initNATS(ctx); err != nil {
		return fmt.Errorf("failed to init NATS: %w", err)
	}

	if err := s.initServices(ctx); err != nil {
		return fmt.Errorf("failed to init services: %w", err)
	}

	if err := s.subscribeNATS(ctx); err != nil {
		return fmt.Errorf("failed to subscribe NATS: %w", err)
	}

	if err := s.startHTTP(ctx); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	return nil
}

// Stop 停止服务器（优雅关闭）
func (s *Server) Stop(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Stopping app-runtime server...")

	// 反向顺序关闭资源
	// HTTP server 已经不需要特殊关闭，端口会自动释放

	s.unsubscribeNATS(ctx)
	s.stopServices(ctx)
	s.closeNATS(ctx)
	s.closeDatabase(ctx)

	logger.Infof(ctx, "[Server] App-runtime server stopped")
	return nil
}

// initDatabase 初始化数据库
func (s *Server) initDatabase(ctx context.Context) error {
	// 数据库文件路径
	dbPath := filepath.Join("data", "app-runtime", "app_runtime.db")

	// 获取绝对路径
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// 确保目录存在
	dbDir := filepath.Dir(absPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// 连接数据库
	db, err := gorm.Open(sqlite.Open(absPath), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Info),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// 自动迁移表结构
	if err := model.InitTables(db); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	s.db = db
	return nil
}

// closeDatabase 关闭数据库连接
func (s *Server) closeDatabase(ctx context.Context) {
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err == nil {
			if err := sqlDB.Close(); err != nil {
				logger.Errorf(ctx, "[Server] Failed to close database: %v", err)
			} else {
				logger.Infof(ctx, "[Server] Database closed")
			}
		}
	}
}

// initNATS 初始化 NATS 连接
func (s *Server) initNATS(ctx context.Context) error {

	conn, err := nats.Connect(s.cfg.Nats.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	s.natsConn = conn
	return nil
}

// closeNATS 关闭 NATS 连接
func (s *Server) closeNATS(ctx context.Context) {
	if s.natsConn != nil {
		s.natsConn.Close()
		logger.Infof(ctx, "[Server] NATS connection closed")
	}
}

// initServices 初始化所有业务服务
func (s *Server) initServices(ctx context.Context) error {

	// 初始化容器服务
	s.containerService = service.NewDefaultContainerOperator()
	if err := s.containerService.Start(ctx); err != nil {
		return fmt.Errorf("failed to start container service: %w", err)
	}

	// 初始化应用仓库
	appRepo := repository.NewAppRepository(s.db)

	// 初始化应用发现服务（需要在 AppManageService 之前）
	s.appDiscoveryService = service.NewAppDiscoveryService(s.natsConn, s.cfg.AppManage.AppDir.BasePath)

	// 设置回调函数
	s.appDiscoveryService.SetCallbacks(
		s.handleAppStartupFromDiscovery,
		s.handleAppCloseFromDiscovery,
	)

	if err := s.appDiscoveryService.Start(); err != nil {
		return fmt.Errorf("failed to start app discovery service: %w", err)
	}

	// 初始化 Fork 服务（需要在 AppManageService 之前）
	s.forkService = service.NewForkService(&s.cfg.AppManage)

	// 初始化创建函数服务（需要在 AppManageService 之前）
	createFunctionService := service.NewCreateFunctionService(&s.cfg.AppManage)

	// 初始化应用管理服务
	wd, _ := os.Getwd()
	s.appManageService = service.NewAppManageService(
		builder.NewBuilder(wd),
		&s.cfg.AppManage,
		s.cfg, // 传入完整的运行时配置（用于获取网关地址等）
		s.containerService,
		appRepo,
		s.appDiscoveryService,
		s.natsConn,
		s.forkService,         // 传入 Fork 服务
		createFunctionService, // 传入创建函数服务
	)

	// 启动 QPS 跟踪器清理任务
	go s.appManageService.QPSTracker.StartCleanup(ctx)

	// 启动应用清理任务
	go s.appManageService.StartCleanupTask(ctx)

	// 初始化服务目录管理服务
	s.serviceTreeService = service.NewServiceTreeService(&s.cfg.AppManage)
	// 设置依赖关系
	s.serviceTreeService.SetAppManageService(s.appManageService)

	return nil
}

// handleAppStartupFromDiscovery 处理来自 AppDiscoveryService 的启动通知
func (s *Server) handleAppStartupFromDiscovery(user, app, version string, startTime time.Time) {
	//ctx := context.Background()
	//logger.Infof(ctx, "[Server] Received startup notification from discovery: %s/%s/%s", user, app, version)

	// 构建通知对象
	notification := &service.StartupNotification{
		User:      user,
		App:       app,
		Version:   version,
		Status:    "started",
		StartTime: startTime,
	}

	// 通知应用管理服务
	s.appManageService.NotifyStartup(notification)
}

// handleAppCloseFromDiscovery 处理来自 AppDiscoveryService 的关闭通知
func (s *Server) handleAppCloseFromDiscovery(user, app, version string) {
	ctx := context.Background()

	// 应用关闭状态通过discovery service跟踪，不需要更新数据库
	logger.Infof(ctx, "[Server] App closed: %s/%s/%s", user, app, version)

	// 构建关闭通知对象
	notification := &service.CloseNotification{
		User:      user,
		App:       app,
		Version:   version,
		CloseTime: time.Now(),
	}

	// 通知应用管理服务（用于优雅关闭流程的第三次握手）
	s.appManageService.NotifyClose(notification)
}

// stopServices 停止所有业务服务
func (s *Server) stopServices(ctx context.Context) {
	if s.appDiscoveryService != nil {
		s.appDiscoveryService.Stop()
		logger.Infof(ctx, "[Server] App discovery service stopped")
	}
}

// subscribeNATS 订阅所有 NATS 主题
func (s *Server) subscribeNATS(ctx context.Context) error {

	// 暂时导入 subject 包以使用现有的 handler
	// TODO: 后续可以考虑将 handler 改为 Server 的方法
	var err error
	var sub *nats.Subscription

	// 订阅应用创建请求（使用队列组）
	sub, err = s.natsConn.QueueSubscribe(
		subjects.GetAppRuntime2AppCreateRequestSubject(),
		"app-runtime-create-workers",
		s.handleAppCreate,
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to app create: %w", err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	// 订阅应用更新请求（使用队列组）
	sub, err = s.natsConn.QueueSubscribe(
		subjects.GetAppRuntime2AppUpdateRequestSubject(),
		"app-runtime-update-workers",
		s.handleAppUpdate,
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to app update: %w", err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	// 订阅服务目录创建请求（使用队列组）
	sub, err = s.natsConn.QueueSubscribe(
		subjects.GetAppRuntime2ServiceTreeCreateRequestSubject(),
		"app-runtime-service-tree-workers",
		s.handleServiceTreeCreate,
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to service tree create: %w", err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	// 订阅来自 app-server 的请求（使用队列组）
	sub, err = s.natsConn.QueueSubscribe(
		subjects.GetFunctionServer2AppRuntimeRequestSubject(),
		"app-runtime-request-workers",
		s.handleFunctionServerRequest,
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to function server request: %w", err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	//// 订阅应用发现响应
	//sub, err = s.natsConn.Subscribe(
	//	subjects.GetAppDiscoveryResponseSubject(),
	//	s.handleAppDiscoveryResponse,
	//)
	//if err != nil {
	//	return fmt.Errorf("failed to subscribe to app discovery response: %w", err)
	//}
	//s.subscriptions = append(s.subscriptions, sub)

	// 订阅应用删除请求（使用队列组）
	sub, err = s.natsConn.QueueSubscribe(
		subjects.GetAppServer2AppRuntimeDeleteRequestSubject(),
		"app-runtime-delete-workers",
		s.handleAppDelete,
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to app delete: %w", err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	// 订阅读取目录文件请求（使用队列组）
	sub, err = s.natsConn.QueueSubscribe(
		subjects.GetAppServer2AppRuntimeReadDirectoryFilesRequestSubject(),
		"app-runtime-read-directory-files-workers",
		s.handleReadDirectoryFiles,
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to read directory files: %w", err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	// 订阅批量创建目录树请求（使用队列组）
	sub, err = s.natsConn.QueueSubscribe(
		subjects.GetAppServer2AppRuntimeBatchCreateDirectoryTreeRequestSubject(),
		"app-runtime-batch-create-directory-tree-workers",
		s.handleBatchCreateDirectoryTree,
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to batch create directory tree: %w", err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	// 订阅更新服务树请求（使用队列组）
	sub, err = s.natsConn.QueueSubscribe(
		subjects.GetAppServer2AppRuntimeUpdateServiceTreeRequestSubject(),
		"app-runtime-update-service-tree-workers",
		s.handleUpdateServiceTree,
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to update service tree: %w", err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	// 订阅批量写文件请求（使用队列组）
	sub, err = s.natsConn.QueueSubscribe(
		subjects.GetAppServer2AppRuntimeBatchWriteFilesRequestSubject(),
		"app-runtime-batch-write-files-workers",
		s.handleBatchWriteFiles,
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to batch write files: %w", err)
	}
	s.subscriptions = append(s.subscriptions, sub)

	// Runtime 状态主题由 AppDiscoveryService 统一处理，不需要重复订阅

	// 旧的订阅已移除，现在通过 runtime.status 主题统一处理

	return nil
}

// unsubscribeNATS 取消所有 NATS 订阅
func (s *Server) unsubscribeNATS(ctx context.Context) {
	for _, sub := range s.subscriptions {
		if err := sub.Unsubscribe(); err != nil {
			logger.Warnf(ctx, "[Server] Failed to unsubscribe: %v", err)
		}
	}
	logger.Infof(ctx, "[Server] All NATS subscriptions closed")
}

// startHTTP 检测是否重复启动（通过端口占用检测）
// 如果端口已被占用，说明已有实例运行，直接 panic
func (s *Server) startHTTP(ctx context.Context) error {
	port := fmt.Sprintf(":%d", s.cfg.Runtime.Port)

	// 尝试监听端口，如果失败说明已有实例运行
	listener, err := net.Listen("tcp", port)
	if err != nil {
		logger.Errorf(ctx, "[Server] Port %s already in use, another instance is running", port)
		panic(fmt.Sprintf("启动失败：端口 %s 已被占用，可能有其他实例正在运行", port))
	}

	// 保持端口监听，作为实例运行的标识
	// 当进程退出时，端口会自动释放

	// 将 listener 保存到 httpServer 的 Addr 字段，用于后续关闭
	s.httpServer = &http.Server{
		Addr: port,
	}

	// 启动一个 goroutine 保持监听
	go func() {
		// 简单接受连接但不处理，只是占住端口
		for {
			conn, err := listener.Accept()
			if err != nil {
				// listener 关闭时会返回错误，正常退出
				return
			}
			// 立即关闭连接
			conn.Close()
		}
	}()

	return nil
}

// GetAppManageService 获取应用管理服务（供 NATS handler 使用）
func (s *Server) GetAppManageService() *service.AppManageService {
	return s.appManageService
}

// GetAppDiscoveryService 获取应用发现服务
func (s *Server) GetAppDiscoveryService() *service.AppDiscoveryService {
	return s.appDiscoveryService
}

// GetNatsConn 获取 NATS 连接
func (s *Server) GetNatsConn() *nats.Conn {
	return s.natsConn
}

// GetDB 获取数据库连接
func (s *Server) GetDB() *gorm.DB {
	return s.db
}

// GetServiceTreeService 获取服务目录管理服务
func (s *Server) GetServiceTreeService() *service.ServiceTreeService {
	return s.serviceTreeService
}
