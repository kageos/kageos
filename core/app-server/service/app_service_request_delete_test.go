package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAppServiceRequestDeleteTestDeps(t *testing.T) (*repository.AppRepository, *repository.ServiceTreeRepository, *gorm.DB) {
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
	return repository.NewAppRepository(db), repository.NewServiceTreeRepository(db), db
}

func createRequestDeleteTestApp(t *testing.T, appRepo *repository.AppRepository) *model.App {
	t.Helper()
	app := &model.App{
		User:    "alice",
		Code:    "demo",
		Name:    "Demo",
		Version: "v7",
		NatsID:  88,
		HostID:  99,
	}
	if err := appRepo.CreateApp(context.Background(), app); err != nil {
		t.Fatalf("create app: %v", err)
	}
	return app
}

func TestAppServiceRequestAppInjectsVersionSourceContextAndUsesNatsID(t *testing.T) {
	appRepo, serviceTreeRepo, _ := newAppServiceRequestDeleteTestDeps(t)
	app := createRequestDeleteTestApp(t, appRepo)
	if err := serviceTreeRepo.Create(context.Background(), &model.ServiceTree{
		AppID:        app.ID,
		Type:         model.ServiceTreeTypePackage,
		Name:         "Tickets",
		Code:         "ticket",
		FullCodePath: "/alice/demo/ticket",
	}); err != nil {
		t.Fatalf("create parent tree: %v", err)
	}
	if err := serviceTreeRepo.Create(context.Background(), &model.ServiceTree{
		AppID:        app.ID,
		Type:         model.ServiceTreeTypeFunction,
		Name:         "Ticket List",
		Code:         "list.table",
		TemplateType: "table",
		FullCodePath: "/alice/demo/ticket/list.table",
	}); err != nil {
		t.Fatalf("create source tree: %v", err)
	}
	client := &fakeAppRuntimeClient{
		requestResp: &dto.RequestAppResp{
			Version: "runtime-version",
			Result:  "ok",
		},
	}
	service := NewAppService(AppServiceDependencies{RuntimeClient: client, AppRepository: appRepo, ServiceTreeRepo: serviceTreeRepo})

	resp, err := service.RequestApp(context.Background(), &dto.RequestAppReq{
		TraceId: "trace-1",
		User:    "alice",
		App:     "demo",
		Version: "stale",
		Router:  "ticket/list.table",
		Method:  "GET",
	})
	if err != nil {
		t.Fatalf("RequestApp error = %v", err)
	}

	if client.requestHostID != 88 {
		t.Fatalf("runtime natsID = %d, want 88", client.requestHostID)
	}
	if client.requestReq == nil {
		t.Fatal("runtime request was not captured")
	}
	if client.requestReq.Version != "v7" || resp.Version != "v7" {
		t.Fatalf("version not injected from DB: req=%q resp=%q", client.requestReq.Version, resp.Version)
	}
	if client.requestReq.SourcePath != "/alice/demo/ticket/list.table" {
		t.Fatalf("source_path = %q", client.requestReq.SourcePath)
	}
	if client.requestReq.SourceTitle != "Ticket List" || client.requestReq.SourceTemplateType != "table" {
		t.Fatalf("source display not hydrated: %#v", client.requestReq)
	}
	if client.requestReq.SourceParentPath != "/alice/demo/ticket" || client.requestReq.SourceParentTitle != "Tickets" {
		t.Fatalf("parent source display not hydrated: %#v", client.requestReq)
	}
	if resp.Result != "ok" {
		t.Fatalf("runtime result changed: %#v", resp)
	}
}

func TestAppServiceRequestAppReturnsRuntimeError(t *testing.T) {
	appRepo, _, _ := newAppServiceRequestDeleteTestDeps(t)
	createRequestDeleteTestApp(t, appRepo)
	client := &fakeAppRuntimeClient{requestErr: errors.New("runtime down")}
	service := NewAppService(AppServiceDependencies{RuntimeClient: client, AppRepository: appRepo})

	_, err := service.RequestApp(context.Background(), &dto.RequestAppReq{
		User:   "alice",
		App:    "demo",
		Router: "ticket/list.table",
		Method: "GET",
	})
	if err == nil || !strings.Contains(err.Error(), "runtime down") {
		t.Fatalf("expected runtime error, got %v", err)
	}
	if client.requestHostID != 88 {
		t.Fatalf("runtime should be called with natsID 88, got %d", client.requestHostID)
	}
}

func TestAppServiceDeleteAppDeletesDatabaseOnlyAfterRuntimeSuccess(t *testing.T) {
	appRepo, _, _ := newAppServiceRequestDeleteTestDeps(t)
	createRequestDeleteTestApp(t, appRepo)
	client := &fakeAppRuntimeClient{
		deleteResp: &dto.DeleteAppResp{User: "alice", App: "demo"},
	}
	service := NewAppService(AppServiceDependencies{RuntimeClient: client, AppRepository: appRepo})

	resp, err := service.DeleteApp(context.Background(), &dto.DeleteAppReq{ResourcePath: "/alice/demo/ticket"})
	if err != nil {
		t.Fatalf("DeleteApp error = %v", err)
	}
	if resp.User != "alice" || resp.App != "demo" {
		t.Fatalf("unexpected delete response: %#v", resp)
	}
	if client.deleteHostID != 99 {
		t.Fatalf("runtime hostID = %d, want 99", client.deleteHostID)
	}
	if client.deleteReq == nil || client.deleteReq.User != "alice" || client.deleteReq.App != "demo" {
		t.Fatalf("runtime delete request not forwarded: %#v", client.deleteReq)
	}
	if _, err := appRepo.GetAppByUserName(context.Background(), "alice", "demo"); err == nil {
		t.Fatal("expected app to be deleted after runtime success")
	}
}

func TestAppServiceDeleteAppKeepsDatabaseWhenRuntimeFails(t *testing.T) {
	appRepo, _, _ := newAppServiceRequestDeleteTestDeps(t)
	createRequestDeleteTestApp(t, appRepo)
	client := &fakeAppRuntimeClient{deleteErr: errors.New("runtime delete failed")}
	service := NewAppService(AppServiceDependencies{RuntimeClient: client, AppRepository: appRepo})

	_, err := service.DeleteApp(context.Background(), &dto.DeleteAppReq{ResourcePath: "/alice/demo"})
	if err == nil || !strings.Contains(err.Error(), "runtime delete failed") {
		t.Fatalf("expected runtime delete error, got %v", err)
	}
	app, err := appRepo.GetAppByUserName(context.Background(), "alice", "demo")
	if err != nil {
		t.Fatalf("app should remain when runtime delete fails: %v", err)
	}
	if app.Version != "v7" {
		t.Fatalf("app changed unexpectedly: %#v", app)
	}
}
