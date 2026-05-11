package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/workflow-server/model"
	"gorm.io/gorm"
)

type WorkflowDefinitionRepository struct {
	db *gorm.DB
}

func NewWorkflowDefinitionRepository(db *gorm.DB) *WorkflowDefinitionRepository {
	return &WorkflowDefinitionRepository{db: db}
}

func (r *WorkflowDefinitionRepository) WithDB(db *gorm.DB) *WorkflowDefinitionRepository {
	return &WorkflowDefinitionRepository{db: db}
}

func (r *WorkflowDefinitionRepository) Create(item *model.WorkflowDefinition) error {
	return r.db.Create(item).Error
}

func (r *WorkflowDefinitionRepository) Update(item *model.WorkflowDefinition) error {
	return r.db.Save(item).Error
}

func (r *WorkflowDefinitionRepository) GetByID(id int64) (*model.WorkflowDefinition, error) {
	var item model.WorkflowDefinition
	if err := r.db.Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *WorkflowDefinitionRepository) List(req ListWorkflowDefinitionsFilter) ([]*model.WorkflowDefinition, int64, error) {
	query := r.db.Model(&model.WorkflowDefinition{})
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.AppID > 0 {
		query = query.Where("app_id = ?", req.AppID)
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
	var list []*model.WorkflowDefinition
	if err := query.Order("updated_at DESC, id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

type ListWorkflowDefinitionsFilter struct {
	Status    string
	AppID     int64
	CreatedBy string
	Offset    int
	Limit     int
}

func (r *WorkflowDefinitionRepository) SetLatestVersion(id, versionID int64, status string) error {
	return r.db.Model(&model.WorkflowDefinition{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"latest_version_id": versionID,
			"status":            status,
		}).Error
}
