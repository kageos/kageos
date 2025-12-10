package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"gorm.io/gorm"
)

type FunctionService struct {
	functionRepo    *repository.FunctionRepository
	sourceCodeRepo  *repository.SourceCodeRepository
	appRepo         *repository.AppRepository
	serviceTreeRepo *repository.ServiceTreeRepository
	appRuntime      *AppRuntime
	appService      *AppService
}

// NewFunctionService 创建函数服务
func NewFunctionService(
	functionRepo *repository.FunctionRepository,
	sourceCodeRepo *repository.SourceCodeRepository,
	appRepo *repository.AppRepository,
	serviceTreeRepo *repository.ServiceTreeRepository,
	appRuntime *AppRuntime,
	appService *AppService,
) *FunctionService {
	return &FunctionService{
		functionRepo:    functionRepo,
		sourceCodeRepo:  sourceCodeRepo,
		appRepo:         appRepo,
		serviceTreeRepo: serviceTreeRepo,
		appRuntime:      appRuntime,
		appService:      appService,
	}
}

// GetFunction 获取函数详情
func (f *FunctionService) GetFunction(ctx context.Context, functionID int64) (*dto.GetFunctionResp, error) {
	// 从数据库获取函数信息
	function, err := f.functionRepo.GetFunctionByID(functionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("函数不存在")
		}
		return nil, fmt.Errorf("获取函数失败: %w", err)
	}

	// 转换为响应格式
	resp := &dto.GetFunctionResp{
		ID:           function.ID,
		AppID:        function.AppID,
		TreeID:       function.TreeID,
		Method:       function.Method,
		Router:       function.Router,
		HasConfig:    function.HasConfig,
		CreateTables: function.CreateTables,
		Callbacks:    function.Callbacks,
		TemplateType: function.TemplateType,
		CreatedAt:    time.Time(function.CreatedAt).Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    time.Time(function.UpdatedAt).Format("2006-01-02T15:04:05Z"),
	}

	// 将json.RawMessage转换为interface{}以便返回JSON对象
	if len(function.Request) > 0 {
		var requestMap interface{}
		if err := json.Unmarshal(function.Request, &requestMap); err != nil {
			requestMap = map[string]interface{}{}
		}
		resp.Request = requestMap
	} else {
		resp.Request = map[string]interface{}{}
	}

	if len(function.Response) > 0 {
		var responseMap interface{}
		if err := json.Unmarshal(function.Response, &responseMap); err != nil {
			responseMap = map[string]interface{}{}
		}
		resp.Response = responseMap
	} else {
		resp.Response = map[string]interface{}{}
	}

	logger.Infof(ctx, "[FunctionService] GetFunction success: functionID=%d", functionID)
	return resp, nil
}

// GetFunctionsByApp 获取应用下所有函数
func (f *FunctionService) GetFunctionsByApp(ctx context.Context, appID int64) (*dto.GetFunctionsByAppResp, error) {
	// 从数据库获取应用的所有函数
	functions, err := f.functionRepo.GetFunctionsByAppID(appID)
	if err != nil {
		return nil, fmt.Errorf("获取应用函数列表失败: %w", err)
	}

	// 转换为响应格式
	functionInfos := make([]dto.FunctionInfo, len(functions))
	for i, function := range functions {
		functionInfos[i] = dto.FunctionInfo{
			ID:           function.ID,
			AppID:        function.AppID,
			TreeID:       function.TreeID,
			Method:       function.Method,
			Router:       function.Router,
			HasConfig:    function.HasConfig,
			CreateTables: function.CreateTables,
			Callbacks:    function.Callbacks,
			TemplateType: function.TemplateType,
			CreatedAt:    time.Time(function.CreatedAt).Format("2006-01-02T15:04:05Z"),
			UpdatedAt:    time.Time(function.UpdatedAt).Format("2006-01-02T15:04:05Z"),
		}
	}

	resp := &dto.GetFunctionsByAppResp{
		Functions: functionInfos,
	}

	logger.Infof(ctx, "[FunctionService] GetFunctionsByApp success: appID=%d, count=%d", appID, len(functions))
	return resp, nil
}

// ForkFunctionGroup 批量 Fork 函数组（使用 map 形式，每个源可以指定不同的目标）
func (f *FunctionService) ForkFunctionGroup(ctx context.Context, req *dto.ForkFunctionGroupReq) (*dto.ForkFunctionGroupResp, error) {
	// 1. 获取目标应用信息
	targetApp, err := f.appRepo.GetAppByID(req.TargetAppID)
	if err != nil {
		return nil, fmt.Errorf("获取目标应用失败: %w", err)
	}

	// 2. 验证 map 不为空
	if len(req.SourceToTargetMap) == 0 {
		return nil, fmt.Errorf("source_to_target_map 不能为空")
	}

	logger.Infof(ctx, "[FunctionService] 开始批量 Fork 函数组: mapCount=%d, targetAppID=%d", len(req.SourceToTargetMap), req.TargetAppID)

	// 3. 按目标 package 分组，准备批量写入
	// key: target_package_full_group_code, value: 文件列表
	targetPackageFiles := make(map[string][]*dto.ForkFunctionGroupFile)

	for sourceFullGroupCode, targetPackageFullGroupCode := range req.SourceToTargetMap {
		// 解析目标 package 的 FullGroupCode
		targetParts := strings.Split(strings.Trim(targetPackageFullGroupCode, "/"), "/")
		if len(targetParts) < 3 {
			return nil, fmt.Errorf("目标 package FullGroupCode 格式错误: %s", targetPackageFullGroupCode)
		}
		targetUser := targetParts[0]
		targetAppCode := targetParts[1]

		// 验证目标应用信息
		if targetApp.User != targetUser || targetApp.Code != targetAppCode {
			return nil, fmt.Errorf("目标应用信息不匹配: expected=%s/%s, actual=%s/%s",
				targetUser, targetAppCode, targetApp.User, targetApp.Code)
		}

		// 查询源 SourceCode 记录
		sourceCode, err := f.sourceCodeRepo.GetByFullGroupCode(sourceFullGroupCode)
		if err != nil {
			logger.Errorf(ctx, "[FunctionService] 获取源 SourceCode 失败: fullGroupCode=%s, error=%v", sourceFullGroupCode, err)
			return nil, fmt.Errorf("获取源 SourceCode 失败 (%s): %w", sourceFullGroupCode, err)
		}

		// 从 FullGroupCode 解析出源 package 名称
		sourceParts := strings.Split(strings.Trim(sourceCode.FullGroupCode, "/"), "/")
		if len(sourceParts) < 4 {
			return nil, fmt.Errorf("源 FullGroupCode 格式错误: %s", sourceCode.FullGroupCode)
		}
		sourcePackage := strings.Join(sourceParts[2:len(sourceParts)-1], "/") // 去掉最后一部分（group_code）

		// 添加到对应目标 package 的文件列表
		if targetPackageFiles[targetPackageFullGroupCode] == nil {
			targetPackageFiles[targetPackageFullGroupCode] = make([]*dto.ForkFunctionGroupFile, 0)
		}
		targetPackageFiles[targetPackageFullGroupCode] = append(targetPackageFiles[targetPackageFullGroupCode], &dto.ForkFunctionGroupFile{
			GroupCode:     sourceCode.GroupCode,
			SourceCode:    sourceCode.Content,
			SourcePackage: sourcePackage,
		})

		logger.Infof(ctx, "[FunctionService] 准备 Fork: source=%s, target=%s, groupCode=%s",
			sourceFullGroupCode, targetPackageFullGroupCode, sourceCode.GroupCode)
	}

	// 4. 构建批量请求，一次调用处理所有 package
	packages := make([]*dto.ForkPackageInfo, 0, len(targetPackageFiles))
	totalFileCount := 0

	for targetPackageFullGroupCode, files := range targetPackageFiles {
		// 解析目标 package 信息
		targetParts := strings.Split(strings.Trim(targetPackageFullGroupCode, "/"), "/")
		targetPackage := strings.Join(targetParts[2:], "/") // 支持多级 package

		packages = append(packages, &dto.ForkPackageInfo{
			Package: targetPackage,
			Files:   files,
		})

		totalFileCount += len(files)
		logger.Infof(ctx, "[FunctionService] 准备 Fork package %s: fileCount=%d", targetPackageFullGroupCode, len(files))
	}

	// 4. 直接调用 UpdateApp，传入 ForkPackages（合并 fork 和更新操作）
	logger.Infof(ctx, "[FunctionService] 准备调用 UpdateApp（包含 fork 操作）: packageCount=%d, totalFileCount=%d", len(packages), totalFileCount)

	updateReq := &dto.UpdateAppReq{
		User:         targetApp.User,
		App:          targetApp.Code,
		ForkPackages: packages, // 将 fork 包列表传递给 UpdateApp
	}

	_, err = f.appService.UpdateApp(ctx, updateReq)
	if err != nil {
		logger.Errorf(ctx, "[FunctionService] UpdateApp（包含 fork）失败: error=%v", err)
		return nil, fmt.Errorf("UpdateApp 失败: %w", err)
	}

	logger.Infof(ctx, "[FunctionService] 批量 Fork 完成: totalFileCount=%d, targetPackageCount=%d", totalFileCount, len(targetPackageFiles))

	return &dto.ForkFunctionGroupResp{
		Message: fmt.Sprintf("成功 Fork %d 个函数组到 %d 个目标目录", totalFileCount, len(targetPackageFiles)),
	}, nil
}

// GetFunctionGroupInfo 获取函数组信息（用于 Hub 发布）
func (f *FunctionService) GetFunctionGroupInfo(ctx context.Context, fullGroupCode string) (*dto.GetFunctionGroupInfoResp, error) {
	// 1. 获取源代码
	sourceCode, err := f.sourceCodeRepo.GetByFullGroupCode(fullGroupCode)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("函数组不存在: %s", fullGroupCode)
		}
		return nil, fmt.Errorf("获取源代码失败: %w", err)
	}

	// 2. 获取 ServiceTree 信息（用于获取 group_name 和函数名称/描述）
	// 注意：同一个 full_group_code 下可能有多个函数，每个函数一条 ServiceTree 记录
	// 它们共享相同的 group_name，但每个函数的 description 是不同的
	// 函数组没有统一的描述，所以 description 返回空字符串，让用户在 Hub 上自己填写
	serviceTrees, err := f.serviceTreeRepo.GetServiceTreesByFullGroupCode(fullGroupCode)
	var groupName string
	if err == nil && len(serviceTrees) > 0 {
		// 取第一条的 group_name（同一个函数组下的 group_name 应该相同）
		if serviceTrees[0].GroupName != "" {
			groupName = serviceTrees[0].GroupName
		} else if serviceTrees[0].Name != "" {
			groupName = serviceTrees[0].Name
		}
	} else if err != nil {
		// ServiceTree 不存在不影响，使用默认值
		logger.Warnf(ctx, "[FunctionService] 获取 ServiceTree 失败: fullGroupCode=%s, error=%v", fullGroupCode, err)
		serviceTrees = []*model.ServiceTree{} // 确保不为 nil
	}

	// 3. 获取函数列表（快照）
	functions, err := f.functionRepo.GetBySourceCodeID(sourceCode.ID)
	if err != nil {
		logger.Warnf(ctx, "[FunctionService] 获取函数列表失败: sourceCodeID=%d, error=%v", sourceCode.ID, err)
		functions = []*model.Function{} // 确保不为 nil
	}

	// 4. 获取应用信息
	app, err := f.appRepo.GetAppByID(sourceCode.AppID)
	if err != nil {
		logger.Warnf(ctx, "[FunctionService] 获取应用信息失败: appID=%d, error=%v", sourceCode.AppID, err)
	}

	// 5. 建立 ServiceTree RefID -> ServiceTree 的映射（用于获取函数名称和描述）
	// ServiceTree 的 RefID 指向 Function.ID，type = 'function'
	serviceTreeMap := make(map[int64]*model.ServiceTree)
	if len(serviceTrees) > 0 {
		for _, st := range serviceTrees {
			if st.RefID > 0 {
				serviceTreeMap[st.RefID] = st
			}
		}
	}

	// 6. 转换为 FunctionInfo 列表
	functionInfos := make([]dto.FunctionInfo, len(functions))
	for i, function := range functions {
		var functionName, functionDescription string
		if st, exists := serviceTreeMap[function.ID]; exists {
			functionName = st.Name
			functionDescription = st.Description
		}
		
		functionInfos[i] = dto.FunctionInfo{
			ID:           function.ID,
			AppID:        function.AppID,
			TreeID:       function.TreeID,
			Method:       function.Method,
			Router:       function.Router,
			HasConfig:    function.HasConfig,
			CreateTables: function.CreateTables,
			Callbacks:    function.Callbacks,
			TemplateType: function.TemplateType,
			CreatedAt:    time.Time(function.CreatedAt).Format("2006-01-02T15:04:05Z"),
			UpdatedAt:    time.Time(function.UpdatedAt).Format("2006-01-02T15:04:05Z"),
			Name:         functionName,        // 函数名称
			Description:  functionDescription, // 函数描述
		}
	}

	// 7. 组装响应数据
	resp := &dto.GetFunctionGroupInfoResp{
		// 核心数据
		SourceCode:  sourceCode.Content,
		Description: "", // 函数组没有统一的描述（每个函数的描述不同），返回空字符串，让用户在 Hub 上自己填写
		// 快照信息
		FullGroupCode: sourceCode.FullGroupCode,
		GroupCode:     sourceCode.GroupCode,
		GroupName:     groupName, // 从 ServiceTree 获取 group_name
		FullPath:      sourceCode.FullPath,
		Version:       sourceCode.Version,
		AppID:         sourceCode.AppID,
		AppName:       "",
		FunctionCount: len(functions),
		Functions:     functionInfos, // 函数列表
	}

	// 从 App 获取应用名称
	if app != nil {
		resp.AppName = app.Code
	}

	logger.Infof(ctx, "[FunctionService] GetFunctionGroupInfo success: fullGroupCode=%s, functionCount=%d", fullGroupCode, resp.FunctionCount)
	return resp, nil
}
