package service

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/kageos/kageos/pkg/serviceconfig"
)

const scheduledTaskReconcilePageSize = 100

type scheduledTaskReconcileClient interface {
	ListTasks(context.Context, scheduledsdk.ListTasksRequest) (*scheduledsdk.ListTasksResponse, error)
	DeleteTask(context.Context, int64) error
}

type ScheduledTaskReconcileResult struct {
	Checked int
	Deleted int
	Skipped int
}

// ScheduledTaskReconciler keeps directory-bound timer tasks aligned with the
// Service Tree, which is the source of truth for resources that can be run.
type ScheduledTaskReconciler struct {
	serviceTreeRepo *repository.ServiceTreeRepository
	client          scheduledTaskReconcileClient
	mu              sync.Mutex
}

func NewScheduledTaskReconciler(serviceTreeRepo *repository.ServiceTreeRepository) *ScheduledTaskReconciler {
	return newScheduledTaskReconciler(serviceTreeRepo, scheduledsdk.NewClient(scheduledsdk.Options{
		BaseURL: serviceconfig.BuildGatewayURL("/timer/api/v1"),
	}))
}

func newScheduledTaskReconciler(serviceTreeRepo *repository.ServiceTreeRepository, client scheduledTaskReconcileClient) *ScheduledTaskReconciler {
	return &ScheduledTaskReconciler{serviceTreeRepo: serviceTreeRepo, client: client}
}

// DeleteTasksByResourcePrefix removes tasks bound to a node or any descendant.
// It runs before Service Tree deletion so a timer outage cannot silently leave
// an active task pointing at a resource the user believes was removed.
func (r *ScheduledTaskReconciler) DeleteTasksByResourcePrefix(ctx context.Context, resourcePath string) (int, error) {
	if r == nil || r.client == nil {
		return 0, fmt.Errorf("定时任务对账服务未初始化")
	}
	resourcePath = normalizeScheduledResourcePath(resourcePath)
	if resourcePath == "" {
		return 0, fmt.Errorf("资源路径不能为空")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	tasks, err := r.listTasks(ctx, scheduledsdk.ListTasksRequest{ResourceKeyPrefix: resourcePath})
	if err != nil {
		return 0, err
	}
	managedCtx := contextx.WithClientSource(ctx, scheduledTaskSourceManifest)
	deleted := 0
	for _, task := range tasks {
		if task == nil || task.ID <= 0 {
			continue
		}
		// The scheduler's SQL prefix filter uses LIKE, where underscores in valid
		// package names are wildcards. Re-check the path boundary before deletion.
		if !scheduledResourceWithinPrefix(task.ResourceKey, resourcePath) {
			continue
		}
		if err := r.client.DeleteTask(managedCtx, task.ID); err != nil {
			return deleted, fmt.Errorf("删除定时任务 %d 失败: %w", task.ID, err)
		}
		deleted++
	}
	return deleted, nil
}

// ReconcileOrphans deletes only task kinds whose resource contract is owned by
// app-server. Unknown scopes/executors are deliberately ignored.
func (r *ScheduledTaskReconciler) ReconcileOrphans(ctx context.Context) (ScheduledTaskReconcileResult, error) {
	var result ScheduledTaskReconcileResult
	if r == nil || r.client == nil || r.serviceTreeRepo == nil {
		return result, fmt.Errorf("定时任务对账服务未初始化")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	tasks, err := r.listTasks(ctx, scheduledsdk.ListTasksRequest{})
	if err != nil {
		return result, err
	}

	candidates := make([]*scheduledsdk.Task, 0, len(tasks))
	paths := make([]string, 0, len(tasks))
	for _, task := range tasks {
		if _, ok := scheduledTaskExpectedTreeType(task); !ok {
			result.Skipped++
			continue
		}
		path := normalizeScheduledResourcePath(task.ResourceKey)
		if path == "" {
			result.Skipped++
			continue
		}
		result.Checked++
		candidates = append(candidates, task)
		paths = append(paths, path)
	}

	nodes, err := r.serviceTreeRepo.GetServiceTreeByFullPaths(paths)
	if err != nil {
		return result, fmt.Errorf("查询定时任务绑定资源失败: %w", err)
	}

	managedCtx := contextx.WithClientSource(ctx, scheduledTaskSourceManifest)
	for _, task := range candidates {
		path := normalizeScheduledResourcePath(task.ResourceKey)
		expectedType, _ := scheduledTaskExpectedTreeType(task)
		node := nodes[path]
		if node != nil && node.Type == expectedType {
			continue
		}
		if err := r.client.DeleteTask(managedCtx, task.ID); err != nil {
			return result, fmt.Errorf("删除孤儿定时任务 %d（%s）失败: %w", task.ID, path, err)
		}
		result.Deleted++
	}
	return result, nil
}

func (r *ScheduledTaskReconciler) listTasks(ctx context.Context, filter scheduledsdk.ListTasksRequest) ([]*scheduledsdk.Task, error) {
	all := make([]*scheduledsdk.Task, 0)
	for page := 1; ; page++ {
		filter.Page = page
		filter.PageSize = scheduledTaskReconcilePageSize
		resp, err := r.client.ListTasks(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("查询定时任务失败: %w", err)
		}
		if resp == nil || len(resp.List) == 0 {
			break
		}
		all = append(all, resp.List...)
		if int64(len(all)) >= resp.Total || len(resp.List) < scheduledTaskReconcilePageSize {
			break
		}
	}
	return all, nil
}

func scheduledTaskExpectedTreeType(task *scheduledsdk.Task) (string, bool) {
	if task == nil {
		return "", false
	}
	switch {
	case task.ExecutorKey == ScheduledFunctionExecutorKey && task.ResourceScope == "function":
		return model.ServiceTreeTypeFunction, true
	case task.ExecutorKey == ScheduledAgentSessionExecutorKey && task.ResourceScope == "workspace_directory":
		return model.ServiceTreeTypePackage, true
	default:
		return "", false
	}
}

func normalizeScheduledResourcePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimRight(path, "/")
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func scheduledResourceWithinPrefix(path, prefix string) bool {
	path = normalizeScheduledResourcePath(path)
	prefix = normalizeScheduledResourcePath(prefix)
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}
