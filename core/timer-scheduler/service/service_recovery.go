package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/core/timer-scheduler/model"
	"github.com/kageos/kageos/core/timer-scheduler/repository"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"gorm.io/gorm"
)

func (s *Service) RecoverStaleExecutions(ctx context.Context, limit int) (int, error) {
	now := s.now()
	execs, err := s.executionRepo.ListStale(ctx, now, limit)
	if err != nil {
		return 0, err
	}
	recovered := 0
	var recoverErr error
	for _, exec := range execs {
		switch exec.Status {
		case string(scheduledsdk.ExecutionStatusQueued):
			if exec.Attempt < s.opts.MaxDispatchAttempts {
				if err := s.requeueExecution(ctx, exec, now); err != nil {
					if !errors.Is(err, ErrInvalidTaskStatus) {
						recoverErr = errors.Join(recoverErr, err)
					}
					continue
				}
				recovered++
				continue
			}
			if err := s.timeoutExecution(ctx, exec, now, "timer-scheduler execution was not picked up before timeout"); err != nil {
				if !errors.Is(err, ErrInvalidTaskStatus) {
					recoverErr = errors.Join(recoverErr, err)
				}
				continue
			}
			recovered++
		case string(scheduledsdk.ExecutionStatusRunning):
			handled, err := s.handleExpiredRunningExecution(ctx, exec, now)
			if err != nil {
				if !errors.Is(err, ErrInvalidTaskStatus) {
					recoverErr = errors.Join(recoverErr, err)
				}
				continue
			}
			if handled {
				recovered++
			}
		}
	}
	cleared, err := s.recoverBrokenInflightReferences(limit)
	if err != nil {
		recoverErr = errors.Join(recoverErr, err)
	}
	recovered += cleared
	return recovered, recoverErr
}

func (s *Service) handleExpiredRunningExecution(ctx context.Context, exec *model.TimerExecution, now time.Time) (bool, error) {
	misses := exec.HeartbeatMisses + 1
	if misses < s.opts.MaxHeartbeatMisses {
		ok, err := s.executionRepo.TryRecordHeartbeatMiss(ctx, exec, now, now.Add(s.opts.ExecutionLeaseDuration), misses)
		if err != nil || !ok {
			return ok, err
		}
		return true, nil
	}
	return true, s.timeoutExecution(ctx, exec, now, "timer-scheduler execution heartbeat expired")
}

func (s *Service) requeueExecution(ctx context.Context, exec *model.TimerExecution, now time.Time) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		execRepo := s.executionRepo.WithDB(tx)
		taskRepo := s.taskRepo.WithDB(tx)
		outboxRepo := s.outboxRepo.WithDB(tx)
		task, err := taskRepo.GetByID(ctx, exec.TaskID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return s.timeoutOrphanExecutionWithRepo(execRepo, exec, now, "timer-scheduler task not found during recovery")
			}
			return err
		}
		if isTerminalTaskStatus(task.Status) {
			message := "timer-scheduler task is not eligible for execution recovery"
			ok, finishErr := execRepo.TryFinish(ctx, exec.TaskID, exec.ID, map[string]interface{}{
				"status":        string(scheduledsdk.ExecutionStatusCancelled),
				"finished_at":   now,
				"lease_until":   nil,
				"error_message": message,
			})
			if finishErr != nil {
				return finishErr
			}
			if ok {
				_, _ = taskRepo.TryClearInflight(ctx, task.ID, exec.ID, message)
			}
			return nil
		}
		ok, err := execRepo.TryRequeueQueued(ctx, exec, now, now.Add(s.opts.QueueAckTimeout))
		if err != nil {
			return err
		}
		if !ok {
			return ErrInvalidTaskStatus
		}
		exec.Attempt++
		exec.LeaseUntil = ptrTime(now.Add(s.opts.QueueAckTimeout))
		exec.LastDispatchedAt = &now
		return outboxRepo.Create(ctx, s.executionRequestedOutbox(task, exec))
	})
}

func (s *Service) recoverBrokenInflightReferences(limit int) (int, error) {
	tasks, err := s.taskRepo.ListBrokenInflightReferences(context.Background(), limit)
	if err != nil {
		return 0, err
	}
	recovered := 0
	var recoverErr error
	for _, task := range tasks {
		executionID := task.InflightExecutionID
		if executionID == 0 {
			continue
		}
		message := fmt.Sprintf("timer-scheduler cleared stale inflight execution reference: execution_id=%d", executionID)
		ok, err := s.taskRepo.TryClearInflight(context.Background(), task.ID, executionID, message)
		if err != nil {
			recoverErr = errors.Join(recoverErr, err)
			continue
		}
		if ok {
			recovered++
		}
	}
	return recovered, recoverErr
}

func (s *Service) recoverBrokenInflightReferenceForTask(taskID int64) (bool, error) {
	task, err := s.taskRepo.GetBrokenInflightReferenceByID(context.Background(), taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	executionID := task.InflightExecutionID
	if executionID == 0 {
		return false, nil
	}
	message := fmt.Sprintf("timer-scheduler cleared stale inflight execution reference: execution_id=%d", executionID)
	return s.taskRepo.TryClearInflight(context.Background(), task.ID, executionID, message)
}

func (s *Service) timeoutExecution(ctx context.Context, exec *model.TimerExecution, now time.Time, message string) error {
	durationMillis := int64(0)
	if exec.StartedAt != nil && now.After(*exec.StartedAt) {
		durationMillis = now.Sub(*exec.StartedAt).Milliseconds()
	}
	err := s.markExecutionFinished(ctx, scheduledsdk.MarkExecutionFinishedRequest{
		TaskID:         exec.TaskID,
		ExecutionID:    exec.ID,
		Status:         scheduledsdk.ExecutionStatusTimeout,
		FinishedAt:     now,
		DurationMillis: durationMillis,
		ErrorMessage:   message,
	}, false)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.timeoutOrphanExecution(exec, now, message)
	}
	if errors.Is(err, ErrTaskBusy) {
		return s.timeoutOrphanExecution(exec, now, message)
	}
	return err
}

func (s *Service) timeoutOrphanExecution(exec *model.TimerExecution, now time.Time, message string) error {
	return s.timeoutOrphanExecutionWithRepo(s.executionRepo, exec, now, message)
}

func (s *Service) timeoutOrphanExecutionWithRepo(execRepo *repository.TimerExecutionRepository, exec *model.TimerExecution, now time.Time, message string) error {
	durationMillis := int64(0)
	if exec.StartedAt != nil && now.After(*exec.StartedAt) {
		durationMillis = now.Sub(*exec.StartedAt).Milliseconds()
	}
	ok, err := execRepo.TryFinish(context.Background(), exec.TaskID, exec.ID, map[string]interface{}{
		"status":          string(scheduledsdk.ExecutionStatusTimeout),
		"finished_at":     now,
		"duration_millis": durationMillis,
		"error_message":   strings.TrimSpace(message),
		"lease_until":     nil,
	})
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidTaskStatus
	}
	return nil
}
