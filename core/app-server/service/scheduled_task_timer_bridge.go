package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/pkg/scheduledsdk"
)

const (
	scheduledTaskExecutorKey      = "app.function"
	scheduledTaskTimerPayloadType = "scheduled_task"
)

type scheduledTaskTimerPayload struct {
	Type        string `json:"type,omitempty"`
	TaskID      int64  `json:"task_id"`
	BusinessRef string `json:"business_ref,omitempty"`
}

func (s *ScheduledTaskService) UsesTimerScheduler() bool {
	return s != nil && s.options.TimerClient != nil
}

func (s *ScheduledTaskService) createTimerTask(ctx context.Context, task *model.ScheduledTask) error {
	if s == nil || s.options.TimerClient == nil {
		return fmt.Errorf("timer-scheduler client 未配置")
	}
	payload, err := json.Marshal(scheduledTaskTimerPayload{
		Type:        scheduledTaskTimerPayloadType,
		TaskID:      task.ID,
		BusinessRef: scheduledTaskRef(task.ID),
	})
	if err != nil {
		return err
	}
	timerTask, err := s.options.TimerClient.CreateTask(ctx, scheduledsdk.CreateTaskRequest{
		Title:             task.Name,
		Description:       task.FullCodePath,
		Category:          "app",
		Tags:              []string{"app", "function", task.Action},
		ExecutorKey:       scheduledTaskExecutorKey,
		ExecutorPayload:   payload,
		Metadata:          scheduledTaskTimerMetadata(task),
		Schedule:          scheduledTaskTimerSchedule(task),
		RequestUser:       task.RequestUser,
		RequestUserDept:   task.RequestUserDept,
		NotifyUsers:       SplitScheduledTaskRecipientsForAPI(task.NotifyUsers),
		NotifyDepartments: SplitScheduledTaskRecipientsForAPI(task.NotifyDepartments),
		NotifyOn:          scheduledsdk.NotifyOn(task.NotifyOn),
		SourceType:        "app.function",
		SourceRef:         task.FullCodePath,
	})
	if err != nil {
		return err
	}
	task.TimerTaskID = timerTask.ID
	if err := s.taskRepo.Update(task); err != nil {
		_ = s.options.TimerClient.CancelTask(ctx, timerTask.ID)
		return err
	}
	return nil
}

func (s *ScheduledTaskService) cancelTimerTask(ctx context.Context, taskID int64) error {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return err
	}
	if task.TimerTaskID <= 0 {
		return nil
	}
	if s == nil || s.options.TimerClient == nil {
		return fmt.Errorf("timer-scheduler client 未配置")
	}
	return s.options.TimerClient.CancelTask(ctx, task.TimerTaskID)
}

func scheduledTaskTimerSchedule(task *model.ScheduledTask) scheduledsdk.Schedule {
	schedule := scheduledsdk.Schedule{
		Type:            scheduledsdk.ScheduleType(task.ScheduleType),
		CronExpr:        task.CronExpr,
		IntervalSeconds: task.IntervalSeconds,
		MaxRuns:         task.MaxRuns,
		Timezone:        task.Timezone,
	}
	if !task.RunAt.IsZero() {
		schedule.RunAt = task.RunAt
	}
	return schedule
}

func scheduledTaskTimerMetadata(task *model.ScheduledTask) map[string]string {
	metadata := map[string]string{
		"app_code":       task.App,
		"business_type":  scheduledTaskTimerPayloadType,
		"full_code_path": task.FullCodePath,
		"action":         task.Action,
	}
	if task.User != "" {
		metadata["tenant_user"] = task.User
	}
	if task.Method != "" {
		metadata["method"] = task.Method
	}
	return metadata
}

func scheduledTaskRef(taskID int64) string {
	return fmt.Sprintf("%s:%d", scheduledTaskTimerPayloadType, taskID)
}

func parseScheduledTaskTimerPayload(event scheduledsdk.ExecutionRequestedEvent) (int64, error) {
	var payload scheduledTaskTimerPayload
	if len(event.ExecutorPayload) > 0 {
		if err := json.Unmarshal(event.ExecutorPayload, &payload); err != nil {
			return 0, err
		}
	}
	if payload.TaskID > 0 {
		return payload.TaskID, nil
	}
	ref := strings.TrimSpace(payload.BusinessRef)
	if ref == "" {
		ref = strings.TrimSpace(event.SourceRef)
	}
	const prefix = scheduledTaskTimerPayloadType + ":"
	if strings.HasPrefix(ref, prefix) {
		id, err := strconv.ParseInt(strings.TrimPrefix(ref, prefix), 10, 64)
		if err != nil || id <= 0 {
			return 0, fmt.Errorf("invalid scheduled task ref %q", ref)
		}
		return id, nil
	}
	return 0, fmt.Errorf("scheduled task id missing in executor_payload")
}

func (s *ScheduledTaskService) HandleTimerExecution(ctx context.Context, event scheduledsdk.ExecutionRequestedEvent) (*scheduledsdk.ExecutionResult, error) {
	taskID, err := parseScheduledTaskTimerPayload(event)
	if err != nil {
		return nil, err
	}
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, err
	}
	if !s.tryAcquireWorkerSlot() {
		return nil, fmt.Errorf("后台执行并发已满，请稍后重试")
	}
	defer s.releaseWorkerSlot()

	scheduledAt := event.ScheduledAt
	if scheduledAt.IsZero() {
		scheduledAt = time.Now()
	}
	task.NextRunAt = &scheduledAt
	exec := s.executeOne(ctx, task)
	if exec == nil {
		return nil, fmt.Errorf("定时任务执行记录创建失败")
	}
	status := scheduledsdk.ExecutionStatus(exec.Status)
	if status == "" {
		status = scheduledsdk.ExecutionStatusFailed
	}
	return &scheduledsdk.ExecutionResult{
		Status:         status,
		ExecutorRunID:  exec.TraceID,
		OutputSummary:  exec.Status,
		ResultPayload:  append(json.RawMessage(nil), exec.ResponsePayload...),
		ErrorMessage:   exec.ErrorMessage,
		DurationMillis: exec.DurationMillis,
	}, nil
}
