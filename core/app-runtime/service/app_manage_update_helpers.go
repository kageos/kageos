package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	sharedDto "github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type updateAppState struct {
	appDirRel  string
	absPaths   runtimeAppPaths
	oldVersion string
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

	if s.containerService == nil {
		return fmt.Errorf("container operator not available")
	}

	logger.Infof(ctx, "[UpdateApp] Creating new version container for %s/%s/%s", user, app, newVersion)
	if err := s.createVersionContainer(ctx, user, app, newVersion, state.appDirRel); err != nil {
		logStr.WriteString(fmt.Sprintf("Failed to create version container: %v\t", err))
		return fmt.Errorf("failed to create version container: %w", err)
	}
	logStr.WriteString("New version container created\t")

	s.waitForUpdatedVersionStartup(ctx, user, app, newVersion, waiterChan, logStr)
	s.stopPreviousVersionAfterUpdate(ctx, user, app, state.oldVersion, logStr)

	logStr.WriteString(fmt.Sprintf("Update completed: %s->%s", state.oldVersion, newVersion))
	logger.Infof(ctx, logStr.String())
	return nil
}

func (s *AppManageService) waitForUpdatedVersionStartup(
	ctx context.Context,
	user, app, newVersion string,
	waiterChan <-chan *StartupNotification,
	logStr *strings.Builder,
) {
	logger.Infof(ctx, "[UpdateApp] Waiting for startup notification for %s/%s/%s (first handshake)", user, app, newVersion)

	select {
	case notification := <-waiterChan:
		logStr.WriteString(fmt.Sprintf("Startup confirmed at %s\t", notification.StartTime.Format(time.DateTime)))
		logger.Infof(ctx, "[UpdateApp] ✅ Startup confirmed: %s/%s/%s (first handshake completed)", user, app, newVersion)
		if err := s.updateAppStatusToActive(ctx, user, app); err != nil {
			logger.Warnf(ctx, "[UpdateApp] Failed to update app status to active: %v", err)
		} else {
			logger.Infof(ctx, "[UpdateApp] App status updated to active: %s/%s", user, app)
		}
	case <-time.After(60 * time.Second):
		logStr.WriteString("Startup timeout\t")
		logger.Warnf(ctx, "[UpdateApp] ⚠️ Startup notification timeout for %s/%s/%s, but continue anyway", user, app, newVersion)
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
