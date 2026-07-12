package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type personalWorkspaceRuntimeClient struct {
	mu            sync.Mutex
	createReqs    []*dto.CreateAppReq
	deleteReqs    []*dto.DeleteAppRuntimeReq
	createErr     error
	createStarted chan struct{}
	releaseCreate <-chan struct{}
	startOnce     sync.Once
}

func (c *personalWorkspaceRuntimeClient) CreateApp(_ context.Context, _ int64, req *dto.CreateAppReq) (*dto.CreateAppResp, error) {
	copyReq := *req
	c.mu.Lock()
	c.createReqs = append(c.createReqs, &copyReq)
	c.mu.Unlock()
	if c.createStarted != nil {
		c.startOnce.Do(func() { close(c.createStarted) })
	}
	if c.releaseCreate != nil {
		<-c.releaseCreate
	}
	if c.createErr != nil {
		return nil, c.createErr
	}
	return &dto.CreateAppResp{User: req.User, App: req.Code}, nil
}

func (c *personalWorkspaceRuntimeClient) UpdateApp(context.Context, int64, *dto.UpdateAppRuntimeReq) (*dto.UpdateAppResp, error) {
	return &dto.UpdateAppResp{}, nil
}

func (c *personalWorkspaceRuntimeClient) RequestApp(context.Context, int64, *dto.RequestAppReq) (*dto.RequestAppResp, error) {
	return &dto.RequestAppResp{}, nil
}

func (c *personalWorkspaceRuntimeClient) DeleteApp(_ context.Context, _ int64, req *dto.DeleteAppRuntimeReq) (*dto.DeleteAppResp, error) {
	c.mu.Lock()
	c.deleteReqs = append(c.deleteReqs, req)
	c.mu.Unlock()
	return &dto.DeleteAppResp{}, nil
}

func (c *personalWorkspaceRuntimeClient) createCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.createReqs)
}

func (c *personalWorkspaceRuntimeClient) firstCreateReq() *dto.CreateAppReq {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.createReqs) == 0 {
		return nil
	}
	copyReq := *c.createReqs[0]
	return &copyReq
}

func (c *personalWorkspaceRuntimeClient) deleteCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.deleteReqs)
}

func newPersonalWorkspaceBootstrapTestDeps(t *testing.T) (*AppService, *repository.AppRepository, *gorm.DB, *personalWorkspaceRuntimeClient) {
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
	if err := db.AutoMigrate(&model.Host{}, &model.App{}, &model.ServiceTree{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Create(&model.Host{Domain: "bootstrap-test", NatsID: 9, Status: "enabled"}).Error; err != nil {
		t.Fatalf("create host: %v", err)
	}

	appRepo := repository.NewAppRepository(db)
	runtimeClient := &personalWorkspaceRuntimeClient{}
	service := NewAppService(runtimeClient, appRepo, nil, repository.NewServiceTreeRepository(db), nil)
	return service, appRepo, db, runtimeClient
}

func TestBootstrapPersonalWorkspaceCreatesPrivateHome(t *testing.T) {
	service, appRepo, db, runtimeClient := newPersonalWorkspaceBootstrapTestDeps(t)

	resp, err := service.BootstrapPersonalWorkspace(context.Background(), "alice")
	if err != nil {
		t.Fatalf("BootstrapPersonalWorkspace() error = %v", err)
	}
	if !resp.Created {
		t.Fatal("Created = false, want true")
	}
	if resp.App.User != "alice" || resp.App.Code != PersonalWorkspaceCode || resp.App.Name != PersonalWorkspaceName {
		t.Fatalf("unexpected app: %#v", resp.App)
	}
	if resp.App.IsPublic {
		t.Fatalf("personal workspace must be private: %#v", resp.App)
	}
	if runtimeClient.createCount() != 1 {
		t.Fatalf("runtime create count = %d, want 1", runtimeClient.createCount())
	}
	req := runtimeClient.firstCreateReq()
	if req == nil || req.User != "alice" || req.Code != PersonalWorkspaceCode || req.Name != PersonalWorkspaceName || req.IsPublic == nil || *req.IsPublic {
		t.Fatalf("unexpected runtime create request: %#v", req)
	}

	app, err := appRepo.GetAppByUserName("alice", PersonalWorkspaceCode)
	if err != nil {
		t.Fatalf("load home: %v", err)
	}
	if app.IsPublic {
		t.Fatalf("persisted personal workspace must be private: %#v", app)
	}
	if !app.IsPersonalWorkspace {
		t.Fatalf("persisted personal workspace must be marked: %#v", app)
	}
	var root model.ServiceTree
	if err := db.Where("app_id = ? AND full_code_path = ?", app.ID, "/alice/home").First(&root).Error; err != nil {
		t.Fatalf("load root node: %v", err)
	}
	if root.Name != PersonalWorkspaceName || root.Code != PersonalWorkspaceCode || root.Type != model.ServiceTreeTypePackage {
		t.Fatalf("unexpected root: %#v", root)
	}
}

func TestPersistCreatedAppKeepsExplicitPrivateVisibility(t *testing.T) {
	service, appRepo, _, _ := newPersonalWorkspaceBootstrapTestDeps(t)
	private := false
	app, rootNode := service.buildInitialAppAndRoot("alice", "alice", &dto.CreateAppReq{
		User:     "alice",
		Code:     "private_space",
		Name:     "私有空间",
		IsPublic: &private,
	}, &model.Host{NatsID: 9, Status: "enabled"})
	if err := service.persistCreatedApp(context.Background(), app, rootNode); err != nil {
		t.Fatalf("persistCreatedApp() error = %v", err)
	}
	persisted, err := appRepo.GetAppByUserName("alice", "private_space")
	if err != nil {
		t.Fatalf("load private workspace: %v", err)
	}
	if persisted.IsPublic {
		t.Fatalf("persisted workspace must keep explicit is_public=false: %#v", persisted)
	}
}

func TestBootstrapPersonalWorkspaceIsIdempotentForExistingHome(t *testing.T) {
	service, appRepo, _, runtimeClient := newPersonalWorkspaceBootstrapTestDeps(t)
	if err := appRepo.CreateApp(&model.App{
		User:     "alice",
		Code:     PersonalWorkspaceCode,
		Name:     "旧的个人空间",
		IsPublic: true,
		Status:   "enabled",
	}); err != nil {
		t.Fatalf("create existing home: %v", err)
	}

	resp, err := service.BootstrapPersonalWorkspace(context.Background(), "alice")
	if err != nil {
		t.Fatalf("BootstrapPersonalWorkspace() error = %v", err)
	}
	if resp.Created {
		t.Fatal("Created = true, want false")
	}
	if resp.App.Code != PersonalWorkspaceCode || resp.App.Name != "旧的个人空间" || !resp.App.IsPublic {
		t.Fatalf("existing home should not be rewritten: %#v", resp.App)
	}
	if runtimeClient.createCount() != 0 {
		t.Fatalf("runtime create count = %d, want 0", runtimeClient.createCount())
	}
}

func TestBootstrapPersonalWorkspaceReturnsExistingOwnedWorkspaceWithoutCreatingHome(t *testing.T) {
	service, appRepo, _, runtimeClient := newPersonalWorkspaceBootstrapTestDeps(t)
	if err := appRepo.CreateApp(&model.App{
		User:   "alice",
		Code:   "operations",
		Name:   "运营空间",
		Status: "enabled",
	}); err != nil {
		t.Fatalf("create existing workspace: %v", err)
	}

	resp, err := service.BootstrapPersonalWorkspace(context.Background(), "alice")
	if err != nil {
		t.Fatalf("BootstrapPersonalWorkspace() error = %v", err)
	}
	if resp.Created || resp.App.Code != "operations" {
		t.Fatalf("unexpected fallback response: %#v", resp)
	}
	if runtimeClient.createCount() != 0 {
		t.Fatalf("runtime create count = %d, want 0", runtimeClient.createCount())
	}
	if _, err := appRepo.GetAppByUserName("alice", PersonalWorkspaceCode); !errorsIsRecordNotFound(err) {
		t.Fatalf("home should not be created, err = %v", err)
	}
}

func TestBootstrapPersonalWorkspaceCoalescesConcurrentRequests(t *testing.T) {
	service, appRepo, db, runtimeClient := newPersonalWorkspaceBootstrapTestDeps(t)
	releaseCreate := make(chan struct{})
	runtimeClient.createStarted = make(chan struct{})
	runtimeClient.releaseCreate = releaseCreate

	const callers = 8
	responses := make(chan error, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := service.BootstrapPersonalWorkspace(context.Background(), "alice")
			if err == nil && (resp == nil || resp.App.Code != PersonalWorkspaceCode) {
				err = fmt.Errorf("unexpected response: %#v", resp)
			}
			responses <- err
		}()
	}

	<-runtimeClient.createStarted
	close(releaseCreate)
	wg.Wait()
	close(responses)
	for err := range responses {
		if err != nil {
			t.Fatal(err)
		}
	}
	if runtimeClient.createCount() != 1 {
		t.Fatalf("runtime create count = %d, want 1", runtimeClient.createCount())
	}
	var count int64
	if err := db.Model(&model.App{}).Where("user = ? AND code = ?", "alice", PersonalWorkspaceCode).Count(&count).Error; err != nil {
		t.Fatalf("count homes: %v", err)
	}
	if count != 1 {
		t.Fatalf("home count = %d, want 1", count)
	}
	if _, err := appRepo.GetAppByUserName("alice", PersonalWorkspaceCode); err != nil {
		t.Fatalf("load home: %v", err)
	}
}

func TestBootstrapPersonalWorkspaceRejectsMissingUser(t *testing.T) {
	service, _, _, runtimeClient := newPersonalWorkspaceBootstrapTestDeps(t)
	if _, err := service.BootstrapPersonalWorkspace(context.Background(), " "); err == nil {
		t.Fatal("expected missing user error")
	}
	if runtimeClient.createCount() != 0 {
		t.Fatalf("runtime create count = %d, want 0", runtimeClient.createCount())
	}
}

func TestDeleteAppRejectsPersonalWorkspace(t *testing.T) {
	service, appRepo, _, runtimeClient := newPersonalWorkspaceBootstrapTestDeps(t)
	if err := appRepo.CreateApp(&model.App{
		User:                "alice",
		Code:                PersonalWorkspaceCode,
		Name:                PersonalWorkspaceName,
		Status:              "enabled",
		IsPersonalWorkspace: true,
	}); err != nil {
		t.Fatalf("create home: %v", err)
	}

	_, err := service.DeleteApp(context.Background(), &dto.DeleteAppReq{ResourcePath: "/alice/home"})
	if err == nil || !strings.Contains(err.Error(), "不支持删除") {
		t.Fatalf("DeleteApp() error = %v, want default-workspace protection", err)
	}
	if runtimeClient.deleteCount() != 0 {
		t.Fatalf("runtime delete count = %d, want 0", runtimeClient.deleteCount())
	}
	if _, err := appRepo.GetAppByUserName("alice", PersonalWorkspaceCode); err != nil {
		t.Fatalf("home should remain: %v", err)
	}
}

func TestDeleteAppAllowsLegacyWorkspaceUsingHomeCode(t *testing.T) {
	service, appRepo, _, runtimeClient := newPersonalWorkspaceBootstrapTestDeps(t)
	if err := appRepo.CreateApp(&model.App{
		User:   "alice",
		Code:   PersonalWorkspaceCode,
		Name:   "旧 home 空间",
		Status: "enabled",
	}); err != nil {
		t.Fatalf("create legacy home: %v", err)
	}

	if _, err := service.DeleteApp(context.Background(), &dto.DeleteAppReq{ResourcePath: "/alice/home"}); err != nil {
		t.Fatalf("DeleteApp legacy home error: %v", err)
	}
	if runtimeClient.deleteCount() != 1 {
		t.Fatalf("runtime delete count = %d, want 1", runtimeClient.deleteCount())
	}
	if _, err := appRepo.GetAppByUserName("alice", PersonalWorkspaceCode); !errorsIsRecordNotFound(err) {
		t.Fatalf("legacy home should be deleted, err = %v", err)
	}
}

func errorsIsRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
