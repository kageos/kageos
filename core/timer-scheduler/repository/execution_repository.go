package repository

import (
	"context"
	"time"

	"github.com/kageos/kageos/core/timer-scheduler/model"
	"gorm.io/gorm"
)

type TimerExecutionRepository struct {
	db *gorm.DB
}

func NewTimerExecutionRepository(db *gorm.DB) *TimerExecutionRepository {
	return &TimerExecutionRepository{db: db}
}

func (r *TimerExecutionRepository) WithDB(db *gorm.DB) *TimerExecutionRepository {
	return &TimerExecutionRepository{db: db}
}

func (r *TimerExecutionRepository) Create(ctx context.Context, exec *model.TimerExecution) error {
	return r.db.WithContext(ctx).Create(exec).Error
}

func (r *TimerExecutionRepository) GetByID(ctx context.Context, taskID, executionID int64) (*model.TimerExecution, error) {
	var exec model.TimerExecution
	if err := r.db.WithContext(ctx).Where("task_id = ? AND id = ?", taskID, executionID).First(&exec).Error; err != nil {
		return nil, err
	}
	return &exec, nil
}

func (r *TimerExecutionRepository) ListByTaskID(ctx context.Context, taskID int64, status string, offset, limit int) ([]*model.TimerExecution, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.TimerExecution{}).Where("task_id = ?", taskID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	var list []*model.TimerExecution
	if err := query.Order("created_at DESC, id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *TimerExecutionRepository) ListStale(ctx context.Context, now time.Time, limit int) ([]*model.TimerExecution, error) {
	query := r.db.WithContext(ctx).
		Where("status IN ? AND lease_until IS NOT NULL AND lease_until < ?", []string{"queued", "running"}, now).
		Order("lease_until ASC, id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	var list []*model.TimerExecution
	if err := query.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *TimerExecutionRepository) TryMarkRunning(ctx context.Context, taskID, executionID int64, workerID, executorRunID string, startedAt, leaseUntil time.Time) (bool, error) {
	updates := map[string]interface{}{
		"status":           "running",
		"started_at":       startedAt,
		"worker_id":        workerID,
		"heartbeat_at":     startedAt,
		"heartbeat_misses": 0,
		"lease_until":      leaseUntil,
	}
	if executorRunID != "" {
		updates["executor_run_id"] = executorRunID
	}
	result := r.db.WithContext(ctx).Model(&model.TimerExecution{}).
		Where("task_id = ? AND id = ? AND status = ?", taskID, executionID, "queued").
		Where(eligibleTimerTaskForExecutionSQL, []string{"pending", "paused"}, "manual", "cancelled").
		Updates(updates)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TimerExecutionRepository) IsRunningForActiveTask(ctx context.Context, taskID, executionID int64, workerID, executorRunID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.TimerExecution{}).
		Where("task_id = ? AND id = ? AND status = ?", taskID, executionID, "running").
		Where("worker_id = ? AND executor_run_id = ?", workerID, executorRunID).
		Where(eligibleTimerTaskForExecutionSQL, []string{"pending", "paused"}, "manual", "cancelled").
		Count(&count).Error
	return count > 0, err
}

func (r *TimerExecutionRepository) CancelActiveByTaskID(ctx context.Context, taskID int64, finishedAt time.Time, message string) error {
	return r.db.WithContext(ctx).Model(&model.TimerExecution{}).
		Where("task_id = ? AND status IN ?", taskID, []string{"queued", "running"}).
		Updates(map[string]interface{}{
			"status":        "cancelled",
			"finished_at":   finishedAt,
			"lease_until":   nil,
			"error_message": message,
		}).Error
}

const eligibleTimerTaskForExecutionSQL = `EXISTS (
	SELECT 1 FROM timer_task AS eligible_task
	WHERE eligible_task.id = timer_execution.task_id
	  AND eligible_task.deleted_at IS NULL
	  AND (
	    eligible_task.status IN ?
	    OR (timer_execution.trigger_type = ? AND eligible_task.status <> ?)
	  )
)`

func (r *TimerExecutionRepository) TryHeartbeat(ctx context.Context, taskID, executionID int64, workerID, executorRunID string, heartbeatAt, leaseUntil time.Time) (bool, error) {
	query := r.db.WithContext(ctx).Model(&model.TimerExecution{}).
		Where("task_id = ? AND id = ? AND status = ? AND executor_run_id = ?", taskID, executionID, "running", executorRunID)
	if workerID != "" {
		query = query.Where("worker_id = ?", workerID)
	}
	result := query.Updates(map[string]interface{}{
		"heartbeat_at":     heartbeatAt,
		"heartbeat_misses": 0,
		"lease_until":      leaseUntil,
	})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TimerExecutionRepository) TryFinishByRunID(ctx context.Context, taskID, executionID int64, executorRunID string, updates map[string]interface{}) (bool, error) {
	result := r.db.WithContext(ctx).Model(&model.TimerExecution{}).
		Where("task_id = ? AND id = ? AND status = ? AND executor_run_id = ?", taskID, executionID, "running", executorRunID).
		Updates(updates)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TimerExecutionRepository) TryRecordHeartbeatMiss(ctx context.Context, exec *model.TimerExecution, now, leaseUntil time.Time, heartbeatMisses int) (bool, error) {
	result := r.db.WithContext(ctx).Model(&model.TimerExecution{}).
		Where("task_id = ? AND id = ? AND status = ? AND lease_until IS NOT NULL AND lease_until < ?", exec.TaskID, exec.ID, "running", now).
		Updates(map[string]interface{}{
			"heartbeat_misses": heartbeatMisses,
			"lease_until":      leaseUntil,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TimerExecutionRepository) TryFinish(ctx context.Context, taskID, executionID int64, updates map[string]interface{}) (bool, error) {
	result := r.db.WithContext(ctx).Model(&model.TimerExecution{}).
		Where("task_id = ? AND id = ? AND status IN ?", taskID, executionID, []string{"queued", "running"}).
		Updates(updates)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TimerExecutionRepository) TryRequeueQueued(ctx context.Context, exec *model.TimerExecution, now, leaseUntil time.Time) (bool, error) {
	result := r.db.WithContext(ctx).Model(&model.TimerExecution{}).
		Where("task_id = ? AND id = ? AND status = ? AND lease_until IS NOT NULL AND lease_until < ?", exec.TaskID, exec.ID, "queued", now).
		Updates(map[string]interface{}{
			"attempt":            exec.Attempt + 1,
			"lease_until":        leaseUntil,
			"last_dispatched_at": now,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}
