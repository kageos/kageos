package service

import (
	"github.com/kageos/kageos/core/timer-scheduler/model"
	"github.com/kageos/kageos/pkg/scheduledsdk"
)

func taskToSDK(task *model.TimerTask) *scheduledsdk.Task {
	if task == nil {
		return nil
	}
	idempotencyKey := ""
	if task.IdempotencyKey != nil {
		idempotencyKey = *task.IdempotencyKey
	}
	schedule := scheduledsdk.Schedule{
		Type:            scheduledsdk.ScheduleType(task.ScheduleType),
		CronExpr:        task.CronExpr,
		IntervalSeconds: task.IntervalSeconds,
		Timezone:        task.Timezone,
		MaxRuns:         task.MaxRuns,
	}
	if task.RunAt != nil {
		schedule.RunAt = *task.RunAt
	}
	return &scheduledsdk.Task{
		ID:                  task.ID,
		Title:               task.Title,
		Description:         task.Description,
		Category:            task.Category,
		Tags:                decodeStringList(task.TagsJSON),
		IdempotencyKey:      idempotencyKey,
		ExecutorKey:         task.ExecutorKey,
		ExecutorPayload:     cloneRaw(task.ExecutorPayload),
		Metadata:            decodeStringMap(task.MetadataJSON),
		Status:              scheduledsdk.TaskStatus(task.Status),
		Schedule:            schedule,
		NextRunAt:           task.NextRunAt,
		RunCount:            task.RunCount,
		InflightExecutionID: task.InflightExecutionID,
		LastExecutionID:     task.LastExecutionID,
		LastErrorMessage:    task.LastErrorMessage,
		SourceType:          task.SourceType,
		SourceRef:           task.SourceRef,
		ResourceScope:       task.ResourceScope,
		ResourceKey:         task.ResourceKey,
		RequestUser:         task.RequestUser,
		RequestUserDept:     task.RequestUserDept,
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
		ID:               exec.ID,
		TaskID:           exec.TaskID,
		ExecutorKey:      exec.ExecutorKey,
		Status:           scheduledsdk.ExecutionStatus(exec.Status),
		TriggerType:      exec.TriggerType,
		ExecutorRunID:    exec.ExecutorRunID,
		ScheduledAt:      exec.ScheduledAt,
		StartedAt:        exec.StartedAt,
		FinishedAt:       exec.FinishedAt,
		WorkerID:         exec.WorkerID,
		LeaseUntil:       exec.LeaseUntil,
		HeartbeatAt:      exec.HeartbeatAt,
		Attempt:          exec.Attempt,
		DurationMillis:   exec.DurationMillis,
		OutputSummary:    exec.OutputSummary,
		ResultPayload:    cloneRaw(exec.ResultPayload),
		ErrorMessage:     exec.ErrorMessage,
		TraceID:          exec.TraceID,
		SourceType:       exec.SourceType,
		SourceRef:        exec.SourceRef,
		ResourceScope:    exec.ResourceScope,
		ResourceKey:      exec.ResourceKey,
		RequestUser:      exec.RequestUser,
		RequestUserDept:  exec.RequestUserDept,
		LastDispatchedAt: exec.LastDispatchedAt,
		CreatedAt:        exec.CreatedAt,
		UpdatedAt:        exec.UpdatedAt,
	}
}
