package repository

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/timer-scheduler/model"
	"gorm.io/gorm"
)

type TimerTaskRepository struct {
	db *gorm.DB
}

func NewTimerTaskRepository(db *gorm.DB) *TimerTaskRepository {
	return &TimerTaskRepository{db: db}
}

func (r *TimerTaskRepository) DB() *gorm.DB {
	return r.db
}

func (r *TimerTaskRepository) WithDB(db *gorm.DB) *TimerTaskRepository {
	return &TimerTaskRepository{db: db}
}

func (r *TimerTaskRepository) Create(task *model.TimerTask) error {
	return r.db.Create(task).Error
}

func (r *TimerTaskRepository) GetByID(id int64) (*model.TimerTask, error) {
	var task model.TimerTask
	if err := r.db.Where("id = ?", id).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *TimerTaskRepository) List(req ListTasksFilter) ([]*model.TimerTask, int64, error) {
	query := r.db.Model(&model.TimerTask{})
	if req.ExecutorKey != "" {
		query = query.Where("executor_key = ?", req.ExecutorKey)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.Category != "" {
		query = query.Where("category = ?", req.Category)
	}
	if req.SourceType != "" {
		query = query.Where("source_type = ?", req.SourceType)
	}
	if req.SourceRef != "" {
		query = query.Where("source_ref = ?", req.SourceRef)
	}
	if req.CreatedBy != "" {
		query = query.Where("created_by = ?", req.CreatedBy)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if req.Limit > 0 {
		query = query.Limit(req.Limit)
	}
	if req.Offset > 0 {
		query = query.Offset(req.Offset)
	}

	var list []*model.TimerTask
	if err := query.Order("created_at DESC, id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

type ListTasksFilter struct {
	ExecutorKey string
	Status      string
	Category    string
	SourceType  string
	SourceRef   string
	CreatedBy   string
	Offset      int
	Limit       int
}

func (r *TimerTaskRepository) Update(task *model.TimerTask) error {
	return r.db.Save(task).Error
}

func (r *TimerTaskRepository) ListDue(now time.Time, limit int) ([]*model.TimerTask, error) {
	query := r.db.
		Where("status = ? AND next_run_at IS NOT NULL AND next_run_at <= ? AND inflight_execution_id = 0 AND (lease_until IS NULL OR lease_until < ?)", "pending", now, now).
		Order("next_run_at ASC, id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	var list []*model.TimerTask
	if err := query.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *TimerTaskRepository) TryAcquireDispatch(id int64, owner string, now, leaseUntil time.Time) (bool, error) {
	result := r.db.Model(&model.TimerTask{}).
		Where("id = ? AND status = ? AND next_run_at IS NOT NULL AND next_run_at <= ? AND inflight_execution_id = 0 AND (lease_until IS NULL OR lease_until < ?)", id, "pending", now, now).
		Updates(map[string]interface{}{
			"lease_owner": owner,
			"lease_until": leaseUntil,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TimerTaskRepository) TrySetInflight(taskID, executionID int64, leaseOwner string) (bool, error) {
	result := r.db.Model(&model.TimerTask{}).
		Where("id = ? AND status = ? AND inflight_execution_id = 0 AND lease_owner = ?", taskID, "pending", leaseOwner).
		Updates(map[string]interface{}{
			"inflight_execution_id": executionID,
			"last_execution_id":     executionID,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TimerTaskRepository) TrySetManualInflight(taskID, executionID int64) (bool, error) {
	result := r.db.Model(&model.TimerTask{}).
		Where("id = ? AND status IN ? AND inflight_execution_id = 0", taskID, []string{"pending", "paused"}).
		Updates(map[string]interface{}{
			"inflight_execution_id": executionID,
			"last_execution_id":     executionID,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TimerTaskRepository) TryCompleteExecution(task *model.TimerTask, executionID int64) (bool, error) {
	result := r.db.Model(&model.TimerTask{}).
		Where("id = ? AND inflight_execution_id = ?", task.ID, executionID).
		Updates(map[string]interface{}{
			"status":                task.Status,
			"next_run_at":           task.NextRunAt,
			"run_count":             task.RunCount,
			"last_execution_id":     executionID,
			"last_error_message":    task.LastErrorMessage,
			"inflight_execution_id": 0,
			"lease_owner":           "",
			"lease_until":           nil,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *TimerTaskRepository) Pause(id int64) error {
	return r.db.Model(&model.TimerTask{}).
		Where("id = ? AND status = ?", id, "pending").
		Update("status", "paused").Error
}

func (r *TimerTaskRepository) Resume(id int64, nextRunAt *time.Time) error {
	return r.db.Model(&model.TimerTask{}).
		Where("id = ? AND status = ?", id, "paused").
		Updates(map[string]interface{}{
			"status":      "pending",
			"next_run_at": nextRunAt,
		}).Error
}

func (r *TimerTaskRepository) Cancel(id int64) error {
	return r.db.Model(&model.TimerTask{}).
		Where("id = ? AND status IN ?", id, []string{"pending", "paused"}).
		Updates(map[string]interface{}{
			"status":                "cancelled",
			"next_run_at":           nil,
			"inflight_execution_id": 0,
			"lease_owner":           "",
			"lease_until":           nil,
		}).Error
}
