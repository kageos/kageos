package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kageos/kageos/pkg/logger"
)

type directoryReplaceState struct {
	targetDir     string
	stashDir      string
	stashRoot     string
	targetExisted bool
	mainEntry     *fileRollbackEntry
}

func (s *WorkspaceFileService) beginDirectoryReplace(
	ctx context.Context,
	user, app, targetRootFullCodePath string,
) (*directoryReplaceState, string, error) {
	appPaths, err := s.prepareWritableWorkspace(user, app)
	if err != nil {
		return nil, "", err
	}

	packagePath, err := validateBatchWritePackagePath(user, app, targetRootFullCodePath)
	if err != nil {
		return nil, "", err
	}
	apiDir := appPaths.APIDir()
	targetDir := filepath.Join(apiDir, packagePath)
	if err := ensurePathWithinBase(apiDir, targetDir); err != nil {
		return nil, "", err
	}

	mainEntry, err := s.captureFileRollbackEntry(appPaths.MainGoPath())
	if err != nil {
		return nil, "", fmt.Errorf("备份 main.go 失败: %w", err)
	}

	state := &directoryReplaceState{
		targetDir: targetDir,
		mainEntry: mainEntry,
	}

	info, statErr := os.Stat(targetDir)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return state, packagePath, nil
		}
		return nil, "", fmt.Errorf("检查目标目录失败: %w", statErr)
	}
	if !info.IsDir() {
		return nil, "", fmt.Errorf("目标路径不是目录: %s", targetDir)
	}

	stashRoot := filepath.Join(appPaths.WorkplaceSubDir("replace-stash"), fmt.Sprintf("%d", time.Now().UnixNano()))
	stashDir := filepath.Join(stashRoot, packagePath)
	if err := os.MkdirAll(filepath.Dir(stashDir), 0755); err != nil {
		return nil, "", fmt.Errorf("创建替换暂存目录失败: %w", err)
	}
	if err := os.Rename(targetDir, stashDir); err != nil {
		_ = os.RemoveAll(stashRoot)
		return nil, "", fmt.Errorf("暂存旧目录失败: %w", err)
	}

	state.stashRoot = stashRoot
	state.stashDir = stashDir
	state.targetExisted = true
	logger.Infof(ctx, "[WorkspaceFileService] 已暂存待替换目录: target=%s stash=%s", targetDir, stashDir)
	return state, packagePath, nil
}

func (s *WorkspaceFileService) rollbackDirectoryReplace(ctx context.Context, state *directoryReplaceState) {
	if state == nil {
		return
	}

	if state.targetDir != "" {
		if err := os.RemoveAll(state.targetDir); err != nil {
			logger.Warnf(ctx, "[WorkspaceFileService] 回滚替换时删除新目录失败: path=%s, error=%v", state.targetDir, err)
		}
	}

	if state.targetExisted && state.stashDir != "" {
		if err := os.MkdirAll(filepath.Dir(state.targetDir), 0755); err != nil {
			logger.Warnf(ctx, "[WorkspaceFileService] 回滚替换时创建父目录失败: path=%s, error=%v", state.targetDir, err)
		} else if err := os.Rename(state.stashDir, state.targetDir); err != nil {
			logger.Warnf(ctx, "[WorkspaceFileService] 回滚替换时恢复旧目录失败: stash=%s target=%s error=%v", state.stashDir, state.targetDir, err)
		} else {
			logger.Infof(ctx, "[WorkspaceFileService] 已恢复旧目录: %s", state.targetDir)
		}
	}

	if state.mainEntry != nil {
		s.rollbackFiles(ctx, map[string]*fileRollbackEntry{state.mainEntry.Path: state.mainEntry}, []string{state.mainEntry.Path})
	}

	if state.stashRoot != "" {
		_ = os.RemoveAll(state.stashRoot)
	}
}

func (s *WorkspaceFileService) commitDirectoryReplace(ctx context.Context, state *directoryReplaceState) {
	if state == nil || state.stashRoot == "" {
		return
	}
	if err := os.RemoveAll(state.stashRoot); err != nil {
		logger.Warnf(ctx, "[WorkspaceFileService] 清理替换暂存目录失败: path=%s, error=%v", state.stashRoot, err)
		return
	}
	logger.Infof(ctx, "[WorkspaceFileService] 已清理替换暂存目录: %s", state.stashRoot)
}
