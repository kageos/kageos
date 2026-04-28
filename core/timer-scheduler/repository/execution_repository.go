package repository

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/timer-scheduler/model"
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

func (r *TimerExecutionRepository) GetByID(taskID, id int64) (*model.TimerExecution, error) {
	var exec model.TimerExecution
	if err := r.db.Where("task_id = ? AND id = ?", taskID, id).First(&exec).Error; err != nil {
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
	if err := query.Order("scheduled_at DESC, id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *TimerExecutionRepository) ListStale(now, queuedBefore time.Time, limit int) ([]*model.TimerExecution, error) {
	query := r.db.
		Where("(status = ? AND scheduled_at < ?) OR (status = ? AND lease_until IS NOT NULL AND lease_until < ?)", "queued", queuedBefore, "running", now).
		Order("scheduled_at ASC, id ASC")
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
	result := r.db.Model(&model.TimerExecution{}).
		Where("task_id = ? AND id = ? AND status = ?", taskID, executionID, "queued").
		Updates(map[string]interface{}{
			"status":          "running",
			"worker_id":       workerID,
			"executor_run_id": executorRunID,
			"started_at":      startedAt,
			"heartbeat_at":    startedAt,
			"lease_until":     leaseUntil,
			"attempt":         gorm.Expr("attempt + ?", 1),
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
