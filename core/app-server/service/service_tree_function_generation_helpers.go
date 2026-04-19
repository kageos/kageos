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

	fileName := strings.TrimSpace(req.FileName)
	if fileName != "" {
		fileName = strings.TrimSuffix(fileName, ".go")
	}
	if fileName == "" {
		fileName = targetTree.Code
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

func processFunctionGenResultImpl(s *serviceTreeFunctionService, ctx context.Context, req *dto.AddFunctionsReq) error {
	if strings.TrimSpace(req.FullCodePath) == "" {
		return fmt.Errorf("full_code_path 必填")
	}
	targetTree, err := s.loadTargetTree(ctx, strings.TrimSpace(req.FullCodePath))
	if err != nil {
		logger.Errorf(ctx, "[ServiceTreeService] 获取 ServiceTree 失败: FullCodePath=%s, error=%v", req.FullCodePath, err)
		return err
	}

	sourceCode := req.SourceCode
	if sourceCode == "" {
		logger.Errorf(ctx, "[ServiceTreeService] SourceCode 为空，无法处理函数生成结果")
		return fmt.Errorf("SourceCode 不能为空")
	}

	fileName := strings.TrimSpace(req.FileName)
	if fileName != "" {
		fileName = strings.TrimSuffix(fileName, ".go")
	}
	if fileName == "" {
		fileName = targetTree.Code
		logger.Infof(ctx, "[ServiceTreeService] 异步处理：未传 file_name，使用目录 Code - FullCodePath: %s, FileName: %s", req.FullCodePath, fileName)
	} else {
		logger.Infof(ctx, "[ServiceTreeService] 异步处理：直接写入 full_code_path - FullCodePath: %s, FileName: %s", req.FullCodePath, fileName)
	}

	packagePath := targetTree.GetPackagePathForFileCreation()

	logger.Infof(ctx, "[ServiceTreeService] 处理完成 - TargetTreeID: %d, DirectoryPath: %s, FileName: %s, SourceCodeLength: %d",
		targetTree.ID, packagePath, fileName, len(sourceCode))

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

	logger.Infof(ctx, "[ServiceTreeService] 调用 AppService.UpdateApp: User=%s, App=%s, DirectoryPath=%s, FileName=%s",
		updateReq.User, updateReq.App, packagePath, fileName)

	updateResp, err := s.appService.UpdateApp(ctx, updateReq)
	if err != nil {
		logger.Errorf(ctx, "[ServiceTreeService] AppService.UpdateApp 失败: error=%v", err)
		return err
	}
	if updateResp != nil && len(updateResp.Warnings) > 0 {
		logger.Warnf(ctx, "[ServiceTreeService] AppService.UpdateApp warnings: %s", strings.Join(updateResp.Warnings, " | "))
	}

	logger.Infof(ctx, "[ServiceTreeService] 函数创建成功: DirectoryPath=%s, FileName=%s", packagePath, fileName)

	fullCodePaths := make([]string, 0)
	if updateResp.Diff != nil {
		fullCodePaths = updateResp.Diff.GetAddFullCodePaths()
		logger.Infof(ctx, "[ServiceTreeService] 获取新增函数完整代码路径 - Count: %d, FullCodePaths: %v", len(fullCodePaths), fullCodePaths)
	}

	return nil
}
