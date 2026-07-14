package service

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/pkg/logger"
)

func readDirectorySnapshotsRecursively(
	ctx context.Context,
	serviceTreeRepo *repository.ServiceTreeRepository,
	fileSnapshotRepo *repository.FileSnapshotRepository,
	appID int64,
	rootDirectoryPath string,
) (map[string][]*model.FileSnapshot, error) {
	rootTree, err := serviceTreeRepo.GetServiceTreeByFullPath(ctx, rootDirectoryPath)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warnf(ctx, "[ServiceTreeService] 根目录节点不存在: path=%s", rootDirectoryPath)
			return make(map[string][]*model.FileSnapshot), nil
		}
		return nil, fmt.Errorf("获取根目录节点失败: %w", err)
	}

	descendants, err := serviceTreeRepo.GetDescendantDirectories(ctx, appID, rootDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("查询子目录失败: %w", err)
	}

	allTrees := make([]*model.ServiceTree, 0, len(descendants)+1)
	allTrees = append(allTrees, rootTree)
	allTrees = append(allTrees, descendants...)

	treeIDs := make([]int64, 0, len(allTrees))
	treeIDToPath := make(map[int64]string)
	for _, tree := range allTrees {
		treeIDs = append(treeIDs, tree.ID)
		treeIDToPath[tree.ID] = tree.FullCodePath
	}

	allSnapshots, err := fileSnapshotRepo.GetCurrentSnapshotsByServiceTreeIDs(ctx, treeIDs)
	if err != nil {
		return nil, fmt.Errorf("批量查询文件快照失败: %w", err)
	}

	result := make(map[string][]*model.FileSnapshot)
	for _, tree := range allTrees {
		result[tree.FullCodePath] = make([]*model.FileSnapshot, 0)
	}

	for _, snapshot := range allSnapshots {
		path := treeIDToPath[snapshot.ServiceTreeID]
		if path != "" {
			result[path] = append(result[path], snapshot)
		}
	}

	totalFiles := 0
	for _, files := range result {
		totalFiles += len(files)
	}

	logger.Infof(ctx, "[ServiceTreeService] 递归获取快照完成: 根目录=%s, 目录数=%d, 总文件数=%d",
		rootDirectoryPath, len(allTrees), totalFiles)

	return result, nil
}

func readDirectoryFilesFromRuntimeRecursively(
	ctx context.Context,
	serviceTreeRepo *repository.ServiceTreeRepository,
	runtimeWorkspace *runtimeWorkspaceBridge,
	appID int64,
	rootDirectoryPath string,
) (map[string][]*model.FileSnapshot, error) {
	app, err := runtimeWorkspace.getRuntimeBoundAppByID(ctx, appID, "读取目录文件")
	if err != nil {
		return nil, err
	}

	rootTree, err := serviceTreeRepo.GetServiceTreeByFullPath(ctx, rootDirectoryPath)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warnf(ctx, "[ServiceTreeService] 根目录节点不存在: path=%s", rootDirectoryPath)
			return make(map[string][]*model.FileSnapshot), nil
		}
		return nil, fmt.Errorf("获取根目录节点失败: %w", err)
	}

	descendants, err := serviceTreeRepo.GetDescendantDirectories(ctx, appID, rootDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("查询子目录失败: %w", err)
	}

	allTrees := make([]*model.ServiceTree, 0, len(descendants)+1)
	allTrees = append(allTrees, rootTree)
	allTrees = append(allTrees, descendants...)

	result := make(map[string][]*model.FileSnapshot)
	for _, tree := range allTrees {
		result[tree.FullCodePath] = make([]*model.FileSnapshot, 0)
	}

	for _, tree := range allTrees {
		runtimeResp, err := runtimeWorkspace.readDirectoryFilesFromApp(ctx, app, tree.FullCodePath)
		if err != nil {
			return nil, fmt.Errorf("从 runtime 读取目录文件失败 path=%s: %w", tree.FullCodePath, err)
		}
		if runtimeResp == nil || !runtimeResp.Success {
			logger.Warnf(ctx, "[ServiceTreeService] runtime 返回失败或空: path=%s", tree.FullCodePath)
			continue
		}
		for _, f := range runtimeResp.Files {
			if isInternalWorkspaceManifestFile(f.RelativePath, f.FileName) {
				continue
			}
			fileType := strings.TrimSpace(f.FileType)
			if fileType == "" {
				fileType = "go"
			}
			result[tree.FullCodePath] = append(result[tree.FullCodePath], &model.FileSnapshot{
				FileName:     f.FileName,
				RelativePath: f.RelativePath,
				Content:      f.Content,
				FileType:     fileType,
				FileVersion:  "",
			})
		}
	}

	totalFiles := 0
	for _, files := range result {
		totalFiles += len(files)
	}
	logger.Infof(ctx, "[ServiceTreeService] 从 runtime 递归读取目录文件完成: 根目录=%s, 目录数=%d, 总文件数=%d",
		rootDirectoryPath, len(allTrees), totalFiles)
	return result, nil
}
