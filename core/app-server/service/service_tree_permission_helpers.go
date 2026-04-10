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

var appAdminActionCode = permission.BuildActionCode(permission.ResourceTypeApp, "admin")

type workspacePermissionContext struct {
	workspaceUser  string
	workspaceApp   string
	rawPermissions map[string]map[string]bool
	admins         string
}

func isWorkspaceAdmin(username, admins string) bool {
	if username == "" || admins == "" {
		return false
	}

	for _, admin := range strings.Split(admins, ",") {
		if strings.TrimSpace(admin) == username {
			return true
		}
	}

	return false
}

func buildRawPermissions(records []dto.PermissionRecord) map[string]map[string]bool {
	rawPermissions := make(map[string]map[string]bool)
	for _, record := range records {
		resourcePath := record.Resource
		action := record.Action

		if rawPermissions[resourcePath] == nil {
			rawPermissions[resourcePath] = make(map[string]bool)
		}
		rawPermissions[resourcePath][action] = true
	}

	return rawPermissions
}

func getPermissionActionsForNodeImpl(nodeType string, templateType string) []string {
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

func copyPermissionMap(src map[string]bool) map[string]bool {
	dst := make(map[string]bool)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func initializeNodePermissions(actions []string, rawPerms map[string]bool) map[string]bool {
	nodePerms := make(map[string]bool, len(actions)+1)
	for _, action := range actions {
		nodePerms[action] = rawPerms != nil && rawPerms[action]
	}
	return nodePerms
}

func grantAllNodePermissions(nodePerms map[string]bool) {
	for actionCode := range nodePerms {
		nodePerms[actionCode] = true
	}
}

func grantAppAdminPermission(nodePerms map[string]bool) {
	grantAllNodePermissions(nodePerms)
	nodePerms[appAdminActionCode] = true
}

func hasAppAdminPermission(perms map[string]bool) bool {
	return perms != nil && perms[appAdminActionCode]
}

func mergePermissionMaps(base map[string]bool, extra map[string]bool) map[string]bool {
	merged := make(map[string]bool)
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range extra {
		merged[k] = v
	}
	return merged
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
	}

	return permCtx, nil
}

func (q *serviceTreeQueryView) buildQueryNodePermissions(
	nodeType string,
	templateType string,
	fullCodePath string,
	permCtx *workspacePermissionContext,
	username string,
) map[string]bool {
	actions := getPermissionActionsForNodeImpl(nodeType, templateType)
	nodePerms := initializeNodePermissions(actions, permCtx.rawPermissions[fullCodePath])
	if len(actions) == 0 {
		return nodePerms
	}

	for _, parentPath := range permission.GetParentPaths(fullCodePath) {
		if parentPerms, ok := permCtx.rawPermissions[parentPath]; ok {
			applyPermissionInheritanceImpl(nodeType, templateType, parentPerms, nodePerms)
		}
	}

	if isWorkspaceAdmin(username, permCtx.admins) {
		grantAppAdminPermission(nodePerms)
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

func (q *serviceTreeQueryView) buildAllAdminPermissionsMap(trees []*model.ServiceTree) map[string]map[string]bool {
	permissionsMap := make(map[string]map[string]bool)

	var setAllPermissions func(nodes []*model.ServiceTree)
	setAllPermissions = func(nodes []*model.ServiceTree) {
		for _, node := range nodes {
			actions := getPermissionActionsForNodeImpl(node.Type, node.TemplateType)
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

func calculatePermissionsImpl(
	q *serviceTreeQueryView,
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
			actions := getPermissionActionsForNodeImpl(node.Type, node.TemplateType)
			if len(actions) == 0 {
				if len(node.Children) > 0 {
					calculatePermissionsRecursive(node.Children, inheritedPerms)
				}
				continue
			}

			nodePerms := initializeNodePermissions(actions, rawPermissions[node.FullCodePath])

			if inheritedPerms != nil {
				applyPermissionInheritanceImpl(node.Type, node.TemplateType, inheritedPerms, nodePerms)
			}

			currentNodePerms := copyPermissionMap(rawPermissions[node.FullCodePath])

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

func applyPermissionInheritanceImpl(
	nodeType string,
	templateType string,
	parentPerms map[string]bool,
	nodePerms map[string]bool,
) {
	resourceType := permission.GetResourceType(nodeType, templateType)
	if resourceType == "" {
		return
	}

	for parentActionCode := range parentPerms {
		parentResourceType, actionType, ok := permission.ParseActionCode(parentActionCode)
		if !ok {
			continue
		}

		if parentResourceType == permission.ResourceTypeDirectory {
			if actionType == "admin" {
				for actionCode := range nodePerms {
					nodePerms[actionCode] = true
				}
				return
			}

			childActionCode := permission.BuildActionCode(resourceType, actionType)
			if _, exists := nodePerms[childActionCode]; exists {
				nodePerms[childActionCode] = true
			}
			continue
		}

		if parentResourceType == permission.ResourceTypeApp && actionType == "admin" {
			for actionCode := range nodePerms {
				nodePerms[actionCode] = true
			}
			return
		}
	}
}
