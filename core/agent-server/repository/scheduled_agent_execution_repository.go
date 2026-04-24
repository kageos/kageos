package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"gorm.io/gorm"
)

type ScheduledAgentExecutionRepository struct {
	db *gorm.DB
}

func NewScheduledAgentExecutionRepository(db *gorm.DB) *ScheduledAgentExecutionRepository {
	return &ScheduledAgentExecutionRepository{db: db}
}

func (r *ScheduledAgentExecutionRepository) Create(exec *model.ScheduledAgentExecution) error {
	return r.db.Create(exec).Error
}

func (r *ScheduledAgentExecutionRepository) Update(exec *model.ScheduledAgentExecution) error {
	return r.db.Save(exec).Error
}

func (r *ScheduledAgentExecutionRepository) GetByID(taskID, id int64) (*model.ScheduledAgentExecution, error) {
	var exec model.ScheduledAgentExecution
	if err := r.db.Where("task_id = ? AND id = ?", taskID, id).First(&exec).Error; err != nil {
		return nil, err
	}
	return &exec, nil
}

func (r *ScheduledAgentExecutionRepository) ListByTaskID(taskID int64, status string, offset, limit int) ([]*model.ScheduledAgentExecution, int64, error) {
	var list []*model.ScheduledAgentExecution
	query := r.db.Model(&model.ScheduledAgentExecution{}).Where("task_id = ?", taskID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 20
	}
	if err := query.Offset(offset).Limit(limit).Order("scheduled_at DESC, id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
