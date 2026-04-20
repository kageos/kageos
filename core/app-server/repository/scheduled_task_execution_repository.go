package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"gorm.io/gorm"
)

// ScheduledTaskExecutionRepository 定时任务执行记录仓储
type ScheduledTaskExecutionRepository struct {
	db *gorm.DB
}

func NewScheduledTaskExecutionRepository(db *gorm.DB) *ScheduledTaskExecutionRepository {
	return &ScheduledTaskExecutionRepository{db: db}
}

func (r *ScheduledTaskExecutionRepository) Create(exec *model.ScheduledTaskExecution) error {
	return r.db.Create(exec).Error
}

func (r *ScheduledTaskExecutionRepository) GetByID(taskID int64, id int64) (*model.ScheduledTaskExecution, error) {
	var exec model.ScheduledTaskExecution
	if err := r.db.Where("task_id = ? AND id = ?", taskID, id).First(&exec).Error; err != nil {
		return nil, err
	}
	return &exec, nil
}

// ListByTaskID 某任务的执行记录列表，按 executed_at 倒序，分页
func (r *ScheduledTaskExecutionRepository) ListByTaskID(taskID int64, status string, offset, limit int) ([]*model.ScheduledTaskExecution, int64, error) {
	var list []*model.ScheduledTaskExecution
	query := r.db.Model(&model.ScheduledTaskExecution{}).Where("task_id = ?", taskID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Offset(offset).Limit(limit).Order("executed_at DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
