package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

func addFunctionsImpl(s *serviceTreeFunctionService, ctx context.Context, req *dto.AddFunctionsReq) (*dto.AddFunctionsResp, error) {
	if strings.TrimSpace(req.FullCodePath) == "" {
		return &dto.AddFunctionsResp{Success: false, Error: "full_code_path 必填"}, fmt.Errorf("full_code_path 必填")
	}
	targetTree, err := s.loadTargetTree(ctx, strings.TrimSpace(req.FullCodePath))
	if err != nil {
		logger.Errorf(ctx, "[ServiceTreeService] 获取 ServiceTree 失败: FullCodePath=%s, error=%v", req.FullCodePath, err)
		return &dto.AddFunctionsResp{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	sourceCode := req.SourceCode
	if sourceCode == "" {
		logger.Errorf(ctx, "[ServiceTreeService] SourceCode 为空，无法创建函数")
		return &dto.AddFunctionsResp{
			Success: false,
			Error:   "SourceCode 不能为空",
		}, fmt.Errorf("SourceCode 不能为空")
	}

	fileName, err := normalizeAddFunctionsGoFileName(req.FileName, targetTree.Code)
	if err != nil {
		logger.Errorf(ctx, "[ServiceTreeService] 非法 file_name: %v", err)
		return &dto.AddFunctionsResp{
			Success: false,
			Error:   err.Error(),
		}, err
	}
	if strings.TrimSpace(req.FileName) == "" {
		logger.Infof(ctx, "[ServiceTreeService] 同步添加函数：未传 file_name，使用目录 Code - FullCodePath: %s, FileName: %s", req.FullCodePath, fileName)
	} else {
		logger.Infof(ctx, "[ServiceTreeService] 同步添加函数：直接写入 full_code_path - FullCodePath: %s, FileName: %s", req.FullCodePath, fileName)
	}

	packagePath := targetTree.GetPackagePathForFileCreation()
	logger.Infof(ctx, "[ServiceTreeService] 添加函数: DirectoryPath=%s, FileName=%s, SourceCodeLength=%d", packagePath, fileName, len(sourceCode))

	sourceFile := &dto.SourceFileWrite{
		DirectoryPath: packagePath,
		FileName:      fileName,
		SourceCode:    sourceCode,
	}

	updateReq := &dto.UpdateAppReq{
		User:        targetTree.App.User,
		App:         targetTree.App.Code,
		SourceFiles: []*dto.SourceFileWrite{sourceFile},
		WriteOnly:   req.SkipBuild,
	}

	updateResp, err := s.appService.UpdateApp(ctx, updateReq)
	if err != nil {
		logger.Errorf(ctx, "[ServiceTreeService] AppService.UpdateApp 失败: error=%v", err)
		return &dto.AddFunctionsResp{
			Success: false,
			Error:   err.Error(),
		}, err
	}
	if updateResp != nil && len(updateResp.Warnings) > 0 {
		logger.Warnf(ctx, "[ServiceTreeService] AppService.UpdateApp warnings: %s", strings.Join(updateResp.Warnings, " | "))
	}

	addResp := &dto.AddFunctionsResp{
		Success: true,
		AppID:   targetTree.App.ID,
		AppCode: targetTree.App.Code,
	}
	if !req.SkipBuild && updateResp != nil {
		addResp.BuildOldVersion = updateResp.OldVersion
		addResp.BuildNewVersion = updateResp.NewVersion
		if updateResp.Diff != nil {
			for _, api := range updateResp.Diff.Add {
				if api != nil {
					route := api.Router
					if route == "" {
						route = api.Code
					}
					if route != "" {
						addResp.BuildDiffAdd = append(addResp.BuildDiffAdd, route)
					}
				}
			}
			for _, api := range updateResp.Diff.Update {
				if api != nil {
					route := api.Router
					if route == "" {
						route = api.Code
					}
					if route != "" {
						addResp.BuildDiffUpdate = append(addResp.BuildDiffUpdate, route)
					}
				}
			}
			for _, api := range updateResp.Diff.Delete {
				if api != nil {
					route := api.Router
					if route == "" {
						route = api.Code
					}
					if route != "" {
						addResp.BuildDiffDelete = append(addResp.BuildDiffDelete, route)
					}
				}
			}
		}
	}
	return addResp, nil
}

func normalizeAddFunctionsGoFileName(rawFileName, fallbackFileName string) (string, error) {
	fileName := strings.TrimSpace(rawFileName)
	if fileName == "" {
		fileName = strings.TrimSpace(fallbackFileName)
	}
	for strings.HasSuffix(fileName, ".go") {
		fileName = strings.TrimSuffix(fileName, ".go")
	}
	if strings.HasSuffix(fileName, "_test") {
		return "", fmt.Errorf("file_name 不能使用 _test.go 结尾，测试文件不会参与应用 API 注册")
	}
	return fileName, nil
}
