package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type serviceTreeQueryView struct {
	serviceTreeRepo   *repository.ServiceTreeRepository
	appRepo           *repository.AppRepository
	permissionService *PermissionService
}

func newServiceTreeQueryView(
	serviceTreeRepo *repository.ServiceTreeRepository,
	appRepo *repository.AppRepository,
	permissionService *PermissionService,
) *serviceTreeQueryView {
	return &serviceTreeQueryView{
		serviceTreeRepo:   serviceTreeRepo,
		appRepo:           appRepo,
		permissionService: permissionService,
	}
}

func (q *serviceTreeQueryView) getServiceTreeByAppModel(ctx context.Context, appModel *model.App, nodeType string) ([]*dto.GetServiceTreeResp, error) {
	var trees []*model.ServiceTree
	var err error
	if nodeType != "" {
		trees, err = q.serviceTreeRepo.BuildServiceTreeByType(appModel.ID, nodeType)
	} else {
		trees, err = q.serviceTreeRepo.BuildServiceTree(appModel.ID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to build service tree: %w", err)
	}

	if len(trees) == 0 {
		return nil, fmt.Errorf("服务树为空，app_id=%d，请检查根节点是否已创建", appModel.ID)
	}

	if !trees[0].IsRoot() || trees[0].RefID != appModel.ID {
		return nil, fmt.Errorf("根节点无效，app_id=%d, full_code_path=%s, ref_id=%d", appModel.ID, trees[0].FullCodePath, trees[0].RefID)
	}

	rootNode := trees[0]
	logger.Debugf(ctx, "[ServiceTreeService] 获取服务树成功: root_id=%d, app_id=%d, full_code_path=%s, children_count=%d",
		rootNode.ID, appModel.ID, rootNode.FullCodePath, len(rootNode.Children))

	var permissionsMap map[string]map[string]bool
	var isAdmin bool
	username := contextx.GetRequestUser(ctx)

	if !appModel.PermissionEnforced {
		permissionsMap = q.buildAllAdminPermissionsMap(trees)
		isAdmin = true
		logger.Debugf(ctx, "[ServiceTreeService] 工作空间未启用权限管控，按全量可访问返回服务树: app_id=%d", appModel.ID)
	} else if username != "" && appModel.ID > 0 && q.permissionService != nil {
		if appModel.IsOwnerOrAdmin(username) {
			isAdmin = true
			permissionsMap = q.buildAllAdminPermissionsMap(trees)
			logger.Debugf(ctx, "[ServiceTreeService] 用户 %s 是工作空间 owner/创建者/管理员，直接返回所有权限", username)
		} else {
			permsMap, err := q.calculatePermissions(ctx, appModel.User, appModel.Code, trees, appModel.Admins, username)
			if err != nil {
				logger.Warnf(ctx, "[ServiceTreeService] 计算权限失败: app_id=%d, error=%v，继续返回服务树（无权限信息）", appModel.ID, err)
			} else {
				permissionsMap = permsMap
				logger.Debugf(ctx, "[ServiceTreeService] 权限计算完成: app_id=%d, username=%s, isAdmin=%v", appModel.ID, username, isAdmin)
			}
		}
	}

	rootResp := q.convertToGetServiceTreeResp(rootNode, permissionsMap, isAdmin)

	if appModel.PermissionEnforced && appModel.ShowOnlyPermitted && !isAdmin && permissionsMap != nil {
		rootResp = q.filterTreeByPermission(rootResp)
		logger.Debugf(ctx, "[ServiceTreeService] 已按权限过滤服务树: app_id=%d, username=%s", appModel.ID, username)
	}

	logger.Debugf(ctx, "[ServiceTreeService] 服务树转换完成: root_id=%d, children_count=%d",
		rootNode.ID, len(rootResp.Children))

	return []*dto.GetServiceTreeResp{rootResp}, nil
}

func (q *serviceTreeQueryView) filterTreeByPermission(rootResp *dto.GetServiceTreeResp) *dto.GetServiceTreeResp {
	rootResp.Children = q.collectVisibleChildren(rootResp.Children)
	return rootResp
}

func (q *serviceTreeQueryView) collectVisibleChildren(children []*dto.GetServiceTreeResp) []*dto.GetServiceTreeResp {
	out := make([]*dto.GetServiceTreeResp, 0, len(children))
	for _, child := range children {
		out = append(out, q.collectVisibleFromNode(child)...)
	}
	return out
}

func (q *serviceTreeQueryView) collectVisibleFromNode(node *dto.GetServiceTreeResp) []*dto.GetServiceTreeResp {
	hasAnyTrue := false
	if node.Permissions != nil {
		for _, v := range node.Permissions {
			if v {
				hasAnyTrue = true
				break
			}
		}
	}
	if !hasAnyTrue {
		return q.collectVisibleChildren(node.Children)
	}
	node.Children = q.collectVisibleChildren(node.Children)
	return []*dto.GetServiceTreeResp{node}
}

func (q *serviceTreeQueryView) GetAppWithServiceTree(ctx context.Context, req *dto.GetAppWithServiceTreeReq) (*dto.GetAppWithServiceTreeResp, error) {
	user, appCode, err := resolveUserAppFromResourcePath(req.ResourcePath, req.User, req.App)
	if err != nil {
		return nil, err
	}
	req.User = user
	req.App = appCode

	appModel, err := q.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("应用不存在: %s/%s", req.User, req.App)
		}
		return nil, fmt.Errorf("获取应用信息失败: %w", err)
	}

	appInfo := dto.AppInfo{
		ID:                 appModel.ID,
		User:               appModel.User,
		Code:               appModel.Code,
		Name:               appModel.Name,
		Status:             appModel.Status,
		Version:            appModel.Version,
		NatsID:             appModel.NatsID,
		HostID:             appModel.HostID,
		IsPublic:           appModel.IsPublic,
		Admins:             appModel.Admins,
		Type:               int(appModel.Type),
		ShowOnlyPermitted:  appModel.ShowOnlyPermitted,
		PermissionEnforced: appModel.PermissionEnforced,
		CreatedAt:          time.Time(appModel.CreatedAt).Format("2006-01-02 15:04:05"),
		UpdatedAt:          time.Time(appModel.UpdatedAt).Format("2006-01-02 15:04:05"),
	}

	serviceTreeResp, err := q.getServiceTreeByAppModel(ctx, appModel, req.Type)
	if err != nil {
		return nil, fmt.Errorf("获取服务目录树失败: %w", err)
	}

	expandedKeys := q.calculateExpandedKeys(serviceTreeResp)

	return &dto.GetAppWithServiceTreeResp{
		App:          appInfo,
		ServiceTree:  serviceTreeResp,
		ExpandedKeys: expandedKeys,
	}, nil
}

func (q *serviceTreeQueryView) GetServiceTreeDetail(ctx context.Context, req *dto.GetServiceTreeDetailReq) (*dto.GetServiceTreeDetailResp, error) {
	var tree *model.ServiceTree
	var err error

	if req.ID > 0 {
		tree, err = q.serviceTreeRepo.GetServiceTreeByID(req.ID)
	} else if req.FullCodePath != "" {
		tree, err = q.serviceTreeRepo.GetServiceTreeByFullPath(req.FullCodePath)
	} else {
		return nil, fmt.Errorf("必须提供 ID 或 full_code_path")
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("服务目录不存在")
		}
		return nil, fmt.Errorf("获取服务目录失败: %w", err)
	}

	resp := &dto.GetServiceTreeDetailResp{
		ID:              tree.ID,
		Name:            tree.Name,
		Code:            tree.Code,
		Type:            tree.Type,
		Description:     tree.Description,
		Tags:            tree.Tags,
		AppID:           tree.AppID,
		RefID:           tree.RefID,
		FullCodePath:    tree.FullCodePath,
		TemplateType:    tree.TemplateType,
		Version:         tree.Version,
		VersionNum:      tree.VersionNum,
		HubFullCodePath: tree.HubFullCodePath,
		HubVersionNum:   tree.HubVersionNum,
		RunCount:        tree.RunCount,
	}

	username := contextx.GetRequestUser(ctx)
	appModel, appErr := q.appRepo.GetAppByID(tree.AppID)
	if appErr != nil {
		logger.Warnf(ctx, "[ServiceTreeService] 查询工作空间失败: app_id=%d, error=%v，继续按权限记录计算", tree.AppID, appErr)
	}

	if appModel != nil && !appModel.PermissionEnforced {
		resp.Permissions = initializeNodePermissions(permissionActionsForNode(tree.Type, tree.TemplateType), nil)
		grantAppAdminPermission(resp.Permissions)
	} else if username != "" && tree.FullCodePath != "" && q.permissionService != nil {
		permCtx, err := q.loadWorkspacePermissionContext(ctx, tree.FullCodePath)
		if err != nil {
			logger.Warnf(ctx, "[ServiceTreeService] 查询权限失败: fullCodePath=%s, error=%v，继续返回详情（无权限信息）", tree.FullCodePath, err)
			resp.Permissions = make(map[string]bool)
		} else {
			resp.Permissions = q.buildQueryNodePermissions(tree.Type, tree.TemplateType, tree.FullCodePath, permCtx, username, tree.Admins, tree.CreatedBy)
		}
	} else {
		resp.Permissions = make(map[string]bool)
	}

	return resp, nil
}

func (q *serviceTreeQueryView) convertToGetServiceTreeResp(tree *model.ServiceTree, permissionsMap map[string]map[string]bool, isAdmin bool) *dto.GetServiceTreeResp {
	resp := &dto.GetServiceTreeResp{
		ID:              tree.ID,
		Name:            tree.Name,
		Code:            tree.Code,
		RefID:           tree.RefID,
		Type:            tree.Type,
		Description:     tree.Description,
		Tags:            tree.Tags,
		Admins:          tree.Admins,
		PendingCount:    tree.PendingCount,
		Owner:           tree.CreatedBy,
		AppID:           tree.AppID,
		FullCodePath:    tree.FullCodePath,
		TemplateType:    tree.TemplateType,
		Version:         tree.Version,
		VersionNum:      tree.VersionNum,
		HubFullCodePath: tree.HubFullCodePath,
		HubVersionNum:   tree.HubVersionNum,
		RunCount:        tree.RunCount,
		IsAdmin:         isAdmin,
	}

	if tree.FullCodePath != "" && permissionsMap != nil {
		if nodePerms, ok := permissionsMap[tree.FullCodePath]; ok {
			resp.Permissions = nodePerms
		} else {
			resp.Permissions = make(map[string]bool)
		}
	} else {
		resp.Permissions = make(map[string]bool)
	}

	if len(tree.Children) > 0 {
		for _, child := range tree.Children {
			childResp := q.convertToGetServiceTreeResp(child, permissionsMap, isAdmin)
			resp.Children = append(resp.Children, childResp)
		}
	}

	if tree.Type == model.ServiceTreeTypePackage {
		resp.HasFunction = q.hasFunctionInDirectChildren(tree)
	}

	return resp
}

func (q *serviceTreeQueryView) calculateExpandedKeys(trees []*dto.GetServiceTreeResp) []int64 {
	expandedKeysMap := make(map[int64]bool)

	for _, tree := range trees {
		segments := strings.Split(strings.Trim(tree.FullCodePath, "/"), "/")
		if tree.Type == model.ServiceTreeTypePackage && len(segments) == 2 {
			expandedKeysMap[tree.ID] = true
		}
	}

	var findNodesWithPending func(nodes []*dto.GetServiceTreeResp, parentPath []int64)
	findNodesWithPending = func(nodes []*dto.GetServiceTreeResp, parentPath []int64) {
		for _, node := range nodes {
			currentPath := append(parentPath, node.ID)

			if node.PendingCount > 0 {
				for _, id := range currentPath {
					expandedKeysMap[id] = true
				}
			}

			if len(node.Children) > 0 {
				findNodesWithPending(node.Children, currentPath)
			}
		}
	}

	findNodesWithPending(trees, []int64{})

	expandedKeys := make([]int64, 0, len(expandedKeysMap))
	for id := range expandedKeysMap {
		expandedKeys = append(expandedKeys, id)
	}

	return expandedKeys
}

func (q *serviceTreeQueryView) hasFunctionInDirectChildren(node *model.ServiceTree) bool {
	if node == nil {
		return false
	}
	for _, child := range node.Children {
		if child.Type == model.ServiceTreeTypeFunction {
			return true
		}
	}
	return false
}
