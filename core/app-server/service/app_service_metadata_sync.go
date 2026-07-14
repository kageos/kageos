package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kageos/kageos/pkg/contextx"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/functionschema"
	"github.com/kageos/kageos/pkg/logger"
	"gorm.io/gorm"
)

// processAPIDiff 处理API差异，包括新增、更新、删除
func (a *AppService) processAPIDiff(ctx context.Context, appID int64, diffData *dto.DiffData) error {
	state, err := a.loadAppMetadataSyncState(ctx, appID)
	if err != nil {
		return err
	}

	// ⭐ 第一步：目录对账——SDK 返回全量 package 列表，先与数据库比对，缺失的统一创建
	if len(diffData.Packages) > 0 {
		if err := a.reconcilePackages(ctx, state, diffData.Packages); err != nil {
			return fmt.Errorf("目录对账失败: %w", err)
		}
		if err := a.reconcilePackageDocs(ctx, state, diffData.Packages); err != nil {
			return fmt.Errorf("同步默认文档失败: %w", err)
		}
	}

	if err := a.syncAddedAPIs(ctx, state, diffData.Add); err != nil {
		return err
	}
	if err := a.syncUpdatedAPIs(ctx, state, diffData.Update); err != nil {
		return err
	}
	scheduledAPIs := append([]*dto.ApiInfo{}, diffData.Add...)
	scheduledAPIs = append(scheduledAPIs, diffData.Update...)
	if err := a.reconcileFormSchedules(ctx, state, scheduledAPIs); err != nil {
		return fmt.Errorf("同步默认定时任务失败: %w", err)
	}
	if err := a.reconcilePackageAgentTasks(ctx, state, diffData.Packages); err != nil {
		return fmt.Errorf("同步默认 Agent 任务失败: %w", err)
	}

	// 处理删除的API
	if len(diffData.Delete) > 0 {
		err := a.deleteFunctionsForAPIs(ctx, state.app, diffData.Delete)
		if err != nil {
			return fmt.Errorf("删除function和service_tree记录失败: %w", err)
		}
	}

	return nil
}

type appMetadataSyncState struct {
	app               *model.App
	currentVersionNum int
	requestUser       string
}

func (a *AppService) loadAppMetadataSyncState(ctx context.Context, appID int64) (*appMetadataSyncState, error) {
	app, err := a.appRepo.GetAppByIDContext(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("获取应用信息失败: %w", err)
	}

	return &appMetadataSyncState{
		app:               app,
		currentVersionNum: app.GetVersionNumber(),
		requestUser:       contextx.GetRequestUser(ctx),
	}, nil
}

func (a *AppService) syncAddedAPIs(ctx context.Context, state *appMetadataSyncState, apis []*dto.ApiInfo) error {
	if len(apis) == 0 {
		return nil
	}

	functions, err := a.convertApiInfoToFunctions(state.app.ID, apis, state.requestUser)
	if err != nil {
		return fmt.Errorf("转换新增API失败: %w", err)
	}

	if err := a.functionRepo.CreateFunctions(ctx, functions); err != nil {
		return fmt.Errorf("创建function记录失败: %w", err)
	}

	if err := a.createServiceTreesForAPIs(ctx, state, apis, functions); err != nil {
		return fmt.Errorf("创建service_tree记录失败: %w", err)
	}
	if err := a.syncSensitiveFieldsForAPIs(ctx, state, apis, functions); err != nil {
		return fmt.Errorf("同步敏感字段失败: %w", err)
	}

	return nil
}

func (a *AppService) syncUpdatedAPIs(ctx context.Context, state *appMetadataSyncState, apis []*dto.ApiInfo) error {
	if len(apis) == 0 {
		return nil
	}

	functions, err := a.convertApiInfoToFunctions(state.app.ID, apis, state.requestUser)
	if err != nil {
		return fmt.Errorf("转换更新API失败: %w", err)
	}

	if err := a.updateFunctionsForAPIs(ctx, state.app.ID, apis, functions); err != nil {
		return fmt.Errorf("更新function记录失败: %w", err)
	}

	if err := a.updateServiceTreesForAPIs(ctx, state, apis, functions); err != nil {
		return fmt.Errorf("更新service_tree记录失败: %w", err)
	}
	if err := a.syncSensitiveFieldsForAPIs(ctx, state, apis, functions); err != nil {
		return fmt.Errorf("同步敏感字段失败: %w", err)
	}

	return nil
}

func (a *AppService) syncSensitiveFieldsForAPIs(ctx context.Context, state *appMetadataSyncState, apis []*dto.ApiInfo, functions []*model.Function) error {
	if a.sensitiveFields == nil || len(apis) == 0 {
		return nil
	}
	for i, api := range apis {
		if api == nil || api.Schema == nil {
			continue
		}
		var functionID int64
		if i < len(functions) && functions[i] != nil {
			functionID = functions[i].ID
		}
		if err := a.sensitiveFields.SyncFunctionSchema(ctx, state.app.User, state.app.Code, api.BuildFullCodePath(), functionID, api.Schema); err != nil {
			return fmt.Errorf("同步 %s 敏感字段失败: %w", api.BuildFullCodePath(), err)
		}
	}
	return nil
}

// convertApiInfoToFunctions 将ApiInfo转换为Function模型
func (a *AppService) convertApiInfoToFunctions(appID int64, apis []*dto.ApiInfo, username string) ([]*model.Function, error) {
	functions := make([]*model.Function, len(apis))

	for i, api := range apis {
		schemaJSON, err := functionschema.Marshal(api.Schema)
		if err != nil {
			return nil, fmt.Errorf("序列化function schema失败: %w", err)
		}
		if api.TemplateType != "" && api.Schema != nil && api.TemplateType != api.Schema.Type {
			return nil, fmt.Errorf("template_type 与 schema.type 不一致: template_type=%s schema.type=%s", api.TemplateType, api.Schema.Type)
		}

		endpoints := normalizeConnectorEndpoints(api.ConnectorEndpoints)
		connectors := normalizeConnectorCodes(append(append([]string{}, api.Connectors...), connectorCodesFromEndpoints(endpoints)...))
		function := &model.Function{
			AppID:              appID,
			Method:             api.Method,
			Router:             api.BuildFullCodePath(),
			Schema:             schemaJSON,
			HasConfig:          false, // 预留字段，默认为false
			Connectors:         joinConnectorCodes(connectors),
			ConnectorEndpoints: joinConnectorEndpoints(endpoints),
			TemplateType:       api.TemplateType,
		}
		// 设置创建者用户名（通过嵌入的 Base 结构体）
		function.CreatedBy = username
		if api.CreateTables != nil {
			function.CreateTables = strings.Join(api.CreateTables, ",")
		}

		functions[i] = function
	}

	return functions, nil
}

// createServiceTreesForAPIs 为新增的API创建ServiceTree记录
func (a *AppService) createServiceTreesForAPIs(ctx context.Context, state *appMetadataSyncState, apis []*dto.ApiInfo, functions []*model.Function) error {
	parentNodes, err := a.loadParentPackageNodes(ctx, apis)
	if err != nil {
		return err
	}

	for i, api := range apis {
		var parentAdmins string
		parentPath := api.GetParentFullCodePath()

		if parentPath != "" {
			parent, exists := parentNodes[parentPath]
			if !exists {
				return fmt.Errorf("父级package节点不存在: %s（目录对账可能未覆盖此路径）", parentPath)
			}
			parentAdmins = parent.Admins
		}

		treeID, err := a.createFunctionNode(ctx, state, api, functions[i].ID, parentAdmins)
		if err != nil {
			return fmt.Errorf("创建function节点失败: %w", err)
		}
		api.TreeID = treeID
	}
	return nil
}

// reconcilePackages 目录对账：拿 SDK 返回的全量 package 列表与数据库比对，缺失的统一创建
// SDK 每次 update 都会返回当前应用注册的全量 package（已按深度从浅到深排序），
// 所以按顺序遍历即可保证父节点先于子节点被创建
func (a *AppService) reconcilePackages(ctx context.Context, state *appMetadataSyncState, packages []*dto.PackageInfo) error {
	// 1. 提取所有 fullPath，批量查库看哪些已存在
	allPaths := make([]string, 0, len(packages)+1)
	for _, pkg := range packages {
		allPaths = append(allPaths, pkg.FullPath)
	}

	rootPath := fmt.Sprintf("/%s/%s", state.app.User, state.app.Code)
	allPaths = append(allPaths, rootPath)

	existingNodes, err := a.serviceTreeRepo.GetServiceTreeByFullPaths(ctx, allPaths)
	if err != nil {
		return fmt.Errorf("批量查询package节点失败: %w", err)
	}

	// 2. 找出缺失的 package
	var missing []*dto.PackageInfo
	for _, pkg := range packages {
		if _, exists := existingNodes[pkg.FullPath]; !exists {
			missing = append(missing, pkg)
		}
	}

	if len(missing) == 0 {
		logger.Infof(ctx, "[reconcilePackages] 目录对账完成: %d 个 package 全部存在，无需创建", len(packages))
		return nil
	}

	logger.Infof(ctx, "[reconcilePackages] 目录对账: 共 %d 个 package，其中 %d 个缺失，开始创建", len(packages), len(missing))

	// 3. 按顺序创建缺失的 package（SDK 已按深度排序，父在前子在后）
	for _, pkg := range missing {
		parts := strings.Split(strings.Trim(pkg.FullPath, "/"), "/")
		// parts 至少 3 段：user/app/pkg，parentPath = /user/app（即根节点）
		if len(parts) < 3 {
			logger.Warnf(ctx, "[reconcilePackages] 跳过非法路径: %s (深度不足)", pkg.FullPath)
			continue
		}
		parentPath := "/" + strings.Join(parts[:len(parts)-1], "/")
		_, ok := existingNodes[parentPath]
		if !ok {
			return fmt.Errorf("[reconcilePackages] 父节点不存在: %s (package=%s)，请检查根节点或上级目录是否已创建", parentPath, pkg.FullPath)
		}

		node := &model.ServiceTree{
			AppID:            state.app.ID,
			Type:             model.ServiceTreeTypePackage,
			Code:             pkg.Code,
			Name:             pkg.Name,
			Description:      pkg.Desc,
			FullCodePath:     pkg.FullPath,
			AddVersionNum:    state.currentVersionNum,
			UpdateVersionNum: 0,
		}
		if state.requestUser != "" {
			node.CreatedBy = state.requestUser
			node.Admins = state.requestUser
		}

		if err := a.serviceTreeRepo.Create(ctx, node); err != nil {
			return fmt.Errorf("创建 package 节点失败 (%s): %w", pkg.FullPath, err)
		}

		existingNodes[pkg.FullPath] = node
		logger.Infof(ctx, "[reconcilePackages] 创建 package: %s (code=%s, name=%s, parentPath=%s)", pkg.FullPath, pkg.Code, pkg.Name, parentPath)
	}

	logger.Infof(ctx, "[reconcilePackages] 目录对账完成: 成功创建 %d 个缺失的 package 节点", len(missing))
	return nil
}

func (a *AppService) loadParentPackageNodes(ctx context.Context, apis []*dto.ApiInfo) (map[string]*model.ServiceTree, error) {
	parentPaths := make(map[string]bool)
	for _, api := range apis {
		parentPath := api.GetParentFullCodePath()
		if parentPath != "" {
			parentPaths[parentPath] = true
		}
	}

	parentPathList := make([]string, 0, len(parentPaths))
	for path := range parentPaths {
		parentPathList = append(parentPathList, path)
	}

	parentNodes, err := a.serviceTreeRepo.GetServiceTreeByFullPaths(ctx, parentPathList)
	if err != nil {
		return nil, fmt.Errorf("批量查询父级package节点失败: %w", err)
	}

	for path, node := range parentNodes {
		if !node.IsPackage() {
			return nil, fmt.Errorf("路径 %s 已存在，但类型不是package，当前类型: %s", path, node.Type)
		}
	}

	return parentNodes, nil
}

// createFunctionNode 创建function节点，返回创建的TreeID
func (a *AppService) createFunctionNode(
	ctx context.Context,
	state *appMetadataSyncState,
	api *dto.ApiInfo,
	functionID int64,
	parentAdmins string,
) (int64, error) {
	// 检查是否已存在（full_name_path全局唯一）
	existingNode, err := a.serviceTreeRepo.GetServiceTreeByFullPath(ctx, api.FullCodePath)
	if err == nil {
		a.applyFunctionNodeMetadata(existingNode, api, functionID)

		// 如果节点是新增的（AddVersionNum为0），设置添加版本号
		if existingNode.AddVersionNum == 0 {
			existingNode.AddVersionNum = state.currentVersionNum
		} else {
			// 如果节点已存在，更新更新版本号
			existingNode.UpdateVersionNum = state.currentVersionNum
		}

		// 更新节点信息
		err = a.serviceTreeRepo.UpdateServiceTree(ctx, existingNode)
		if err != nil {
			return 0, err
		}
		// 返回已存在的节点ID
		return existingNode.ID, nil
	}
	// 如果是记录不存在的错误，这是正常的，继续创建
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, fmt.Errorf("查询路径失败: %w", err)
	}
	// err是gorm.ErrRecordNotFound，说明路径不存在，可以继续创建

	// 获取创建者用户名
	requestUser := state.requestUser

	// 确定 Admins 字段：优先使用父节点的 Admins，如果没有父节点或父节点没有 Admins，则使用当前用户
	admins := parentAdmins
	if admins == "" && requestUser != "" {
		admins = requestUser
	}

	// 创建新的function节点
	serviceTree := &model.ServiceTree{
		AppID:            state.app.ID,
		Type:             model.ServiceTreeTypeFunction,
		FullCodePath:     api.FullCodePath,        // 直接使用api.FullCodePath，不需要重新计算
		AddVersionNum:    state.currentVersionNum, // 设置添加版本号
		UpdateVersionNum: 0,                       // 新增节点，更新版本号为0
		Admins:           admins,                  // 设置管理员列表（从父节点继承，或使用当前用户）
	}
	a.applyFunctionNodeMetadata(serviceTree, api, functionID)

	// 设置创建者
	if requestUser != "" {
		serviceTree.CreatedBy = requestUser
	}

	// 创建ServiceTree节点（GORM Create会自动填充ID）
	err = a.serviceTreeRepo.CreateServiceTreeWithParentPath(ctx, serviceTree, "")
	if err != nil {
		return 0, err
	}

	// 返回创建的节点ID
	return serviceTree.ID, nil
}

func (a *AppService) applyFunctionNodeMetadata(tree *model.ServiceTree, api *dto.ApiInfo, functionID int64) {
	tree.Code = api.Code
	tree.Name = api.Name
	tree.Description = api.Desc
	tree.TemplateType = api.TemplateType
	tree.RefID = functionID
	tree.Tags = strings.Join(api.Tags, ",")
	endpoints := normalizeConnectorEndpoints(api.ConnectorEndpoints)
	tree.Connectors = joinConnectorCodes(append(append([]string{}, api.Connectors...), connectorCodesFromEndpoints(endpoints)...))
	tree.ConnectorEndpoints = joinConnectorEndpoints(endpoints)
}

// updateFunctionsForAPIs 更新API对应的Function记录
func (a *AppService) updateFunctionsForAPIs(ctx context.Context, appID int64, apis []*dto.ApiInfo, functions []*model.Function) error {
	// 对于每个要更新的API，先查找现有的Function记录获取ID
	for i, api := range apis {
		router := api.BuildFullCodePath()
		existingFunction, err := a.functionRepo.GetFunctionByKey(ctx, appID, api.Method, router)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Function不存在，创建新的（这种情况不应该发生，但为了容错处理）
				newFunctions := []*model.Function{functions[i]}
				err = a.functionRepo.CreateFunctions(ctx, newFunctions)
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
	return a.functionRepo.UpdateFunctions(ctx, functions)
}

// updateServiceTreesForAPIs 更新API对应的ServiceTree记录
func (a *AppService) updateServiceTreesForAPIs(ctx context.Context, state *appMetadataSyncState, apis []*dto.ApiInfo, functions []*model.Function) error {
	parentNodes, err := a.loadParentPackageNodes(ctx, apis)
	if err != nil {
		return err
	}

	// 更新function节点
	for i, api := range apis {
		// 根据FullCodePath查找现有的ServiceTree
		existingTree, err := a.serviceTreeRepo.GetServiceTreeByFullPath(ctx, api.FullCodePath)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 如果不存在，创建新的节点（这种情况不应该发生，但为了容错处理）
				var parentAdmins string
				parentPath := api.GetParentFullCodePath()
				if parentPath != "" {
					parent, exists := parentNodes[parentPath]
					if exists {
						parentAdmins = parent.Admins
					}
				}
				treeID, err := a.createFunctionNode(ctx, state, api, functions[i].ID, parentAdmins)
				if err != nil {
					return fmt.Errorf("创建function节点失败: %w", err)
				}
				// 赋值TreeID，方便后续写快照时入库
				api.TreeID = treeID
				continue
			}
			return fmt.Errorf("查询service_tree失败: %w", err)
		}

		// 更新节点信息并设置更新版本号
		a.applyFunctionNodeMetadata(existingTree, api, functions[i].ID)
		// 更新版本号：如果AddVersionNum为0，说明是新增的，设置为当前版本；否则更新UpdateVersionNum
		if existingTree.AddVersionNum == 0 {
			existingTree.AddVersionNum = state.currentVersionNum
		} else {
			existingTree.UpdateVersionNum = state.currentVersionNum
		}

		// 保存更新后的节点
		if err := a.serviceTreeRepo.UpdateServiceTree(ctx, existingTree); err != nil {
			return fmt.Errorf("更新service_tree节点失败: %w", err)
		}
		// 赋值TreeID，方便后续写快照时入库
		api.TreeID = existingTree.ID
	}
	return nil
}

// deleteFunctionsForAPIs 删除API对应的Function和ServiceTree记录
func (a *AppService) deleteFunctionsForAPIs(ctx context.Context, app *model.App, apis []*dto.ApiInfo) error {
	// 收集需要删除的router和method
	routers := make([]string, 0, len(apis))
	methods := make([]string, 0, len(apis))

	for _, api := range apis {
		// 根据FullCodePath查找ServiceTree
		serviceTree, err := a.serviceTreeRepo.GetServiceTreeByFullPath(ctx, api.FullCodePath)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// ServiceTree不存在，跳过
				continue
			}
			return fmt.Errorf("查询service_tree失败: %w", err)
		}

		// 删除ServiceTree（会级联删除子节点）
		err = a.serviceTreeRepo.DeleteServiceTree(ctx, serviceTree.ID)
		if err != nil {
			return fmt.Errorf("删除service_tree失败: %w", err)
		}

		// 收集Function的router和method用于删除
		router := api.BuildFullCodePath()
		routers = append(routers, router)
		methods = append(methods, api.Method)
		if a.sensitiveFields != nil {
			if err := a.sensitiveFields.DeleteFunction(ctx, app.User, app.Code, router); err != nil {
				return fmt.Errorf("删除function敏感字段失败: %w", err)
			}
		}
	}

	// 批量删除Function记录
	if len(routers) > 0 {
		err := a.functionRepo.DeleteFunctions(ctx, app.ID, routers, methods)
		if err != nil {
			return fmt.Errorf("删除function记录失败: %w", err)
		}
	}

	return nil
}
