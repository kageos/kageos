package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/codegen/metadata"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/permission"
)

// extractVersionNum 从版本号字符串中提取数字部分（如 "v1" -> 1, "v20" -> 20）
func extractVersionNumForServiceTree(version string) int {
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

// assignAdminRoleToUser 给用户分配管理员角色（目录节点）
// ⭐ 使用角色系统，分配"admin"角色（拥有 directory:manage 权限）
func (s *ServiceTreeService) assignAdminRoleToUser(ctx context.Context, user, app, username, resourcePath string) error {
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

	// ⭐ 使用角色系统，分配"admin"角色（拥有 directory:manage 权限）
	// 目录节点使用 directory 资源类型
	assignReq := &dto.AssignRoleToUserReq{
		User:         user,
		App:          app,
		Username:     username,
		RoleCode:     "admin",     // 管理员角色
		ResourceType: "directory", // ⭐ 目录节点使用 directory 资源类型
		ResourcePath: resourcePath,
		StartTime:    nil, // 永久权限
		EndTime:      nil, // 永久权限
	}

	_, err := permissionService.AssignRoleToUser(ctx, assignReq)
	if err != nil {
		return fmt.Errorf("分配管理员角色失败: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] 分配管理员角色成功: user=%s, app=%s, username=%s, resource=%s",
		user, app, username, resourcePath)
	return nil
}

// removeAdminRoleFromUser 移除用户的管理员角色（目录节点）
// ⭐ 从 resourcePath 自动解析 user 和 app
func (s *ServiceTreeService) removeAdminRoleFromUser(ctx context.Context, resourcePath, username string) error {
	// 从 resourcePath 解析 user 和 app
	parts := strings.Split(strings.Trim(resourcePath, "/"), "/")
	if len(parts) < 2 {
		return fmt.Errorf("无法从资源路径解析 user 和 app: %s", resourcePath)
	}
	user := parts[0]
	app := parts[1]

	return s.removeAdminRoleFromUserWithUserApp(ctx, user, app, username, resourcePath)
}

// removeAdminRoleFromUserWithUserApp 移除用户的管理员角色（目录节点，带 user 和 app 参数）
func (s *ServiceTreeService) removeAdminRoleFromUserWithUserApp(ctx context.Context, user, app, username, resourcePath string) error {
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
	// 目录节点使用 directory 资源类型
	removeReq := &dto.RemoveRoleFromUserReq{
		User:         user,
		App:          app,
		Username:     username,
		RoleCode:     "admin",     // 管理员角色
		ResourceType: "directory", // ⭐ 目录节点使用 directory 资源类型
		ResourcePath: resourcePath,
	}

	err := permissionService.RemoveRoleFromUser(ctx, removeReq)
	if err != nil {
		return fmt.Errorf("移除管理员角色失败: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] 移除管理员角色成功: user=%s, app=%s, username=%s, resource=%s",
		user, app, username, resourcePath)
	return nil
}

type ServiceTreeService struct {
	serviceTreeRepo   *repository.ServiceTreeRepository
	appRepo           *repository.AppRepository
	appRuntime        *AppRuntime
	fileSnapshotRepo  *repository.FileSnapshotRepository
	appService        *AppService
	permissionService *PermissionService // ⭐ 添加 PermissionService 依赖，用于查询权限
}

// NewServiceTreeService 创建服务目录服务
func NewServiceTreeService(
	serviceTreeRepo *repository.ServiceTreeRepository,
	appRepo *repository.AppRepository,
	appRuntime *AppRuntime,
	fileSnapshotRepo *repository.FileSnapshotRepository,
	appService *AppService,
	permissionService *PermissionService, // ⭐ 新增 PermissionService 依赖
) *ServiceTreeService {
	return &ServiceTreeService{
		serviceTreeRepo:   serviceTreeRepo,
		appRepo:           appRepo,
		appRuntime:        appRuntime,
		fileSnapshotRepo:  fileSnapshotRepo,
		appService:        appService,
		permissionService: permissionService,
	}
}

// CreateServiceTree 创建服务目录
func (s *ServiceTreeService) CreateServiceTree(ctx context.Context, req *dto.CreateServiceTreeReq) (*dto.CreateServiceTreeResp, error) {
	// 获取应用信息
	app, err := s.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	var parentTree *model.ServiceTree

	if req.ParentID != 0 {
		// 检查名称是否已存在
		parentTree, err = s.serviceTreeRepo.GetByID(req.ParentID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("failed to check name exists: %s", err)
		}
	}

	fullCodePath := fmt.Sprintf("/%s/%s/%s", app.User, app.Code, req.Code)
	if parentTree != nil {
		fullCodePath = parentTree.FullCodePath + "/" + req.Code
	}
	exists, err := s.serviceTreeRepo.CheckNameExists(req.ParentID, req.Code, app.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check name exists: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("directory %s already exists", req.Code)
	}

	// 提取当前版本号数字
	currentVersionNum := extractVersionNumForServiceTree(app.Version)

	// 获取创建者用户名
	requestUser := contextx.GetRequestUser(ctx)

	// 创建服务目录记录
	serviceTree := &model.ServiceTree{
		Name:             req.Name,
		Code:             req.Code,
		ParentID:         req.ParentID,
		Type:             model.ServiceTreeTypePackage,
		Description:      req.Description,
		Tags:             req.Tags,
		Admins:           req.Admins, // 设置管理员列表
		AppID:            app.ID,
		FullCodePath:     fullCodePath,
		AddVersionNum:    currentVersionNum, // 设置添加版本号
		UpdateVersionNum: 0,                 // 新增节点，更新版本号为0
	}

	// 设置创建者
	if requestUser != "" {
		serviceTree.CreatedBy = requestUser
	}

	// 保存到数据库
	if err := s.serviceTreeRepo.CreateServiceTreeWithParentPath(serviceTree, ""); err != nil {
		return nil, fmt.Errorf("failed to create service tree: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] Created service tree: %s/%s/%s", req.User, req.App, req.Code)

	// ⭐ 自动给创建者和管理员分配管理员角色（拥有 directory:manage 权限）
	// 1. 给创建者分配管理员角色
	if requestUser != "" {
		if err := s.assignAdminRoleToUser(ctx, req.User, req.App, requestUser, serviceTree.FullCodePath); err != nil {
			// 权限添加失败不应该影响目录创建，只记录警告日志
			logger.Warnf(ctx, "[ServiceTreeService] 自动添加创建者管理员角色失败: user=%s, app=%s, username=%s, resource=%s, error=%v",
				req.User, req.App, requestUser, serviceTree.FullCodePath, err)
		}
	}

	// 2. 给管理员列表中的用户分配管理员角色
	if req.Admins != "" {
		admins := strings.Split(req.Admins, ",")
		for _, admin := range admins {
			admin = strings.TrimSpace(admin)
			if admin != "" && admin != requestUser { // 避免重复分配（创建者已经在上面分配了）
				if err := s.assignAdminRoleToUser(ctx, req.User, req.App, admin, serviceTree.FullCodePath); err != nil {
					// 权限添加失败不应该影响目录创建，只记录警告日志
					logger.Warnf(ctx, "[ServiceTreeService] 自动添加管理员角色失败: user=%s, app=%s, username=%s, resource=%s, error=%v",
						req.User, req.App, admin, serviceTree.FullCodePath, err)
				}
			}
		}
	}

	// 发送NATS消息给app-runtime创建目录结构
	if err := s.sendCreateServiceTreeMessage(ctx, req.User, req.App, serviceTree); err != nil {
		logger.Warnf(ctx, "[ServiceTreeService] Failed to send NATS message: %v", err)
		// 不返回错误，因为数据库记录已创建成功
	}

	// 返回响应
	resp := &dto.CreateServiceTreeResp{
		ID:           serviceTree.ID,
		Name:         serviceTree.Name,
		Code:         serviceTree.Code,
		ParentID:     serviceTree.ParentID,
		Type:         serviceTree.Type,
		Description:  serviceTree.Description,
		Tags:         serviceTree.Tags,
		AppID:        serviceTree.AppID,
		FullCodePath: serviceTree.FullCodePath,
		Version:      serviceTree.Version,
		VersionNum:   serviceTree.VersionNum,
		Status:       "created",
	}

	return resp, nil
}

// getServiceTreeByAppModel 根据 appModel 获取服务目录树（内部方法，避免重复获取 appModel）
// ⭐ 优化：在服务树中直接返回权限信息，一次性获取所有权限（只需要8ms）
func (s *ServiceTreeService) getServiceTreeByAppModel(ctx context.Context, appModel *model.App, nodeType string) ([]*dto.GetServiceTreeResp, error) {
	// 构建树形结构（如果指定了类型，则只返回该类型的节点）
	var trees []*model.ServiceTree
	var err error
	if nodeType != "" {
		trees, err = s.serviceTreeRepo.BuildServiceTreeByType(appModel.ID, nodeType)
	} else {
		trees, err = s.serviceTreeRepo.BuildServiceTree(appModel.ID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to build service tree: %w", err)
	}

	// ⭐ 查询权限（如果权限功能启用且 PermissionService 可用）
	// ⭐ 使用 GetWorkspacePermissions 获取原始权限记录（只需要8ms），然后自己实现权限继承
	var permissionsMap map[string]map[string]bool // resourcePath -> action -> bool
	var isAdmin bool                              // ⭐ 是否是管理员
	licenseMgr := license.GetManager()
	username := contextx.GetRequestUser(ctx)

	if licenseMgr.HasFeature(enterprise.FeaturePermission) && username != "" && appModel.ID > 0 && s.permissionService != nil {
		// ⭐ 优先检查：如果当前用户是工作空间管理员，设置 isAdmin = true
		if username != "" && appModel.Admins != "" {
			adminList := strings.Split(appModel.Admins, ",")
			for _, admin := range adminList {
				admin = strings.TrimSpace(admin)
				if admin == username {
					isAdmin = true
					logger.Debugf(ctx, "[ServiceTreeService] 用户 %s 是工作空间管理员，设置 isAdmin=true", username)
					break
				}
			}
		}

		// 直接计算权限（不使用缓存）
		permsMap, err := s.calculatePermissions(ctx, appModel.User, appModel.Code, trees, appModel.Admins, username)
		if err != nil {
			logger.Warnf(ctx, "[ServiceTreeService] 计算权限失败: app_id=%d, error=%v，继续返回服务树（无权限信息）", appModel.ID, err)
			// 权限计算失败不影响服务树返回，只是没有权限信息
		} else {
			permissionsMap = permsMap
			logger.Debugf(ctx, "[ServiceTreeService] 权限计算完成: app_id=%d, username=%s, isAdmin=%v", appModel.ID, username, isAdmin)
		}
	}

	// 转换为响应格式，并合并权限信息
	var resp []*dto.GetServiceTreeResp
	for _, tree := range trees {
		resp = append(resp, s.convertToGetServiceTreeResp(ctx, tree, permissionsMap, isAdmin))
	}

	return resp, nil
}

// GetServiceTree 获取服务目录
func (s *ServiceTreeService) GetServiceTree(ctx context.Context, user, app string, nodeType string) ([]*dto.GetServiceTreeResp, error) {
	// 获取应用信息
	appModel, err := s.appRepo.GetAppByUserName(user, app)
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	return s.getServiceTreeByAppModel(ctx, appModel, nodeType)
}

// GetAppWithServiceTree 获取应用详情和服务目录树（合并接口，减少请求次数）
// 这个方法放在 ServiceTreeService 中，因为：
// 1. ServiceTreeService 已经有 appService 依赖，可以直接调用
// 2. 避免 AppService 和 ServiceTreeService 之间的循环依赖
// 3. 职责清晰：ServiceTreeService 负责服务目录树相关的所有操作，包括组合操作
// 优化：只获取一次 appModel，避免重复查询数据库
func (s *ServiceTreeService) GetAppWithServiceTree(ctx context.Context, req *dto.GetAppWithServiceTreeReq) (*dto.GetAppWithServiceTreeResp, error) {
	// 获取应用信息（只获取一次，避免重复调用）
	appModel, err := s.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("应用不存在: %s/%s", req.User, req.App)
		}
		return nil, fmt.Errorf("获取应用信息失败: %w", err)
	}

	// 转换为 AppInfo 响应格式
	appInfo := dto.AppInfo{
		ID:        appModel.ID,
		User:      appModel.User,
		Code:      appModel.Code,
		Name:      appModel.Name,
		Status:    appModel.Status,
		Version:   appModel.Version,
		NatsID:    appModel.NatsID,
		HostID:    appModel.HostID,
		IsPublic:  appModel.IsPublic,
		Admins:    appModel.Admins,
		CreatedAt: time.Time(appModel.CreatedAt).Format("2006-01-02 15:04:05"),
		UpdatedAt: time.Time(appModel.UpdatedAt).Format("2006-01-02 15:04:05"),
	}

	// 使用内部方法获取服务目录树（复用 appModel，避免重复查询）
	serviceTreeResp, err := s.getServiceTreeByAppModel(ctx, appModel, req.Type)
	if err != nil {
		return nil, fmt.Errorf("获取服务目录树失败: %w", err)
	}

	// ⭐ 计算需要自动展开的节点ID列表（包含所有 pending_count > 0 的节点及其父节点）
	expandedKeys := s.calculateExpandedKeys(serviceTreeResp)

	return &dto.GetAppWithServiceTreeResp{
		App:          appInfo,
		ServiceTree:  serviceTreeResp,
		ExpandedKeys: expandedKeys,
	}, nil
}

// GetServiceTreeDetail 获取服务目录详情（包含权限信息）
// ⭐ 优化：按需查询权限，只在获取详情时查询
func (s *ServiceTreeService) GetServiceTreeDetail(ctx context.Context, req *dto.GetServiceTreeDetailReq) (*dto.GetServiceTreeDetailResp, error) {
	var tree *model.ServiceTree
	var err error

	// 优先使用 ID，如果没有则使用 full-code-path
	if req.ID > 0 {
		tree, err = s.serviceTreeRepo.GetServiceTreeByID(req.ID)
	} else if req.FullCodePath != "" {
		tree, err = s.serviceTreeRepo.GetServiceTreeByFullPath(req.FullCodePath)
	} else {
		return nil, fmt.Errorf("必须提供 ID 或 full_code_path")
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("服务目录不存在")
		}
		return nil, fmt.Errorf("获取服务目录失败: %w", err)
	}

	// 转换为响应格式
	resp := &dto.GetServiceTreeDetailResp{
		ID:             tree.ID,
		Name:           tree.Name,
		Code:           tree.Code,
		ParentID:       tree.ParentID,
		Type:           tree.Type,
		Description:    tree.Description,
		Tags:           tree.Tags,
		AppID:          tree.AppID,
		RefID:          tree.RefID,
		FullCodePath:   tree.FullCodePath,
		TemplateType:   tree.TemplateType,
		Version:        tree.Version,
		VersionNum:     tree.VersionNum,
		HubDirectoryID: tree.HubDirectoryID,
		HubVersion:     tree.HubVersion,
		HubVersionNum:  tree.HubVersionNum,
	}

	// ⭐ 查询权限信息（企业版功能）
	// ⭐ 优化：使用 GetWorkspacePermissions + applyPermissionInheritance（与 getServiceTreeByAppModel 保持一致）
	licenseMgr := license.GetManager()
	username := contextx.GetRequestUser(ctx)

	if licenseMgr.HasFeature(enterprise.FeaturePermission) && username != "" && tree.FullCodePath != "" && s.permissionService != nil {
		// ⭐ 从 FullCodePath 解析 user 和 app（格式：/user/app/...）
		_, workspaceUser, workspaceApp := permission.ParseFullCodePath(tree.FullCodePath)
		if workspaceUser == "" || workspaceApp == "" {
			logger.Warnf(ctx, "[ServiceTreeService] 无法从 FullCodePath 解析 user 和 app: fullCodePath=%s", tree.FullCodePath)
			resp.Permissions = make(map[string]bool)
		} else {
			// 1. 调用 GetWorkspacePermissions 获取所有权限记录（只需要8ms）
			permReq := &dto.GetWorkspacePermissionsReq{
				User: workspaceUser,
				App:  workspaceApp,
			}
			permResp, err := s.permissionService.GetWorkspacePermissions(ctx, permReq)
			if err != nil {
				logger.Warnf(ctx, "[ServiceTreeService] 查询权限失败: user=%s, app=%s, error=%v，继续返回详情（无权限信息）", workspaceUser, workspaceApp, err)
				// 权限查询失败不影响详情返回，只是没有权限信息
				resp.Permissions = make(map[string]bool)
			} else if permResp != nil {
				// 2. 将权限记录转换为 Map<resourcePath, Set<action>>（原始权限）
				rawPermissions := make(map[string]map[string]bool) // resourcePath -> action -> true
				for _, record := range permResp.Records {
					resourcePath := record.Resource
					action := record.Action

					if rawPermissions[resourcePath] == nil {
						rawPermissions[resourcePath] = make(map[string]bool)
					}
					rawPermissions[resourcePath][action] = true
				}

				// 3. ⭐ 确定需要查询的权限点（使用新的权限点格式：resource_type:action_type）
				var nodeTypeStr string
				if tree.Type == model.ServiceTreeTypePackage {
					nodeTypeStr = "package"
				} else if tree.Type == model.ServiceTreeTypeFunction {
					nodeTypeStr = "function"
				}

				// ⭐ 获取节点需要的权限点（格式：resource_type:action_type，如 table:read, directory:write）
				actions := permission.GetActionsForNode(nodeTypeStr, tree.TemplateType)
				nodePerms := make(map[string]bool)

				if len(actions) > 0 {
					// 4.1 先设置直接权限（权限点格式：resource_type:action_type）
					if rawPerms, ok := rawPermissions[tree.FullCodePath]; ok {
						for _, actionCode := range actions {
							nodePerms[actionCode] = rawPerms[actionCode]
						}
					} else {
						// 如果没有直接权限，初始化为 false
						for _, actionCode := range actions {
							nodePerms[actionCode] = false
						}
					}

					// 4.2 ⭐ 检查父目录权限继承（权限点格式：resource_type:action_type）
					parentPaths := permission.GetParentPaths(tree.FullCodePath)
					for _, parentPath := range parentPaths {
						// 检查精确路径权限（如 /parent）
						if parentPerms, ok := rawPermissions[parentPath]; ok {
							s.applyPermissionInheritance(nodeTypeStr, tree.TemplateType, parentPerms, nodePerms)
						}
					}

					// 4.3 ⭐ 检查应用级别权限（权限点格式：app:admin）
					// ⭐ 优先检查：如果当前用户是工作空间管理员，直接返回所有权限
					appModel, err := s.appRepo.GetAppByUserName(workspaceUser, workspaceApp)
					if err == nil && appModel != nil && appModel.Admins != "" && username != "" {
						adminList := strings.Split(appModel.Admins, ",")
						for _, admin := range adminList {
							admin = strings.TrimSpace(admin)
							if admin == username {
								// 当前用户是管理员，直接返回所有权限
								logger.Debugf(ctx, "[ServiceTreeService] 用户 %s 是工作空间管理员，直接返回所有权限", username)
								for _, actionCode := range actions {
									nodePerms[actionCode] = true
								}
								// ⭐ 同时将 app:admin 权限添加到节点权限中，方便前端检查
								appAdminCode := permission.BuildActionCode(permission.ResourceTypeApp, "admin")
								nodePerms[appAdminCode] = true
								resp.Permissions = nodePerms
								return resp, nil
							}
						}
					}

					// ⭐ 如果不是管理员，检查应用级别权限（从角色分配表查询）
					appPath := permission.GetAppPath(tree.FullCodePath)
					if appPath != "" {
						// 检查精确路径权限（如 /user/app）
						if appPerms, ok := rawPermissions[appPath]; ok {
							appAdminCode := permission.BuildActionCode(permission.ResourceTypeApp, "admin")
							if appPerms[appAdminCode] {
								for _, actionCode := range actions {
									nodePerms[actionCode] = true
								}
								// ⭐ 同时将 app:admin 权限添加到节点权限中，方便前端检查
								nodePerms[appAdminCode] = true
							}
						}
					}
				}

				resp.Permissions = nodePerms
			} else {
				// 如果没有权限记录，初始化为空 map
				resp.Permissions = make(map[string]bool)
			}
		}
	} else {
		// 如果权限功能未启用或缺少必要信息，初始化为空 map
		resp.Permissions = make(map[string]bool)
	}

	return resp, nil
}

// GetPackageInfo 获取目录信息（仅用于获取目录权限，不包含函数）
// ⭐ 优化：专门用于获取目录权限，函数权限从函数详情接口获取
func (s *ServiceTreeService) GetPackageInfo(ctx context.Context, req *dto.GetPackageInfoReq) (*dto.GetPackageInfoResp, error) {
	var tree *model.ServiceTree
	var err error

	// 优先使用 ID，如果没有则使用 full-code-path
	if req.ID > 0 {
		tree, err = s.serviceTreeRepo.GetServiceTreeByID(req.ID)
	} else if req.FullCodePath != "" {
		tree, err = s.serviceTreeRepo.GetServiceTreeByFullPath(req.FullCodePath)
	} else {
		return nil, fmt.Errorf("必须提供 ID 或 full_code_path")
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("目录不存在")
		}
		return nil, fmt.Errorf("获取目录失败: %w", err)
	}

	// 只处理目录类型（package），函数类型应该使用函数详情接口
	if tree.Type != model.ServiceTreeTypePackage {
		return nil, fmt.Errorf("该接口仅用于获取目录信息，函数信息请使用函数详情接口")
	}

	// 转换为响应格式
	resp := &dto.GetPackageInfoResp{
		ID:           tree.ID,
		Name:         tree.Name,
		Code:         tree.Code,
		FullCodePath: tree.FullCodePath,
	}

	// ⭐ 查询权限信息（企业版功能）
	// ⭐ 优化：使用 GetWorkspacePermissions + applyPermissionInheritance（与 getServiceTreeByAppModel 保持一致）
	licenseMgr := license.GetManager()
	username := contextx.GetRequestUser(ctx)

	if licenseMgr.HasFeature(enterprise.FeaturePermission) && username != "" && tree.FullCodePath != "" && s.permissionService != nil {
		// ⭐ 从 FullCodePath 解析 user 和 app（格式：/user/app/...）
		_, workspaceUser, workspaceApp := permission.ParseFullCodePath(tree.FullCodePath)
		if workspaceUser == "" || workspaceApp == "" {
			logger.Warnf(ctx, "[ServiceTreeService] 无法从 FullCodePath 解析 user 和 app: fullCodePath=%s", tree.FullCodePath)
			resp.Permissions = make(map[string]bool)
		} else {
			// 1. 调用 GetWorkspacePermissions 获取所有权限记录（只需要8ms）
			permReq := &dto.GetWorkspacePermissionsReq{
				User: workspaceUser,
				App:  workspaceApp,
			}
			permResp, err := s.permissionService.GetWorkspacePermissions(ctx, permReq)
			if err != nil {
				logger.Warnf(ctx, "[ServiceTreeService] 查询权限失败: user=%s, app=%s, error=%v，继续返回目录信息（无权限信息）", workspaceUser, workspaceApp, err)
				// 权限查询失败不影响目录信息返回，只是没有权限信息
				resp.Permissions = make(map[string]bool)
			} else if permResp != nil {
				// 2. 将权限记录转换为 Map<resourcePath, Set<action>>（原始权限）
				rawPermissions := make(map[string]map[string]bool) // resourcePath -> action -> true
				for _, record := range permResp.Records {
					resourcePath := record.Resource
					action := record.Action

					if rawPermissions[resourcePath] == nil {
						rawPermissions[resourcePath] = make(map[string]bool)
					}
					rawPermissions[resourcePath][action] = true
				}

				// 3. 确定需要查询的权限点
				actions := permission.GetActionsForNode("package", "")
				nodePerms := make(map[string]bool)

				// 4.1 先设置直接权限
				if rawPerms, ok := rawPermissions[tree.FullCodePath]; ok {
					for _, action := range actions {
						nodePerms[action] = rawPerms[action]
					}
				} else {
					// 如果没有直接权限，初始化为 false
					for _, action := range actions {
						nodePerms[action] = false
					}
				}

				// 4.2 ⭐ 检查父目录权限继承（权限点格式：resource_type:action_type）
				// ⭐ 注意：不再支持通配符路径（parentPath + "/*"），因为角色系统不支持通配符
				parentPaths := permission.GetParentPaths(tree.FullCodePath)
				for _, parentPath := range parentPaths {
					// 检查精确路径权限（如 /parent）
					if parentPerms, ok := rawPermissions[parentPath]; ok {
						s.applyPermissionInheritance("package", "", parentPerms, nodePerms)
					}
				}

				// 4.3 ⭐ 检查应用级别权限（权限点格式：app:admin）
				// ⭐ 优先检查：如果当前用户是工作空间管理员，直接返回所有权限
				appModel, err := s.appRepo.GetAppByUserName(workspaceUser, workspaceApp)
				if err == nil && appModel != nil && appModel.Admins != "" && username != "" {
					adminList := strings.Split(appModel.Admins, ",")
					for _, admin := range adminList {
						admin = strings.TrimSpace(admin)
						if admin == username {
							// 当前用户是管理员，直接返回所有权限
							logger.Debugf(ctx, "[ServiceTreeService] 用户 %s 是工作空间管理员，直接返回所有权限", username)
							for _, actionCode := range actions {
								nodePerms[actionCode] = true
							}
							// ⭐ 同时将 app:admin 权限添加到节点权限中，方便前端检查
							appAdminCode := permission.BuildActionCode(permission.ResourceTypeApp, "admin")
							nodePerms[appAdminCode] = true
							resp.Permissions = nodePerms
							return resp, nil
						}
					}
				}

				// ⭐ 如果不是管理员，检查应用级别权限（从角色分配表查询）
				appPath := permission.GetAppPath(tree.FullCodePath)
				if appPath != "" {
					// 检查精确路径权限（如 /user/app）
					if appPerms, ok := rawPermissions[appPath]; ok {
						appAdminCode := permission.BuildActionCode(permission.ResourceTypeApp, "admin")
						if appPerms[appAdminCode] {
							for _, actionCode := range actions {
								nodePerms[actionCode] = true
							}
							// ⭐ 同时将 app:admin 权限添加到节点权限中，方便前端检查
							nodePerms[appAdminCode] = true
						}
					}
				}

				resp.Permissions = nodePerms
			} else {
				// 如果没有权限记录，初始化为空 map
				resp.Permissions = make(map[string]bool)
			}
		}
	} else {
		// 如果权限功能未启用或缺少必要信息，初始化为空 map
		resp.Permissions = make(map[string]bool)
	}

	return resp, nil
}

// UpdateServiceTree 更新服务目录
func (s *ServiceTreeService) UpdateServiceTreeMetadata(ctx context.Context, req *dto.UpdateServiceTreeMetadataReq) error {
	// 获取服务目录
	serviceTree, err := s.serviceTreeRepo.GetServiceTreeByID(req.ID)
	if err != nil {
		return fmt.Errorf("failed to get service tree: %w", err)
	}

	// 更新字段
	if req.Name != "" {
		serviceTree.Name = req.Name
	}
	if req.Code != "" {
		// 检查新名称是否已存在
		exists, err := s.serviceTreeRepo.CheckNameExists(serviceTree.ParentID, req.Code, serviceTree.AppID)
		if err != nil {
			return fmt.Errorf("failed to check name exists: %w", err)
		}
		if exists {
			return fmt.Errorf("service tree name '%s' already exists in this parent directory", req.Code)
		}
		serviceTree.Code = req.Code
	}
	if req.Description != "" {
		serviceTree.Description = req.Description
	}
	if req.Tags != "" {
		serviceTree.Tags = req.Tags
	}
	// ⭐ 更新管理员列表并同步更新角色分配
	oldAdminsStr := serviceTree.Admins
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
	serviceTree.Admins = req.Admins
	if err := s.serviceTreeRepo.UpdateServiceTree(serviceTree); err != nil {
		return fmt.Errorf("failed to update service tree: %w", err)
	}

	// ⭐ 同步更新角色分配
	// 从 FullCodePath 解析 user 和 app
	parts := strings.Split(strings.Trim(serviceTree.FullCodePath, "/"), "/")
	if len(parts) >= 2 {
		user := parts[0]
		app := parts[1]

		// 1. 移除不再担任管理员的用户角色
		for oldAdmin := range oldAdmins {
			if !newAdmins[oldAdmin] {
				// 该用户不再是管理员，移除其管理员角色
				if err := s.removeAdminRoleFromUserWithUserApp(ctx, user, app, oldAdmin, serviceTree.FullCodePath); err != nil {
					// 角色移除失败不应该影响更新，只记录警告日志
					logger.Warnf(ctx, "[ServiceTreeService] 移除管理员角色失败: resource=%s, username=%s, error=%v",
						serviceTree.FullCodePath, oldAdmin, err)
				}
			}
		}

		// 2. 给新管理员分配角色
		for newAdmin := range newAdmins {
			if !oldAdmins[newAdmin] {
				// 该用户是新管理员，分配管理员角色
				if err := s.assignAdminRoleToUser(ctx, user, app, newAdmin, serviceTree.FullCodePath); err != nil {
					// 角色分配失败不应该影响更新，只记录警告日志
					logger.Warnf(ctx, "[ServiceTreeService] 分配管理员角色失败: resource=%s, username=%s, error=%v",
						serviceTree.FullCodePath, newAdmin, err)
				}
			}
		}
	}

	logger.Infof(ctx, "[ServiceTreeService] Updated service tree: ID=%d", req.ID)
	return nil
}

// DeleteServiceTree 删除服务目录
func (s *ServiceTreeService) DeleteServiceTree(ctx context.Context, id int64) error {
	// 获取服务目录信息（用于日志）
	serviceTree, err := s.serviceTreeRepo.GetServiceTreeByID(id)
	if err != nil {
		return fmt.Errorf("failed to get service tree: %w", err)
	}

	// 删除服务目录（级联删除子目录）
	if err := s.serviceTreeRepo.DeleteServiceTree(id); err != nil {
		return fmt.Errorf("failed to delete service tree: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] Deleted service tree: ID=%d, Code=%s", id, serviceTree.Code)
	return nil
}

// convertToGetServiceTreeResp 转换为响应格式（包含权限信息）
// ⭐ 优化：在服务树中直接返回权限信息，一次性获取所有权限（只需要8ms）
func (s *ServiceTreeService) convertToGetServiceTreeResp(ctx context.Context, tree *model.ServiceTree, permissionsMap map[string]map[string]bool, isAdmin bool) *dto.GetServiceTreeResp {
	resp := &dto.GetServiceTreeResp{
		ID:             tree.ID,
		Name:           tree.Name,
		Code:           tree.Code,
		ParentID:       tree.ParentID,
		RefID:          tree.RefID,
		Type:           tree.Type,
		Description:    tree.Description,
		Tags:           tree.Tags,
		Admins:         tree.Admins,
		PendingCount:   tree.PendingCount, // ⭐ 待审批的权限申请数量
		Owner:          tree.CreatedBy,
		AppID:          tree.AppID,
		FullCodePath:   tree.FullCodePath,
		TemplateType:   tree.TemplateType,
		Version:        tree.Version,
		VersionNum:     tree.VersionNum,
		HubDirectoryID: tree.HubDirectoryID,
		HubVersion:     tree.HubVersion,
		HubVersionNum:  tree.HubVersionNum,
		IsAdmin:        isAdmin, // ⭐ 是否是管理员（前端优先判断此字段）
	}

	// ⭐ 设置权限信息
	// ⭐ 确保所有节点都有权限信息，即使权限功能未启用或查询失败
	if tree.FullCodePath != "" {
		if permissionsMap != nil {
			if nodePerms, ok := permissionsMap[tree.FullCodePath]; ok {
				resp.Permissions = nodePerms
			} else {
				// 如果没有权限信息，初始化为空 map（表示没有权限）
				// ⭐ 注意：即使没有权限，也要返回空 map，而不是 nil，这样前端可以判断
				resp.Permissions = make(map[string]bool)
			}
		} else {
			// ⭐ 如果 permissionsMap 为 nil（权限功能未启用或查询失败），也初始化为空 map
			// ⭐ 确保所有节点都有权限字段，方便前端判断
			resp.Permissions = make(map[string]bool)
		}
	} else {
		// ⭐ 即使 FullCodePath 为空，也初始化权限字段
		resp.Permissions = make(map[string]bool)
	}

	// 递归处理子节点
	if len(tree.Children) > 0 {
		for _, child := range tree.Children {
			childResp := s.convertToGetServiceTreeResp(ctx, child, permissionsMap, isAdmin)
			resp.Children = append(resp.Children, childResp)
		}
	}

	// ⭐ 判断 package 类型节点是否有函数（只检查直接子节点，不递归）
	if tree.Type == model.ServiceTreeTypePackage {
		resp.HasFunction = s.hasFunctionInDirectChildren(tree)
	}

	return resp
}

// calculateExpandedKeys 计算需要自动展开的节点ID列表
// ⭐ 包含所有 pending_count > 0 的节点及其所有父节点
func (s *ServiceTreeService) calculateExpandedKeys(trees []*dto.GetServiceTreeResp) []int64 {
	expandedKeysMap := make(map[int64]bool)

	// 递归查找所有 pending_count > 0 的节点，并收集其路径上的所有节点ID
	var findNodesWithPending func(nodes []*dto.GetServiceTreeResp, parentPath []int64)
	findNodesWithPending = func(nodes []*dto.GetServiceTreeResp, parentPath []int64) {
		for _, node := range nodes {
			currentPath := append(parentPath, node.ID)

			// 如果当前节点有 pending_count > 0，将路径上的所有节点ID添加到展开列表
			if node.PendingCount > 0 {
				for _, id := range currentPath {
					expandedKeysMap[id] = true
				}
			}

			// 递归处理子节点
			if len(node.Children) > 0 {
				findNodesWithPending(node.Children, currentPath)
			}
		}
	}

	findNodesWithPending(trees, []int64{})

	// 转换为切片
	expandedKeys := make([]int64, 0, len(expandedKeysMap))
	for id := range expandedKeysMap {
		expandedKeys = append(expandedKeys, id)
	}

	return expandedKeys
}

// calculatePermissions 计算权限（内部方法）
// ⭐ 优先检查 app.Admins 字段，如果当前用户在管理员列表中，直接返回所有权限
// ⭐ 如果不是管理员，再使用 PermissionCalculator 计算权限，支持角色权限
func (s *ServiceTreeService) calculatePermissions(ctx context.Context, user, app string, trees []*model.ServiceTree, admins string, username string) (map[string]map[string]bool, error) {
	// ⭐ 优先检查：如果当前用户是工作空间管理员，直接返回所有权限
	if username != "" && admins != "" {
		adminList := strings.Split(admins, ",")
		for _, admin := range adminList {
			admin = strings.TrimSpace(admin)
			if admin == username {
				// 当前用户是管理员，直接返回所有权限
				logger.Debugf(ctx, "[ServiceTreeService] 用户 %s 是工作空间管理员，直接返回所有权限", username)
				permissionsMap := make(map[string]map[string]bool)
				appAdminCode := permission.BuildActionCode(permission.ResourceTypeApp, "admin")

				// 递归遍历所有节点，给每个节点设置所有权限
				var setAllPermissions func(nodes []*model.ServiceTree)
				setAllPermissions = func(nodes []*model.ServiceTree) {
					for _, node := range nodes {
						// 获取节点需要的权限点
						actions := permission.GetActionsForNode(node.Type, node.TemplateType)
						nodePerms := make(map[string]bool)

						// 设置所有权限为 true
						for _, actionCode := range actions {
							nodePerms[actionCode] = true
						}
						// ⭐ 同时添加 app:admin 权限，方便前端检查
						nodePerms[appAdminCode] = true

						permissionsMap[node.FullCodePath] = nodePerms

						// 递归处理子节点
						if len(node.Children) > 0 {
							setAllPermissions(node.Children)
						}
					}
				}

				setAllPermissions(trees)
				return permissionsMap, nil
			}
		}
	}

	// ⭐ 如果不是管理员，使用 PermissionCalculator 计算权限（支持角色权限）
	// 通过 enterprise.GetPermissionService() 获取 PermissionCalculator
	enterprisePermService := enterprise.GetPermissionService()
	if enterprisePermService == nil {
		// 如果权限服务未初始化，返回空权限
		logger.Warnf(ctx, "[ServiceTreeService] 权限服务未初始化，返回空权限")
		return make(map[string]map[string]bool), nil
	}

	// ⭐ 方案：使用 GetWorkspacePermissions（已支持角色权限），然后自己实现权限继承
	// 这样可以避免循环依赖，同时支持角色权限
	permReq := &dto.GetWorkspacePermissionsReq{
		User: user,
		App:  app,
	}
	permResp, err := s.permissionService.GetWorkspacePermissions(ctx, permReq)
	if err != nil {
		return nil, fmt.Errorf("查询权限失败: %w", err)
	}

	if permResp == nil || len(permResp.Records) == 0 {
		logger.Debugf(ctx, "[ServiceTreeService] 没有权限记录: user=%s, app=%s", user, app)
		return make(map[string]map[string]bool), nil
	}

	// 将权限记录转换为 Map<resourcePath, Set<action>>（原始权限，已包含角色权限）
	rawPermissions := make(map[string]map[string]bool) // resourcePath -> action -> true
	for _, record := range permResp.Records {
		resourcePath := record.Resource
		action := record.Action

		if rawPermissions[resourcePath] == nil {
			rawPermissions[resourcePath] = make(map[string]bool)
		}
		rawPermissions[resourcePath][action] = true
	}

	// ⭐ 自顶向下计算权限（与 PermissionCalculator 保持一致）
	permissionsMap := make(map[string]map[string]bool)

	// ⭐ 获取应用级别权限（所有节点共享）
	// ⭐ 注意：不再支持通配符路径（/user/app/*），因为角色系统不支持通配符
	appPath := ""
	var appPerms map[string]bool
	if len(trees) > 0 && trees[0].FullCodePath != "" {
		appPath = permission.GetAppPath(trees[0].FullCodePath)
		if appPath != "" {
			// 检查精确路径权限（如 /user/app）
			if perms, ok := rawPermissions[appPath]; ok {
				appPerms = perms
			}

			// ⭐ 检查 app.Admins 字段，如果当前用户在管理员列表中，直接添加 app:admin 权限
			if username != "" && admins != "" {
				adminList := strings.Split(admins, ",")
				for _, admin := range adminList {
					admin = strings.TrimSpace(admin)
					if admin == username {
						// 当前用户在管理员列表中，添加 app:admin 权限
						if appPerms == nil {
							appPerms = make(map[string]bool)
						}
						appAdminCode := permission.BuildActionCode(permission.ResourceTypeApp, "admin")
						appPerms[appAdminCode] = true
						logger.Debugf(ctx, "[ServiceTreeService] 用户 %s 在应用管理员列表中，添加 app:admin 权限", username)
						break
					}
				}
			}
		}
	}

	// 自顶向下递归计算权限
	var calculatePermissionsRecursive func(nodes []*model.ServiceTree, inheritedPerms map[string]bool)
	calculatePermissionsRecursive = func(nodes []*model.ServiceTree, inheritedPerms map[string]bool) {
		for _, node := range nodes {
			// 获取节点需要的权限点
			actions := permission.GetActionsForNode(node.Type, node.TemplateType)
			if len(actions) == 0 {
				// 如果没有需要的权限点，继续处理子节点（传递继承权限）
				if len(node.Children) > 0 {
					calculatePermissionsRecursive(node.Children, inheritedPerms)
				}
				continue
			}

			// 初始化节点权限
			nodePerms := make(map[string]bool)
			for _, action := range actions {
				nodePerms[action] = false
			}

			// 1. 设置直接权限（已包含角色权限）
			if rawPerms, ok := rawPermissions[node.FullCodePath]; ok {
				// ⭐ 根据节点类型和模板类型获取资源类型
				resourceType := permission.GetResourceType(node.Type, node.TemplateType)

				// ⭐ 获取该资源类型可用的权限点列表
				availableActions := permission.GetActionsForResourceType(resourceType)

				// 只设置该资源类型支持的权限点
				for _, action := range actions {
					// 检查权限点是否在该资源类型的可用权限点列表中
					isAvailable := false
					for _, availableAction := range availableActions {
						if action == availableAction {
							isAvailable = true
							break
						}
					}

					if isAvailable && rawPerms[action] {
						nodePerms[action] = true
					}
				}
			}

			// 2. 应用继承权限（从父节点传递下来的）
			if inheritedPerms != nil {
				s.applyPermissionInheritance(node.Type, node.TemplateType, inheritedPerms, nodePerms)
			}

			// 3. ⭐ 检查当前节点的精确路径权限（用于继承给子节点）
			// ⭐ 注意：不再支持通配符路径（parentPath + "/*"），因为角色系统不支持通配符
			currentNodePerms := make(map[string]bool)
			// 精确路径权限
			if rawPerms, ok := rawPermissions[node.FullCodePath]; ok {
				for k, v := range rawPerms {
					currentNodePerms[k] = v
				}
			}

			// 4. ⭐ 应用级别权限（权限点格式：app:admin）
			if appPerms != nil {
				appAdminCode := permission.BuildActionCode(permission.ResourceTypeApp, "admin")
				if appPerms[appAdminCode] {
					for _, actionCode := range actions {
						nodePerms[actionCode] = true
					}
					// ⭐ 同时将 app:admin 权限添加到节点权限中，方便前端检查
					nodePerms[appAdminCode] = true
					// app:admin 也传递给子节点
					currentNodePerms[appAdminCode] = true
				}
			}

			// 保存节点权限
			permissionsMap[node.FullCodePath] = nodePerms

			// 5. 计算传递给子节点的继承权限
			childInheritedPerms := make(map[string]bool)
			// 合并当前节点权限和父节点传递的权限
			for k, v := range inheritedPerms {
				childInheritedPerms[k] = v
			}
			for k, v := range currentNodePerms {
				childInheritedPerms[k] = v
			}

			// 6. 递归处理子节点
			if len(node.Children) > 0 {
				calculatePermissionsRecursive(node.Children, childInheritedPerms)
			}
		}
	}

	// 从根节点开始计算（初始继承权限为空）
	calculatePermissionsRecursive(trees, nil)

	logger.Debugf(ctx, "[ServiceTreeService] 权限计算完成（支持角色权限）: 节点数=%d, 权限节点数=%d", len(trees), len(permissionsMap))

	return permissionsMap, nil
}

// applyPermissionInheritance 应用权限继承规则
// ⭐ 新权限系统实现权限继承逻辑
// applyPermissionInheritance 应用权限继承规则（新格式：resource_type:action_type）
// ⭐ 权限点格式：resource_type:action_type（如 directory:read, table:write）
// 目录权限继承：directory:read -> table:read（需要转换）
func (s *ServiceTreeService) applyPermissionInheritance(
	nodeType string,
	templateType string,
	parentPerms map[string]bool, // 父目录的权限（格式：actionCode -> true，如 directory:read -> true）
	nodePerms map[string]bool, // 子节点的权限（格式：actionCode -> true，如 table:read -> true）
) {
	// 获取子节点的资源类型
	resourceType := permission.GetResourceType(nodeType, templateType)
	if resourceType == "" {
		return
	}

	// 遍历父目录的权限，应用继承规则
	for parentActionCode := range parentPerms {
		// 解析父目录权限点编码
		parentResourceType, actionType, ok := permission.ParseActionCode(parentActionCode)
		if !ok {
			continue
		}

		// 如果是目录权限，需要转换为子节点的资源类型
		if parentResourceType == permission.ResourceTypeDirectory {
			if actionType == "admin" {
				// directory:admin -> 所有权限
				for actionCode := range nodePerms {
					nodePerms[actionCode] = true
				}
				return
			} else {
				// directory:read -> table:read（需要转换）
				childActionCode := permission.BuildActionCode(resourceType, actionType)
				if _, exists := nodePerms[childActionCode]; exists {
					nodePerms[childActionCode] = true
				}
			}
		} else if parentResourceType == permission.ResourceTypeApp {
			if actionType == "admin" {
				// app:admin -> 所有权限
				for actionCode := range nodePerms {
					nodePerms[actionCode] = true
				}
				return
			}
		}
	}
}

// hasFunctionInDirectChildren 只检查直接子节点是否有 function 类型（不递归）
func (s *ServiceTreeService) hasFunctionInDirectChildren(node *model.ServiceTree) bool {
	if node == nil {
		return false
	}

	// 只检查直接子节点，不递归检查子目录的子节点
	for _, child := range node.Children {
		if child.Type == model.ServiceTreeTypeFunction {
			return true
		}
	}

	return false
}

// ⭐ 已废弃：collectPermissionInfo 和 convertToGetServiceTreeRespWithPermissions 方法已不再使用
// 服务树不再返回权限信息，权限在详情接口中查询

// getPermissionActionsForNode 根据节点类型和模板类型，获取需要检查的权限点
// ⭐ 优化：使用公共函数，避免代码重复
func (s *ServiceTreeService) getPermissionActionsForNode(nodeType string, templateType string) []string {
	// 将 model.ServiceTreeTypePackage 和 model.ServiceTreeTypeFunction 转换为字符串
	var nodeTypeStr string
	if nodeType == model.ServiceTreeTypePackage {
		nodeTypeStr = "package"
	} else if nodeType == model.ServiceTreeTypeFunction {
		nodeTypeStr = "function"
	} else {
		return []string{}
	}

	return permission.GetActionsForNode(nodeTypeStr, templateType)
}

// GetServiceTreeByFullPath 根据完整路径获取服务目录（用于权限检查）
func (s *ServiceTreeService) GetServiceTreeByFullPath(ctx context.Context, fullPath string) (*model.ServiceTree, error) {
	return s.serviceTreeRepo.GetServiceTreeByFullPath(fullPath)
}

// sendCreateServiceTreeMessage 发送创建服务目录的NATS消息
func (s *ServiceTreeService) sendCreateServiceTreeMessage(ctx context.Context, user, app string, serviceTree *model.ServiceTree) error {
	// 获取应用信息以获取HostID
	appModel, err := s.appRepo.GetAppByUserName(user, app)
	if err != nil {
		return fmt.Errorf("failed to get app info: %w", err)
	}

	// 构建消息
	req := dto.CreateServiceTreeRuntimeReq{
		User: user,
		App:  app,
		ServiceTree: &dto.ServiceTreeRuntimeData{
			ID:           serviceTree.ID,
			Name:         serviceTree.Name,
			Code:         serviceTree.Code,
			ParentID:     serviceTree.ParentID,
			Type:         serviceTree.Type,
			Description:  serviceTree.Description,
			Tags:         serviceTree.Tags,
			AppID:        serviceTree.AppID,
			FullCodePath: serviceTree.FullCodePath,
		},
	}

	// 调用 app-runtime 创建服务目录，使用应用所属的 HostID
	_, err = s.appRuntime.CreateServiceTree(ctx, appModel.HostID, &req)
	if err != nil {
		return fmt.Errorf("failed to create service tree via app-runtime: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] Service tree created successfully via app-runtime: %s/%s/%s",
		user, app, serviceTree.Code)

	return nil
}

// GetDirectorySnapshotsRecursively 递归获取目录及其所有子目录的文件快照
// GetDirectorySnapshotsRecursively 递归获取目录及其所有子目录的当前版本文件快照
// 优化：使用 ServiceTreeID 和 IsCurrent 字段，性能更好
// 返回：map[目录路径][]文件快照
func (s *ServiceTreeService) GetDirectorySnapshotsRecursively(ctx context.Context, appID int64, rootDirectoryPath string) (map[string][]*model.FileSnapshot, error) {
	// 1. 获取根目录节点（ServiceTree）
	rootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(rootDirectoryPath)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warnf(ctx, "[ServiceTreeService] 根目录节点不存在: path=%s", rootDirectoryPath)
			return make(map[string][]*model.FileSnapshot), nil
		}
		return nil, fmt.Errorf("获取根目录节点失败: %w", err)
	}

	// 2. 递归查询所有子目录节点（包括根目录）
	descendants, err := s.serviceTreeRepo.GetDescendantDirectories(appID, rootDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("查询子目录失败: %w", err)
	}

	// 3. 构建所有目录节点的列表（包括根目录）
	allTrees := make([]*model.ServiceTree, 0, len(descendants)+1)
	allTrees = append(allTrees, rootTree)
	allTrees = append(allTrees, descendants...)

	// 4. 收集所有目录的 ServiceTreeID
	treeIDs := make([]int64, 0, len(allTrees))
	treeIDToPath := make(map[int64]string) // ServiceTreeID -> FullCodePath
	for _, tree := range allTrees {
		treeIDs = append(treeIDs, tree.ID)
		treeIDToPath[tree.ID] = tree.FullCodePath
	}

	// 5. 批量查询所有目录的当前版本快照（使用 ServiceTreeID 和 IsCurrent）
	// 一次性查询所有目录的快照，性能更好
	allSnapshots, err := s.fileSnapshotRepo.GetCurrentSnapshotsByServiceTreeIDs(treeIDs)
	if err != nil {
		return nil, fmt.Errorf("批量查询文件快照失败: %w", err)
	}

	// 6. 按目录路径分组快照
	result := make(map[string][]*model.FileSnapshot)
	for _, tree := range allTrees {
		// 初始化每个目录的空列表（即使没有文件也要包含）
		result[tree.FullCodePath] = make([]*model.FileSnapshot, 0)
	}

	// 7. 将快照按目录分组
	for _, snapshot := range allSnapshots {
		path := treeIDToPath[snapshot.ServiceTreeID]
		if path != "" {
			result[path] = append(result[path], snapshot)
		}
	}

	totalFiles := 0
	for _, files := range result {
		totalFiles += len(files)
	}

	logger.Infof(ctx, "[ServiceTreeService] 递归获取快照完成: 根目录=%s, 目录数=%d, 总文件数=%d",
		rootDirectoryPath, len(allTrees), totalFiles)

	return result, nil
}

// extractPackageFromPath 从完整路径提取 package 路径（去掉应用前缀）
func extractPackageFromPath(fullCodePath string) string {
	// 格式：/user/app/package1/package2
	// 返回：package1/package2
	parts := strings.Split(strings.Trim(fullCodePath, "/"), "/")
	if len(parts) < 3 {
		return ""
	}
	return strings.Join(parts[2:], "/")
}

// CopyServiceTree 复制服务目录（递归复制目录及其所有子目录）
// 支持两种模式：
// 1. 本地复制：从本地工作空间复制目录
// 2. Hub 复制：从 Hub 链接复制目录（自动检测 hub:// 前缀）
func (s *ServiceTreeService) CopyServiceTree(ctx context.Context, req *dto.CopyDirectoryReq) (*dto.CopyDirectoryResp, error) {
	// 1. 获取目标应用信息
	targetApp, err := s.appRepo.GetAppByID(req.TargetAppID)
	if err != nil {
		return nil, fmt.Errorf("获取目标应用失败: %w", err)
	}

	// 2. 检测是否为 Hub 链接
	if strings.HasPrefix(req.SourceDirectoryPath, "hub://") {
		// Hub 复制模式：使用 PullDirectoryFromHub 的逻辑
		return s.copyFromHub(ctx, req, targetApp)
	}

	// 3. 本地复制模式：原有的本地复制逻辑
	return s.copyFromLocal(ctx, req, targetApp)
}

// copyFromLocal 从本地工作空间复制目录
func (s *ServiceTreeService) copyFromLocal(ctx context.Context, req *dto.CopyDirectoryReq, targetApp *model.App) (*dto.CopyDirectoryResp, error) {
	// 解析源目录信息
	sourceParts := strings.Split(strings.Trim(req.SourceDirectoryPath, "/"), "/")
	if len(sourceParts) < 3 {
		return nil, fmt.Errorf("源目录路径格式错误: %s", req.SourceDirectoryPath)
	}
	sourceUser := sourceParts[0]
	sourceAppCode := sourceParts[1]

	// 3. 获取源应用信息
	sourceApp, err := s.appRepo.GetAppByUserName(sourceUser, sourceAppCode)
	if err != nil {
		return nil, fmt.Errorf("获取源应用失败: %w", err)
	}

	// 4. 获取源目录的 ServiceTree 信息（包括所有子目录）
	sourceRootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取源目录信息失败: %w", err)
	}

	// 获取所有子目录（package 类型）
	sourceDescendants, err := s.serviceTreeRepo.GetDescendantDirectories(sourceApp.ID, req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取源子目录失败: %w", err)
	}

	// 构建源目录映射：fullCodePath -> ServiceTree
	sourceTrees := make(map[string]*model.ServiceTree)
	sourceTrees[sourceRootTree.FullCodePath] = sourceRootTree
	for _, desc := range sourceDescendants {
		sourceTrees[desc.FullCodePath] = desc
	}

	// 5. 递归获取所有目录的文件快照（包括根目录和所有子目录）
	directoryFiles, err := s.GetDirectorySnapshotsRecursively(ctx, sourceApp.ID, req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取目录快照失败: %w", err)
	}

	if len(directoryFiles) == 0 {
		return nil, fmt.Errorf("未找到任何目录快照，请确保源目录已创建快照")
	}

	// 6. 获取目标根目录的 ServiceTree（用于确定父目录ID和完整路径）
	targetRootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.TargetDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取目标目录信息失败: %w", err)
	}

	// 使用目标根目录的完整路径作为基础路径
	targetRootPath := targetRootTree.FullCodePath

	// 8. 按层级顺序创建目标目录的 ServiceTree 记录
	// 先按路径长度排序，确保先创建父目录，再创建子目录
	type dirInfo struct {
		sourcePath string
		targetPath string
		sourceTree *model.ServiceTree
	}
	dirsToCreate := make([]dirInfo, 0, len(sourceTrees))

	for sourcePath, sourceTree := range sourceTrees {
		// 计算相对路径（相对于源根目录）
		relativePath := strings.TrimPrefix(sourcePath, req.SourceDirectoryPath)
		relativePath = strings.TrimPrefix(relativePath, "/")

		// 构建目标路径
		var targetPath string
		if relativePath == "" {
			// 根目录：直接在目标根目录下创建同名目录
			// 例如：从 /user/app/a/b 复制到 /user/app/e，应该在 /user/app/e/b 创建目录
			// 提取源目录的最后一部分作为目录名
			sourcePathParts := strings.Split(strings.Trim(req.SourceDirectoryPath, "/"), "/")
			if len(sourcePathParts) < 3 {
				return nil, fmt.Errorf("源目录路径格式错误: %s", req.SourceDirectoryPath)
			}
			dirName := sourcePathParts[len(sourcePathParts)-1] // 获取最后一部分（b）
			targetPath = targetRootPath + "/" + dirName
		} else {
			// 子目录：保持相对路径结构
			// 例如：从 /user/app/a/b/c 复制到 /user/app/e，应该在 /user/app/e/b/c 创建目录
			sourcePathParts := strings.Split(strings.Trim(req.SourceDirectoryPath, "/"), "/")
			if len(sourcePathParts) < 3 {
				return nil, fmt.Errorf("源目录路径格式错误: %s", req.SourceDirectoryPath)
			}
			dirName := sourcePathParts[len(sourcePathParts)-1] // 获取源根目录名（b）
			targetPath = targetRootPath + "/" + dirName + "/" + relativePath
		}

		dirsToCreate = append(dirsToCreate, dirInfo{
			sourcePath: sourcePath,
			targetPath: targetPath,
			sourceTree: sourceTree,
		})
	}

	// 按路径长度排序（短的在前，确保先创建父目录）
	sort.Slice(dirsToCreate, func(i, j int) bool {
		return len(dirsToCreate[i].targetPath) < len(dirsToCreate[j].targetPath)
	})

	// 更新目标路径（因为根目录会在目标根目录下创建同名目录）
	// 例如：从 /user/app/a/b 复制到 /user/app/e，实际目标路径应该是 /user/app/e/b
	if len(dirsToCreate) > 0 {
		// 使用第一个目录（根目录）的目标路径作为新的目标根路径
		targetRootPath = dirsToCreate[0].targetPath
	}

	// 8. 构建批量创建请求（分离目录和文件，参考 PullDirectoryFromHub 的实现）
	directoryItems := make([]*dto.DirectoryTreeItem, 0)
	fileItems := make([]*dto.DirectoryTreeItem, 0)

	// 8.1 添加目录项
	for _, dirInfo := range dirsToCreate {
		directoryItems = append(directoryItems, &dto.DirectoryTreeItem{
			FullCodePath: dirInfo.targetPath,
			Type:         "directory",
			Name:         dirInfo.sourceTree.Name,
			Description:  dirInfo.sourceTree.Description,
			Tags:         dirInfo.sourceTree.Tags,
		})
	}

	// 8.2 添加文件项
	totalFileCount := 0
	logger.Infof(ctx, "[CopyServiceTree] 开始处理文件快照: 源目录=%s, 目标根路径=%s, 目录数=%d",
		req.SourceDirectoryPath, targetRootPath, len(directoryFiles))

	for sourcePath, fileSnapshots := range directoryFiles {
		logger.Infof(ctx, "[CopyServiceTree] 处理目录文件快照: sourcePath=%s, fileCount=%d",
			sourcePath, len(fileSnapshots))

		// 计算相对路径（相对于源根目录）
		relativePath := strings.TrimPrefix(sourcePath, req.SourceDirectoryPath)
		relativePath = strings.TrimPrefix(relativePath, "/")

		// 构建目标路径
		var targetPath string
		if relativePath == "" {
			// 根目录
			targetPath = targetRootPath
		} else {
			// 子目录：保持相对路径结构
			// 例如：源根目录是 /luobei/operations/other/fo，子目录是 /luobei/operations/other/fo/ticket
			// relativePath 应该是 "ticket"，目标路径应该是 targetRootPath + "/ticket"
			targetPath = targetRootPath + "/" + relativePath
		}

		logger.Infof(ctx, "[CopyServiceTree] 目录路径映射: sourcePath=%s -> targetPath=%s, relativePath=%s",
			sourcePath, targetPath, relativePath)

		for _, fileSnapshot := range fileSnapshots {
			// 使用 FileName，如果为空则从相对路径提取
			fileName := fileSnapshot.FileName
			if fileName == "" {
				// 从相对路径提取文件名
				fileName = strings.TrimSuffix(fileSnapshot.RelativePath, ".go")
				if lastSlash := strings.LastIndex(fileName, "/"); lastSlash >= 0 {
					fileName = fileName[lastSlash+1:]
				}
			}

			// 忽略 init_.go 文件（init_.go 是创建目录时动态生成的，不能复制）
			if fileName == "init_" || fileSnapshot.RelativePath == "init_.go" || strings.HasSuffix(fileSnapshot.RelativePath, "/init_.go") {
				logger.Infof(ctx, "[ServiceTreeService] 跳过 init_.go 文件: %s", fileSnapshot.RelativePath)
				continue
			}

			// 构建文件完整路径（FullCodePath 应该只包含目录路径，不包含文件名）
			// BatchWriteFiles 会从 FullCodePath 提取 package 路径，从 FileName 获取文件名
			// 所以 FullCodePath 应该只是目录路径，如 /user/app/package，而不是 /user/app/package/file
			// 参考 buildItemsFromTree 的实现，它也是错误的，但我们需要修复这里

			// 添加文件项
			fileItems = append(fileItems, &dto.DirectoryTreeItem{
				FullCodePath: targetPath, // 只包含目录路径，不包含文件名，如 /user/app/package
				Type:         "file",
				FileName:     fileName,              // 文件名（不含扩展名），如 test
				FileType:     fileSnapshot.FileType, // 文件类型，如 go
				Content:      fileSnapshot.Content,
				RelativePath: fileSnapshot.RelativePath,
			})
			totalFileCount++
		}

		logger.Infof(ctx, "[ServiceTreeService] 准备复制目录: source=%s, target=%s, fileCount=%d",
			sourcePath, targetPath, len(fileSnapshots))
	}

	// 9. 先批量创建目录
	var directoryCount int
	if len(directoryItems) > 0 {
		batchCreateReq := &dto.BatchCreateDirectoryTreeReq{
			User:  targetApp.User,
			App:   targetApp.Code,
			Items: directoryItems,
		}

		batchCreateResp, err := s.BatchCreateDirectoryTree(ctx, batchCreateReq)
		if err != nil {
			logger.Errorf(ctx, "[ServiceTreeService] 批量创建目录失败: error=%v", err)
			return nil, fmt.Errorf("批量创建目录失败: %w", err)
		}

		directoryCount = batchCreateResp.DirectoryCount
		logger.Infof(ctx, "[ServiceTreeService] 批量创建目录完成: directoryCount=%d", directoryCount)
	}

	// 10. 再批量写文件（会编译并返回 diff）
	var fileCount int
	var oldVersion, newVersion, gitCommitHash string
	if len(fileItems) > 0 {
		batchWriteReq := &dto.BatchWriteFilesReq{
			User:  targetApp.User,
			App:   targetApp.Code,
			Files: fileItems,
		}

		batchWriteResp, err := s.BatchWriteFiles(ctx, batchWriteReq)
		if err != nil {
			logger.Errorf(ctx, "[ServiceTreeService] 批量写文件失败: error=%v", err)
			return nil, fmt.Errorf("批量写文件失败: %w", err)
		}

		fileCount = batchWriteResp.FileCount
		oldVersion = batchWriteResp.OldVersion
		newVersion = batchWriteResp.NewVersion
		gitCommitHash = batchWriteResp.GitCommitHash
		logger.Infof(ctx, "[ServiceTreeService] 批量写文件完成: fileCount=%d, oldVersion=%s, newVersion=%s, gitCommitHash=%s",
			fileCount, oldVersion, newVersion, gitCommitHash)
	}

	logger.Infof(ctx, "[ServiceTreeService] 复制目录完成: 目录数=%d, 文件数=%d, oldVersion=%s, newVersion=%s",
		directoryCount, fileCount, oldVersion, newVersion)

	return &dto.CopyDirectoryResp{
		Message:        fmt.Sprintf("复制目录成功，共复制 %d 个目录，%d 个文件", directoryCount, fileCount),
		DirectoryCount: directoryCount,
		FileCount:      fileCount,
		OldVersion:     oldVersion,
		NewVersion:     newVersion,
		GitCommitHash:  gitCommitHash,
	}, nil
}

// copyFromHub 从 Hub 链接复制目录
func (s *ServiceTreeService) copyFromHub(ctx context.Context, req *dto.CopyDirectoryReq, targetApp *model.App) (*dto.CopyDirectoryResp, error) {
	// 1. 解析 Hub 链接
	hubLinkInfo, err := ParseHubLink(req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("解析 Hub 链接失败: %w", err)
	}

	logger.Infof(ctx, "[CopyServiceTree] 解析 Hub 链接成功: host=%s, fullCodePath=%s, version=%s",
		hubLinkInfo.Host, hubLinkInfo.FullCodePath, hubLinkInfo.Version)

	// 2. 从 Hub 获取目录详情（包含目录树和文件内容，如果指定了版本号则查询指定版本）
	hubDetail, err := apicall.GetHubDirectoryDetailFromHost(ctx, &dto.GetHubDirectoryDetailFromHostReq{
		Host:         hubLinkInfo.Host,
		FullCodePath: hubLinkInfo.FullCodePath,
		Version:      hubLinkInfo.Version,
		IncludeTree:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("获取 Hub 目录详情失败: %w", err)
	}

	// 详细日志：检查 Hub 返回的目录树结构
	if hubDetail.DirectoryTree != nil {
		logger.Infof(ctx, "[CopyServiceTree] Hub 目录树根节点信息: Name=%s, Code=%s, Path=%s, Files数量=%d, Subdirectories数量=%d",
			hubDetail.DirectoryTree.Name, hubDetail.DirectoryTree.Code, hubDetail.DirectoryTree.Path,
			len(hubDetail.DirectoryTree.Files), len(hubDetail.DirectoryTree.Subdirectories))

		// 递归打印所有节点的详细信息
		s.logDirectoryTree(ctx, hubDetail.DirectoryTree, 0)
	} else {
		logger.Warnf(ctx, "[CopyServiceTree] Hub 目录树为空")
	}

	// 3. 如果指定了版本号，验证版本是否匹配
	if hubLinkInfo.Version != "" && hubDetail.Version != hubLinkInfo.Version {
		return nil, fmt.Errorf("版本不匹配：请求版本 %s，实际版本 %s", hubLinkInfo.Version, hubDetail.Version)
	}

	// 4. 确定目标目录路径
	targetPath := req.TargetDirectoryPath

	// 5. 从 DirectoryTree 构建批量创建请求
	if hubDetail.DirectoryTree == nil {
		return nil, fmt.Errorf("Hub 目录树为空")
	}

	// 5.1 构建目录项列表
	directoryItems := make([]*dto.DirectoryTreeItem, 0)
	fileItems := make([]*dto.DirectoryTreeItem, 0)

	// 递归遍历目录树，构建目录和文件项
	s.buildItemsFromTree(hubDetail.DirectoryTree, targetPath, &directoryItems, &fileItems)

	// 6. 先批量创建目录
	var directoryCount int
	if len(directoryItems) > 0 {
		batchCreateReq := &dto.BatchCreateDirectoryTreeReq{
			User:  targetApp.User,
			App:   targetApp.Code,
			Items: directoryItems,
		}

		batchCreateResp, err := s.BatchCreateDirectoryTree(ctx, batchCreateReq)
		if err != nil {
			return nil, fmt.Errorf("批量创建目录失败: %w", err)
		}

		directoryCount = batchCreateResp.DirectoryCount
		logger.Infof(ctx, "[CopyServiceTree] 批量创建目录完成: directoryCount=%d", directoryCount)
	}

	// 7. 再批量写文件（会编译并返回 diff）
	var fileCount int
	var oldVersion, newVersion, gitCommitHash string
	if len(fileItems) > 0 {
		batchWriteReq := &dto.BatchWriteFilesReq{
			User:  targetApp.User,
			App:   targetApp.Code,
			Files: fileItems,
		}

		batchWriteResp, err := s.BatchWriteFiles(ctx, batchWriteReq)
		if err != nil {
			return nil, fmt.Errorf("批量写文件失败: %w", err)
		}

		fileCount = batchWriteResp.FileCount
		oldVersion = batchWriteResp.OldVersion
		newVersion = batchWriteResp.NewVersion
		gitCommitHash = batchWriteResp.GitCommitHash
		logger.Infof(ctx, "[CopyServiceTree] 批量写文件完成: fileCount=%d, oldVersion=%s, newVersion=%s, gitCommitHash=%s",
			fileCount, oldVersion, newVersion, gitCommitHash)
	}

	// 8. 获取根目录的 ServiceTree ID（用于建立双向绑定）
	// 直接使用 Code 字段
	rootDirPath := targetPath
	if hubDetail.DirectoryTree != nil && hubDetail.DirectoryTree.Code != "" {
		rootDirPath = fmt.Sprintf("%s/%s", targetPath, hubDetail.DirectoryTree.Code)
	}
	rootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(rootDirPath)
	if err != nil {
		logger.Warnf(ctx, "[CopyServiceTree] 获取根目录 ServiceTree 失败: path=%s, error=%v", rootDirPath, err)
	}

	// 9. 建立双向绑定：更新根目录节点的 HubDirectoryID 和版本信息
	if rootTree != nil && hubDetail.ID > 0 {
		rootTree.HubDirectoryID = hubDetail.ID
		rootTree.HubVersion = hubDetail.Version
		rootTree.HubVersionNum = hubDetail.VersionNum
		if err := s.serviceTreeRepo.UpdateServiceTree(rootTree); err != nil {
			logger.Warnf(ctx, "[CopyServiceTree] 更新ServiceTree的Hub信息失败: treeID=%d, hubDirectoryID=%d, hubVersion=%s, error=%v",
				rootTree.ID, hubDetail.ID, hubDetail.Version, err)
		} else {
			logger.Infof(ctx, "[CopyServiceTree] 成功建立双向绑定: treeID=%d, hubDirectoryID=%d, hubVersion=%s", rootTree.ID, hubDetail.ID, hubDetail.Version)
		}
	}

	logger.Infof(ctx, "[CopyServiceTree] 从 Hub 复制目录完成: 目录数=%d, 文件数=%d, oldVersion=%s, newVersion=%s",
		directoryCount, fileCount, oldVersion, newVersion)

	return &dto.CopyDirectoryResp{
		Message:        fmt.Sprintf("从 Hub 复制目录成功，共复制 %d 个目录，%d 个文件", directoryCount, fileCount),
		DirectoryCount: directoryCount,
		FileCount:      fileCount,
		OldVersion:     oldVersion,
		NewVersion:     newVersion,
		GitCommitHash:  gitCommitHash,
	}, nil
}

// PublishDirectoryToHub 发布目录到 Hub
func (s *ServiceTreeService) PublishDirectoryToHub(ctx context.Context, req *dto.PublishDirectoryToHubReq) (*dto.PublishDirectoryToHubResp, error) {
	// 1. 获取应用信息
	sourceApp, err := s.appRepo.GetAppByUserName(req.SourceUser, req.SourceApp)
	if err != nil {
		return nil, fmt.Errorf("获取源应用失败: %w", err)
	}

	// 2. 验证源目录是否存在
	sourceTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取源目录信息失败: %w", err)
	}

	// 3. 获取根目录节点和所有子目录节点（用于构建父子关系）
	rootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取根目录节点失败: %w", err)
	}

	descendants, err := s.serviceTreeRepo.GetDescendantDirectories(sourceApp.ID, req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("查询子目录失败: %w", err)
	}

	// 构建所有目录节点的列表和映射
	allTrees := make([]*model.ServiceTree, 0, len(descendants)+1)
	allTrees = append(allTrees, rootTree)
	allTrees = append(allTrees, descendants...)

	// 构建路径到 ServiceTree 的映射，用于快速查找父目录
	pathToTree := make(map[string]*model.ServiceTree)
	idToTree := make(map[int64]*model.ServiceTree)
	for _, tree := range allTrees {
		pathToTree[tree.FullCodePath] = tree
		idToTree[tree.ID] = tree
	}

	// 4. 递归获取所有目录的文件快照（包括根目录和所有子目录）
	directoryFiles, err := s.GetDirectorySnapshotsRecursively(ctx, sourceApp.ID, req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取目录快照失败: %w", err)
	}

	if len(directoryFiles) == 0 {
		return nil, fmt.Errorf("未找到任何目录快照，请确保源目录已创建快照")
	}

	// 5. 获取所有函数节点（function 类型，属于当前目录树下的）
	// 使用路径前缀匹配，只查询属于当前目录的函数
	normalizedPath := strings.TrimSuffix(req.SourceDirectoryPath, "/") + "/"
	allFunctions, err := s.serviceTreeRepo.GetServiceTreesByAppIDAndType(sourceApp.ID, model.ServiceTreeTypeFunction)
	if err != nil {
		return nil, fmt.Errorf("查询函数节点失败: %w", err)
	}

	// 构建函数映射：ParentID -> []Function
	functionMap := make(map[int64][]*model.ServiceTree)
	for _, fn := range allFunctions {
		// 只包含属于当前目录树下的函数（路径前缀匹配）
		if strings.HasPrefix(fn.FullCodePath, normalizedPath) || fn.FullCodePath == req.SourceDirectoryPath {
			// 找到函数所属的目录节点（通过 ParentID 匹配）
			if dirTree, exists := idToTree[fn.ParentID]; exists {
				// 确保目录节点也在当前目录树下
				if strings.HasPrefix(dirTree.FullCodePath, normalizedPath) || dirTree.FullCodePath == req.SourceDirectoryPath {
					functionMap[dirTree.ID] = append(functionMap[dirTree.ID], fn)
				}
			}
		}
	}

	// 6. 构建树形结构（包含函数）
	directoryTree := s.buildDirectoryTree(rootTree, allTrees, directoryFiles, idToTree, functionMap)

	// 6. 构建 Hub 请求
	hubReq := &dto.PublishHubDirectoryReq{
		SourceUser:           req.SourceUser,
		SourceApp:            req.SourceApp,
		SourceDirectoryPath:  req.SourceDirectoryPath,
		Name:                 req.Name,
		Description:          req.Description,
		Category:             req.Category,
		Tags:                 req.Tags,
		ServiceFeePersonal:   req.ServiceFeePersonal,
		ServiceFeeEnterprise: req.ServiceFeeEnterprise,
		Version:              sourceTree.Version,
		DirectoryTree:        directoryTree,
	}

	// 7. 调用 Hub API（直接传 ctx，内部会提取 token、trace_id 等）
	hubResp, err := apicall.PublishDirectoryToHub(ctx, hubReq)
	if err != nil {
		return nil, fmt.Errorf("调用 Hub API 失败: %w", err)
	}

	// 8. 建立双向绑定：更新根目录节点的 HubDirectoryID 和版本信息
	// 需要查询 Hub 获取版本信息（因为 hubResp 可能不包含版本）
	hubDetail, err := apicall.GetHubDirectoryDetail(ctx, &dto.GetHubDirectoryDetailReq{
		FullCodePath: req.SourceDirectoryPath,
		Version:      "",
		IncludeTree:  false,
	})
	if err != nil {
		logger.Warnf(ctx, "[PublishDirectoryToHub] 获取Hub目录详情失败，无法记录版本信息: hubDirectoryID=%d, error=%v", hubResp.HubDirectoryID, err)
		// 即使获取详情失败，也记录 HubDirectoryID
		rootTree.HubDirectoryID = hubResp.HubDirectoryID
	} else {
		// 记录 Hub 信息（ID 和版本）
		rootTree.HubDirectoryID = hubDetail.ID
		rootTree.HubVersion = hubDetail.Version
		rootTree.HubVersionNum = hubDetail.VersionNum
	}

	if err := s.serviceTreeRepo.UpdateServiceTree(rootTree); err != nil {
		logger.Warnf(ctx, "[PublishDirectoryToHub] 更新ServiceTree的Hub信息失败: treeID=%d, hubDirectoryID=%d, hubVersion=%s, error=%v",
			rootTree.ID, rootTree.HubDirectoryID, rootTree.HubVersion, err)
		// 不返回错误，因为发布已经成功，只是绑定失败
	} else {
		logger.Infof(ctx, "[PublishDirectoryToHub] 成功建立双向绑定: treeID=%d, hubDirectoryID=%d, hubVersion=%s", rootTree.ID, rootTree.HubDirectoryID, rootTree.HubVersion)
	}

	// 9. 返回结果
	return &dto.PublishDirectoryToHubResp{
		HubDirectoryID:  hubResp.HubDirectoryID,
		HubDirectoryURL: hubResp.HubDirectoryURL,
		DirectoryCount:  hubResp.DirectoryCount,
		FileCount:       hubResp.FileCount,
	}, nil
}

// PushDirectoryToHub 推送目录到 Hub（更新已发布的目录，类似 git push）
func (s *ServiceTreeService) PushDirectoryToHub(ctx context.Context, req *dto.PushDirectoryToHubReq) (*dto.PushDirectoryToHubResp, error) {
	// 1. 获取应用信息
	sourceApp, err := s.appRepo.GetAppByUserName(req.SourceUser, req.SourceApp)
	if err != nil {
		return nil, fmt.Errorf("获取源应用失败: %w", err)
	}

	// 2. 验证源目录是否存在
	sourceTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取源目录信息失败: %w", err)
	}

	// 3. 检查目录是否已发布到 Hub
	if sourceTree.HubDirectoryID == 0 {
		return nil, fmt.Errorf("目录尚未发布到 Hub，请先使用 PublishDirectoryToHub 发布")
	}

	// 4. 获取根目录节点和所有子目录节点（用于构建父子关系）
	rootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取根目录节点失败: %w", err)
	}

	descendants, err := s.serviceTreeRepo.GetDescendantDirectories(sourceApp.ID, req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("查询子目录失败: %w", err)
	}

	// 构建所有目录节点的列表和映射
	allTrees := make([]*model.ServiceTree, 0, len(descendants)+1)
	allTrees = append(allTrees, rootTree)
	allTrees = append(allTrees, descendants...)

	// 构建路径到 ServiceTree 的映射，用于快速查找父目录
	idToTree := make(map[int64]*model.ServiceTree)
	for _, tree := range allTrees {
		idToTree[tree.ID] = tree
	}

	// 5. 递归获取所有目录的文件快照（包括根目录和所有子目录）
	directoryFiles, err := s.GetDirectorySnapshotsRecursively(ctx, sourceApp.ID, req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取目录快照失败: %w", err)
	}

	if len(directoryFiles) == 0 {
		return nil, fmt.Errorf("未找到任何目录快照，请确保源目录已创建快照")
	}

	// 6. 获取所有函数节点（function 类型，属于当前目录树下的）
	// 使用路径前缀匹配，只查询属于当前目录的函数
	normalizedPath := strings.TrimSuffix(req.SourceDirectoryPath, "/") + "/"
	allFunctions, err := s.serviceTreeRepo.GetServiceTreesByAppIDAndType(sourceApp.ID, model.ServiceTreeTypeFunction)
	if err != nil {
		return nil, fmt.Errorf("查询函数节点失败: %w", err)
	}

	// 构建函数映射：ParentID -> []Function
	functionMap := make(map[int64][]*model.ServiceTree)
	for _, fn := range allFunctions {
		// 只包含属于当前目录树下的函数（路径前缀匹配）
		if strings.HasPrefix(fn.FullCodePath, normalizedPath) || fn.FullCodePath == req.SourceDirectoryPath {
			// 找到函数所属的目录节点（通过 ParentID 匹配）
			if dirTree, exists := idToTree[fn.ParentID]; exists {
				// 确保目录节点也在当前目录树下
				if strings.HasPrefix(dirTree.FullCodePath, normalizedPath) || dirTree.FullCodePath == req.SourceDirectoryPath {
					functionMap[dirTree.ID] = append(functionMap[dirTree.ID], fn)
				}
			}
		}
	}

	// 7. 构建树形结构（包含函数）
	directoryTree := s.buildDirectoryTree(rootTree, allTrees, directoryFiles, idToTree, functionMap)

	// 7. 构建 Hub 请求
	hubReq := &dto.UpdateHubDirectoryReq{
		APIKey:               req.APIKey,
		HubDirectoryID:       sourceTree.HubDirectoryID,
		SourceDirectoryPath:  req.SourceDirectoryPath,
		Name:                 req.Name,
		Description:          req.Description,
		Category:             req.Category,
		Tags:                 req.Tags,
		ServiceFeePersonal:   req.ServiceFeePersonal,
		ServiceFeeEnterprise: req.ServiceFeeEnterprise,
		Version:              req.Version,
		DirectoryTree:        directoryTree,
	}

	// 8. 调用 Hub API
	// 调用 Hub API（直接传 ctx，内部会提取 token、trace_id 等）
	hubResp, err := apicall.UpdateDirectoryToHub(ctx, hubReq)
	if err != nil {
		return nil, fmt.Errorf("调用 Hub API 失败: %w", err)
	}

	// 9. 更新根目录节点的版本信息
	rootTree.HubVersion = hubResp.NewVersion
	rootTree.HubVersionNum = extractVersionNumForServiceTree(hubResp.NewVersion)
	if err := s.serviceTreeRepo.UpdateServiceTree(rootTree); err != nil {
		logger.Warnf(ctx, "[PushDirectoryToHub] 更新ServiceTree的Hub版本信息失败: treeID=%d, hubDirectoryID=%d, newVersion=%s, error=%v",
			rootTree.ID, hubResp.HubDirectoryID, hubResp.NewVersion, err)
		// 不返回错误，因为推送已经成功，只是更新版本信息失败
	} else {
		logger.Infof(ctx, "[PushDirectoryToHub] 成功更新Hub版本: treeID=%d, hubDirectoryID=%d, oldVersion=%s, newVersion=%s",
			rootTree.ID, hubResp.HubDirectoryID, hubResp.OldVersion, hubResp.NewVersion)
	}

	// 10. 返回结果
	return &dto.PushDirectoryToHubResp{
		HubDirectoryID:  hubResp.HubDirectoryID,
		HubDirectoryURL: hubResp.HubDirectoryURL,
		DirectoryCount:  hubResp.DirectoryCount,
		FileCount:       hubResp.FileCount,
		OldVersion:      hubResp.OldVersion,
		NewVersion:      hubResp.NewVersion,
	}, nil
}

// BatchCreateDirectoryTree 批量创建目录树（用于 copy 和 pull from hub）
func (s *ServiceTreeService) BatchCreateDirectoryTree(
	ctx context.Context,
	req *dto.BatchCreateDirectoryTreeReq,
) (*dto.BatchCreateDirectoryTreeResp, error) {
	// 1. 获取应用信息
	app, err := s.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		return nil, fmt.Errorf("获取应用信息失败: %w", err)
	}

	// 2. 构建 runtime 请求
	runtimeReq := &dto.BatchCreateDirectoryTreeRuntimeReq{
		User:  req.User,
		App:   req.App,
		Items: req.Items,
	}

	// 3. 调用 runtime 批量创建
	runtimeResp, err := s.appRuntime.BatchCreateDirectoryTree(ctx, app.HostID, runtimeReq)
	if err != nil {
		return nil, fmt.Errorf("批量创建目录树失败: %w", err)
	}

	// 4. 创建 ServiceTree 记录（批量创建数据库记录）
	// 这里需要根据 Items 创建对应的 ServiceTree 记录
	// 先按路径排序，确保先创建父目录
	sortedItems := make([]*dto.DirectoryTreeItem, len(req.Items))
	copy(sortedItems, req.Items)

	// 按路径长度排序
	sort.Slice(sortedItems, func(i, j int) bool {
		return len(sortedItems[i].FullCodePath) < len(sortedItems[j].FullCodePath)
	})

	// 创建路径到 ServiceTree 的映射
	pathToTree := make(map[string]*model.ServiceTree)
	currentVersionNum := extractVersionNumForServiceTree(app.Version)

	// 遍历所有项，创建 ServiceTree 记录（只处理目录）
	for _, item := range sortedItems {
		// 只处理目录，文件不在这里处理
		if item.Type != "directory" {
			continue
		}

		// 提取目录代码（路径的最后一部分）
		pathParts := strings.Split(strings.Trim(item.FullCodePath, "/"), "/")
		if len(pathParts) < 3 {
			continue // 跳过无效路径
		}
		dirCode := pathParts[len(pathParts)-1]

		// 查找父目录
		var parentID int64 = 0
		parentPath := getParentPath(item.FullCodePath)
		if parentPath != "" {
			// 先从本次创建的目录中查找
			if parentTree, exists := pathToTree[parentPath]; exists {
				parentID = parentTree.ID
			} else {
				// 如果本次创建中没有，从数据库中查找（可能父目录已存在）
				if existingParent, err := s.serviceTreeRepo.GetServiceTreeByFullPath(parentPath); err == nil {
					parentID = existingParent.ID
					// 将已存在的父目录也加入映射，供后续子目录使用
					pathToTree[parentPath] = existingParent
				}
			}
		}

		// 创建 ServiceTree 记录
		newTree := &model.ServiceTree{
			Name:             item.Name,
			Code:             dirCode,
			ParentID:         parentID,
			Type:             model.ServiceTreeTypePackage,
			Description:      item.Description,
			Tags:             item.Tags,
			AppID:            app.ID,
			FullCodePath:     item.FullCodePath,
			AddVersionNum:    currentVersionNum,
			UpdateVersionNum: 0,
		}

		// 保存到数据库
		if err := s.serviceTreeRepo.CreateServiceTreeWithParentPath(newTree, ""); err != nil {
			logger.Warnf(ctx, "[BatchCreateDirectoryTree] 创建 ServiceTree 记录失败: path=%s, error=%v",
				item.FullCodePath, err)
			// 不返回错误，因为目录已经创建成功
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

// getParentPath 获取父目录路径（辅助函数）
// 路径格式：/user/app/package1/package2/...
// 根目录是 /user/app（2部分），第一个 package 是 /user/app/package1（3部分）
// 所以只有当路径部分数 <= 2 时才是根目录，没有父目录
func getParentPath(fullCodePath string) string {
	pathParts := strings.Split(strings.Trim(fullCodePath, "/"), "/")
	if len(pathParts) <= 2 {
		return "" // 没有父目录（已经是根目录 /user/app）
	}
	// 去掉最后一部分，重新组合
	parentParts := pathParts[:len(pathParts)-1]
	return "/" + strings.Join(parentParts, "/")
}

// AddFunctions 向服务目录添加函数（同步处理）
func (s *ServiceTreeService) AddFunctions(ctx context.Context, req *dto.AddFunctionsReq) (*dto.AddFunctionsResp, error) {
	// 1. 根据 TreeID 获取 ServiceTree（需要预加载 App）
	serviceTree, err := s.serviceTreeRepo.GetByID(req.TreeID)
	if err != nil {
		logger.Errorf(ctx, "[ServiceTreeService] 获取 ServiceTree 失败: TreeID=%d, error=%v", req.TreeID, err)
		return &dto.AddFunctionsResp{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// 预加载 App 信息（如果还没有加载）
	if serviceTree.App == nil {
		app, err := s.appRepo.GetAppByID(serviceTree.AppID)
		if err != nil {
			logger.Errorf(ctx, "[ServiceTreeService] 获取 App 失败: AppID=%d, error=%v", serviceTree.AppID, err)
			return &dto.AddFunctionsResp{
				Success: false,
				Error:   err.Error(),
			}, err
		}
		serviceTree.App = app
	}

	// 2. 从 ServiceTree 中提取 package 路径（使用 model 方法）
	packagePath := serviceTree.GetPackagePathForFileCreation()

	// 3. 使用 agent-server 处理后的结构化数据
	fileName := req.FileName
	if fileName == "" {
		logger.Warnf(ctx, "[ServiceTreeService] agent-server 未提取到文件名，使用 ServiceTree.Code 作为 fallback: %s", serviceTree.Code)
		fileName = serviceTree.Code
	}

	sourceCode := req.SourceCode
	if sourceCode == "" {
		logger.Errorf(ctx, "[ServiceTreeService] SourceCode 为空，无法创建函数")
		return &dto.AddFunctionsResp{
			Success: false,
			Error:   "SourceCode 不能为空",
		}, fmt.Errorf("SourceCode 不能为空")
	}

	logger.Infof(ctx, "[ServiceTreeService] 添加函数: DirectoryPath=%s, FileName=%s, SourceCodeLength=%d", packagePath, fileName, len(sourceCode))

	// 4. 构建 CreateFunctionInfo
	createFunction := &dto.CreateFunctionInfo{
		DirectoryPath: packagePath,
		FileName:      fileName,
		SourceCode:    sourceCode,
	}

	// 5. 调用 AppService.UpdateApp
	updateReq := &dto.UpdateAppReq{
		User:            req.User,
		App:             serviceTree.App.Code,
		CreateFunctions: []*dto.CreateFunctionInfo{createFunction},
	}

	_, err = s.appService.UpdateApp(ctx, updateReq)
	if err != nil {
		logger.Errorf(ctx, "[ServiceTreeService] AppService.UpdateApp 失败: error=%v", err)
		return &dto.AddFunctionsResp{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// 6. 返回同步结果（不发送回调）
	return &dto.AddFunctionsResp{
		Success: true,
		AppID:   serviceTree.App.ID,
		AppCode: serviceTree.App.Code,
	}, nil
}

// ProcessFunctionGenResult 处理函数生成结果（接收 agent-server 处理后的结构化数据）
// 解析元数据、创建目录、创建函数文件、发送回调
func (s *ServiceTreeService) ProcessFunctionGenResult(ctx context.Context, req *dto.AddFunctionsReq) error {
	// 1. 根据 TreeID 获取父目录 ServiceTree（需要预加载 App）
	parentTree, err := s.serviceTreeRepo.GetByID(req.TreeID)
	if err != nil {
		logger.Errorf(ctx, "[ServiceTreeService] 获取父目录 ServiceTree 失败: TreeID=%d, error=%v", req.TreeID, err)
		return err
	}

	// 预加载 App 信息（如果还没有加载）
	if parentTree.App == nil {
		app, err := s.appRepo.GetAppByID(parentTree.AppID)
		if err != nil {
			logger.Errorf(ctx, "[ServiceTreeService] 获取 App 失败: AppID=%d, error=%v", parentTree.AppID, err)
			return err
		}
		parentTree.App = app
	}

	// 2. ⭐ 解析代码中的元数据
	sourceCode := req.SourceCode
	if sourceCode == "" {
		logger.Errorf(ctx, "[ServiceTreeService] SourceCode 为空，无法处理函数生成结果")
		return fmt.Errorf("SourceCode 不能为空")
	}

	var meta metadata.Metadata
	var targetTree *model.ServiceTree = parentTree // 默认使用父目录
	var fileName string

	// 尝试解析元数据
	if err := metadata.ParseMetadata(sourceCode, &meta); err == nil {
		// 元数据解析成功
		logger.Infof(ctx, "[ServiceTreeService] 元数据解析成功 - DirectoryCode: %s, DirectoryName: %s, File: %s",
			meta.DirectoryCode, meta.DirectoryName, meta.File)

		// 验证必需字段
		if meta.DirectoryCode != "" && meta.File != "" {
			// 3. ⭐ 根据元数据创建或查找目录
			targetTree, err = s.createOrFindDirectory(ctx, parentTree, &meta)
			if err != nil {
				logger.Errorf(ctx, "[ServiceTreeService] 创建或查找目录失败: %v", err)
				return err
			}

			// ⭐ 确保 targetTree 的 App 已加载（createOrFindDirectory 返回的目录可能没有预加载 App）
			if targetTree.App == nil {
				app, err := s.appRepo.GetAppByID(targetTree.AppID)
				if err != nil {
					logger.Errorf(ctx, "[ServiceTreeService] 获取 App 失败: AppID=%d, error=%v", targetTree.AppID, err)
					return err
				}
				targetTree.App = app
			}

			// 从元数据中提取文件名（去掉 .go 后缀）
			fileName = strings.TrimSuffix(meta.File, ".go")
		} else {
			logger.Warnf(ctx, "[ServiceTreeService] 元数据缺少必需字段，使用父目录 - DirectoryCode: %s, File: %s",
				meta.DirectoryCode, meta.File)
		}
	} else {
		logger.Warnf(ctx, "[ServiceTreeService] 元数据解析失败，使用父目录: %v", err)
	}

	// 如果文件名仍为空，使用 ServiceTree.Code 作为 fallback
	if fileName == "" {
		logger.Warnf(ctx, "[ServiceTreeService] 未从元数据提取到文件名，使用 ServiceTree.Code 作为 fallback: %s", targetTree.Code)
		fileName = targetTree.Code
	}

	// ⭐ 确保 targetTree 的 App 已加载（即使使用父目录，也要确保 App 已加载）
	if targetTree.App == nil {
		app, err := s.appRepo.GetAppByID(targetTree.AppID)
		if err != nil {
			logger.Errorf(ctx, "[ServiceTreeService] 获取 App 失败: AppID=%d, error=%v", targetTree.AppID, err)
			return err
		}
		targetTree.App = app
	}

	// 4. 从目标目录中提取 package 路径
	packagePath := targetTree.GetPackagePathForFileCreation()

	logger.Infof(ctx, "[ServiceTreeService] 处理完成 - TargetTreeID: %d, DirectoryPath: %s, FileName: %s, SourceCodeLength: %d",
		targetTree.ID, packagePath, fileName, len(sourceCode))

	// 5. 构建 CreateFunctionInfo
	// ⭐ 不再修复 package 声明，应该保证生成的代码是正确的（package 名称应该由元数据中的 directory_code 决定）
	createFunction := &dto.CreateFunctionInfo{
		DirectoryPath: packagePath,
		FileName:      fileName,
		SourceCode:    sourceCode,
	}

	// 6. 调用 AppService.UpdateApp，传入 CreateFunctions
	updateReq := &dto.UpdateAppReq{
		User:            req.User,
		App:             targetTree.App.Code,
		CreateFunctions: []*dto.CreateFunctionInfo{createFunction},
	}

	logger.Infof(ctx, "[ServiceTreeService] 调用 AppService.UpdateApp: User=%s, App=%s, DirectoryPath=%s, FileName=%s",
		updateReq.User, updateReq.App, packagePath, fileName)

	updateResp, err := s.appService.UpdateApp(ctx, updateReq)
	if err != nil {
		logger.Errorf(ctx, "[ServiceTreeService] AppService.UpdateApp 失败: error=%v", err)
		return err
	}

	logger.Infof(ctx, "[ServiceTreeService] 函数创建成功: DirectoryPath=%s, FileName=%s", packagePath, fileName)

	// 7. 获取新增的 FullCodePaths
	fullCodePaths := make([]string, 0)
	if updateResp.Diff != nil {
		fullCodePaths = updateResp.Diff.GetAddFullCodePaths()
		logger.Infof(ctx, "[ServiceTreeService] 获取新增函数完整代码路径 - Count: %d, FullCodePaths: %v", len(fullCodePaths), fullCodePaths)
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
		logger.Errorf(ctx, "[ServiceTreeService] 通知工作空间更新完成失败: error=%v", err)
		// 不中断流程，记录日志即可
	} else {
		if len(fullCodePaths) > 0 {
			logger.Infof(ctx, "[ServiceTreeService] 工作空间更新完成通知已发送 (HTTP) - RecordID: %d, MessageID: %d, FullCodePaths: %v, AppCode: %s",
				req.RecordID, req.MessageID, fullCodePaths, targetTree.App.Code)
		} else {
			logger.Infof(ctx, "[ServiceTreeService] 工作空间更新完成通知已发送 (HTTP) - RecordID: %d, MessageID: %d, FullCodePaths: [] (无新增函数), AppCode: %s",
				req.RecordID, req.MessageID, targetTree.App.Code)
		}
	}

	return nil
}

// createOrFindDirectory 根据元数据创建或查找目录
func (s *ServiceTreeService) createOrFindDirectory(ctx context.Context, parentTree *model.ServiceTree, meta *metadata.Metadata) (*model.ServiceTree, error) {
	// 1. 先尝试查找目录是否已存在（通过 FullCodePath）
	expectedFullCodePath := parentTree.FullCodePath + "/" + meta.DirectoryCode
	existingTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(expectedFullCodePath)
	if err == nil && existingTree != nil {
		logger.Infof(ctx, "[ServiceTreeService] 目录已存在 - TreeID: %d, FullCodePath: %s", existingTree.ID, expectedFullCodePath)
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

	createResp, err := s.CreateServiceTree(ctx, createReq)
	if err != nil {
		// 如果目录已存在（并发创建的情况），再次尝试查找
		if strings.Contains(err.Error(), "already exists") {
			existingTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(expectedFullCodePath)
			if err == nil && existingTree != nil {
				logger.Infof(ctx, "[ServiceTreeService] 目录已存在（并发创建） - TreeID: %d, FullCodePath: %s", existingTree.ID, expectedFullCodePath)
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

	logger.Infof(ctx, "[ServiceTreeService] 目录创建成功 - TreeID: %d, DirectoryCode: %s, FullCodePath: %s",
		newTree.ID, meta.DirectoryCode, newTree.FullCodePath)

	return newTree, nil
}

// buildDirectoryTree 构建目录树结构（递归，包含函数）
// rootTree: 根目录节点
// allTrees: 所有目录节点（包括根目录和子目录）
// directoryFiles: 目录路径到文件快照的映射
// idToTree: ServiceTreeID 到 ServiceTree 的映射
// functionMap: 目录ID到函数列表的映射
func (s *ServiceTreeService) buildDirectoryTree(
	rootTree *model.ServiceTree,
	allTrees []*model.ServiceTree,
	directoryFiles map[string][]*model.FileSnapshot,
	idToTree map[int64]*model.ServiceTree,
	functionMap map[int64][]*model.ServiceTree,
) *dto.DirectoryTreeNode {
	return s.buildDirectoryTreeNode(rootTree, allTrees, directoryFiles, idToTree, functionMap)
}

// buildDirectoryTreeNode 递归构建目录树节点（包含函数）
func (s *ServiceTreeService) buildDirectoryTreeNode(
	tree *model.ServiceTree,
	allTrees []*model.ServiceTree,
	directoryFiles map[string][]*model.FileSnapshot,
	idToTree map[int64]*model.ServiceTree,
	functionMap map[int64][]*model.ServiceTree,
) *dto.DirectoryTreeNode {
	// 构建文件列表（init_.go 已在 app-runtime 层过滤，这里不需要再过滤）
	files := make([]*dto.FileSnapshotInfo, 0)
	if fileSnapshots, exists := directoryFiles[tree.FullCodePath]; exists {
		for _, file := range fileSnapshots {
			files = append(files, &dto.FileSnapshotInfo{
				FileName:     file.FileName,
				RelativePath: file.RelativePath,
				Content:      file.Content,
				FileType:     file.FileType,
				FileVersion:  file.FileVersion,
			})
		}
	}

	// 构建函数列表
	functions := make([]*dto.HubFunctionInfo, 0)
	if functionList, exists := functionMap[tree.ID]; exists {
		for _, fn := range functionList {
			functions = append(functions, &dto.HubFunctionInfo{
				ID:           fn.ID,
				Name:         fn.Name,
				Code:         fn.Code,
				FullCodePath: fn.FullCodePath,
				Description:  fn.Description,
				TemplateType: fn.TemplateType,
				Tags:         fn.GetTagsSlice(),
				RefID:        fn.RefID,
				Version:      fn.Version,
				VersionNum:   fn.VersionNum,
			})
		}
	}

	// 查找所有直接子目录
	subdirectories := make([]*dto.DirectoryTreeNode, 0)
	for _, childTree := range allTrees {
		if childTree.ParentID == tree.ID {
			// 递归构建子目录节点
			childNode := s.buildDirectoryTreeNode(childTree, allTrees, directoryFiles, idToTree, functionMap)
			subdirectories = append(subdirectories, childNode)
		}
	}

	return &dto.DirectoryTreeNode{
		Type:           "package", // DirectoryTreeNode 始终是 package 类型
		Name:           tree.Name, // 使用 ServiceTree 的 Name 字段（中文显示名称）
		Code:           tree.Code, // 使用 ServiceTree 的 Code 字段（英文标识）
		Path:           tree.FullCodePath,
		Files:          files,
		Functions:      functions,
		Subdirectories: subdirectories,
	}
}

// HubLinkInfo Hub 链接信息
type HubLinkInfo struct {
	Host         string // Hub 主机地址（如 hub.example.com:8080）
	FullCodePath string // 目录完整路径（如 /user/app/plugins/cashier）
	Version      string // 版本号（可选）
}

// ParseHubLink 解析 Hub 链接
// 格式：hub://{host}/{full_code_path} 或 hub://{host}/{full_code_path}@v1.0.0
// 例如：hub://hub.example.com/luobei/demo/plugins/cashier 或 hub://hub.example.com/luobei/demo/plugins/cashier@v1.0.0
func ParseHubLink(hubLink string) (*HubLinkInfo, error) {
	// 移除协议前缀
	if !strings.HasPrefix(hubLink, "hub://") {
		return nil, fmt.Errorf("无效的 Hub 链接格式，必须以 hub:// 开头")
	}

	link := strings.TrimPrefix(hubLink, "hub://")

	// 解析版本号（可选）
	var version string
	if idx := strings.LastIndex(link, "@"); idx != -1 {
		version = link[idx+1:]
		link = link[:idx]
	}

	// 解析主机和 full-code-path
	parts := strings.SplitN(link, "/", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("无效的 Hub 链接格式，缺少 full-code-path")
	}

	host := parts[0]
	fullCodePath := "/" + parts[1] // 确保以 / 开头

	// 验证 full-code-path 格式（应该至少包含 /user/app/...）
	pathParts := strings.Split(strings.Trim(fullCodePath, "/"), "/")
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("无效的 full-code-path 格式，应该至少包含 /user/app/...")
	}

	return &HubLinkInfo{
		Host:         host,
		FullCodePath: fullCodePath,
		Version:      version,
	}, nil
}

// PullDirectoryFromHub 从 Hub 拉取目录到工作空间（类似 git pull）
func (s *ServiceTreeService) PullDirectoryFromHub(ctx context.Context, req *dto.PullDirectoryFromHubReq) (*dto.PullDirectoryFromHubResp, error) {
	// 1. 解析 Hub 链接
	hubLinkInfo, err := ParseHubLink(req.HubLink)
	if err != nil {
		return nil, fmt.Errorf("解析 Hub 链接失败: %w", err)
	}

	logger.Infof(ctx, "[PullDirectoryFromHub] 解析 Hub 链接成功: host=%s, fullCodePath=%s, version=%s",
		hubLinkInfo.Host, hubLinkInfo.FullCodePath, hubLinkInfo.Version)

	// 2. 从 Hub 获取目录详情（包含目录树和文件内容，如果指定了版本号则查询指定版本）
	hubDetail, err := apicall.GetHubDirectoryDetailFromHost(ctx, &dto.GetHubDirectoryDetailFromHostReq{
		Host:         hubLinkInfo.Host,
		FullCodePath: hubLinkInfo.FullCodePath,
		Version:      hubLinkInfo.Version,
		IncludeTree:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("获取 Hub 目录详情失败: %w", err)
	}

	// 3. 如果指定了版本号，验证版本是否匹配
	if hubLinkInfo.Version != "" && hubDetail.Version != hubLinkInfo.Version {
		return nil, fmt.Errorf("版本不匹配：请求版本 %s，实际版本 %s", hubLinkInfo.Version, hubDetail.Version)
	}

	// 4. 获取目标应用信息
	targetApp, err := s.appRepo.GetAppByUserName(req.TargetUser, req.TargetApp)
	if err != nil {
		return nil, fmt.Errorf("获取目标应用失败: %w", err)
	}

	// 5. 确定目标目录路径
	targetPath := req.TargetDirectoryPath
	if targetPath == "" {
		targetPath = fmt.Sprintf("/%s/%s", targetApp.User, targetApp.Code)
	}

	// 6. 从 DirectoryTree 构建批量创建请求
	if hubDetail.DirectoryTree == nil {
		return nil, fmt.Errorf("Hub 目录树为空")
	}

	// 6.1 构建目录项列表
	directoryItems := make([]*dto.DirectoryTreeItem, 0)
	fileItems := make([]*dto.DirectoryTreeItem, 0)

	// 递归遍历目录树，构建目录和文件项
	s.buildItemsFromTree(hubDetail.DirectoryTree, targetPath, &directoryItems, &fileItems)

	// 7. 先批量创建目录
	if len(directoryItems) > 0 {
		batchCreateReq := &dto.BatchCreateDirectoryTreeReq{
			User:  req.TargetUser,
			App:   req.TargetApp,
			Items: directoryItems,
		}

		batchCreateResp, err := s.BatchCreateDirectoryTree(ctx, batchCreateReq)
		if err != nil {
			return nil, fmt.Errorf("批量创建目录失败: %w", err)
		}

		logger.Infof(ctx, "[PullDirectoryFromHub] 批量创建目录完成: directoryCount=%d", batchCreateResp.DirectoryCount)
	}

	// 8. 再批量写文件（会编译并返回 diff）
	if len(fileItems) > 0 {
		batchWriteReq := &dto.BatchWriteFilesReq{
			User:  req.TargetUser,
			App:   req.TargetApp,
			Files: fileItems,
		}

		batchWriteResp, err := s.BatchWriteFiles(ctx, batchWriteReq)
		if err != nil {
			return nil, fmt.Errorf("批量写文件失败: %w", err)
		}
		logger.Infof(ctx, "[PullDirectoryFromHub] 批量写文件完成: fileCount=%d", batchWriteResp.FileCount)
	}

	// 9. 获取根目录的 ServiceTree ID（用于建立双向绑定）
	// 直接使用 Code 字段
	rootDirPath := targetPath
	if hubDetail.DirectoryTree != nil && hubDetail.DirectoryTree.Code != "" {
		rootDirPath = fmt.Sprintf("%s/%s", targetPath, hubDetail.DirectoryTree.Code)
	}
	rootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(rootDirPath)
	if err != nil {
		logger.Warnf(ctx, "[PullDirectoryFromHub] 获取根目录 ServiceTree 失败: path=%s, error=%v", targetPath, err)
	}

	// 10. 建立双向绑定：更新根目录节点的 HubDirectoryID 和版本信息
	if rootTree != nil && hubDetail.ID > 0 {
		rootTree.HubDirectoryID = hubDetail.ID
		rootTree.HubVersion = hubDetail.Version
		rootTree.HubVersionNum = hubDetail.VersionNum
		if err := s.serviceTreeRepo.UpdateServiceTree(rootTree); err != nil {
			logger.Warnf(ctx, "[PullDirectoryFromHub] 更新ServiceTree的Hub信息失败: treeID=%d, hubDirectoryID=%d, hubVersion=%s, error=%v",
				rootTree.ID, hubDetail.ID, hubDetail.Version, err)
		} else {
			logger.Infof(ctx, "[PullDirectoryFromHub] 成功建立双向绑定: treeID=%d, hubDirectoryID=%d, hubVersion=%s", rootTree.ID, hubDetail.ID, hubDetail.Version)
		}
	}

	return &dto.PullDirectoryFromHubResp{
		Message:             fmt.Sprintf("从 Hub 安装目录成功，共安装 %d 个目录，%d 个文件", len(directoryItems), len(fileItems)),
		DirectoryCount:      len(directoryItems),
		FileCount:           len(fileItems),
		TargetDirectoryPath: rootDirPath,
		ServiceTreeID: func() int64 {
			if rootTree != nil {
				return rootTree.ID
			} else {
				return 0
			}
		}(),
		HubDirectoryID:   hubDetail.ID,
		HubDirectoryName: hubDetail.Name,
		HubVersion:       hubDetail.Version,
		HubVersionNum:    hubDetail.VersionNum,
	}, nil
}

// BatchWriteFiles 批量写文件（app-server 端，调用 runtime）
func (s *ServiceTreeService) BatchWriteFiles(ctx context.Context, req *dto.BatchWriteFilesReq) (*dto.BatchWriteFilesResp, error) {
	// 1. 获取应用信息
	app, err := s.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		return nil, fmt.Errorf("获取应用信息失败: %w", err)
	}

	// 2. 构建 runtime 请求
	runtimeReq := &dto.BatchWriteFilesRuntimeReq{
		User:  req.User,
		App:   req.App,
		Files: req.Files,
	}

	// 3. 调用 runtime 批量写文件
	runtimeResp, err := s.appRuntime.BatchWriteFiles(ctx, app.HostID, runtimeReq)
	if err != nil {
		return nil, fmt.Errorf("批量写文件失败: %w", err)
	}

	// 4. 处理 diff，更新 ServiceTree 的 function 节点（通过 processAPIDiff）
	if runtimeResp.Diff != nil {
		// 构建 UpdateAppReq 用于 processAPIDiff（兼容旧接口）
		updateAppReq := &dto.UpdateAppReq{
			User: req.User,
			App:  req.App,
		}
		// 调用 appService.processAPIDiff 处理 API 差异
		if err := s.appService.processAPIDiff(ctx, app.ID, runtimeResp.Diff, updateAppReq, 0, runtimeResp.GitCommitHash); err != nil {
			logger.Warnf(ctx, "[BatchWriteFiles] 处理 API diff 失败: %v", err)
			// 不返回错误，因为文件已经写入成功
		}
	}

	// 5. 更新数据库中的版本信息（基于 runtimeResp 返回的版本信息）
	if runtimeResp.NewVersion != "" {
		if err := s.appRepo.UpdateAppVersion(req.User, req.App, runtimeResp.NewVersion); err != nil {
			logger.Warnf(ctx, "[BatchWriteFiles] 更新应用版本失败: oldVersion=%s, newVersion=%s, error=%v",
				runtimeResp.OldVersion, runtimeResp.NewVersion, err)
			// 不返回错误，因为文件已经写入成功，只是版本更新失败
		} else {
			logger.Infof(ctx, "[BatchWriteFiles] 应用版本更新成功: oldVersion=%s, newVersion=%s",
				runtimeResp.OldVersion, runtimeResp.NewVersion)
		}
	}

	return &dto.BatchWriteFilesResp{
		FileCount:     runtimeResp.FileCount,
		WrittenPaths:  runtimeResp.WrittenPaths,
		Diff:          runtimeResp.Diff,
		OldVersion:    runtimeResp.OldVersion,
		NewVersion:    runtimeResp.NewVersion,
		GitCommitHash: runtimeResp.GitCommitHash,
	}, nil
}

// countFilesInTree 递归统计目录树中的文件数量（用于调试）
func (s *ServiceTreeService) countFilesInTree(node *dto.DirectoryTreeNode) int {
	count := len(node.Files)
	for _, subdir := range node.Subdirectories {
		count += s.countFilesInTree(subdir)
	}
	return count
}

// logDirectoryTree 递归打印目录树详细信息（用于调试）
func (s *ServiceTreeService) logDirectoryTree(ctx context.Context, node *dto.DirectoryTreeNode, level int) {
	indent := strings.Repeat("  ", level)
	logger.Infof(ctx, "%s[logDirectoryTree] 节点: Name=%s, Code=%s, Path=%s, Files数量=%d, Subdirectories数量=%d",
		indent, node.Name, node.Code, node.Path, len(node.Files), len(node.Subdirectories))

	// 打印文件详情
	for i, file := range node.Files {
		logger.Infof(ctx, "%s  [文件%d] FileName=%s, RelativePath=%s, FileType=%s, Content长度=%d",
			indent, i+1, file.FileName, file.RelativePath, file.FileType, len(file.Content))
	}

	// 递归打印子目录
	for i, subdir := range node.Subdirectories {
		logger.Infof(ctx, "%s  [子目录%d]", indent, i+1)
		s.logDirectoryTree(ctx, subdir, level+1)
	}
}

// buildItemsFromTree 递归构建目录和文件项列表
func (s *ServiceTreeService) buildItemsFromTree(
	node *dto.DirectoryTreeNode,
	targetBasePath string,
	directoryItems *[]*dto.DirectoryTreeItem,
	fileItems *[]*dto.DirectoryTreeItem,
) {
	// 直接使用 Code 字段（英文标识）
	dirCode := node.Code
	logger.Infof(context.Background(), "[buildItemsFromTree] 处理节点: Name=%s, Code=%s, Path=%s, Files数量=%d",
		node.Name, node.Code, node.Path, len(node.Files))

	// 如果 Code 为空，记录警告但不使用 fallback
	if dirCode == "" {
		logger.Warnf(context.Background(), "[buildItemsFromTree] ⚠️ Code 字段为空！Name=%s, Path=%s", node.Name, node.Path)
	}

	// 计算当前目录的目标路径（使用代码名称）
	currentTargetPath := fmt.Sprintf("%s/%s", targetBasePath, dirCode)

	// 添加目录项（使用代码名称作为 Name，但保留原始 Name 作为显示名称）
	*directoryItems = append(*directoryItems, &dto.DirectoryTreeItem{
		FullCodePath: currentTargetPath,
		Type:         "directory",
		Name:         dirCode, // 使用代码名称（从 Path 提取）
		Description:  "",      // Hub 目录树可能没有描述
		Tags:         "",      // Hub 目录树可能没有标签
	})

	// 添加文件项
	logger.Infof(context.Background(), "[buildItemsFromTree] 处理目录 %s，文件数量: %d", currentTargetPath, len(node.Files))
	if len(node.Files) == 0 {
		logger.Warnf(context.Background(), "[buildItemsFromTree] ⚠️ 目录 %s 没有文件！Name=%s, Code=%s, Path=%s", currentTargetPath, node.Name, node.Code, node.Path)
	}
	for i, file := range node.Files {
		logger.Infof(context.Background(), "[buildItemsFromTree] 处理文件[%d]: FileName=%s, RelativePath=%s, FileType=%s, Content长度=%d",
			i+1, file.FileName, file.RelativePath, file.FileType, len(file.Content))
		// 从 RelativePath 提取文件名（不含扩展名）
		fileName := file.FileName
		if fileName == "" {
			// 从 RelativePath 提取
			pathParts := strings.Split(file.RelativePath, "/")
			fileName = pathParts[len(pathParts)-1]
			if ext := strings.LastIndex(fileName, "."); ext != -1 {
				fileName = fileName[:ext]
			}
		}

		// 检查文件内容是否为空
		if file.Content == "" {
			logger.Warnf(context.Background(), "[buildItemsFromTree] 文件 %s 内容为空", fileName)
		}

		// FullCodePath 应该只包含目录路径，不包含文件名
		// BatchWriteFiles 会从 FullCodePath 提取 package 路径，从 FileName 获取文件名
		*fileItems = append(*fileItems, &dto.DirectoryTreeItem{
			FullCodePath: currentTargetPath, // 只包含目录路径，不包含文件名
			Type:         "file",
			FileName:     fileName,
			FileType:     file.FileType,
			Content:      file.Content, // 文件内容
			RelativePath: file.RelativePath,
		})
		logger.Infof(context.Background(), "[buildItemsFromTree] 添加文件: %s, 内容长度: %d", fileName, len(file.Content))
	}

	// 递归处理子目录
	for _, subdir := range node.Subdirectories {
		s.buildItemsFromTree(subdir, currentTargetPath, directoryItems, fileItems)
	}
}

// GetHubInfo 获取目录的 Hub 信息
func (s *ServiceTreeService) GetHubInfo(ctx context.Context, req *dto.GetHubInfoReq) (*dto.GetHubInfoResp, error) {
	// 1. 根据 FullCodePath 获取 ServiceTree 节点
	tree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.FullCodePath)
	if err != nil {
		return nil, fmt.Errorf("获取目录信息失败: %w", err)
	}

	// 2. 检查是否已发布到 Hub
	if tree.HubDirectoryID == 0 {
		return nil, fmt.Errorf("目录未发布到 Hub")
	}

	// 3. 调用 Hub API 获取目录信息（用于获取 URL 和发布时间）
	// 获取 Hub 目录详情（直接传 ctx，内部会提取 token、trace_id 等）
	hubDetail, err := apicall.GetHubDirectoryDetail(ctx, &dto.GetHubDirectoryDetailReq{
		FullCodePath: req.FullCodePath,
		Version:      "",
		IncludeTree:  false,
	})
	if err != nil {
		logger.Warnf(ctx, "[GetHubInfo] 获取 Hub 目录详情失败: fullCodePath=%s, error=%v", req.FullCodePath, err)
		// 即使获取详情失败，也返回基本信息
		return &dto.GetHubInfoResp{
			HubDirectoryID:  tree.HubDirectoryID,
			HubDirectoryURL: fmt.Sprintf("/hub/directory/%d", tree.HubDirectoryID),
			PublishedAt:     "", // 无法获取发布时间
		}, nil
	}

	// 4. 构建 Hub URL（使用 full-code-path）
	hubURL := fmt.Sprintf("/hub/directory/%s", req.FullCodePath)

	return &dto.GetHubInfoResp{
		HubDirectoryID:  hubDetail.ID,
		HubDirectoryURL: hubURL,
		PublishedAt:     hubDetail.PublishedAt,
	}, nil
}

// SearchFunctions 搜索函数
func (s *ServiceTreeService) SearchFunctions(ctx context.Context, req *dto.SearchFunctionsReq) (*dto.SearchFunctionsResp, error) {
	// 调用 Repository 搜索函数
	functions, total, err := s.serviceTreeRepo.SearchFunctions(
		req.User,
		req.App,
		req.Keyword,
		req.TemplateType,
		req.Page,
		req.PageSize,
	)
	if err != nil {
		return nil, fmt.Errorf("搜索函数失败: %w", err)
	}

	// 转换为响应格式
	functionResults := make([]*dto.FunctionSearchResult, 0, len(functions))
	for _, fn := range functions {
		result := &dto.FunctionSearchResult{
			ID:           fn.ID,
			Name:         fn.Name,
			Code:         fn.Code,
			FullCodePath: fn.FullCodePath,
			Description:  fn.Description,
			TemplateType: fn.TemplateType,
			AppID:        fn.AppID,
		}

		// 如果预加载了 App，填充 AppUser 和 AppCode
		if fn.App != nil {
			result.AppUser = fn.App.User
			result.AppCode = fn.App.Code
		}

		functionResults = append(functionResults, result)
	}

	return &dto.SearchFunctionsResp{
		Functions: functionResults,
		Total:     total,
		Page:      req.Page,
		PageSize:  req.PageSize,
	}, nil
}
