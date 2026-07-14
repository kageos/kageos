package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/core/timer-scheduler/model"
	"github.com/kageos/kageos/core/timer-scheduler/repository"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/kageos/kageos/pkg/subjects"
	"gorm.io/gorm"
)

func (s *Service) GetExecution(ctx context.Context, taskID, executionID int64) (*scheduledsdk.Execution, error) {
	exec, err := s.executionRepo.GetByID(ctx, taskID, executionID)
	if err != nil {
		return nil, err
	}
	return executionToSDK(exec), nil
}

func (s *Service) ListExecutions(ctx context.Context, taskID int64, req scheduledsdk.ListExecutionsRequest) (*scheduledsdk.ListExecutionsResponse, error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	list, total, err := s.executionRepo.ListByTaskID(ctx, taskID, req.Status, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, err
	}
	out := make([]*scheduledsdk.Execution, 0, len(list))
	for _, item := range list {
		out = append(out, executionToSDK(item))
	}
	return &scheduledsdk.ListExecutionsResponse{List: out, Total: total}, nil
}

func (s *Service) MarkExecutionStarted(ctx context.Context, req scheduledsdk.MarkExecutionStartedRequest) error {
	executorRunID := strings.TrimSpace(req.ExecutorRunID)
	if executorRunID == "" {
		return fmt.Errorf("%w: executor_run_id is required", scheduledsdk.ErrInvalidRequest)
	}
	startedAt := req.StartedAt
	if startedAt.IsZero() {
		startedAt = s.now()
	}
	workerID := strings.TrimSpace(req.WorkerID)
	ok, err := s.executionRepo.TryMarkRunning(ctx, req.TaskID, req.ExecutionID, workerID, executorRunID, startedAt, startedAt.Add(s.opts.ExecutionLeaseDuration))
	if err != nil {
		return err
	}
	if !ok {
		idempotent, lookupErr := s.executionRepo.IsRunningForActiveTask(ctx, req.TaskID, req.ExecutionID, workerID, executorRunID)
		if lookupErr != nil {
			return lookupErr
		}
		if idempotent {
			return nil
		}
		return ErrInvalidTaskStatus
	}
	return nil
}

func (s *Service) MarkExecutionHeartbeat(ctx context.Context, req scheduledsdk.MarkExecutionHeartbeatRequest) error {
	executorRunID := strings.TrimSpace(req.ExecutorRunID)
	if executorRunID == "" {
		return fmt.Errorf("%w: executor_run_id is required", scheduledsdk.ErrInvalidRequest)
	}
	heartbeatAt := req.HeartbeatAt
	if heartbeatAt.IsZero() {
		heartbeatAt = s.now()
	}
	ok, err := s.executionRepo.TryHeartbeat(ctx, req.TaskID, req.ExecutionID, strings.TrimSpace(req.WorkerID), executorRunID, heartbeatAt, heartbeatAt.Add(s.opts.ExecutionLeaseDuration))
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidTaskStatus
	}
	return nil
}

func (s *Service) MarkExecutionFinished(ctx context.Context, req scheduledsdk.MarkExecutionFinishedRequest) error {
	if strings.TrimSpace(req.ExecutorRunID) == "" {
		return fmt.Errorf("%w: executor_run_id is required", scheduledsdk.ErrInvalidRequest)
	}
	return s.markExecutionFinished(ctx, req, true)
}

func (s *Service) markExecutionFinished(ctx context.Context, req scheduledsdk.MarkExecutionFinishedRequest, workerBound bool) error {
	if req.Status == "" {
		return fmt.Errorf("%w: execution status is required", scheduledsdk.ErrInvalidRequest)
	}
	finishedAt := req.FinishedAt
	if finishedAt.IsZero() {
		finishedAt = s.now()
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		execRepo := s.executionRepo.WithDB(tx)
		taskRepo := s.taskRepo.WithDB(tx)
		outboxRepo := s.outboxRepo.WithDB(tx)

		exec, err := execRepo.GetByID(ctx, req.TaskID, req.ExecutionID)
		if err != nil {
			return err
		}
		updates := map[string]interface{}{
			"status":          string(req.Status),
			"finished_at":     finishedAt,
			"duration_millis": req.DurationMillis,
			"output_summary":  strings.TrimSpace(req.OutputSummary),
			"result_payload":  cloneRaw(req.ResultPayload),
			"error_message":   strings.TrimSpace(req.ErrorMessage),
			"lease_until":     nil,
		}
		var ok bool
		if workerBound {
			ok, err = execRepo.TryFinishByRunID(ctx, req.TaskID, req.ExecutionID, strings.TrimSpace(req.ExecutorRunID), updates)
		} else {
			ok, err = execRepo.TryFinish(ctx, req.TaskID, req.ExecutionID, updates)
		}
		if err != nil {
			return err
		}
		if !ok {
			return ErrInvalidTaskStatus
		}

		task, err := taskRepo.GetByIDForUpdate(ctx, req.TaskID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return s.createExecutionFinishedOutbox(ctx, outboxRepo, req, exec, finishedAt)
			}
			return err
		}
		expectedTaskStatus := task.Status
		if task.LastExecutionID < req.ExecutionID {
			task.LastExecutionID = req.ExecutionID
		}
		task.LastErrorMessage = strings.TrimSpace(req.ErrorMessage)
		if exec.TriggerType != triggerManual {
			task.RunCount++
			switch scheduledsdk.TaskStatus(task.Status) {
			case scheduledsdk.TaskStatusPending:
				task.Status, task.NextRunAt, task.LastErrorMessage = computeTaskNextState(task, req.Status == scheduledsdk.ExecutionStatusSuccess, finishedAt)
			case scheduledsdk.TaskStatusCancelled:
				task.NextRunAt = nil
			}
		}
		clearInflight := task.InflightExecutionID == req.ExecutionID
		if exec.TriggerType != triggerManual || clearInflight {
			settled, settleErr := taskRepo.TrySettleExecution(ctx, task, req.ExecutionID, expectedTaskStatus, clearInflight)
			if settleErr != nil {
				return settleErr
			}
			if !settled {
				return ErrTaskBusy
			}
		}
		return s.createExecutionFinishedOutbox(ctx, outboxRepo, req, exec, finishedAt)
	})
}

func (s *Service) createExecutionFinishedOutbox(ctx context.Context, outboxRepo *repository.TimerOutboxRepository, req scheduledsdk.MarkExecutionFinishedRequest, exec *model.TimerExecution, finishedAt time.Time) error {
	payload, err := json.Marshal(map[string]interface{}{
		"task_id":        req.TaskID,
		"execution_id":   req.ExecutionID,
		"executor_key":   exec.ExecutorKey,
		"status":         req.Status,
		"finished_at":    finishedAt,
		"trace_id":       exec.TraceID,
		"trigger_type":   exec.TriggerType,
		"source_type":    exec.SourceType,
		"source_ref":     exec.SourceRef,
		"resource_scope": exec.ResourceScope,
		"resource_key":   exec.ResourceKey,
	})
	if err != nil {
		return err
	}
	return outboxRepo.Create(ctx, &model.TimerOutboxEvent{
		EventID:     fmt.Sprintf("timer-execution-finished-%d", req.ExecutionID),
		EventType:   eventTypeFinished,
		Subject:     subjects.TimerExecutionFinishedSubject,
		AggregateID: req.ExecutionID,
		Payload:     payload,
		Status:      outboxStatusPending,
	})
}
