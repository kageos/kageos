package repository

import (
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

func (r *TimerExecutionRepository) Create(exec *model.TimerExecution) error {
	return r.db.Create(exec).Error
}

func (r *TimerExecutionRepository) GetByID(taskID, executionID int64) (*model.TimerExecution, error) {
	var exec model.TimerExecution
	if err := r.db.Where("task_id = ? AND id = ?", taskID, executionID).First(&exec).Error; err != nil {
		return nil, err
	}
	return &exec, nil
}

func (r *TimerExecutionRepository) ListByTaskID(taskID int64, status string, offset, limit int) ([]*model.TimerExecution, int64, error) {
	query := r.db.Model(&model.TimerExecution{}).Where("task_id = ?", taskID)
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

func (r *TimerExecutionRepository) ListStale(now time.Time, limit int) ([]*model.TimerExecution, error) {
	query := r.db.
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

func (r *TimerExecutionRepository) TryMarkRunning(taskID, executionID int64, workerID, executorRunID string, startedAt, leaseUntil time.Time) (bool, error) {
	updates := map[string]interface{}{
		"status":       "running",
		"started_at":   startedAt,
		"worker_id":    workerID,
		"heartbeat_at": startedAt,
		"lease_until":  leaseUntil,
	}
	if executorRunID != "" {
		updates["executor_run_id"] = executorRunID
	}
	result := r.db.Model(&model.TimerExecution{}).
		Where("task_id = ? AND id = ? AND status = ?", taskID, executionID, "queued").
		Updates(updates)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TimerExecutionRepository) TryHeartbeat(taskID, executionID int64, workerID string, heartbeatAt, leaseUntil time.Time) (bool, error) {
	query := r.db.Model(&model.TimerExecution{}).
		Where("task_id = ? AND id = ? AND status = ?", taskID, executionID, "running")
	if workerID != "" {
		query = query.Where("worker_id = ?", workerID)
	}
	result := query.Updates(map[string]interface{}{
		"heartbeat_at": heartbeatAt,
		"lease_until":  leaseUntil,
	})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TimerExecutionRepository) TryFinish(taskID, executionID int64, updates map[string]interface{}) (bool, error) {
	result := r.db.Model(&model.TimerExecution{}).
		Where("task_id = ? AND id = ? AND status IN ?", taskID, executionID, []string{"queued", "running"}).
		Updates(updates)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TimerExecutionRepository) TryRequeueQueued(exec *model.TimerExecution, now, leaseUntil time.Time) (bool, error) {
	result := r.db.Model(&model.TimerExecution{}).
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
