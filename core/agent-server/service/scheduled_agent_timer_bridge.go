package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/pkg/scheduledsdk"
)

const (
	scheduledAgentExecutorKey      = "agent.session"
	scheduledAgentTimerPayloadType = "scheduled_agent_task"
)

type scheduledAgentTimerPayload struct {
	Type        string `json:"type,omitempty"`
	TaskID      int64  `json:"task_id"`
	BusinessRef string `json:"business_ref,omitempty"`
}

func (s *ScheduledAgentTaskService) UsesTimerScheduler() bool {
	return s != nil && s.options.TimerClient != nil
}

func (s *ScheduledAgentTaskService) createTimerTask(ctx context.Context, task *model.ScheduledAgentTask) error {
	if s == nil || s.options.TimerClient == nil {
		return fmt.Errorf("timer-scheduler client 未配置")
	}
	payload, err := json.Marshal(scheduledAgentTimerPayload{
		Type:        scheduledAgentTimerPayloadType,
		TaskID:      task.ID,
		BusinessRef: scheduledAgentTaskRef(task.ID),
	})
	if err != nil {
		return err
	}
	timerTask, err := s.options.TimerClient.CreateTask(ctx, scheduledsdk.CreateTaskRequest{
		Title:             task.Name,
		Description:       task.Goal,
		Category:          "agent",
		Tags:              []string{"agent", "workspace"},
		ExecutorKey:       scheduledAgentExecutorKey,
		ExecutorPayload:   payload,
		Metadata:          scheduledAgentTimerMetadata(task),
		Schedule:          scheduledAgentTimerSchedule(task),
		RequestUser:       task.RequestUser,
		RequestUserDept:   task.RequestUserDept,
		NotifyUsers:       SplitScheduledAgentRecipientsForAPI(task.NotifyUsers),
		NotifyDepartments: SplitScheduledAgentRecipientsForAPI(task.NotifyDepartments),
		NotifyOn:          scheduledsdk.NotifyOn(task.NotifyOn),
		SourceType:        task.SourceType,
		SourceRef:         task.SourceRef,
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

func (s *ScheduledAgentTaskService) updateTimerTask(ctx context.Context, task *model.ScheduledAgentTask) error {
	if task.TimerTaskID <= 0 {
		return fmt.Errorf("timer_task_id 为空，无法同步 timer-scheduler")
	}
	if s == nil || s.options.TimerClient == nil {
		return fmt.Errorf("timer-scheduler client 未配置")
	}
	title := task.Name
	description := task.Goal
	category := "agent"
	tags := []string{"agent", "workspace"}
	executorKey := scheduledAgentExecutorKey
	metadata := scheduledAgentTimerMetadata(task)
	schedule := scheduledAgentTimerSchedule(task)
	requestUser := task.RequestUser
	requestUserDept := task.RequestUserDept
	notifyUsers := SplitScheduledAgentRecipientsForAPI(task.NotifyUsers)
	notifyDepartments := SplitScheduledAgentRecipientsForAPI(task.NotifyDepartments)
	notifyOn := scheduledsdk.NotifyOn(task.NotifyOn)
	sourceType := task.SourceType
	sourceRef := task.SourceRef
	_, err := s.options.TimerClient.UpdateTask(ctx, task.TimerTaskID, scheduledsdk.UpdateTaskRequest{
		Title:             &title,
		Description:       &description,
		Category:          &category,
		Tags:              &tags,
		ExecutorKey:       &executorKey,
		Metadata:          &metadata,
		Schedule:          &schedule,
		RequestUser:       &requestUser,
		RequestUserDept:   &requestUserDept,
		NotifyUsers:       &notifyUsers,
		NotifyDepartments: &notifyDepartments,
		NotifyOn:          &notifyOn,
		SourceType:        &sourceType,
		SourceRef:         &sourceRef,
	})
	return err
}

func (s *ScheduledAgentTaskService) pauseTimerTask(ctx context.Context, taskID int64) error {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return err
	}
	if task.TimerTaskID <= 0 {
		return fmt.Errorf("timer_task_id 为空，无法同步 timer-scheduler")
	}
	if s == nil || s.options.TimerClient == nil {
		return fmt.Errorf("timer-scheduler client 未配置")
	}
	return s.options.TimerClient.PauseTask(ctx, task.TimerTaskID)
}

func (s *ScheduledAgentTaskService) resumeTimerTask(ctx context.Context, taskID int64) error {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return err
	}
	if task.TimerTaskID <= 0 {
		return fmt.Errorf("timer_task_id 为空，无法同步 timer-scheduler")
	}
	if s == nil || s.options.TimerClient == nil {
		return fmt.Errorf("timer-scheduler client 未配置")
	}
	return s.options.TimerClient.ResumeTask(ctx, task.TimerTaskID)
}

func (s *ScheduledAgentTaskService) cancelTimerTask(ctx context.Context, taskID int64) error {
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

func scheduledAgentTimerSchedule(task *model.ScheduledAgentTask) scheduledsdk.Schedule {
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

func scheduledAgentTimerMetadata(task *model.ScheduledAgentTask) map[string]string {
	metadata := map[string]string{
		"app_code":       "agent",
		"business_type":  scheduledAgentTimerPayloadType,
		"full_code_path": task.FullCodePath,
	}
	if task.ModeCode != "" {
		metadata["mode_code"] = task.ModeCode
	}
	if task.SourceType != "" {
		metadata["source_type"] = task.SourceType
	}
	return metadata
}

func parseScheduledAgentTimerPayload(event scheduledsdk.ExecutionRequestedEvent) (int64, error) {
	var payload scheduledAgentTimerPayload
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
	const prefix = "scheduled_agent_task:"
	if strings.HasPrefix(ref, prefix) {
		id, err := strconv.ParseInt(strings.TrimPrefix(ref, prefix), 10, 64)
		if err != nil || id <= 0 {
			return 0, fmt.Errorf("invalid scheduled agent task ref %q", ref)
		}
		return id, nil
	}
	return 0, fmt.Errorf("scheduled agent task id missing in executor_payload")
}

func (s *ScheduledAgentTaskService) HandleTimerExecution(ctx context.Context, event scheduledsdk.ExecutionRequestedEvent) (*scheduledsdk.ExecutionResult, error) {
	taskID, err := parseScheduledAgentTimerPayload(event)
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
	exec, err := s.createExecution(ctx, task, scheduledAt, task.RequestUser, model.ScheduledAgentExecutionStatusRunning)
	if err != nil {
		return nil, err
	}
	s.executeOne(ctx, task, exec, true)
	status := scheduledsdk.ExecutionStatus(exec.Status)
	if status == "" {
		status = scheduledsdk.ExecutionStatusFailed
	}
	return &scheduledsdk.ExecutionResult{
		Status:         status,
		ExecutorRunID:  exec.SessionID,
		OutputSummary:  exec.OutputSummary,
		ErrorMessage:   exec.ErrorMessage,
		DurationMillis: exec.DurationMillis,
	}, nil
}
