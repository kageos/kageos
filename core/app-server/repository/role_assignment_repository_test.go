package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/pkg/access"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRoleAssignmentUpsertIsIdempotentAndRevivesRevokedGrant(t *testing.T) {
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.WorkspaceRoleAssignment{}); err != nil {
		t.Fatal(err)
	}
	repo := NewRoleAssignmentRepository(db)

	first := &model.WorkspaceRoleAssignment{
		TenantUser:    "alice",
		App:           "ops",
		PrincipalType: string(access.PrincipalUser),
		PrincipalKey:  "bob",
		ResourcePath:  "/alice/ops/ticket",
		RoleCode:      string(access.RoleMember),
	}
	first.CreatedBy = "alice"
	first.UpdatedBy = "alice"
	if err := repo.UpsertAssignment(context.Background(), first); err != nil {
		t.Fatal(err)
	}

	laterExpiry := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	second := &model.WorkspaceRoleAssignment{
		TenantUser:    "alice",
		App:           "ops",
		PrincipalType: string(access.PrincipalUser),
		PrincipalKey:  "bob",
		ResourcePath:  "/alice/ops/ticket",
		RoleCode:      string(access.RoleMember),
		ExpiresAt:     &laterExpiry,
	}
	second.CreatedBy = "carol"
	second.UpdatedBy = "carol"
	if err := repo.UpsertAssignment(context.Background(), second); err != nil {
		t.Fatal(err)
	}

	var assignment model.WorkspaceRoleAssignment
	if err := db.First(&assignment).Error; err != nil {
		t.Fatal(err)
	}
	if assignment.CreatedBy != "alice" || assignment.UpdatedBy != "carol" {
		t.Fatalf("grant audit fields = created:%q updated:%q", assignment.CreatedBy, assignment.UpdatedBy)
	}
	if assignment.ExpiresAt == nil || !assignment.ExpiresAt.Equal(laterExpiry) {
		t.Fatalf("expires_at = %v, want %v", assignment.ExpiresAt, laterExpiry)
	}

	if _, err := repo.RemoveAssignment(
		context.Background(),
		"alice",
		"ops",
		access.Principal{Type: access.PrincipalUser, Key: "bob"},
		"/alice/ops/ticket",
		access.RoleMember,
	); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertAssignment(context.Background(), second); err != nil {
		t.Fatal(err)
	}

	var activeCount, totalCount int64
	if err := db.Model(&model.WorkspaceRoleAssignment{}).Count(&activeCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Unscoped().Model(&model.WorkspaceRoleAssignment{}).Count(&totalCount).Error; err != nil {
		t.Fatal(err)
	}
	if activeCount != 1 || totalCount != 1 {
		t.Fatalf("assignment counts = active:%d total:%d, want 1/1", activeCount, totalCount)
	}
}
