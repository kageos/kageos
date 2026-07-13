package repository

import (
	"context"
	"testing"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/pkg/access"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetAppsWithPagePaginatesOwnedGrantedAndOpenTogether(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:app-pagination?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.App{}, &model.WorkspaceRoleAssignment{}); err != nil {
		t.Fatal(err)
	}
	apps := []*model.App{
		{User: "alice", Code: "owned", Name: "Owned"},
		{User: "bob", Code: "granted", Name: "Granted"},
		{User: "carol", Code: "open", Name: "Open", AccessMode: model.AppAccessModeOpenCollaboration},
		{User: "dave", Code: "private", Name: "Private"},
	}
	for _, app := range apps {
		if err := db.Create(app).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&model.WorkspaceRoleAssignment{
		TenantUser: "bob", App: "granted", Username: "alice", ResourcePath: "/bob/granted", RoleCode: string(access.RoleViewer),
	}).Error; err != nil {
		t.Fatal(err)
	}

	repo := NewAppRepository(db)
	page1, total, err := repo.GetAppsWithPage(context.Background(), "alice", 1, 2, "", false, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(page1) != 2 {
		t.Fatalf("page1 len=%d total=%d, want len=2 total=3", len(page1), total)
	}
	page2, total, err := repo.GetAppsWithPage(context.Background(), "alice", 2, 2, "", false, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(page2) != 1 {
		t.Fatalf("page2 len=%d total=%d, want len=1 total=3", len(page2), total)
	}

	expired := time.Now().Add(-time.Minute)
	if err := db.Model(&model.WorkspaceRoleAssignment{}).Where("username = ?", "alice").Update("expires_at", expired).Error; err != nil {
		t.Fatal(err)
	}
	_, total, err = repo.GetAppsWithPage(context.Background(), "alice", 1, 10, "", false, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Fatalf("total after expiration = %d, want 2", total)
	}
}
