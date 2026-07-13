package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/buildtrace"
	appconfig "github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/subjects"
)

func (s *AppManageService) createVersionContainer(ctx context.Context, user, app, version, appDir string) error {
	ref := AppVersionRef{User: user, App: app, Version: version}
	runtimeName := ref.RuntimeName()
	logger.Infof(ctx, "[createVersionContainer] Creating app runtime instance: %s for %s/%s/%s", runtimeName, user, app, version)

	if s.runtimeDriver == nil {
		logger.Errorf(ctx, "App runtime driver not available")
		return fmt.Errorf("app runtime driver not available")
	}

	// 检查运行时实例是否已存在
	checkSpan := buildtrace.Start(ctx, "runtime.check_version_running", buildtrace.String("runtime_name", runtimeName))
	exists, err := s.runtimeDriver.IsAppVersionRunning(ctx, ref)
	if err != nil {
		checkSpan.Finish(err)
		return fmt.Errorf("failed to check app runtime instance existence: %w", err)
	}
	checkSpan.Finish(nil)

	if exists {
		logger.Infof(ctx, "[createVersionContainer] App runtime instance %s already exists and is running; reusing it", runtimeName)
		return nil
	}

	specSpan := buildtrace.Start(ctx, "runtime.build_app_version_spec", buildtrace.String("runtime_name", runtimeName))
	spec, err := s.buildAppVersionSpec(ctx, ref, appDir)
	if err != nil {
		specSpan.Finish(err)
		return err
	}
	specSpan.Finish(nil)
	createSpan := buildtrace.Start(ctx, "runtime.driver_create_app_version",
		buildtrace.String("runtime_name", runtimeName),
		buildtrace.String("image", spec.Image),
	)
	if err := s.runtimeDriver.CreateAppVersion(ctx, spec); err != nil {
		createSpan.Finish(err)
		return err
	}
	createSpan.Finish(nil)
	logger.Infof(ctx, "App runtime instance started successfully with runtime image %s", spec.Image)
	s.MarkContainerCleanupDirty() // 有新运行时实例，下次巡检周期会做对账
	return nil
}

// buildAppVersionSpec 构建应用版本运行时启动参数。
func (s *AppManageService) buildAppVersionSpec(ctx context.Context, ref AppVersionRef, appDir string) (AppVersionSpec, error) {
	logger.Infof(ctx, "Building app runtime spec: %s, appDir: %s, version: %s", ref.RuntimeName(), appDir, ref.Version)

	// 使用 runtime 配置里的应用基础镜像启动容器，挂载应用目录。
	image := s.runtimeConfig.GetContainerBaseImage()
	// 将相对路径转换为绝对路径，避免 Podman 把它当成卷名
	absHostPath, err := filepath.Abs(appDir)
	if err != nil {
		logger.Errorf(ctx, "Failed to get absolute path: %v", err)
		return AppVersionSpec{}, fmt.Errorf("failed to get absolute path: %w", err)
	}
	containerPath := s.runtimeConfig.GetContainerPath()
	binaryName := s.appBinaryName(ref.User, ref.App, ref.Version)
	hostBinaryPath := filepath.Join(absHostPath, "workplace", "bin", "releases", binaryName)

	logger.Infof(ctx, "[buildAppVersionSpec] Runtime mount: image=%s, name=%s, hostPath=%s, containerPath=%s", image, ref.RuntimeName(), absHostPath, containerPath)

	// 设置环境变量
	envVars := []string{}

	// 注入 SDK 配置（专门用于容器内访问平台服务）。
	// SDK 进程启动后会在自身网络命名空间内自动探测 127.0.0.1 /
	// host.containers.internal 等本地候选地址，避免 prod host 网络和 dev bridge
	// 网络使用同一份静态地址。
	//
	// SDK 配置会在构建时注入为环境变量：
	//   - nats_url -> NATS_URL 无凭据 endpoint；URL userinfo 以 Podman secret 注入
	//   - gateway_url -> GATEWAY_URL 环境变量
	//   - env_vars 中的键值对 -> 对应的环境变量
	sdkConfig := appconfig.GetSDKConfig()

	// 从 SDK 配置获取所有环境变量（包括固定字段和 env_vars 中的）
	sdkEnvVars := sdkConfig.GetEnvVars()
	runtimeSecrets, err := prepareAppRuntimeNATSSecret(ctx, ref, hostBinaryPath, sdkEnvVars)
	if err != nil {
		return AppVersionSpec{}, err
	}
	for key, value := range sdkEnvVars {
		envVars = append(envVars, fmt.Sprintf("%s=%s", key, value))
		logger.Infof(ctx, "[buildAppVersionSpec] Injecting SDK env %s into app runtime", key)
	}

	containerWorkDir := filepath.ToSlash(filepath.Join(containerPath, "workplace", "bin"))
	containerBinDir := filepath.ToSlash(filepath.Join(containerPath, "workplace", "bin", "releases"))
	runtimeID := s.runtimeInstanceID()

	// 注入版本信息到环境变量（新架构：每个容器对应特定版本）。
	// 启动脚本优先消费这些 env，metadata 文件仅做兼容兜底。
	envVars = append(envVars,
		fmt.Sprintf("KAGEOS_APP_USER=%s", ref.User),
		fmt.Sprintf("KAGEOS_APP_NAME=%s", ref.App),
		fmt.Sprintf("APP_VERSION=%s", ref.Version),
		fmt.Sprintf("APP_BINARY_NAME=%s", binaryName),
		fmt.Sprintf("KAGEOS_APP_WORK_DIR=%s", containerWorkDir),
		fmt.Sprintf("KAGEOS_APP_BIN_DIR=%s", containerBinDir),
	)
	if runtimeID != "" {
		envVars = append(envVars, fmt.Sprintf("KAGEOS_RUNTIME_INSTANCE_ID=%s", runtimeID))
	}
	logger.Infof(ctx, "[buildAppVersionSpec] Injecting app runtime metadata env keys: %v", []string{
		"KAGEOS_APP_USER",
		"KAGEOS_APP_NAME",
		"APP_VERSION",
		"APP_BINARY_NAME",
		"KAGEOS_APP_WORK_DIR",
		"KAGEOS_APP_BIN_DIR",
		"KAGEOS_RUNTIME_INSTANCE_ID",
	})

	return AppVersionSpec{
		Ref:           ref,
		Image:         image,
		HostPath:      absHostPath,
		ContainerPath: containerPath,
		Command:       []string{"/start.sh"},
		EnvVars:       envVars,
		Secrets:       runtimeSecrets,
	}, nil
}

// prepareAppRuntimeNATSSecret enables secret-file authentication only for app
// binaries that contain the current SDK's fixed credentials path marker. Older
// or temporarily unreadable binaries retain the legacy authenticated NATS_URL
// so an app is not taken offline during the rolling SDK migration.
func prepareAppRuntimeNATSSecret(ctx context.Context, ref AppVersionRef, binaryPath string, sdkEnvVars map[string]string) ([]ContainerSecret, error) {
	rawURL, ok := sdkEnvVars["NATS_URL"]
	if !ok || strings.TrimSpace(rawURL) == "" {
		return nil, nil
	}

	_, hasUserInfo, err := stripAppRuntimeNATSUserInfo(rawURL)
	if err != nil {
		return nil, fmt.Errorf("prepare app NATS runtime credentials: %w", err)
	}
	if !hasUserInfo {
		return nil, nil
	}

	supported, scanErr := appBinaryContainsMarker(binaryPath, []byte(appNATSCredentialsSDKMarker))
	if scanErr != nil {
		logger.Warnf(ctx, "[NATS Credentials Migration] runtime=%s marker_status=unreadable; retaining legacy NATS_URL authentication until this app is rebuilt with the current SDK", ref.RuntimeName())
		return nil, nil
	}
	if !supported {
		logger.Warnf(ctx, "[NATS Credentials Migration] runtime=%s marker_status=missing; retaining legacy NATS_URL authentication until this app is rebuilt with the current SDK", ref.RuntimeName())
		return nil, nil
	}

	return extractAppRuntimeNATSSecret(ref, sdkEnvVars)
}

func appBinaryContainsMarker(path string, marker []byte) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	return readerContainsMarker(file, marker)
}

func readerContainsMarker(reader io.Reader, marker []byte) (bool, error) {
	if len(marker) == 0 {
		return true, nil
	}

	buffer := make([]byte, appBinaryMarkerScanBufferSize+len(marker)-1)
	carried := 0
	for {
		n, err := reader.Read(buffer[carried : carried+appBinaryMarkerScanBufferSize])
		total := carried + n
		if bytes.Contains(buffer[:total], marker) {
			return true, nil
		}

		keep := len(marker) - 1
		if total < keep {
			keep = total
		}
		copy(buffer[:keep], buffer[total-keep:total])
		carried = keep

		if err != nil {
			if err == io.EOF {
				return false, nil
			}
			return false, err
		}
	}
}

// extractAppRuntimeNATSSecret removes URL userinfo from NATS_URL and returns a
// Podman file secret containing the original private URL. The SDK reads the
// mounted secret from /run/secrets/kageos-nats. URLs without authentication are
// left untouched and require no secret, preserving local/no-auth deployments.
func extractAppRuntimeNATSSecret(ref AppVersionRef, sdkEnvVars map[string]string) ([]ContainerSecret, error) {
	rawURL, ok := sdkEnvVars["NATS_URL"]
	if !ok || strings.TrimSpace(rawURL) == "" {
		return nil, nil
	}

	endpoint, hasUserInfo, err := stripAppRuntimeNATSUserInfo(rawURL)
	if err != nil {
		return nil, fmt.Errorf("prepare app NATS runtime credentials: %w", err)
	}
	if !hasUserInfo {
		return nil, nil
	}

	sdkEnvVars["NATS_URL"] = endpoint
	digest := sha256.Sum256([]byte(ref.User + "\x00" + ref.App + "\x00" + ref.Version))
	return []ContainerSecret{
		{
			Name:   fmt.Sprintf("kageos-nats-%x", digest[:12]),
			Target: appNATSCredentialsSecretTarget,
			Data:   []byte(strings.TrimSpace(rawURL)),
		},
	}, nil
}

func stripAppRuntimeNATSUserInfo(raw string) (endpoint string, hasUserInfo bool, err error) {
	servers := strings.Split(strings.TrimSpace(raw), ",")
	for i, server := range servers {
		server = strings.TrimSpace(server)
		parsed, parseErr := url.Parse(server)
		if parseErr != nil || parsed.Host == "" {
			if strings.Contains(server, "@") {
				return "", false, fmt.Errorf("invalid NATS URL containing userinfo")
			}
			servers[i] = server
			continue
		}
		if parsed.User != nil {
			hasUserInfo = true
			parsed.User = nil
		}
		servers[i] = parsed.String()
	}
	return strings.Join(servers, ","), hasUserInfo, nil
}

// stopOldVersionContainer 优雅关闭旧版本容器（三次握手流程）
// 这是新架构的核心方法：优雅关闭旧版本容器
func (s *AppManageService) stopOldVersionContainer(ctx context.Context, user, app, oldVersion string) error {
	ref := AppVersionRef{User: user, App: app, Version: oldVersion}
	runtimeName := ref.RuntimeName()
	logger.Infof(ctx, "[stopOldVersionContainer] Starting graceful shutdown for old runtime instance: %s", runtimeName)

	if s.runtimeDriver == nil {
		logger.Warnf(ctx, "[stopOldVersionContainer] App runtime driver not available, skipping")
		return nil
	}

	// 1. 检查运行时实例是否存在
	exists, err := s.runtimeDriver.IsAppVersionRunning(ctx, ref)
	if err != nil {
		logger.Warnf(ctx, "[stopOldVersionContainer] Failed to check runtime instance existence: %v", err)
		return nil // 不返回错误，继续执行
	}
	if !exists {
		logger.Infof(ctx, "[stopOldVersionContainer] Old runtime instance %s not found, skipping", runtimeName)
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

	// 5. 停止运行时实例（不删除，保留以便快速回滚）
	logger.Infof(ctx, "[stopOldVersionContainer] Stopping runtime instance %s (not removing)", runtimeName)
	if err := s.runtimeDriver.StopAppVersion(ctx, ref); err != nil {
		return fmt.Errorf("failed to stop app runtime instance: %w", err)
	}

	logger.Infof(ctx, "[stopOldVersionContainer] Old runtime instance %s stopped successfully", runtimeName)
	s.MarkContainerCleanupDirty() // 有运行时实例被停，下次巡检周期会做对账
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

	if err := s.appControlClient.PublishShutdown(ctx, user, app, version, &message); err != nil {
		return err
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

	// 关闭旧版本（基于完整静默期，优先服务稳定）
	// 注意：这里简化逻辑，因为内存中的版本信息不包含创建时间
	// 实际应用中，应该根据业务需求决定关闭策略
	versionsToShutdown := runningVersions[keepVersions:]
	for _, version := range versionsToShutdown {
		if !s.isVersionQuietForCleanup(ctx, user, app, version, "ShutdownOldVersions") {
			continue
		}

		if err := s.shutdownVersionGracefullyForCleanup(ctx, user, app, version, "ShutdownOldVersions"); err != nil {
			logger.Warnf(ctx, "[ShutdownOldVersions] 本轮跳过版本 %s: %v", version, err)
		} else {
			logger.Infof(ctx, "[ShutdownOldVersions] Version %s closed gracefully", version)
		}
	}

	return nil
}

// StartCleanupTask 启动定时清理任务
// 进程级清理 + 容器级巡检 + release 二进制清理合并为「一次完整清理」，在凌晨 4 点与有变动时执行。
func (s *AppManageService) StartAppVersion(ctx context.Context, user, app, version string) error {
	logger.Infof(ctx, "[StartAppVersion] Starting version %s/%s/%s", user, app, version)

	// 先检查应用是否已经在运行（避免重复启动）
	if s.appDiscoveryService != nil {
		if s.appDiscoveryService.IsAppVersionRunning(user, app, version) {
			logger.Infof(ctx, "[StartAppVersion] Version %s/%s/%s is already running, skipping startup", user, app, version)
			return nil
		}
	}

	ref := AppVersionRef{User: user, App: app, Version: version}
	runtimeName := ref.RuntimeName()

	// 注册启动等待器（统一在外层注册）
	waiterChan := s.registerStartupWaiter(user, app, version)
	// 确保在方法结束时清理等待器
	defer s.unregisterStartupWaiter(user, app, version)

	if s.runtimeDriver == nil {
		return fmt.Errorf("app runtime driver not available")
	}

	// 检查运行时实例是否存在且运行中
	exists, err := s.runtimeDriver.IsAppVersionRunning(ctx, ref)
	if err != nil {
		logger.Warnf(ctx, "[StartAppVersion] Failed to check runtime status: %v, will try to start", err)
		exists = false
	}

	if exists {
		// 运行时实例已存在且运行中，应用应该已经启动，等待启动通知
		logger.Infof(ctx, "[StartAppVersion] Runtime instance %s already exists and is running, waiting for startup notification", runtimeName)
	} else {
		// 运行时实例不存在或已停止，需要创建或启动实例
		if err := s.EnsureAppVersionRuntimeRunning(ctx, user, app, version); err != nil {
			return err
		}
		logger.Infof(ctx, "[StartAppVersion] Runtime instance %s started successfully", runtimeName)
	}

	startupTimeout := s.appStartupNotificationTimeout()
	logger.Infof(ctx, "[StartAppVersion] Waiting for startup notification from version %s (timeout: %s)...", version, startupTimeout)

	notification, err := s.waitForStartupNotificationOrRuntimeExit(ctx, ref, waiterChan, startupTimeout)
	if err != nil {
		logger.Warnf(ctx, "[StartAppVersion] Failed waiting for startup notification from version %s after %s: %v", version, startupTimeout, err)
		return err
	}
	logger.Infof(ctx, "[StartAppVersion] Received startup notification: %s/%s/%s, status=%s",
		notification.User, notification.App, notification.Version, notification.Status)

	if notification.Status == "running" {
		logger.Infof(ctx, "[StartAppVersion] Version %s started successfully", version)
		return nil
	}
	if notification.Error != "" {
		return fmt.Errorf("app startup failed: %s", notification.Error)
	}
	return fmt.Errorf("app started but status is not running: %s", notification.Status)
}
