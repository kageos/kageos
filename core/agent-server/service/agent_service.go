package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/utils"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"gorm.io/gorm"
)

// normalizeMetadata 规范化 metadata 字段，确保是有效的 JSON 或 NULL
func normalizeMetadata(metadata string) (*string, error) {
	// 如果为空，返回 nil（允许 NULL）
	if metadata == "" {
		return nil, nil
	}

	// 验证是否为有效的 JSON
	var temp interface{}
	if err := json.Unmarshal([]byte(metadata), &temp); err != nil {
		return nil, fmt.Errorf("metadata 不是有效的 JSON: %w", err)
	}

	// 重新序列化以确保格式正确
	normalized, err := json.Marshal(temp)
	if err != nil {
		return nil, fmt.Errorf("序列化 metadata 失败: %w", err)
	}

	result := string(normalized)
	return &result, nil
}

// AgentService 智能体服务
type AgentService struct {
	repo          *repository.AgentRepository
	knowledgeRepo *repository.KnowledgeRepository
}

// NewAgentService 创建智能体服务
func NewAgentService(repo *repository.AgentRepository, knowledgeRepo *repository.KnowledgeRepository) *AgentService {
	return &AgentService{
		repo:          repo,
		knowledgeRepo: knowledgeRepo,
	}
}

// GetAgent 获取智能体
func (s *AgentService) GetAgent(ctx context.Context, id int64) (*model.Agent, error) {
	agent, err := s.repo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("智能体不存在")
		}
		return nil, fmt.Errorf("获取智能体失败: %w", err)
	}
	return agent, nil
}

// ListAgents 获取智能体列表
func (s *AgentService) ListAgents(ctx context.Context, req dto.AgentListReq) ([]*model.Agent, int64, error) {
	currentUser := contextx.GetRequestUser(ctx)
	return s.repo.List(req, currentUser)
}

// CreateAgent 创建智能体
func (s *AgentService) CreateAgent(ctx context.Context, agent *model.Agent) error {
	// 获取用户信息
	user := contextx.GetRequestUser(ctx)
	agent.CreatedBy = user
	agent.UpdatedBy = user

	// 验证必填字段
	if agent.Name == "" {
		return fmt.Errorf("智能体名称不能为空")
	}
	if agent.AgentType == "" {
		return fmt.Errorf("智能体类型不能为空")
	}
	if agent.ChatType == "" {
		return fmt.Errorf("聊天类型不能为空")
	}
	if agent.KnowledgeBaseID == 0 {
		return fmt.Errorf("知识库ID不能为空")
	}

	// 验证知识库是否存在
	_, err := s.knowledgeRepo.GetByID(agent.KnowledgeBaseID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("知识库不存在")
		}
		return fmt.Errorf("验证知识库失败: %w", err)
	}

	// 如果是 plugin 类型，验证 PluginFunctionPath
	if agent.AgentType == "plugin" {
		if agent.PluginFunctionPath == "" {
			return fmt.Errorf("插件类型智能体必须指定插件函数路径")
		}
		// ⭐ 可以在这里验证 PluginFunctionPath 是否存在（可选，需要调用 app-server 的 API）
	} else {
		// 非 plugin 类型，清空 PluginFunctionPath
		agent.PluginFunctionPath = ""
	}

	// 规范化 metadata 字段
	metadataStr := ""
	if agent.Metadata != nil {
		metadataStr = *agent.Metadata
	}
	normalizedMetadata, err := normalizeMetadata(metadataStr)
	if err != nil {
		return err
	}
	agent.Metadata = normalizedMetadata

	// 设置默认管理员（如果为空，设置为创建用户）
	if agent.Admin == "" {
		agent.Admin = user
	}

	// 设置默认可见性（如果未设置，默认为公开）
	// Visibility 字段在模型中已有默认值 0，这里确保设置

	// 创建智能体
	// 注意：新架构中，plugin 类型的智能体使用 Plugin.Subject，不再需要 MsgSubject
	err = s.repo.Create(agent)
	if err != nil {
		return err
	}

	return nil
}

// UpdateAgent 更新智能体
func (s *AgentService) UpdateAgent(ctx context.Context, agent *model.Agent) error {
	// 获取用户信息
	user := contextx.GetRequestUser(ctx)
	agent.UpdatedBy = user

	// 检查权限：只有管理员可以修改资源
	existing, err := s.repo.GetByID(agent.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("智能体不存在")
		}
		return fmt.Errorf("获取智能体失败: %w", err)
	}

	if !utils.IsAdmin(existing.Admin, user) {
		return fmt.Errorf("无权限：只有管理员可以修改此资源")
	}

	// 验证必填字段
	if agent.Name == "" {
		return fmt.Errorf("智能体名称不能为空")
	}
	if agent.AgentType == "" {
		return fmt.Errorf("智能体类型不能为空")
	}
	if agent.ChatType == "" {
		return fmt.Errorf("聊天类型不能为空")
	}
	if agent.KnowledgeBaseID == 0 {
		return fmt.Errorf("知识库ID不能为空")
	}

	// 验证知识库是否存在
	_, err = s.knowledgeRepo.GetByID(agent.KnowledgeBaseID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("知识库不存在")
		}
		return fmt.Errorf("验证知识库失败: %w", err)
	}

	// 如果是 plugin 类型，验证 PluginFunctionPath
	if agent.AgentType == "plugin" {
		if agent.PluginFunctionPath == "" {
			return fmt.Errorf("插件类型智能体必须指定插件函数路径")
		}
		// ⭐ 可以在这里验证 PluginFunctionPath 是否存在（可选，需要调用 app-server 的 API）
	} else {
		// 非 plugin 类型，清空 PluginFunctionPath
		agent.PluginFunctionPath = ""
	}

	// 规范化 metadata 字段
	metadataStr := ""
	if agent.Metadata != nil {
		metadataStr = *agent.Metadata
	}
	normalizedMetadata, err := normalizeMetadata(metadataStr)
	if err != nil {
		return err
	}
	agent.Metadata = normalizedMetadata

	return s.repo.Update(agent)
}

// DeleteAgent 删除智能体
func (s *AgentService) DeleteAgent(ctx context.Context, id int64) error {
	// 检查权限：只有管理员可以删除资源
	user := contextx.GetRequestUser(ctx)
	existing, err := s.repo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("智能体不存在")
		}
		return fmt.Errorf("获取智能体失败: %w", err)
	}

	if !utils.IsAdmin(existing.Admin, user) {
		return fmt.Errorf("无权限：只有管理员可以删除此资源")
	}
	return s.repo.Delete(id)
}

// EnableAgent 启用智能体
func (s *AgentService) EnableAgent(ctx context.Context, id int64) error {
	// 检查权限：只有管理员可以启用资源
	user := contextx.GetRequestUser(ctx)
	existing, err := s.repo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("智能体不存在")
		}
		return fmt.Errorf("获取智能体失败: %w", err)
	}

	if !utils.IsAdmin(existing.Admin, user) {
		return fmt.Errorf("无权限：只有管理员可以启用此资源")
	}
	return s.repo.Enable(id)
}

// DisableAgent 禁用智能体
func (s *AgentService) DisableAgent(ctx context.Context, id int64) error {
	// 检查权限：只有管理员可以禁用资源
	user := contextx.GetRequestUser(ctx)
	existing, err := s.repo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("智能体不存在")
		}
		return fmt.Errorf("获取智能体失败: %w", err)
	}

	if !utils.IsAdmin(existing.Admin, user) {
		return fmt.Errorf("无权限：只有管理员可以禁用此资源")
	}
	return s.repo.Disable(id)
}
