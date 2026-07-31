package service

import (
	"context"
	"strings"
	"testing"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUpdateWorkspaceRenamesAppAndRootTogether(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.App{}, &model.ServiceTree{}); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewAppRepository(db)
	app := &model.App{User: "alice", Code: "home", Name: "我的空间", Version: "v1"}
	if err := repo.CreateApp(app); err != nil {
		t.Fatal(err)
	}
	if !app.IsPublic {
		t.Fatal("test setup expected the workspace to start public")
	}
	root := &model.ServiceTree{AppID: app.ID, RefID: app.ID, Name: app.Name, Code: app.Code, Type: model.ServiceTreeTypePackage, FullCodePath: "/alice/home"}
	if err := db.Create(root).Error; err != nil {
		t.Fatal(err)
	}

	name := "客户交付"
	isPublic := false
	hideUnauthorizedNodes := true
	service := NewAppService(AppServiceDependencies{
		AppRuntimeClient:      &fakeAppRuntimeClient{},
		AppRepository:         repo,
		ServiceTreeRepository: repository.NewServiceTreeRepository(db),
	})
	resp, err := service.UpdateWorkspace(context.Background(), &dto.UpdateWorkspaceReq{
		ResourcePath:          "/alice/home",
		Name:                  &name,
		IsPublic:              &isPublic,
		HideUnauthorizedNodes: &hideUnauthorizedNodes,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Name != name {
		t.Fatalf("response name = %q", resp.Name)
	}
	if resp.IsPublic || !resp.HideUnauthorizedNodes {
		t.Fatalf("response settings not updated: %#v", resp)
	}
	var updatedApp model.App
	var updatedRoot model.ServiceTree
	_ = db.First(&updatedApp, app.ID).Error
	_ = db.First(&updatedRoot, root.ID).Error
	if updatedApp.Name != name || updatedApp.Code != "home" || updatedApp.IsPublic || !updatedApp.HideUnauthorizedNodes || updatedRoot.Name != name || updatedRoot.FullCodePath != "/alice/home" {
		t.Fatalf("rename changed identity or missed root: app=%#v root=%#v", updatedApp, updatedRoot)
	}
}

func TestCanEnterWorkspaceSeparatesVisibilityFromDirectoryPermission(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.App{}, &model.WorkspaceRoleAssignment{}, &model.OperateLog{}); err != nil {
		t.Fatal(err)
	}
	appRepo := repository.NewAppRepository(db)
	apps := []*model.App{
		{User: "alice", Code: "public", Name: "公开空间", Version: "v1", IsPublic: true},
		{User: "alice", Code: "private", Name: "私有空间", Version: "v1"},
	}
	for _, app := range apps {
		if err := appRepo.CreateApp(app); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Model(apps[1]).Update("is_public", false).Error; err != nil {
		t.Fatal(err)
	}
	appRepo.InvalidateAppCacheBoth(apps[1].User, apps[1].Code, apps[1].ID)
	permission := NewPermissionService(
		repository.NewRoleAssignmentRepository(db),
		repository.NewOperateLogRepository(db),
		appRepo,
	)
	permission.userLookup = func(_ context.Context, username string) (*dto.UserInfo, error) {
		return &dto.UserInfo{Username: username}, nil
	}
	service := NewAppService(AppServiceDependencies{AppRepository: appRepo, PermissionService: permission})

	canEnter, err := service.CanEnterWorkspace(context.Background(), "/alice/public", "bob")
	if err != nil || !canEnter {
		t.Fatalf("logged-in user should enter public workspace: canEnter=%t err=%v", canEnter, err)
	}
	canEnter, err = service.CanEnterWorkspace(context.Background(), "/alice/public", "")
	if err != nil || canEnter {
		t.Fatalf("anonymous user should not enter authenticated public workspace: canEnter=%t err=%v", canEnter, err)
	}
	canEnter, err = service.CanEnterWorkspace(context.Background(), "/alice/private", "bob")
	if err != nil || canEnter {
		t.Fatalf("user without assignment should not enter private workspace: canEnter=%t err=%v", canEnter, err)
	}
	canReadDirectory, err := permission.HasPermission(context.Background(), "alice", "public", "bob", "/alice/public/finance", access.ActionRead)
	if err != nil || canReadDirectory {
		t.Fatalf("public visibility must not grant directory read: canRead=%t err=%v", canReadDirectory, err)
	}
	listResp, err := service.GetApps(context.Background(), &dto.GetAppsReq{
		PageInfoReq: dto.PageInfoReq{Page: 1, PageSize: 20},
		User:        "bob",
		IncludeAll:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	items, ok := listResp.Items.([]*dto.AppInfo)
	if !ok || len(items) != 1 || items[0].Code != "public" {
		t.Fatalf("public discovery should return only public workspace, got %#v", listResp.Items)
	}
}
