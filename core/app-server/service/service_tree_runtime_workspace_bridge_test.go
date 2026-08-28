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

type fakeRuntimeWorkspaceClient struct {
	batchWriteHostID int64
	batchWriteReq    *dto.BatchWriteFilesRuntimeReq
	batchWriteResp   *dto.BatchWriteFilesRuntimeResp

	replaceHostID int64
	replaceReq    *dto.ReplaceDirectoryTreeRuntimeReq
	replaceResp   *dto.ReplaceDirectoryTreeRuntimeResp

	deleteTreeHostID int64
	deleteTreeReq    *dto.DeleteServiceTreeRuntimeReq
	deleteTreeResp   *dto.DeleteServiceTreeRuntimeResp

	readDirHostID int64
	readDirReq    *dto.ReadDirectoryFilesRuntimeReq
	readDirResp   *dto.ReadDirectoryFilesRuntimeResp

	writeFileHostID int64
	writeFileReq    *dto.WriteFileRuntimeReq
	writeFileResp   *dto.WriteFileRuntimeResp
}

func (c *fakeRuntimeWorkspaceClient) BatchCreateDirectoryTree(context.Context, int64, *dto.BatchCreateDirectoryTreeRuntimeReq) (*dto.BatchCreateDirectoryTreeRuntimeResp, error) {
	return &dto.BatchCreateDirectoryTreeRuntimeResp{}, nil
}

func (c *fakeRuntimeWorkspaceClient) BatchWriteFiles(_ context.Context, hostID int64, req *dto.BatchWriteFilesRuntimeReq) (*dto.BatchWriteFilesRuntimeResp, error) {
	c.batchWriteHostID = hostID
	c.batchWriteReq = req
	if c.batchWriteResp != nil {
		return c.batchWriteResp, nil
	}
	return &dto.BatchWriteFilesRuntimeResp{}, nil
}

func (c *fakeRuntimeWorkspaceClient) ReplaceDirectoryTree(_ context.Context, hostID int64, req *dto.ReplaceDirectoryTreeRuntimeReq) (*dto.ReplaceDirectoryTreeRuntimeResp, error) {
	c.replaceHostID = hostID
	c.replaceReq = req
	if c.replaceResp != nil {
		return c.replaceResp, nil
	}
	return &dto.ReplaceDirectoryTreeRuntimeResp{}, nil
}

func (c *fakeRuntimeWorkspaceClient) DeleteServiceTree(_ context.Context, hostID int64, req *dto.DeleteServiceTreeRuntimeReq) (*dto.DeleteServiceTreeRuntimeResp, error) {
	c.deleteTreeHostID = hostID
	c.deleteTreeReq = req
	if c.deleteTreeResp != nil {
		return c.deleteTreeResp, nil
	}
	return &dto.DeleteServiceTreeRuntimeResp{}, nil
}

func (c *fakeRuntimeWorkspaceClient) ReadDirectoryFiles(_ context.Context, hostID int64, req *dto.ReadDirectoryFilesRuntimeReq) (*dto.ReadDirectoryFilesRuntimeResp, error) {
	c.readDirHostID = hostID
	c.readDirReq = req
	if c.readDirResp != nil {
		return c.readDirResp, nil
	}
	return &dto.ReadDirectoryFilesRuntimeResp{}, nil
}

func (c *fakeRuntimeWorkspaceClient) ReplaceInFileBatch(context.Context, int64, *dto.ReplaceInFileBatchReq) (*dto.ReplaceInFileBatchResp, error) {
	return &dto.ReplaceInFileBatchResp{}, nil
}

func (c *fakeRuntimeWorkspaceClient) WriteFile(_ context.Context, hostID int64, req *dto.WriteFileRuntimeReq) (*dto.WriteFileRuntimeResp, error) {
	c.writeFileHostID = hostID
	c.writeFileReq = req
	if c.writeFileResp != nil {
		return c.writeFileResp, nil
	}
	return &dto.WriteFileRuntimeResp{Success: true, Message: "写入成功"}, nil
}

func (c *fakeRuntimeWorkspaceClient) DeleteFile(context.Context, int64, *dto.DeleteFileRuntimeReq) (*dto.DeleteFileRuntimeResp, error) {
	return &dto.DeleteFileRuntimeResp{}, nil
}

func (c *fakeRuntimeWorkspaceClient) ReadAppLog(context.Context, int64, *dto.ReadAppLogRuntimeReq) (*dto.ReadAppLogRuntimeResp, error) {
	return &dto.ReadAppLogRuntimeResp{}, nil
}

func newRuntimeWorkspaceBridgeTestRepo(t *testing.T) *repository.AppRepository {
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
	if err := db.AutoMigrate(&model.App{}); err != nil {
		t.Fatalf("migrate app: %v", err)
	}
	return repository.NewAppRepository(db)
}

func createRuntimeWorkspaceBridgeTestApp(t *testing.T, appRepo *repository.AppRepository, hostID int64) *model.App {
	t.Helper()
	app := &model.App{
		User:   "alice",
		Code:   "demo",
		Name:   "Demo",
		HostID: hostID,
	}
	if err := appRepo.CreateApp(app); err != nil {
		t.Fatalf("create app: %v", err)
	}
	return app
}

func TestRuntimeWorkspaceBridgeBatchWriteFilesUsesRuntimeBoundHost(t *testing.T) {
	appRepo := newRuntimeWorkspaceBridgeTestRepo(t)
	if err := appRepo.GetDB().Create(&model.App{
		User:   "alice",
		Code:   "demo",
		Name:   "Demo",
		HostID: 42,
	}).Error; err != nil {
		t.Fatalf("create app: %v", err)
	}
	client := &fakeRuntimeWorkspaceClient{
		batchWriteResp: &dto.BatchWriteFilesRuntimeResp{
			FileCount:    1,
			WrittenPaths: []string{"api/ticket/ticket.go"},
			NewVersion:   "v2",
		},
	}
	bridge := newRuntimeWorkspaceBridge(appRepo, client)

	app, resp, err := bridge.batchWriteFiles(context.Background(), &dto.BatchWriteFilesReq{
		User:           "alice",
		App:            "demo",
		OperationName:  "install_capability_bundle",
		OperationLabel: "导入目录",
		ForceDiff:      true,
		Files: []*dto.FileWriteItem{
			{FullCodePath: "/alice/demo/ticket", RelativePath: "ticket.go", Content: "package ticket"},
		},
	})
	if err != nil {
		t.Fatalf("batchWriteFiles error = %v", err)
	}
	if app.HostID != 42 {
		t.Fatalf("app.HostID = %d, want 42", app.HostID)
	}
	if resp.NewVersion != "v2" || resp.FileCount != 1 {
		t.Fatalf("unexpected runtime response: %#v", resp)
	}
	if client.batchWriteHostID != 42 {
		t.Fatalf("runtime hostID = %d, want 42", client.batchWriteHostID)
	}
	if client.batchWriteReq == nil {
		t.Fatal("runtime request was not captured")
	}
	if client.batchWriteReq.User != "alice" || client.batchWriteReq.App != "demo" || !client.batchWriteReq.ForceDiff {
		t.Fatalf("unexpected runtime request: %#v", client.batchWriteReq)
	}
	if client.batchWriteReq.OperationLabel != "导入目录" || client.batchWriteReq.OperationName != "install_capability_bundle" {
		t.Fatalf("operation metadata not forwarded: %#v", client.batchWriteReq)
	}
}

func TestRuntimeWorkspaceBridgeBatchWriteFilesRequiresRuntimeBinding(t *testing.T) {
	appRepo := newRuntimeWorkspaceBridgeTestRepo(t)
	if err := appRepo.GetDB().Create(&model.App{
		User: "alice",
		Code: "demo",
		Name: "Demo",
	}).Error; err != nil {
		t.Fatalf("create app: %v", err)
	}
	client := &fakeRuntimeWorkspaceClient{}
	bridge := newRuntimeWorkspaceBridge(appRepo, client)

	_, _, err := bridge.batchWriteFiles(context.Background(), &dto.BatchWriteFilesReq{
		User: "alice",
		App:  "demo",
	})
	if err == nil {
		t.Fatal("expected runtime binding error")
	}
	if !strings.Contains(err.Error(), "应用未关联 runtime") {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.batchWriteReq != nil {
		t.Fatalf("runtime should not be called without HostID, got %#v", client.batchWriteReq)
	}
}

func TestRuntimeWorkspaceBridgeReadDirectoryFilesUsesRuntimeBoundHost(t *testing.T) {
	appRepo := newRuntimeWorkspaceBridgeTestRepo(t)
	app := createRuntimeWorkspaceBridgeTestApp(t, appRepo, 42)
	client := &fakeRuntimeWorkspaceClient{
		readDirResp: &dto.ReadDirectoryFilesRuntimeResp{
			Success: true,
			Files: []dto.DirectoryFileInfo{
				{FileName: "handler", RelativePath: "handler.go", Content: "package ticket"},
			},
		},
	}
	bridge := newRuntimeWorkspaceBridge(appRepo, client)

	appModel, resp, err := bridge.readDirectoryFiles(context.Background(), app.ID, "/alice/demo/ticket")
	if err != nil {
		t.Fatalf("readDirectoryFiles error = %v", err)
	}
	if appModel.ID != app.ID || appModel.HostID != 42 {
		t.Fatalf("unexpected app model: %#v", appModel)
	}
	if !resp.Success || len(resp.Files) != 1 || resp.Files[0].RelativePath != "handler.go" {
		t.Fatalf("unexpected runtime response: %#v", resp)
	}
	if client.readDirHostID != 42 {
		t.Fatalf("runtime hostID = %d, want 42", client.readDirHostID)
	}
	if client.readDirReq == nil || client.readDirReq.User != "alice" || client.readDirReq.App != "demo" || client.readDirReq.DirectoryPath != "/alice/demo/ticket" {
		t.Fatalf("runtime read directory request not forwarded: %#v", client.readDirReq)
	}
}

func TestRuntimeWorkspaceBridgeReadDirectoryFilesRequiresRuntimeBinding(t *testing.T) {
	appRepo := newRuntimeWorkspaceBridgeTestRepo(t)
	app := createRuntimeWorkspaceBridgeTestApp(t, appRepo, 0)
	client := &fakeRuntimeWorkspaceClient{}
	bridge := newRuntimeWorkspaceBridge(appRepo, client)

	_, _, err := bridge.readDirectoryFiles(context.Background(), app.ID, "/alice/demo/ticket")
	if err == nil {
		t.Fatal("expected runtime binding error")
	}
	if !strings.Contains(err.Error(), "应用未关联 runtime") {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.readDirReq != nil {
		t.Fatalf("runtime should not be called without HostID, got %#v", client.readDirReq)
	}
}

func TestRuntimeWorkspaceBridgeDeleteDirectoryScaffoldUsesRuntimeBoundHost(t *testing.T) {
	appRepo := newRuntimeWorkspaceBridgeTestRepo(t)
	app := createRuntimeWorkspaceBridgeTestApp(t, appRepo, 42)
	client := &fakeRuntimeWorkspaceClient{
		deleteTreeResp: &dto.DeleteServiceTreeRuntimeResp{Success: true},
	}
	bridge := newRuntimeWorkspaceBridge(appRepo, client)

	appModel, resp, err := bridge.deleteDirectoryScaffold(context.Background(), app.ID, "ticket/sub")
	if err != nil {
		t.Fatalf("deleteDirectoryScaffold error = %v", err)
	}
	if appModel.ID != app.ID || appModel.HostID != 42 {
		t.Fatalf("unexpected app model: %#v", appModel)
	}
	if resp == nil || !resp.Success {
		t.Fatalf("unexpected runtime response: %#v", resp)
	}
	if client.deleteTreeHostID != 42 {
		t.Fatalf("runtime hostID = %d, want 42", client.deleteTreeHostID)
	}
	if client.deleteTreeReq == nil || client.deleteTreeReq.User != "alice" || client.deleteTreeReq.App != "demo" || client.deleteTreeReq.PackagePath != "ticket/sub" {
		t.Fatalf("runtime delete tree request not forwarded: %#v", client.deleteTreeReq)
	}
}

func TestDeletePackageKeepsServiceTreeWhenRuntimeCleanupFails(t *testing.T) {
	appRepo := newRuntimeWorkspaceBridgeTestRepo(t)
	app := createRuntimeWorkspaceBridgeTestApp(t, appRepo, 42)
	db := appRepo.GetDB()
	if err := db.AutoMigrate(&model.ServiceTree{}); err != nil {
		t.Fatalf("migrate service tree: %v", err)
	}
	node := &model.ServiceTree{
		AppID: app.ID, Type: model.ServiceTreeTypePackage, Code: "ticket", FullCodePath: "/alice/demo/ticket",
	}
	if err := db.Create(node).Error; err != nil {
		t.Fatalf("create service tree: %v", err)
	}

	client := &fakeRuntimeWorkspaceClient{
		deleteTreeResp: &dto.DeleteServiceTreeRuntimeResp{Success: false, Error: "database cleanup failed"},
	}
	serviceTreeRepo := repository.NewServiceTreeRepository(db)
	mutation := newServiceTreeMutationService(
		serviceTreeRepo,
		appRepo,
		newRuntimeWorkspaceBridge(appRepo, client),
		nil,
		nil,
	)

	err := mutation.DeletePackage(context.Background(), node.ID)
	if err == nil || !strings.Contains(err.Error(), "database cleanup failed") {
		t.Fatalf("expected runtime cleanup error, got %v", err)
	}
	if _, err := serviceTreeRepo.GetServiceTreeByID(node.ID); err != nil {
		t.Fatalf("service tree should remain after runtime cleanup failure: %v", err)
	}
}
