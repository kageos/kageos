package model

import (
	"testing"

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
