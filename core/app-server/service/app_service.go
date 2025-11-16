package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"strconv"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"gorm.io/gorm"
)

type AppService struct {
	appRuntime      *AppRuntime
	userRepo        *repository.UserRepository
	appRepo         *repository.AppRepository
	functionRepo    *repository.FunctionRepository
	serviceTreeRepo *repository.ServiceTreeRepository
	sourceCodeRepo  *repository.SourceCodeRepository
}

// NewAppService 创建 AppService（依赖注入）
func NewAppService(appRuntime *AppRuntime, userRepo *repository.UserRepository, appRepo *repository.AppRepository, functionRepo *repository.FunctionRepository, serviceTreeRepo *repository.ServiceTreeRepository, sourceCodeRepo *repository.SourceCodeRepository) *AppService {
	return &AppService{
		appRuntime:      appRuntime,
		userRepo:        userRepo,
		appRepo:         appRepo,
		functionRepo:    functionRepo,
		serviceTreeRepo: serviceTreeRepo,
		sourceCodeRepo:  sourceCodeRepo,
	}
}

// CreateApp 创建应用
func (a *AppService) CreateApp(ctx context.Context, req *dto.CreateAppReq) (*dto.CreateAppResp, error) {
	// 从请求体中获取租户用户信息（应用所有者）
	tenantUser := req.User
	if tenantUser == "" {
		return nil, fmt.Errorf("租户用户信息不能为空")
	}

	// 从 context 中获取请求用户信息（实际发起请求的用户）
	requestUser := contextx.GetRequestUser(ctx)
	if requestUser == "" {
		return nil, fmt.Errorf("请求用户信息不能为空")
	}

	// 根据租户用户获取主机和 NATS 信息
	user, err := a.userRepo.GetUserByUsernameWithHostAndNats(tenantUser)
	if err != nil {
		return nil, fmt.Errorf("获取租户用户 %s 的主机信息失败: %w", tenantUser, err)
	}

	// 创建前校验：同一用户下应用中文名称是否重复
	if exists, err := a.appRepo.ExistsAppNameForUser(tenantUser, req.Name); err != nil {
		return nil, fmt.Errorf("检查应用名称唯一性失败: %w", err)
	} else if exists {
		return nil, fmt.Errorf("应用名称已存在: %s", req.Name)
	}

	// 创建包含用户信息的请求对象（内部使用）

	resp, err := a.appRuntime.CreateApp(ctx, user.HostID, req)
	if err != nil {
		return nil, err
	}

	// 写入数据库记录
	app := model.App{
		Base: models.Base{
			CreatedBy: requestUser, // 记录实际请求用户（谁发起的请求）
		},
		Version: "v1",
		Code:    req.Code,
		Name:    req.Name,   // 应用名称
		User:    tenantUser, // 记录租户用户（应用所有者）
		NatsID:  user.Host.NatsID,
		HostID:  user.Host.ID,
		Status:  "enabled",
	}
	err = a.appRepo.CreateApp(&app)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// UpdateApp 更新应用
func (a *AppService) UpdateApp(ctx context.Context, req *dto.UpdateAppReq) (*dto.UpdateAppResp, error) {
	// 根据应用信息获取 NATS 连接，而不是根据当前用户
	app, err := a.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		return nil, err
	}

	// 调用 app-runtime 更新应用，使用应用所属的 HostID
	resp, err := a.appRuntime.UpdateApp(ctx, app.HostID, req)
	if err != nil {
		return nil, err
	}

	// 更新数据库中的版本信息
	app.Version = resp.NewVersion
	err = a.appRepo.UpdateApp(app)
	if err != nil {
		return nil, err
	}

	// 处理API差异，将API信息入库到function表
	if resp.Diff != nil {
		err = a.processAPIDiff(ctx, app.ID, resp.Diff)
		if err != nil {
			// API入库失败不应该影响主流程，记录日志即可
			fmt.Printf("API入库失败: %v\n", err)
		}
	}

	return resp, nil
}

// extractVersionNum 从版本号字符串中提取数字部分（如 "v1" -> 1, "v20" -> 20）
func extractVersionNum(version string) int {
	if version == "" {
		return 0
	}
	// 去掉 "v" 前缀
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")
	// 提取数字部分
	num, err := strconv.Atoi(version)
	if err != nil {
		return 0
	}
	return num
}

// RequestApp 请求应用
func (a *AppService) RequestApp(ctx context.Context, req *dto.RequestAppReq) (*dto.RequestAppResp, error) {
	app, err := a.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		return nil, err
	}
	req.Version = app.Version
	resp, err := a.appRuntime.RequestApp(ctx, app.NatsID, req)
	if err != nil {
		return nil, err
	}
	resp.Version = req.Version
	return resp, nil
}

// processAPIDiff 处理API差异，包括新增、更新、删除
func (a *AppService) processAPIDiff(ctx context.Context, appID int64, diffData *dto.DiffData) error {
	// 获取应用信息（用于获取版本号）
	app, err := a.appRepo.GetAppByID(appID)
	if err != nil {
		return fmt.Errorf("获取应用信息失败: %w", err)
	}

	// 处理新增的API
	if len(diffData.Add) > 0 {
		// 1. 先转换API为Function模型（但不创建）
		functions, err := a.convertApiInfoToFunctions(appID, diffData.Add)
		if err != nil {
			return fmt.Errorf("转换新增API失败: %w", err)
		}

		// 2. 先保存源代码（按函数组分组），并设置 Function 的 SourceCodeID
		err = a.saveSourceCodeForAPIs(ctx, appID, app.Version, diffData.Add, functions)
		if err != nil {
			return fmt.Errorf("保存源代码失败: %w", err)
		}

		// 3. 创建Function记录（此时 SourceCodeID 已经设置好）
		err = a.functionRepo.CreateFunctions(functions)
		if err != nil {
			return fmt.Errorf("创建function记录失败: %w", err)
		}

		// 4. 创建ServiceTree记录，使用Function的ID作为RefID
		err = a.createServiceTreesForAPIs(ctx, appID, diffData.Add, functions)
		if err != nil {
			return fmt.Errorf("创建service_tree记录失败: %w", err)
		}
	}

	// 处理更新的API
	if len(diffData.Update) > 0 {
		// 1. 转换更新的API为Function模型
		functions, err := a.convertApiInfoToFunctions(appID, diffData.Update)
		if err != nil {
			return fmt.Errorf("转换更新API失败: %w", err)
		}

		// 2. 先更新源代码（按函数组分组），并设置 Function 的 SourceCodeID
		err = a.saveSourceCodeForAPIs(ctx, appID, app.Version, diffData.Update, functions)
		if err != nil {
			return fmt.Errorf("更新源代码失败: %w", err)
		}

		// 3. 更新Function记录（此时 SourceCodeID 已经设置好）
		err = a.updateFunctionsForAPIs(ctx, appID, diffData.Update, functions)
		if err != nil {
			return fmt.Errorf("更新function记录失败: %w", err)
		}

		// 4. 更新ServiceTree记录
		err = a.updateServiceTreesForAPIs(ctx, appID, diffData.Update, functions)
		if err != nil {
			return fmt.Errorf("更新service_tree记录失败: %w", err)
		}
	}

	// 处理删除的API
	if len(diffData.Delete) > 0 {
		err := a.deleteFunctionsForAPIs(ctx, appID, diffData.Delete)
		if err != nil {
			return fmt.Errorf("删除function和service_tree记录失败: %w", err)
		}
	}

	return nil
}

// convertApiInfoToFunctions 将ApiInfo转换为Function模型
func (a *AppService) convertApiInfoToFunctions(appID int64, apis []*dto.ApiInfo) ([]*model.Function, error) {
	functions := make([]*model.Function, len(apis))

	for i, api := range apis {
		// 序列化request字段
		var requestJSON json.RawMessage
		if len(api.Request) > 0 {
			requestData, err := json.Marshal(api.Request)
			if err != nil {
				return nil, fmt.Errorf("序列化request字段失败: %w", err)
			}
			requestJSON = requestData
		}

		// 序列化response字段
		var responseJSON json.RawMessage
		if len(api.Response) > 0 {
			responseData, err := json.Marshal(api.Response)
			if err != nil {
				return nil, fmt.Errorf("序列化response字段失败: %w", err)
			}
			responseJSON = responseData
		}

		// 序列化create_tables字段

		function := &model.Function{
			AppID:        appID,
			Method:       api.Method,
			Router:       api.BuildFullCodePath(),
			Request:      requestJSON,
			Response:     responseJSON,
			HasConfig:    false, // 预留字段，默认为false
			TemplateType: api.TemplateType,
			Callbacks:    strings.Join(api.Callback, ","),
		}
		if api.CreateTables != nil {
			function.CreateTables = strings.Join(api.CreateTables, ",")
		}

		functions[i] = function
	}

	return functions, nil
}

// createServiceTreesForAPIs 为新增的API创建ServiceTree记录
func (a *AppService) createServiceTreesForAPIs(ctx context.Context, appID int64, apis []*dto.ApiInfo, functions []*model.Function) error {
	// 获取应用信息，用于预加载到ServiceTree

	// 收集所有需要查询的父级路径
	parentPaths := make(map[string]bool)
	for _, api := range apis {
		parentPath := api.GetParentFullCodePath()
		if parentPath != "" {
			parentPaths[parentPath] = true
		}
	}

	// 批量查询所有父级package节点
	parentPathList := make([]string, 0, len(parentPaths))
	for path := range parentPaths {
		parentPathList = append(parentPathList, path)
	}

	parentNodes, err := a.serviceTreeRepo.GetServiceTreeByFullPaths(parentPathList)
	if err != nil {
		return fmt.Errorf("批量查询父级package节点失败: %w", err)
	}

	// 验证所有父级节点都是package类型
	for path, node := range parentNodes {
		if !node.IsPackage() {
			return fmt.Errorf("路径 %s 已存在，但类型不是package，当前类型: %s", path, node.Type)
		}
	}

	// 创建function节点
	for i, api := range apis {
		var parentID int64 = 0
		parentPath := api.GetParentFullCodePath()

		if parentPath != "" {
			parent, exists := parentNodes[parentPath]
			if !exists {
				return fmt.Errorf("父级package节点不存在: %s", parentPath)
			}
			parentID = parent.ID
		}

		// 创建function节点，使用Function的ID作为RefID
		err = a.createFunctionNode(appID, parentID, api, functions[i].ID)
		if err != nil {
			return fmt.Errorf("创建function节点失败: %w", err)
		}
	}
	return nil
}

// createFunctionNode 创建function节点
func (a *AppService) createFunctionNode(appID int64, parentID int64, api *dto.ApiInfo, functionID int64) error {
	// 检查是否已存在（full_name_path全局唯一）
	existingNode, err := a.serviceTreeRepo.GetServiceTreeByFullPath(api.FullCodePath)
	if err == nil {
		// 如果路径已存在，更新版本号而不是创建新节点
		// 获取应用当前版本
		app, err := a.appRepo.GetAppByID(appID)
		if err != nil {
			return fmt.Errorf("获取应用信息失败: %w", err)
		}
		currentVersionNum := extractVersionNum(app.Version)

		// 如果节点是新增的（AddVersionNum为0），设置添加版本号
		if existingNode.AddVersionNum == 0 {
			existingNode.AddVersionNum = currentVersionNum
		} else {
			// 如果节点已存在，更新更新版本号
			existingNode.UpdateVersionNum = currentVersionNum
		}

		// 更新节点信息
		return a.serviceTreeRepo.UpdateServiceTree(existingNode)
	}
	// 如果是记录不存在的错误，这是正常的，继续创建
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询路径失败: %w", err)
	}
	// err是gorm.ErrRecordNotFound，说明路径不存在，可以继续创建

	// 获取应用当前版本
	app, err := a.appRepo.GetAppByID(appID)
	if err != nil {
		return fmt.Errorf("获取应用信息失败: %w", err)
	}
	currentVersionNum := extractVersionNum(app.Version)

	// 构建 FullGroupCode：{full_path}/{group_code}
	fullGroupCode := fmt.Sprintf("%s/%s", api.GetParentFullCodePath(), api.FunctionGroupCode)
	
	// 创建新的function节点，预加载完整的app对象
	serviceTree := &model.ServiceTree{
		AppID:           appID,
		ParentID:         parentID,
		FullGroupCode:   fullGroupCode, // 完整函数组代码：{full_path}/{group_code}，与 source_code.full_group_code 对齐
		GroupName:       api.FunctionGroupName,
		Type:            model.ServiceTreeTypeFunction,
		Code:            api.Code, // API的code作为ServiceTree的code
		Name:            api.Name, // API的name作为ServiceTree的name
		Description:     api.Desc,
		RefID:           functionID,        // 指向Function记录的ID
		FullCodePath:    api.FullCodePath,  // 直接使用api.FullCodePath，不需要重新计算
		AddVersionNum:   currentVersionNum, // 设置添加版本号
		UpdateVersionNum: 0,                 // 新增节点，更新版本号为0
	}

	if len(api.Tags) > 0 {
		serviceTree.Tags = strings.Join(api.Tags, ",")
	}

	// 创建ServiceTree节点
	return a.serviceTreeRepo.CreateServiceTreeWithParentPath(serviceTree, "")
}

// updateFunctionsForAPIs 更新API对应的Function记录
func (a *AppService) updateFunctionsForAPIs(ctx context.Context, appID int64, apis []*dto.ApiInfo, functions []*model.Function) error {
	// 对于每个要更新的API，先查找现有的Function记录获取ID
	for i, api := range apis {
		router := api.BuildFullCodePath()
		existingFunction, err := a.functionRepo.GetFunctionByKey(appID, api.Method, router)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Function不存在，创建新的（这种情况不应该发生，但为了容错处理）
				newFunctions := []*model.Function{functions[i]}
				err = a.functionRepo.CreateFunctions(newFunctions)
				if err != nil {
					return fmt.Errorf("创建function记录失败: %w", err)
				}
				// 更新functions[i]的ID
				functions[i].ID = newFunctions[0].ID
				continue
			}
			return fmt.Errorf("查询function记录失败: %w", err)
		}
		// 保留现有的ID
		functions[i].ID = existingFunction.ID
	}

	// 批量更新Function记录
	return a.functionRepo.UpdateFunctions(functions)
}

// updateServiceTreesForAPIs 更新API对应的ServiceTree记录
func (a *AppService) updateServiceTreesForAPIs(ctx context.Context, appID int64, apis []*dto.ApiInfo, functions []*model.Function) error {
	// 获取应用当前版本
	app, err := a.appRepo.GetAppByID(appID)
	if err != nil {
		return fmt.Errorf("获取应用信息失败: %w", err)
	}
	currentVersionNum := extractVersionNum(app.Version)

	// 收集所有需要查询的父级路径
	parentPaths := make(map[string]bool)
	for _, api := range apis {
		parentPath := api.GetParentFullCodePath()
		if parentPath != "" {
			parentPaths[parentPath] = true
		}
	}

	// 批量查询所有父级package节点
	parentPathList := make([]string, 0, len(parentPaths))
	for path := range parentPaths {
		parentPathList = append(parentPathList, path)
	}

	parentNodes, err := a.serviceTreeRepo.GetServiceTreeByFullPaths(parentPathList)
	if err != nil {
		return fmt.Errorf("批量查询父级package节点失败: %w", err)
	}

	// 验证所有父级节点都是package类型
	for path, node := range parentNodes {
		if !node.IsPackage() {
			return fmt.Errorf("路径 %s 已存在，但类型不是package，当前类型: %s", path, node.Type)
		}
	}

	// 更新function节点
	for i, api := range apis {
		// 根据FullCodePath查找现有的ServiceTree
		existingTree, err := a.serviceTreeRepo.GetServiceTreeByFullPath(api.FullCodePath)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 如果不存在，创建新的节点（这种情况不应该发生，但为了容错处理）
				var parentID int64 = 0
				parentPath := api.GetParentFullCodePath()
				if parentPath != "" {
					parent, exists := parentNodes[parentPath]
					if exists {
						parentID = parent.ID
					}
				}
				err = a.createFunctionNode(appID, parentID, api, functions[i].ID)
				if err != nil {
					return fmt.Errorf("创建function节点失败: %w", err)
				}
				continue
			}
			return fmt.Errorf("查询service_tree失败: %w", err)
		}

		// 构建 FullGroupCode：{full_path}/{group_code}
		fullGroupCode := fmt.Sprintf("%s/%s", api.GetParentFullCodePath(), api.FunctionGroupCode)
		
		// 更新节点信息并设置更新版本号
		existingTree.RefID = functions[i].ID
		existingTree.Name = api.Name
		existingTree.Description = api.Desc
		existingTree.FullGroupCode = fullGroupCode // 完整函数组代码：{full_path}/{group_code}，与 source_code.full_group_code 对齐
		existingTree.GroupName = api.FunctionGroupName
		// 更新版本号：如果AddVersionNum为0，说明是新增的，设置为当前版本；否则更新UpdateVersionNum
		if existingTree.AddVersionNum == 0 {
			existingTree.AddVersionNum = currentVersionNum
		} else {
			existingTree.UpdateVersionNum = currentVersionNum
		}

		if len(api.Tags) > 0 {
			existingTree.Tags = strings.Join(api.Tags, ",")
		}

		// 保存更新后的节点
		if err := a.serviceTreeRepo.UpdateServiceTree(existingTree); err != nil {
			return fmt.Errorf("更新service_tree节点失败: %w", err)
		}
	}
	return nil
}

// deleteFunctionsForAPIs 删除API对应的Function和ServiceTree记录
func (a *AppService) deleteFunctionsForAPIs(ctx context.Context, appID int64, apis []*dto.ApiInfo) error {
	// 收集需要删除的router和method
	routers := make([]string, 0, len(apis))
	methods := make([]string, 0, len(apis))

	for _, api := range apis {
		// 根据FullCodePath查找ServiceTree
		serviceTree, err := a.serviceTreeRepo.GetServiceTreeByFullPath(api.FullCodePath)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// ServiceTree不存在，跳过
				continue
			}
			return fmt.Errorf("查询service_tree失败: %w", err)
		}

		// 删除ServiceTree（会级联删除子节点）
		err = a.serviceTreeRepo.DeleteServiceTree(serviceTree.ID)
		if err != nil {
			return fmt.Errorf("删除service_tree失败: %w", err)
		}

		// 收集Function的router和method用于删除
		router := api.BuildFullCodePath()
		routers = append(routers, router)
		methods = append(methods, api.Method)
	}

	// 批量删除Function记录
	if len(routers) > 0 {
		err := a.functionRepo.DeleteFunctions(appID, routers, methods)
		if err != nil {
			return fmt.Errorf("删除function记录失败: %w", err)
		}
	}

	return nil
}

// saveSourceCodeForAPIs 保存源代码（按函数组分组）
// 同一个函数组（GroupCode）下的所有函数共享同一个 SourceCode 记录
func (a *AppService) saveSourceCodeForAPIs(ctx context.Context, appID int64, version string, apis []*dto.ApiInfo, functions []*model.Function) error {
	logger.Infof(ctx, "[saveSourceCodeForAPIs] 开始保存源代码: appID=%d, version=%s, apiCount=%d", appID, version, len(apis))
	
	// 按函数组分组（GroupCode）
	groupMap := make(map[string]*struct {
		apis      []*dto.ApiInfo
		functions []*model.Function
		fullPath  string
		groupCode string
		sourceCode string
	})

	for i, api := range apis {
		groupCode := api.FunctionGroupCode
		if groupCode == "" {
			// 记录警告，但不中断流程
			logger.Warnf(ctx, "[saveSourceCodeForAPIs] API %s %s 没有 FunctionGroupCode，跳过源代码保存", api.Method, api.Router)
			continue // 跳过没有函数组的API
		}

		// 计算 fullPath（package 的完整路径）
		// 从 SourceCodeFilePath 中提取，格式为：/{user}/{app}/{package_path}/{group_code}
		// 我们需要提取到 package_path 部分
		fullPath := api.GetParentFullCodePath() // 例如：/luobei/testgroup/tools
		
		logger.Infof(ctx, "[saveSourceCodeForAPIs] 处理API: method=%s, router=%s, groupCode=%s, fullPath=%s, sourceCodeLength=%d", 
			api.Method, api.Router, groupCode, fullPath, len(api.SourceCode))
		
		// 如果 groupMap 中还没有这个函数组，创建新的
		if _, exists := groupMap[groupCode]; !exists {
			groupMap[groupCode] = &struct {
				apis       []*dto.ApiInfo
				functions  []*model.Function
				fullPath   string
				groupCode  string
				sourceCode string
			}{
				apis:       []*dto.ApiInfo{},
				functions:  []*model.Function{},
				fullPath:   fullPath,
				groupCode:  groupCode,
				sourceCode: api.SourceCode, // 使用第一个API的源代码
			}
			logger.Infof(ctx, "[saveSourceCodeForAPIs] 创建新的函数组: groupCode=%s, fullPath=%s, sourceCodeLength=%d", 
				groupCode, fullPath, len(api.SourceCode))
		}

		// 添加到对应的函数组
		groupMap[groupCode].apis = append(groupMap[groupCode].apis, api)
		groupMap[groupCode].functions = append(groupMap[groupCode].functions, functions[i])
		
		// 如果当前API的源代码不为空，更新源代码（同一个函数组应该使用相同的源代码）
		if api.SourceCode != "" {
			groupMap[groupCode].sourceCode = api.SourceCode
			logger.Infof(ctx, "[saveSourceCodeForAPIs] 更新函数组源代码: groupCode=%s, sourceCodeLength=%d", 
				groupCode, len(api.SourceCode))
		}
	}

	logger.Infof(ctx, "[saveSourceCodeForAPIs] 函数组分组完成: groupCount=%d", len(groupMap))

	// 为每个函数组保存或更新 SourceCode 记录
	for groupCode, group := range groupMap {
		if group.sourceCode == "" {
			logger.Warnf(ctx, "[saveSourceCodeForAPIs] 函数组 %s 的源代码为空，跳过保存", groupCode)
			continue // 跳过没有源代码的函数组
		}

		// 构建 FullGroupCode：{full_path}/{group_code}
		fullGroupCode := fmt.Sprintf("%s/%s", group.fullPath, groupCode)
		logger.Infof(ctx, "[saveSourceCodeForAPIs] 准备保存源代码: fullGroupCode=%s, groupCode=%s, fullPath=%s, contentLength=%d, functionCount=%d", 
			fullGroupCode, groupCode, group.fullPath, len(group.sourceCode), len(group.functions))

		// 创建或更新 SourceCode 记录
		sourceCode := &model.SourceCode{
			FullGroupCode: fullGroupCode, // 完整函数组代码：{full_path}/{group_code}，与 service_tree.full_group_code 对齐
			FullPath:      group.fullPath,
			GroupCode:     groupCode,
			Content:       group.sourceCode,
			AppID:         appID,
			Version:       version,
		}

		logger.Infof(ctx, "[saveSourceCodeForAPIs] 调用 GetOrCreate: fullGroupCode=%s, appID=%d, version=%s", fullGroupCode, appID, version)
		savedSourceCode, err := a.sourceCodeRepo.GetOrCreate(sourceCode)
		if err != nil {
			logger.Errorf(ctx, "[saveSourceCodeForAPIs] 保存源代码失败: fullGroupCode=%s, error=%v", fullGroupCode, err)
			return fmt.Errorf("保存源代码失败（函数组: %s）: %w", groupCode, err)
		}

		logger.Infof(ctx, "[saveSourceCodeForAPIs] 源代码保存成功: ID=%d, fullGroupCode=%s, contentLength=%d", 
			savedSourceCode.ID, fullGroupCode, len(savedSourceCode.Content))

		// 设置所有函数的 SourceCodeID（在创建/更新 Function 之前）
		for _, function := range group.functions {
			sourceCodeID := savedSourceCode.ID
			function.SourceCodeID = &sourceCodeID
			logger.Infof(ctx, "[saveSourceCodeForAPIs] 设置 Function SourceCodeID: functionID=%d, sourceCodeID=%d, method=%s, router=%s", 
				function.ID, sourceCodeID, function.Method, function.Router)
		}
	}

	logger.Infof(ctx, "[saveSourceCodeForAPIs] 源代码保存完成: appID=%d, version=%s", appID, version)
	return nil
}

// DeleteApp 删除应用
func (a *AppService) DeleteApp(ctx context.Context, req *dto.DeleteAppReq) (*dto.DeleteAppResp, error) {
	// 根据应用信息获取 NATS 连接
	app, err := a.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		return nil, err
	}

	// 调用 app-runtime 删除应用
	resp, err := a.appRuntime.DeleteApp(ctx, app.HostID, req)
	if err != nil {
		return nil, err
	}

	// 删除数据库记录
	err = a.appRepo.DeleteAppAndVersions(req.User, req.App)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetApps 获取应用列表
func (a *AppService) GetApps(ctx context.Context, req *dto.GetAppsReq) (*dto.GetAppsResp, error) {
	// 设置分页默认值
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10 // 默认每页10条
	}

	// 从数据库获取用户的分页应用列表（支持搜索）
	apps, totalCount, err := a.appRepo.GetAppsByUserWithPage(req.User, page, pageSize, req.Search)
	if err != nil {
		return nil, fmt.Errorf("获取应用列表失败: %w", err)
	}

	return &dto.GetAppsResp{
		PageInfoResp: dto.PageInfoResp{
			Page:       page,
			PageSize:   pageSize,
			TotalCount: int(totalCount),
			Items:      apps,
		},
	}, nil
}
