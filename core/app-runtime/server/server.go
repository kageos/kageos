package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	v1 "github.com/ai-agent-os/ai-agent-os/core/app-runtime/api/v1"
	"github.com/ai-agent-os/ai-agent-os/core/app-runtime/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-runtime/repository"
	"github.com/ai-agent-os/ai-agent-os/core/app-runtime/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/builder"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/dbx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/natsx"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
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
	dbPath, err := resolveRuntimeDBPath()
	if err != nil {
		return fmt.Errorf("failed to resolve runtime db path: %w", err)
	}

	// 获取绝对路径
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	db, err := dbx.OpenSQLite(absPath, dbx.OpenOptions{})
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

func resolveRuntimeDBPath() (string, error) {
	root := config.GetAgentOSRoot()
	dataRoot := "data"
	if root != "" {
		dataRoot = filepath.Join(root, "data")
	}

	currentPath := filepath.Join(dataRoot, "runtime", "app-runtime", "app_runtime.db")
	legacyPath := filepath.Join(dataRoot, "app-runtime", "app_runtime.db")

	if err := migrateLegacyRuntimeDB(currentPath, legacyPath); err != nil {
		return "", err
	}

	return currentPath, nil
}

func migrateLegacyRuntimeDB(currentPath, legacyPath string) error {
	if _, err := os.Stat(currentPath); err == nil {
		return nil
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	if _, err := os.Stat(legacyPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if err := os.MkdirAll(filepath.Dir(currentPath), 0o755); err != nil {
		return fmt.Errorf("failed to create runtime db directory: %w", err)
	}
	if err := os.Rename(legacyPath, currentPath); err != nil {
		return fmt.Errorf("failed to migrate runtime db from %s to %s: %w", legacyPath, currentPath, err)
	}

	_ = os.Remove(filepath.Dir(legacyPath))
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

	natsConfig := config.GetGlobalSharedConfig().Nats
	conn, err := natsx.Connect(natsConfig.URL)
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
	runtimeID := s.cfg.GetRuntimeInstanceID()
	s.appDiscoveryService = service.NewAppDiscoveryServiceWithRuntimeID(s.natsConn, s.cfg.AppManage.AppDir.BasePath, runtimeID)
	logger.Infof(ctx, "[Server] App discovery runtime_id=%s", runtimeID)

	// 设置回调函数
	s.appDiscoveryService.SetCallbacks(
		s.handleAppStartupFromDiscovery,
		s.handleAppCloseFromDiscovery,
	)

	if err := s.appDiscoveryService.Start(); err != nil {
		return fmt.Errorf("failed to start app discovery service: %w", err)
	}

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

	// 启动基础设施看门狗（以 NATS 连接状态为探针，1 秒轮询，断开时触发恢复）
	watchdog := service.NewInfraWatchdog(s.natsConn, s.containerService)
	watchdog.SetOnRecovered(s.reconcileAppContainers)
	go watchdog.Start(ctx)

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

// subscribeNATS 订阅所有 NATS 主题（Gin 风格：api/v1 放 handler，router 里注册）
func (s *Server) subscribeNATS(ctx context.Context) error {
	appH := v1.NewAppHandler(s.appManageService)
	serviceTreeH := v1.NewServiceTreeHandler(s.serviceTreeService)
	workspaceH := v1.NewWorkspaceHandler(s.appManageService)
	requestTransport := NewAppRequestTransport(s.natsConn, s.containerService, s.appManageService, s.appDiscoveryService)
	requestH := v1.NewRequestHandler(s.appManageService, requestTransport)
	if err := RegisterNATS(ctx, s.natsConn, &s.subscriptions, appH, serviceTreeH, workspaceH, requestH); err != nil {
		return err
	}
	// Runtime 生命周期事件主题由 AppDiscoveryService 统一处理，不需要在此订阅
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

// reconcileAppContainers 对账应用容器
// Podman 重启后，内存中标记为 running 的应用容器实际已停止。
// 遍历内存中 running 的应用，检查容器是否真的在跑，没跑就拉起来。
func (s *Server) reconcileAppContainers(ctx context.Context) {
	runningApps := s.appDiscoveryService.GetRunningApps()
	if len(runningApps) == 0 {
		logger.Infof(ctx, "[Reconcile] 内存中无 running 应用，跳过对账")
		return
	}

	logger.Infof(ctx, "[Reconcile] 开始对账应用容器 | 内存中 running 应用=%d", len(runningApps))

	restarted := 0
	alreadyRunning := 0
	failed := 0

	for appKey, appInfo := range runningApps {
		for _, version := range appInfo.Versions {
			containerName := service.BuildContainerName(appInfo.User, appInfo.App, version.Version)

			running, err := s.containerService.IsContainerRunning(ctx, containerName)
			if err != nil {
				logger.Warnf(ctx, "[Reconcile] 无法检查容器 %s: %v", containerName, err)
				failed++
				continue
			}

			if running {
				alreadyRunning++
				continue
			}

			// 容器没在跑，拉起来
			logger.Infof(ctx, "[Reconcile] 重启应用容器 | 应用=%s | 版本=%s | 容器=%s",
				appKey, version.Version, containerName)
			startTime := time.Now()

			if err := s.containerService.StartContainer(ctx, containerName); err != nil {
				logger.Warnf(ctx, "[Reconcile] ❌ 重启失败 | 容器=%s | 错误=%v", containerName, err)
				failed++
			} else {
				logger.Infof(ctx, "[Reconcile] ✅ 重启成功 | 容器=%s | 耗时=%s",
					containerName, time.Since(startTime).Round(time.Millisecond))
				restarted++
			}
		}
	}

	logger.Infof(ctx, "[Reconcile] 应用容器对账完成 | 重启=%d | 已在运行=%d | 失败=%d",
		restarted, alreadyRunning, failed)
}
