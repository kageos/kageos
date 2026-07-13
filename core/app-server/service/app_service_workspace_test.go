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

func newAppServiceWorkspaceTestDeps(t *testing.T) (*AppService, *repository.AppRepository, *repository.ServiceTreeRepository) {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.App{}, &model.ServiceTree{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	appRepo := repository.NewAppRepository(db)
	treeRepo := repository.NewServiceTreeRepository(db)
	return NewAppService(AppServiceDependencies{RuntimeClient: &fakeAppRuntimeClient{}, AppRepository: appRepo, ServiceTreeRepo: treeRepo}), appRepo, treeRepo
}

func TestAppServiceUpdateWorkspaceRenamesRootAndAppTogether(t *testing.T) {
	service, appRepo, treeRepo := newAppServiceWorkspaceTestDeps(t)
	app := &model.App{User: "alice", Code: "home", Name: "我的空间", Version: "v1"}
	if err := appRepo.CreateApp(context.Background(), app); err != nil {
		t.Fatalf("create app: %v", err)
	}
	if err := treeRepo.Create(context.Background(), &model.ServiceTree{
		AppID:        app.ID,
		RefID:        app.ID,
		Name:         app.Name,
		Code:         app.Code,
		Type:         model.ServiceTreeTypePackage,
		FullCodePath: "/alice/home",
	}); err != nil {
		t.Fatalf("create root node: %v", err)
	}

	name := "客户交付"
	resp, err := service.UpdateWorkspace(context.Background(), &dto.UpdateWorkspaceReq{
		ResourcePath: "/alice/home",
		Name:         &name,
	})
	if err != nil {
		t.Fatalf("UpdateWorkspace error: %v", err)
	}
	if resp.Name != name {
		t.Fatalf("response name = %q, want %q", resp.Name, name)
	}

	updatedApp, err := appRepo.GetAppByUserName(context.Background(), "alice", "home")
	if err != nil {
		t.Fatalf("reload app: %v", err)
	}
	if updatedApp.Name != name || updatedApp.Code != "home" {
		t.Fatalf("unexpected app after rename: %#v", updatedApp)
	}
	root, err := treeRepo.GetRootNodeByAppID(context.Background(), app.ID)
	if err != nil {
		t.Fatalf("reload root: %v", err)
	}
	if root.Name != name || root.Code != "home" || root.FullCodePath != "/alice/home" {
		t.Fatalf("unexpected root after rename: %#v", root)
	}
}

func TestAppServiceUpdateWorkspaceRejectsBlankName(t *testing.T) {
	service, appRepo, treeRepo := newAppServiceWorkspaceTestDeps(t)
	app := &model.App{User: "alice", Code: "home", Name: "我的空间", Version: "v1"}
	if err := appRepo.CreateApp(context.Background(), app); err != nil {
		t.Fatalf("create app: %v", err)
	}
	if err := treeRepo.Create(context.Background(), &model.ServiceTree{
		AppID:        app.ID,
		RefID:        app.ID,
		Name:         app.Name,
		Code:         app.Code,
		Type:         model.ServiceTreeTypePackage,
		FullCodePath: "/alice/home",
	}); err != nil {
		t.Fatalf("create root node: %v", err)
	}

	blank := "  "
	if _, err := service.UpdateWorkspace(context.Background(), &dto.UpdateWorkspaceReq{
		ResourcePath: "/alice/home",
		Name:         &blank,
	}); err == nil || !strings.Contains(err.Error(), "名称不能为空") {
		t.Fatalf("expected blank name error, got %v", err)
	}
}

func TestAppServiceUpdateWorkspaceRejectsDuplicateName(t *testing.T) {
	service, appRepo, treeRepo := newAppServiceWorkspaceTestDeps(t)
	app := &model.App{User: "alice", Code: "home", Name: "我的空间", Version: "v1"}
	if err := appRepo.CreateApp(context.Background(), app); err != nil {
		t.Fatalf("create app: %v", err)
	}
	if err := treeRepo.Create(context.Background(), &model.ServiceTree{
		AppID:        app.ID,
		RefID:        app.ID,
		Name:         app.Name,
		Code:         app.Code,
		Type:         model.ServiceTreeTypePackage,
		FullCodePath: "/alice/home",
	}); err != nil {
		t.Fatalf("create root node: %v", err)
	}
	if err := appRepo.CreateApp(context.Background(), &model.App{User: "alice", Code: "sales", Name: "销售空间", Version: "v1"}); err != nil {
		t.Fatalf("create existing app: %v", err)
	}

	name := "销售空间"
	if _, err := service.UpdateWorkspace(context.Background(), &dto.UpdateWorkspaceReq{
		ResourcePath: "/alice/home",
		Name:         &name,
	}); err == nil || !strings.Contains(err.Error(), "名称已存在") {
		t.Fatalf("expected duplicate name error, got %v", err)
	}
}

func TestAppServiceUpdateWorkspaceAccessMode(t *testing.T) {
	service, appRepo, treeRepo := newAppServiceWorkspaceTestDeps(t)
	app := &model.App{User: "alice", Code: "community", Name: "社区", Version: "v1"}
	if err := appRepo.CreateApp(context.Background(), app); err != nil {
		t.Fatalf("create app: %v", err)
	}
	if err := treeRepo.Create(context.Background(), &model.ServiceTree{
		AppID:        app.ID,
		RefID:        app.ID,
		Name:         app.Name,
		Code:         app.Code,
		Type:         model.ServiceTreeTypePackage,
		FullCodePath: "/alice/community",
	}); err != nil {
		t.Fatalf("create root node: %v", err)
	}

	mode := string(model.AppAccessModeOpenCollaboration)
	resp, err := service.UpdateWorkspace(context.Background(), &dto.UpdateWorkspaceReq{
		ResourcePath: "/alice/community",
		AccessMode:   &mode,
	})
	if err != nil {
		t.Fatalf("UpdateWorkspace error: %v", err)
	}
	if resp.AccessMode != mode {
		t.Fatalf("response access_mode = %q, want %q", resp.AccessMode, mode)
	}

	updatedApp, err := appRepo.GetAppByUserName(context.Background(), "alice", "community")
	if err != nil {
		t.Fatal(err)
	}
	if !updatedApp.IsOpenCollaboration() || !updatedApp.IsPublic {
		t.Fatalf("unexpected app after access mode update: %#v", updatedApp)
	}
}

func TestAppServiceUpdateWorkspaceRejectsInvalidAccessMode(t *testing.T) {
	service, appRepo, _ := newAppServiceWorkspaceTestDeps(t)
	if err := appRepo.CreateApp(context.Background(), &model.App{User: "alice", Code: "community", Name: "社区", Version: "v1"}); err != nil {
		t.Fatalf("create app: %v", err)
	}

	mode := "everyone_is_admin"
	if _, err := service.UpdateWorkspace(context.Background(), &dto.UpdateWorkspaceReq{
		ResourcePath: "/alice/community",
		AccessMode:   &mode,
	}); err == nil || !strings.Contains(err.Error(), "访问模式无效") {
		t.Fatalf("expected invalid access mode error, got %v", err)
	}
}
