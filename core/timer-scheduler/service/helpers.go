package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/timer-scheduler/model"
	"github.com/ai-agent-os/ai-agent-os/pkg/scheduledsdk"
)

func validateCreateTaskRequest(req scheduledsdk.CreateTaskRequest) error {
	if strings.TrimSpace(req.Title) == "" {
		return fmt.Errorf("timer-scheduler: title is required")
	}
	if strings.TrimSpace(req.ExecutorKey) == "" {
		return fmt.Errorf("timer-scheduler: executor_key is required")
	}
	return req.Schedule.Validate()
}

func nextRunForSchedule(schedule scheduledsdk.Schedule, now time.Time) (*time.Time, error) {
	switch schedule.Type {
	case scheduledsdk.ScheduleAt:
		runAt := schedule.RunAt
		return &runAt, nil
	case scheduledsdk.ScheduleCron:
		parsed, err := cronParser.Parse(schedule.CronExpr)
		if err != nil {
			return nil, fmt.Errorf("timer-scheduler: cron_expr parse failed: %w", err)
		}
		next := parsed.Next(now)
		return &next, nil
	case scheduledsdk.ScheduleEvery:
		next := now.Add(time.Duration(schedule.IntervalSeconds) * time.Second)
		return &next, nil
	default:
		return nil, fmt.Errorf("timer-scheduler: unsupported schedule type %q", schedule.Type)
	}
}

func nextRunForTask(task *model.TimerTask, now time.Time) (*time.Time, error) {
	schedule := scheduledsdk.Schedule{
		Type:            scheduledsdk.ScheduleType(task.ScheduleType),
		CronExpr:        task.CronExpr,
		IntervalSeconds: task.IntervalSeconds,
		MaxRuns:         task.MaxRuns,
		Timezone:        task.Timezone,
	}
	if task.RunAt != nil {
		schedule.RunAt = *task.RunAt
	}
	return nextRunForSchedule(schedule, now)
}

func scheduledAtForTask(task *model.TimerTask, fallback time.Time) time.Time {
	if task.NextRunAt != nil && !task.NextRunAt.IsZero() {
		return *task.NextRunAt
	}
	return fallback
}

func computeTaskNextState(task *model.TimerTask, scheduledAt time.Time, success bool) (string, *time.Time, string) {
	switch task.ScheduleType {
	case string(scheduledsdk.ScheduleAt):
		if success {
			return string(scheduledsdk.TaskStatusDone), nil, task.LastErrorMessage
		}
		return string(scheduledsdk.TaskStatusFailed), nil, task.LastErrorMessage
	case string(scheduledsdk.ScheduleCron):
		next, err := nextRunForTask(task, scheduledAt)
		if err != nil {
			return string(scheduledsdk.TaskStatusFailed), nil, err.Error()
		}
		return string(scheduledsdk.TaskStatusPending), next, task.LastErrorMessage
	case string(scheduledsdk.ScheduleEvery):
		if task.MaxRuns > 0 && task.RunCount >= task.MaxRuns {
			if success {
				return string(scheduledsdk.TaskStatusDone), nil, task.LastErrorMessage
			}
			return string(scheduledsdk.TaskStatusFailed), nil, task.LastErrorMessage
		}
		next := scheduledAt.Add(time.Duration(task.IntervalSeconds) * time.Second)
		return string(scheduledsdk.TaskStatusPending), &next, task.LastErrorMessage
	default:
		if success {
			return string(scheduledsdk.TaskStatusDone), nil, task.LastErrorMessage
		}
		return string(scheduledsdk.TaskStatusFailed), nil, task.LastErrorMessage
	}
}

func taskToSDK(task *model.TimerTask) *scheduledsdk.Task {
	if task == nil {
		return nil
	}
	return &scheduledsdk.Task{
		ID:                  task.ID,
		Title:               task.Title,
		Description:         task.Description,
		Category:            task.Category,
		Tags:                decodeStringSlice(task.TagsJSON),
		ExecutorKey:         task.ExecutorKey,
		Status:              scheduledsdk.TaskStatus(task.Status),
		Schedule:            taskSchedule(task),
		NextRunAt:           task.NextRunAt,
		RunCount:            task.RunCount,
		InflightExecutionID: task.InflightExecutionID,
		LastExecutionID:     task.LastExecutionID,
		LastErrorMessage:    task.LastErrorMessage,
		SourceType:          task.SourceType,
		SourceRef:           task.SourceRef,
		RequestUser:         task.RequestUser,
		RequestUserDept:     task.RequestUserDept,
		NotifyUsers:         splitStringList(task.NotifyUsers),
		NotifyDepartments:   splitStringList(task.NotifyDepartments),
		NotifyOn:            scheduledsdk.NotifyOn(task.NotifyOn),
		Metadata:            decodeStringMap(task.MetadataJSON),
		ExecutorPayload:     cloneRaw(task.ExecutorPayload),
		CreatedBy:           task.CreatedBy,
		CreatedAt:           task.CreatedAt,
		UpdatedAt:           task.UpdatedAt,
	}
}

func executionToSDK(exec *model.TimerExecution) *scheduledsdk.Execution {
	if exec == nil {
		return nil
	}
	return &scheduledsdk.Execution{
		ID:             exec.ID,
		TaskID:         exec.TaskID,
		ExecutorKey:    exec.ExecutorKey,
		Status:         scheduledsdk.ExecutionStatus(exec.Status),
		ExecutorRunID:  exec.ExecutorRunID,
		ScheduledAt:    exec.ScheduledAt,
		StartedAt:      exec.StartedAt,
		FinishedAt:     exec.FinishedAt,
		WorkerID:       exec.WorkerID,
		LeaseUntil:     exec.LeaseUntil,
		HeartbeatAt:    exec.HeartbeatAt,
		Attempt:        exec.Attempt,
		DurationMillis: exec.DurationMillis,
		OutputSummary:  exec.OutputSummary,
		ResultPayload:  cloneRaw(exec.ResultPayload),
		ErrorMessage:   exec.ErrorMessage,
		TraceID:        exec.TraceID,
		SourceType:     exec.SourceType,
		SourceRef:      exec.SourceRef,
		CreatedAt:      exec.CreatedAt,
		UpdatedAt:      exec.UpdatedAt,
	}
}

func taskSchedule(task *model.TimerTask) scheduledsdk.Schedule {
	schedule := scheduledsdk.Schedule{
		Type:            scheduledsdk.ScheduleType(task.ScheduleType),
		CronExpr:        task.CronExpr,
		IntervalSeconds: task.IntervalSeconds,
		MaxRuns:         task.MaxRuns,
		Timezone:        task.Timezone,
	}
	if task.RunAt != nil {
		schedule.RunAt = *task.RunAt
	}
	return schedule
}

func normalizeNotifyOn(value scheduledsdk.NotifyOn) string {
	switch value {
	case scheduledsdk.NotifyAll, scheduledsdk.NotifySuccess, scheduledsdk.NotifyFailed:
		return string(value)
	default:
		return string(scheduledsdk.NotifyNone)
	}
}

func joinStringList(values []string) string {
	normalized := normalizeStringList(values)
	return strings.Join(normalized, ",")
}

func splitStringList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return normalizeStringList(strings.Split(raw, ","))
}

func normalizeStringList(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultListPageSize
	}
	return page, pageSize
}

func mustJSON(v interface{}) json.RawMessage {
	if v == nil {
		return nil
	}
	data, err := json.Marshal(v)
	if err != nil || string(data) == "null" {
		return nil
	}
	return data
}

func decodeStringSlice(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func decodeStringMap(raw json.RawMessage) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	var out map[string]string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func cloneRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	out := make([]byte, len(raw))
	copy(out, raw)
	return json.RawMessage(out)
}
