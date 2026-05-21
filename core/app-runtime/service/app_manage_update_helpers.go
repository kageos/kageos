package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	sharedDto "github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
)

type updateAppState struct {
	appDirRel  string
	absPaths   runtimeAppPaths
	oldVersion string
}

func (s *AppManageService) noteUnknownUpdateVersion(state *updateAppState, logStr *strings.Builder) {
	if state != nil && state.oldVersion == "unknown" {
		logStr.WriteString("Failed to get current version\t")
	}
}

func (s *AppManageService) buildWriteOnlyUpdateResp(
	ctx context.Context,
	user, app, version string,
) *sharedDto.UpdateAppResp {
	logger.Infof(ctx, "[UpdateApp] WriteOnly=true，仅写文件不编译不部署")
	return &sharedDto.UpdateAppResp{
		User:       user,
		App:        app,
		OldVersion: version,
		NewVersion: version,
	}
}

func (s *AppManageService) completeUpdatedRelease(
	ctx context.Context,
	user, app string,
	release *appReleaseResult,
) (*sharedDto.UpdateAppResp, error) {
	diffData, err := s.requestRequiredVersionDiff(ctx, user, app, release.newVersion, "UpdateApp")
	if err != nil {
		logger.Warnf(ctx, "[UpdateApp] Aborting update result to avoid API state drift")
		return nil, err
	}

	return s.buildUpdateAppResp(user, app, release, diffData), nil
}

func (s *AppManageService) writeSourceFilesForUpdate(
	ctx context.Context,
	user, app string,
	sourceFiles []*sharedDto.SourceFileWrite,
) (*batchWriteState, error) {
	if len(sourceFiles) == 0 {
		return nil, nil
	}
	if s.workspaceFileService == nil {
		return nil, fmt.Errorf("workspace file service not available")
	}

	logger.Infof(ctx, "[UpdateApp] 检测到 SourceFiles，先写入源码文件: fileCount=%d", len(sourceFiles))

	state, err := s.workspaceFileService.writeSourceFiles(ctx, user, app, sourceFiles)
	if err != nil {
		logger.Errorf(ctx, "[UpdateApp] 写入源码文件失败: error=%v", err)
		return nil, fmt.Errorf("写入源码文件失败: %w", err)
	}

	logger.Infof(ctx, "[UpdateApp] 源码文件写入成功: fileCount=%d", len(state.writtenPaths))
	return state, nil
}

func (s *AppManageService) prepareUpdateAppState(
	ctx context.Context,
	user, app string,
) (*updateAppState, error) {
	appPaths, absAppDir, err := s.prepareExistingAppPaths(user, app)
	if err != nil {
		return nil, err
	}
	appDirRel := appPaths.AppDir()

	if err := s.appRepo.EnsureAppExists(user, app); err != nil {
		logger.Warnf(ctx, "[UpdateApp] ensure app record failed (non-blocking): %v", err)
	}

	absPaths := newRuntimeAppPathsFromAppDir(absAppDir, user, app)
	oldVersion := s.getReleaseCurrentVersion(ctx, appPaths, app, "UpdateApp")

	return &updateAppState{
		appDirRel:  appDirRel,
		absPaths:   absPaths,
		oldVersion: oldVersion,
	}, nil
}

func (s *AppManageService) prepareExistingAppPaths(user, app string) (runtimeAppPaths, string, error) {
	appPaths := newRuntimeAppPaths(s.config.GetBasePath(), user, app)
	appDirRel := appPaths.AppDir()
	absAppDir, err := filepath.Abs(appDirRel)
	if err != nil {
		return runtimeAppPaths{}, "", fmt.Errorf("failed to get absolute path: %w", err)
	}
	if _, err := os.Stat(absAppDir); err != nil {
		if os.IsNotExist(err) {
			return runtimeAppPaths{}, "", fmt.Errorf("app not found: %s/%s", user, app)
		}
		return runtimeAppPaths{}, "", fmt.Errorf("检查应用目录失败: %w", err)
	}
	return appPaths, absAppDir, nil
}

func (s *AppManageService) deployUpdatedVersion(
	ctx context.Context,
	user, app string,
	state *updateAppState,
	newVersion string,
	logStr *strings.Builder,
) error {
	waiterChan := s.registerStartupWaiter(user, app, newVersion)
	defer s.unregisterStartupWaiter(user, app, newVersion)

	if s.runtimeDriver == nil {
		return fmt.Errorf("app runtime driver not available")
	}

	logger.Infof(ctx, "[UpdateApp] Creating new version container for %s/%s/%s", user, app, newVersion)
	if err := s.createVersionContainer(ctx, user, app, newVersion, state.appDirRel); err != nil {
		logStr.WriteString(fmt.Sprintf("Failed to create version container: %v\t", err))
		return fmt.Errorf("failed to create version container: %w", err)
	}
	logStr.WriteString("New version container created\t")

	if err := s.waitForUpdatedVersionStartup(ctx, user, app, newVersion, waiterChan, logStr); err != nil {
		return err
	}
	s.stopPreviousVersionAfterUpdate(ctx, user, app, state.oldVersion, logStr)

	logStr.WriteString(fmt.Sprintf("Update completed: %s->%s", state.oldVersion, newVersion))
	logger.Infof(ctx, logStr.String())
	return nil
}

func (s *AppManageService) buildAndDeployUpdatedRelease(
	ctx context.Context,
	user, app string,
	state *updateAppState,
	sourceWriteState *batchWriteState,
	requirement string,
	changeDescription string,
	forceDiff bool,
	logStr *strings.Builder,
) (*appReleaseResult, error) {
	release, err := s.prepareAppRelease(
		ctx,
		user,
		app,
		state.absPaths,
		state.oldVersion,
		"UpdateApp",
		requirement,
		changeDescription,
	)
	if err != nil {
		s.rollbackWrittenFilesAfterFailedBuild(ctx, "UpdateApp", sourceWriteState)
		return nil, fmt.Errorf("failed to build app: %w", err)
	}

	if forceDiff {
		s.clearAPILogs(ctx, user, app, "UpdateApp")
	}

	if err := s.deployUpdatedVersion(ctx, user, app, state, release.newVersion, logStr); err != nil {
		return nil, err
	}

	return release, nil
}

func (s *AppManageService) rollbackWrittenFilesAfterFailedBuild(
	ctx context.Context,
	logPrefix string,
	sourceWriteState *batchWriteState,
) {
	if sourceWriteState == nil || len(sourceWriteState.writtenPaths) == 0 {
		return
	}

	logger.Warnf(ctx, "[%s] 编译失败，开始回滚已写入的文件: fileCount=%d", logPrefix, len(sourceWriteState.writtenPaths))
	s.workspaceFileService.rollbackWriteState(ctx, sourceWriteState)
}

func (s *AppManageService) waitForUpdatedVersionStartup(
	ctx context.Context,
	user, app, newVersion string,
	waiterChan <-chan *StartupNotification,
	logStr *strings.Builder,
) error {
	startupTimeout := s.appStartupNotificationTimeout()
	logger.Infof(ctx, "[UpdateApp] Waiting for startup notification for %s/%s/%s (first handshake, timeout: %s)", user, app, newVersion, startupTimeout)

	select {
	case notification := <-waiterChan:
		if notification.Status != "" && notification.Status != "running" {
			if notification.Error != "" {
				return fmt.Errorf("app startup failed: %s", notification.Error)
			}
			return fmt.Errorf("app startup failed with status: %s", notification.Status)
		}
		logStr.WriteString(fmt.Sprintf("Startup confirmed at %s\t", notification.StartTime.Format(time.DateTime)))
		logger.Infof(ctx, "[UpdateApp] ✅ Startup confirmed: %s/%s/%s (first handshake completed)", user, app, newVersion)
		if err := s.updateAppStatusToActive(ctx, user, app); err != nil {
			logger.Warnf(ctx, "[UpdateApp] Failed to update app status to active: %v", err)
		} else {
			logger.Infof(ctx, "[UpdateApp] App status updated to active: %s/%s", user, app)
		}
		return nil
	case <-time.After(startupTimeout):
		running, err := s.runtimeDriver.IsAppVersionRunning(ctx, AppVersionRef{User: user, App: app, Version: newVersion})
		if err != nil {
			logger.Warnf(ctx, "[UpdateApp] Failed to check runtime after startup timeout: %v", err)
		}
		if running {
			logStr.WriteString("Startup notification missed but runtime is running\t")
			logger.Infof(ctx, "[UpdateApp] Runtime %s/%s/%s is running after missed startup notification; treating as started", user, app, newVersion)
			if err := s.updateAppStatusToActive(ctx, user, app); err != nil {
				logger.Warnf(ctx, "[UpdateApp] Failed to update app status to active: %v", err)
			} else {
				logger.Infof(ctx, "[UpdateApp] App status updated to active: %s/%s", user, app)
			}
			return nil
		}
		logStr.WriteString("Startup timeout\t")
		return fmt.Errorf("timeout waiting for app startup notification: %s/%s/%s", user, app, newVersion)
	}
}

func (s *AppManageService) stopPreviousVersionAfterUpdate(
	ctx context.Context,
	user, app, oldVersion string,
	logStr *strings.Builder,
) {
	if oldVersion == "" || oldVersion == "unknown" {
		logger.Infof(ctx, "[UpdateApp] No old version to stop (oldVersion: %s)", oldVersion)
		return
	}

	logger.Infof(ctx, "[UpdateApp] Starting graceful shutdown for old version %s/%s/%s", user, app, oldVersion)
	if err := s.stopOldVersionContainer(ctx, user, app, oldVersion); err != nil {
		logStr.WriteString(fmt.Sprintf("Failed to stop old container: %v\t", err))
		logger.Warnf(ctx, "[UpdateApp] ⚠️ Failed to stop old container: %v, but continue anyway", err)
		return
	}

	logStr.WriteString("Old container stopped gracefully\t")
	logger.Infof(ctx, "[UpdateApp] ✅ Old container stopped gracefully: %s/%s/%s", user, app, oldVersion)
}
