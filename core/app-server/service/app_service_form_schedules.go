package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/kageos/kageos/pkg/serviceconfig"
)

type appScheduleClient interface {
	CreateTask(context.Context, scheduledsdk.CreateTaskRequest) (*scheduledsdk.Task, error)
	UpdateTask(context.Context, int64, scheduledsdk.UpdateTaskRequest) (*scheduledsdk.Task, error)
	ListTasks(context.Context, scheduledsdk.ListTasksRequest) (*scheduledsdk.ListTasksResponse, error)
}

var newAppScheduleClient = func() appScheduleClient {
	return scheduledsdk.NewClient(scheduledsdk.Options{
		BaseURL: serviceconfig.BuildInternalTimerSchedulerURL("/timer/api/v1"),
	})
}

func (a *AppService) reconcileFormSchedules(ctx context.Context, state *appMetadataSyncState, apis []*dto.ApiInfo) error {
	if len(apis) == 0 {
		return nil
	}
	client := newAppScheduleClient()
	for _, api := range apis {
		if api == nil || strings.TrimSpace(api.TemplateType) != "form" || len(api.Schedules) == 0 {
			continue
		}
		for _, schedule := range api.Schedules {
			req, err := buildFormScheduleTaskRequest(ctx, state, api, schedule)
			if err != nil {
				return err
			}
			task, err := client.CreateTask(ctx, req)
			if err != nil {
				return fmt.Errorf("创建默认定时任务 %s/%s 失败: %w", api.FullCodePath, schedule.Code, err)
			}
			if task == nil || task.ID <= 0 {
				return fmt.Errorf("创建默认定时任务 %s/%s 未返回有效 task_id", api.FullCodePath, schedule.Code)
			}
			if _, err := client.UpdateTask(ctx, task.ID, updateTaskRequestFromCreate(req)); err != nil {
				return fmt.Errorf("更新默认定时任务 %s/%s 失败: %w", api.FullCodePath, schedule.Code, err)
			}
		}
	}
	return nil
}

func buildFormScheduleTaskRequest(ctx context.Context, state *appMetadataSyncState, api *dto.ApiInfo, formSchedule dto.FormScheduleConfig) (scheduledsdk.CreateTaskRequest, error) {
	fullCodePath := strings.TrimSpace(api.FullCodePath)
	if fullCodePath == "" {
		fullCodePath = api.BuildFullCodePath()
	}
	if fullCodePath == "" {
		return scheduledsdk.CreateTaskRequest{}, fmt.Errorf("默认定时任务 %q 缺少函数路径", formSchedule.Code)
	}
	code := strings.TrimSpace(formSchedule.Code)
	if code == "" {
		return scheduledsdk.CreateTaskRequest{}, fmt.Errorf("默认定时任务 %s 缺少 code", fullCodePath)
	}
	schedule, err := scheduledSDKScheduleFromFormSchedule(ctx, fullCodePath, code, formSchedule)
	if err != nil {
		return scheduledsdk.CreateTaskRequest{}, fmt.Errorf("默认定时任务 %s/%s 计划错误: %w", fullCodePath, code, err)
	}
	body, err := normalizeFormScheduleBody(formSchedule.Body)
	if err != nil {
		return scheduledsdk.CreateTaskRequest{}, fmt.Errorf("默认定时任务 %s/%s body 错误: %w", fullCodePath, code, err)
	}
	executorPayload := map[string]interface{}{
		"full_code_path": fullCodePath,
		"template_type":  "form",
		"action":         "execute",
		"method":         "POST",
		"payload":        json.RawMessage(body),
	}
	title := strings.TrimSpace(formSchedule.Title)
	if title == "" {
		title = strings.TrimSpace(api.Name)
	}
	if title == "" {
		title = "Form 默认定时任务"
	}
	requestUser := strings.TrimSpace(contextx.GetRequestUser(ctx))
	if requestUser == "" && state != nil && state.requestUser != "" {
		requestUser = state.requestUser
	}
	if requestUser == "" && state != nil && state.app != nil {
		requestUser = state.app.User
	}
	if requestUser == "" {
		requestUser = "system"
	}
	status := scheduledsdk.TaskStatusPaused
	if formSchedule.Enabled {
		status = scheduledsdk.TaskStatusPending
	}
	return scheduledsdk.CreateTaskRequest{
		Title:           title,
		Description:     strings.TrimSpace(formSchedule.Description),
		Category:        "scheduled_function",
		Tags:            []string{"function", "execute", "app_manifest"},
		IdempotencyKey:  formScheduleIdempotencyKey(fullCodePath, code),
		ExecutorKey:     ScheduledFunctionExecutorKey,
		ExecutorPayload: mustRawJSON(executorPayload),
		Metadata: map[string]string{
			"kind":          "scheduled_function",
			"action":        "execute",
			"method":        "POST",
			"template_type": "form",
			"managed_by":    "app_manifest",
			"schedule_code": code,
		},
		Schedule:        schedule,
		SourceType:      "function",
		SourceRef:       fullCodePath,
		ResourceScope:   "function",
		ResourceKey:     fullCodePath,
		RequestUser:     requestUser,
		RequestUserDept: contextx.GetRequestDepartmentFullPath(ctx),
		CreatedBy:       requestUser,
		Status:          status,
	}, nil
}

func scheduledSDKScheduleFromFormSchedule(ctx context.Context, fullCodePath string, code string, formSchedule dto.FormScheduleConfig) (scheduledsdk.Schedule, error) {
	cronExpr := strings.TrimSpace(formSchedule.CronExpr)
	hasCron := cronExpr != ""
	hasEvery := formSchedule.EverySeconds > 0
	if hasCron == hasEvery {
		return scheduledsdk.Schedule{}, fmt.Errorf("必须且只能设置 cron_expr 或 every_seconds")
	}
	schedule := scheduledsdk.Schedule{
		MaxRuns: formSchedule.MaxRuns,
	}
	if hasCron {
		schedule.Type = scheduledsdk.ScheduleCron
		schedule.CronExpr = cronExpr
		schedule.Timezone = formScheduleTimezone(ctx, fullCodePath, code, formSchedule.Timezone)
		return schedule, schedule.Validate()
	}
	schedule.Type = scheduledsdk.ScheduleEvery
	schedule.IntervalSeconds = formSchedule.EverySeconds
	return schedule, schedule.Validate()
}

func formScheduleTimezone(ctx context.Context, fullCodePath string, code string, timezone string) string {
	timezone = strings.TrimSpace(timezone)
	if timezone == "" {
		return ""
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		logger.Warnf(ctx, "[FormSchedule] invalid timezone, fallback to runtime local timezone: full_code_path=%s code=%s timezone=%q error=%v", fullCodePath, code, timezone, err)
		return ""
	}
	return timezone
}

func normalizeFormScheduleBody(body json.RawMessage) (json.RawMessage, error) {
	body = json.RawMessage(strings.TrimSpace(string(body)))
	if len(body) == 0 || string(body) == "null" {
		return json.RawMessage(`{}`), nil
	}
	if !json.Valid(body) {
		return nil, fmt.Errorf("不是合法 JSON")
	}
	if body[0] != '{' {
		return nil, fmt.Errorf("必须是 JSON object")
	}
	return body, nil
}

func formScheduleIdempotencyKey(fullCodePath string, code string) string {
	parts := strings.Join([]string{strings.TrimSpace(fullCodePath), strings.TrimSpace(code)}, "\x00")
	sum := sha1.Sum([]byte(parts))
	return "app-form-schedule-" + hex.EncodeToString(sum[:])
}

func mustRawJSON(value interface{}) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return data
}

func updateTaskRequestFromCreate(req scheduledsdk.CreateTaskRequest) scheduledsdk.UpdateTaskRequest {
	return scheduledsdk.UpdateTaskRequest{
		Title:           stringPtr(req.Title),
		Description:     stringPtr(req.Description),
		Category:        stringPtr(req.Category),
		Tags:            &req.Tags,
		ExecutorPayload: req.ExecutorPayload,
		Metadata:        &req.Metadata,
		Schedule:        &req.Schedule,
		SourceType:      stringPtr(req.SourceType),
		SourceRef:       stringPtr(req.SourceRef),
		ResourceScope:   stringPtr(req.ResourceScope),
		ResourceKey:     stringPtr(req.ResourceKey),
		RequestUser:     stringPtr(req.RequestUser),
		RequestUserDept: stringPtr(req.RequestUserDept),
	}
}

func stringPtr(v string) *string {
	return &v
}
