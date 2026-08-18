package service

import (
	"context"
	"strings"
	"testing"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newPersonalWorkspaceTestService(t *testing.T) (*AppService, *repository.AppRepository, *fakeAppRuntimeClient) {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Host{}, &model.App{}, &model.ServiceTree{}, &model.PersonalWorkspaceBootstrap{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Host{Status: "enabled", NatsID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	repo := repository.NewAppRepository(db)
	client := &fakeAppRuntimeClient{}
	return NewAppService(AppServiceDependencies{
		AppRuntimeClient:      client,
		AppRepository:         repo,
		ServiceTreeRepository: repository.NewServiceTreeRepository(db),
	}), repo, client
}

func TestBootstrapPersonalWorkspaceCreatesPrivateHomeIdempotently(t *testing.T) {
	service, repo, client := newPersonalWorkspaceTestService(t)
	first, err := service.BootstrapPersonalWorkspace(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	if !first.Created || first.App.Code != PersonalWorkspaceCode || first.App.IsPublic || !first.App.IsPersonalWorkspace {
		t.Fatalf("unexpected first bootstrap: %#v", first)
	}
	if first.App.Name != "alice 的默认空间" {
		t.Fatalf("default workspace name = %q", first.App.Name)
	}
	second, err := service.BootstrapPersonalWorkspace(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	if second.Created || second.App.ID != first.App.ID || client.createCalls != 1 {
		t.Fatalf("bootstrap not idempotent: first=%#v second=%#v calls=%d", first, second, client.createCalls)
	}
	var stored model.App
	if err := repo.GetDB().Where("id = ?", first.App.ID).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.IsPublic || !stored.IsPersonalWorkspace {
		t.Fatalf("home persistence is incorrect: %#v", stored)
	}
}

func TestBootstrapPersonalWorkspaceMigratesLegacyGeneratedName(t *testing.T) {
	service, repo, _ := newPersonalWorkspaceTestService(t)
	app := &model.App{User: "alice", Code: PersonalWorkspaceCode, Name: legacyPersonalWorkspaceName, Version: "v1", IsPersonalWorkspace: true}
	if err := repo.CreateApp(app); err != nil {
		t.Fatal(err)
	}
	root := &model.ServiceTree{AppID: app.ID, RefID: app.ID, Name: legacyPersonalWorkspaceName, Code: PersonalWorkspaceCode, Type: model.ServiceTreeTypePackage, FullCodePath: "/alice/home"}
	if err := repo.GetDB().Create(root).Error; err != nil {
		t.Fatal(err)
	}

	resp, err := service.BootstrapPersonalWorkspace(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	if resp.App.Name != "alice 的默认空间" {
		t.Fatalf("migrated app name = %q", resp.App.Name)
	}
	var storedRoot model.ServiceTree
	if err := repo.GetDB().Where("id = ?", root.ID).First(&storedRoot).Error; err != nil {
		t.Fatal(err)
	}
	if storedRoot.Name != "alice 的默认空间" {
		t.Fatalf("migrated root name = %q", storedRoot.Name)
	}
}

func TestBootstrapPersonalWorkspaceKeepsLegacyWorkspace(t *testing.T) {
	service, repo, client := newPersonalWorkspaceTestService(t)
	legacy := &model.App{User: "alice", Code: "sales", Name: "销售", Version: "v1", IsPublic: true}
	if err := repo.CreateApp(legacy); err != nil {
		t.Fatal(err)
	}
	resp, err := service.BootstrapPersonalWorkspace(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Created || resp.App.ID != legacy.ID || client.createCalls != 0 {
		t.Fatalf("legacy workspace should be reused: %#v", resp)
	}
}
