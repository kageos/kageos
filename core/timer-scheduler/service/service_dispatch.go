package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kageos/kageos/core/timer-scheduler/model"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"gorm.io/gorm"
)

func (s *Service) DispatchDue(ctx context.Context, owner string, limit int) ([]*scheduledsdk.Execution, error) {
	now := s.now()
	tasks, err := s.taskRepo.ListDue(ctx, now, limit)
	if err != nil {
		return nil, err
	}
	out := make([]*scheduledsdk.Execution, 0, len(tasks))
	var dispatchErr error
	for _, task := range tasks {
		ok, err := s.taskRepo.TryAcquireDispatch(ctx, task.ID, owner, now, now.Add(s.opts.DispatchLeaseDuration))
		if err != nil {
			dispatchErr = errors.Join(dispatchErr, fmt.Errorf("timer-scheduler dispatch acquire task %d: %w", task.ID, err))
			continue
		}
		if !ok {
			continue
		}
		exec, err := s.dispatchTask(ctx, task, owner, scheduledAtForTask(task, now), triggerScheduled)
		if err != nil {
			dispatchErr = errors.Join(dispatchErr, fmt.Errorf("timer-scheduler dispatch task %d: %w", task.ID, err))
			continue
		}
		out = append(out, executionToSDK(exec))
	}
	return out, dispatchErr
}

func (s *Service) dispatchTask(ctx context.Context, task *model.TimerTask, owner string, scheduledAt time.Time, triggerType string) (*model.TimerExecution, error) {
	if task == nil {
		return nil, ErrInvalidTaskStatus
	}
	if triggerType == "" {
		triggerType = triggerScheduled
	}
	var created *model.TimerExecution
	now := s.now()
	err := s.db.Transaction(func(tx *gorm.DB) error {
		execRepo := s.executionRepo.WithDB(tx)
		taskRepo := s.taskRepo.WithDB(tx)
		outboxRepo := s.outboxRepo.WithDB(tx)

		exec := &model.TimerExecution{
			TaskID:           task.ID,
			ExecutorKey:      task.ExecutorKey,
			Status:           string(scheduledsdk.ExecutionStatusQueued),
			TriggerType:      triggerType,
			ScheduledAt:      scheduledAt,
			LeaseUntil:       ptrTime(now.Add(s.opts.QueueAckTimeout)),
			Attempt:          1,
			LastDispatchedAt: &now,
			TraceID:          uuid.NewString(),
			SourceType:       task.SourceType,
			SourceRef:        task.SourceRef,
			ResourceScope:    task.ResourceScope,
			ResourceKey:      task.ResourceKey,
			RequestUser:      scheduledTaskRequestUser(task),
			RequestUserDept:  task.RequestUserDept,
		}
		if err := execRepo.Create(ctx, exec); err != nil {
			return err
		}
		var (
			ok  bool
			err error
		)
		if triggerType == triggerManual {
			ok, err = taskRepo.TrySetManualInflight(ctx, task.ID, exec.ID)
			if err == nil && !ok {
				ok, err = taskRepo.RecordManualExecutionSubmitted(ctx, task.ID, exec.ID)
			}
		} else {
			nextRunAt, nextErr := nextRunAfterScheduledDispatch(task, now)
			if nextErr != nil {
				return nextErr
			}
			ok, err = taskRepo.TrySetInflight(ctx, task.ID, exec.ID, owner, nextRunAt)
		}
		if err != nil {
			return err
		}
		if !ok {
			if triggerType == triggerManual {
				return ErrInvalidTaskStatus
			}
			return ErrTaskBusy
		}
		if err := outboxRepo.Create(ctx, s.executionRequestedOutbox(task, exec)); err != nil {
			return err
		}
		created = exec
		return nil
	})
	return created, err
}
