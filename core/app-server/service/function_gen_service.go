package service

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// FunctionGenService 函数生成服务
type FunctionGenService struct {
	appService      *AppService
	serviceTreeRepo *repository.ServiceTreeRepository
	appRepo         *repository.AppRepository
}

// NewFunctionGenService 创建函数生成服务
func NewFunctionGenService(
	appService *AppService,
	serviceTreeRepo *repository.ServiceTreeRepository,
	appRepo *repository.AppRepository,
) *FunctionGenService {
	return &FunctionGenService{
		appService:      appService,
		serviceTreeRepo: serviceTreeRepo,
		appRepo:         appRepo,
	}
}

// ProcessFunctionGenResult 处理函数生成结果（接收 agent-server 处理后的结构化数据）
func (s *FunctionGenService) ProcessFunctionGenResult(ctx context.Context, req *dto.AddFunctionsReq) error {
	// 1. 根据 TreeID 获取 ServiceTree（需要预加载 App）
	serviceTree, err := s.serviceTreeRepo.GetByID(req.TreeID)
	if err != nil {
		logger.Errorf(ctx, "[FunctionGenService] 获取 ServiceTree 失败: TreeID=%d, error=%v", req.TreeID, err)
		return err
	}

	// 预加载 App 信息（如果还没有加载）
	if serviceTree.App == nil {
		app, err := s.appRepo.GetAppByID(serviceTree.AppID)
		if err != nil {
			logger.Errorf(ctx, "[FunctionGenService] 获取 App 失败: AppID=%d, error=%v", serviceTree.AppID, err)
			return err
		}
		serviceTree.App = app
	}

	// 2. 从 ServiceTree 中提取 package 路径（使用 model 方法）
	packagePath := serviceTree.GetPackagePathForFileCreation()

	// 3. 使用 agent-server 处理后的结构化数据
	// agent-server 已经处理了代码（提取代码、解析文件名）
	fileName := req.FileName
	if fileName == "" {
		// 如果 agent-server 没有提取到文件名，使用 ServiceTree.Code 作为 fallback
		logger.Warnf(ctx, "[FunctionGenService] agent-server 未提取到文件名，使用 ServiceTree.Code 作为 fallback: %s", serviceTree.Code)
		fileName = serviceTree.Code
	}

	sourceCode := req.SourceCode
	if sourceCode == "" {
		// 如果 agent-server 没有处理代码，使用原始代码（向后兼容）
		logger.Warnf(ctx, "[FunctionGenService] agent-server 未处理代码，使用原始代码")
		sourceCode = req.Code
	}

	logger.Infof(ctx, "[FunctionGenService] 接收处理后的数据: Package=%s, FileName=%s, SourceCodeLength=%d", packagePath, fileName, len(sourceCode))

	// 4. 构建 CreateFunctionInfo
	createFunction := &dto.CreateFunctionInfo{
		Package:    packagePath,
		GroupCode:  fileName, // 使用 FileName 作为 GroupCode（向后兼容）
		SourceCode: sourceCode, // 使用 agent-server 处理后的代码
	}

	// 6. 调用 AppService.UpdateApp，传入 CreateFunctions
	updateReq := &dto.UpdateAppReq{
		User:            req.User,
		App:             serviceTree.App.Code,
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
		AppID:         serviceTree.App.ID,
		AppCode:       serviceTree.App.Code,
		Error:         "",
	}

	// 从 ctx 中获取 traceID 和 token
	traceID := contextx.GetTraceId(ctx)
	token := contextx.GetToken(ctx)
	
	// 通过 HTTP 发送回调到 agent-server
	// ⭐ 服务间调用需要 token（用于权限验证），不需要 requestUser（网关会从 token 解析）
	apicallHeader := &apicall.Header{
		TraceID:     traceID,
		RequestUser: "", // 服务间调用不需要 requestUser，网关会从 token 解析
		Token:       token,
	}

	if err := apicall.NotifyWorkspaceUpdateComplete(apicallHeader, callbackData); err != nil {
		logger.Errorf(ctx, "[FunctionGenService] 通知工作空间更新完成失败: error=%v", err)
		// 不中断流程，记录日志即可
		} else {
			if len(fullCodePaths) > 0 {
				logger.Infof(ctx, "[FunctionGenService] 工作空间更新完成通知已发送 (HTTP) - RecordID: %d, MessageID: %d, FullCodePaths: %v, AppCode: %s",
					req.RecordID, req.MessageID, fullCodePaths, serviceTree.App.Code)
			} else {
				logger.Infof(ctx, "[FunctionGenService] 工作空间更新完成通知已发送 (HTTP) - RecordID: %d, MessageID: %d, FullCodePaths: [] (无新增函数), AppCode: %s",
					req.RecordID, req.MessageID, serviceTree.App.Code)
			}
		}

	return nil
}

// ProcessFunctionGenResultAsync 异步处理函数生成结果（通过回调通知结果）
func (s *FunctionGenService) ProcessFunctionGenResultAsync(ctx context.Context, req *dto.AddFunctionsReq) error {
	// 1. 根据 TreeID 获取 ServiceTree（需要预加载 App）
	serviceTree, err := s.serviceTreeRepo.GetByID(req.TreeID)
	if err != nil {
		logger.Errorf(ctx, "[FunctionGenService] 获取 ServiceTree 失败: TreeID=%d, error=%v", req.TreeID, err)
		// 发送失败回调（serviceTree 可能为 nil，但 sendCallback 会处理）
		s.sendCallback(ctx, req, nil, false, []string{}, err.Error())
		return err
	}

	// 预加载 App 信息（如果还没有加载）
	if serviceTree.App == nil {
		app, err := s.appRepo.GetAppByID(serviceTree.AppID)
		if err != nil {
			logger.Errorf(ctx, "[FunctionGenService] 获取 App 失败: AppID=%d, error=%v", serviceTree.AppID, err)
			s.sendCallback(ctx, req, serviceTree, false, []string{}, err.Error())
			return err
		}
		serviceTree.App = app
	}

	// 2. 从 ServiceTree 中提取 package 路径
	packagePath := serviceTree.GetPackagePathForFileCreation()

	// 3. 使用 agent-server 处理后的结构化数据
	fileName := req.FileName
	if fileName == "" {
		logger.Warnf(ctx, "[FunctionGenService] agent-server 未提取到文件名，使用 ServiceTree.Code 作为 fallback: %s", serviceTree.Code)
		fileName = serviceTree.Code
	}

	sourceCode := req.SourceCode
	if sourceCode == "" {
		logger.Warnf(ctx, "[FunctionGenService] agent-server 未处理代码，使用原始代码")
		sourceCode = req.Code
	}

	logger.Infof(ctx, "[FunctionGenService] 异步处理: Package=%s, FileName=%s, SourceCodeLength=%d", packagePath, fileName, len(sourceCode))

	// 4. 构建 CreateFunctionInfo
	createFunction := &dto.CreateFunctionInfo{
		Package:    packagePath,
		GroupCode:  fileName,
		SourceCode: sourceCode,
	}

	// 5. 调用 AppService.UpdateApp
	updateReq := &dto.UpdateAppReq{
		User:            req.User,
		App:             serviceTree.App.Code,
		CreateFunctions: []*dto.CreateFunctionInfo{createFunction},
	}

	updateResp, err := s.appService.UpdateApp(ctx, updateReq)
	if err != nil {
		logger.Errorf(ctx, "[FunctionGenService] AppService.UpdateApp 失败: error=%v", err)
		s.sendCallback(ctx, req, serviceTree, false, []string{}, err.Error())
		return err
	}

	// 6. 获取新增的 FullCodePaths
	fullCodePaths := make([]string, 0)
	if updateResp.Diff != nil {
		fullCodePaths = updateResp.Diff.GetAddFullCodePaths()
		logger.Infof(ctx, "[FunctionGenService] 异步处理完成 - Count: %d, FullCodePaths: %v", len(fullCodePaths), fullCodePaths)
	}

	// 7. 发送回调通知（异步模式必须发送回调）
	s.sendCallback(ctx, req, serviceTree, true, fullCodePaths, "")

	return nil
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

	// 从 ctx 中获取 traceID 和 token
	traceID := contextx.GetTraceId(ctx)
	token := contextx.GetToken(ctx)
	
	// ⭐ 服务间调用需要 token（用于权限验证），不需要 requestUser（网关会从 token 解析）
	apicallHeader := &apicall.Header{
		TraceID:     traceID,
		RequestUser: "", // 服务间调用不需要 requestUser，网关会从 token 解析
		Token:       token,
	}

	if err := apicall.NotifyWorkspaceUpdateComplete(apicallHeader, callbackData); err != nil {
		logger.Errorf(ctx, "[FunctionGenService] 发送回调失败: error=%v", err)
	} else {
		logger.Infof(ctx, "[FunctionGenService] 回调已发送 - RecordID: %d, Success: %v", req.RecordID, success)
	}
}

