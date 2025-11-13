package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	agentModel "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/model"
	"gorm.io/gorm"
)

type AppService struct {
	appRuntime      *AppRuntime
	userRepo        *repository.UserRepository
	appRepo         *repository.AppRepository
	functionRepo    *repository.FunctionRepository
	serviceTreeRepo *repository.ServiceTreeRepository
}

// NewAppService 创建 AppService（依赖注入）
func NewAppService(appRuntime *AppRuntime, userRepo *repository.UserRepository, appRepo *repository.AppRepository, functionRepo *repository.FunctionRepository, serviceTreeRepo *repository.ServiceTreeRepository) *AppService {
	return &AppService{
		appRuntime:      appRuntime,
		userRepo:        userRepo,
		appRepo:         appRepo,
		functionRepo:    functionRepo,
		serviceTreeRepo: serviceTreeRepo,
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
func (a *AppService) processAPIDiff(ctx context.Context, appID int64, diffData *agentModel.DiffData) error {
	// 处理新增的API
	if len(diffData.Add) > 0 {
		// 1. 先创建Function记录获取ID
		functions, err := a.convertApiInfoToFunctions(appID, diffData.Add)
		if err != nil {
			return fmt.Errorf("转换新增API失败: %w", err)
		}

		// 保存Function记录获取ID
		err = a.functionRepo.CreateFunctions(functions)
		if err != nil {
			return fmt.Errorf("创建function记录失败: %w", err)
		}

		// 2. 创建ServiceTree记录，使用Function的ID作为RefID
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

		// 2. 更新Function记录
		err = a.updateFunctionsForAPIs(ctx, appID, diffData.Update, functions)
		if err != nil {
			return fmt.Errorf("更新function记录失败: %w", err)
		}

		// 3. 更新ServiceTree记录
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
func (a *AppService) convertApiInfoToFunctions(appID int64, apis []*agentModel.ApiInfo) ([]*model.Function, error) {
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
			Router:       fmt.Sprintf("/%s/%s/%s", api.User, api.App, strings.Trim(api.Router, "/")),
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
func (a *AppService) createServiceTreesForAPIs(ctx context.Context, appID int64, apis []*agentModel.ApiInfo, functions []*model.Function) error {
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
func (a *AppService) createFunctionNode(appID int64, parentID int64, api *agentModel.ApiInfo, functionID int64) error {
	// 检查是否已存在（full_name_path全局唯一）
	_, err := a.serviceTreeRepo.GetServiceTreeByFullPath(api.FullCodePath)
	if err == nil {
		// 如果路径已存在，直接抛错误，不做隐式更新
		return fmt.Errorf("路径 %s 已存在，不能重复创建function节点", api.FullCodePath)
	}
	// 如果是记录不存在的错误，这是正常的，继续创建
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询路径失败: %w", err)
	}
	// err是gorm.ErrRecordNotFound，说明路径不存在，可以继续创建

	// 创建新的function节点，预加载完整的app对象
	serviceTree := &model.ServiceTree{
		AppID:        appID,
		ParentID:     parentID,
		GroupCode:    api.FunctionGroupCode,
		GroupName:    api.FunctionGroupName,
		Type:         model.ServiceTreeTypeFunction,
		Code:         api.Code, // API的code作为ServiceTree的code
		Name:         api.Name, // API的name作为ServiceTree的name
		Description:  api.Desc,
		RefID:        functionID,       // 指向Function记录的ID
		FullCodePath: api.FullCodePath, // 直接使用api.FullCodePath，不需要重新计算
	}

	if len(api.Tags) > 0 {
		serviceTree.Tags = strings.Join(api.Tags, ",")
	}

	// 创建ServiceTree节点
	return a.serviceTreeRepo.CreateServiceTreeWithParentPath(serviceTree, "")
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

	// 从数据库获取用户的分页应用列表
	apps, totalCount, err := a.appRepo.GetAppsByUserWithPage(req.User, page, pageSize)
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
