package service

import (
	"context"
	"os"

	sharedDto "github.com/kageos/kageos/dto"
	appPkg "github.com/kageos/kageos/pkg/app"
	"github.com/kageos/kageos/pkg/buildtrace"
	"github.com/kageos/kageos/pkg/logger"
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
	forceDiff bool,
	logPrefix string,
) (*appReleaseResult, error) {
	if logPrefix == "" {
		logPrefix = "BatchWriteFiles"
	}

	oldVersion := s.getReleaseCurrentVersion(ctx, appPaths, app, logPrefix)
	release, err := s.prepareAppRelease(ctx, user, app, appPaths, oldVersion, logPrefix, "", "")
	if err != nil {
		return nil, err
	}
	if forceDiff {
		s.clearAPILogs(ctx, user, app, logPrefix)
	}
	release.diff = s.collectVersionDiffFromTemporaryContainer(ctx, user, app, release.newVersion, appPaths.AppDir())
	return release, nil
}

func (s *AppManageService) clearAPILogs(ctx context.Context, user, app, logPrefix string) {
	apiLogsDir := newRuntimeAppPaths(s.config.GetBasePath(), user, app).WorkplaceSubDir("api-logs")
	if err := os.RemoveAll(apiLogsDir); err != nil {
		logger.Warnf(ctx, "[%s] 清理 api-logs 失败: path=%s, error=%v", logPrefix, apiLogsDir, err)
		return
	}
	logger.Infof(ctx, "[%s] 已清理 api-logs，下一次 SDK diff 将重新计算新增 API: path=%s", logPrefix, apiLogsDir)
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
	buildSpan := buildtrace.Start(ctx, "runtime.build_app_for_written_changes", buildtrace.String("old_version", oldVersion))
	newVersion, err := s.buildAppForWrittenChanges(ctx, user, app, appPaths)
	if err != nil {
		buildSpan.Finish(err)
		return nil, err
	}
	buildSpan.Finish(nil)

	metadataSpan := buildtrace.Start(ctx, "runtime.update_release_version_metadata", buildtrace.String("new_version", newVersion))
	s.updateReleaseVersionMetadata(ctx, user, app, appPaths, newVersion, logPrefix)
	metadataSpan.Finish(nil)

	commitSpan := buildtrace.Start(ctx, "runtime.commit_release", buildtrace.String("new_version", newVersion))
	gitCommitHash := s.commitRelease(ctx, user, app, newVersion, logPrefix, requirement, changeDescription)
	commitSpan.Finish(nil)

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
		OutputDir:        appPaths.BuildOutputDir(s.config.GetBuildOutputDir()),
		BinaryNameFormat: s.config.GetBinaryNameFormat(),
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

	if err := s.updateVersionJson(ctx, appPaths.AppDir(), user, app, newVersion); err != nil {
		logger.Warnf(ctx, "[%s] 更新版本信息失败: %v，继续执行", logPrefix, err)
	}

	if err := s.writeBuiltRuntimeManifest(user, app, appPaths, newVersion); err != nil {
		logger.Warnf(ctx, "[%s] 写入 runtime manifest 失败: %v，继续执行", logPrefix, err)
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
