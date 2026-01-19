package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/codegen/metadata"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// FunctionGenService 函数生成服务
type FunctionGenService struct {
	appService        *AppService
	serviceTreeRepo   *repository.ServiceTreeRepository
	serviceTreeService *ServiceTreeService // ⭐ 新增：用于创建目录
	appRepo           *repository.AppRepository
}

// NewFunctionGenService 创建函数生成服务
func NewFunctionGenService(
	appService *AppService,
	serviceTreeRepo *repository.ServiceTreeRepository,
	serviceTreeService *ServiceTreeService, // ⭐ 新增：用于创建目录
	appRepo *repository.AppRepository,
) *FunctionGenService {
	return &FunctionGenService{
		appService:        appService,
		serviceTreeRepo:   serviceTreeRepo,
		serviceTreeService: serviceTreeService,
		appRepo:           appRepo,
	}
}

// ProcessFunctionGenResult 处理函数生成结果（接收 agent-server 处理后的结构化数据）
func (s *FunctionGenService) ProcessFunctionGenResult(ctx context.Context, req *dto.AddFunctionsReq) error {
	// 1. 根据 TreeID 获取父目录 ServiceTree（需要预加载 App）
	parentTree, err := s.serviceTreeRepo.GetByID(req.TreeID)
	if err != nil {
		logger.Errorf(ctx, "[FunctionGenService] 获取父目录 ServiceTree 失败: TreeID=%d, error=%v", req.TreeID, err)
		return err
	}

	// 预加载 App 信息（如果还没有加载）
	if parentTree.App == nil {
		app, err := s.appRepo.GetAppByID(parentTree.AppID)
		if err != nil {
			logger.Errorf(ctx, "[FunctionGenService] 获取 App 失败: AppID=%d, error=%v", parentTree.AppID, err)
			return err
		}
		parentTree.App = app
	}

	// 2. ⭐ 解析代码中的元数据
	sourceCode := req.SourceCode
	if sourceCode == "" {
		// 如果 agent-server 没有处理代码，使用原始代码（向后兼容）
		logger.Warnf(ctx, "[FunctionGenService] agent-server 未处理代码，使用原始代码")
		sourceCode = req.Code
	}

	var meta metadata.Metadata
	var targetTree *model.ServiceTree = parentTree // 默认使用父目录
	var fileName string

	// 尝试解析元数据
	if err := metadata.ParseMetadata(sourceCode, &meta); err == nil {
		// 元数据解析成功
		logger.Infof(ctx, "[FunctionGenService] 元数据解析成功 - DirectoryCode: %s, DirectoryName: %s, File: %s",
			meta.DirectoryCode, meta.DirectoryName, meta.File)

		// 验证必需字段
		if meta.DirectoryCode != "" && meta.File != "" {
			// 3. ⭐ 根据元数据创建或查找目录
			targetTree, err = s.createOrFindDirectory(ctx, parentTree, &meta)
			if err != nil {
				logger.Errorf(ctx, "[FunctionGenService] 创建或查找目录失败: %v", err)
				return err
			}

			// 从元数据中提取文件名（去掉 .go 后缀）
			fileName = strings.TrimSuffix(meta.File, ".go")
		} else {
			logger.Warnf(ctx, "[FunctionGenService] 元数据缺少必需字段，使用父目录 - DirectoryCode: %s, File: %s",
				meta.DirectoryCode, meta.File)
		}
	} else {
		logger.Warnf(ctx, "[FunctionGenService] 元数据解析失败，使用父目录: %v", err)
	}

	// 如果文件名仍为空，使用 ServiceTree.Code 作为 fallback
	if fileName == "" {
		logger.Warnf(ctx, "[FunctionGenService] 未从元数据提取到文件名，使用 ServiceTree.Code 作为 fallback: %s", targetTree.Code)
		fileName = targetTree.Code
	}

	// 4. 从目标目录中提取 package 路径
	packagePath := targetTree.GetPackagePathForFileCreation()

	logger.Infof(ctx, "[FunctionGenService] 处理完成 - TargetTreeID: %d, Package: %s, FileName: %s, SourceCodeLength: %d",
		targetTree.ID, packagePath, fileName, len(sourceCode))

	// 5. 构建 CreateFunctionInfo
	createFunction := &dto.CreateFunctionInfo{
		Package:    packagePath,
		GroupCode:  fileName,
		SourceCode: sourceCode,
	}

	// 6. 调用 AppService.UpdateApp，传入 CreateFunctions
	updateReq := &dto.UpdateAppReq{
		User:            req.User,
		App:             targetTree.App.Code,
		CreateFunctions: []*dto.CreateFunctionInfo{createFunction},
	}

	logger.Infof(ctx, "[FunctionGenService] 调用 AppService.UpdateApp: User=%s, App=%s, Package=%s, FileName=%s",
		updateReq.User, updateReq.App, packagePath, fileName)

	updateResp, err := s.appService.UpdateApp(ctx, updateReq)
	if err != nil {
		logger.Errorf(ctx, "[FunctionGenService] AppService.UpdateApp 失败: error=%v", err)
		return err
	}

	logger.Infof(ctx, "[FunctionGenService] 函数创建成功: Package=%s, FileName=%s", packagePath, fileName)

	// 7. 获取新增的 FullCodePaths
	fullCodePaths := make([]string, 0)
	if updateResp.Diff != nil {
		fullCodePaths = updateResp.Diff.GetAddFullCodePaths()
		logger.Infof(ctx, "[FunctionGenService] 获取新增函数完整代码路径 - Count: %d, FullCodePaths: %v", len(fullCodePaths), fullCodePaths)
	}

	// 8. 发送回调消息给 agent-server
	// ⭐ 只要处理成功就发送回调，确保状态能正确更新为 completed
	callbackData := &dto.FunctionGenCallback{
		RecordID:      req.RecordID,
		MessageID:     req.MessageID,
		Success:       true,
		FullCodePaths: fullCodePaths,
		AppID:         targetTree.App.ID,
		AppCode:       targetTree.App.Code,
		Error:         "",
	}

	// 通过 HTTP 发送回调到 agent-server（直接传 ctx，内部会提取 token、trace_id 等）
	if err := apicall.NotifyWorkspaceUpdateComplete(ctx, callbackData); err != nil {
		logger.Errorf(ctx, "[FunctionGenService] 通知工作空间更新完成失败: error=%v", err)
		// 不中断流程，记录日志即可
	} else {
		if len(fullCodePaths) > 0 {
			logger.Infof(ctx, "[FunctionGenService] 工作空间更新完成通知已发送 (HTTP) - RecordID: %d, MessageID: %d, FullCodePaths: %v, AppCode: %s",
				req.RecordID, req.MessageID, fullCodePaths, targetTree.App.Code)
		} else {
			logger.Infof(ctx, "[FunctionGenService] 工作空间更新完成通知已发送 (HTTP) - RecordID: %d, MessageID: %d, FullCodePaths: [] (无新增函数), AppCode: %s",
				req.RecordID, req.MessageID, targetTree.App.Code)
		}
	}

	return nil
}

// createOrFindDirectory 根据元数据创建或查找目录
func (s *FunctionGenService) createOrFindDirectory(ctx context.Context, parentTree *model.ServiceTree, meta *metadata.Metadata) (*model.ServiceTree, error) {
	// 1. 先尝试查找目录是否已存在（通过 FullCodePath）
	expectedFullCodePath := parentTree.FullCodePath + "/" + meta.DirectoryCode
	existingTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(expectedFullCodePath)
	if err == nil && existingTree != nil {
		logger.Infof(ctx, "[FunctionGenService] 目录已存在 - TreeID: %d, FullCodePath: %s", existingTree.ID, expectedFullCodePath)
		return existingTree, nil
	}

	// 2. 目录不存在，创建新目录
	// 从父目录的 FullCodePath 提取 User 和 App
	fullCodePath := parentTree.FullCodePath
	pathParts := strings.Split(strings.Trim(fullCodePath, "/"), "/")
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("无效的 FullCodePath: %s", fullCodePath)
	}
	targetUser := pathParts[0]
	targetApp := pathParts[1]

	createReq := &dto.CreateServiceTreeReq{
		User:        targetUser,
		App:         targetApp,
		Name:        meta.DirectoryName,
		Code:        meta.DirectoryCode,
		ParentID:    parentTree.ID,
		Description: meta.DirectoryDesc,
		Tags:        strings.Join(meta.Tags, ","),
		Admins:      contextx.GetRequestUser(ctx), // 将当前用户设置为管理员
	}

	createResp, err := s.serviceTreeService.CreateServiceTree(ctx, createReq)
	if err != nil {
		// 如果目录已存在（并发创建的情况），再次尝试查找
		if strings.Contains(err.Error(), "already exists") {
			existingTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(expectedFullCodePath)
			if err == nil && existingTree != nil {
				logger.Infof(ctx, "[FunctionGenService] 目录已存在（并发创建） - TreeID: %d, FullCodePath: %s", existingTree.ID, expectedFullCodePath)
				return existingTree, nil
			}
		}
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	// 3. 获取创建的目录信息
	newTree, err := s.serviceTreeRepo.GetByID(createResp.ID)
	if err != nil {
		return nil, fmt.Errorf("获取新创建的目录信息失败: %w", err)
	}

	logger.Infof(ctx, "[FunctionGenService] 目录创建成功 - TreeID: %d, DirectoryCode: %s, FullCodePath: %s",
		newTree.ID, meta.DirectoryCode, newTree.FullCodePath)

	return newTree, nil
}

// ProcessFunctionGenResultAsync 异步处理函数生成结果（通过回调通知结果）
func (s *FunctionGenService) ProcessFunctionGenResultAsync(ctx context.Context, req *dto.AddFunctionsReq) error {
	// 复用同步处理的逻辑
	return s.ProcessFunctionGenResult(ctx, req)
}

// sendCallback 发送回调通知（内部辅助方法）
func (s *FunctionGenService) sendCallback(ctx context.Context, req *dto.AddFunctionsReq, serviceTree *model.ServiceTree, success bool, fullCodePaths []string, errorMsg string) {
	var appID int64
	var appCode string
	
	// 从 serviceTree 获取 App 信息
	if serviceTree != nil && serviceTree.App != nil {
		appID = serviceTree.App.ID
		appCode = serviceTree.App.Code
	} else if serviceTree != nil {
		// 如果 App 未加载，尝试从数据库获取
		app, err := s.appRepo.GetAppByID(serviceTree.AppID)
		if err == nil && app != nil {
			appID = app.ID
			appCode = app.Code
		}
	}
	
	callbackData := &dto.FunctionGenCallback{
		RecordID:      req.RecordID,
		MessageID:     req.MessageID,
		Success:       success,
		FullCodePaths: fullCodePaths,
		AppID:         appID,
		AppCode:       appCode,
		Error:         errorMsg,
	}

	// 通过 HTTP 发送回调到 agent-server（直接传 ctx，内部会提取 token、trace_id 等）
	if err := apicall.NotifyWorkspaceUpdateComplete(ctx, callbackData); err != nil {
		logger.Errorf(ctx, "[FunctionGenService] 发送回调失败: error=%v", err)
	} else {
		logger.Infof(ctx, "[FunctionGenService] 回调已发送 - RecordID: %d, Success: %v", req.RecordID, success)
	}
}

