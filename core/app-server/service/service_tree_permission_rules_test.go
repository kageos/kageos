package service

import (
	"testing"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/permission"
)

func TestIsWorkspaceAdmin_TrimsAdminList(t *testing.T) {
	if !isWorkspaceAdmin("alice", " bob, alice ,carol ") {
		t.Fatal("expected alice to be recognized as workspace admin")
	}
	if isWorkspaceAdmin("dave", " bob, alice ,carol ") {
		t.Fatal("did not expect dave to be recognized as workspace admin")
	}
}

func TestBuildRawPermissions_GroupsByResourcePath(t *testing.T) {
	raw := buildRawPermissions([]dto.PermissionRecord{
		{Resource: "/u/a", Action: permission.BuildActionCode(permission.ResourceTypeApp, "read")},
		{Resource: "/u/a/pkg", Action: permission.BuildActionCode(permission.ResourceTypeDirectory, "write")},
		{Resource: "/u/a/pkg", Action: permission.BuildActionCode(permission.ResourceTypeDirectory, "admin")},
	})

	if len(raw) != 2 {
		t.Fatalf("expected 2 resource groups, got %d", len(raw))
	}
	if !raw["/u/a/pkg"][permission.BuildActionCode(permission.ResourceTypeDirectory, "write")] {
		t.Fatal("expected directory write permission to exist")
	}
	if !raw["/u/a/pkg"][permission.BuildActionCode(permission.ResourceTypeDirectory, "admin")] {
		t.Fatal("expected directory admin permission to exist")
	}
}

func TestPermissionActionsForNode_MapsFunctionTemplateType(t *testing.T) {
	actions := permissionActionsForNode(model.ServiceTreeTypeFunction, "form")
	if len(actions) == 0 {
		t.Fatal("expected form function to expose permission actions")
	}
	if actions[0] != permission.BuildActionCode(permission.ResourceTypeForm, "read") {
		t.Fatalf("unexpected first action: %s", actions[0])
	}
}

func TestApplyPermissionInheritance_DirectoryWriteMapsToChildResourceType(t *testing.T) {
	nodePerms := initializeNodePermissions(permissionActionsForNode(model.ServiceTreeTypeFunction, "table"), nil)
	parentPerms := map[string]bool{
		permission.BuildActionCode(permission.ResourceTypeDirectory, "write"): true,
	}

	applyPermissionInheritance(model.ServiceTreeTypeFunction, "table", parentPerms, nodePerms)

	if !nodePerms[permission.BuildActionCode(permission.ResourceTypeTable, "write")] {
		t.Fatal("expected directory write to inherit table write")
	}
	if nodePerms[permission.BuildActionCode(permission.ResourceTypeTable, "admin")] {
		t.Fatal("did not expect directory write to inherit table admin")
	}
}

func TestApplyPermissionInheritance_DirectoryAdminGrantsAllNodePermissions(t *testing.T) {
	nodePerms := initializeNodePermissions(permissionActionsForNode(model.ServiceTreeTypePackage, ""), nil)
	parentPerms := map[string]bool{
		permission.BuildActionCode(permission.ResourceTypeDirectory, "admin"): true,
	}

	applyPermissionInheritance(model.ServiceTreeTypePackage, "", parentPerms, nodePerms)

	for actionCode, granted := range nodePerms {
		if !granted {
			t.Fatalf("expected %s to be granted by directory admin", actionCode)
		}
	}
}
