package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/gofmt"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// WorkspaceFileService 管理工作区源码文件读写，不承载编译、发布等生命周期语义。
type WorkspaceFileService struct {
	config *config.AppManageServiceConfig
}

// NewWorkspaceFileService 创建工作区文件服务。
func NewWorkspaceFileService(config *config.AppManageServiceConfig) *WorkspaceFileService {
	return &WorkspaceFileService{config: config}
}

func (s *WorkspaceFileService) prepareWritableWorkspace(user, app string) (runtimeAppPaths, error) {
	appPaths := newRuntimeAppPaths(s.config.GetBasePath(), user, app)
	if _, err := os.Stat(appPaths.AppDir()); err != nil {
		if os.IsNotExist(err) {
			return runtimeAppPaths{}, fmt.Errorf("app not found: %s/%s", user, app)
		}
		return runtimeAppPaths{}, fmt.Errorf("检查应用目录失败: %w", err)
	}
	return appPaths, nil
}

// WriteSourceFiles 写入一组源码文件，返回相对应用目录的文件路径列表。
func (s *WorkspaceFileService) WriteSourceFiles(
	ctx context.Context,
	user, app string,
	files []*dto.SourceFileWrite,
) (*dto.WriteSourceFilesResp, error) {
	state, err := s.writeSourceFiles(ctx, user, app, files)
	if err != nil {
		return nil, err
	}

	return &dto.WriteSourceFilesResp{
		Success:      true,
		Message:      fmt.Sprintf("成功写入 %d 个源码文件", len(state.writtenPaths)),
		WrittenFiles: state.writtenPaths,
	}, nil
}

func (s *WorkspaceFileService) writeSourceFiles(
	ctx context.Context,
	user, app string,
	files []*dto.SourceFileWrite,
) (*batchWriteState, error) {
	logger.Infof(ctx, "[WorkspaceFileService] 开始写入源码文件: target=%s/%s, fileCount=%d", user, app, len(files))

	appPaths, err := s.prepareWritableWorkspace(user, app)
	if err != nil {
		return nil, err
	}
	state := newBatchWriteState(len(files))

	for _, file := range files {
		packageDir, targetFilePath, err := resolveSourceFileWriteTarget(appPaths, file)
		if err != nil {
			s.rollbackWriteState(ctx, state)
			return nil, fmt.Errorf("非法源码写入目标 (%s/%s): %w", file.DirectoryPath, file.FileName, err)
		}
		if err := s.ensureDirectoryTreeWithRollback(appPaths.APIDir(), packageDir, state); err != nil {
			s.rollbackWriteState(ctx, state)
			return nil, fmt.Errorf("创建目录失败 (%s): %w", file.DirectoryPath, err)
		}

		if _, exists := state.rollbackEntries[targetFilePath]; !exists {
			entry, err := s.captureFileRollbackEntry(targetFilePath)
			if err != nil {
				s.rollbackWriteState(ctx, state)
				return nil, fmt.Errorf("备份文件失败 (%s): %w", targetFilePath, err)
			}
			state.rollbackEntries[targetFilePath] = entry
			state.rollbackOrder = append(state.rollbackOrder, targetFilePath)
		}

		codeToWrite := file.SourceCode
		fixedCode, err := gofmt.FixGoImport(targetFilePath, []byte(file.SourceCode))
		if err != nil {
			logger.Warnf(ctx, "[WorkspaceFileService] 修复 import 失败，使用原代码: file=%s, error=%v", targetFilePath, err)
		} else {
			codeToWrite = fixedCode
		}

		if err := writeFileAtomic(targetFilePath, []byte(codeToWrite), 0644); err != nil {
			s.rollbackWriteState(ctx, state)
			return nil, fmt.Errorf("写入源码文件失败 (%s): %w", targetFilePath, err)
		}

		relPath, err := filepath.Rel(appPaths.AppDir(), targetFilePath)
		if err != nil {
			relPath = targetFilePath
		}
		state.writtenPaths = append(state.writtenPaths, relPath)
		logger.Infof(ctx, "[WorkspaceFileService] 源码文件写入成功: %s", targetFilePath)
	}

	return state, nil
}

// ReadDirectoryFiles 读取目录下直接包含的 Go 文件。
func (s *WorkspaceFileService) ReadDirectoryFiles(ctx context.Context, user, app, fullCodePath string) ([]dto.DirectoryFileInfo, error) {
	logger.Infof(ctx, "[WorkspaceFileService] 开始读取目录文件: user=%s, app=%s, path=%s", user, app, fullCodePath)

	appPaths := newRuntimeAppPaths(s.config.GetBasePath(), user, app)
	directoryPath, err := resolveWorkspaceDirectoryPath(appPaths, fullCodePath)
	if err != nil {
		return nil, err
	}

	var files []dto.DirectoryFileInfo
	entries, err := os.ReadDir(directoryPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warnf(ctx, "[WorkspaceFileService] 目录不存在: path=%s", directoryPath)
			return []dto.DirectoryFileInfo{}, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}

		path := filepath.Join(directoryPath, name)
		content, err := os.ReadFile(path)
		if err != nil {
			logger.Warnf(ctx, "[WorkspaceFileService] 读取文件失败: path=%s, error=%v", path, err)
			continue
		}

		fileName := strings.TrimSuffix(name, ".go")
		files = append(files, dto.DirectoryFileInfo{
			FileName:     fileName,
			RelativePath: name,
			Content:      string(content),
		})
	}

	logger.Infof(ctx, "[WorkspaceFileService] 读取目录文件完成: path=%s, fileCount=%d", directoryPath, len(files))
	return files, nil
}

// ReplaceInFileBatch 在指定文件中做多组 search-replace，全部校验通过后再原子写盘。
func (s *WorkspaceFileService) ReplaceInFileBatch(
	ctx context.Context,
	user, app, directoryPath, fileName string,
	replacements []dto.ReplaceItemRuntime,
	allOrNothing, returnFullContent bool,
) (totalCount int, newContent string, details []dto.ReplaceItemResultRuntime, err error) {
	appPaths := newRuntimeAppPaths(s.config.GetBasePath(), user, app)
	filePath, err := resolveWorkspaceFilePath(appPaths, directoryPath, fileName)
	if err != nil {
		return 0, "", nil, err
	}
	if filepath.Base(filePath) == "init_.go" {
		return 0, "", nil, fmt.Errorf("不允许修改 init_.go，由脚手架生成")
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, "", nil, fmt.Errorf("文件不存在: %s", filePath)
		}
		return 0, "", nil, err
	}
	current := string(content)

	for i, item := range replacements {
		if item.SearchString == "" {
			return 0, "", nil, fmt.Errorf("第 %d 项 search_string 不能为空", i+1)
		}
		expected := item.ExpectedCount
		if expected <= 0 {
			expected = 1
		}
		actual := strings.Count(current, item.SearchString)
		if allOrNothing && actual != expected {
			details = append(details, dto.ReplaceItemResultRuntime{Index: i, ExpectedCount: expected, ActualCount: actual})
		}
	}

	if allOrNothing && len(details) > 0 {
		return 0, "", details, fmt.Errorf("有 %d 项实际匹配次数与预期不符，未落盘", len(details))
	}

	for _, item := range replacements {
		n := strings.Count(current, item.SearchString)
		totalCount += n
		current = strings.ReplaceAll(current, item.SearchString, item.ReplaceString)
	}

	if current == string(content) {
		logger.Infof(ctx, "[WorkspaceFileService] 未发生替换: %s", filePath)
		return 0, current, nil, nil
	}
	if err := writeFileAtomic(filePath, []byte(current), 0644); err != nil {
		return 0, "", nil, err
	}

	logger.Infof(ctx, "[WorkspaceFileService] 替换完成: path=%s, totalCount=%d", filePath, totalCount)
	if !returnFullContent {
		current = ""
	}
	return totalCount, current, nil, nil
}

// DeleteFile 删除指定源码文件。
func (s *WorkspaceFileService) DeleteFile(ctx context.Context, user, app, directoryPath, fileName string) error {
	logger.Infof(ctx, "[WorkspaceFileService] 删除文件: user=%s, app=%s, path=%s, file=%s", user, app, directoryPath, fileName)

	appPaths := newRuntimeAppPaths(s.config.GetBasePath(), user, app)
	filePath, err := resolveWorkspaceFilePath(appPaths, directoryPath, fileName)
	if err != nil {
		return err
	}
	if filepath.Base(filePath) == "init_.go" {
		return fmt.Errorf("不允许删除 init_.go，由脚手架生成")
	}

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			logger.Warnf(ctx, "[WorkspaceFileService] 文件已不存在: %s", filePath)
			return nil
		}
		return err
	}

	logger.Infof(ctx, "[WorkspaceFileService] 已删除: %s", filePath)
	return nil
}
