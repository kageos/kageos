package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/appcall"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"gorm.io/gorm"
)

type AppService struct {
	appCall                    *appcall.Client
	appRepo                    *repository.AppRepository
	functionRepo               *repository.FunctionRepository
	serviceTreeRepo            *repository.ServiceTreeRepository
	operateLogRepo             *repository.OperateLogRepository
	fileSnapshotRepo           *repository.FileSnapshotRepository
	directoryUpdateHistoryRepo *repository.DirectoryUpdateHistoryRepository
}

// NewAppService 创建 AppService（依赖注入）
func NewAppService(appCall *appcall.Client, appRepo *repository.AppRepository, functionRepo *repository.FunctionRepository, serviceTreeRepo *repository.ServiceTreeRepository, operateLogRepo *repository.OperateLogRepository, fileSnapshotRepo *repository.FileSnapshotRepository, directoryUpdateHistoryRepo *repository.DirectoryUpdateHistoryRepository) *AppService {
	return &AppService{
		appCall:                    appCall,
		appRepo:                    appRepo,
		functionRepo:               functionRepo,
		serviceTreeRepo:            serviceTreeRepo,
		operateLogRepo:             operateLogRepo,
		fileSnapshotRepo:           fileSnapshotRepo,
		directoryUpdateHistoryRepo: directoryUpdateHistoryRepo,
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

	// ⭐ 检查应用数量限制（全局限制）
	appCount, err := a.appRepo.CountApps()
	if err != nil {
		logger.Warnf(ctx, "[AppService] Failed to count apps: %v", err)
	} else {
		licenseMgr := license.GetManager()
		if err := licenseMgr.CheckAppLimit(int(appCount)); err != nil {
			return nil, err
		}
	}

	// 验证用户是否存在（通过 hr-server 接口验证）
	// ⭐ 使用服务间调用验证用户，不再直接访问 user 表
	// 获取用户信息（直接传 ctx，内部会提取 token、trace_id 等）
	_, err = apicall.GetUserByUsername(ctx, &dto.QueryUserReq{Username: tenantUser})
	if err != nil {
		return nil, fmt.Errorf("租户用户 %s 不存在: %w", tenantUser, err)
	}

	// 创建前校验：同一用户下应用中文名称是否重复
	if exists, err := a.appRepo.ExistsAppNameForUser(tenantUser, req.Name); err != nil {
		return nil, fmt.Errorf("检查应用名称唯一性失败: %w", err)
	} else if exists {
		return nil, fmt.Errorf("应用名称已存在: %s", req.Name)
	}

	// 分配可用的 Host（选择 app_count 最小的 host，实现负载均衡）
	hostRepo := repository.NewHostRepository(a.appRepo.GetDB())
	hosts, err := hostRepo.GetHostList()
	if err != nil || len(hosts) == 0 {
		return nil, fmt.Errorf("无法获取可用的主机: %w", err)
	}

	// 选择 app_count 最小的 host（负载均衡）
	var selectedHost *model.Host
	for _, host := range hosts {
		if host.Status == "enabled" {
			if selectedHost == nil || host.AppCount < selectedHost.AppCount {
				selectedHost = host
			}
		}
	}
	if selectedHost == nil {
		return nil, fmt.Errorf("没有可用的主机")
	}

	// 创建包含用户信息的请求对象（内部使用）
	resp, err := a.appCall.CreateApp(ctx, selectedHost.ID, req)
	if err != nil {
		return nil, err
	}

	// 写入数据库记录
	isPublic := true // 默认公开
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}
	app := model.App{
		Base: models.Base{
			CreatedBy: requestUser, // 记录实际请求用户（谁发起的请求）
		},
		Version:  "v1",
		Code:     req.Code,
		Name:     req.Name,   // 应用名称
		User:     tenantUser, // 记录租户用户（应用所有者）
		NatsID:   selectedHost.NatsID,
		HostID:   selectedHost.ID,
		Status:   "enabled",
		IsPublic: isPublic,   // 是否公开
		Admins:   req.Admins, // 管理员列表，逗号分隔的用户名
	}
	err = a.appRepo.CreateApp(&app)
	if err != nil {
		return nil, err
	}

	// ⭐ 创建 service_tree 根节点（新架构）
	// 每个 app 都在 service_tree 表中有对应的根节点
	rootNode := &model.ServiceTree{
		Name:         app.Name,
		Code:         app.Code,
		ParentID:     0,  // 根节点
		Type:         model.ServiceTreeTypePackage,  // 统一为 package 类型
		Admins:       app.Admins,
		PendingCount: 0,
		AppID:        app.ID,
		RefID:        app.ID,  // ⭐ ref_id 指向 app 表，标识这是根节点
		FullCodePath: fmt.Sprintf("/%s/%s", tenantUser, req.Code),
		Version:      "v1",
		VersionNum:   1,
		Base: models.Base{
			CreatedBy: requestUser,
			UpdatedBy: requestUser,
		},
	}

	err = a.serviceTreeRepo.Create(rootNode)
	if err != nil {
		logger.Errorf(ctx, "[AppService] 创建 service_tree 根节点失败: app_id=%d, error=%v", app.ID, err)
		// ⚠️ 根节点创建失败会导致服务树无法显示，应该返回错误
		// TODO: 将根节点创建失败改为阻塞性错误，并回滚应用创建
		return nil, fmt.Errorf("创建工作空间根节点失败: %w", err)
	}
	
	logger.Infof(ctx, "[AppService] 创建 service_tree 根节点成功: app_id=%d, root_id=%d, full_code_path=%s", 
		app.ID, rootNode.ID, rootNode.FullCodePath)

	// ⭐ 自动给创建者和管理员分配应用管理员角色（拥有 app:admin 权限）
	resourcePath := fmt.Sprintf("/%s/%s", tenantUser, req.Code)

	// 1. 给创建者分配应用管理员角色
	if err := a.assignAppAdminRoleToUser(ctx, tenantUser, req.Code, tenantUser, resourcePath); err != nil {
		// 权限添加失败不应该影响应用创建，只记录警告日志
		logger.Warnf(ctx, "[AppService] 自动添加创建者应用管理员角色失败: user=%s, app=%s, username=%s, resource=%s, error=%v",
			tenantUser, req.Code, tenantUser, resourcePath, err)
	}

	// 2. 给管理员列表中的用户分配应用管理员角色
	if req.Admins != "" {
		admins := strings.Split(req.Admins, ",")
		for _, admin := range admins {
			admin = strings.TrimSpace(admin)
			if admin != "" && admin != tenantUser { // 避免重复分配（创建者已经在上面分配了）
				if err := a.assignAppAdminRoleToUser(ctx, tenantUser, req.Code, admin, resourcePath); err != nil {
					// 权限添加失败不应该影响应用创建，只记录警告日志
					logger.Warnf(ctx, "[AppService] 自动添加应用管理员角色失败: user=%s, app=%s, username=%s, resource=%s, error=%v",
						tenantUser, req.Code, admin, resourcePath, err)
				}
			}
		}
	}

	return resp, nil
}

// assignAppAdminRoleToUser 给用户分配应用管理员角色
// ⭐ 使用角色系统，分配"admin"角色（拥有 app:admin 权限）
func (a *AppService) assignAppAdminRoleToUser(ctx context.Context, user, app, username, resourcePath string) error {
	// 检查权限功能是否启用（企业版）
	licenseMgr := license.GetManager()
	if !licenseMgr.HasFeature(enterprise.FeaturePermission) {
		// 权限功能未启用，跳过
		return nil
	}

	// 获取权限服务
	permissionService := enterprise.GetPermissionService()
	if permissionService == nil {
		return fmt.Errorf("权限服务未初始化")
	}

	// ⭐ 使用角色系统，分配"admin"角色（拥有 directory:admin 权限）
	// 根目录使用 directory 资源类型（工作空间 = 根目录）
	assignReq := &dto.AssignRoleToUserReq{
		User:         user,
		App:          app,
		Username:     username,
		RoleCode:     "admin", // 管理员角色
		ResourceType: "directory",   // ⭐ 根目录使用 directory 资源类型
		ResourcePath: resourcePath,
		StartTime:    nil, // 永久权限
		EndTime:      nil, // 永久权限
	}

	_, err := permissionService.AssignRoleToUser(ctx, assignReq)
	if err != nil {
		return fmt.Errorf("分配应用管理员角色失败: %w", err)
	}

	logger.Infof(ctx, "[AppService] 分配应用管理员角色成功: user=%s, app=%s, username=%s, resource=%s",
		user, app, username, resourcePath)
	return nil
}

// removeAppAdminRoleFromUser 移除用户的应用管理员角色
func (a *AppService) removeAppAdminRoleFromUser(ctx context.Context, user, app, username, resourcePath string) error {
	// 检查权限功能是否启用（企业版）
	licenseMgr := license.GetManager()
	if !licenseMgr.HasFeature(enterprise.FeaturePermission) {
		// 权限功能未启用，跳过
		return nil
	}

	// 获取权限服务
	permissionService := enterprise.GetPermissionService()
	if permissionService == nil {
		return fmt.Errorf("权限服务未初始化")
	}

	// ⭐ 使用角色系统，移除"admin"角色
	// 根目录使用 directory 资源类型（工作空间 = 根目录）
	removeReq := &dto.RemoveRoleFromUserReq{
		User:         user,
		App:          app,
		Username:     username,
		RoleCode:     "admin", // 管理员角色
		ResourceType: "directory",   // ⭐ 根目录使用 directory 资源类型
		ResourcePath: resourcePath,
	}

	err := permissionService.RemoveRoleFromUser(ctx, removeReq)
	if err != nil {
		return fmt.Errorf("移除应用管理员角色失败: %w", err)
	}

	logger.Infof(ctx, "[AppService] 移除应用管理员角色成功: user=%s, app=%s, username=%s, resource=%s",
		user, app, username, resourcePath)
	return nil
}

// UpdateApp 更新应用（更新应用代码并重新编译部署）
func (a *AppService) UpdateApp(ctx context.Context, req *dto.UpdateAppReq) (*dto.UpdateAppResp, error) {
	// 记录开始时间（用于计算变更耗时）
	startTime := time.Now()

	// 根据应用信息获取 NATS 连接，而不是根据当前用户
	app, err := a.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		return nil, err
	}

	// 调用 app-runtime 更新应用，使用应用所属的 HostID
	resp, err := a.appCall.UpdateApp(ctx, app.HostID, req)
	if err != nil {
		return nil, err
	}

	// 更新数据库中的版本信息
	app.Version = resp.NewVersion
	err = a.appRepo.UpdateApp(app)
	if err != nil {
		return nil, err
	}

	// 计算变更耗时（毫秒）
	duration := time.Since(startTime).Milliseconds()

	// 处理API差异，将API信息入库到function表
	if resp.Diff != nil {
		err = a.processAPIDiff(ctx, app.ID, resp.Diff, req, duration, resp.GitCommitHash)
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
	resp, err := a.appCall.RequestApp(ctx, app.NatsID, req)
	if err != nil {
		return nil, err
	}
	resp.Version = req.Version
	return resp, nil
}

// RecordTableOperateLog 记录 Table 操作日志（OnTableAddRow, OnTableUpdateRow, OnTableDeleteRows）
// 策略：社区版和企业版都记录完整日志，但只有企业版可以查看
func (a *AppService) RecordTableOperateLog(ctx context.Context, req *dto.RecordTableOperateLogReq) error {
	// 获取应用信息（用于获取版本号）
	app, err := a.appRepo.GetAppByUserName(req.TenantUser, req.App)
	if err != nil {
		return fmt.Errorf("获取应用信息失败: %w", err)
	}

	// 直接使用企业版的操作日志记录器
	operateLogger := enterprise.GetOperateLogger()

	// 根据操作类型处理不同的记录逻辑
	switch req.Action {
	case "OnTableUpdateRow":
		// 更新操作：记录 updates 和 old_values
		operateLogReq := &dto.CreateOperateLoggerReq{
			User:       req.RequestUser,
			Action:     req.Action,
			Resource:   "table",
			ResourceID: fmt.Sprintf("%s/%s/%s", req.TenantUser, req.App, strings.TrimPrefix(req.Router, "/")),
			RowID:      req.RowID,
			Updates:    req.Updates,
			OldValues:  req.OldValues,
			Version:    app.Version,
			TraceID:    req.TraceID,
			IPAddress:  req.IPAddress,
			UserAgent:  req.UserAgent,
		}

		// 异步记录操作日志（不阻塞主流程）
		go func() {
			if _, err := operateLogger.CreateOperateLogger(operateLogReq); err != nil {
				logger.Warnf(ctx, "[RecordTableOperateLog] 记录 Table 更新操作日志失败: %v", err)
			}
		}()

	case "OnTableDeleteRows":
		// 删除操作：为每个删除的记录创建一条日志
		for _, rowID := range req.RowIDs {
			operateLogReq := &dto.CreateOperateLoggerReq{
				User:       req.RequestUser,
				Action:     req.Action,
				Resource:   "table",
				ResourceID: fmt.Sprintf("%s/%s/%s", req.TenantUser, req.App, strings.TrimPrefix(req.Router, "/")),
				RowID:      rowID,
				Version:    app.Version,
				TraceID:    req.TraceID,
				IPAddress:  req.IPAddress,
				UserAgent:  req.UserAgent,
			}

			// 异步记录操作日志（不阻塞主流程）
			go func(id int64) {
				if _, err := operateLogger.CreateOperateLogger(operateLogReq); err != nil {
					logger.Warnf(ctx, "[RecordTableOperateLog] 记录 Table 删除操作日志失败: %v", err)
				}
			}(rowID)
		}
	}

	return nil
}

// processAPIDiff 处理API差异，包括新增、更新、删除
func (a *AppService) processAPIDiff(ctx context.Context, appID int64, diffData *dto.DiffData, req *dto.UpdateAppReq, duration int64, gitCommitHash string) error {
	// 校验应用存在
	if _, err := a.appRepo.GetAppByID(appID); err != nil {
		return fmt.Errorf("获取应用信息失败: %w", err)
	}

	// 从 context 获取当前用户名
	username := contextx.GetRequestUser(ctx)

	// 处理新增的API
	if len(diffData.Add) > 0 {
		// 1. 先转换API为Function模型（但不创建）
		functions, err := a.convertApiInfoToFunctions(ctx, appID, diffData.Add, username)
		if err != nil {
			return fmt.Errorf("转换新增API失败: %w", err)
		}

		// 2. 创建Function记录
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
		functions, err := a.convertApiInfoToFunctions(ctx, appID, diffData.Update, username)
		if err != nil {
			return fmt.Errorf("转换更新API失败: %w", err)
		}

		// 2. 更新Function记录
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

	// 5. 已移除：不再写入快照表，发布/推送 Hub 与工作台均从 runtime 实时读目录文件
	// err = a.createDirectorySnapshots(...)

	return nil
}

// convertApiInfoToFunctions 将ApiInfo转换为Function模型
func (a *AppService) convertApiInfoToFunctions(ctx context.Context, appID int64, apis []*dto.ApiInfo, username string) ([]*model.Function, error) {
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

		// 获取父节点的 Admins（如果有父节点）
		var parentAdmins string
		if parentID > 0 {
			if parent, exists := parentNodes[parentPath]; exists {
				parentAdmins = parent.Admins
			}
		}

		// 创建function节点，使用Function的ID作为RefID
		treeID, err := a.createFunctionNode(ctx, appID, parentID, api, functions[i].ID, parentAdmins)
		if err != nil {
			return fmt.Errorf("创建function节点失败: %w", err)
		}
		// 赋值TreeID，方便后续写快照时入库
		api.TreeID = treeID

		// ⭐ 自动给创建者添加函数执行权限（使用角色系统）
		// 资源路径：函数的 FullCodePath，权限：function:admin（所有权）
		// 注意：函数权限分配暂时跳过，因为函数权限应该通过目录权限继承
		// 如果需要单独给函数分配权限，可以后续添加
		requestUser := contextx.GetRequestUser(ctx)
		if requestUser != "" && api.FullCodePath != "" {
			// 从 FullCodePath 解析 user 和 app
			parts := strings.Split(strings.Trim(api.FullCodePath, "/"), "/")
			if len(parts) >= 2 {
				user := parts[0]
				app := parts[1]
				// 暂时跳过函数权限分配，因为函数权限应该通过目录权限继承
				logger.Debugf(ctx, "[AppService] 函数权限通过目录权限继承，跳过单独分配: user=%s, app=%s, resource=%s",
					user, app, api.FullCodePath)
			}
		}
	}
	return nil
}

// createFunctionNode 创建function节点，返回创建的TreeID
func (a *AppService) createFunctionNode(ctx context.Context, appID int64, parentID int64, api *dto.ApiInfo, functionID int64, parentAdmins string) (int64, error) {
	// 检查是否已存在（full_name_path全局唯一）
	existingNode, err := a.serviceTreeRepo.GetServiceTreeByFullPath(api.FullCodePath)
	if err == nil {
		// 如果路径已存在，更新版本号而不是创建新节点
		// 获取应用当前版本
		app, err := a.appRepo.GetAppByID(appID)
		if err != nil {
			return 0, fmt.Errorf("获取应用信息失败: %w", err)
		}

		// 如果节点是新增的（AddVersionNum为0），设置添加版本号
		if existingNode.AddVersionNum == 0 {
			existingNode.AddVersionNum = app.GetVersionNumber()
		} else {
			// 如果节点已存在，更新更新版本号
			existingNode.UpdateVersionNum = app.GetVersionNumber()
		}

		// 更新节点信息
		err = a.serviceTreeRepo.UpdateServiceTree(existingNode)
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

	// 获取应用当前版本
	app, err := a.appRepo.GetAppByID(appID)
	if err != nil {
		return 0, fmt.Errorf("获取应用信息失败: %w", err)
	}

	// 获取创建者用户名
	requestUser := contextx.GetRequestUser(ctx)

	// 确定 Admins 字段：优先使用父节点的 Admins，如果没有父节点或父节点没有 Admins，则使用当前用户
	admins := parentAdmins
	if admins == "" && requestUser != "" {
		admins = requestUser
	}

	// 创建新的function节点，预加载完整的app对象
	serviceTree := &model.ServiceTree{
		AppID:            appID,
		ParentID:         parentID,
		Type:             model.ServiceTreeTypeFunction,
		Code:             api.Code, // API的code作为ServiceTree的code
		Name:             api.Name, // API的name作为ServiceTree的name
		Description:      api.Desc,
		TemplateType:     api.TemplateType,
		RefID:            functionID,             // 指向Function记录的ID
		FullCodePath:     api.FullCodePath,       // 直接使用api.FullCodePath，不需要重新计算
		AddVersionNum:    app.GetVersionNumber(), // 设置添加版本号
		UpdateVersionNum: 0,                      // 新增节点，更新版本号为0
		Admins:           admins,                 // 设置管理员列表（从父节点继承，或使用当前用户）
	}

	// 设置创建者
	if requestUser != "" {
		serviceTree.CreatedBy = requestUser
	}

	if len(api.Tags) > 0 {
		serviceTree.Tags = strings.Join(api.Tags, ",")
	}

	// 创建ServiceTree节点（GORM Create会自动填充ID）
	err = a.serviceTreeRepo.CreateServiceTreeWithParentPath(serviceTree, "")
	if err != nil {
		return 0, err
	}

	// ⭐ 自动给创建者添加函数所有权权限
	// 资源路径：函数的 FullCodePath，权限：function:manage（所有权）
	// 注意：createFunctionNode 方法没有 ctx 参数，需要从调用方传入
	// 权限授予在 createServiceTreesForAPIs 中进行

	// 返回创建的节点ID
	return serviceTree.ID, nil
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
				var parentAdmins string
				parentPath := api.GetParentFullCodePath()
				if parentPath != "" {
					parent, exists := parentNodes[parentPath]
					if exists {
						parentID = parent.ID
						parentAdmins = parent.Admins
					}
				}
				treeID, err := a.createFunctionNode(ctx, appID, parentID, api, functions[i].ID, parentAdmins)
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
		existingTree.RefID = functions[i].ID
		existingTree.Name = api.Name
		existingTree.Description = api.Desc
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
		// 赋值TreeID，方便后续写快照时入库
		api.TreeID = existingTree.ID
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

// DeleteApp 删除应用
func (a *AppService) DeleteApp(ctx context.Context, req *dto.DeleteAppReq) (*dto.DeleteAppResp, error) {
	// 根据应用信息获取 NATS 连接
	app, err := a.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		return nil, err
	}

	// 调用 app-runtime 删除应用
	resp, err := a.appCall.DeleteApp(ctx, app.HostID, req)
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

	// 从数据库获取应用列表（支持搜索和过滤）
	apps, totalCount, err := a.appRepo.GetAppsWithPage(req.User, page, pageSize, req.Search, req.IncludeAll, req.Type)
	if err != nil {
		return nil, fmt.Errorf("获取应用列表失败: %w", err)
	}

	// 转换为 AppInfo 列表
	appInfos := make([]*dto.AppInfo, len(apps))
	for i, app := range apps {
		appInfos[i] = &dto.AppInfo{
			ID:        app.ID,
			User:      app.User,
			Code:      app.Code,
			Name:      app.Name,
			Status:    app.Status,
			Version:   app.Version,
			NatsID:    app.NatsID,
			HostID:    app.HostID,
			IsPublic:  app.IsPublic,
			Admins:    app.Admins,
			Type:      int(app.Type),
			CreatedAt: time.Time(app.CreatedAt).Format("2006-01-02 15:04:05"),
			UpdatedAt: time.Time(app.UpdatedAt).Format("2006-01-02 15:04:05"),
		}
	}

	return &dto.GetAppsResp{
		PageInfoResp: dto.PageInfoResp{
			Page:       page,
			PageSize:   pageSize,
			TotalCount: int(totalCount),
			Items:      appInfos,
		},
	}, nil
}

// GetAppDetail 获取应用详情
func (a *AppService) GetAppDetail(ctx context.Context, req *dto.GetAppDetailReq) (*dto.GetAppDetailResp, error) {
	// 从数据库获取应用信息
	app, err := a.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("应用不存在: %s/%s", req.User, req.App)
		}
		return nil, fmt.Errorf("获取应用详情失败: %w", err)
	}

	// 转换为响应格式
	return &dto.GetAppDetailResp{
		AppInfo: dto.AppInfo{
			ID:        app.ID,
			User:      app.User,
			Code:      app.Code,
			Name:      app.Name,
			Status:    app.Status,
			Version:   app.Version,
			NatsID:    app.NatsID,
			HostID:    app.HostID,
			IsPublic:  app.IsPublic,
			Admins:    app.Admins,
			CreatedAt: time.Time(app.CreatedAt).Format("2006-01-02 15:04:05"),
			UpdatedAt: time.Time(app.UpdatedAt).Format("2006-01-02 15:04:05"),
		},
	}, nil
}

// GetAppByUserName 根据用户名和应用名获取应用信息
func (a *AppService) GetAppByUserName(ctx context.Context, user, app string) (*model.App, error) {
	return a.appRepo.GetAppByUserName(user, app)
}

// UpdateWorkspace 更新工作空间（只更新 MySQL 记录，不涉及容器更新）
func (a *AppService) UpdateWorkspace(ctx context.Context, req *dto.UpdateWorkspaceReq) (*dto.UpdateWorkspaceResp, error) {
	// 获取应用信息
	app, err := a.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		return nil, fmt.Errorf("获取应用信息失败: %w", err)
	}

	// ⭐ 更新管理员列表并同步更新角色分配
	oldAdminsStr := app.Admins
	newAdminsStr := req.Admins

	// 解析旧管理员列表
	oldAdmins := make(map[string]bool)
	if oldAdminsStr != "" {
		for _, admin := range strings.Split(oldAdminsStr, ",") {
			admin = strings.TrimSpace(admin)
			if admin != "" {
				oldAdmins[admin] = true
			}
		}
	}

	// 解析新管理员列表
	newAdmins := make(map[string]bool)
	if newAdminsStr != "" {
		for _, admin := range strings.Split(newAdminsStr, ",") {
			admin = strings.TrimSpace(admin)
			if admin != "" {
				newAdmins[admin] = true
			}
		}
	}

	// 更新数据库中的管理员列表
	app.Admins = req.Admins
	if err := a.appRepo.UpdateApp(app); err != nil {
		return nil, fmt.Errorf("更新工作空间失败: %w", err)
	}

	// ⭐ 同步更新角色分配
	resourcePath := fmt.Sprintf("/%s/%s", req.User, req.App)

	// 1. 移除不再担任管理员的用户角色
	for oldAdmin := range oldAdmins {
		if !newAdmins[oldAdmin] {
			// 该用户不再是管理员，移除其应用管理员角色
			if err := a.removeAppAdminRoleFromUser(ctx, req.User, req.App, oldAdmin, resourcePath); err != nil {
				// 角色移除失败不应该影响更新，只记录警告日志
				logger.Warnf(ctx, "[AppService] 移除应用管理员角色失败: resource=%s, username=%s, error=%v",
					resourcePath, oldAdmin, err)
			}
		}
	}

	// 2. 给新管理员分配角色
	for newAdmin := range newAdmins {
		if !oldAdmins[newAdmin] {
			// 该用户是新管理员，分配应用管理员角色
			if err := a.assignAppAdminRoleToUser(ctx, req.User, req.App, newAdmin, resourcePath); err != nil {
				// 角色分配失败不应该影响更新，只记录警告日志
				logger.Warnf(ctx, "[AppService] 分配应用管理员角色失败: resource=%s, username=%s, error=%v",
					resourcePath, newAdmin, err)
			}
		}
	}

	logger.Infof(ctx, "[AppService] 更新工作空间成功: user=%s, app=%s, admins=%s", req.User, req.App, req.Admins)

	return &dto.UpdateWorkspaceResp{
		User:   req.User,
		App:    req.App,
		Admins: req.Admins,
	}, nil
}

// GetFunctionByFullCodePath 根据 full-code-path 获取函数信息
// fullCodePath 参数应该是完整的路径（如 /luobei/operations/tools/pdftools/to_images）
// full-code-path 是全局唯一的，不需要 method 参数
func (a *AppService) GetFunctionByFullCodePath(ctx context.Context, fullCodePath string) (*model.Function, error) {
	function, err := a.functionRepo.GetFunctionByFullCodePath(fullCodePath)
	if err != nil {
		return nil, fmt.Errorf("获取函数信息失败: %w", err)
	}

	return function, nil
}

// createDirectorySnapshots 创建目录快照（检测目录变更并创建快照）
func (a *AppService) createDirectorySnapshots(ctx context.Context, appID int64, app *model.App, diffData *dto.DiffData, req *dto.UpdateAppReq, duration int64, gitCommitHash string) error {
	// 构建 summary：优先使用 Summary，如果没有则组合 Requirement 和 ChangeDescription
	summary := req.Summary
	if summary == "" {
		if req.Requirement != "" && req.ChangeDescription != "" {
			summary = fmt.Sprintf("需求：%s\n\n变更描述：%s", req.Requirement, req.ChangeDescription)
		} else if req.Requirement != "" {
			summary = req.Requirement
		} else if req.ChangeDescription != "" {
			summary = req.ChangeDescription
		}
	}
	// 1. 按目录分组变更
	directoryChanges := a.groupChangesByDirectory(diffData)
	if len(directoryChanges) == 0 {
		logger.Infof(ctx, "[createDirectorySnapshots] 没有目录变更，跳过快照创建")
		return nil
	}

	currentAppVersion := app.Version
	currentAppVersionNum := extractVersionNum(currentAppVersion)

	// 2. 为每个有变更的目录创建快照
	for directoryPath, changes := range directoryChanges {
		logger.Infof(ctx, "[createDirectorySnapshots] 检测到目录变更: path=%s, add=%d, update=%d, delete=%d",
			directoryPath, len(changes.Add), len(changes.Update), len(changes.Delete))

		// 获取目录节点（ServiceTree）
		serviceTree, err := a.serviceTreeRepo.GetServiceTreeByFullPath(directoryPath)
		var currentVersionNum int
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// 如果目录节点不存在，从 v1 开始（这种情况不应该发生，因为目录应该已经存在）
				logger.Warnf(ctx, "[createDirectorySnapshots] 目录节点不存在: path=%s，从 v1 开始", directoryPath)
				currentVersionNum = 1
			} else {
				logger.Warnf(ctx, "[createDirectorySnapshots] 获取目录节点失败: path=%s, error=%v", directoryPath, err)
				continue
			}
		} else {
			// 从 ServiceTree 获取当前版本
			if serviceTree.VersionNum > 0 {
				currentVersionNum = serviceTree.VersionNum
			} else {
				// 如果版本为0，从 v1 开始
				currentVersionNum = 1
			}
		}

		// 计算下一个版本
		nextVersionNum := currentVersionNum + 1
		nextVersion := fmt.Sprintf("v%d", nextVersionNum)

		// 读取目录下所有文件的代码（从文件系统读取，用于创建快照）
		files, err := a.readDirectoryFilesFromFS(ctx, app.User, app.Code, directoryPath)
		if err != nil {
			logger.Warnf(ctx, "[createDirectorySnapshots] 读取目录文件失败: path=%s, error=%v", directoryPath, err)
			continue
		}

		if len(files) == 0 {
			logger.Warnf(ctx, "[createDirectorySnapshots] 目录下没有文件，跳过快照创建: path=%s", directoryPath)
			continue
		}

		// 批量获取所有文件的最新快照（用于变更检测）
		fileNames := make([]string, 0, len(files))
		fileNameMap := make(map[string]*directoryFile) // fileName -> file
		for _, file := range files {
			// 从相对路径提取文件名（最后一个 / 之后的部分）
			fileNameFromPath := file.RelativePath
			if lastSlash := strings.LastIndex(file.RelativePath, "/"); lastSlash >= 0 {
				fileNameFromPath = file.RelativePath[lastSlash+1:]
			}
			// 优先使用 FileName，如果没有则从路径提取
			fileName := file.FileName
			if fileName == "" {
				fileName = strings.TrimSuffix(fileNameFromPath, ".go")
			}
			fileNames = append(fileNames, fileName)
			fileNameMap[fileName] = file
		}

		// 批量获取文件最新快照
		latestSnapshots, err := a.fileSnapshotRepo.GetLatestFileSnapshots(appID, directoryPath, fileNames)
		if err != nil {
			logger.Warnf(ctx, "[createDirectorySnapshots] 获取文件最新快照失败: path=%s, error=%v", directoryPath, err)
			// 如果获取失败，继续处理，所有文件都当作新文件处理
			latestSnapshots = make(map[string]*model.FileSnapshot)
		}

		// 构建文件快照列表（每个文件一行记录）
		fileSnapshots := make([]*model.FileSnapshot, 0, len(files))

		for _, file := range files {
			// 从相对路径提取文件名（最后一个 / 之后的部分）
			fileNameFromPath := file.RelativePath
			if lastSlash := strings.LastIndex(file.RelativePath, "/"); lastSlash >= 0 {
				fileNameFromPath = file.RelativePath[lastSlash+1:]
			}
			// 优先使用 FileName，如果没有则从路径提取
			fileName := file.FileName
			if fileName == "" {
				fileName = strings.TrimSuffix(fileNameFromPath, ".go")
			}

			// 判断文件类型
			fileType := "go"
			if strings.HasSuffix(file.RelativePath, ".go") {
				fileType = "go"
			} else if strings.HasSuffix(file.RelativePath, ".json") {
				fileType = "json"
			} else if strings.HasSuffix(file.RelativePath, ".yaml") || strings.HasSuffix(file.RelativePath, ".yml") {
				fileType = "yaml"
			} else if strings.HasSuffix(file.RelativePath, ".md") {
				fileType = "markdown"
			}

			// 获取文件最新快照，判断文件是否变更
			latestSnapshot := latestSnapshots[fileName]
			var fileVersionNum int
			var fileVersion string

			if latestSnapshot == nil {
				// 新文件，文件版本从 v1 开始
				fileVersionNum = 1
				fileVersion = "v1"
				logger.Infof(ctx, "[createDirectorySnapshots] 检测到新文件: path=%s, file=%s", directoryPath, fileName)
			} else {
				// TODO: 优化内容比较策略
				// 当前使用直接字符串比较，后续可以考虑：
				// 1. 使用内容哈希（MD5/SHA256）比较，提高性能和准确性
				// 2. 使用 diff 算法，记录变更类型和位置
				// 3. 忽略空白字符和换行符的差异
				// 比较文件内容，判断是否变更
				if latestSnapshot.Content != file.Content {
					// 内容变更，文件版本+1
					fileVersionNum = latestSnapshot.FileVersionNum + 1
					fileVersion = fmt.Sprintf("v%d", fileVersionNum)
					logger.Infof(ctx, "[createDirectorySnapshots] 检测到文件变更: path=%s, file=%s, oldVersion=%s, newVersion=%s",
						directoryPath, fileName, latestSnapshot.FileVersion, fileVersion)
				} else {
					// 内容未变更，文件版本不变
					fileVersionNum = latestSnapshot.FileVersionNum
					fileVersion = latestSnapshot.FileVersion
					logger.Infof(ctx, "[createDirectorySnapshots] 文件未变更: path=%s, file=%s, version=%s",
						directoryPath, fileName, fileVersion)
				}
			}

			// 计算文件行数
			lineCount := calculateLineCount(file.Content)
			
			// 创建文件快照（所有文件都创建新快照，记录新的目录版本）
			fileSnapshot := &model.FileSnapshot{
				AppID:          appID,
				ServiceTreeID:  0, // 默认值，如果 serviceTree 存在则赋值
				FullCodePath:   directoryPath,
				FileName:       fileName,
				RelativePath:   file.RelativePath,
				Content:        file.Content,
				DirVersion:     nextVersion,
				DirVersionNum:  nextVersionNum,
				FileVersion:    fileVersion,
				FileVersionNum: fileVersionNum,
				AppVersion:     currentAppVersion,
				AppVersionNum:  currentAppVersionNum,
				FileType:       fileType,
				IsCurrent:      true, // 新快照标记为当前版本
				LineCount:      lineCount,
			}

			// 如果目录节点存在，赋值 ServiceTreeID（方便后续查询和构建目录树）
			if serviceTree != nil {
				fileSnapshot.ServiceTreeID = serviceTree.ID
			}

			fileSnapshots = append(fileSnapshots, fileSnapshot)
		}

		// 批量创建文件快照
		err = a.fileSnapshotRepo.CreateBatch(fileSnapshots)
		if err != nil {
			logger.Warnf(ctx, "[createDirectorySnapshots] 创建文件快照失败: path=%s, error=%v", directoryPath, err)
			continue
		}

		// 批量更新旧快照的 IsCurrent 状态为 false
		// 收集需要更新的旧快照ID（只更新那些 IsCurrent = true 的旧快照）
		oldSnapshotIDs := make([]int64, 0)
		for _, file := range files {
			fileName := file.FileName
			if fileName == "" {
				// 从相对路径提取文件名
				fileNameFromPath := file.RelativePath
				if lastSlash := strings.LastIndex(file.RelativePath, "/"); lastSlash >= 0 {
					fileNameFromPath = file.RelativePath[lastSlash+1:]
				}
				fileName = strings.TrimSuffix(fileNameFromPath, ".go")
			}

			latestSnapshot := latestSnapshots[fileName]
			if latestSnapshot != nil && latestSnapshot.IsCurrent {
				oldSnapshotIDs = append(oldSnapshotIDs, latestSnapshot.ID)
			}
		}

		// 批量更新旧快照的 IsCurrent 状态
		if len(oldSnapshotIDs) > 0 {
			err = a.fileSnapshotRepo.BatchUpdateIsCurrent(oldSnapshotIDs, false)
			if err != nil {
				logger.Warnf(ctx, "[createDirectorySnapshots] 批量更新旧快照 IsCurrent 状态失败: path=%s, count=%d, error=%v", directoryPath, len(oldSnapshotIDs), err)
				// 不中断流程，继续处理
			} else {
				logger.Infof(ctx, "[createDirectorySnapshots] 批量更新旧快照 IsCurrent 状态成功: path=%s, count=%d", directoryPath, len(oldSnapshotIDs))
			}
		}

		// 更新 ServiceTree 的版本
		if serviceTree != nil {
			serviceTree.Version = nextVersion
			serviceTree.VersionNum = nextVersionNum
			err = a.serviceTreeRepo.UpdateServiceTree(serviceTree)
			if err != nil {
				logger.Warnf(ctx, "[createDirectorySnapshots] 更新节点版本失败: path=%s, error=%v", directoryPath, err)
				continue
			}
		} else {
			logger.Warnf(ctx, "[createDirectorySnapshots] 目录节点不存在，无法更新版本: path=%s", directoryPath)
		}

		logger.Infof(ctx, "[createDirectorySnapshots] 目录快照创建成功: path=%s, version=%s, fileCount=%d",
			directoryPath, nextVersion, len(files))

		// 🔥 新增：记录目录变更历史
		err = a.recordDirectoryUpdateHistory(ctx, appID, app, directoryPath, nextVersion, nextVersionNum, changes, req.Requirement, req.ChangeDescription, summary, duration, gitCommitHash)
		if err != nil {
			// 历史记录失败不应该影响主流程，记录日志即可
			logger.Warnf(ctx, "[createDirectorySnapshots] 记录目录变更历史失败: path=%s, error=%v", directoryPath, err)
		}
	}

	return nil
}

// DirectoryChanges 目录变更信息
type DirectoryChanges struct {
	Add    []*dto.ApiInfo
	Update []*dto.ApiInfo
	Delete []*dto.ApiInfo
}

// groupChangesByDirectory 按目录分组变更
func (a *AppService) groupChangesByDirectory(diffData *dto.DiffData) map[string]*DirectoryChanges {
	directoryChanges := make(map[string]*DirectoryChanges)

	// 处理新增的API
	for _, api := range diffData.Add {
		dirPath := api.GetParentFullCodePath()
		if dirPath == "" {
			// 如果无法获取目录路径，跳过
			continue
		}
		if directoryChanges[dirPath] == nil {
			directoryChanges[dirPath] = &DirectoryChanges{
				Add:    []*dto.ApiInfo{},
				Update: []*dto.ApiInfo{},
				Delete: []*dto.ApiInfo{},
			}
		}
		directoryChanges[dirPath].Add = append(directoryChanges[dirPath].Add, api)
	}

	// 处理更新的API
	for _, api := range diffData.Update {
		dirPath := api.GetParentFullCodePath()
		if dirPath == "" {
			continue
		}
		if directoryChanges[dirPath] == nil {
			directoryChanges[dirPath] = &DirectoryChanges{
				Add:    []*dto.ApiInfo{},
				Update: []*dto.ApiInfo{},
				Delete: []*dto.ApiInfo{},
			}
		}
		directoryChanges[dirPath].Update = append(directoryChanges[dirPath].Update, api)
	}

	// 处理删除的API
	for _, api := range diffData.Delete {
		dirPath := api.GetParentFullCodePath()
		if dirPath == "" {
			continue
		}
		if directoryChanges[dirPath] == nil {
			directoryChanges[dirPath] = &DirectoryChanges{
				Add:    []*dto.ApiInfo{},
				Update: []*dto.ApiInfo{},
				Delete: []*dto.ApiInfo{},
			}
		}
		directoryChanges[dirPath].Delete = append(directoryChanges[dirPath].Delete, api)
	}

	return directoryChanges
}

// recordDirectoryUpdateHistory 记录目录更新历史
func (a *AppService) recordDirectoryUpdateHistory(
	ctx context.Context,
	appID int64,
	app *model.App,
	directoryPath string,
	dirVersion string,
	dirVersionNum int,
	changes *DirectoryChanges,
	requirement string,
	changeDescription string,
	summary string,
	duration int64,
	gitCommitHash string,
) error {
	// 构建API摘要列表（直接使用 ApiInfo 中的 TemplateType）
	addedSummaries := make([]*model.ApiSummary, 0, len(changes.Add))
	for _, api := range changes.Add {
		addedSummaries = append(addedSummaries, &model.ApiSummary{
			Code:         api.Code, // 使用 API 的 Code 而不是 FunctionGroupCode
			Name:         api.Name,
			Desc:         api.Desc,
			Router:       api.Router,
			Method:       api.Method,
			FullCodePath: api.BuildFullCodePath(),
			TemplateType: api.TemplateType, // 直接使用 ApiInfo 中的 TemplateType
		})
	}

	updatedSummaries := make([]*model.ApiSummary, 0, len(changes.Update))
	for _, api := range changes.Update {
		updatedSummaries = append(updatedSummaries, &model.ApiSummary{
			Code:         api.Code, // 使用 API 的 Code 而不是 FunctionGroupCode
			Name:         api.Name,
			Desc:         api.Desc,
			Router:       api.Router,
			Method:       api.Method,
			FullCodePath: api.BuildFullCodePath(),
			TemplateType: api.TemplateType, // 直接使用 ApiInfo 中的 TemplateType
		})
	}

	deletedSummaries := make([]*model.ApiSummary, 0, len(changes.Delete))
	for _, api := range changes.Delete {
		deletedSummaries = append(deletedSummaries, &model.ApiSummary{
			Code:         api.Code, // 使用 API 的 Code 而不是 FunctionGroupCode
			Name:         api.Name,
			Desc:         api.Desc,
			Router:       api.Router,
			Method:       api.Method,
			FullCodePath: api.BuildFullCodePath(),
			TemplateType: api.TemplateType, // 直接使用 ApiInfo 中的 TemplateType
		})
	}

	// 序列化JSON（使用 json.RawMessage，GORM 会自动处理）
	addedJSON, _ := json.Marshal(addedSummaries)
	updatedJSON, _ := json.Marshal(updatedSummaries)
	deletedJSON, _ := json.Marshal(deletedSummaries)

	// 获取当前用户
	updatedBy := contextx.GetRequestUser(ctx)
	if updatedBy == "" {
		updatedBy = "system"
	}

	// 创建历史记录
	history := &model.DirectoryUpdateHistory{
		AppID:             appID,
		AppVersion:        app.Version,
		AppVersionNum:     extractVersionNum(app.Version),
		FullCodePath:      directoryPath,
		DirVersion:        dirVersion,
		DirVersionNum:     dirVersionNum,
		AddedAPIs:         addedJSON,   // json.RawMessage，GORM 会自动处理
		UpdatedAPIs:       updatedJSON, // json.RawMessage，GORM 会自动处理
		DeletedAPIs:       deletedJSON, // json.RawMessage，GORM 会自动处理
		AddedCount:        len(changes.Add),
		UpdatedCount:      len(changes.Update),
		DeletedCount:      len(changes.Delete),
		Summary:           summary,           // 变更摘要（详情），可能是大模型返回的摘要信息，也可能是用户的变更需求
		Requirement:       requirement,       // 变更需求（用户在前端输入的）
		ChangeDescription: changeDescription, // 变更描述（大模型输出的）
		Duration:          duration,          // 变更耗时（毫秒）
		GitCommitHash:     gitCommitHash,     // Git 提交哈希（用于回滚）
		UpdatedBy:         updatedBy,
	}

	return a.directoryUpdateHistoryRepo.CreateUpdateHistory(history)
}

// directoryFile 目录文件结构（用于创建快照，内部使用）
type directoryFile struct {
	FileName     string
	RelativePath string
	Content      string
}

// readDirectoryFilesFromFS 从 app-runtime 读取目录下的所有文件（用于创建快照）
// 通过 NATS 调用 app-runtime 的接口，而不是直接访问文件系统
func (a *AppService) readDirectoryFilesFromFS(ctx context.Context, user, app, fullCodePath string) ([]*directoryFile, error) {
	// 获取应用信息（用于获取 HostID）
	appModel, err := a.appRepo.GetAppByUserName(user, app)
	if err != nil {
		return nil, fmt.Errorf("获取应用信息失败: %w", err)
	}

	// 构建请求
	req := &dto.ReadDirectoryFilesRuntimeReq{
		User:          user,
		App:           app,
		DirectoryPath: fullCodePath,
	}

	// 通过 NATS 调用 app-runtime 读取目录文件
	resp, err := a.appCall.ReadDirectoryFiles(ctx, appModel.HostID, req)
	if err != nil {
		return nil, fmt.Errorf("读取目录文件失败: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("读取目录文件失败: %s", resp.Message)
	}

	// 转换为内部格式
	files := make([]*directoryFile, 0, len(resp.Files))
	for _, file := range resp.Files {
		files = append(files, &directoryFile{
			FileName:     file.FileName,
			RelativePath: file.RelativePath,
			Content:      file.Content,
		})
	}

	logger.Infof(ctx, "[readDirectoryFilesFromFS] 通过 NATS 读取目录文件成功: path=%s, fileCount=%d", fullCodePath, len(files))
	return files, nil
}

// calculateLineCount 计算文件内容的总行数
func calculateLineCount(content string) int {
	if content == "" {
		return 0
	}
	lines := strings.Split(content, "\n")
	lineCount := len(lines)
	// 如果最后一行是空行（文件末尾有换行符），不计入总行数
	if lineCount > 0 && lines[lineCount-1] == "" {
		lineCount--
	}
	return lineCount
}
