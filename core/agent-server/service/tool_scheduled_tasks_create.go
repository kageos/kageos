package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/scheduledsdk"
)

func runCreateScheduledFunctionTask(ctx context.Context, args createScheduledFunctionTaskArgs, currentFullCodePath string) ToolResult {
	ctx = withAgentToolClientSource(ctx)
	args = normalizeCreateScheduledFunctionTaskArgs(args)
	fullCodePath := resolveFullCodePathArg(args.FullCodePath, currentFullCodePath)
	if fullCodePath == "" {
		return toolResult("create_scheduled_function_task 需传 full_code_path。", true)
	}
	action := strings.TrimSpace(args.Action)
	if action == "" {
		action = "execute"
	}
	if !isScheduledFunctionAction(action) {
		return toolResult("create_scheduled_function_task action 仅支持 execute/table_create/table_update/table_delete。", true)
	}
	if requiresScheduledFunctionBody(fullCodePath, action) && strings.TrimSpace(args.Body) == "" {
		return toolResult("create_scheduled_function_task 创建写入类函数定时任务时必须传 body，且 body 要直接包含已确认的业务字段；不要传空参数或 invoke_params 包装。", true)
	}
	payload, err := parseScheduledPayload(args.Body)
	if err != nil {
		return toolResult("create_scheduled_function_task body 需为合法 JSON: "+err.Error(), true)
	}
	schedule, err := buildScheduledTaskSchedule(args.scheduleArgs())
	if err != nil {
		return toolResult("create_scheduled_function_task 计划参数错误: "+err.Error(), true)
	}
	if err := requireScheduledTaskPermission(ctx, fullCodePath, scheduledFunctionRequiredAction(fullCodePath, action)); err != nil {
		return toolResult("create_scheduled_function_task 权限校验失败: "+err.Error(), true)
	}
	user, err := scheduledTaskCurrentUser(ctx)
	if err != nil {
		return toolResult(err.Error(), true)
	}
	executorPayload := map[string]interface{}{
		"full_code_path": fullCodePath,
		"template_type":  scheduledFunctionTemplateType(fullCodePath),
		"action":         action,
		"method":         methodForScheduledFunctionAction(action),
		"payload":        payload,
	}
	req := scheduledsdk.CreateTaskRequest{
		Title:           firstNonEmptyString(args.Title, defaultScheduledFunctionTitle(fullCodePath)),
		Description:     strings.TrimSpace(args.Description),
		Category:        "scheduled_function",
		Tags:            []string{"function", action},
		IdempotencyKey:  scheduledIdempotencyKey(args.IdempotencyKey, "function", fullCodePath, action, schedule, payload),
		ExecutorKey:     "app.function",
		ExecutorPayload: mustRawJSON(executorPayload),
		Metadata: map[string]string{
			"kind":          "scheduled_function",
			"action":        action,
			"method":        methodForScheduledFunctionAction(action),
			"template_type": scheduledFunctionTemplateType(fullCodePath),
		},
		Schedule:        schedule,
		SourceType:      "function",
		SourceRef:       fullCodePath,
		ResourceScope:   "function",
		ResourceKey:     fullCodePath,
		RequestUser:     user,
		RequestUserDept: contextx.GetRequestDepartmentFullPath(ctx),
		CreatedBy:       user,
	}
	task, err := scheduledTaskClient().CreateTask(ctx, req)
	if err != nil {
		return toolResult("create_scheduled_function_task 调用失败: "+err.Error(), true)
	}
	return toolResultWithStructuredData(map[string]interface{}{
		"task":        task,
		"next_run_at": task.NextRunAt,
		"message":     "函数任务已创建",
	}, false)
}

func runCreateScheduledAgentTask(ctx context.Context, args createScheduledAgentTaskArgs, currentFullCodePath string) ToolResult {
	ctx = withAgentToolClientSource(ctx)
	args = normalizeCreateScheduledAgentTaskArgs(args)
	fullCodePath := resolveFullCodePathArg(args.FullCodePath, currentFullCodePath)
	if fullCodePath == "" {
		return toolResult("create_scheduled_agent_task 需传 full_code_path。", true)
	}
	message := strings.TrimSpace(args.Message)
	if message == "" {
		return toolResult("create_scheduled_agent_task 需传 message。message 是到点后交给 Agent 的执行说明。", true)
	}
	schedule, err := buildScheduledTaskSchedule(args.scheduleArgs())
	if err != nil {
		return toolResult("create_scheduled_agent_task 计划参数错误: "+err.Error(), true)
	}
	if err := requireScheduledTaskPermission(ctx, fullCodePath, access.ActionWrite); err != nil {
		return toolResult("create_scheduled_agent_task 权限校验失败: "+err.Error(), true)
	}
	user, err := scheduledTaskCurrentUser(ctx)
	if err != nil {
		return toolResult(err.Error(), true)
	}
	modeCode := strings.TrimSpace(args.ModeCode)
	if modeCode == "" {
		modeCode = "dev"
	}
	executorPayload := map[string]interface{}{
		"full_code_path":  fullCodePath,
		"message":         message,
		"display_content": message,
	}
	if modeCode != "" && modeCode != "dev" {
		executorPayload["mode_code"] = modeCode
	}
	if files := strings.TrimSpace(args.Files); files != "" {
		executorPayload["files"] = files
	}
	if args.LLMConfigID > 0 {
		executorPayload["llm_config_id"] = args.LLMConfigID
	}
	if args.MaxDurationSeconds > 0 {
		executorPayload["max_duration_seconds"] = args.MaxDurationSeconds
	}
	req := scheduledsdk.CreateTaskRequest{
		Title:           firstNonEmptyString(args.Title, defaultScheduledAgentTitle(fullCodePath, message)),
		Description:     strings.TrimSpace(args.Description),
		Category:        "scheduled_agent_session",
		Tags:            []string{"agent", "session"},
		IdempotencyKey:  scheduledIdempotencyKey(args.IdempotencyKey, "agent_session", fullCodePath, modeCode, schedule, message),
		ExecutorKey:     "agent.session",
		ExecutorPayload: mustRawJSON(executorPayload),
		Metadata: map[string]string{
			"kind":      "scheduled_agent_session",
			"mode_code": modeCode,
		},
		Status:          scheduledsdk.TaskStatusPaused,
		Schedule:        schedule,
		OverlapPolicy:   scheduledsdk.OverlapPolicy(strings.TrimSpace(args.OverlapPolicy)),
		MaxParallelism:  args.MaxParallelism,
		SourceType:      "agent_session",
		SourceRef:       fullCodePath,
		ResourceScope:   "workspace_directory",
		ResourceKey:     fullCodePath,
		RequestUser:     user,
		RequestUserDept: contextx.GetRequestDepartmentFullPath(ctx),
		CreatedBy:       user,
	}
	task, err := scheduledTaskClient().CreateTask(ctx, req)
	if err != nil {
		return toolResult("create_scheduled_agent_task 调用失败: "+err.Error(), true)
	}
	return toolResultWithStructuredData(map[string]interface{}{
		"task":        task,
		"next_run_at": task.NextRunAt,
		"message":     "Agent 任务已创建，默认暂停；需要启用后才会无人值守执行。",
	}, false)
}

func runUpdateScheduledAgentTask(ctx context.Context, args updateScheduledAgentTaskArgs) ToolResult {
	ctx = withAgentToolClientSource(ctx)
	if args.TaskID <= 0 {
		return toolResult("update_scheduled_agent_task 需传合法 task_id。", true)
	}
	client := scheduledTaskClient()
	task, err := client.GetTask(ctx, args.TaskID)
	if err != nil {
		return toolResult("update_scheduled_agent_task 查询任务失败: "+err.Error(), true)
	}
	if err := ensureScheduledTaskOwnedByCurrentUser(ctx, task); err != nil {
		return toolResult("update_scheduled_agent_task 权限校验失败: "+err.Error(), true)
	}
	if task.ExecutorKey != "agent.session" {
		return toolResult("该任务不是 Agent 任务，无法使用此工具更新", true)
	}
	req := scheduledsdk.UpdateTaskRequest{}
	if title := strings.TrimSpace(args.Title); title != "" {
		req.Title = &title
	}
	if desc := strings.TrimSpace(args.Description); desc != "" {
		req.Description = &desc
	}
	if policy := strings.TrimSpace(args.OverlapPolicy); policy != "" {
		overlapPolicy := scheduledsdk.OverlapPolicy(policy)
		req.OverlapPolicy = &overlapPolicy
	}
	if args.MaxParallelism > 0 {
		maxParallelism := args.MaxParallelism
		req.MaxParallelism = &maxParallelism
	}

	hasScheduleArgs := strings.TrimSpace(args.ScheduleType) != "" || strings.TrimSpace(args.RunAt) != "" || strings.TrimSpace(args.CronExpr) != "" || args.IntervalSeconds > 0
	if hasScheduleArgs {
		schedule, err := buildScheduledTaskSchedule(args.scheduleArgs())
		if err != nil {
			return toolResult("update_scheduled_agent_task 计划参数错误: "+err.Error(), true)
		}
		req.Schedule = &schedule
	}

	message := strings.TrimSpace(args.Message)
	modeCode := strings.TrimSpace(args.ModeCode)
	files := strings.TrimSpace(args.Files)

	hasPayloadChanges := message != "" || modeCode != "" || files != "" || args.LLMConfigID > 0 || args.MaxDurationSeconds > 0
	if hasPayloadChanges {
		var payload map[string]interface{}
		if len(task.ExecutorPayload) > 0 {
			if err := json.Unmarshal(task.ExecutorPayload, &payload); err != nil {
				payload = make(map[string]interface{})
			}
		} else {
			payload = make(map[string]interface{})
		}

		if message != "" {
			payload["message"] = message
			payload["display_content"] = message
		}
		if modeCode != "" {
			payload["mode_code"] = modeCode
		}
		if files != "" {
			payload["files"] = files
		}
		if args.LLMConfigID > 0 {
			payload["llm_config_id"] = args.LLMConfigID
		}
		if args.MaxDurationSeconds > 0 {
			payload["max_duration_seconds"] = args.MaxDurationSeconds
		}
		req.ExecutorPayload = mustRawJSON(payload)
	}

	updated, err := client.UpdateTask(ctx, args.TaskID, req)
	if err != nil {
		return toolResult("update_scheduled_agent_task 更新失败: "+err.Error(), true)
	}
	return toolResultWithStructuredData(map[string]interface{}{
		"task":        updated,
		"next_run_at": updated.NextRunAt,
		"message":     "Agent 任务已更新",
	}, false)
}
