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

// AppManageService 应用管理服务 - 负责应用的增删改查
type AppManageService struct {
	builder             *builder.Builder
	config              *appconfig.AppManageServiceConfig
	containerService    ContainerOperator         // 容器服务依赖
	appRepo             *repository.AppRepository // 应用数据访问层
	appDiscoveryService *AppDiscoveryService      // 应用发现服务，用于获取运行状态
	natsConn            *nats.Conn                // NATS 连接，用于发送关闭命令
	QPSTracker          *QPSTracker               // QPS 跟踪器

	// 启动等待器 - 用于等待应用启动完成通知
	startupWaiters   map[string]chan *StartupNotification // key: user/app/version
	startupWaitersMu sync.RWMutex

	// 定时任务控制
	cleanupTicker *time.Ticker
	cleanupDone   chan struct{}
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
func NewAppManageService(builder *builder.Builder, config *appconfig.AppManageServiceConfig, containerService ContainerOperator, appRepo *repository.AppRepository, appDiscoveryService *AppDiscoveryService, natsConn *nats.Conn) *AppManageService {
	return &AppManageService{
		builder:             builder,
		config:              config,
		containerService:    containerService,
		appRepo:             appRepo,
		appDiscoveryService: appDiscoveryService,
		natsConn:            natsConn,
		QPSTracker:          NewQPSTracker(60*time.Second, 10*time.Second), // 60秒窗口，10秒检查间隔
		startupWaiters:      make(map[string]chan *StartupNotification),
		cleanupDone:         make(chan struct{}),
	}
}

// NotifyStartup 通知应用启动完成（由 NATS 消息处理器调用）
func (s *AppManageService) NotifyStartup(notification *StartupNotification) {
	s.notifyStartupWaiter(notification.User, notification.App, notification.Version, notification)
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
	key := fmt.Sprintf("%s/%s/%s", user, app, version)

	// 注册等待器
	ch := make(chan *StartupNotification, 1)
	s.startupWaitersMu.Lock()
	s.startupWaiters[key] = ch
	s.startupWaitersMu.Unlock()

	// 确保清理
	defer func() {
		s.startupWaitersMu.Lock()
		delete(s.startupWaiters, key)
		close(ch)
		s.startupWaitersMu.Unlock()
	}()

	//logger.Infof(ctx, "[waitForStartup] Waiting for: %s (timeout: %v)", key, timeout)

	select {
	case notification := <-ch:
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
func (s *AppManageService) UpdateApp(ctx context.Context, user, app string) (*UpdateResult, error) {

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

	// 2. 检查应用是否正在运行
	isRunning, err := s.IsAppRunning(ctx, user, app)
	if err != nil {
		logStr.WriteString(fmt.Sprintf("Failed to check app running status: %v\t", err))
		isRunning = false
	}

	// 使用 VersionManager 获取当前版本
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
	containerName := fmt.Sprintf("%s-%s", user, app)

	if s.containerService == nil {
		return nil, fmt.Errorf("container operator not available")
	}

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
		// 先注册等待器，再执行启动命令，避免错过通知
		// 使用同步方式注册等待器，确保注册完成后再启动应用
		waiterChan := s.registerStartupWaiter(user, app, newVersion)

		// 确保等待器已注册后再启动应用
		logger.Infof(ctx, "[StartAppVersion] Waiting for startup notification for %s/%s/%s", user, app, newVersion)

		// 执行启动命令
		startCmd := fmt.Sprintf("cd /app/workplace/bin && setsid nohup ./releases/%s </dev/null >/dev/null 2>&1 &", binaryName)
		output, err := s.containerService.ExecCommand(ctx, containerName, []string{"/bin/sh", "-c", startCmd})
		if err != nil {
			logStr.WriteString(fmt.Sprintf("Failed to start new version: %v, output: %s\t", err, output))
			return nil, fmt.Errorf("failed to start new version in container: %w", err)
		}

		logStr.WriteString("Command executed\t")

		// 等待启动通知结果（同步等待）
		select {
		case notification := <-waiterChan:
			logStr.WriteString(fmt.Sprintf("Startup confirmed at %s\t", notification.StartTime.Format(time.DateTime)))
			logger.Infof(ctx, "[UpdateApp] New version startup confirmed: %s/%s/%s", user, app, newVersion)
		case <-time.After(30 * time.Second):
			logStr.WriteString("Startup timeout\t")
			logger.Warnf(ctx, "[UpdateApp] Startup notification timeout for %s/%s/%s, but continue anyway", user, app, newVersion)
			// 不返回错误，超时不应阻止更新流程
		}

		// 清理等待器
		s.unregisterStartupWaiter(user, app, newVersion)

		logStr.WriteString("New version started in container\t")
	} else {
		// 应用没有运行：先启动容器，再启动应用
		logger.Infof(ctx, "[UpdateApp] App is not running, starting container and app: %s", containerName)

		// 启动容器（挂载目录和可执行文件）
		if err := s.startAppContainer(ctx, containerName, appDirRel, ""); err != nil {
			return nil, fmt.Errorf("failed to start app container: %w", err)
		}

		logger.Infof(ctx, "[UpdateApp] Container started successfully, app will start automatically via start.sh")
	}

	logStr.WriteString(fmt.Sprintf("Update completed: %s->%s", oldVersion, newVersion))

	// 统一打印所有日志
	logger.Infof(ctx, logStr.String())

	//todo 发送diff 回调，什么是diff回调，就是当你更新时候，有可能新增api，有可能删除api，有可能更新api
	//这时候我们要做个diff，给出这个变更的明细，返回出去，这样server层可以更新
	//这里可以复用那个status主题来发送

	return &UpdateResult{
		User:       user,
		App:        app,
		OldVersion: oldVersion,
		NewVersion: newVersion,
	}, nil
}

// UpdateResult 更新结果
type UpdateResult struct {
	User       string
	App        string
	OldVersion string
	NewVersion string
}

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
func (s *AppManageService) IsAppRunning(ctx context.Context, user, app string) (bool, error) {
	containerName := fmt.Sprintf("%s-%s", user, app)

	if s.containerService == nil {
		return false, fmt.Errorf("container operator not available")
	}

	// 检查容器是否存在且运行中
	containerList, err := s.containerService.ListContainers(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to list containers: %w", err)
	}

	// 查找指定名称的容器
	var exists, isRunning bool
	for _, container := range containerList {
		if container.Names[0] == containerName {
			exists = true
			isRunning = container.State == "running"
			break
		}
	}

	//logger.Infof(ctx, "[IsAppRunning] Container %s: exists=%v, running=%v", containerName, exists, isRunning)
	return exists && isRunning, nil
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

// startAppContainer 启动应用容器
func (s *AppManageService) startAppContainer(ctx context.Context, containerName, appDir, executablePath string) error {
	logger.Infof(ctx, "Starting container: %s, appDir: %s, executablePath: %s", containerName, appDir, executablePath)

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

	// 启动容器，使用 ai-agent-os 镜像的启动脚本
	// 启动脚本会自动读取 metadata/version.json 来获取版本信息
	logger.Infof(ctx, "[startAppContainer] Creating container with ai-agent-os image: %s", containerName)
	if err := s.containerService.RunContainerWithCommand(ctx, image, containerName, absHostPath, containerPath, []string{"/start.sh"}, envVars...); err != nil {
		logger.Errorf(ctx, "[startAppContainer] Failed to start container: %v", err)
		return fmt.Errorf("failed to start container: %w", err)
	}

	logger.Infof(ctx, "Container started successfully with ai-agent-os image")
	return nil
}

// UpdateAppStatus 更新应用状态
func (s *AppManageService) UpdateAppStatus(ctx context.Context, user, app, version, status string) error {
	// 更新应用版本状态
	versions, err := s.appRepo.GetAppVersions(user, app)
	if err != nil {
		return fmt.Errorf("failed to get app versions: %w", err)
	}

	for _, v := range versions {
		if v.Version == version {
			v.Status = status
			v.LastSeen = time.Now()
			if status == "stopped" {
				now := time.Now()
				v.StopTime = &now
			}
			if err := s.appRepo.UpdateAppVersion(v); err != nil {
				return fmt.Errorf("failed to update app version status: %w", err)
			}
			logger.Infof(ctx, "[UpdateAppStatus] Updated version %s status to %s", version, status)
			break
		}
	}

	// 如果是当前版本，也更新应用主记录
	appRecord, err := s.appRepo.GetApp(user, app)
	if err == nil && appRecord.Version == version {
		appRecord.Status = status
		appRecord.LastSeen = time.Now()
		if err := s.appRepo.UpdateApp(appRecord); err != nil {
			return fmt.Errorf("failed to update app status: %w", err)
		}
		logger.Infof(ctx, "[UpdateAppStatus] Updated app %s/%s status to %s", user, app, status)
	}

	return nil
}

// ShutdownAppVersion 主动关闭指定版本的应用
func (s *AppManageService) ShutdownAppVersion(ctx context.Context, user, app, version string) error {
	//logger.Infof(ctx, "[ShutdownAppVersion] Sending shutdown command to %s/%s/%s", user, app, version)

	// 构建关闭命令消息（使用 subjects.Message 格式）
	message := subjects.Message{
		Type:      subjects.MessageTypeShutdown,
		User:      user,
		App:       app,
		Version:   version,
		Data:      map[string]interface{}{"command": "shutdown"},
		Timestamp: time.Now().Format(time.RFC3339),
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
			//logger.Errorf(ctx, "[CleanupNonCurrentVersions] Failed to shutdown version %s: %v", version.Version, err)
		} else {
			//logger.Infof(ctx, "[CleanupNonCurrentVersions] Successfully sent shutdown command to version %s", version.Version)
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

	containerName := fmt.Sprintf("%s_%s", user, app)

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

	// 注册启动等待器
	key := fmt.Sprintf("%s/%s/%s", user, app, version)
	s.startupWaitersMu.Lock()
	s.startupWaiters[key] = make(chan *StartupNotification, 1)
	s.startupWaitersMu.Unlock()

	// 等待启动完成通知（30秒超时）
	logger.Infof(ctx, "[StartAppVersion] Waiting for startup notification from version %s...", version)

	s.startupWaitersMu.RLock()
	waiterChan := s.startupWaiters[key]
	s.startupWaitersMu.RUnlock()

	select {
	case notification := <-waiterChan:
		logger.Infof(ctx, "[StartAppVersion] Received startup notification: %s/%s/%s, status=%s",
			notification.User, notification.App, notification.Version, notification.Status)

		// 清理等待器
		s.startupWaitersMu.Lock()
		delete(s.startupWaiters, key)
		s.startupWaitersMu.Unlock()

		if notification.Status == "running" {
			logger.Infof(ctx, "[StartAppVersion] Version %s started successfully", version)
			return nil
		}
		return fmt.Errorf("app started but status is not running: %s", notification.Status)

	case <-time.After(30 * time.Second):
		// 清理等待器
		s.startupWaitersMu.Lock()
		delete(s.startupWaiters, key)
		s.startupWaitersMu.Unlock()

		logger.Warnf(ctx, "[StartAppVersion] Timeout waiting for startup notification from version %s", version)
		return fmt.Errorf("timeout waiting for app startup notification")
	}
}
