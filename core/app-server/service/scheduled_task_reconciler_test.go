package service

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeScheduledTaskReconcileClient struct {
	tasks         []*scheduledsdk.Task
	deleted       []int64
	deleteSources []string
	listErr       error
	deleteErrID   int64
}

func (f *fakeScheduledTaskReconcileClient) ListTasks(_ context.Context, req scheduledsdk.ListTasksRequest) (*scheduledsdk.ListTasksResponse, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	filtered := make([]*scheduledsdk.Task, 0, len(f.tasks))
	for _, task := range f.tasks {
		if task == nil {
			continue
		}
		if req.ResourceKeyPrefix != "" && !scheduledResourcePathMatchesPrefix(task.ResourceKey, req.ResourceKeyPrefix) {
			continue
		}
		filtered = append(filtered, task)
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = len(filtered)
	}
	start := (page - 1) * pageSize
	if start >= len(filtered) {
		return &scheduledsdk.ListTasksResponse{List: nil, Total: int64(len(filtered))}, nil
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	return &scheduledsdk.ListTasksResponse{List: filtered[start:end], Total: int64(len(filtered))}, nil
}

func (f *fakeScheduledTaskReconcileClient) DeleteTask(ctx context.Context, taskID int64) error {
	if taskID == f.deleteErrID {
		return fmt.Errorf("delete failed")
	}
	f.deleted = append(f.deleted, taskID)
	f.deleteSources = append(f.deleteSources, contextx.ResolveClientSource(ctx))
	return nil
}

func scheduledResourcePathMatchesPrefix(path, prefix string) bool {
	return scheduledResourceWithinPrefix(path, prefix)
}

func newScheduledTaskReconcileTestRepo(t *testing.T) (*repository.ServiceTreeRepository, *gorm.DB) {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ServiceTree{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return repository.NewServiceTreeRepository(db), db
}

func TestDeleteTasksByResourcePrefixDeletesNodeAndDescendantsOnly(t *testing.T) {
	client := &fakeScheduledTaskReconcileClient{tasks: []*scheduledsdk.Task{
		{ID: 1, ResourceKey: "/alice/demo/video"},
		{ID: 2, ResourceKey: "/alice/demo/video/refresh.form"},
		{ID: 3, ResourceKey: "/alice/demo/video2/refresh.form"},
	}}
	reconciler := newScheduledTaskReconciler(nil, client)

	deleted, err := reconciler.DeleteTasksByResourcePrefix(context.Background(), "/alice/demo/video/")
	if err != nil {
		t.Fatalf("DeleteTasksByResourcePrefix: %v", err)
	}
	if deleted != 2 || fmt.Sprint(client.deleted) != "[1 2]" {
		t.Fatalf("deleted=%d ids=%v, want 2 and [1 2]", deleted, client.deleted)
	}
	for _, source := range client.deleteSources {
		if source != scheduledTaskSourceManifest {
			t.Fatalf("delete source=%q, want %q", source, scheduledTaskSourceManifest)
		}
	}
}

func TestScheduledResourceWithinPrefixTreatsUnderscoreLiterally(t *testing.T) {
	prefix := "/alice/demo/minimax_hailuo_23"
	if !scheduledResourceWithinPrefix(prefix+"/refresh.form", prefix) {
		t.Fatal("descendant should match resource prefix")
	}
	if scheduledResourceWithinPrefix("/alice/demo/minimaxXhailuoY23/refresh.form", prefix) {
		t.Fatal("underscore must not behave like a SQL LIKE wildcard")
	}
	if scheduledResourceWithinPrefix(prefix+"2/refresh.form", prefix) {
		t.Fatal("adjacent package name must not match resource prefix")
	}
}

func TestReconcileOrphansDeletesMissingOrMismatchedDirectoryResources(t *testing.T) {
	repo, db := newScheduledTaskReconcileTestRepo(t)
	for _, node := range []*model.ServiceTree{
		{Type: model.ServiceTreeTypePackage, FullCodePath: "/alice/demo/video"},
		{Type: model.ServiceTreeTypeFunction, FullCodePath: "/alice/demo/video/refresh.form"},
		{Type: model.ServiceTreeTypePackage, FullCodePath: "/alice/demo/wrong.form"},
	} {
		if err := db.Create(node).Error; err != nil {
			t.Fatalf("create node: %v", err)
		}
	}
	client := &fakeScheduledTaskReconcileClient{tasks: []*scheduledsdk.Task{
		{ID: 1, ExecutorKey: ScheduledFunctionExecutorKey, ResourceScope: "function", ResourceKey: "/alice/demo/video/refresh.form"},
		{ID: 2, ExecutorKey: ScheduledAgentSessionExecutorKey, ResourceScope: "workspace_directory", ResourceKey: "/alice/demo/video"},
		{ID: 3, ExecutorKey: ScheduledFunctionExecutorKey, ResourceScope: "function", ResourceKey: "/alice/demo/missing.form"},
		{ID: 4, ExecutorKey: ScheduledAgentSessionExecutorKey, ResourceScope: "workspace_directory", ResourceKey: "/alice/demo/missing"},
		{ID: 5, ExecutorKey: ScheduledFunctionExecutorKey, ResourceScope: "function", ResourceKey: "/alice/demo/wrong.form"},
		{ID: 6, ExecutorKey: "platform.log_archive", ResourceScope: "system", ResourceKey: "log_archive"},
	}}
	reconciler := newScheduledTaskReconciler(repo, client)

	result, err := reconciler.ReconcileOrphans(context.Background())
	if err != nil {
		t.Fatalf("ReconcileOrphans: %v", err)
	}
	if result.Checked != 5 || result.Deleted != 3 || result.Skipped != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if fmt.Sprint(client.deleted) != "[3 4 5]" {
		t.Fatalf("deleted=%v, want [3 4 5]", client.deleted)
	}
}

func TestDeleteServiceTreeCascadesScheduledTasksBeforeTreeRemoval(t *testing.T) {
	repo, db := newScheduledTaskReconcileTestRepo(t)
	root := &model.ServiceTree{Type: model.ServiceTreeTypePackage, FullCodePath: "/alice/demo"}
	if err := db.Create(root).Error; err != nil {
		t.Fatalf("create root: %v", err)
	}
	child := &model.ServiceTree{Type: model.ServiceTreeTypeFunction, FullCodePath: "/alice/demo/refresh.form"}
	if err := db.Create(child).Error; err != nil {
		t.Fatalf("create child: %v", err)
	}
	client := &fakeScheduledTaskReconcileClient{tasks: []*scheduledsdk.Task{
		{ID: 9, ExecutorKey: ScheduledFunctionExecutorKey, ResourceScope: "function", ResourceKey: child.FullCodePath},
	}}
	mutation := newServiceTreeMutationService(repo, nil, nil, nil, newScheduledTaskReconciler(repo, client))

	if err := mutation.DeleteServiceTree(context.Background(), root.ID); err != nil {
		t.Fatalf("DeleteServiceTree: %v", err)
	}
	if fmt.Sprint(client.deleted) != "[9]" {
		t.Fatalf("deleted tasks=%v, want [9]", client.deleted)
	}
	var count int64
	if err := db.Model(&model.ServiceTree{}).Count(&count).Error; err != nil {
		t.Fatalf("count trees: %v", err)
	}
	if count != 0 {
		t.Fatalf("remaining service trees=%d, want 0", count)
	}
}

func TestDeleteServiceTreeStopsWhenScheduledTaskCleanupFails(t *testing.T) {
	repo, db := newScheduledTaskReconcileTestRepo(t)
	node := &model.ServiceTree{Type: model.ServiceTreeTypeFunction, FullCodePath: "/alice/demo/refresh.form"}
	if err := db.Create(node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}
	client := &fakeScheduledTaskReconcileClient{
		tasks:       []*scheduledsdk.Task{{ID: 9, ResourceKey: node.FullCodePath}},
		deleteErrID: 9,
	}
	mutation := newServiceTreeMutationService(repo, nil, nil, nil, newScheduledTaskReconciler(repo, client))

	err := mutation.DeleteServiceTree(context.Background(), node.ID)
	if err == nil || !strings.Contains(err.Error(), "删除目录关联定时任务失败") {
		t.Fatalf("expected scheduled cleanup error, got %v", err)
	}
	if _, err := repo.GetServiceTreeByID(node.ID); err != nil {
		t.Fatalf("service tree should remain after cleanup failure: %v", err)
	}
}
