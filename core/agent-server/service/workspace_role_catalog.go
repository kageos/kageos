package service

import (
	"strings"

	workspaceroles "github.com/kageos/kageos/core/agent-server/workspace/roles"
)

const (
	WorkspaceRoleRouter              = workspaceroles.Router
	WorkspaceRoleProductManager      = workspaceroles.ProductManager
	WorkspaceRoleAppDeveloper        = workspaceroles.AppDeveloper
	WorkspaceRoleBuildEngineer       = workspaceroles.BuildEngineer
	WorkspaceRoleQAEngineer          = workspaceroles.QAEngineer
	WorkspaceRoleAppOperator         = workspaceroles.AppOperator
	WorkspaceRoleAutomationOperator  = workspaceroles.AutomationOperator
	WorkspaceRoleMaintenanceEngineer = workspaceroles.MaintenanceEngineer
	WorkspaceRolePlatformEngineer    = workspaceroles.PlatformEngineer
	WorkspaceRoleDataOperator        = workspaceroles.DataOperator
	WorkspaceRoleReviewer            = workspaceroles.Reviewer
)

type workspaceRoleSpec = workspaceroles.Spec
type nextWorkspaceRole = workspaceroles.NextRole
type roleRuntimeContract = workspaceroles.RuntimeContract

func workspaceRoleSpecs() map[string]workspaceRoleSpec {
	return workspaceroles.Specs()
}

func workspaceRoleAliases() map[string]string {
	return workspaceroles.Aliases()
}

func normalizeWorkspaceRole(role string) string {
	return workspaceroles.Normalize(role)
}

func isKnownWorkspaceRole(role string) bool {
	return workspaceroles.IsKnown(role)
}

func workspaceRoleSpecFor(role string) (workspaceRoleSpec, bool) {
	return workspaceroles.SpecFor(role)
}

func workspaceStandardRoleIDs() []string {
	return append([]string(nil), workspaceroles.RouteOrder()...)
}

func workspaceRoleDisplayName(role string) string {
	return workspaceroles.DisplayName(role)
}

func workspaceRoleTransitionWhen(fromRole, toRole string) (string, bool) {
	return workspaceroles.TransitionWhen(fromRole, toRole)
}

func workspaceRoleAllowedTools(role string) []string {
	if definition, ok := workspaceRoleDefinitionFor(role); ok {
		return append([]string(nil), definition.AllowedTools...)
	}
	return append([]string(nil), workspaceRoleBaseReadOnlyTools()...)
}

func workspaceRoleForbiddenTools(role string) []string {
	if definition, ok := workspaceRoleDefinitionFor(role); ok {
		return append([]string(nil), definition.ForbiddenTools...)
	}
	return nil
}

func containsWorkspaceRoleString(list []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, item := range list {
		if strings.TrimSpace(item) == target {
			return true
		}
	}
	return false
}
