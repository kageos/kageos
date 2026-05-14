package service

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	permissionpkg "github.com/ai-agent-os/ai-agent-os/pkg/permission"
)

// DefaultWorkspacePermissionEnforced returns the creation default for RBAC enforcement.
func DefaultWorkspacePermissionEnforced() bool {
	return permissionpkg.EnforcementEnabled()
}

// IsWorkspacePermissionEnforced returns the effective RBAC state.
func IsWorkspacePermissionEnforced(app *model.App) bool {
	if app == nil || !permissionpkg.EnforcementEnabled() {
		return false
	}
	return app.PermissionEnforced
}
