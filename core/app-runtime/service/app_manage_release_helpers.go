package service

import (
	"context"
	"os"

	sharedDto "github.com/ai-agent-os/ai-agent-os/dto"
	appPkg "github.com/ai-agent-os/ai-agent-os/pkg/app"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type appReleaseResult struct {
	oldVersion    string
	newVersion    string
	gitCommitHash string
	diff          *sharedDto.DiffData
}

func (s *AppManageService) finalizeWrittenAppChanges(
	ctx context.Context,
	user, app string,
	appPaths runtimeAppPaths,
) (*appReleaseResult, error) {
	oldVersion := s.getReleaseCurrentVersion(ctx, appPaths, app, "BatchWriteFiles")
	release, err := s.prepareAppRelease(ctx, user, app, appPaths, oldVersion, "BatchWriteFiles", "", "")
	if err != nil {
		return nil, err
	}
	release.diff = s.collectVersionDiffFromTemporaryContainer(ctx, user, app, release.newVersion, appPaths.AppDir())
	return release, nil
}

func (s *AppManageService) prepareAppRelease(
	ctx context.Context,
	user, app string,
	appPaths runtimeAppPaths,
	oldVersion string,
	logPrefix string,
	requirement string,
	changeDescription string,
) (*appReleaseResult, error) {
	newVersion, err := s.buildAppForWrittenChanges(ctx, user, app, appPaths)
	if err != nil {
		return nil, err
	}

	s.updateReleaseVersionMetadata(ctx, user, app, appPaths, newVersion, logPrefix)
	gitCommitHash := s.commitRelease(ctx, user, app, newVersion, logPrefix, requirement, changeDescription)

	return &appReleaseResult{
		oldVersion:    oldVersion,
		newVersion:    newVersion,
		gitCommitHash: gitCommitHash,
	}, nil
}

func (s *AppManageService) getReleaseCurrentVersion(ctx context.Context, appPaths runtimeAppPaths, app string, logPrefix string) string {
	vm := appPkg.NewVersionManager(appPaths.UserDir(), app)
	oldVersion, err := vm.GetCurrentVersion()
	if err != nil {
		logger.Warnf(ctx, "[%s] 获取当前版本失败: %v，使用 unknown", logPrefix, err)
		return "unknown"
	}
	return oldVersion
}

func (s *AppManageService) buildAppForWrittenChanges(
	ctx context.Context,
	user, app string,
	appPaths runtimeAppPaths,
) (string, error) {
	buildOpts := &BuildOpts{
		SourceDir:        appPaths.CmdAppDir(),
		OutputDir:        appPaths.BuildOutputDir(s.config.Build.OutputDir),
		BinaryNameFormat: s.config.Build.BinaryNameFormat,
	}

	buildResult, err := s.BuildApp(ctx, user, app, buildOpts)
	if err != nil {
		return "", err
	}
	return buildResult.Version, nil
}

func (s *AppManageService) updateReleaseVersionMetadata(
	ctx context.Context,
	user, app string,
	appPaths runtimeAppPaths,
	newVersion string,
	logPrefix string,
) {
	versionFile := appPaths.VersionJSONPath()
	if _, err := os.Stat(versionFile); os.IsNotExist(err) {
		logger.Infof(ctx, "[%s] Version file not found, creating initial version file...", logPrefix)
		if err := s.createVersionFiles(user, app); err != nil {
			logger.Warnf(ctx, "[%s] 创建版本文件失败: %v，继续执行", logPrefix, err)
		}
	}

	if err := s.updateVersionJson(appPaths.AppDir(), user, app, newVersion); err != nil {
		logger.Warnf(ctx, "[%s] 更新版本信息失败: %v，继续执行", logPrefix, err)
	}
}

func (s *AppManageService) commitRelease(
	ctx context.Context,
	user, app, newVersion string,
	logPrefix string,
	requirement string,
	changeDescription string,
) string {
	hash, err := s.commitToGit(ctx, user, app, newVersion, requirement, changeDescription)
	if err != nil {
		logger.Warnf(ctx, "[%s] Git 提交失败: %v，继续执行", logPrefix, err)
		return ""
	}
	return hash
}
