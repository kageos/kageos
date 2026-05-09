package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/permission"
)

type workspacePermissionContext struct {
	workspaceUser  string
	workspaceApp   string
	rawPermissions map[string]map[string]bool
	admins         string
	appModel       *model.App
}

func (q *serviceTreeQueryView) loadWorkspacePermissionContext(ctx context.Context, fullCodePath string) (*workspacePermissionContext, error) {
	if q.permissionService == nil || q.permissionService.permissionBackend() == nil {
		return nil, fmt.Errorf("权限服务未初始化")
	}

	_, workspaceUser, workspaceApp := permission.ParseFullCodePath(fullCodePath)
	if workspaceUser == "" || workspaceApp == "" {
		return nil, fmt.Errorf("无法从 FullCodePath 解析 user 和 app: %s", fullCodePath)
	}

	permReq := &dto.GetWorkspacePermissionsReq{
		ResourcePath: permission.GetAppPath(fullCodePath),
	}
	permResp, err := q.permissionService.GetWorkspacePermissions(ctx, permReq)
	if err != nil {
		return nil, err
	}

	permCtx := &workspacePermissionContext{
		workspaceUser:  workspaceUser,
		workspaceApp:   workspaceApp,
		rawPermissions: make(map[string]map[string]bool),
	}
	if permResp != nil {
		permCtx.rawPermissions = buildRawPermissions(permResp.Records)
	}

	appModel, err := q.appRepo.GetAppByUserName(workspaceUser, workspaceApp)
	if err == nil && appModel != nil {
		permCtx.admins = appModel.Admins
		permCtx.appModel = appModel
	}

	return permCtx, nil
}

func (q *serviceTreeQueryView) buildQueryNodePermissions(
	nodeType string,
	templateType string,
	fullCodePath string,
	permCtx *workspacePermissionContext,
	username string,
	nodeAdmins string,
	nodeCreatedBy string,
) map[string]bool {
	actions := permissionActionsForNode(nodeType, templateType)
	nodePerms := initializeNodePermissions(actions, permCtx.rawPermissions[fullCodePath])
	if len(actions) == 0 {
		return nodePerms
	}

	for _, parentPath := range permission.GetParentPaths(fullCodePath) {
		if parentPerms, ok := permCtx.rawPermissions[parentPath]; ok {
			applyPermissionInheritance(nodeType, templateType, parentPerms, nodePerms)
		}
	}

	if isWorkspaceAdmin(username, permCtx.admins) || (permCtx.appModel != nil && permCtx.appModel.IsOwnerOrAdmin(username)) {
		grantAppAdminPermission(nodePerms)
		return nodePerms
	}

	if isUserNodeAdmin(username, nodeAdmins, nodeCreatedBy) || q.hasServiceTreeOwnerOrAdmin(username, fullCodePath) {
		grantAllNodePermissions(nodePerms)
		return nodePerms
	}

	appPath := permission.GetAppPath(fullCodePath)
	if appPath != "" {
		if appPerms, ok := permCtx.rawPermissions[appPath]; ok && hasAppAdminPermission(appPerms) {
			grantAppAdminPermission(nodePerms)
		}
	}

	return nodePerms
}

func (q *serviceTreeQueryView) hasServiceTreeOwnerOrAdmin(username string, fullCodePath string) bool {
	if q == nil || q.serviceTreeRepo == nil {
		return false
	}

	username = strings.TrimSpace(username)
	if username == "" {
		return false
	}

	paths := append([]string{fullCodePath}, permission.GetParentPaths(fullCodePath)...)
	nodes, err := q.serviceTreeRepo.GetServiceTreeByFullPaths(paths)
	if err != nil {
		return false
	}

	for _, path := range paths {
		node := nodes[path]
		if node != nil && node.IsOwnerOrAdmin(username) {
			return true
		}
	}

	return false
}

func (q *serviceTreeQueryView) buildAllAdminPermissionsMap(trees []*model.ServiceTree) map[string]map[string]bool {
	permissionsMap := make(map[string]map[string]bool)

	var setAllPermissions func(nodes []*model.ServiceTree)
	setAllPermissions = func(nodes []*model.ServiceTree) {
		for _, node := range nodes {
			actions := permissionActionsForNode(node.Type, node.TemplateType)
			nodePerms := initializeNodePermissions(actions, nil)
			grantAppAdminPermission(nodePerms)
			permissionsMap[node.FullCodePath] = nodePerms

			if len(node.Children) > 0 {
				setAllPermissions(node.Children)
			}
		}
	}

	setAllPermissions(trees)
	return permissionsMap
}

func (q *serviceTreeQueryView) calculatePermissions(
	ctx context.Context,
	user string,
	app string,
	trees []*model.ServiceTree,
	admins string,
	username string,
) (map[string]map[string]bool, error) {
	if isWorkspaceAdmin(username, admins) {
		logger.Debugf(ctx, "[ServiceTreeService] 用户 %s 是工作空间管理员，直接返回所有权限", username)
		return q.buildAllAdminPermissionsMap(trees), nil
	}

	if q.permissionService == nil || q.permissionService.permissionBackend() == nil {
		logger.Warnf(ctx, "[ServiceTreeService] 权限服务未初始化，返回空权限")
		return make(map[string]map[string]bool), nil
	}

	permReq := &dto.GetWorkspacePermissionsReq{
		ResourcePath: fmt.Sprintf("/%s/%s", user, app),
	}
	permResp, err := q.permissionService.GetWorkspacePermissions(ctx, permReq)
	if err != nil {
		return nil, fmt.Errorf("查询权限失败: %w", err)
	}

	if permResp == nil || len(permResp.Records) == 0 {
		logger.Debugf(ctx, "[ServiceTreeService] 没有权限记录: user=%s, app=%s", user, app)
		return make(map[string]map[string]bool), nil
	}

	rawPermissions := buildRawPermissions(permResp.Records)
	permissionsMap := make(map[string]map[string]bool)

	var appPerms map[string]bool
	if len(trees) > 0 && trees[0].FullCodePath != "" {
		appPath := permission.GetAppPath(trees[0].FullCodePath)
		if appPath != "" {
			appPerms = rawPermissions[appPath]
		}
	}

	var calculatePermissionsRecursive func(nodes []*model.ServiceTree, inheritedPerms map[string]bool)
	calculatePermissionsRecursive = func(nodes []*model.ServiceTree, inheritedPerms map[string]bool) {
		for _, node := range nodes {
			actions := permissionActionsForNode(node.Type, node.TemplateType)
			if len(actions) == 0 {
				if len(node.Children) > 0 {
					calculatePermissionsRecursive(node.Children, inheritedPerms)
				}
				continue
			}

			nodePerms := initializeNodePermissions(actions, rawPermissions[node.FullCodePath])

			if inheritedPerms != nil {
				applyPermissionInheritance(node.Type, node.TemplateType, inheritedPerms, nodePerms)
			}

			currentNodePerms := copyPermissionMap(rawPermissions[node.FullCodePath])

			if node.IsOwnerOrAdmin(username) {
				grantAllNodePermissions(nodePerms)
				currentNodePerms[permission.BuildActionCode(permission.ResourceTypeDirectory, permission.ActionAdmin)] = true
			}

			if hasAppAdminPermission(appPerms) {
				grantAppAdminPermission(nodePerms)
				currentNodePerms[appAdminActionCode] = true
			}

			permissionsMap[node.FullCodePath] = nodePerms

			childInheritedPerms := mergePermissionMaps(inheritedPerms, currentNodePerms)
			if len(node.Children) > 0 {
				calculatePermissionsRecursive(node.Children, childInheritedPerms)
			}
		}
	}

	calculatePermissionsRecursive(trees, nil)

	logger.Debugf(ctx, "[ServiceTreeService] 权限计算完成（支持角色权限）: 节点数=%d, 权限节点数=%d", len(trees), len(permissionsMap))
	return permissionsMap, nil
}
