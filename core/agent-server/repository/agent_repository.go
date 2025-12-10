package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"gorm.io/gorm"
)

// AgentRepository 智能体数据访问层
type AgentRepository struct {
	db *gorm.DB
}

// NewAgentRepository 创建智能体 Repository
func NewAgentRepository(db *gorm.DB) *AgentRepository {
	return &AgentRepository{db: db}
}

// Create 创建智能体
func (r *AgentRepository) Create(agent *model.Agent) error {
	return r.db.Create(agent).Error
}

// GetByID 根据 ID 获取智能体（预加载关联数据）
func (r *AgentRepository) GetByID(id int64) (*model.Agent, error) {
	var agent model.Agent
	if err := r.db.
		Preload("Plugin").
		Preload("KnowledgeBase").
		Preload("LLMConfig").
		Where("id = ?", id).
		First(&agent).Error; err != nil {
		return nil, err
	}
	return &agent, nil
}

// List 获取智能体列表
func (r *AgentRepository) List(agentType string, enabled *bool, offset, limit int) ([]*model.Agent, int64, error) {
	var agents []*model.Agent
	var total int64

	query := r.db.Model(&model.Agent{})

	if agentType != "" {
		query = query.Where("agent_type = ?", agentType)
	}

	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表（预加载关联数据）
	if err := query.
		Preload("Plugin").
		Preload("KnowledgeBase").
		Preload("LLMConfig").
		Offset(offset).
		Limit(limit).
		Order("id DESC").
		Find(&agents).Error; err != nil {
		return nil, 0, err
	}

	return agents, total, nil
}

// Update 更新智能体
func (r *AgentRepository) Update(agent *model.Agent) error {
	return r.db.Save(agent).Error
}

// Delete 删除智能体
func (r *AgentRepository) Delete(id int64) error {
	return r.db.Where("id = ?", id).Delete(&model.Agent{}).Error
}

// Enable 启用智能体
func (r *AgentRepository) Enable(id int64) error {
	return r.db.Model(&model.Agent{}).Where("id = ?", id).Update("enabled", true).Error
}

// Disable 禁用智能体
func (r *AgentRepository) Disable(id int64) error {
	return r.db.Model(&model.Agent{}).Where("id = ?", id).Update("enabled", false).Error
}

