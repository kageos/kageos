package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"gorm.io/gorm"
)

// FunctionGroupAgentRepository 函数组和智能体关联数据访问层
type FunctionGroupAgentRepository struct {
	db *gorm.DB
}

// NewFunctionGroupAgentRepository 创建函数组和智能体关联 Repository
func NewFunctionGroupAgentRepository(db *gorm.DB) *FunctionGroupAgentRepository {
	return &FunctionGroupAgentRepository{db: db}
}

// Create 创建关联记录
func (r *FunctionGroupAgentRepository) Create(fga *model.FunctionGroupAgent) error {
	return r.db.Create(fga).Error
}

// GetByFullGroupCode 根据 FullGroupCode 获取关联记录
func (r *FunctionGroupAgentRepository) GetByFullGroupCode(fullGroupCode string) (*model.FunctionGroupAgent, error) {
	var fga model.FunctionGroupAgent
	if err := r.db.
		Preload("Agent").
		Preload("Record").
		Where("full_group_code = ?", fullGroupCode).
		Order("created_at DESC").
		First(&fga).Error; err != nil {
		return nil, err
	}
	return &fga, nil
}

// ListByAgentID 根据 AgentID 获取关联记录列表
func (r *FunctionGroupAgentRepository) ListByAgentID(agentID int64, offset, limit int) ([]*model.FunctionGroupAgent, int64, error) {
	var fgas []*model.FunctionGroupAgent
	var total int64

	query := r.db.Model(&model.FunctionGroupAgent{}).Where("agent_id = ?", agentID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	if err := query.
		Preload("Agent").
		Preload("Record").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&fgas).Error; err != nil {
		return nil, 0, err
	}

	return fgas, total, nil
}

// ListByRecordID 根据 RecordID 获取关联记录列表
func (r *FunctionGroupAgentRepository) ListByRecordID(recordID int64) ([]*model.FunctionGroupAgent, error) {
	var fgas []*model.FunctionGroupAgent
	if err := r.db.
		Preload("Agent").
		Where("record_id = ?", recordID).
		Find(&fgas).Error; err != nil {
		return nil, err
	}
	return fgas, nil
}

// ListByAppCode 根据 AppCode 获取关联记录列表
func (r *FunctionGroupAgentRepository) ListByAppCode(appCode string, offset, limit int) ([]*model.FunctionGroupAgent, int64, error) {
	var fgas []*model.FunctionGroupAgent
	var total int64

	query := r.db.Model(&model.FunctionGroupAgent{}).Where("app_code = ?", appCode)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	if err := query.
		Preload("Agent").
		Preload("Record").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&fgas).Error; err != nil {
		return nil, 0, err
	}

	return fgas, total, nil
}

