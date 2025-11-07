package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-runtime/dto"

	"github.com/ai-agent-os/ai-agent-os/core/app-runtime/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-runtime/repository"
	appPkg "github.com/ai-agent-os/ai-agent-os/pkg/app"
	"github.com/ai-agent-os/ai-agent-os/pkg/builder"
	appconfig "github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// CreateOpts 创建选项
type CreateOpts struct {
	Env     map[string]string
	Volumes []string
}

// BuildOpts 编译选项
type BuildOpts struct {
	SourceDir        string            // 源代码目录
	OutputDir        string            // 输出目录
	Platform         string            // 目标平台
	BinaryNameFormat string            // 二进制文件名格式
	BuildTags        []string          // 编译标签
	LdFlags          []string          // 链接参数
	Env              map[string]string // 编译环境变量
}

// StartupNotification 启动通知
type StartupNotification struct {
	User      string
	App       string
	Version   string
	Status    string
	StartTime time.Time
}

// CloseNotification 关闭通知
type CloseNotification struct {
	User      string
	App       string
	Version   string
	CloseTime time.Time
}

// AppManageService 应用管理服务 - 负责应用的增删改查
type AppManageService struct {
	builder             *builder.Builder
	config              *appconfig.AppManageServiceConfig
	runtimeConfig       *appconfig.AppRuntimeConfig // 运行时完整配置（用于获取网关地址等）
	containerService    ContainerOperator           // 容器服务依赖
	appRepo             *repository.AppRepository   // 应用数据访问层
	appDiscoveryService *AppDiscoveryService        // 应用发现服务，用于获取运行状态
	natsConn            *nats.Conn                  // NATS 连接，用于发送关闭命令
	QPSTracker          *QPSTracker                 // QPS 跟踪器

	// 启动等待器 - 用于等待应用启动完成通知
	startupWaiters   map[string]chan *StartupNotification // key: user/app/version
	startupWaitersMu sync.RWMutex

	// 关闭等待器 - 用于等待应用关闭完成通知
	closeWaiters   map[string]chan *CloseNotification // key: user/app/version
	closeWaitersMu sync.RWMutex

	// 定时任务控制
	cleanupTicker *time.Ticker
	cleanupDone   chan struct{}
}

// ============================================================================
// 容器名工具函数
// ============================================================================

// BuildContainerName 构建容器名（新格式：{user}-{app}-{version}）
// 公开函数，供其他包使用
func BuildContainerName(user, app, version string) string {
	return fmt.Sprintf("%s-%s-%s", user, app, version)
}

// buildContainerName 构建容器名（内部使用，调用公开函数）
func buildContainerName(user, app, version string) string {
	return BuildContainerName(user, app, version)
}

// parseContainerName 解析容器名（格式：{user}-{app}-{version}）
// 返回：user, app, version, error
func parseContainerName(containerName string) (string, string, string, error) {
	parts := strings.Split(containerName, "-")
	if len(parts) < 3 {
		return "", "", "", fmt.Errorf("invalid container name format: %s, expected {user}-{app}-{version}", containerName)
	}
	// 最后一部分是 version
	version := parts[len(parts)-1]
	// 前面是 user-app（假设 user 和 app 都不包含连字符）
	user := parts[0]
	app := strings.Join(parts[1:len(parts)-1], "-")
	return user, app, version, nil
}

// ============================================================================
// 启动等待器管理方法
// ============================================================================

// registerStartupWaiter 注册启动等待器
func (s *AppManageService) registerStartupWaiter(user, app, version string) chan *StartupNotification {
	key := fmt.Sprintf("%s/%s/%s", user, app, version)
	s.startupWaitersMu.Lock()
	defer s.startupWaitersMu.Unlock()

	waiterChan := make(chan *StartupNotification, 1)
	s.startupWaiters[key] = waiterChan
	return waiterChan
}

// unregisterStartupWaiter 注销启动等待器
func (s *AppManageService) unregisterStartupWaiter(user, app, version string) {
	key := fmt.Sprintf("%s/%s/%s", user, app, version)
	s.startupWaitersMu.Lock()
	defer s.startupWaitersMu.Unlock()

	if waiterChan, exists := s.startupWaiters[key]; exists {
		close(waiterChan)
		delete(s.startupWaiters, key)
	}
}

// notifyStartupWaiter 通知启动等待器
func (s *AppManageService) notifyStartupWaiter(user, app, version string, notification *StartupNotification) {
	key := fmt.Sprintf("%s/%s/%s", user, app, version)
	s.startupWaitersMu.RLock()
	waiterChan, exists := s.startupWaiters[key]
	s.startupWaitersMu.RUnlock()

	if exists {
		select {
		case waiterChan <- notification:
		default:
		}
	}
}

// NewAppManageService 创建应用管理服务（依赖注入）
func NewAppManageService(builder *builder.Builder, config *appconfig.AppManageServiceConfig, runtimeConfig *appconfig.AppRuntimeConfig, containerService ContainerOperator, appRepo *repository.AppRepository, appDiscoveryService *AppDiscoveryService, natsConn *nats.Conn) *AppManageService {
	return &AppManageService{
		builder:             builder,
		config:              config,
		runtimeConfig:       runtimeConfig,
		containerService:    containerService,
		appRepo:             appRepo,
		appDiscoveryService: appDiscoveryService,
		natsConn:            natsConn,
		QPSTracker:          NewQPSTracker(60*time.Second, 10*time.Second), // 60秒窗口，10秒检查间隔
		startupWaiters:      make(map[string]chan *StartupNotification),
		closeWaiters:        make(map[string]chan *CloseNotification),
		cleanupDone:         make(chan struct{}),
	}
}

// ============================================================================
// 关闭等待器管理方法
// ============================================================================

// registerCloseWaiter 注册关闭等待器
func (s *AppManageService) registerCloseWaiter(user, app, version string) chan *CloseNotification {
	key := fmt.Sprintf("%s/%s/%s", user, app, version)
	s.closeWaitersMu.Lock()
	defer s.closeWaitersMu.Unlock()

	waiterChan := make(chan *CloseNotification, 1)
	s.closeWaiters[key] = waiterChan
	return waiterChan
}

// unregisterCloseWaiter 注销关闭等待器
func (s *AppManageService) unregisterCloseWaiter(user, app, version string) {
	key := fmt.Sprintf("%s/%s/%s", user, app, version)
	s.closeWaitersMu.Lock()
	defer s.closeWaitersMu.Unlock()

	if waiterChan, exists := s.closeWaiters[key]; exists {
		close(waiterChan)
		delete(s.closeWaiters, key)
	}
}

// notifyCloseWaiter 通知关闭等待器
func (s *AppManageService) notifyCloseWaiter(user, app, version string, notification *CloseNotification) {
	key := fmt.Sprintf("%s/%s/%s", user, app, version)
	s.closeWaitersMu.RLock()
	waiterChan, exists := s.closeWaiters[key]
	s.closeWaitersMu.RUnlock()

	if exists {
		select {
		case waiterChan <- notification:
		default:
			// 通道已满或已关闭，忽略
		}
	}
}

// NotifyStartup 通知应用启动完成（由 NATS 消息处理器调用）
func (s *AppManageService) NotifyStartup(notification *StartupNotification) {
	s.notifyStartupWaiter(notification.User, notification.App, notification.Version, notification)
}

// NotifyClose 通知应用关闭完成（由 NATS 消息处理器调用）
func (s *AppManageService) NotifyClose(notification *CloseNotification) {
	s.notifyCloseWaiter(notification.User, notification.App, notification.Version, notification)
}

// RegisterStartupWaiter 注册启动等待器
func (s *AppManageService) RegisterStartupWaiter(key string) {
	s.startupWaitersMu.Lock()
	defer s.startupWaitersMu.Unlock()

	// 如果已存在，不重复创建
	if _, exists := s.startupWaiters[key]; !exists {
		s.startupWaiters[key] = make(chan *StartupNotification, 1)
	}
}

// UnregisterStartupWaiter 注销启动等待器
func (s *AppManageService) UnregisterStartupWaiter(key string) {
	s.startupWaitersMu.Lock()
	defer s.startupWaitersMu.Unlock()

	delete(s.startupWaiters, key)
}

// GetStartupWaiter 获取启动等待器
func (s *AppManageService) GetStartupWaiter(key string) chan *StartupNotification {
	s.startupWaitersMu.RLock()
	defer s.startupWaitersMu.RUnlock()

	return s.startupWaiters[key]
}

// waitForStartup 等待应用启动完成（内部方法）
func (s *AppManageService) waitForStartup(ctx context.Context, user, app, version string, timeout time.Duration) (*StartupNotification, error) {
	// 使用统一的等待器注册方法
	waiterChan := s.registerStartupWaiter(user, app, version)
	// 确保在方法结束时清理等待器
	defer s.unregisterStartupWaiter(user, app, version)

	//logger.Infof(ctx, "[waitForStartup] Waiting for: %s/%s/%s (timeout: %v)", user, app, version, timeout)

	select {
	case notification := <-waiterChan:
		return notification, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for startup notification")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// CreateApp 创建应用目录结构
func (s *AppManageService) CreateApp(ctx context.Context, user, app string, opts ...*CreateOpts) (string, error) {
	logger.Infof(ctx, "[CreateApp] *** ENTRY *** user=%s, app=%s", user, app)

	// 1. 获取应用目录的绝对路径（使用配置中的基础路径）
	appDirRel := filepath.Join(s.config.AppDir.BasePath, user, app)
	absAppDir, err := filepath.Abs(appDirRel)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// 2. 定义完整的目录结构（使用配置中的结构）
	dirs := []string{
		// 应用根目录
		absAppDir,
	}

	// 添加配置中定义的目录结构
	for _, dir := range s.config.AppDir.Structure {
		dirs = append(dirs, filepath.Join(absAppDir, dir))
	}

	// 3. 创建所有目录
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// 4. 启动脚本已内置在 ai-agent-os 镜像中，无需复制

	// 5. 创建应用时不创建版本文件，版本文件将在第一次编译时创建
	// 这样可以避免创建时就写入版本信息
	logger.Infof(ctx, "[CreateApp] Skipping version files creation, will be created on first build")

	// 6. 创建 main.go 文件
	mainGoPath := filepath.Join(absAppDir, "code/cmd/app/main.go")
	if err := s.createMainGoFile(mainGoPath, user, app); err != nil {
		return "", fmt.Errorf("failed to create main.go file: %w", err)
	}

	// 8. 保存应用信息到数据库
	if err := s.appRepo.CreateApp(user, app); err != nil {
		return "", fmt.Errorf("failed to create app in database: %w", err)
	}

	// 9. 创建应用时不编译和启动，节省资源
	// 编译和启动将在 UpdateApp 时进行
	logger.Infof(ctx, "[CreateApp] App directory structure created successfully, skipping build and container start to save resources")

	logger.Infof(ctx, "[CreateApp] *** EXIT *** user=%s, app=%s, appDir=%s", user, app, appDirRel)
	return appDirRel, nil
}

// BuildApp 编译应用
func (s *AppManageService) BuildApp(ctx context.Context, user, app string, opts ...*BuildOpts) (*builder.BuildResult, error) {
	//logger.Infof(ctx, "[BuildApp] *** ENTRY *** user=%s, app=%s", user, app)

	// 设置默认编译选项（使用配置中的平台和格式）
	buildOpts := &builder.BuildOpts{
		Platform:         s.config.Build.Platform,
		BinaryNameFormat: s.config.Build.BinaryNameFormat,
	}

	if opts != nil {
		opt := opts[0]
		// 转换类型，保留所有字段
		buildOpts = &builder.BuildOpts{
			User:             user, // 设置用户
			App:              app,  // 设置应用
			SourceDir:        opt.SourceDir,
			OutputDir:        opt.OutputDir,
			Platform:         opt.Platform,
			BinaryNameFormat: opt.BinaryNameFormat,
			BuildTags:        opt.BuildTags,
			LdFlags:          opt.LdFlags,
			Env:              opt.Env,
		}
	}

	// 执行编译
	result, err := s.builder.Build(ctx, user, app, buildOpts)
	if err != nil {
		logger.Errorf(ctx, "[BuildApp] *** FAILED *** user=%s, app=%s, error=%v", user, app, err)
		return nil, err
	}

	return result, nil
}

// ListApps 列出所有应用
func (s *AppManageService) ListApps(ctx context.Context, user string) ([]string, error) {
	// TODO: 实现列出应用逻辑
	userDir := fmt.Sprintf("namespace/%s", user)
	entries, err := os.ReadDir(userDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read user directory: %w", err)
	}

	var apps []string
	for _, entry := range entries {
		if entry.IsDir() {
			apps = append(apps, entry.Name())
		}
	}
	return apps, nil
}

// DeleteApp 删除应用
func (s *AppManageService) DeleteApp(ctx context.Context, user, app string) error {
	logger.Infof(ctx, "[DeleteApp] *** ENTRY *** user=%s, app=%s", user, app)

	// 1. 停止并删除容器
	containerName := fmt.Sprintf("%s-%s", user, app)
	if s.containerService != nil {

		// 先尝试停止容器（如果正在运行）
		if err := s.containerService.StopContainer(ctx, containerName); err != nil {
			logger.Warnf(ctx, "[DeleteApp] Failed to stop container %s (may not be running): %v", containerName, err)
		} else {
			logger.Infof(ctx, "[DeleteApp] Container %s stopped successfully", containerName)
		}

		// 强制删除容器（无论是否正在运行）
		if err := s.containerService.RemoveContainer(ctx, containerName); err != nil {
			logger.Errorf(ctx, "[DeleteApp] Failed to remove container %s: %v", containerName, err)
			return fmt.Errorf("failed to remove container %s: %w", containerName, err)
		} else {
			logger.Infof(ctx, "[DeleteApp] Container %s removed successfully", containerName)
		}
	} else {
		logger.Warnf(ctx, "[DeleteApp] Container operator is nil, skipping container deletion")
	}

	// 2. 删除应用目录
	appDirRel := filepath.Join(s.config.AppDir.BasePath, user, app)
	absAppDir, err := filepath.Abs(appDirRel)
	if err != nil {
		logger.Warnf(ctx, "[DeleteApp] Failed to get absolute path: %v", err)
	} else {
		if err := os.RemoveAll(absAppDir); err != nil {
			logger.Warnf(ctx, "[DeleteApp] Failed to remove app directory %s: %v", absAppDir, err)
		} else {
			logger.Infof(ctx, "[DeleteApp] App directory removed: %s", absAppDir)
		}
	}

	// 3. 删除数据库记录
	if err := s.appRepo.DeleteAppAndVersions(user, app); err != nil {
		logger.Warnf(ctx, "[DeleteApp] Failed to delete app and versions from database: %v", err)
	}

	logger.Infof(ctx, "[DeleteApp] *** EXIT *** user=%s, app=%s", user, app)
	return nil
}

// UpdateApp 更新应用（重新编译并重启容器）
func (s *AppManageService) UpdateApp(ctx context.Context, user, app string) (*dto.UpdateAppResp, error) {

	logStr := strings.Builder{}
	logStr.WriteString(fmt.Sprintf("[UpdateApp] Starting update: %s/%s\t", user, app))

	// 1. 获取当前版本
	appDirRel := filepath.Join(s.config.AppDir.BasePath, user, app)
	absAppDir, err := filepath.Abs(appDirRel)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// 检查应用是否存在
	if _, err := os.Stat(absAppDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("app not found: %s/%s", user, app)
	}

	// 2. 查询应用状态，判断是否为未激活状态
	appRecord, err := s.appRepo.GetApp(user, app)
	if err != nil {
		return nil, fmt.Errorf("failed to get app record: %w", err)
	}

	isInactive := appRecord.IsInactive()
	logStr.WriteString(fmt.Sprintf("App status: %s\t", appRecord.Status))

	// 3. 检查应用是否正在运行
	isRunning, err := s.IsAppRunning(ctx, user, app)
	if err != nil {
		logStr.WriteString(fmt.Sprintf("Failed to check app running status: %v\t", err))
		isRunning = false
	}

	// 4. 使用 VersionManager 获取当前版本
	vm := appPkg.NewVersionManager(filepath.Join(s.config.AppDir.BasePath, user), app)
	oldVersion, err := vm.GetCurrentVersion()
	if err != nil {
		logStr.WriteString(fmt.Sprintf("Failed to get current version: %v\t", err))
		oldVersion = "unknown"
	}

	// 2. 重新编译应用（Builder 会自动生成新版本号）
	sourceDir := filepath.Join(absAppDir, "code/cmd/app")
	outputDir := filepath.Join(absAppDir, s.config.Build.OutputDir)

	buildOpts := &BuildOpts{
		SourceDir:        sourceDir,
		OutputDir:        outputDir,
		Platform:         s.config.Build.Platform,
		BinaryNameFormat: s.config.Build.BinaryNameFormat,
	}

	buildResult, err := s.BuildApp(ctx, user, app, buildOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to build app: %w", err)
	}

	newVersion := buildResult.Version

	// 4. 更新或创建 version.json 文件
	metadataDir := filepath.Join(absAppDir, "workplace/metadata")
	versionFile := filepath.Join(metadataDir, "version.json")

	// 检查版本文件是否存在，如果不存在则创建
	if _, err := os.Stat(versionFile); os.IsNotExist(err) {
		logger.Infof(ctx, "[UpdateApp] Version file not found, creating initial version file...")
		if err := s.createVersionFiles(metadataDir, user, app); err != nil {
			return nil, fmt.Errorf("failed to create initial version files: %w", err)
		}
	}

	// 更新版本信息
	if err := s.updateVersionJson(absAppDir, user, app, newVersion); err != nil {
		return nil, fmt.Errorf("failed to update version.json: %w", err)
	}

	// 5. 根据应用运行状态决定启动方式
	// 注意：这里暂时保留旧格式，后续重构时会改为新格式
	// 新格式：containerName := buildContainerName(user, app, newVersion)
	containerName := fmt.Sprintf("%s-%s", user, app)

	if s.containerService == nil {
		return nil, fmt.Errorf("container operator not available")
	}

	// 统一在外层注册启动等待器，因为无论哪种启动方式都需要等待
	// 先注册等待器，再执行启动命令，避免错过通知
	waiterChan := s.registerStartupWaiter(user, app, newVersion)
	// 确保在方法结束时清理等待器
	defer s.unregisterStartupWaiter(user, app, newVersion)

	if isRunning {
		// 应用正在运行：在容器内启动新版本（灰度发布）
		//logger.Infof(ctx, "[UpdateApp] Starting new version in container for gray deployment: %s", containerName)

		// 构建新版本的二进制文件名
		binaryName := fmt.Sprintf("%s_%s_%s", user, app, newVersion)

		// ====================================================================================
		// 为什么使用 setsid nohup 而不是简单的 & 后台运行？
		// ====================================================================================
		//
		// 问题背景：
		// 当使用 podman exec 在容器内启动进程时：
		//   podman exec container /bin/sh -c "cd /app && ./app &"
		//
		// 进程关系：
		//   PID 1: 容器主进程 (./releases/beiluo_aaa_v1)
		//     └─ sh (podman exec 创建的临时 shell)
		//         └─ ./app (后台进程)
		//
		// 问题 1 - 僵尸进程：
		//   - podman exec 执行完命令后，sh 进程会立即退出
		//   - ./app 的父进程 sh 退出后，./app 成为孤儿进程，被 PID 1 接管
		//   - 但容器的 PID 1 是应用程序（不是 init 系统），不会调用 wait() 回收子进程
		//   - 当 ./app 退出时，就会变成僵尸进程 [app] <defunct>
		//
		// 问题 2 - 管道依赖：
		//   - 简单的 & 后台运行，进程的 stdin/stdout/stderr 仍连接到 podman exec 的管道
		//   - 管道关闭后，进程写入输出可能收到 SIGPIPE 信号而异常退出
		//
		// ====================================================================================
		// 解决方案：setsid nohup ... </dev/null >/dev/null 2>&1 &
		// ====================================================================================
		//
		// 1. setsid: 创建新会话
		//    - 创建新的会话和进程组，进程成为会话领导者
		//    - 脱离原来的控制终端，不再依附于 sh 进程
		//    - 即使 sh 退出，新进程也不会被挂到 PID 1 下，而是独立运行
		//
		// 2. nohup: 忽略挂断信号
		//    - 忽略 SIGHUP 信号，当终端/父进程关闭时进程不会被终止
		//    - 确保进程在后台持续运行
		//
		// 3. </dev/null: 关闭标准输入
		//    - 标准输入从 /dev/null 读取（永远返回 EOF）
		//    - 避免进程等待输入导致阻塞
		//
		// 4. >/dev/null 2>&1: 重定向输出
		//    - 标准输出和错误输出重定向到 /dev/null（丢弃）
		//    - 应用自己会写日志文件，不需要通过标准输出
		//    - 断开与 podman exec 管道的连接，避免 SIGPIPE
		//
		// 5. &: 后台运行
		//    - 在后台异步执行，命令立即返回
		//
		// 最终效果：
		//   - 进程完全独立运行，不依赖任何终端或父进程
		//   - 真正的守护进程，支持灰度发布（新旧版本同时运行）
		//   - 不会变成僵尸进程
		// ====================================================================================

		// 执行启动命令
		startCmd := fmt.Sprintf("cd /app/workplace/bin && setsid nohup ./releases/%s </dev/null >/dev/null 2>&1 &", binaryName)
		output, err := s.containerService.ExecCommand(ctx, containerName, []string{"/bin/sh", "-c", startCmd})
		if err != nil {
			logStr.WriteString(fmt.Sprintf("Failed to start new version: %v, output: %s\t", err, output))
			return nil, fmt.Errorf("failed to start new version in container: %w", err)
		}

		logStr.WriteString("Command executed\t")
	} else {
		// 应用没有运行：先启动容器，再启动应用
		logger.Infof(ctx, "[UpdateApp] App is not running, starting container and app: %s", containerName)

		// 启动容器（挂载目录和可执行文件）
		if err := s.startAppContainer(ctx, containerName, appDirRel, newVersion); err != nil {
			return nil, fmt.Errorf("failed to start app container: %w", err)
		}

		logger.Infof(ctx, "[UpdateApp] Container started successfully")
	}

	// 统一在外层等待启动通知，无论哪种启动方式都需要等待
	logger.Infof(ctx, "[UpdateApp] Waiting for startup notification for %s/%s/%s", user, app, newVersion)

	select {
	case notification := <-waiterChan:
		logStr.WriteString(fmt.Sprintf("Startup confirmed at %s\t", notification.StartTime.Format(time.DateTime)))
		logger.Infof(ctx, "[UpdateApp] Startup confirmed: %s/%s/%s", user, app, newVersion)

		// 如果应用之前是未激活状态，现在启动成功后更新为已激活
		if isInactive {
			if err := s.updateAppStatusToActive(ctx, user, app); err != nil {
				logger.Warnf(ctx, "[UpdateApp] Failed to update app status to active: %v", err)
			} else {
				logger.Infof(ctx, "[UpdateApp] App status updated to active: %s/%s", user, app)
			}
		}
	case <-time.After(60 * time.Second):
		logStr.WriteString("Startup timeout\t")
		logger.Warnf(ctx, "[UpdateApp] Startup notification timeout for %s/%s/%s, but continue anyway", user, app, newVersion)
		// 不返回错误，超时不应阻止更新流程
	}

	logStr.WriteString(fmt.Sprintf("Update completed: %s->%s", oldVersion, newVersion))

	// 统一打印所有日志
	logger.Infof(ctx, logStr.String())

	// 使用 NATS Request/Reply 模式获取 API diff 结果
	logger.Infof(ctx, "[UpdateApp] 🚀 Using NATS Request/Reply to get update callback from %s/%s/%s", user, app, newVersion)

	updateCallbackResponse, callbackErr := s.sendUpdateCallbackAndWait(ctx, user, app, newVersion)
	if callbackErr != nil {
		logger.Warnf(ctx, "[UpdateApp] ❌ Update callback failed: %v", callbackErr)
		logger.Warnf(ctx, "[UpdateApp] Continuing without diff information")
	} else {
		logger.Infof(ctx, "[UpdateApp] ✅ Update callback response received from %s/%s/%s: %+v", user, app, newVersion, updateCallbackResponse)
	}

	// 构建 UpdateResult，包含 diff 信息（如果有）
	// 解析嵌套的 diff 数据，避免双嵌套

	result := &dto.UpdateAppResp{
		User:       user,
		App:        app,
		OldVersion: oldVersion,
		NewVersion: newVersion,
		Diff:       updateCallbackResponse.Data, // 修复后的 diff 信息
		Error:      callbackErr,
	}

	return result, nil
}

// updateAppStatusToActive 将应用状态更新为active（已激活）
func (s *AppManageService) updateAppStatusToActive(ctx context.Context, user, app string) error {
	appRecord, err := s.appRepo.GetApp(user, app)
	if err != nil {
		return fmt.Errorf("failed to get app record: %w", err)
	}

	// 更新状态为active
	appRecord.Status = "active"
	if err := s.appRepo.UpdateApp(appRecord); err != nil {
		return fmt.Errorf("failed to update app status to active: %w", err)
	}

	return nil
}

// sendUpdateCallbackAndWait 使用 NATS Request/Reply 模式发送 update 回调并等待响应
func (s *AppManageService) sendUpdateCallbackAndWait(ctx context.Context, user, app, version string) (*subjects.Message, error) {
	if s.natsConn == nil {
		return nil, fmt.Errorf("NATS connection is nil")
	}

	// 构建更新回调请求
	request := subjects.Message{
		Type:      subjects.MessageTypeStatusOnAppUpdate,
		User:      user,
		App:       app,
		Version:   version,
		Data:      map[string]interface{}{"trigger": "update_callback"},
		Timestamp: time.Now(),
	}

	// 构建请求主题
	//subject := subjects.GetAppUpdateCallbackRequestSubject(user, app, version)
	subject := subjects.BuildAppStatusSubject(user, app, version)

	logger.Infof(ctx, "[sendUpdateCallbackAndWait] 📤 Sending update callback request to subject: %s", subject)
	logger.Infof(ctx, "[sendUpdateCallbackAndWait] Request data: %+v", request)

	// 使用原生 NATS Request 模式，避免依赖 gin context
	msg := nats.NewMsg(subject)
	requestData, err := json.Marshal(request)
	if err != nil {
		logger.Errorf(ctx, "[sendUpdateCallbackAndWait] Failed to marshal request: %v", err)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	msg.Data = requestData

	// 发送请求并等待响应（60秒超时）
	responseMsg, err := s.natsConn.RequestMsg(msg, 60*time.Second)
	if err != nil {
		logger.Errorf(ctx, "[sendUpdateCallbackAndWait] ❌ Request failed: %v", err)
		return nil, fmt.Errorf("update callback request failed: %w", err)
	}

	var rsp subjects.Message

	// 解析响应
	if err := json.Unmarshal(responseMsg.Data, &rsp); err != nil {
		logger.Errorf(ctx, "[sendUpdateCallbackAndWait] Failed to unmarshal response: %v", err)
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &rsp, nil
}

// UpdateResult 更新结果
//type UpdateResult struct {
//	User       string
//	App        string
//	OldVersion string
//	NewVersion string
//	Diff       interface{} `json:"diff,omitempty"`  // API diff 信息
//	Error      error       `json:"error,omitempty"` // 回调过程中的错误
//}

// GetAppInfo 获取应用信息
func (s *AppManageService) GetAppInfo(ctx context.Context, user, app string) (map[string]interface{}, error) {
	appDir := fmt.Sprintf("namespace/%s/%s", user, app)

	// 检查应用是否存在
	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("app not found: %s/%s", user, app)
	}

	// 读取版本信息
	versionFile := filepath.Join(appDir, "workplace/metadata/version.txt")
	versionData, _ := os.ReadFile(versionFile)

	return map[string]interface{}{
		"user":    user,
		"app":     app,
		"app_dir": appDir,
		"version": string(versionData),
	}, nil
}

// IsAppRunning 检查应用是否正在运行
// 使用discovery service检查运行状态，比调用podman更高效
func (s *AppManageService) IsAppRunning(ctx context.Context, user, app string) (bool, error) {
	if s.appDiscoveryService == nil {
		return false, fmt.Errorf("app discovery service not available")
	}

	// 使用discovery service检查应用是否正在运行
	return s.appDiscoveryService.IsAppRunning(user, app), nil
}

// createDirIfNotExists 创建目录（如果不存在）
func (s *AppManageService) createDirIfNotExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// createVersionFiles 创建版本文件
func (s *AppManageService) createVersionFiles(metadataDir, user, app string) error {
	// 创建版本数据结构
	versionData := VersionData{
		User:           user,
		App:            app,
		CurrentVersion: "v1",
		LatestVersion:  "v1",
		Versions: []VersionInfo{
			{
				Version:   "v1",
				CreatedAt: time.Now().Format(time.RFC3339),
				Status:    "active",
			},
		},
	}

	// 序列化为 JSON
	data, err := json.MarshalIndent(versionData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal version.json: %w", err)
	}

	// 写入文件
	versionFile := filepath.Join(metadataDir, "version.json")
	if err := os.WriteFile(versionFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write version.json: %w", err)
	}

	return nil
}

// VersionInfo 版本信息结构体
type VersionInfo struct {
	Version   string `json:"version"`
	CreatedAt string `json:"created_at"`
	Status    string `json:"status"`
}

// VersionData version.json 文件结构体
type VersionData struct {
	User           string        `json:"user"`
	App            string        `json:"app"`
	CurrentVersion string        `json:"current_version"`
	LatestVersion  string        `json:"latest_version"`
	Versions       []VersionInfo `json:"versions"`
}

// updateVersionJson 更新 version.json 文件
func (s *AppManageService) updateVersionJson(appDir, user, app, newVersion string) error {
	versionFile := filepath.Join(appDir, "workplace/metadata/version.json")

	// 读取现有的 version.json
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return fmt.Errorf("failed to read version.json: %w", err)
	}

	// 解析现有数据
	var versionData VersionData
	if err := json.Unmarshal(data, &versionData); err != nil {
		return fmt.Errorf("failed to parse version.json: %w", err)
	}

	// 将旧版本状态改为 inactive
	for i := range versionData.Versions {
		if versionData.Versions[i].Status == "active" {
			versionData.Versions[i].Status = "inactive"
		}
	}

	// 检查新版本是否已存在，如果存在则更新，否则添加
	var versionExists bool
	for i := range versionData.Versions {
		if versionData.Versions[i].Version == newVersion {
			// 更新现有版本
			versionData.Versions[i].Status = "active"
			versionData.Versions[i].CreatedAt = time.Now().Format(time.RFC3339)
			versionExists = true
			break
		}
	}

	// 如果版本不存在，则添加新版本
	if !versionExists {
		newVersionInfo := VersionInfo{
			Version:   newVersion,
			CreatedAt: time.Now().Format(time.RFC3339),
			Status:    "active",
		}
		versionData.Versions = append(versionData.Versions, newVersionInfo)
	}

	// 更新版本信息
	versionData.CurrentVersion = newVersion
	versionData.LatestVersion = newVersion

	// 写回文件
	updatedData, err := json.MarshalIndent(versionData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal version.json: %w", err)
	}

	if err := os.WriteFile(versionFile, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write version.json: %w", err)
	}

	// 同时维护纯文本文件，用于极速启动
	if err := s.updateCurrentVersionFiles(versionData.User, versionData.App, newVersion); err != nil {
		logger.Warnf(context.Background(), "[updateVersionJson] Failed to update current version files: %v", err)
		// 不返回错误，纯文本文件失败不应阻止更新
	}

	//logger.Infof(context.Background(), "[updateVersionJson] Updated version.json: current_version=%s, latest_version=%s", newVersion, newVersion)
	return nil
}

// updateCurrentVersionFiles 更新纯文本版本文件，用于极速启动
func (s *AppManageService) updateCurrentVersionFiles(user, app, version string) error {
	metadataDir := filepath.Join("namespace", user, app, "workplace", "metadata")

	// 确保 metadata 目录存在
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	// 更新 current_version.txt
	versionFile := filepath.Join(metadataDir, "current_version.txt")
	if err := os.WriteFile(versionFile, []byte(version), 0644); err != nil {
		return fmt.Errorf("failed to write current_version.txt: %w", err)
	}

	// 更新 current_app.txt
	appFile := filepath.Join(metadataDir, "current_app.txt")
	appName := fmt.Sprintf("%s_%s", user, app)
	if err := os.WriteFile(appFile, []byte(appName), 0644); err != nil {
		return fmt.Errorf("failed to write current_app.txt: %w", err)
	}

	//logger.Infof(context.Background(), "[updateCurrentVersionFiles] Updated current_version.txt=%s, current_app.txt=%s", version, appName)
	return nil
}

// createMainGoFile 创建 main.go 文件
func (s *AppManageService) createMainGoFile(mainGoPath, user, app string) error {
	content := []byte(`package main

import (
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
)

func main() {
	err := app.Run()
	if err != nil {
		panic(err)
	}
}
`)

	return os.WriteFile(mainGoPath, content, 0644)
}

// createVersionContainer 创建版本容器
// 这是新架构的核心方法：每个版本使用独立的容器
func (s *AppManageService) createVersionContainer(ctx context.Context, user, app, version, appDir string) error {
	containerName := buildContainerName(user, app, version)
	logger.Infof(ctx, "[createVersionContainer] Creating version container: %s for %s/%s/%s", containerName, user, app, version)

	// 检查容器是否已存在
	exists, err := s.containerService.IsContainerRunning(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to check container existence: %w", err)
	}

	if exists {
		logger.Warnf(ctx, "[createVersionContainer] Container %s already exists and is running", containerName)
		return fmt.Errorf("container %s already exists and is running", containerName)
	}

	// 调用现有的 startAppContainer，但使用新的容器名
	// startAppContainer 会创建并启动容器
	return s.startAppContainer(ctx, containerName, appDir, version)
}

// startAppContainer 启动应用容器
func (s *AppManageService) startAppContainer(ctx context.Context, containerName, appDir, version string) error {
	logger.Infof(ctx, "Starting container: %s, appDir: %s, version: %s", containerName, appDir, version)

	// 获取容器操作器
	if s.containerService == nil {
		logger.Errorf(ctx, "Container operator not available")
		return fmt.Errorf("container operator not available")
	}

	// 使用自定义的 ai-agent-os 镜像启动容器，挂载应用目录
	image := "ai-agent-os:latest"
	// 将相对路径转换为绝对路径，避免 Podman 把它当成卷名
	absHostPath, err := filepath.Abs(appDir)
	if err != nil {
		logger.Errorf(ctx, "Failed to get absolute path: %v", err)
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	containerPath := "/app"

	logger.Infof(ctx, "[startAppContainer] Running container with mount: image=%s, name=%s, hostPath=%s, containerPath=%s", image, containerName, absHostPath, containerPath)

	// 设置环境变量
	envVars := []string{
		"NATS_URL=nats://host.containers.internal:4223", // 使用 host.containers.internal 访问宿主机 NATS
	}

	// 注入网关地址（从 runtime 配置读取）
	gatewayURL := s.runtimeConfig.Runtime.GatewayURL
	if gatewayURL == "" {
		// 如果没有配置，使用默认值
		gatewayURL = "http://localhost:9090"
	}
	envVars = append(envVars, fmt.Sprintf("GATEWAY_URL=%s", gatewayURL))
	logger.Infof(ctx, "[startAppContainer] Injecting GATEWAY_URL=%s into container", gatewayURL)

	// 启动容器，使用 ai-agent-os 镜像的启动脚本
	// 启动脚本会自动读取 metadata/version.json 来获取版本信息，或者使用传入的版本参数
	logger.Infof(ctx, "[startAppContainer] Creating container with ai-agent-os image: %s", containerName)
	if err := s.containerService.RunContainerWithCommand(ctx, image, containerName, absHostPath, containerPath, []string{"/start.sh", version}, envVars...); err != nil {
		logger.Errorf(ctx, "[startAppContainer] Failed to start container: %v", err)
		return fmt.Errorf("failed to start container: %w", err)
	}

	logger.Infof(ctx, "Container started successfully with ai-agent-os image")
	return nil
}

// stopOldVersionContainer 优雅关闭旧版本容器（三次握手流程）
// 这是新架构的核心方法：优雅关闭旧版本容器
func (s *AppManageService) stopOldVersionContainer(ctx context.Context, user, app, oldVersion string) error {
	containerName := buildContainerName(user, app, oldVersion)
	logger.Infof(ctx, "[stopOldVersionContainer] Starting graceful shutdown for old container: %s", containerName)

	// 1. 检查容器是否存在
	exists, err := s.containerService.IsContainerRunning(ctx, containerName)
	if err != nil {
		logger.Warnf(ctx, "[stopOldVersionContainer] Failed to check container existence: %v", err)
		return nil // 不返回错误，继续执行
	}
	if !exists {
		logger.Infof(ctx, "[stopOldVersionContainer] Old container %s not found, skipping", containerName)
		return nil
	}

	// 2. 发送 shutdown 命令给旧版本（第二次握手）
	logger.Infof(ctx, "[stopOldVersionContainer] Sending shutdown command to %s/%s/%s (second handshake)", user, app, oldVersion)
	if err := s.ShutdownAppVersion(ctx, user, app, oldVersion); err != nil {
		logger.Warnf(ctx, "[stopOldVersionContainer] Failed to send shutdown command: %v", err)
		// 不返回错误，继续执行
	}

	// 3. 注册关闭等待器，等待旧版本的 close 通知（第三次握手）
	closeWaiterChan := s.registerCloseWaiter(user, app, oldVersion)
	defer s.unregisterCloseWaiter(user, app, oldVersion)

	// 4. 等待旧版本关闭确认（最多30秒，与旧版本等待函数完成的时间一致）
	logger.Infof(ctx, "[stopOldVersionContainer] Waiting for close notification from %s/%s/%s (third handshake, timeout: 30s)", user, app, oldVersion)
	select {
	case notification := <-closeWaiterChan:
		logger.Infof(ctx, "[stopOldVersionContainer] Received close notification from old version %s/%s/%s at %s",
			notification.User, notification.App, notification.Version, notification.CloseTime.Format(time.DateTime))
	case <-time.After(30 * time.Second):
		logger.Warnf(ctx, "[stopOldVersionContainer] Timeout waiting for close notification from old version %s/%s/%s, forcing stop", user, app, oldVersion)
		// 超时后强制停止
	}

	// 5. 停止容器（不删除，保留以便快速回滚）
	logger.Infof(ctx, "[stopOldVersionContainer] Stopping container %s (not removing)", containerName)
	if err := s.containerService.StopContainer(ctx, containerName); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	logger.Infof(ctx, "[stopOldVersionContainer] Old container %s stopped successfully", containerName)
	return nil
}

// ShutdownAppVersion 主动关闭指定版本的应用
func (s *AppManageService) ShutdownAppVersion(ctx context.Context, user, app, version string) error {
	//logger.Infof(ctx, "[ShutdownAppVersion] Sending shutdown command to %s/%s/%s", user, app, version)

	// 构建关闭命令消息（使用 subjects.Message 格式）
	message := subjects.Message{
		Type:      subjects.MessageTypeStatusShutdown,
		User:      user,
		App:       app,
		Version:   version,
		Data:      map[string]interface{}{"command": "shutdown"},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal shutdown command: %w", err)
	}

	// 发送关闭命令到应用状态主题
	subject := subjects.BuildAppStatusSubject(user, app, version)

	if err := s.natsConn.Publish(subject, data); err != nil {
		return fmt.Errorf("failed to publish shutdown command to %s: %w", subject, err)
	}

	//logger.Infof(ctx, "[ShutdownAppVersion] Shutdown command sent to %s", subject)
	return nil
}

// ShutdownOldVersions 关闭旧版本的应用（保留指定数量的最新版本）
func (s *AppManageService) ShutdownOldVersions(ctx context.Context, user, app string, keepVersions int) error {
	logger.Infof(ctx, "[ShutdownOldVersions] Shutting down old versions for %s/%s, keeping %d versions", user, app, keepVersions)

	// 从内存中获取运行中的版本（通过 AppDiscoveryService）
	runningApps := s.appDiscoveryService.GetRunningApps()
	appKey := user + "/" + app
	appInfo, exists := runningApps[appKey]
	if !exists {
		logger.Infof(ctx, "[ShutdownOldVersions] No running versions found for %s/%s", user, app)
		return nil
	}

	// 转换为版本列表
	var runningVersions []string
	for versionKey := range appInfo.Versions {
		runningVersions = append(runningVersions, versionKey)
	}

	if len(runningVersions) <= keepVersions {
		logger.Infof(ctx, "[ShutdownOldVersions] Only %d versions running, no need to shutdown", len(runningVersions))
		return nil
	}

	// 关闭旧版本（基于 QPS 安全检查）
	// 注意：这里简化逻辑，因为内存中的版本信息不包含创建时间
	// 实际应用中，应该根据业务需求决定关闭策略
	versionsToShutdown := runningVersions[keepVersions:]
	for _, version := range versionsToShutdown {
		// 检查是否可以安全关闭
		if !s.QPSTracker.IsSafeToShutdown(user, app, version) {
			logger.Warnf(ctx, "[ShutdownOldVersions] Version %s still has traffic, skipping shutdown", version)
			continue
		}

		if err := s.ShutdownAppVersion(ctx, user, app, version); err != nil {
			logger.Errorf(ctx, "[ShutdownOldVersions] Failed to shutdown version %s: %v", version, err)
		} else {
			logger.Infof(ctx, "[ShutdownOldVersions] Shutdown command sent to version %s", version)
		}
	}

	return nil
}

// StartCleanupTask 启动定时清理任务
func (s *AppManageService) StartCleanupTask(ctx context.Context) {

	// 每30秒检查一次是否需要关闭旧版本
	s.cleanupTicker = time.NewTicker(30 * time.Second)

	go func() {
		defer s.cleanupTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Infof(ctx, "[AppManageService] Cleanup task stopped by context")
				return
			case <-s.cleanupDone:
				logger.Infof(ctx, "[AppManageService] Cleanup task stopped by signal")
				return
			case <-s.cleanupTicker.C:
				s.performCleanup(ctx)
			}
		}
	}()

}

// StopCleanupTask 停止定时清理任务
func (s *AppManageService) StopCleanupTask(ctx context.Context) {
	if s.cleanupTicker != nil {
		s.cleanupTicker.Stop()
	}

	select {
	case s.cleanupDone <- struct{}{}:
	default:
	}

	logger.Infof(ctx, "[AppManageService] Cleanup task stopped")
}

// performCleanup 执行清理任务
func (s *AppManageService) performCleanup(ctx context.Context) {
	//logger.Infof(ctx, "[AppManageService] Performing cleanup check...")

	// 获取所有应用
	apps, err := s.getAllApps(ctx)
	if err != nil {
		logger.Errorf(ctx, "[AppManageService] Failed to get apps: %v", err)
		return
	}

	if len(apps) == 0 {
		return
	}

	// 为每个应用执行清理
	for _, app := range apps {
		// 清理非当前版本的无流量版本
		if err := s.CleanupNonCurrentVersions(ctx, app.User, app.App); err != nil {
			logger.Errorf(ctx, "[AppManageService] Failed to cleanup versions for %s/%s: %v", app.User, app.App, err)
		}

	}
}

// getAllApps 获取所有应用
func (s *AppManageService) getAllApps(ctx context.Context) ([]*model.App, error) {
	return s.appRepo.GetAllApps()
}

// CleanupNonCurrentVersions 清理非当前版本的无流量版本
// 策略：只保留 current_version（metadata 中的当前版本），其他版本只要 QPS 为 0 就停掉
func (s *AppManageService) CleanupNonCurrentVersions(ctx context.Context, user, app string) error {
	//logger.Infof(ctx, "[CleanupNonCurrentVersions] Checking %s/%s", user, app)

	// 1. 读取 current_version
	currentVersion, err := s.getCurrentVersion(ctx, user, app)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	if currentVersion == "" {
		//logger.Warnf(ctx, "[CleanupNonCurrentVersions] No current version found for %s/%s", user, app)
		return nil
	}

	//logger.Infof(ctx, "[CleanupNonCurrentVersions] Current version: %s", currentVersion)

	// 2. 从内存中获取所有运行中的版本
	runningApps := s.appDiscoveryService.GetRunningApps()
	appKey := user + "/" + app
	appInfo, exists := runningApps[appKey]
	if !exists {
		//logger.Infof(ctx, "[CleanupNonCurrentVersions] No running versions found for %s/%s", user, app)
		return nil
	}

	// 3. 关闭非当前版本且无流量的版本
	for _, version := range appInfo.Versions {
		// 跳过当前版本
		if version.Version == currentVersion {
			//logger.Infof(ctx, "[CleanupNonCurrentVersions] Skipping current version: %s", version.Version)
			continue
		}

		// 检查是否可以安全关闭（QPS 为 0）
		if !s.QPSTracker.IsSafeToShutdown(user, app, version.Version) {
			//logger.Infof(ctx, "[CleanupNonCurrentVersions] Version %s still has traffic, skipping", version.Version)
			continue
		}

		// 关闭该版本
		//logger.Infof(ctx, "[CleanupNonCurrentVersions] Shutting down non-current version %s (no traffic)", version.Version)
		if err := s.ShutdownAppVersion(ctx, user, app, version.Version); err != nil {
			logger.Errorf(ctx, "[CleanupNonCurrentVersions] Failed to shutdown version %s: %v", version.Version, err)
		} else {
			logger.Infof(ctx, "[CleanupNonCurrentVersions] 停掉进程 成功  %s_%s_%s ", user, app, version.Version)
		}
	}

	return nil
}

// getCurrentVersion 获取应用的当前版本（从 metadata/current_version.txt）
func (s *AppManageService) getCurrentVersion(ctx context.Context, user, app string) (string, error) {
	// 读取 current_version.txt
	versionFile := filepath.Join(s.config.AppDir.BasePath, user, app, "workplace/metadata/current_version.txt")

	data, err := os.ReadFile(versionFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // 文件不存在，返回空
		}
		return "", fmt.Errorf("failed to read current version file: %w", err)
	}

	currentVersion := strings.TrimSpace(string(data))
	return currentVersion, nil
}

// StartAppVersion 启动指定版本的应用（兜底启动）
// 用于应用挂了或更新失败时重新启动目标版本
func (s *AppManageService) StartAppVersion(ctx context.Context, user, app, version string) error {
	logger.Infof(ctx, "[StartAppVersion] Starting version %s/%s/%s", user, app, version)

	// 使用新的容器命名格式：{user}-{app}-{version}
	containerName := buildContainerName(user, app, version)

	// 注册启动等待器（统一在外层注册）
	waiterChan := s.registerStartupWaiter(user, app, version)
	// 确保在方法结束时清理等待器
	defer s.unregisterStartupWaiter(user, app, version)

	// 读取 current_app.txt 获取二进制前缀
	appFile := filepath.Join(s.config.AppDir.BasePath, user, app, "workplace/metadata/current_app.txt")
	appData, err := os.ReadFile(appFile)
	if err != nil {
		return fmt.Errorf("failed to read current_app.txt: %w", err)
	}
	binaryPrefix := strings.TrimSpace(string(appData))
	binaryName := fmt.Sprintf("%s_%s", binaryPrefix, version)

	// 构建启动命令（使用 setsid nohup 确保进程后台运行）
	startCmd := fmt.Sprintf(
		"cd /app/workplace/bin && setsid nohup ./releases/%s </dev/null >/dev/null 2>&1 &",
		binaryName,
	)

	logger.Infof(ctx, "[StartAppVersion] Executing startup command in container %s: %s", containerName, startCmd)

	// 执行启动命令
	output, err := s.containerService.ExecCommand(ctx, containerName, []string{"sh", "-c", startCmd})
	if err != nil {
		return fmt.Errorf("failed to execute startup command: %w, output: %s", err, output)
	}

	logger.Infof(ctx, "[StartAppVersion] Startup command executed, output: %s", output)

	// 等待启动完成通知（30秒超时）
	logger.Infof(ctx, "[StartAppVersion] Waiting for startup notification from version %s...", version)

	select {
	case notification := <-waiterChan:
		logger.Infof(ctx, "[StartAppVersion] Received startup notification: %s/%s/%s, status=%s",
			notification.User, notification.App, notification.Version, notification.Status)

		if notification.Status == "running" {
			logger.Infof(ctx, "[StartAppVersion] Version %s started successfully", version)
			return nil
		}
		return fmt.Errorf("app started but status is not running: %s", notification.Status)

	case <-time.After(30 * time.Second):
		logger.Warnf(ctx, "[StartAppVersion] Timeout waiting for startup notification from version %s", version)
		return fmt.Errorf("timeout waiting for app startup notification")
	}
}
