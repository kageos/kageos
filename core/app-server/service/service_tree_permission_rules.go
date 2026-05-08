package service

import (
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/permission"
)

var appAdminActionCode = permission.BuildActionCode(permission.ResourceTypeApp, permission.ActionAdmin)

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

func permissionActionsForNode(nodeType string, templateType string) []string {
	var nodeTypeStr string
	if nodeType == model.ServiceTreeTypePackage {
		nodeTypeStr = model.ServiceTreeTypePackage
	} else if nodeType == model.ServiceTreeTypeFunction {
		nodeTypeStr = model.ServiceTreeTypeFunction
	} else if nodeType == model.ServiceTreeTypeDocs {
		nodeTypeStr = model.ServiceTreeTypeDocs
	} else if nodeType == model.ServiceTreeTypeBoard {
		nodeTypeStr = model.ServiceTreeTypeBoard
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

func applyPermissionInheritance(
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
			if actionType == permission.ActionAdmin {
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

		if parentResourceType == permission.ResourceTypeApp && actionType == permission.ActionAdmin {
			for actionCode := range nodePerms {
				nodePerms[actionCode] = true
			}
			return
		}
	}
}
