package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type fileRollbackEntry struct {
	Path    string
	Existed bool
	Content []byte
	Mode    os.FileMode
}

type batchWriteState struct {
	writtenPaths    []string
	rollbackEntries map[string]*fileRollbackEntry
	rollbackOrder   []string
	createdDirs     []string
	createdDirSet   map[string]struct{}
}

func newBatchWriteState(fileCount int) *batchWriteState {
	return &batchWriteState{
		writtenPaths:    make([]string, 0, fileCount),
		rollbackEntries: make(map[string]*fileRollbackEntry, fileCount),
		rollbackOrder:   make([]string, 0, fileCount),
		createdDirs:     make([]string, 0, fileCount),
		createdDirSet:   make(map[string]struct{}, fileCount),
	}
}

func (s *WorkspaceFileService) writeDirectoryTreeFiles(
	ctx context.Context,
	user, app string,
	files []*dto.FileWriteItem,
) (*batchWriteState, error) {
	appPaths, err := s.prepareWritableWorkspace(user, app)
	if err != nil {
		return nil, err
	}

	apiDir := appPaths.APIDir()
	state := newBatchWriteState(len(files))

	for _, item := range files {
		packageDir, filePath, _, err := resolveBatchWriteTarget(user, app, apiDir, item)
		if err != nil {
			s.rollbackWriteState(ctx, state)
			return nil, fmt.Errorf("非法文件写入目标 (%s): %w", item.FullCodePath, err)
		}

		if err := s.ensureDirectoryTreeWithRollback(apiDir, packageDir, state); err != nil {
			s.rollbackWriteState(ctx, state)
			return nil, fmt.Errorf("创建目录失败: %w", err)
		}

		if _, exists := state.rollbackEntries[filePath]; !exists {
			entry, err := s.captureFileRollbackEntry(filePath)
			if err != nil {
				s.rollbackWriteState(ctx, state)
				return nil, fmt.Errorf("备份文件失败 (%s): %w", item.FullCodePath, err)
			}
			state.rollbackEntries[filePath] = entry
			state.rollbackOrder = append(state.rollbackOrder, filePath)
		}

		if err := writeFileAtomic(filePath, []byte(item.Content), 0644); err != nil {
			s.rollbackWriteState(ctx, state)
			return nil, fmt.Errorf("写入文件失败 (%s): %w", item.FullCodePath, err)
		}

		state.writtenPaths = append(state.writtenPaths, item.FullCodePath)
		logger.Infof(ctx, "[WorkspaceFileService] 文件写入成功: %s", filePath)
	}

	logger.Infof(ctx, "[WorkspaceFileService] 批量写文件完成: fileCount=%d", len(state.writtenPaths))
	return state, nil
}

func (s *WorkspaceFileService) rollbackWriteState(ctx context.Context, state *batchWriteState) {
	if state == nil {
		return
	}
	s.rollbackFiles(ctx, state.rollbackEntries, state.rollbackOrder)
	s.rollbackCreatedDirectories(ctx, state.createdDirs)
}

func (s *WorkspaceFileService) ensureDirectoryTreeWithRollback(baseDir, targetDir string, state *batchWriteState) error {
	if err := ensurePathWithinBase(baseDir, targetDir); err != nil {
		return err
	}

	dirsToCreate := make([]string, 0, 4)
	currentDir := filepath.Clean(targetDir)
	baseDir = filepath.Clean(baseDir)

	for {
		info, err := os.Stat(currentDir)
		if err == nil {
			if !info.IsDir() {
				return fmt.Errorf("路径不是目录: %s", currentDir)
			}
			break
		}
		if !os.IsNotExist(err) {
			return err
		}

		dirsToCreate = append(dirsToCreate, currentDir)
		if currentDir == baseDir {
			break
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	for _, dir := range dirsToCreate {
		if _, exists := state.createdDirSet[dir]; exists {
			continue
		}
		state.createdDirSet[dir] = struct{}{}
		state.createdDirs = append(state.createdDirs, dir)
	}

	return nil
}

func (s *WorkspaceFileService) captureFileRollbackEntry(filePath string) (*fileRollbackEntry, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &fileRollbackEntry{
				Path:    filePath,
				Existed: false,
			}, nil
		}
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("路径是目录，无法按文件回滚: %s", filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return &fileRollbackEntry{
		Path:    filePath,
		Existed: true,
		Content: content,
		Mode:    info.Mode(),
	}, nil
}

func (s *WorkspaceFileService) rollbackFiles(ctx context.Context, entries map[string]*fileRollbackEntry, order []string) {
	logger.Warnf(ctx, "[WorkspaceFileService] 开始回滚已写入的文件: fileCount=%d", len(order))

	restoredCount := 0
	deletedCount := 0
	for i := len(order) - 1; i >= 0; i-- {
		filePath := order[i]
		entry := entries[filePath]
		if entry == nil {
			continue
		}

		if !entry.Existed {
			if err := os.Remove(filePath); err != nil {
				if os.IsNotExist(err) {
					continue
				}
				logger.Errorf(ctx, "[WorkspaceFileService] 删除新文件失败: file=%s, error=%v", filePath, err)
				continue
			}
			deletedCount++
			logger.Infof(ctx, "[WorkspaceFileService] 已删除新文件: %s", filePath)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			logger.Errorf(ctx, "[WorkspaceFileService] 创建回滚目录失败: file=%s, error=%v", filePath, err)
			continue
		}

		if err := writeFileAtomic(filePath, entry.Content, entry.Mode.Perm()); err != nil {
			logger.Errorf(ctx, "[WorkspaceFileService] 恢复文件内容失败: file=%s, error=%v", filePath, err)
			continue
		}
		if err := os.Chmod(filePath, entry.Mode); err != nil {
			logger.Warnf(ctx, "[WorkspaceFileService] 恢复文件权限失败: file=%s, error=%v", filePath, err)
		}

		restoredCount++
		logger.Infof(ctx, "[WorkspaceFileService] 已恢复原文件: %s", filePath)
	}

	logger.Infof(ctx, "[WorkspaceFileService] 文件回滚完成: restoredCount=%d, deletedCount=%d, totalCount=%d",
		restoredCount, deletedCount, len(order))
}

func (s *WorkspaceFileService) rollbackCreatedDirectories(ctx context.Context, dirs []string) {
	if len(dirs) == 0 {
		return
	}

	logger.Warnf(ctx, "[WorkspaceFileService] 开始回滚新建目录: dirCount=%d", len(dirs))

	removedCount := 0
	for _, dir := range dirs {
		if err := os.Remove(dir); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			logger.Warnf(ctx, "[WorkspaceFileService] 目录未移除（可能仍非空）: dir=%s, error=%v", dir, err)
			continue
		}
		removedCount++
		logger.Infof(ctx, "[WorkspaceFileService] 已移除新建目录: %s", dir)
	}

	logger.Infof(ctx, "[WorkspaceFileService] 目录回滚完成: removedCount=%d, totalCount=%d", removedCount, len(dirs))
}
