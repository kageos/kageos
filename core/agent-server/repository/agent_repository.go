package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
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
func (r *AgentRepository) List(req dto.AgentListReq, currentUser string) ([]*model.Agent, int64, error) {
	var agents []*model.Agent
	var total int64

	offset := (req.Page - 1) * req.PageSize
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	dbQuery := r.db.Model(&model.Agent{})

	// 权限过滤：根据 scope 参数
	if req.Scope == "mine" {
		// 我的：显示当前用户是管理员的资源（公开+私有）
		// 使用 FIND_IN_SET 函数（MySQL）
		dbQuery = dbQuery.Where("FIND_IN_SET(?, admin) > 0", currentUser)
	} else if req.Scope == "market" {
		// 市场：显示所有公开的资源
		dbQuery = dbQuery.Where("visibility = ?", 0)
	}
	// 默认：显示所有（向后兼容，或根据需求调整）

	if req.AgentType != "" {
		dbQuery = dbQuery.Where("agent_type = ?", req.AgentType)
	}

	if req.Enabled != nil {
		dbQuery = dbQuery.Where("enabled = ?", *req.Enabled)
	}

	if req.KnowledgeBaseID != nil && *req.KnowledgeBaseID > 0 {
		dbQuery = dbQuery.Where("knowledge_base_id = ?", *req.KnowledgeBaseID)
	}

	if req.LLMConfigID != nil {
		// 如果 llmConfigID 为 0，表示查询使用默认LLM的智能体
		dbQuery = dbQuery.Where("llm_config_id = ?", *req.LLMConfigID)
	}

	if req.PluginID != nil && *req.PluginID > 0 {
		dbQuery = dbQuery.Where("plugin_id = ?", *req.PluginID)
	}

	// 获取总数
	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表（预加载关联数据）
	if err := dbQuery.
		Preload("Plugin").
		Preload("KnowledgeBase").
		Preload("LLMConfig").
		Offset(offset).
		Limit(req.PageSize).
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

// IncrementGenerationCount 增加智能体的生成次数（原子操作）
func (r *AgentRepository) IncrementGenerationCount(agentID int64) error {
	return r.db.Model(&model.Agent{}).
		Where("id = ?", agentID).
		Update("generation_count", gorm.Expr("generation_count + ?", 1)).Error
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

