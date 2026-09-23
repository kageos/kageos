package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kageos/kageos/core/timer-scheduler/model"
	"github.com/kageos/kageos/core/timer-scheduler/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func taskAuditSnapshot(task *model.TimerTask) json.RawMessage {
	if task == nil {
		return nil
	}
	// Executor payloads can contain credentials, form values or Agent prompts.
	// Record whether they changed without copying their contents into audit logs.
	return mustJSON(map[string]any{
		"title": task.Title, "status": task.Status, "executor": task.ExecutorKey,
		"resource_path": task.ResourceKey, "schedule_type": task.ScheduleType,
		"run_at": task.RunAt, "cron": task.CronExpr, "interval_seconds": task.IntervalSeconds,
		"timezone": task.Timezone, "max_runs": task.MaxRuns, "overlap_policy": task.OverlapPolicy,
		"max_parallelism": task.MaxParallelism, "execution_user": task.RequestUser,
		"payload_digest": fmt.Sprintf("%x", sha256.Sum256(task.ExecutorPayload)),
		"description":    task.Description, "category": task.Category, "tags": json.RawMessage(task.TagsJSON),
		"metadata_digest": fmt.Sprintf("%x", sha256.Sum256(task.MetadataJSON)),
	})
}

func (s *Service) enqueueTaskAudit(ctx context.Context, action string, task *model.TimerTask, before, after json.RawMessage, executionID int64, operationErr error) error {
	if task == nil {
		return nil
	}
	path := task.ResourceKey
	if task.ResourceScope == "system" {
		path = "/system/platform"
	}
	if !strings.HasPrefix(path, "/") || len(strings.Split(strings.Trim(path, "/"), "/")) < 2 {
		// Tasks without a workspace resource remain visible in system history.
		path = "/system/platform"
	}
	actor := contextx.GetRequestUser(ctx)
	if actor == "" {
		actor = "unknown"
	}
	event := dto.TaskAuditEvent{EventID: uuid.NewString(), OccurredAt: s.now(), Action: action, TaskID: task.ID,
		ResourcePath: path, ResourceName: task.Title, Actor: actor, Initiator: contextx.GetInitiatorUser(ctx),
		Source: contextx.GetAuditClientSource(ctx), SourceType: contextx.GetSourceType(ctx), SourceRef: contextx.GetSourceRef(ctx),
		SessionID: contextx.GetWorkspaceSessionID(ctx), ToolCallID: contextx.GetToolCallID(ctx), TraceID: contextx.GetTraceId(ctx),
		Status: "success", ExecutionID: executionID, Before: before, After: after}
	if operationErr != nil {
		event.Status = "failed"
		event.Error = operationErr.Error()
	}
	return s.outboxRepo.Create(&model.TimerOutboxEvent{EventID: event.EventID, EventType: "timer.task.audit", Subject: dto.TaskAuditSubject,
		AggregateID: task.ID, Payload: mustJSON(event), Status: outboxStatusPending})
}

// The task mutation and success event commit together. Failed transactions emit a
// separate failure event; they never retain the uncommitted "after" snapshot.
func (s *Service) withTaskAudit(ctx context.Context, action string, taskID int64, candidate *model.TimerTask, key string, operation func(*Service) (int64, int64, error)) error {
	var before json.RawMessage
	var target *model.TimerTask
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		local := *s
		local.db = tx
		local.taskRepo = repository.NewTimerTaskRepository(tx)
		local.executionRepo = repository.NewTimerExecutionRepository(tx)
		local.outboxRepo = repository.NewTimerOutboxRepository(tx)
		if taskID > 0 {
			var row model.TimerTask
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, taskID).Error; err != nil {
				return err
			}
			target = &row
		} else if key != "" {
			existing, lookupErr := local.taskRepo.GetByIdempotencyKey(key)
			if lookupErr != nil && !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
				return lookupErr
			}
			target = existing
		}
		before = taskAuditSnapshot(target)
		if target == nil {
			target = candidate
		}
		id, executionID, err := operation(&local)
		if err != nil {
			return err
		}
		var after json.RawMessage
		if action != "timer.task.deleted" {
			current, err := local.taskRepo.GetByID(id)
			if err != nil {
				return err
			}
			target = current
			after = taskAuditSnapshot(current)
		}
		if bytes.Equal(before, after) && action != "timer.task.run_now" {
			return nil
		}
		return local.enqueueTaskAudit(ctx, action, target, before, after, executionID, nil)
	})
	if err != nil && target != nil {
		writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if auditErr := s.enqueueTaskAudit(writeCtx, action, target, before, nil, 0, err); auditErr != nil {
			return errors.Join(err, fmt.Errorf("record failed task operation: %w", auditErr))
		}
	}
	return err
}

func (s *Service) CreateTask(ctx context.Context, req scheduledsdk.CreateTaskRequest) (result *scheduledsdk.Task, err error) {
	err = s.withTaskAudit(ctx, "timer.task.created", 0, &model.TimerTask{Title: req.Title, ResourceKey: req.ResourceKey, ResourceScope: req.ResourceScope}, req.IdempotencyKey, func(local *Service) (int64, int64, error) {
		var callErr error
		result, callErr = local.createTask(ctx, req)
		if callErr != nil {
			return 0, 0, callErr
		}
		return result.ID, 0, nil
	})
	if err != nil {
		result = nil
	}
	return
}

func (s *Service) UpdateTask(ctx context.Context, id int64, req scheduledsdk.UpdateTaskRequest) (result *scheduledsdk.Task, err error) {
	err = s.withTaskAudit(ctx, "timer.task.updated", id, nil, "", func(local *Service) (int64, int64, error) {
		var callErr error
		result, callErr = local.updateTask(ctx, id, req)
		return id, 0, callErr
	})
	if err != nil {
		result = nil
	}
	return
}

func (s *Service) taskStateAudit(ctx context.Context, id int64, action string, operation func(*Service) error) error {
	return s.withTaskAudit(ctx, action, id, nil, "", func(local *Service) (int64, int64, error) { return id, 0, operation(local) })
}

func (s *Service) PauseTask(ctx context.Context, id int64) error {
	return s.taskStateAudit(ctx, id, "timer.task.paused", func(local *Service) error { return local.pauseTask(ctx, id) })
}
func (s *Service) ResumeTask(ctx context.Context, id int64) error {
	return s.taskStateAudit(ctx, id, "timer.task.resumed", func(local *Service) error { return local.resumeTask(ctx, id) })
}
func (s *Service) CancelTask(ctx context.Context, id int64) error {
	return s.taskStateAudit(ctx, id, "timer.task.cancelled", func(local *Service) error { return local.cancelTask(ctx, id) })
}
func (s *Service) DeleteTask(ctx context.Context, id int64) error {
	return s.taskStateAudit(ctx, id, "timer.task.deleted", func(local *Service) error { return local.deleteTask(ctx, id) })
}
func (s *Service) RunNow(ctx context.Context, id int64) (result *scheduledsdk.Execution, err error) {
	err = s.withTaskAudit(ctx, "timer.task.run_now", id, nil, "", func(local *Service) (int64, int64, error) {
		var callErr error
		result, callErr = local.runNow(ctx, id)
		if callErr != nil {
			return id, 0, callErr
		}
		return id, result.ID, nil
	})
	if err != nil {
		result = nil
	}
	return
}
