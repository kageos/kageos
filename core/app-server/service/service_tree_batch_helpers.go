package service

import (
	"context"
	"sort"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

func executeBatchCreateDirectoryTree(
	ctx context.Context,
	serviceTreeRepo *repository.ServiceTreeRepository,
	runtimeWorkspace *runtimeWorkspaceBridge,
	req *dto.BatchCreateDirectoryTreeReq,
) (*dto.BatchCreateDirectoryTreeResp, error) {
	app, runtimeResp, err := runtimeWorkspace.batchCreateDirectoryTree(ctx, req)
	if err != nil {
		return nil, err
	}

	sortedItems := make([]*dto.DirectoryScaffoldItem, len(req.Items))
	copy(sortedItems, req.Items)

	sort.Slice(sortedItems, func(i, j int) bool {
		return len(sortedItems[i].FullCodePath) < len(sortedItems[j].FullCodePath)
	})

	pathToTree := make(map[string]*model.ServiceTree)
	currentVersionNum := extractVersionNumForServiceTree(app.Version)

	for _, item := range sortedItems {
		pathParts := strings.Split(strings.Trim(item.FullCodePath, "/"), "/")
		if len(pathParts) < 3 {
			continue
		}
		dirCode := pathParts[len(pathParts)-1]

		parentPath := getParentPathForBatch(item.FullCodePath)
		if parentPath != "" {
			if _, exists := pathToTree[parentPath]; !exists {
				if existingParent, err := serviceTreeRepo.GetServiceTreeByFullPath(parentPath); err == nil {
					pathToTree[parentPath] = existingParent
				}
			}
		}

		newTree := &model.ServiceTree{
			Name:             item.Name,
			Code:             dirCode,
			Type:             model.ServiceTreeTypePackage,
			Description:      item.Description,
			Tags:             item.Tags,
			AppID:            app.ID,
			FullCodePath:     item.FullCodePath,
			AddVersionNum:    currentVersionNum,
			UpdateVersionNum: 0,
		}

		if err := serviceTreeRepo.CreateServiceTreeWithParentPath(newTree, ""); err != nil {
			logger.Warnf(ctx, "[BatchCreateDirectoryTree] 创建 ServiceTree 记录失败: path=%s, error=%v",
				item.FullCodePath, err)
		} else {
			pathToTree[item.FullCodePath] = newTree
		}
	}

	return &dto.BatchCreateDirectoryTreeResp{
		DirectoryCount: runtimeResp.DirectoryCount,
		FileCount:      runtimeResp.FileCount,
		CreatedPaths:   runtimeResp.CreatedPaths,
	}, nil
}

func batchCreateDirectoryTreeImpl(
	s *ServiceTreeService,
	ctx context.Context,
	req *dto.BatchCreateDirectoryTreeReq,
) (*dto.BatchCreateDirectoryTreeResp, error) {
	return executeBatchCreateDirectoryTree(ctx, s.serviceTreeRepo, s.runtimeWorkspace, req)
}

func getParentPathForBatch(fullCodePath string) string {
	pathParts := strings.Split(strings.Trim(fullCodePath, "/"), "/")
	if len(pathParts) <= 2 {
		return ""
	}

	parentParts := pathParts[:len(pathParts)-1]
	return "/" + strings.Join(parentParts, "/")
}

func executeBatchWriteFiles(
	ctx context.Context,
	runtimeWorkspace *runtimeWorkspaceBridge,
	appService *AppService,
	appRepo *repository.AppRepository,
	req *dto.BatchWriteFilesReq,
) (*dto.BatchWriteFilesResp, error) {
	app, runtimeResp, err := runtimeWorkspace.batchWriteFiles(ctx, req)
	if err != nil {
		return nil, err
	}

	if runtimeResp.Diff != nil {
		updateAppReq := &dto.UpdateAppReq{
			User: req.User,
			App:  req.App,
		}
		if err := appService.processAPIDiff(ctx, app.ID, runtimeResp.Diff, updateAppReq, 0, runtimeResp.GitCommitHash); err != nil {
			logger.Warnf(ctx, "[BatchWriteFiles] 处理 API diff 失败: %v", err)
		}
	}

	if runtimeResp.NewVersion != "" {
		if err := appRepo.UpdateAppVersion(req.User, req.App, runtimeResp.NewVersion); err != nil {
			logger.Warnf(ctx, "[BatchWriteFiles] 更新应用版本失败: oldVersion=%s, newVersion=%s, error=%v",
				runtimeResp.OldVersion, runtimeResp.NewVersion, err)
		} else {
			logger.Infof(ctx, "[BatchWriteFiles] 应用版本更新成功: oldVersion=%s, newVersion=%s",
				runtimeResp.OldVersion, runtimeResp.NewVersion)
		}
	}

	return &dto.BatchWriteFilesResp{
		FileCount:     runtimeResp.FileCount,
		WrittenPaths:  runtimeResp.WrittenPaths,
		Diff:          runtimeResp.Diff,
		OldVersion:    runtimeResp.OldVersion,
		NewVersion:    runtimeResp.NewVersion,
		GitCommitHash: runtimeResp.GitCommitHash,
	}, nil
}

func batchWriteFilesImpl(
	s *ServiceTreeService,
	ctx context.Context,
	req *dto.BatchWriteFilesReq,
) (*dto.BatchWriteFilesResp, error) {
	return executeBatchWriteFiles(ctx, s.runtimeWorkspace, s.appService, s.appRepo, req)
}
