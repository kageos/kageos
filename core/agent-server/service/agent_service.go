package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
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
	repo            *repository.AgentRepository
	knowledgeRepo   *repository.KnowledgeRepository
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
func (s *AgentService) ListAgents(ctx context.Context, agentType string, enabled *bool, page, pageSize int) ([]*model.Agent, int64, error) {
	offset := (page - 1) * pageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	return s.repo.List(agentType, enabled, offset, pageSize)
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

	// 创建智能体（AfterCreate 钩子会自动生成 NATS 主题）
	err = s.repo.Create(agent)
	if err != nil {
		return err
	}

	// 如果创建成功且是插件调用类型，确保消息主题已生成
	// （AfterCreate 钩子应该已经处理，这里作为兜底）
	if agent.AgentType == "plugin" && agent.MsgSubject == "" {
		// 使用 subjects 包统一生成消息主题
		agent.MsgSubject = subjects.BuildAgentMsgSubject(agent.ChatType, agent.CreatedBy, agent.ID)
		return s.repo.Update(agent)
	}

	return nil
}

// UpdateAgent 更新智能体
func (s *AgentService) UpdateAgent(ctx context.Context, agent *model.Agent) error {
	// 获取用户信息
	user := contextx.GetRequestUser(ctx)
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

	// 如果是插件调用类型且消息主题为空，自动生成
	if agent.AgentType == "plugin" && agent.MsgSubject == "" {
		// 使用 subjects 包统一生成消息主题
		agent.MsgSubject = subjects.BuildAgentMsgSubject(agent.ChatType, agent.CreatedBy, agent.ID)
	} else if agent.AgentType == "plugin" && agent.MsgSubject != "" {
		// 如果 chat_type 或创建用户发生变化，重新生成消息主题
		expectedSubject := subjects.BuildAgentMsgSubject(agent.ChatType, agent.CreatedBy, agent.ID)
		if agent.MsgSubject != expectedSubject {
			agent.MsgSubject = expectedSubject
		}
	} else if agent.AgentType != "plugin" {
		// 如果不是插件调用类型，清空消息主题
		agent.MsgSubject = ""
	}

	return s.repo.Update(agent)
}

// DeleteAgent 删除智能体
func (s *AgentService) DeleteAgent(ctx context.Context, id int64) error {
	return s.repo.Delete(id)
}

// EnableAgent 启用智能体
func (s *AgentService) EnableAgent(ctx context.Context, id int64) error {
	return s.repo.Enable(id)
}

// DisableAgent 禁用智能体
func (s *AgentService) DisableAgent(ctx context.Context, id int64) error {
	return s.repo.Disable(id)
}

