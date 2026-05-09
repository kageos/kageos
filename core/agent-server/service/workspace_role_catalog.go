package service

import (
	"strings"

	workspaceroles "github.com/ai-agent-os/ai-agent-os/core/agent-server/workspace/roles"
)

const (
	WorkspaceRoleRouter              = workspaceroles.Router
	WorkspaceRoleProductManager      = workspaceroles.ProductManager
	WorkspaceRoleAppDeveloper        = workspaceroles.AppDeveloper
	WorkspaceRoleBuildEngineer       = workspaceroles.BuildEngineer
	WorkspaceRoleQAEngineer          = workspaceroles.QAEngineer
	WorkspaceRoleAppOperator         = workspaceroles.AppOperator
	WorkspaceRoleMaintenanceEngineer = workspaceroles.MaintenanceEngineer
	WorkspaceRoleSchedulerEngineer   = workspaceroles.SchedulerEngineer
	WorkspaceRolePlatformEngineer    = workspaceroles.PlatformEngineer
	WorkspaceRoleDataOperator        = workspaceroles.DataOperator
	WorkspaceRoleReviewer            = workspaceroles.Reviewer
)

type workspaceRoleSpec = workspaceroles.Spec
type nextWorkspaceRole = workspaceroles.NextRole

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

func workspaceRoleDisplayName(role string) string {
	return workspaceroles.DisplayName(role)
}

func workspaceRoleTransitionWhen(fromRole, toRole string) (string, bool) {
	return workspaceroles.TransitionWhen(fromRole, toRole)
}

func workspaceRoleAllowedTools(role string) []string {
	allowed := append([]string(nil), workspaceRoleBaseReadOnlyTools()...)
	if spec, ok := workspaceRoleSpecFor(role); ok {
		for _, tool := range spec.AllowedTools {
			if !containsWorkspaceRoleString(allowed, tool) {
				allowed = append(allowed, tool)
			}
		}
	}
	return allowed
}

func workspaceRoleForbiddenTools(role string) []string {
	if spec, ok := workspaceRoleSpecFor(role); ok {
		return append([]string(nil), spec.ForbiddenTools...)
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
