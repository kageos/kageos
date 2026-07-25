package repository

import (
	"errors"
	"time"

	"github.com/kageos/kageos/core/timer-scheduler/model"
	"gorm.io/gorm"
)

type TimerExecutionRepository struct {
	db *gorm.DB
}

type ActiveExecutionState struct {
	Queued  int64
	Running int64
	Waiting *model.TimerExecution
}

func (s ActiveExecutionState) ActiveCount() int64 {
	return s.Queued + s.Running
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

func (r *TimerExecutionRepository) GetActiveState(taskID int64) (ActiveExecutionState, error) {
	var rows []struct {
		Status string
		Count  int64
	}
	if err := r.db.Model(&model.TimerExecution{}).
		Select("status, COUNT(*) AS count").
		Where("task_id = ? AND status IN ?", taskID, []string{"queued", "running"}).
		Group("status").
		Scan(&rows).Error; err != nil {
		return ActiveExecutionState{}, err
	}
	state := ActiveExecutionState{}
	for _, row := range rows {
		switch row.Status {
		case "queued":
			state.Queued = row.Count
		case "running":
			state.Running = row.Count
		}
	}
	var waiting model.TimerExecution
	err := r.db.Where("task_id = ? AND status = ?", taskID, "waiting").Order("id DESC").First(&waiting).Error
	if err == nil {
		state.Waiting = &waiting
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return ActiveExecutionState{}, err
	}
	return state, nil
}

func (r *TimerExecutionRepository) UpdateWaitingScheduledAt(taskID, executionID int64, scheduledAt time.Time) (bool, error) {
	result := r.db.Model(&model.TimerExecution{}).
		Where("task_id = ? AND id = ? AND status = ?", taskID, executionID, "waiting").
		Update("scheduled_at", scheduledAt)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TimerExecutionRepository) TryPromoteWaiting(exec *model.TimerExecution, now, leaseUntil time.Time) (bool, error) {
	result := r.db.Model(&model.TimerExecution{}).
		Where("task_id = ? AND id = ? AND status = ?", exec.TaskID, exec.ID, "waiting").
		Updates(map[string]interface{}{
			"status":             "queued",
			"attempt":            1,
			"lease_until":        leaseUntil,
			"last_dispatched_at": now,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TimerExecutionRepository) LatestActiveExecutionID(taskID int64) (int64, error) {
	var exec model.TimerExecution
	err := r.db.Select("id").Where("task_id = ? AND status IN ?", taskID, []string{"queued", "running"}).Order("id DESC").First(&exec).Error
	if err != nil {
		return 0, err
	}
	return exec.ID, nil
}

func (r *TimerExecutionRepository) CancelWaitingByTask(taskID int64, now time.Time, message string) error {
	return r.db.Model(&model.TimerExecution{}).
		Where("task_id = ? AND status = ?", taskID, "waiting").
		Updates(map[string]interface{}{
			"status":        "cancelled",
			"finished_at":   now,
			"error_message": message,
		}).Error
}

func (r *TimerExecutionRepository) TryMarkRunning(taskID, executionID int64, workerID, executorRunID string, startedAt, leaseUntil time.Time) (bool, error) {
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
		"heartbeat_at":     heartbeatAt,
		"heartbeat_misses": 0,
		"lease_until":      leaseUntil,
	})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TimerExecutionRepository) TryRecordHeartbeatMiss(exec *model.TimerExecution, now, leaseUntil time.Time, heartbeatMisses int) (bool, error) {
	result := r.db.Model(&model.TimerExecution{}).
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
