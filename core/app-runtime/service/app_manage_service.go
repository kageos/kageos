package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	sharedDto "github.com/kageos/kageos/dto"

	"github.com/kageos/kageos/core/app-runtime/repository"
	"github.com/kageos/kageos/pkg/builder"
	"github.com/kageos/kageos/pkg/buildtrace"
	appconfig "github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/subjects"
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
	Error     string
}

// CloseNotification 关闭通知
type CloseNotification struct {
	User      string
	App       string
	Version   string
	CloseTime time.Time
}

const (
	appNATSCredentialsSecretTarget = "kageos-nats"
	appNATSCredentialsSDKMarker    = "/run/secrets/kageos-nats"
	appBinaryMarkerScanBufferSize  = 64 * 1024
)

// AppManageService 应用管理服务 - 负责应用的增删改查
type AppManageService struct {
	builder              *builder.Builder
	config               *appconfig.AppManageServiceConfig
	runtimeConfig        *appconfig.AppRuntimeConfig // 运行时完整配置（用于获取网关地址等）
	runtimeDriver        AppRuntimeDriver            // 应用版本运行时抽象
	appRepo              *repository.AppRepository   // 应用数据访问层
	appDiscoveryService  *AppDiscoveryService        // 应用发现服务，用于获取运行状态
	appControlClient     *AppControlClient           // runtime -> app 控制调用
	appDatabaseService   *AppDatabaseService         // runtime-managed app DB capability issuer
	QPSTracker           *QPSTracker                 // QPS 跟踪器
	workspaceFileService *WorkspaceFileService       // 工作区源码文件服务

	// 启动等待器 - 用于等待应用启动完成通知
	startupWaiters   map[string]chan *StartupNotification // key: user/app/version
	startupWaitersMu sync.RWMutex

	// 关闭等待器 - 用于等待应用关闭完成通知
	closeWaiters   map[string]chan *CloseNotification // key: user/app/version
	closeWaitersMu sync.RWMutex

	// 周期清理控制（进程级+容器级合并为一次完整清理，由 cron + 有变动时触发）
	cleanupDone chan struct{}

	// 容器级对账巡检控制（cron 低峰期执行 + 有变动时由 ticker 触发）
	containerCleanupTicker *time.Ticker
	containerCleanupDone   chan struct{}
	containerCleanupCron   *cron.Cron // 每日定点执行，如 "0 4 * * *" 表示凌晨 4 点

	// 有版本/容器变动时置为 true，ticker 检查到后执行一次巡检
	containerCleanupMu    sync.Mutex
	containerCleanupDirty bool
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

// NewAppManageService 创建应用管理服务（依赖注入）
func NewAppManageService(builder *builder.Builder, config *appconfig.AppManageServiceConfig, runtimeConfig *appconfig.AppRuntimeConfig, containerService ContainerOperator, appRepo *repository.AppRepository, appDiscoveryService *AppDiscoveryService, natsConn *nats.Conn, workspaceFileService *WorkspaceFileService, appDatabaseService *AppDatabaseService) *AppManageService {
	return &AppManageService{
		builder:              builder,
		config:               config,
		runtimeConfig:        runtimeConfig,
		runtimeDriver:        NewPodmanAppRuntimeDriver(containerService),
		appRepo:              appRepo,
		appDiscoveryService:  appDiscoveryService,
		appControlClient:     NewAppControlClient(natsConn),
		appDatabaseService:   appDatabaseService,
		QPSTracker:           NewQPSTracker(60*time.Second, 10*time.Second), // 60秒窗口，10秒检查间隔
		workspaceFileService: workspaceFileService,
		startupWaiters:       make(map[string]chan *StartupNotification),
		closeWaiters:         make(map[string]chan *CloseNotification),
		cleanupDone:          make(chan struct{}),
		containerCleanupDone: make(chan struct{}),
	}
}

// CreateApp 创建应用目录结构
func (s *AppManageService) CreateApp(ctx context.Context, user, app string, opts ...*CreateOpts) (string, error) {
	logger.Infof(ctx, "[CreateApp] *** ENTRY *** user=%s, app=%s", user, app)

	appPaths := newRuntimeAppPaths(s.config.GetBasePath(), user, app)
	appDirRel := appPaths.AppDir()
	absAppDir, err := filepath.Abs(appDirRel)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}
	absPaths := newRuntimeAppPathsFromAppDir(absAppDir, user, app)

	// 2. 定义完整的目录结构（使用配置中的结构）
	dirs := []string{
		// 应用根目录
		absAppDir,
	}

	// 添加配置中定义的目录结构
	for _, dir := range s.config.GetStructure() {
		dirs = append(dirs, filepath.Join(absAppDir, dir))
	}

	// 3. 创建所有目录
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// 4. 启动脚本已内置在 kageos 镜像中，无需复制

	// 5. 创建应用时不创建版本文件，版本文件将在第一次编译时创建
	// 这样可以避免创建时就写入版本信息
	logger.Infof(ctx, "[CreateApp] Skipping version files creation, will be created on first build")

	if err := s.ensureAppGoModFile(absPaths); err != nil {
		return "", fmt.Errorf("failed to create go.mod file: %w", err)
	}

	// 6. 创建 main.go 文件
	mainGoPath := absPaths.MainGoPath()
	if err := s.createMainGoFile(mainGoPath, user, app); err != nil {
		return "", fmt.Errorf("failed to create main.go file: %w", err)
	}

	// 8. 保存应用信息到数据库
	if err := s.appRepo.CreateApp(ctx, user, app); err != nil {
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

	appPaths := newRuntimeAppPaths(s.config.GetBasePath(), user, app)

	// 设置默认编译选项（平台由 builder 内部固定为 linux/当前架构）
	buildOpts := &builder.BuildOpts{
		SourceDir:        appPaths.CmdAppDir(),
		OutputDir:        appPaths.BuildOutputDir(s.config.GetBuildOutputDir()),
		BinaryNameFormat: s.config.GetBinaryNameFormat(),
	}

	if len(opts) > 0 && opts[0] != nil {
		opt := opts[0]
		// 转换类型，保留所有字段（平台由 builder 内部固定为 linux/当前架构）
		buildOpts = &builder.BuildOpts{
			User:             user,
			App:              app,
			SourceDir:        nonEmpty(opt.SourceDir, buildOpts.SourceDir),
			OutputDir:        nonEmpty(opt.OutputDir, buildOpts.OutputDir),
			BinaryNameFormat: nonEmpty(opt.BinaryNameFormat, buildOpts.BinaryNameFormat),
			BuildTags:        opt.BuildTags,
			LdFlags:          opt.LdFlags,
			Env:              opt.Env,
		}
	}

	ensureSpan := buildtrace.Start(ctx, "runtime.ensure_go_mod", buildtrace.String("go_mod", appPaths.GoModPath()))
	if err := s.ensureAppGoModFile(appPaths); err != nil {
		ensureSpan.Finish(err)
		return nil, fmt.Errorf("failed to ensure app go.mod: %w", err)
	}
	ensureSpan.Finish(nil)

	// 执行编译
	buildSpan := buildtrace.Start(ctx, "runtime.builder_build_app",
		buildtrace.String("source_dir", buildOpts.SourceDir),
		buildtrace.String("output_dir", buildOpts.OutputDir),
	)
	result, err := s.builder.Build(ctx, user, app, buildOpts)
	if err != nil {
		buildSpan.Finish(err)
		logger.Errorf(ctx, "[BuildApp] *** FAILED *** user=%s, app=%s, error=%v", user, app, err)
		return nil, err
	}
	buildSpan.Finish(nil)

	return result, nil
}

// DeleteApp 删除应用
// 新架构：每个版本有独立容器，需要删除所有版本的容器
func (s *AppManageService) DeleteApp(ctx context.Context, user, app string) error {
	logger.Infof(ctx, "[DeleteApp] *** ENTRY *** user=%s, app=%s", user, app)

	// 1. 获取应用的所有版本，删除每个版本的运行时实例
	if s.runtimeDriver != nil {
		// 获取所有版本
		versions, err := s.appRepo.GetAppVersions(ctx, user, app)
		if err != nil {
			logger.Warnf(ctx, "[DeleteApp] Failed to get app versions: %v, will try to delete runtime instances by pattern", err)
			// 如果获取版本失败，尝试通过运行时名称模式查找并删除。
			// 这里可以扩展 AppRuntimeDriver 接口支持按应用查找，暂时先跳过。
		} else {
			// 删除每个版本的运行时实例
			for _, version := range versions {
				ref := AppVersionRef{User: user, App: app, Version: version.Version}
				runtimeName := ref.RuntimeName()

				// 先尝试停止运行时实例（如果正在运行）
				if err := s.runtimeDriver.StopAppVersion(ctx, ref); err != nil {
					logger.Warnf(ctx, "[DeleteApp] Failed to stop runtime instance %s (may not be running): %v", runtimeName, err)
				} else {
					logger.Infof(ctx, "[DeleteApp] Runtime instance %s stopped successfully", runtimeName)
				}

				// 强制删除运行时实例（无论是否正在运行）
				if err := s.runtimeDriver.RemoveAppVersion(ctx, ref); err != nil {
					logger.Warnf(ctx, "[DeleteApp] Failed to remove runtime instance %s: %v", runtimeName, err)
					// 不返回错误，继续删除其他实例
				} else {
					logger.Infof(ctx, "[DeleteApp] Runtime instance %s removed successfully", runtimeName)
				}
			}
			s.MarkContainerCleanupDirty() // 有运行时实例被删，下次巡检周期会做对账
		}
	} else {
		logger.Warnf(ctx, "[DeleteApp] App runtime driver is nil, skipping runtime deletion")
	}

	// 2. 删除应用目录
	appDirRel := newRuntimeAppPaths(s.config.GetBasePath(), user, app).AppDir()
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
	if err := s.appRepo.DeleteAppAndVersions(ctx, user, app); err != nil {
		logger.Warnf(ctx, "[DeleteApp] Failed to delete app and versions from database: %v", err)
	}

	logger.Infof(ctx, "[DeleteApp] *** EXIT *** user=%s, app=%s", user, app)
	return nil
}

// UpdateApp 更新应用（写入源码文件并重新编译部署）
// 如果提供了 sourceFiles，先执行源码文件写入。
// writeOnly 为 true 时仅写文件，不编译不部署。
func (s *AppManageService) UpdateApp(ctx context.Context, user, app string, sourceFiles []*sharedDto.SourceFileWrite, requirement, changeDescription string, writeOnly bool, forceDiff bool) (resp *sharedDto.UpdateAppResp, err error) {
	ctx, trace := buildtrace.Ensure(ctx, "runtime.update_app", user, app)
	defer func() {
		traceSnapshot := trace.Finalize(err)
		if path, persistErr := s.persistUpdateBuildTrace(ctx, user, app, trace); persistErr != nil {
			logger.Warnf(ctx, "[UpdateApp] build trace persist failed: trace_id=%s, error=%v", traceSnapshot.TraceID, persistErr)
		} else if path != "" {
			traceSnapshot = trace.Snapshot()
		}
		if resp != nil {
			resp.BuildTrace = traceSnapshot
		}
		logger.Infof(ctx, "[UpdateApp] build trace summary: trace_id=%s, status=%s, %s",
			traceSnapshot.TraceID, traceSnapshot.Status, buildtrace.Summary(traceSnapshot, 6))
	}()

	logStr := strings.Builder{}
	logStr.WriteString(fmt.Sprintf("[UpdateApp] Starting update: %s/%s\t", user, app))

	prepareSpan := buildtrace.Start(ctx, "runtime.prepare_update_state", buildtrace.String("user", user), buildtrace.String("app", app))
	state, err := s.prepareUpdateAppState(ctx, user, app)
	if err != nil {
		prepareSpan.Finish(err)
		return nil, err
	}
	prepareSpan.Finish(nil)
	s.noteUnknownUpdateVersion(state, &logStr)

	writeSpan := buildtrace.Start(ctx, "runtime.write_source_files", buildtrace.Int("file_count", len(sourceFiles)))
	sourceWriteState, err := s.writeSourceFilesForUpdate(ctx, user, app, sourceFiles)
	if err != nil {
		writeSpan.Finish(err)
		return nil, err
	}
	writeSpan.Finish(nil)

	if writeOnly {
		writeOnlySpan := buildtrace.Start(ctx, "runtime.write_only_response")
		resp = s.buildWriteOnlyUpdateResp(ctx, user, app, state.oldVersion)
		writeOnlySpan.Finish(nil)
		return resp, nil
	}

	releaseSpan := buildtrace.Start(ctx, "runtime.build_and_deploy_release",
		buildtrace.String("old_version", state.oldVersion),
		buildtrace.Bool("force_diff", forceDiff),
	)
	release, err := s.buildAndDeployUpdatedRelease(
		ctx,
		user,
		app,
		state,
		sourceWriteState,
		requirement,
		changeDescription,
		forceDiff,
		&logStr,
	)
	if err != nil {
		releaseSpan.Finish(err)
		return nil, err
	}
	releaseSpan.Finish(nil)

	completeSpan := buildtrace.Start(ctx, "runtime.complete_update_release", buildtrace.String("new_version", release.newVersion))
	resp, err = s.completeUpdatedRelease(ctx, user, app, release)
	completeSpan.Finish(err)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// updateAppStatusToActive 将应用状态更新为active（已激活）
func (s *AppManageService) updateAppStatusToActive(ctx context.Context, user, app string) error {
	appRecord, err := s.appRepo.GetApp(ctx, user, app)
	if err != nil {
		return fmt.Errorf("failed to get app record: %w", err)
	}

	// 更新状态为active
	appRecord.Status = "active"
	if err := s.appRepo.UpdateApp(ctx, appRecord); err != nil {
		return fmt.Errorf("failed to update app status to active: %w", err)
	}

	return nil
}

func (s *AppManageService) buildUpdateAppResp(
	user, app string,
	release *appReleaseResult,
	diffData *sharedDto.DiffData,
) *sharedDto.UpdateAppResp {
	return &sharedDto.UpdateAppResp{
		User:          user,
		App:           app,
		OldVersion:    release.oldVersion,
		NewVersion:    release.newVersion,
		GitCommitHash: release.gitCommitHash, // Git 提交哈希
		Diff:          diffData,              // 转换后的 diff 信息
		Error:         "",
	}
}

// sendUpdateCallbackAndWait 使用 NATS Request/Reply 模式发送 update 回调并等待响应
func (s *AppManageService) sendUpdateCallbackAndWait(ctx context.Context, user, app, version string) (*subjects.Message, error) {
	if s.appControlClient == nil {
		return nil, fmt.Errorf("NATS connection is nil")
	}

	// 构建更新回调请求
	data := map[string]interface{}{"trigger": "update_callback"}
	if s.appDatabaseService != nil && s.appDatabaseService.IsEnabled() {
		capability, err := s.appDatabaseService.IssueCapability(user, app, version, "")
		if err != nil {
			return nil, fmt.Errorf("issue app database capability: %w", err)
		}
		data["db_capability"] = capability
	}
	request := subjects.Message{
		Type:      subjects.MessageTypeStatusOnAppUpdate,
		User:      user,
		App:       app,
		Version:   version,
		Data:      data,
		Timestamp: time.Now(),
	}

	rsp, err := s.appControlClient.RequestUpdateCallback(ctx, user, app, version, &request, 60*time.Second)
	if err != nil {
		logger.Errorf(ctx, "[sendUpdateCallbackAndWait] ❌ Request failed: %v", err)
		return rsp, err
	}
	return rsp, nil
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
	appDir := newRuntimeAppPaths(s.config.GetBasePath(), user, app).AppDir()

	// 检查应用是否存在
	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("app not found: %s/%s", user, app)
	}

	version, err := s.readCurrentVersion(user, app)
	if err != nil {
		return nil, fmt.Errorf("failed to read current version: %w", err)
	}

	return map[string]interface{}{
		"user":    user,
		"app":     app,
		"app_dir": appDir,
		"version": version,
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

// IsAppVersionRuntimeRunning 检查指定应用版本的底层运行时实例是否正在运行。
func (s *AppManageService) IsAppVersionRuntimeRunning(ctx context.Context, user, app, version string) (bool, error) {
	if s.runtimeDriver == nil {
		return false, fmt.Errorf("app runtime driver not available")
	}
	return s.runtimeDriver.IsAppVersionRunning(ctx, AppVersionRef{User: user, App: app, Version: version})
}

// EnsureAppVersionRuntimeRunning 只保证底层运行时实例已启动，不等待应用启动通知。
func (s *AppManageService) EnsureAppVersionRuntimeRunning(ctx context.Context, user, app, version string) error {
	if s.runtimeDriver == nil {
		return fmt.Errorf("app runtime driver not available")
	}

	ref := AppVersionRef{User: user, App: app, Version: version}
	appDirRel := newRuntimeAppPaths(s.config.GetBasePath(), user, app).AppDir()
	spec, err := s.buildAppVersionSpec(ctx, ref, appDirRel)
	if err != nil {
		return err
	}

	if err := s.runtimeDriver.StartAppVersion(ctx, spec); err != nil {
		return fmt.Errorf("failed to start app runtime version: %w", err)
	}
	logger.Infof(ctx, "[EnsureAppVersionRuntimeRunning] Runtime instance %s started successfully", ref.RuntimeName())
	s.MarkContainerCleanupDirty()
	return nil
}

// createVersionContainer 创建版本容器
// 这是新架构的核心方法：每个版本使用独立的容器
