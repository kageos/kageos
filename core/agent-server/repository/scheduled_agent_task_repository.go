package repository

import (
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"gorm.io/gorm"
)

type ScheduledAgentTaskRepository struct {
	db *gorm.DB
}

func NewScheduledAgentTaskRepository(db *gorm.DB) *ScheduledAgentTaskRepository {
	return &ScheduledAgentTaskRepository{db: db}
}

func (r *ScheduledAgentTaskRepository) Create(task *model.ScheduledAgentTask) error {
	return r.db.Create(task).Error
}

func (r *ScheduledAgentTaskRepository) GetByID(id int64) (*model.ScheduledAgentTask, error) {
	var task model.ScheduledAgentTask
	if err := r.db.Where("id = ?", id).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *ScheduledAgentTaskRepository) Update(task *model.ScheduledAgentTask) error {
	return r.db.Save(task).Error
}

func (r *ScheduledAgentTaskRepository) List(createdBy, status, fullCodePath string, offset, limit int) ([]*model.ScheduledAgentTask, int64, error) {
	var list []*model.ScheduledAgentTask
	query := r.db.Model(&model.ScheduledAgentTask{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if fullCodePath != "" {
		prefix := strings.TrimSuffix(strings.TrimSpace(fullCodePath), "/")
		query = query.Where("full_code_path = ? OR full_code_path LIKE ?", prefix, prefix+"/%")
	} else if createdBy != "" {
		query = query.Where("created_by = ?", createdBy)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 20
	}
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *ScheduledAgentTaskRepository) ListPendingDue(now time.Time, limit int) ([]*model.ScheduledAgentTask, error) {
	var list []*model.ScheduledAgentTask
	query := r.db.
		Where("status = ? AND next_run_at IS NOT NULL AND next_run_at <= ? AND (lease_until IS NULL OR lease_until < ?)", model.ScheduledAgentTaskStatusPending, now, now).
		Order("next_run_at ASC, id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *ScheduledAgentTaskRepository) TryAcquireLease(id int64, owner string, now, leaseUntil time.Time) (bool, error) {
	result := r.db.Model(&model.ScheduledAgentTask{}).
		Where("id = ? AND status = ? AND next_run_at IS NOT NULL AND next_run_at <= ? AND (lease_until IS NULL OR lease_until < ?)", id, model.ScheduledAgentTaskStatusPending, now, now).
		Updates(map[string]interface{}{
			"lease_owner": owner,
			"lease_until": leaseUntil,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *ScheduledAgentTaskRepository) UpdateAfterScheduledRun(task *model.ScheduledAgentTask, leaseOwner string) (bool, error) {
	result := r.db.Model(&model.ScheduledAgentTask{}).
		Where("id = ? AND lease_owner = ?", task.ID, leaseOwner).
		Updates(map[string]interface{}{
			"run_count":          task.RunCount,
			"status":             task.Status,
			"next_run_at":        task.NextRunAt,
			"last_session_id":    task.LastSessionID,
			"last_execution_id":  task.LastExecutionID,
			"last_error_message": task.LastErrorMessage,
			"lease_owner":        "",
			"lease_until":        nil,
			"updated_by":         task.UpdatedBy,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *ScheduledAgentTaskRepository) UpdateAfterManualRun(task *model.ScheduledAgentTask) error {
	return r.db.Model(&model.ScheduledAgentTask{}).
		Where("id = ?", task.ID).
		Updates(map[string]interface{}{
			"last_session_id":    task.LastSessionID,
			"last_execution_id":  task.LastExecutionID,
			"last_error_message": task.LastErrorMessage,
			"updated_by":         task.UpdatedBy,
		}).Error
}

func (r *ScheduledAgentTaskRepository) Pause(id int64, updatedBy string) error {
	return r.db.Model(&model.ScheduledAgentTask{}).
		Where("id = ? AND status = ?", id, model.ScheduledAgentTaskStatusPending).
		Updates(map[string]interface{}{
			"status":      model.ScheduledAgentTaskStatusPaused,
			"lease_owner": "",
			"lease_until": nil,
			"updated_by":  updatedBy,
		}).Error
}

func (r *ScheduledAgentTaskRepository) Resume(id int64, nextRunAt *time.Time, updatedBy string) error {
	return r.db.Model(&model.ScheduledAgentTask{}).
		Where("id = ? AND status = ?", id, model.ScheduledAgentTaskStatusPaused).
		Updates(map[string]interface{}{
			"status":      model.ScheduledAgentTaskStatusPending,
			"next_run_at": nextRunAt,
			"updated_by":  updatedBy,
		}).Error
}

func (r *ScheduledAgentTaskRepository) Cancel(id int64, updatedBy string) error {
	return r.db.Model(&model.ScheduledAgentTask{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      model.ScheduledAgentTaskStatusCancelled,
			"next_run_at": nil,
			"lease_owner": "",
			"lease_until": nil,
			"updated_by":  updatedBy,
		}).Error
}
