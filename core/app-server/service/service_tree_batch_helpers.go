package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/naming"
	"gorm.io/gorm"
)

func executeBatchCreateDirectoryTree(
	ctx context.Context,
	serviceTreeRepo *repository.ServiceTreeRepository,
	runtimeWorkspace *runtimeWorkspaceBridge,
	req *dto.BatchCreateDirectoryTreeReq,
) (*dto.BatchCreateDirectoryTreeResp, error) {
	if req == nil {
		return nil, fmt.Errorf("批量创建目录树请求不能为空")
	}
	if err := validateDirectoryScaffoldItemsForGoPackages(req); err != nil {
		return nil, err
	}

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

		existingTree, err := serviceTreeRepo.GetServiceTreeByFullPath(item.FullCodePath)
		if err == nil && existingTree != nil {
			if existingTree.AppID == app.ID && existingTree.Type == model.ServiceTreeTypePackage {
				pathToTree[item.FullCodePath] = existingTree
				continue
			}
			return nil, fmt.Errorf("目标目录路径已存在但类型或应用不匹配: %s", item.FullCodePath)
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("查询目标目录是否存在失败: path=%s: %w", item.FullCodePath, err)
		}

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

func validateDirectoryScaffoldItemsForGoPackages(req *dto.BatchCreateDirectoryTreeReq) error {
	if len(req.Items) == 0 {
		return fmt.Errorf("目录脚手架项不能为空")
	}

	for index, item := range req.Items {
		if item == nil {
			return fmt.Errorf("目录脚手架项不能为空: items[%d]", index)
		}
		if item.FullCodePath != strings.TrimSpace(item.FullCodePath) {
			return fmt.Errorf("目录 full_code_path 不能包含首尾空格: %s", item.FullCodePath)
		}

		trimmed := strings.Trim(item.FullCodePath, "/")
		if trimmed == "" {
			return fmt.Errorf("目录 full_code_path 不能为空: items[%d]", index)
		}

		parts := strings.Split(trimmed, "/")
		if len(parts) < 3 {
			return fmt.Errorf("目录 full_code_path 必须至少包含 user/app/directory: %s", item.FullCodePath)
		}
		if parts[0] != req.User || parts[1] != req.App {
			return fmt.Errorf("目录 full_code_path 与目标应用不匹配: %s", item.FullCodePath)
		}

		for _, code := range parts[2:] {
			if code != naming.NormalizeGoPackageName(code) {
				return fmt.Errorf("目录 code 不能包含首尾空格: %s", item.FullCodePath)
			}
			if err := naming.ValidateGoPackageName(code, "目录英文标识"); err != nil {
				return fmt.Errorf("目录路径中包含不支持的英文标识 %q: %w", code, err)
			}
		}
	}
	return nil
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
	req *dto.BatchWriteFilesReq,
) (*dto.BatchWriteFilesResp, error) {
	app, runtimeResp, err := runtimeWorkspace.batchWriteFiles(ctx, req)
	if err != nil {
		return nil, err
	}

	operationName := strings.TrimSpace(req.OperationName)
	if operationName == "" {
		operationName = "BatchWriteFiles"
	}

	warnings, err := appService.finalizeReleasedAppMetadata(
		ctx,
		operationName,
		app,
		req.User,
		req.App,
		runtimeResp.NewVersion,
		runtimeResp.Diff,
	)
	if err != nil {
		return nil, err
	}

	return buildBatchWriteFilesResp(runtimeResp, warnings), nil
}

func buildBatchWriteFilesResp(
	runtimeResp *dto.BatchWriteFilesRuntimeResp,
	warnings []string,
) *dto.BatchWriteFilesResp {
	if runtimeResp == nil {
		return &dto.BatchWriteFilesResp{Warnings: warnings}
	}

	return &dto.BatchWriteFilesResp{
		FileCount:     runtimeResp.FileCount,
		WrittenPaths:  runtimeResp.WrittenPaths,
		Diff:          runtimeResp.Diff,
		OldVersion:    runtimeResp.OldVersion,
		NewVersion:    runtimeResp.NewVersion,
		GitCommitHash: runtimeResp.GitCommitHash,
		Warnings:      warnings,
	}
}
