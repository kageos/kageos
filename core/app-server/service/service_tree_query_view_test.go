package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newServiceTreeQueryViewTest(t *testing.T) (*serviceTreeQueryView, *gorm.DB, *model.App) {
	t.Helper()
	oldClientFactory := newServiceTreeScheduleClient
	newServiceTreeScheduleClient = func() serviceTreeScheduleClient {
		return fakeDirectoryOverviewScheduleClient{}
	}
	t.Cleanup(func() { newServiceTreeScheduleClient = oldClientFactory })

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
	if err := appRepo.CreateApp(context.Background(), app); err != nil {
		t.Fatalf("create app: %v", err)
	}

	serviceTreeRepo := repository.NewServiceTreeRepository(db)
	return newServiceTreeQueryView(serviceTreeRepo, appRepo, nil), db, app
}

func newServiceTreeQueryViewAccessTest(t *testing.T, hideUnauthorizedNodes bool) (*serviceTreeQueryView, *gorm.DB, *model.App, *TeamAccessService) {
	t.Helper()
	oldClientFactory := newServiceTreeScheduleClient
	newServiceTreeScheduleClient = func() serviceTreeScheduleClient {
		return fakeDirectoryOverviewScheduleClient{}
	}
	t.Cleanup(func() { newServiceTreeScheduleClient = oldClientFactory })

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
	if err := db.AutoMigrate(&model.App{}, &model.ServiceTree{}, &model.WorkspaceRoleAssignment{}, &model.OperateLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	appRepo := repository.NewAppRepository(db)
	app := &model.App{
		User:                  "alice",
		Code:                  "ops",
		Name:                  "Ops",
		Version:               "v1",
		Admins:                "alice",
		HideUnauthorizedNodes: hideUnauthorizedNodes,
	}
	app.CreatedBy = "alice"
	if err := appRepo.CreateApp(context.Background(), app); err != nil {
		t.Fatalf("create app: %v", err)
	}

	serviceTreeRepo := repository.NewServiceTreeRepository(db)
	teamAccess := NewTeamAccessService(
		repository.NewTeamAccessRepository(db),
		repository.NewOperateLogRepository(db),
		appRepo,
	)
	return newServiceTreeQueryView(serviceTreeRepo, appRepo, teamAccess), db, app, teamAccess
}

func seedServiceTreeVisibilityNodes(t *testing.T, db *gorm.DB, app *model.App) {
	t.Helper()
	nodes := []*model.ServiceTree{
		{
			Name:         "Ops",
			Code:         "ops",
			Type:         model.ServiceTreeTypePackage,
			AppID:        app.ID,
			RefID:        app.ID,
			FullCodePath: "/alice/ops",
		},
		{
			Name:         "人事",
			Code:         "hr",
			Type:         model.ServiceTreeTypePackage,
			AppID:        app.ID,
			FullCodePath: "/alice/ops/hr",
		},
		{
			Name:         "说明",
			Code:         "readme",
			Type:         model.ServiceTreeTypeDocs,
			AppID:        app.ID,
			FullCodePath: "/alice/ops/hr/readme",
		},
		{
			Name:         "保密目录",
			Code:         "secret",
			Type:         model.ServiceTreeTypePackage,
			AppID:        app.ID,
			FullCodePath: "/alice/ops/secret",
		},
	}
	if err := db.Create(&nodes).Error; err != nil {
		t.Fatalf("seed service tree nodes: %v", err)
	}
}

func findServiceTreeRespByPath(nodes []*dto.GetServiceTreeResp, path string) *dto.GetServiceTreeResp {
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if node.FullCodePath == path {
			return node
		}
		if found := findServiceTreeRespByPath(node.Children, path); found != nil {
			return found
		}
	}
	return nil
}

func TestGetAppWithServiceTreeAnnotatesScheduledAgentTaskBadges(t *testing.T) {
	queryView, db, app := newServiceTreeQueryViewTest(t)
	ctx := context.WithValue(context.Background(), contextx.RequestUserHeader, "alice")
	if _, err := ReconcileAppRootServiceTrees(ctx, queryView.appRepo, queryView.serviceTreeRepo); err != nil {
		t.Fatalf("ReconcileAppRootServiceTrees: %v", err)
	}

	nodes := []*model.ServiceTree{
		{
			Name:         "人事",
			Code:         "hr",
			Type:         model.ServiceTreeTypePackage,
			AppID:        app.ID,
			FullCodePath: "/alice/ops/hr",
		},
		{
			Name:         "薪资",
			Code:         "payroll",
			Type:         model.ServiceTreeTypePackage,
			AppID:        app.ID,
			FullCodePath: "/alice/ops/hr/payroll",
		},
	}
	if err := db.Create(&nodes).Error; err != nil {
		t.Fatalf("create package nodes: %v", err)
	}

	oldClientFactory := newServiceTreeScheduleClient
	newServiceTreeScheduleClient = func() serviceTreeScheduleClient {
		return fakeDirectoryOverviewScheduleClient{tasks: map[string][]*scheduledsdk.Task{
			"agent.session|workspace_directory|/alice/ops/hr": {
				{ID: 21, ExecutorKey: "agent.session", Status: scheduledsdk.TaskStatusPaused, ResourceKey: "/alice/ops/hr"},
			},
			"agent.session|workspace_directory|/alice/ops/hr/payroll": {
				{ID: 31, ExecutorKey: "agent.session", Status: scheduledsdk.TaskStatusPaused, ResourceKey: "/alice/ops/hr/payroll"},
				{ID: 32, ExecutorKey: "agent.session", Status: scheduledsdk.TaskStatusPending, ResourceKey: "/alice/ops/hr/payroll"},
			},
		}}
	}
	defer func() { newServiceTreeScheduleClient = oldClientFactory }()

	resp, err := queryView.GetAppWithServiceTree(ctx, &dto.GetAppWithServiceTreeReq{ResourcePath: "/alice/ops"})
	if err != nil {
		t.Fatalf("GetAppWithServiceTree: %v", err)
	}

	root := findServiceTreeRespByPath(resp.ServiceTree, "/alice/ops")
	hr := findServiceTreeRespByPath(resp.ServiceTree, "/alice/ops/hr")
	payroll := findServiceTreeRespByPath(resp.ServiceTree, "/alice/ops/hr/payroll")
	if root == nil || hr == nil || payroll == nil {
		t.Fatalf("expected root/hr/payroll nodes, got root=%v hr=%v payroll=%v", root, hr, payroll)
	}
	if root.ScheduledAgentTasks != 3 || hr.ScheduledAgentTasks != 3 || payroll.ScheduledAgentTasks != 2 {
		t.Fatalf("unexpected scheduled agent task badges: root=%d hr=%d payroll=%d", root.ScheduledAgentTasks, hr.ScheduledAgentTasks, payroll.ScheduledAgentTasks)
	}
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

func TestBatchGetServiceTreeDetailsReturnsReadableItemsInRequestOrder(t *testing.T) {
	queryView, db, app := newServiceTreeQueryViewTest(t)
	ctx := context.WithValue(context.Background(), contextx.RequestUserHeader, "alice")
	nodes := []*model.ServiceTree{
		{
			Name:         "Ops",
			Code:         "ops",
			Type:         model.ServiceTreeTypePackage,
			AppID:        app.ID,
			RefID:        app.ID,
			FullCodePath: "/alice/ops",
		},
		{
			Name:         "订单",
			Code:         "orders",
			Type:         model.ServiceTreeTypeFunction,
			TemplateType: "table",
			Description:  "订单表",
			AppID:        app.ID,
			FullCodePath: "/alice/ops/orders.table",
			RunCount:     12,
		},
		{
			Name:         "说明",
			Code:         "readme",
			Type:         model.ServiceTreeTypeDocs,
			AppID:        app.ID,
			FullCodePath: "/alice/ops/readme.docs",
		},
	}
	if err := db.Create(&nodes).Error; err != nil {
		t.Fatalf("seed service tree nodes: %v", err)
	}

	resp, err := queryView.BatchGetServiceTreeDetails(ctx, &dto.BatchGetServiceTreeDetailsReq{
		FullCodePaths: []string{"/alice/ops/readme.docs", "alice/ops/orders.table", "/alice/ops/orders.table", "/alice/ops/missing"},
	})
	if err != nil {
		t.Fatalf("BatchGetServiceTreeDetails: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("items len = %d, want 2; resp=%+v", len(resp.Items), resp)
	}
	if resp.Items[0].FullCodePath != "/alice/ops/readme.docs" || resp.Items[1].FullCodePath != "/alice/ops/orders.table" {
		t.Fatalf("items should follow deduped request order, got %#v", resp.Items)
	}
	if resp.Items[1].TemplateType != "table" || resp.Items[1].RunCount != 12 || resp.Items[1].Description != "订单表" {
		t.Fatalf("function detail mismatch: %+v", resp.Items[1])
	}
	if len(resp.Missing) != 1 || resp.Missing[0] != "/alice/ops/missing" {
		t.Fatalf("missing = %#v, want [/alice/ops/missing]", resp.Missing)
	}
}

func TestBatchGetServiceTreeDetailsFiltersUnreadableItems(t *testing.T) {
	queryView, db, app, teamAccess := newServiceTreeQueryViewAccessTest(t, false)
	seedServiceTreeVisibilityNodes(t, db, app)
	if err := teamAccess.Assign(actorContext("alice"), access.AssignRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Username:     "bob",
		ResourcePath: "/alice/ops/hr/readme",
		RoleCode:     access.RoleViewer,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatalf("assign read permission: %v", err)
	}

	resp, err := queryView.BatchGetServiceTreeDetails(actorContext("bob"), &dto.BatchGetServiceTreeDetailsReq{
		FullCodePaths: []string{"/alice/ops/hr/readme", "/alice/ops/secret"},
	})
	if err != nil {
		t.Fatalf("BatchGetServiceTreeDetails: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].FullCodePath != "/alice/ops/hr/readme" {
		t.Fatalf("items = %#v, want only readable readme", resp.Items)
	}
	if len(resp.Missing) != 1 || resp.Missing[0] != "/alice/ops/secret" {
		t.Fatalf("missing = %#v, want unreadable secret", resp.Missing)
	}
}

func TestGetAppWithServiceTreeKeepsUnauthorizedNodesByDefault(t *testing.T) {
	queryView, db, app, teamAccess := newServiceTreeQueryViewAccessTest(t, false)
	seedServiceTreeVisibilityNodes(t, db, app)
	if err := teamAccess.Assign(actorContext("alice"), access.AssignRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Username:     "bob",
		ResourcePath: "/alice/ops/hr/readme",
		RoleCode:     access.RoleViewer,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}

	ctx := context.WithValue(context.Background(), contextx.RequestUserHeader, "bob")
	resp, err := queryView.GetAppWithServiceTree(ctx, &dto.GetAppWithServiceTreeReq{ResourcePath: "/alice/ops"})
	if err != nil {
		t.Fatalf("GetAppWithServiceTree: %v", err)
	}
	if findServiceTreeRespByPath(resp.ServiceTree, "/alice/ops/secret") == nil {
		t.Fatal("default tree should keep unauthorized nodes visible")
	}
}

func TestGetAppWithServiceTreeHidesUnauthorizedNodesWhenEnabled(t *testing.T) {
	queryView, db, app, teamAccess := newServiceTreeQueryViewAccessTest(t, true)
	seedServiceTreeVisibilityNodes(t, db, app)
	if err := teamAccess.Assign(actorContext("alice"), access.AssignRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Username:     "bob",
		ResourcePath: "/alice/ops/hr/readme",
		RoleCode:     access.RoleViewer,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}

	ctx := context.WithValue(context.Background(), contextx.RequestUserHeader, "bob")
	resp, err := queryView.GetAppWithServiceTree(ctx, &dto.GetAppWithServiceTreeReq{ResourcePath: "/alice/ops"})
	if err != nil {
		t.Fatalf("GetAppWithServiceTree: %v", err)
	}
	if findServiceTreeRespByPath(resp.ServiceTree, "/alice/ops/secret") != nil {
		t.Fatal("unauthorized branch should be hidden")
	}
	if findServiceTreeRespByPath(resp.ServiceTree, "/alice/ops/hr/readme") == nil {
		t.Fatal("authorized child should remain visible")
	}
	if findServiceTreeRespByPath(resp.ServiceTree, "/alice/ops/hr") == nil {
		t.Fatal("parent container for an authorized child should remain visible")
	}
}

type fakeDirectoryOverviewScheduleClient struct {
	tasks map[string][]*scheduledsdk.Task
	err   error
}

func (f fakeDirectoryOverviewScheduleClient) ListTasks(_ context.Context, req scheduledsdk.ListTasksRequest) (*scheduledsdk.ListTasksResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	key := strings.Join([]string{req.ExecutorKey, req.ResourceScope, req.ResourceKey}, "|")
	if req.ResourceKeyPrefix != "" {
		key = strings.Join([]string{req.ExecutorKey, req.ResourceScope, req.ResourceKeyPrefix}, "|")
	}
	list := f.tasks[key]
	return &scheduledsdk.ListTasksResponse{List: list, Total: int64(len(list))}, nil
}

func TestGetDirectoryOverviewDeduplicatesScheduleLoadWarnings(t *testing.T) {
	queryView, db, app := newServiceTreeQueryViewTest(t)
	ctx := context.WithValue(context.Background(), contextx.RequestUserHeader, "alice")
	if _, err := ReconcileAppRootServiceTrees(ctx, queryView.appRepo, queryView.serviceTreeRepo); err != nil {
		t.Fatalf("ReconcileAppRootServiceTrees: %v", err)
	}
	if err := db.Create(&model.ServiceTree{
		Name:         "巡检",
		Code:         "sweep.form",
		Type:         model.ServiceTreeTypeFunction,
		AppID:        app.ID,
		FullCodePath: "/alice/ops/sweep.form",
		TemplateType: "form",
	}).Error; err != nil {
		t.Fatalf("create function: %v", err)
	}

	oldClientFactory := newServiceTreeScheduleClient
	newServiceTreeScheduleClient = func() serviceTreeScheduleClient {
		return fakeDirectoryOverviewScheduleClient{err: errors.New("scheduledsdk: http 404: 404 page not found")}
	}
	defer func() { newServiceTreeScheduleClient = oldClientFactory }()

	resp, err := queryView.GetDirectoryOverview(ctx, &dto.GetDirectoryOverviewReq{FullCodePath: "/alice/ops"})
	if err != nil {
		t.Fatalf("GetDirectoryOverview: %v", err)
	}
	if !resp.Partial || len(resp.Warnings) != 1 {
		t.Fatalf("warnings = %#v, partial = %v", resp.Warnings, resp.Partial)
	}
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
			"app.function|function|/alice/ops": {
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
			"agent.session|workspace_directory|/alice/ops": {
				{
					ID:                  21,
					Title:               "日报会话",
					ExecutorKey:         "agent.session",
					Status:              scheduledsdk.TaskStatusPending,
					Schedule:            scheduledsdk.Cron("0 9 * * *"),
					NextRunAt:           &nextRun,
					InflightExecutionID: 99,
					ExecutorPayload:     []byte(`{"message":"每天汇总人事动态"}`),
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
	if got := resp.ScheduledAgentTasks[0].Task.Description; got != "每天汇总人事动态" {
		t.Fatalf("agent task summary not preserved: %q", got)
	}
	if len(resp.ScheduledAgentTasks[0].Task.ExecutorPayload) != 0 {
		t.Fatal("agent task payload should be compacted in directory overview")
	}
}
