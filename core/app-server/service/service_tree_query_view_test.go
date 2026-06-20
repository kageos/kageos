package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/scheduledsdk"
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

type fakeDirectoryOverviewScheduleClient struct {
	tasks map[string][]*scheduledsdk.Task
}

func (f fakeDirectoryOverviewScheduleClient) ListTasks(_ context.Context, req scheduledsdk.ListTasksRequest) (*scheduledsdk.ListTasksResponse, error) {
	key := strings.Join([]string{req.ExecutorKey, req.ResourceScope, req.ResourceKey}, "|")
	list := f.tasks[key]
	return &scheduledsdk.ListTasksResponse{List: list, Total: int64(len(list))}, nil
}

func TestGetDirectoryOverviewAggregatesResourcesAndScheduledTasks(t *testing.T) {
	queryView, db, app := newServiceTreeQueryViewTest(t)
	ctx := context.WithValue(context.Background(), contextx.RequestUserHeader, "alice")
	if _, err := ReconcileAppRootServiceTrees(ctx, queryView.appRepo, queryView.serviceTreeRepo); err != nil {
		t.Fatalf("ReconcileAppRootServiceTrees: %v", err)
	}

	hr := &model.ServiceTree{
		Name:         "人事",
		Code:         "hr",
		Type:         model.ServiceTreeTypePackage,
		AppID:        app.ID,
		FullCodePath: "/alice/ops/hr",
	}
	remind := &model.ServiceTree{
		Name:         "提醒表单",
		Code:         "remind.form",
		Type:         model.ServiceTreeTypeFunction,
		AppID:        app.ID,
		FullCodePath: "/alice/ops/hr/remind.form",
		TemplateType: "form",
		RunCount:     7,
	}
	readme := &model.ServiceTree{
		Name:         "说明",
		Code:         "readme",
		Type:         model.ServiceTreeTypeDocs,
		AppID:        app.ID,
		FullCodePath: "/alice/ops/hr/readme",
	}
	if err := db.Create(hr).Error; err != nil {
		t.Fatalf("create hr: %v", err)
	}
	if err := db.Create(remind).Error; err != nil {
		t.Fatalf("create remind: %v", err)
	}
	if err := db.Create(readme).Error; err != nil {
		t.Fatalf("create readme: %v", err)
	}

	nextRun := time.Now().Add(time.Hour).Truncate(time.Second)
	oldClientFactory := newServiceTreeScheduleClient
	newServiceTreeScheduleClient = func() serviceTreeScheduleClient {
		return fakeDirectoryOverviewScheduleClient{tasks: map[string][]*scheduledsdk.Task{
			"app.function|function|/alice/ops/hr/remind.form": {
				{
					ID:          11,
					Title:       "提醒巡检",
					ExecutorKey: ScheduledFunctionExecutorKey,
					Status:      scheduledsdk.TaskStatusPending,
					Schedule:    scheduledsdk.Every(60),
					NextRunAt:   &nextRun,
					ResourceKey: "/alice/ops/hr/remind.form",
				},
			},
			"agent.session|workspace_directory|/alice/ops/hr": {
				{
					ID:                  21,
					Title:               "日报会话",
					ExecutorKey:         "agent.session",
					Status:              scheduledsdk.TaskStatusPending,
					Schedule:            scheduledsdk.Cron("0 9 * * *"),
					NextRunAt:           &nextRun,
					InflightExecutionID: 99,
					ResourceKey:         "/alice/ops/hr",
				},
			},
		}}
	}
	defer func() { newServiceTreeScheduleClient = oldClientFactory }()

	resp, err := queryView.GetDirectoryOverview(ctx, &dto.GetDirectoryOverviewReq{FullCodePath: "/alice/ops"})
	if err != nil {
		t.Fatalf("GetDirectoryOverview: %v", err)
	}
	if resp.Stats.Directories != 1 || resp.Stats.Functions != 1 || resp.Stats.Docs != 1 {
		t.Fatalf("unexpected resource stats: %+v", resp.Stats)
	}
	if resp.Stats.TotalRunCount != 7 {
		t.Fatalf("unexpected total run count: %d", resp.Stats.TotalRunCount)
	}
	if resp.Stats.ScheduledFunctionTasks != 1 || resp.Stats.ScheduledAgentTasks != 1 || resp.Stats.RunningTasks != 1 {
		t.Fatalf("unexpected scheduled stats: %+v", resp.Stats)
	}
	if len(resp.ScheduledFunctionTasks) != 1 || resp.ScheduledFunctionTasks[0].ResourcePath != "/alice/ops/hr/remind.form" {
		t.Fatalf("unexpected function task resources: %+v", resp.ScheduledFunctionTasks)
	}
	if len(resp.ScheduledAgentTasks) != 1 || resp.ScheduledAgentTasks[0].ResourcePath != "/alice/ops/hr" {
		t.Fatalf("unexpected agent task resources: %+v", resp.ScheduledAgentTasks)
	}
}
