package service

import (
	"context"
	"strings"
	"testing"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newWorkspaceContextTestService(t *testing.T) (*serviceTreeWorkspaceService, *gorm.DB) {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ServiceTree{}, &model.Function{}); err != nil {
		t.Fatalf("migrate workspace context models: %v", err)
	}
	treeRepo := repository.NewServiceTreeRepository(db)
	queryView := &serviceTreeQueryView{serviceTreeRepo: treeRepo}
	return newServiceTreeWorkspaceService(treeRepo, nil, nil, queryView), db
}

func TestGetWorkspaceContextResolvesSuffixlessFunctionToParentDirectory(t *testing.T) {
	service, db := newWorkspaceContextTestService(t)
	parent := &model.ServiceTree{
		AppID:        0,
		Type:         model.ServiceTreeTypePackage,
		Code:         "info",
		Name:         "Info",
		FullCodePath: "/system/info",
	}
	if err := db.Create(parent).Error; err != nil {
		t.Fatalf("create parent: %v", err)
	}
	function := &model.ServiceTree{
		AppID:        0,
		Type:         model.ServiceTreeTypeFunction,
		Code:         "site_monitor",
		Name:         "Site Monitor",
		FullCodePath: "/system/info/site_monitor",
	}
	if err := db.Create(function).Error; err != nil {
		t.Fatalf("create function: %v", err)
	}

	resp, err := service.GetWorkspaceContext(context.Background(), &dto.GetWorkspaceContextReq{
		FullCodePath: function.FullCodePath,
	})
	if err != nil {
		t.Fatalf("GetWorkspaceContext: %v", err)
	}
	if resp.Directory.ID != parent.ID || resp.Directory.FullCodePath != parent.FullCodePath || resp.Directory.Type != model.ServiceTreeTypePackage {
		t.Fatalf("directory = %#v, want parent %#v", resp.Directory, parent)
	}
	if len(resp.Children) != 1 || resp.Children[0].ID != function.ID || resp.Children[0].FullCodePath != function.FullCodePath {
		t.Fatalf("children = %#v, want suffix-less function", resp.Children)
	}
}

func TestGetWorkspaceContextKeepsPackageDirectory(t *testing.T) {
	service, db := newWorkspaceContextTestService(t)
	parent := &model.ServiceTree{
		AppID:        0,
		Type:         model.ServiceTreeTypePackage,
		Code:         "site_monitor",
		Name:         "Site Monitor",
		FullCodePath: "/system/info/site_monitor",
		Admins:       "bob,carol",
	}
	parent.CreatedBy = "alice"
	if err := db.Create(parent).Error; err != nil {
		t.Fatalf("create package: %v", err)
	}

	resp, err := service.GetWorkspaceContext(context.Background(), &dto.GetWorkspaceContextReq{
		FullCodePath: parent.FullCodePath,
	})
	if err != nil {
		t.Fatalf("GetWorkspaceContext: %v", err)
	}
	if resp.Directory.ID != parent.ID || resp.Directory.FullCodePath != parent.FullCodePath {
		t.Fatalf("directory = %#v, want requested package %#v", resp.Directory, parent)
	}
	if resp.Directory.Owner != "alice" || resp.Directory.Admins != "bob,carol" {
		t.Fatalf("directory contacts = owner:%q admins:%q, want alice and bob,carol", resp.Directory.Owner, resp.Directory.Admins)
	}
}
