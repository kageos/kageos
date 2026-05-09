package service

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
)

// DefaultWorkspacePermissionEnforced returns the creation default for RBAC enforcement.
// Enterprise permission licenses keep the historical enterprise behavior: RBAC is on.
func DefaultWorkspacePermissionEnforced() bool {
	return license.GetManager().HasFeature(enterprise.FeaturePermission)
}

// IsWorkspacePermissionEnforced returns the effective RBAC state.
// Community can opt in per workspace; enterprise permission licenses enforce RBAC by default.
func IsWorkspacePermissionEnforced(app *model.App) bool {
	if app == nil {
		return false
	}
	return app.PermissionEnforced || DefaultWorkspacePermissionEnforced()
}
