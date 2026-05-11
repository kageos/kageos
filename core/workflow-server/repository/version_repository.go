package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/workflow-server/model"
	"gorm.io/gorm"
)

type WorkflowVersionRepository struct {
	db *gorm.DB
}

func NewWorkflowVersionRepository(db *gorm.DB) *WorkflowVersionRepository {
	return &WorkflowVersionRepository{db: db}
}

func (r *WorkflowVersionRepository) WithDB(db *gorm.DB) *WorkflowVersionRepository {
	return &WorkflowVersionRepository{db: db}
}

func (r *WorkflowVersionRepository) Create(item *model.WorkflowDefinitionVersion) error {
	return r.db.Create(item).Error
}

func (r *WorkflowVersionRepository) GetByID(id int64) (*model.WorkflowDefinitionVersion, error) {
	var item model.WorkflowDefinitionVersion
	if err := r.db.Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *WorkflowVersionRepository) GetLatestPublished(workflowID int64) (*model.WorkflowDefinitionVersion, error) {
	var item model.WorkflowDefinitionVersion
	if err := r.db.
		Where("workflow_id = ? AND status = ?", workflowID, model.VersionStatusPublished).
		Order("version DESC, id DESC").
		First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *WorkflowVersionRepository) NextVersion(workflowID int64) (int, error) {
	var maxVersion int
	if err := r.db.Model(&model.WorkflowDefinitionVersion{}).
		Where("workflow_id = ?", workflowID).
		Select("COALESCE(MAX(version), 0)").
		Scan(&maxVersion).Error; err != nil {
		return 0, err
	}
	return maxVersion + 1, nil
}
