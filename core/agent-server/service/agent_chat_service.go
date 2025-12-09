package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/llms"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

// AgentChatService 智能体聊天服务
type AgentChatService struct {
	agentRepo          *repository.AgentRepository
	llmRepo            *repository.LLMRepository
	knowledgeRepo      *repository.KnowledgeRepository
	functionGenService *FunctionGenService
	cfg                *config.AgentServerConfig

	// Repository for chat sessions and messages
	sessionRepo     *repository.ChatSessionRepository
	messageRepo     *repository.ChatMessageRepository
	functionGenRepo *repository.FunctionGenRepository
}

// NewAgentChatService 创建智能体聊天服务
func NewAgentChatService(
	agentRepo *repository.AgentRepository,
	llmRepo *repository.LLMRepository,
	knowledgeRepo *repository.KnowledgeRepository,
	natsConn *nats.Conn,
	cfg *config.AgentServerConfig,
) *AgentChatService {
	return &AgentChatService{
		agentRepo:          agentRepo,
		llmRepo:            llmRepo,
		knowledgeRepo:      knowledgeRepo,
		functionGenService: NewFunctionGenService(natsConn, cfg),
		cfg:                cfg,
	}
}

// SetRepositories 设置会话和消息相关的 Repository（延迟初始化）
func (s *AgentChatService) SetRepositories(
	sessionRepo *repository.ChatSessionRepository,
	messageRepo *repository.ChatMessageRepository,
	functionGenRepo *repository.FunctionGenRepository,
) {
	s.sessionRepo = sessionRepo
	s.messageRepo = messageRepo
	s.functionGenRepo = functionGenRepo
}

// Chat 智能体聊天
func (s *AgentChatService) Chat(ctx context.Context, agentID int64, messages []llms.Message) (*llms.ChatResponse, error) {
	// 1. 获取智能体信息
	agent, err := s.agentRepo.GetByID(agentID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("智能体不存在")
		}
		return nil, fmt.Errorf("获取智能体失败: %w", err)
	}

	if !agent.Enabled {
		return nil, fmt.Errorf("智能体已禁用")
	}

	// 2. 获取智能体绑定的知识库（如果有）
	// TODO: 后续可以基于知识库构建上下文

	// 3. 获取 LLM 配置（优先使用智能体绑定的 LLM，如果没有则使用默认 LLM）
	var llmConfig *model.LLMConfig
	if agent.LLMConfigID > 0 {
		// 使用智能体绑定的 LLM
		llmConfig, err = s.llmRepo.GetByID(agent.LLMConfigID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("智能体绑定的LLM配置不存在（ID: %d）", agent.LLMConfigID)
			}
			return nil, fmt.Errorf("获取LLM配置失败: %w", err)
		}
	} else {
		// 使用默认 LLM 配置
		llmConfig, err = s.llmRepo.GetDefault()
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("未设置默认LLM配置，请先在LLM管理中设置默认配置，或在智能体中绑定LLM配置")
			}
			return nil, fmt.Errorf("获取LLM配置失败: %w", err)
		}
	}

	// 4. 创建 LLM 客户端（使用自定义配置）
	provider := llms.Provider(llmConfig.Provider)
	options := llms.DefaultClientOptions()
	if llmConfig.Model != "" {
		options = options.WithModel(llmConfig.Model)
	}

	// 设置超时时间
	if llmConfig.Timeout > 0 {
		options = options.WithTimeout(time.Duration(llmConfig.Timeout) * time.Second)
	}

	// 设置自定义 BaseURL（如果提供了）
	if llmConfig.APIBase != "" {
		options = options.WithBaseURL(llmConfig.APIBase)
	}

	// 设置模型名称（如果提供了）
	if llmConfig.Model != "" {
		options = options.WithModel(llmConfig.Model)
	}

	// 创建客户端
	client, err := llms.NewLLMClientWithOptions(provider, llmConfig.APIKey, options)
	if err != nil {
		return nil, fmt.Errorf("创建LLM客户端失败: %w", err)
	}

	// 6. 解析额外配置（如果有）
	var extraConfig map[string]interface{}
	if llmConfig.ExtraConfig != nil && *llmConfig.ExtraConfig != "" {
		if err := json.Unmarshal([]byte(*llmConfig.ExtraConfig), &extraConfig); err != nil {
			// 忽略解析错误，使用默认配置
		}
	}

	// 7. 构建聊天请求
	chatReq := &llms.ChatRequest{
		Messages: messages,
		Model:    llmConfig.Model,
	}

	// 8. 应用额外配置
	if maxTokens, ok := extraConfig["max_tokens"].(float64); ok && maxTokens > 0 {
		chatReq.MaxTokens = int(maxTokens)
	} else if llmConfig.MaxTokens > 0 {
		chatReq.MaxTokens = llmConfig.MaxTokens
	}

	if temperature, ok := extraConfig["temperature"].(float64); ok {
		chatReq.Temperature = temperature
	}

	// 9. 调用 LLM
	resp, err := client.Chat(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("调用LLM失败: %w", err)
	}

	return resp, nil
}
