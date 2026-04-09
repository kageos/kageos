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

func (s *AppManageService) createFunctionsForUpdate(
	ctx context.Context,
	user, app string,
	createFunctions []*sharedDto.CreateFunctionInfo,
) ([]string, error) {
	if len(createFunctions) == 0 {
		return nil, nil
	}

	logger.Infof(ctx, "[UpdateApp] 检测到 CreateFunctions，先执行创建函数操作: functionCount=%d", len(createFunctions))

	createResp, err := s.createFunctionService.CreateFunctions(ctx, user, app, createFunctions)
	if err != nil {
		logger.Errorf(ctx, "[UpdateApp] 创建函数失败: error=%v", err)
		return nil, fmt.Errorf("创建函数失败: %w", err)
	}
	if !createResp.Success {
		logger.Errorf(ctx, "[UpdateApp] 创建函数失败: %s", createResp.Message)
		if len(createResp.WrittenFiles) > 0 {
			s.createFunctionService.rollbackFiles(ctx, createResp.WrittenFiles)
		}
		return nil, fmt.Errorf("创建函数失败: %s", createResp.Message)
	}

	logger.Infof(ctx, "[UpdateApp] 创建函数成功: fileCount=%d", len(createResp.WrittenFiles))
	return createResp.WrittenFiles, nil
}

func (s *AppManageService) prepareUpdateAppState(
	ctx context.Context,
	user, app string,
) (*updateAppState, error) {
	appPaths := newRuntimeAppPaths(s.config.AppDir.BasePath, user, app)
	appDirRel := appPaths.AppDir()
	absAppDir, err := filepath.Abs(appDirRel)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}
	if _, err := os.Stat(absAppDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("app not found: %s/%s", user, app)
	}

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
