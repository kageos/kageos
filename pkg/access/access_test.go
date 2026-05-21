package access

import (
	"testing"
	"time"
)

func TestRolePermissions(t *testing.T) {
	tests := []struct {
		name string
		role RoleCode
		want []Action
		deny []Action
	}{
		{name: "owner", role: RoleOwner, want: []Action{ActionRead, ActionWrite, ActionUpdate, ActionDelete, ActionAdmin, ActionOwner}},
		{name: "admin", role: RoleAdmin, want: []Action{ActionRead, ActionWrite, ActionUpdate, ActionDelete, ActionAdmin}, deny: []Action{ActionOwner}},
		{name: "member", role: RoleMember, want: []Action{ActionRead, ActionWrite, ActionUpdate}, deny: []Action{ActionDelete, ActionAdmin, ActionOwner}},
		{name: "viewer", role: RoleViewer, want: []Action{ActionRead}, deny: []Action{ActionWrite, ActionUpdate, ActionDelete, ActionAdmin, ActionOwner}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perms := RolePermissions(tt.role)
			for _, action := range tt.want {
				if !HasPermission(perms, action) {
					t.Fatalf("expected %s to have %s", tt.role, action)
				}
			}
			for _, action := range tt.deny {
				if HasPermission(perms, action) {
					t.Fatalf("expected %s to deny %s", tt.role, action)
				}
			}
		})
	}
}

func TestIsSystemBuiltinPath(t *testing.T) {
	for _, path := range []string{"/system", "/system/prompt/case_catalog/table/ticket", "system/tools/runtime/python.form"} {
		if !IsSystemBuiltinPath(path) {
			t.Fatalf("%s should be treated as system builtin", path)
		}
	}
	if IsSystemBuiltinPath("/alice/app/system") {
		t.Fatal("user workspace path should not be treated as system builtin")
	}
}

func TestResolveInheritsFromParentPath(t *testing.T) {
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	result := Resolve([]Assignment{
		{
			Username:     "bob",
			ResourcePath: "/alice/ops/ticket",
			RoleCode:     RoleMember,
		},
	}, "/alice/ops/ticket/sub/items.table", now)

	if !HasPermission(result.Permissions, ActionRead) {
		t.Fatal("expected inherited read permission")
	}
	if !HasPermission(result.Permissions, ActionWrite) {
		t.Fatal("expected inherited write permission")
	}
	if HasPermission(result.Permissions, ActionDelete) {
		t.Fatal("member must not inherit delete permission")
	}
	if result.InheritedFrom != "/alice/ops/ticket" {
		t.Fatalf("expected inherited_from parent path, got %q", result.InheritedFrom)
	}
}

func TestResolveSkipsExpiredAssignments(t *testing.T) {
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	expired := now.Add(-time.Minute)
	result := Resolve([]Assignment{
		{
			Username:     "bob",
			ResourcePath: "/alice/ops",
			RoleCode:     RoleAdmin,
			ExpiresAt:    &expired,
		},
	}, "/alice/ops/ticket", now)

	if HasPermission(result.Permissions, ActionAdmin) {
		t.Fatal("expired assignment should not grant admin")
	}
}

func TestResolveMergesMultipleRoles(t *testing.T) {
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	result := Resolve([]Assignment{
		{Username: "bob", ResourcePath: "/alice/ops", RoleCode: RoleViewer},
		{Username: "bob", ResourcePath: "/alice/ops/ticket", RoleCode: RoleAdmin},
	}, "/alice/ops/ticket", now)

	if !HasPermission(result.Permissions, ActionAdmin) {
		t.Fatal("expected admin from child assignment")
	}
	if !HasPermission(result.Permissions, ActionRead) {
		t.Fatal("expected read from merged roles")
	}
}

func TestNormalizeResourcePath(t *testing.T) {
	got := NormalizeResourcePath(" alice/ops/ticket/ ")
	if got != "/alice/ops/ticket" {
		t.Fatalf("unexpected normalized path: %q", got)
	}
}
