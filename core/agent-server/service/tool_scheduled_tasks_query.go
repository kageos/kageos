package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/pkg/scheduledsdk"
)

func runListScheduledTasks(ctx context.Context, args listScheduledTasksArgs, currentFullCodePath string) ToolResult {
	ctx = withAgentToolClientSource(ctx)
	kind := normalizeScheduledTaskKind(args.Kind)
	resourcePath := resolveFullCodePathArg(args.ResourcePath, currentFullCodePath)
	req := listScheduledTasksRequest(kind, resourcePath, strings.TrimSpace(args.Status), args.Page, args.PageSize)
	resp, err := listScheduledTasksAllPages(ctx, scheduledTaskClient(), req)
	if err != nil {
		return toolResult("list_scheduled_tasks 调用失败: "+err.Error(), true)
	}
	return toolResultWithStructuredData(resp, false)
}

func listScheduledTasksRequest(kind string, resourcePath string, status string, page int, pageSize int) scheduledsdk.ListTasksRequest {
	req := scheduledsdk.ListTasksRequest{
		Status:   strings.TrimSpace(status),
		Page:     page,
		PageSize: pageSize,
	}
	if kind != "all" {
		req.ExecutorKey = scheduledExecutorKeyForKind(kind)
	}
	if resourcePath != "" {
		req.ResourceKeyPrefix = resourcePath
	}
	if kind != "all" {
		req.ResourceScope = scheduledResourceScopeForKind(kind)
	}
	return req
}

func listScheduledTasksAllPages(ctx context.Context, client *scheduledsdk.Client, req scheduledsdk.ListTasksRequest) (*scheduledsdk.ListTasksResponse, error) {
	if client == nil {
		client = scheduledTaskClient()
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 100
	}
	var out scheduledsdk.ListTasksResponse
	for page := 1; ; page++ {
		req.Page = page
		resp, err := client.ListTasks(ctx, req)
		if err != nil {
			return nil, err
		}
		if resp == nil || len(resp.List) == 0 {
			break
		}
		out.List = append(out.List, resp.List...)
		out.Total = resp.Total
		if resp.Total > 0 && int64(len(out.List)) >= resp.Total {
			break
		}
		if len(resp.List) < req.PageSize {
			break
		}
	}
	if out.Total == 0 {
		out.Total = int64(len(out.List))
	}
	return &out, nil
}

func runManageScheduledTask(ctx context.Context, args manageScheduledTaskArgs) ToolResult {
	ctx = withAgentToolClientSource(ctx)
	if args.TaskID <= 0 {
		return toolResult("manage_scheduled_task 需传合法 task_id。", true)
	}
	client := scheduledTaskClient()
	task, err := client.GetTask(ctx, args.TaskID)
	if err != nil {
		return toolResult("manage_scheduled_task 查询任务失败: "+err.Error(), true)
	}
	if err := ensureScheduledTaskOwnedByCurrentUser(ctx, task); err != nil {
		return toolResult("manage_scheduled_task 权限校验失败: "+err.Error(), true)
	}
	if resourcePath := resolveFullCodePathArg(args.ResourcePath, ""); resourcePath != "" && task.ResourceKey != resourcePath {
		return toolResult(fmt.Sprintf("manage_scheduled_task 任务资源不匹配：任务属于 %s，不是 %s。", task.ResourceKey, resourcePath), true)
	}
	switch strings.TrimSpace(args.Action) {
	case "pause":
		err = client.PauseTask(ctx, args.TaskID)
	case "resume":
		err = client.ResumeTask(ctx, args.TaskID)
	case "cancel":
		err = client.CancelTask(ctx, args.TaskID)
	case "delete":
		err = client.DeleteTask(ctx, args.TaskID)
	case "run_now":
		exec, runErr := client.RunNow(ctx, args.TaskID)
		if runErr != nil {
			return toolResult("manage_scheduled_task 立即运行失败: "+runErr.Error(), true)
		}
		return toolResultWithStructuredData(map[string]interface{}{
			"task_id":   args.TaskID,
			"execution": exec,
			"message":   "已提交立即运行",
		}, false)
	default:
		return toolResult("manage_scheduled_task action 仅支持 pause/resume/cancel/delete/run_now。", true)
	}
	if err != nil {
		return toolResult("manage_scheduled_task 调用失败: "+err.Error(), true)
	}
	if strings.TrimSpace(args.Action) == "delete" {
		return toolResultWithStructuredData(map[string]interface{}{
			"task_id": args.TaskID,
			"action":  args.Action,
			"message": "定时任务已删除",
		}, false)
	}
	updated, getErr := client.GetTask(ctx, args.TaskID)
	if getErr != nil {
		return toolResultWithStructuredData(map[string]interface{}{
			"task_id": args.TaskID,
			"action":  args.Action,
			"message": "操作已提交，但刷新任务详情失败: " + getErr.Error(),
		}, false)
	}
	return toolResultWithStructuredData(map[string]interface{}{
		"task":    updated,
		"message": "定时任务已" + scheduledManageActionLabel(args.Action),
	}, false)
}

func runListScheduledTaskExecutions(ctx context.Context, args listScheduledTaskExecutionsArgs) ToolResult {
	ctx = withAgentToolClientSource(ctx)
	if args.TaskID <= 0 {
		return toolResult("list_scheduled_task_executions 需传合法 task_id。", true)
	}
	client := scheduledTaskClient()
	if _, err := client.GetTask(ctx, args.TaskID); err != nil {
		return toolResult("list_scheduled_task_executions 查询任务失败: "+err.Error(), true)
	}
	resp, err := client.ListExecutions(ctx, args.TaskID, scheduledsdk.ListExecutionsRequest{
		Status:   strings.TrimSpace(args.Status),
		Page:     args.Page,
		PageSize: args.PageSize,
	})
	if err != nil {
		return toolResult("list_scheduled_task_executions 调用失败: "+err.Error(), true)
	}
	return toolResultWithStructuredData(resp, false)
}
