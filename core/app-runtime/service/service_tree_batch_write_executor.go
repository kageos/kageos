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
}

func newBatchWriteState(fileCount int) *batchWriteState {
	return &batchWriteState{
		writtenPaths:    make([]string, 0, fileCount),
		rollbackEntries: make(map[string]*fileRollbackEntry, fileCount),
		rollbackOrder:   make([]string, 0, fileCount),
	}
}

func (s *ServiceTreeService) writeBatchFilesToDisk(
	ctx context.Context,
	user, app, apiDir string,
	files []*dto.DirectoryTreeItem,
) (*batchWriteState, error) {
	state := newBatchWriteState(len(files))

	for _, item := range files {
		packageDir, filePath, _, err := resolveBatchWriteTarget(user, app, apiDir, item)
		if err != nil {
			s.rollbackBatchWriteState(ctx, state)
			return nil, fmt.Errorf("非法文件写入目标 (%s): %w", item.FullCodePath, err)
		}

		if err := os.MkdirAll(packageDir, 0755); err != nil {
			s.rollbackBatchWriteState(ctx, state)
			return nil, fmt.Errorf("创建目录失败: %w", err)
		}

		if _, exists := state.rollbackEntries[filePath]; !exists {
			entry, err := s.captureFileRollbackEntry(filePath)
			if err != nil {
				s.rollbackBatchWriteState(ctx, state)
				return nil, fmt.Errorf("备份文件失败 (%s): %w", item.FullCodePath, err)
			}
			state.rollbackEntries[filePath] = entry
			state.rollbackOrder = append(state.rollbackOrder, filePath)
		}

		if err := writeFileAtomic(filePath, []byte(item.Content), 0644); err != nil {
			s.rollbackBatchWriteState(ctx, state)
			return nil, fmt.Errorf("写入文件失败 (%s): %w", item.FullCodePath, err)
		}

		state.writtenPaths = append(state.writtenPaths, item.FullCodePath)
		logger.Infof(ctx, "[ServiceTreeService] 文件写入成功: %s", filePath)
	}

	logger.Infof(ctx, "[ServiceTreeService] 批量写文件完成: fileCount=%d", len(state.writtenPaths))
	return state, nil
}

func (s *ServiceTreeService) rollbackBatchWriteState(ctx context.Context, state *batchWriteState) {
	if state == nil {
		return
	}
	s.rollbackFiles(ctx, state.rollbackEntries, state.rollbackOrder)
}

func (s *ServiceTreeService) captureFileRollbackEntry(filePath string) (*fileRollbackEntry, error) {
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

func (s *ServiceTreeService) rollbackFiles(ctx context.Context, entries map[string]*fileRollbackEntry, order []string) {
	logger.Warnf(ctx, "[ServiceTreeService] 开始回滚已写入的文件: fileCount=%d", len(order))

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
				logger.Errorf(ctx, "[ServiceTreeService] 删除新文件失败: file=%s, error=%v", filePath, err)
				continue
			}
			deletedCount++
			logger.Infof(ctx, "[ServiceTreeService] 已删除新文件: %s", filePath)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			logger.Errorf(ctx, "[ServiceTreeService] 创建回滚目录失败: file=%s, error=%v", filePath, err)
			continue
		}

		if err := writeFileAtomic(filePath, entry.Content, entry.Mode.Perm()); err != nil {
			logger.Errorf(ctx, "[ServiceTreeService] 恢复文件内容失败: file=%s, error=%v", filePath, err)
			continue
		}
		if err := os.Chmod(filePath, entry.Mode); err != nil {
			logger.Warnf(ctx, "[ServiceTreeService] 恢复文件权限失败: file=%s, error=%v", filePath, err)
		}

		restoredCount++
		logger.Infof(ctx, "[ServiceTreeService] 已恢复原文件: %s", filePath)
	}

	logger.Infof(ctx, "[ServiceTreeService] 文件回滚完成: restoredCount=%d, deletedCount=%d, totalCount=%d",
		restoredCount, deletedCount, len(order))
}
