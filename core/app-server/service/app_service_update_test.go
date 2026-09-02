package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeAppRuntimeClient struct {
	createCalls  int
	createErr    error
	updateHostID int64
	updateReq    *dto.UpdateAppRuntimeReq
	updateResp   *dto.UpdateAppResp
	updateErr    error

	requestHostID int64
	requestReq    *dto.RequestAppReq
	requestResp   *dto.RequestAppResp
	requestErr    error

	deleteHostID int64
	deleteReq    *dto.DeleteAppRuntimeReq
	deleteResp   *dto.DeleteAppResp
	deleteErr    error
}

func (c *fakeAppRuntimeClient) CreateApp(context.Context, int64, *dto.CreateAppReq) (*dto.CreateAppResp, error) {
	c.createCalls++
	if c.createErr != nil {
		return nil, c.createErr
	}
	return &dto.CreateAppResp{}, nil
}

func (c *fakeAppRuntimeClient) UpdateApp(_ context.Context, hostID int64, req *dto.UpdateAppRuntimeReq) (*dto.UpdateAppResp, error) {
	c.updateHostID = hostID
	c.updateReq = req
	if c.updateErr != nil {
		return nil, c.updateErr
	}
	if c.updateResp != nil {
		return c.updateResp, nil
	}
	return &dto.UpdateAppResp{}, nil
}

func (c *fakeAppRuntimeClient) RequestApp(_ context.Context, hostID int64, req *dto.RequestAppReq) (*dto.RequestAppResp, error) {
	c.requestHostID = hostID
	c.requestReq = req
	if c.requestErr != nil {
		return nil, c.requestErr
	}
	if c.requestResp != nil {
		return c.requestResp, nil
	}
	return &dto.RequestAppResp{}, nil
}

func (c *fakeAppRuntimeClient) DeleteApp(_ context.Context, hostID int64, req *dto.DeleteAppRuntimeReq) (*dto.DeleteAppResp, error) {
	c.deleteHostID = hostID
	c.deleteReq = req
	if c.deleteErr != nil {
		return nil, c.deleteErr
	}
	if c.deleteResp != nil {
		return c.deleteResp, nil
	}
	return &dto.DeleteAppResp{}, nil
}

func newAppServiceUpdateTestRepo(t *testing.T) (*repository.AppRepository, *gorm.DB) {
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
	if err := db.AutoMigrate(&model.App{}, &model.OperateLog{}); err != nil {
		t.Fatalf("migrate app: %v", err)
	}
	return repository.NewAppRepository(db), db
}

func TestAppServiceUpdateAppUsesRuntimeBoundHostAndPersistsVersion(t *testing.T) {
	appRepo, db := newAppServiceUpdateTestRepo(t)
	if err := appRepo.CreateApp(&model.App{
		User:    "alice",
		Code:    "demo",
		Name:    "Demo",
		Version: "v1",
		HostID:  77,
	}); err != nil {
		t.Fatalf("create app: %v", err)
	}
	sourceFiles := []*dto.SourceFileWrite{
		{DirectoryPath: "ticket", FileName: "list", SourceCode: "package ticket"},
	}
	client := &fakeAppRuntimeClient{
		updateResp: &dto.UpdateAppResp{
			User:       "alice",
			App:        "demo",
			OldVersion: "v1",
			NewVersion: "v2",
			Warnings:   []string{"runtime warning"},
		},
	}
	service := NewAppService(AppServiceDependencies{
		AppRuntimeClient:     client,
		AppRepository:        appRepo,
		OperateLogRepository: repository.NewOperateLogRepository(db),
	})

	ctx := contextx.WithRequestUser(context.Background(), "bob")
	resp, err := service.UpdateApp(ctx, &dto.UpdateAppReq{
		ResourcePath:      "/alice/demo/workspace",
		SourceFiles:       sourceFiles,
		Requirement:       "add ticket list",
		ChangeDescription: "create list handler",
		WriteOnly:         true,
		ForceDiff:         true,
	})
	if err != nil {
		t.Fatalf("UpdateApp error = %v", err)
	}

	if client.updateHostID != 77 {
		t.Fatalf("runtime hostID = %d, want 77", client.updateHostID)
	}
	if client.updateReq == nil {
		t.Fatal("runtime update request was not captured")
	}
	if client.updateReq.User != "alice" || client.updateReq.App != "demo" {
		t.Fatalf("runtime user/app = %s/%s, want alice/demo", client.updateReq.User, client.updateReq.App)
	}
	if client.updateReq.SourceFiles[0] != sourceFiles[0] {
		t.Fatalf("source files were not forwarded: %#v", client.updateReq.SourceFiles)
	}
	if client.updateReq.Requirement != "add ticket list" || client.updateReq.ChangeDescription != "create list handler" {
		t.Fatalf("change metadata not forwarded: %#v", client.updateReq)
	}
	if !client.updateReq.WriteOnly || !client.updateReq.ForceDiff {
		t.Fatalf("write flags not forwarded: %#v", client.updateReq)
	}
	if resp.NewVersion != "v2" || len(resp.Warnings) != 1 || resp.Warnings[0] != "runtime warning" {
		t.Fatalf("unexpected response: %#v", resp)
	}

	updated, err := appRepo.GetAppByUserName("alice", "demo")
	if err != nil {
		t.Fatalf("reload app: %v", err)
	}
	if updated.Version != "v2" {
		t.Fatalf("app version = %q, want v2", updated.Version)
	}

	log := waitOperateLog(t, db, workspaceUpdatedAction)
	if log.ActorUser != "bob" || log.Status != "success" || log.ResourceType != "workspace" || log.ResourcePath != "/alice/demo" {
		t.Fatalf("unexpected workspace update log: %#v", log)
	}
	var oldValues, newValues workspaceOperateLogValues
	if err := json.Unmarshal(log.OldValuesJSON, &oldValues); err != nil {
		t.Fatalf("unmarshal old values: %v", err)
	}
	if err := json.Unmarshal(log.NewValuesJSON, &newValues); err != nil {
		t.Fatalf("unmarshal new values: %v", err)
	}
	if oldValues.Version != "v1" || newValues.Version != "v2" {
		t.Fatalf("unexpected workspace version change: old=%#v new=%#v", oldValues, newValues)
	}
}

func TestAppServiceUpdateAppDoesNotPersistVersionWhenRuntimeFails(t *testing.T) {
	appRepo, db := newAppServiceUpdateTestRepo(t)
	if err := appRepo.CreateApp(&model.App{
		User:    "alice",
		Code:    "demo",
		Name:    "Demo",
		Version: "v1",
		HostID:  77,
	}); err != nil {
		t.Fatalf("create app: %v", err)
	}
	service := NewAppService(AppServiceDependencies{
		AppRuntimeClient:     &fakeAppRuntimeClient{updateErr: errors.New("runtime down")},
		AppRepository:        appRepo,
		OperateLogRepository: repository.NewOperateLogRepository(db),
	})

	ctx := contextx.WithRequestUser(context.Background(), "bob")
	_, err := service.UpdateApp(ctx, &dto.UpdateAppReq{ResourcePath: "/alice/demo"})
	if err == nil {
		t.Fatal("expected runtime error")
	}

	updated, err := appRepo.GetAppByUserName("alice", "demo")
	if err != nil {
		t.Fatalf("reload app: %v", err)
	}
	if updated.Version != "v1" {
		t.Fatalf("app version = %q, want v1", updated.Version)
	}
	log := waitOperateLog(t, db, workspaceUpdatedAction)
	if log.ActorUser != "bob" || log.Status != "failed" || !strings.Contains(log.Summary, "failed to update workspace") {
		t.Fatalf("unexpected failed workspace update log: %#v", log)
	}
}
