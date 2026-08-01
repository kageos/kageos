package model

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMigrateLegacyPermissionPrincipals(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:legacy-permission-principals?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
		CREATE TABLE workspace_role_assignments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_user TEXT NOT NULL,
			app TEXT NOT NULL,
			username TEXT NOT NULL,
			resource_path TEXT NOT NULL,
			role_code TEXT NOT NULL
		)
	`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
		INSERT INTO workspace_role_assignments (tenant_user, app, username, resource_path, role_code)
		VALUES ('alice', 'ops', 'bob', '/alice/ops/ticket', 'viewer')
	`).Error; err != nil {
		t.Fatal(err)
	}
	if err := migrateLegacyPermissionPrincipals(db); err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasColumn(&legacyPermissionAssignment{}, "Username") {
		t.Fatal("legacy username column should be removed")
	}

	var assignment struct {
		PrincipalType string
		PrincipalKey  string
	}
	if err := db.Table("workspace_role_assignments").
		Select("principal_type, principal_key").
		First(&assignment).Error; err != nil {
		t.Fatal(err)
	}
	if assignment.PrincipalType != "user" || assignment.PrincipalKey != "bob" {
		t.Fatalf("principal = %s:%s, want user:bob", assignment.PrincipalType, assignment.PrincipalKey)
	}
}

func TestBackfillPermissionAssignmentKeysDeduplicatesLegacyGrants(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:permission-assignment-keys?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&WorkspaceRoleAssignment{}); err != nil {
		t.Fatal(err)
	}

	expiresSoon := time.Now().Add(time.Hour)
	assignments := []*WorkspaceRoleAssignment{
		{
			TenantUser:    "alice",
			App:           "ops",
			PrincipalType: "user",
			PrincipalKey:  "bob",
			ResourcePath:  "/alice/ops/ticket",
			RoleCode:      "member",
			ExpiresAt:     &expiresSoon,
		},
		{
			TenantUser:    "alice",
			App:           "ops",
			PrincipalType: "user",
			PrincipalKey:  "bob",
			ResourcePath:  "/alice/ops/ticket",
			RoleCode:      "member",
			ExpiresAt:     nil,
		},
		{
			TenantUser:    "alice",
			App:           "ops",
			PrincipalType: "user",
			PrincipalKey:  "bob",
			ResourcePath:  "/alice/ops/ticket",
			RoleCode:      "viewer",
			ExpiresAt:     nil,
		},
	}
	for _, assignment := range assignments {
		assignment.CreatedBy = "alice"
		if err := db.Create(assignment).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := backfillPermissionAssignmentKeys(db); err != nil {
		t.Fatal(err)
	}

	var migrated []*WorkspaceRoleAssignment
	if err := db.Order("role_code ASC").Find(&migrated).Error; err != nil {
		t.Fatal(err)
	}
	if len(migrated) != 2 {
		t.Fatalf("assignment count = %d, want one member and one viewer", len(migrated))
	}
	for _, assignment := range migrated {
		if assignment.AssignmentKey == nil || *assignment.AssignmentKey == "" {
			t.Fatalf("assignment key was not backfilled: %+v", assignment)
		}
		if assignment.RoleCode == "member" && assignment.ExpiresAt != nil {
			t.Fatalf("deduplication should preserve the permanent member grant: %+v", assignment)
		}
	}

	if err := backfillPermissionAssignmentKeys(db); err != nil {
		t.Fatalf("re-running key migration should be idempotent: %v", err)
	}
}
