package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/kageos/kageos/core/app-runtime/model"
	"github.com/kageos/kageos/pkg/logger"
)

func (s *AppManageService) StartCleanupTask(ctx context.Context) {
	const containerCleanupCronExpr = "0 4 * * *" // 每天凌晨 4 点（cron：分 时 日 月 周）
	logger.Infof(ctx, "[CleanupTask] 启动定时清理 | 进程级+容器级+二进制=cron(%s)+有变动时 | 顺序=进程级→容器级→二进制 | 保留版本数=%d",
		containerCleanupCronExpr, maxKeepVersions)

	// 凌晨 4 点：先进程级（按当前版本停非当前），再容器级（保留最近 3 版本并删除多余），最后裁剪旧二进制。
	s.containerCleanupCron = cron.New(cron.WithLocation(time.Local))
	_, err := s.containerCleanupCron.AddFunc(containerCleanupCronExpr, func() {
		logger.Infof(ctx, "[CleanupTask] cron 触发 | 执行进程级清理 + 容器级巡检 + release 二进制清理 + workplace(file-cache/output/uploads)清空")
		s.runAllCleanups(ctx)
		s.runWorkplaceTempCleanup(ctx)
	})
	if err != nil {
		logger.Warnf(ctx, "[CleanupTask] cron 添加失败: %v，将仅依赖有变动时触发", err)
	} else {
		s.containerCleanupCron.Start()
	}

	// 每 1 分钟检查是否有“有变动”标记，有则执行一次完整清理（进程级+容器级+二进制）
	s.containerCleanupTicker = time.NewTicker(1 * time.Minute)

	go func() {
		defer s.containerCleanupTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.Infof(ctx, "[CleanupTask] 清理任务已停止 (context canceled)")
				return
			case <-s.containerCleanupDone:
				logger.Infof(ctx, "[CleanupTask] 清理任务已停止 (signal)")
				return
			case <-s.containerCleanupTicker.C:
				s.maybeRunContainerLevelCleanup(ctx)
			}
		}
	}()
}

// runAllCleanups 执行一次完整清理：先进程级（按当前版本停非当前），再容器级（保留最近 3 版本并删除多余），最后裁剪旧 release 二进制。
func (s *AppManageService) runAllCleanups(ctx context.Context) {
	s.performCleanup(ctx)        // 进程级：按 current_version 停掉非当前且无流量的版本
	s.containerLevelCleanup(ctx) // 容器级：每应用保留最近 3 版本，其余 stop+remove
	s.releaseBinaryCleanup(ctx)  // 文件级：每应用保留 current + 最近 3 个 release 二进制
}

// runWorkplaceTempCleanup 清空各应用 workplace 下的临时目录（全部删除，无需保留）
func (s *AppManageService) runWorkplaceTempCleanup(ctx context.Context) {
	apps, err := s.getAllApps(ctx)
	if err != nil {
		logger.Errorf(ctx, "[WorkplaceCleanup] 获取应用列表失败: %v", err)
		return
	}
	for _, app := range apps {
		appPaths := newRuntimeAppPaths(s.config.GetBasePath(), app.User, app.App)
		for _, subdir := range []string{"file-cache", "output", "uploads"} {
			dir := appPaths.WorkplaceSubDir(subdir)
			if _, err := os.Stat(dir); err != nil {
				if !os.IsNotExist(err) {
					logger.Warnf(ctx, "[WorkplaceCleanup] 检查目录失败 %s: %v", dir, err)
				}
				continue
			}
			if err := os.RemoveAll(dir); err != nil {
				logger.Warnf(ctx, "[WorkplaceCleanup] 清空失败 %s: %v", dir, err)
				continue
			}
			if err := os.MkdirAll(dir, 0755); err != nil {
				logger.Warnf(ctx, "[WorkplaceCleanup] 重建目录失败 %s: %v", dir, err)
				continue
			}
			logger.Infof(ctx, "[WorkplaceCleanup] 已清空: %s/%s workplace/%s", app.User, app.App, subdir)
		}
	}
}

// StopCleanupTask 停止定时清理任务
func (s *AppManageService) StopCleanupTask(ctx context.Context) {
	if s.containerCleanupTicker != nil {
		s.containerCleanupTicker.Stop()
	}
	if s.containerCleanupCron != nil {
		s.containerCleanupCron.Stop()
	}

	select {
	case s.cleanupDone <- struct{}{}:
	default:
	}
	select {
	case s.containerCleanupDone <- struct{}{}:
	default:
	}

	logger.Infof(ctx, "[AppManageService] Cleanup tasks stopped")
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
	return s.appRepo.GetAllApps(ctx)
}

const (
	// maxKeepVersions 每个应用保留的最大容器版本数
	maxKeepVersions = 3

	// versionShutdownQuietPeriod 是旧版本释放前必须满足的静默期。
	// 宁可晚释放，也不能在 app-server 版本指针、缓存或长请求尚未稳定时释放旧实例。
	versionShutdownQuietPeriod = 10 * time.Minute

	// versionGracefulShutdownTimeout 只限制本轮清理等待 close 通知的时间。
	// 超时后跳过本轮，不强杀，下一轮继续尝试。
	versionGracefulShutdownTimeout = 45 * time.Second
)

// appContainerInfo 单个应用的容器信息（用于按版本排序清理）
type appContainerInfo struct {
	containerName string
	version       string
	versionNum    int // 从 "v1","v2"... 解析出的数字，用于排序
	exited        bool
}

// maybeRunContainerLevelCleanup 仅在有变动时执行一次完整清理（进程级+容器级）
func (s *AppManageService) maybeRunContainerLevelCleanup(ctx context.Context) {
	s.containerCleanupMu.Lock()
	dirty := s.containerCleanupDirty
	if dirty {
		s.containerCleanupDirty = false
	}
	s.containerCleanupMu.Unlock()

	if !dirty {
		return
	}

	logger.Infof(ctx, "[CleanupTask] 检测到版本/容器变动，执行一次完整清理（进程级→容器级→二进制）")
	s.runAllCleanups(ctx)
}

// MarkContainerCleanupDirty 标记“有容器/版本变动”，下次巡检周期会执行一次对账
func (s *AppManageService) MarkContainerCleanupDirty() {
	s.containerCleanupMu.Lock()
	defer s.containerCleanupMu.Unlock()
	s.containerCleanupDirty = true
}

func (s *AppManageService) isVersionQuietForCleanup(ctx context.Context, user, app, version, logPrefix string) bool {
	if s.QPSTracker == nil {
		logger.Warnf(ctx, "[%s] QPS tracker unavailable, skip cleanup for %s/%s/%s", logPrefix, user, app, version)
		return false
	}
	s.QPSTracker.ObserveVersion(user, app, version)
	if !s.QPSTracker.IsIdleFor(user, app, version, versionShutdownQuietPeriod) {
		logger.Infof(ctx, "[%s] Version is not quiet long enough, skip cleanup: %s/%s/%s quietPeriod=%s",
			logPrefix, user, app, version, versionShutdownQuietPeriod)
		s.MarkContainerCleanupDirty()
		return false
	}
	return true
}

func (s *AppManageService) shutdownVersionGracefullyForCleanup(ctx context.Context, user, app, version, logPrefix string) error {
	closeWaiterChan := s.registerCloseWaiter(user, app, version)
	defer s.unregisterCloseWaiter(user, app, version)

	if err := s.ShutdownAppVersion(ctx, user, app, version); err != nil {
		s.MarkContainerCleanupDirty()
		return fmt.Errorf("send shutdown command: %w", err)
	}

	select {
	case notification := <-closeWaiterChan:
		logger.Infof(ctx, "[%s] Received close notification for %s/%s/%s at %s",
			logPrefix, notification.User, notification.App, notification.Version, notification.CloseTime.Format(time.DateTime))
		s.MarkContainerCleanupDirty()
		return nil
	case <-time.After(versionGracefulShutdownTimeout):
		s.MarkContainerCleanupDirty()
		return fmt.Errorf("timeout waiting for close notification after %s", versionGracefulShutdownTimeout)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// containerLevelCleanup 运行时实例级对账巡检
// 策略：仅处理 app 表中已注册的应用（runtime 构建的），每个应用保留最近 maxKeepVersions 个版本，更老的全部清理。
// 非 runtime 构建的实例（未在 app 表注册的）一律不碰，保证基础设施等安全。
func (s *AppManageService) containerLevelCleanup(ctx context.Context) {
	if s.runtimeDriver == nil || !s.runtimeDriver.IsAvailable() {
		logger.Debugf(ctx, "[ContainerCleanup] 跳过巡检: runtimeDriver 不可用或未运行")
		return
	}

	cleanupStart := time.Now()

	// 1. 获取 app 表中已注册的应用，只清理这些应用的多余容器
	registeredApps, err := s.appRepo.GetAllApps(ctx)
	if err != nil {
		logger.Warnf(ctx, "[ContainerCleanup] 获取已注册应用列表失败: %v，跳过本次巡检", err)
		return
	}
	registeredAppKeys := make(map[string]bool)
	for _, a := range registeredApps {
		registeredAppKeys[a.User+"/"+a.App] = true
	}

	runtimeInstances, err := s.runtimeDriver.ListAppVersions(ctx)
	if err != nil {
		logger.Warnf(ctx, "[ContainerCleanup] 获取运行时实例列表失败: %v", err)
		return
	}

	// 2. 按 "user/app" 分组收集应用运行时实例，且仅收集在 app 表中已注册的应用
	appContainers := make(map[string][]appContainerInfo) // key: "user/app"
	unregisteredCount := 0

	for _, instance := range runtimeInstances {
		appKey := instance.Ref.AppKey()
		if !registeredAppKeys[appKey] {
			unregisteredCount++
			continue
		}

		vNum := parseVersionNumber(instance.Ref.Version)
		appContainers[appKey] = append(appContainers[appKey], appContainerInfo{
			containerName: instance.RuntimeName,
			version:       instance.Ref.Version,
			versionNum:    vNum,
			exited:        !instance.Running,
		})
	}

	totalAppContainers := 0
	for _, cs := range appContainers {
		totalAppContainers += len(cs)
	}

	logger.Infof(ctx, "[ContainerCleanup] 开始巡检 | 总应用运行时实例=%d | 已注册应用实例=%d（%d个应用）| 未注册=%d | 保留策略=每应用最近%d版本",
		len(runtimeInstances), totalAppContainers, len(appContainers), unregisteredCount, maxKeepVersions)

	var cleanedExited, cleanedRunning, skippedTraffic, failedClean int

	for appKey, containers := range appContainers {
		if len(containers) <= maxKeepVersions {
			continue
		}

		sortContainersByVersion(containers)

		// 日志：列出该应用所有版本
		kept := containers[:maxKeepVersions]
		toRemove := containers[maxKeepVersions:]

		keptVersions := make([]string, len(kept))
		for i, c := range kept {
			status := "运行中"
			if c.exited {
				status = "已停止"
			}
			keptVersions[i] = fmt.Sprintf("%s(%s)", c.version, status)
		}
		removeVersions := make([]string, len(toRemove))
		for i, c := range toRemove {
			status := "运行中"
			if c.exited {
				status = "已停止"
			}
			removeVersions[i] = fmt.Sprintf("%s(%s)", c.version, status)
		}

		logger.Infof(ctx, "[ContainerCleanup] 应用 %s | 共%d个版本 | 保留=%v | 待清理=%v",
			appKey, len(containers), keptVersions, removeVersions)

		parts := strings.SplitN(appKey, "/", 2)
		user, app := parts[0], parts[1]

		for _, info := range toRemove {
			if info.exited {
				removeStart := time.Now()
				ref := AppVersionRef{User: user, App: app, Version: info.version}
				if rmErr := s.runtimeDriver.RemoveAppVersion(ctx, ref); rmErr != nil {
					logger.Warnf(ctx, "[ContainerCleanup] ❌ 删除已停止容器失败 | 容器=%s | 错误=%v", info.containerName, rmErr)
					failedClean++
				} else {
					logger.Infof(ctx, "[ContainerCleanup] ✅ 已删除停止容器 | 容器=%s | 版本=%s | 耗时=%s",
						info.containerName, info.version, time.Since(removeStart).Round(time.Millisecond))
					cleanedExited++
				}
			} else {
				if s.isVersionQuietForCleanup(ctx, user, app, info.version, "ContainerCleanup") {
					stopStart := time.Now()
					logger.Infof(ctx, "[ContainerCleanup] 请求旧容器优雅退出 | 容器=%s | 版本=%s（静默期=%s）",
						info.containerName, info.version, versionShutdownQuietPeriod)
					if err := s.shutdownVersionGracefullyForCleanup(ctx, user, app, info.version, "ContainerCleanup"); err != nil {
						logger.Warnf(ctx, "[ContainerCleanup] ⏭ 本轮跳过运行中容器 | 容器=%s | 版本=%s | 原因=%v",
							info.containerName, info.version, err)
						failedClean++
					} else {
						logger.Infof(ctx, "[ContainerCleanup] ✅ 旧容器已优雅退出，后续巡检删除已停止实例 | 容器=%s | 版本=%s | 耗时=%s",
							info.containerName, info.version, time.Since(stopStart).Round(time.Millisecond))
						cleanedRunning++
					}
				} else {
					logger.Infof(ctx, "[ContainerCleanup] ⏭ 跳过运行中容器 | 容器=%s | 版本=%s | 原因=未满足静默期",
						info.containerName, info.version)
					skippedTraffic++
				}
			}
		}
	}

	totalCleaned := cleanedExited + cleanedRunning
	logger.Infof(ctx, "[ContainerCleanup] 巡检完成 | 耗时=%s | 清理=%d（已停止=%d + 运行中=%d）| 跳过=%d（有流量）| 失败=%d",
		time.Since(cleanupStart).Round(time.Millisecond), totalCleaned, cleanedExited, cleanedRunning, skippedTraffic, failedClean)
}

// parseVersionNumber 从 "v1","v2","v10" 等版本字符串中提取数字部分
func parseVersionNumber(version string) int {
	v := strings.TrimPrefix(version, "v")
	num := 0
	for _, ch := range v {
		if ch >= '0' && ch <= '9' {
			num = num*10 + int(ch-'0')
		} else {
			break
		}
	}
	return num
}

// sortContainersByVersion 按版本号降序排列（最新版本在前面）
func sortContainersByVersion(containers []appContainerInfo) {
	for i := 1; i < len(containers); i++ {
		for j := i; j > 0 && containers[j].versionNum > containers[j-1].versionNum; j-- {
			containers[j], containers[j-1] = containers[j-1], containers[j]
		}
	}
}

// CleanupNonCurrentVersions 清理非当前版本的静默版本。
// 策略：只保留 current_version（metadata 中的当前版本），其他版本满足完整静默期后才发起优雅关闭。
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

		if !s.isVersionQuietForCleanup(ctx, user, app, version.Version, "CleanupNonCurrentVersions") {
			continue
		}

		if err := s.shutdownVersionGracefullyForCleanup(ctx, user, app, version.Version, "CleanupNonCurrentVersions"); err != nil {
			logger.Warnf(ctx, "[CleanupNonCurrentVersions] 本轮跳过非当前版本 | %s/%s/%s | 原因=%v", user, app, version.Version, err)
			continue
		}
	}

	return nil
}

// getCurrentVersion 获取应用的当前版本（从 metadata/current_version.txt）
func (s *AppManageService) getCurrentVersion(ctx context.Context, user, app string) (string, error) {
	return s.readCurrentVersion(user, app)
}

// StartAppVersion 启动指定版本的应用（兜底启动）
// 用于应用挂了或更新失败时重新启动目标版本
// 新架构：每个版本有独立容器，直接创建或启动版本容器
