package repository

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/workflow-server/model"
	"gorm.io/gorm"
)

type WorkflowRunRepository struct {
	db *gorm.DB
}

func NewWorkflowRunRepository(db *gorm.DB) *WorkflowRunRepository {
	return &WorkflowRunRepository{db: db}
}

func (r *WorkflowRunRepository) WithDB(db *gorm.DB) *WorkflowRunRepository {
	return &WorkflowRunRepository{db: db}
}

func (r *WorkflowRunRepository) Create(item *model.WorkflowRun) error {
	return r.db.Create(item).Error
}

func (r *WorkflowRunRepository) GetByID(id int64) (*model.WorkflowRun, error) {
	var item model.WorkflowRun
	if err := r.db.Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *WorkflowRunRepository) Finish(runID int64, status string, output []byte, errMsg string, finishedAt time.Time, durationMillis int64) error {
	return r.db.Model(&model.WorkflowRun{}).
		Where("id = ?", runID).
		Updates(map[string]interface{}{
			"status":          status,
			"output":          output,
			"error_message":   errMsg,
			"finished_at":     finishedAt,
			"duration_millis": durationMillis,
		}).Error
}

func (r *WorkflowRunRepository) Cancel(runID int64) error {
	return r.db.Model(&model.WorkflowRun{}).
		Where("id = ? AND status IN ?", runID, []string{model.RunStatusPending, model.RunStatusRunning, model.RunStatusWaiting}).
		Update("status", model.RunStatusCancelled).Error
}

type WorkflowStepRunRepository struct {
	db *gorm.DB
}

func NewWorkflowStepRunRepository(db *gorm.DB) *WorkflowStepRunRepository {
	return &WorkflowStepRunRepository{db: db}
}

func (r *WorkflowStepRunRepository) WithDB(db *gorm.DB) *WorkflowStepRunRepository {
	return &WorkflowStepRunRepository{db: db}
}

func (r *WorkflowStepRunRepository) Create(item *model.WorkflowStepRun) error {
	return r.db.Create(item).Error
}

func (r *WorkflowStepRunRepository) ListByRunID(runID int64) ([]*model.WorkflowStepRun, error) {
	var list []*model.WorkflowStepRun
	if err := r.db.Where("run_id = ?", runID).Order("id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *WorkflowStepRunRepository) Finish(stepRunID int64, status string, output []byte, errMsg string, finishedAt time.Time, durationMillis int64) error {
	return r.db.Model(&model.WorkflowStepRun{}).
		Where("id = ?", stepRunID).
		Updates(map[string]interface{}{
			"status":          status,
			"output":          output,
			"error_message":   errMsg,
			"finished_at":     finishedAt,
			"duration_millis": durationMillis,
		}).Error
}
