package service

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

func getDirectorySnapshotsRecursivelyImpl(
	s *ServiceTreeService,
	ctx context.Context,
	appID int64,
	rootDirectoryPath string,
) (map[string][]*model.FileSnapshot, error) {
	rootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(rootDirectoryPath)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warnf(ctx, "[ServiceTreeService] 根目录节点不存在: path=%s", rootDirectoryPath)
			return make(map[string][]*model.FileSnapshot), nil
		}
		return nil, fmt.Errorf("获取根目录节点失败: %w", err)
	}

	descendants, err := s.serviceTreeRepo.GetDescendantDirectories(appID, rootDirectoryPath)
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

	allSnapshots, err := s.fileSnapshotRepo.GetCurrentSnapshotsByServiceTreeIDs(treeIDs)
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

func getDirectoryFilesFromRuntimeRecursivelyImpl(
	s *ServiceTreeService,
	ctx context.Context,
	appID int64,
	rootDirectoryPath string,
) (map[string][]*model.FileSnapshot, error) {
	app, err := s.appRepo.GetAppByID(appID)
	if err != nil {
		return nil, fmt.Errorf("获取应用失败: %w", err)
	}
	if app.HostID <= 0 {
		return nil, fmt.Errorf("应用未关联 runtime，无法读取目录文件")
	}

	rootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(rootDirectoryPath)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warnf(ctx, "[ServiceTreeService] 根目录节点不存在: path=%s", rootDirectoryPath)
			return make(map[string][]*model.FileSnapshot), nil
		}
		return nil, fmt.Errorf("获取根目录节点失败: %w", err)
	}

	descendants, err := s.serviceTreeRepo.GetDescendantDirectories(appID, rootDirectoryPath)
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
		runtimeReq := &dto.ReadDirectoryFilesRuntimeReq{
			User:          app.User,
			App:           app.Code,
			DirectoryPath: tree.FullCodePath,
		}
		runtimeResp, err := s.appCall.ReadDirectoryFiles(ctx, app.HostID, runtimeReq)
		if err != nil {
			return nil, fmt.Errorf("从 runtime 读取目录文件失败 path=%s: %w", tree.FullCodePath, err)
		}
		if runtimeResp == nil || !runtimeResp.Success {
			logger.Warnf(ctx, "[ServiceTreeService] runtime 返回失败或空: path=%s", tree.FullCodePath)
			continue
		}
		for _, f := range runtimeResp.Files {
			fileType := "go"
			if f.RelativePath != "" && strings.HasSuffix(f.RelativePath, ".go") {
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
