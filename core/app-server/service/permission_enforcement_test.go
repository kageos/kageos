package service

import (
	"testing"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
)

func TestWorkspacePermissionEnforcementIsGloballyDisabled(t *testing.T) {
	if DefaultWorkspacePermissionEnforced() {
		t.Fatal("default workspace permission enforcement should be disabled")
	}

	app := &model.App{PermissionEnforced: true}
	if IsWorkspacePermissionEnforced(app) {
		t.Fatal("workspace permission enforcement should stay disabled even when app flag is true")
	}
}
