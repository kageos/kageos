package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
)

// ReplaceDirectoryTree 完全替换同名目录：旧目录先移出编译路径，编译失败则恢复。
func (s *WorkspaceChangeService) ReplaceDirectoryTree(
	ctx context.Context,
	req *dto.ReplaceDirectoryTreeRuntimeReq,
) (*dto.ReplaceDirectoryTreeRuntimeResp, error) {
	if req == nil {
		return nil, fmt.Errorf("替换目录请求不能为空")
	}
	operationName := strings.TrimSpace(req.OperationName)
	if operationName == "" {
		operationName = "ReplaceDirectoryTree"
	}
	operationLabel := strings.TrimSpace(req.OperationLabel)
	if operationLabel == "" {
		operationLabel = "替换目录"
	}
	if strings.TrimSpace(req.TargetRootFullCodePath) == "" {
		return nil, fmt.Errorf("target_root_full_code_path 不能为空")
	}
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("替换目录必须包含目录脚手架项")
	}
	if s.workspaceFileService == nil {
		return nil, fmt.Errorf("workspaceFileService 未设置，无法替换目录")
	}
	if s.appManageService == nil {
		return nil, fmt.Errorf("appManageService 未设置，无法编译应用")
	}
	if s.packageScaffold == nil {
		return nil, fmt.Errorf("packageScaffold 未设置，无法维护目录脚手架")
	}

	logger.Infof(ctx, "[WorkspaceChangeService] 开始%s: user=%s, app=%s, target=%s, dirCount=%d, fileCount=%d",
		operationLabel, req.User, req.App, req.TargetRootFullCodePath, len(req.Items), len(req.Files))

	replaceState, packagePath, err := s.workspaceFileService.beginDirectoryReplace(ctx, req.User, req.App, req.TargetRootFullCodePath)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			s.workspaceFileService.rollbackDirectoryReplace(ctx, replaceState)
		}
	}()

	if err := s.packageScaffold.removeMainFileImport(ctx, req.User, req.App, packagePath); err != nil {
		return nil, fmt.Errorf("清理旧目录 import 失败: %w", err)
	}

	scaffoldResp, err := s.packageScaffold.BatchCreateDirectoryTree(ctx, &dto.BatchCreateDirectoryTreeRuntimeReq{
		User:  req.User,
		App:   req.App,
		Items: req.Items,
	})
	if err != nil {
		return nil, fmt.Errorf("重建目录脚手架失败: %w", err)
	}

	var writeState *batchWriteState
	if len(req.Files) > 0 {
		writeState, err = s.workspaceFileService.writeDirectoryTreeFiles(ctx, req.User, req.App, req.Files)
		if err != nil {
			return nil, err
		}
	}

	appPaths := newRuntimeAppPaths(s.config.GetBasePath(), req.User, req.App)
	release, err := s.appManageService.finalizeWrittenAppChanges(ctx, req.User, req.App, appPaths, req.ForceDiff, operationName)
	if err != nil {
		return nil, err
	}

	committed = true
	s.workspaceFileService.commitDirectoryReplace(ctx, replaceState)

	fileCount := 0
	writtenPaths := []string{}
	if writeState != nil {
		fileCount = len(writeState.writtenPaths)
		writtenPaths = writeState.writtenPaths
	}
	directoryCount := len(req.Items)
	if scaffoldResp != nil {
		directoryCount = scaffoldResp.DirectoryCount
	}

	logger.Infof(ctx, "[WorkspaceChangeService] %s并编译完成: target=%s oldVersion=%s newVersion=%s",
		operationLabel, req.TargetRootFullCodePath, release.oldVersion, release.newVersion)

	return &dto.ReplaceDirectoryTreeRuntimeResp{
		DirectoryCount:      directoryCount,
		FileCount:           fileCount,
		TargetDirectoryPath: req.TargetRootFullCodePath,
		WrittenPaths:        writtenPaths,
		Diff:                release.diff,
		OldVersion:          release.oldVersion,
		NewVersion:          release.newVersion,
		GitCommitHash:       release.gitCommitHash,
	}, nil
}
