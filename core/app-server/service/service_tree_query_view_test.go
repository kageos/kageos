package service

import (
	"context"
	"strings"
	"testing"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newServiceTreeQueryViewTest(t *testing.T) (*serviceTreeQueryView, *gorm.DB, *model.App) {
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
	app := &model.App{User: "alice", Code: "ops", Name: "Ops", Version: "v3", Admins: "alice"}
	app.CreatedBy = "alice"
	if err := appRepo.CreateApp(app); err != nil {
		t.Fatalf("create app: %v", err)
	}

	serviceTreeRepo := repository.NewServiceTreeRepository(db)
	return newServiceTreeQueryView(serviceTreeRepo, appRepo, nil), db, app
}

func TestReconcileAppRootServiceTreesCreatesMissingRoot(t *testing.T) {
	queryView, db, app := newServiceTreeQueryViewTest(t)
	ctx := context.WithValue(context.Background(), contextx.RequestUserHeader, "alice")

	result, err := ReconcileAppRootServiceTrees(ctx, queryView.appRepo, queryView.serviceTreeRepo)
	if err != nil {
		t.Fatalf("ReconcileAppRootServiceTrees: %v", err)
	}
	if result.Checked != 1 || result.Created != 1 || result.Updated != 0 {
		t.Fatalf("unexpected reconcile result: %+v", result)
	}

	resp, err := queryView.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{FullCodePath: "/alice/ops"})
	if err != nil {
		t.Fatalf("GetServiceTreeDetail: %v", err)
	}
	if resp.ID == 0 {
		t.Fatal("expected repaired root to have service_tree id")
	}
	if resp.FullCodePath != "/alice/ops" || resp.AppID != app.ID || resp.RefID != app.ID {
		t.Fatalf("unexpected root detail: %+v app_id=%d", resp, app.ID)
	}
	if resp.Name != "Ops" || resp.Code != "ops" || resp.Type != model.ServiceTreeTypePackage || resp.VersionNum != 3 {
		t.Fatalf("unexpected repaired metadata: %+v", resp)
	}

	respAgain, err := queryView.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{FullCodePath: "/alice/ops"})
	if err != nil {
		t.Fatalf("GetServiceTreeDetail again: %v", err)
	}
	if respAgain.ID != resp.ID {
		t.Fatalf("expected existing root reused, first id=%d second id=%d", resp.ID, respAgain.ID)
	}

	var count int64
	if err := db.Model(&model.ServiceTree{}).Where("full_code_path = ?", "/alice/ops").Count(&count).Error; err != nil {
		t.Fatalf("count roots: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one repaired root, got %d", count)
	}
}

func TestGetServiceTreeDetailReportsMissingAppRootInvariant(t *testing.T) {
	queryView, _, _ := newServiceTreeQueryViewTest(t)

	_, err := queryView.GetServiceTreeDetail(context.Background(), &dto.GetServiceTreeDetailReq{FullCodePath: "/alice/ops"})
	if err == nil {
		t.Fatal("expected missing root invariant error")
	}
	if !strings.Contains(err.Error(), "工作空间根节点缺失") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetServiceTreeDetailDoesNotRepairNestedMissingPath(t *testing.T) {
	queryView, _, _ := newServiceTreeQueryViewTest(t)
	ctx := context.WithValue(context.Background(), contextx.RequestUserHeader, "alice")

	_, err := queryView.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{FullCodePath: "/alice/ops/missing"})
	if err == nil {
		t.Fatal("expected nested missing path to fail")
	}
	if !strings.Contains(err.Error(), "服务目录不存在") {
		t.Fatalf("unexpected error: %v", err)
	}
}
