package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/llms"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

// AgentChatService 智能体聊天服务
type AgentChatService struct {
	agentRepo         *repository.AgentRepository
	llmRepo           *repository.LLMRepository
	knowledgeRepo     *repository.KnowledgeRepository
	functionGenService *FunctionGenService
	cfg               *config.AgentServerConfig

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
		agentRepo:         agentRepo,
		llmRepo:           llmRepo,
		knowledgeRepo:     knowledgeRepo,
		functionGenService: NewFunctionGenService(natsConn, cfg),
		cfg:               cfg,
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

// FunctionGenChat 函数生成智能体聊天（完整的对话流程）
func (s *AgentChatService) FunctionGenChat(ctx context.Context, req *dto.FunctionGenAgentChatReq) (*dto.FunctionGenAgentChatResp, error) {
	user := contextx.GetRequestUser(ctx)
	traceId := contextx.GetTraceId(ctx)

	logger.Infof(ctx, "[FunctionGenChat] 开始处理 - AgentID: %d, TreeID: %d, SessionID: %s, User: %s, TraceID: %s",
		req.AgentID, req.TreeID, req.SessionID, user, traceId)

	// 1. 获取智能体信息
	logger.Debugf(ctx, "[FunctionGenChat] 获取智能体信息 - AgentID: %d", req.AgentID)
	agent, err := s.agentRepo.GetByID(req.AgentID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Errorf(ctx, "[FunctionGenChat] 智能体不存在 - AgentID: %d, TraceID: %s", req.AgentID, traceId)
			return nil, fmt.Errorf("智能体不存在")
		}
		logger.Errorf(ctx, "[FunctionGenChat] 获取智能体失败 - AgentID: %d, TraceID: %s, Error: %v", req.AgentID, traceId, err)
		return nil, fmt.Errorf("获取智能体失败: %w", err)
	}

	logger.Infof(ctx, "[FunctionGenChat] 智能体信息 - AgentID: %d, Name: %s, Type: %s, ChatType: %s, Enabled: %v, KBID: %d, LLMConfigID: %d",
		agent.ID, agent.Name, agent.AgentType, agent.ChatType, agent.Enabled, agent.KnowledgeBaseID, agent.LLMConfigID)

	if !agent.Enabled {
		logger.Warnf(ctx, "[FunctionGenChat] 智能体已禁用 - AgentID: %d, TraceID: %s", req.AgentID, traceId)
		return nil, fmt.Errorf("智能体已禁用")
	}

	// 2. 会话管理：创建或获取会话
	var sessionID string
	if req.SessionID == "" {
		// 创建新会话
		sessionID = uuid.New().String()
		logger.Infof(ctx, "[FunctionGenChat] 创建新会话 - SessionID: %s, TreeID: %d, AgentID: %d, TraceID: %s",
			sessionID, req.TreeID, req.AgentID, traceId)
		session := &model.AgentChatSession{
			TreeID:    req.TreeID,
			AgentID:   req.AgentID,
			SessionID: sessionID,
			Title:     "", // 可以后续根据第一条消息自动生成
			User:      user,
		}
		session.CreatedBy = user
		session.UpdatedBy = user
		if err := s.sessionRepo.Create(session); err != nil {
			logger.Errorf(ctx, "[FunctionGenChat] 创建会话失败 - SessionID: %s, TraceID: %s, Error: %v", sessionID, traceId, err)
			return nil, fmt.Errorf("创建会话失败: %w", err)
		}
		logger.Infof(ctx, "[FunctionGenChat] 会话创建成功 - SessionID: %s, TraceID: %s", sessionID, traceId)
	} else {
		// 使用现有会话
		sessionID = req.SessionID
		logger.Debugf(ctx, "[FunctionGenChat] 使用现有会话 - SessionID: %s, TraceID: %s", sessionID, traceId)
		_, err := s.sessionRepo.GetBySessionID(sessionID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				logger.Errorf(ctx, "[FunctionGenChat] 会话不存在 - SessionID: %s, TraceID: %s", sessionID, traceId)
				return nil, fmt.Errorf("会话不存在")
			}
			logger.Errorf(ctx, "[FunctionGenChat] 获取会话失败 - SessionID: %s, TraceID: %s, Error: %v", sessionID, traceId, err)
			return nil, fmt.Errorf("获取会话失败: %w", err)
		}
	}

	// 3. 保存用户消息
	logger.Debugf(ctx, "[FunctionGenChat] 保存用户消息 - SessionID: %s, ContentLength: %d, FilesCount: %d, TraceID: %s",
		sessionID, len(req.Message.Content), len(req.Message.Files), traceId)
	userMessage := &model.AgentChatMessage{
		SessionID: sessionID,
		Role:      "user",
		Content:   req.Message.Content,
		User:      user,
	}
	// 处理文件列表（如果有）
	if len(req.Message.Files) > 0 {
		filesJSON, err := json.Marshal(req.Message.Files)
		if err != nil {
			logger.Errorf(ctx, "[FunctionGenChat] 序列化文件列表失败 - SessionID: %s, TraceID: %s, Error: %v", sessionID, traceId, err)
			return nil, fmt.Errorf("序列化文件列表失败: %w", err)
		}
		filesJSONStr := string(filesJSON)
		userMessage.Files = &filesJSONStr
		logger.Infof(ctx, "[FunctionGenChat] 用户消息包含文件 - SessionID: %s, Files: %s, TraceID: %s",
			sessionID, filesJSONStr, traceId)
	} else {
		// 没有文件时，设置为 nil（数据库会存储为 NULL）
		userMessage.Files = nil
	}
	userMessage.CreatedBy = user
	userMessage.UpdatedBy = user
	if err := s.messageRepo.Create(userMessage); err != nil {
		logger.Errorf(ctx, "[FunctionGenChat] 保存用户消息失败 - SessionID: %s, TraceID: %s, Error: %v", sessionID, traceId, err)
		return nil, fmt.Errorf("保存用户消息失败: %w", err)
	}

	// 4. 加载历史消息
	logger.Debugf(ctx, "[FunctionGenChat] 加载历史消息 - SessionID: %s, TraceID: %s", sessionID, traceId)
	historyMessages, err := s.messageRepo.ListBySessionID(sessionID)
	if err != nil {
		logger.Errorf(ctx, "[FunctionGenChat] 加载历史消息失败 - SessionID: %s, TraceID: %s, Error: %v", sessionID, traceId, err)
		return nil, fmt.Errorf("加载历史消息失败: %w", err)
	}
	logger.Infof(ctx, "[FunctionGenChat] 历史消息数量 - SessionID: %s, Count: %d, TraceID: %s", sessionID, len(historyMessages), traceId)

	// 5. 构建 LLM 消息列表
	llmMessages := make([]llms.Message, 0)

	// 5.1 加载知识库内容（每次请求都加载，确保知识库内容是最新的）
	logger.Infof(ctx, "[FunctionGenChat] 加载知识库 - KBID: %d, TraceID: %s", agent.KnowledgeBaseID, traceId)
	docs, err := s.knowledgeRepo.GetAllDocumentsByKBID(agent.KnowledgeBaseID)
	if err != nil {
		logger.Errorf(ctx, "[FunctionGenChat] 加载知识库文档失败 - KBID: %d, TraceID: %s, Error: %v", agent.KnowledgeBaseID, traceId, err)
		return nil, fmt.Errorf("加载知识库文档失败: %w", err)
	}

	logger.Infof(ctx, "[FunctionGenChat] 知识库文档数量 - KBID: %d, Count: %d, TraceID: %s", agent.KnowledgeBaseID, len(docs), traceId)

	// 构建知识库内容
	var knowledgeContent strings.Builder
	completedCount := 0
	for _, doc := range docs {
		if doc.Status == "completed" {
			knowledgeContent.WriteString(fmt.Sprintf("\n## %s\n%s\n", doc.Title, doc.Content))
			completedCount++
		}
	}
	logger.Infof(ctx, "[FunctionGenChat] 知识库内容构建完成 - KBID: %d, CompletedDocs: %d, ContentLength: %d, TraceID: %s",
		agent.KnowledgeBaseID, completedCount, knowledgeContent.Len(), traceId)

	// 5.2 构建 system message：系统提示词 + 换行 + 知识库内容
	var systemPromptContent strings.Builder
	
	// 获取系统提示词模板
	template := agent.SystemPromptTemplate
	if template == "" {
		// 默认模板
		template = "你是一个专业的代码生成助手。"
	}
	
	// 拼接：系统提示词 + 换行 + 知识库内容
	systemPromptContent.WriteString(template)
	if knowledgeContent.Len() > 0 {
		systemPromptContent.WriteString("\n\n")
		systemPromptContent.WriteString(knowledgeContent.String())
	}
	
	// 添加 system message
	llmMessages = append(llmMessages, llms.Message{
		Role:    "system",
		Content: systemPromptContent.String(),
	})
	logger.Infof(ctx, "[FunctionGenChat] System message 构建完成 - SystemPromptLength: %d, KnowledgeLength: %d, TotalLength: %d, TraceID: %s",
		len(template), knowledgeContent.Len(), systemPromptContent.Len(), traceId)

	// 5.3 处理 plugin 类型智能体（在构建用户消息之前）
	var userContent string
	if agent.AgentType == "plugin" {
		logger.Infof(ctx, "[FunctionGenChat] 调用 Plugin - AgentID: %d, MsgSubject: %s, MessageLength: %d, FilesCount: %d, TraceID: %s",
			agent.ID, agent.MsgSubject, len(req.Message.Content), len(req.Message.Files), traceId)
		// 6.1 构建 plugin 请求
		pluginFiles := make([]dto.PluginFile, 0, len(req.Message.Files))
		for _, f := range req.Message.Files {
			pluginFiles = append(pluginFiles, dto.PluginFile{
				Url:    f.Url,
				Remark: f.Remark,
			})
		}
		pluginReq := &dto.PluginRunReq{
			Message: req.Message.Content,
			Files:   pluginFiles,
		}

		// 6.2 调用 plugin 处理文件/输入
		logger.Debugf(ctx, "[FunctionGenChat] 开始调用 Plugin - Subject: %s, TraceID: %s", agent.MsgSubject, traceId)
		pluginResp, err := s.functionGenService.RunPlugin(ctx, agent, pluginReq)
		if err != nil {
			logger.Errorf(ctx, "[FunctionGenChat] Plugin 调用失败 - AgentID: %d, Subject: %s, TraceID: %s, Error: %v",
				agent.ID, agent.MsgSubject, traceId, err)
			return nil, err
		}

		logger.Infof(ctx, "[FunctionGenChat] Plugin 调用成功 - AgentID: %d, DataLength: %d, TraceID: %s",
			agent.ID, len(pluginResp.Data), traceId)

		// 6.3 构建用户消息：原始消息 + 换行 + 插件处理后的数据
		if pluginResp.Data != "" {
			userContent = fmt.Sprintf("%s\n\n%s", req.Message.Content, pluginResp.Data)
		} else {
			userContent = req.Message.Content
		}
		logger.Debugf(ctx, "[FunctionGenChat] 用户消息构建完成（包含插件处理结果） - OriginalLength: %d, PluginDataLength: %d, FinalLength: %d, TraceID: %s",
			len(req.Message.Content), len(pluginResp.Data), len(userContent), traceId)
	} else {
		// 非 plugin 类型，直接使用原始消息
		userContent = req.Message.Content
	}

	// 5.4 添加历史消息（转换为 LLM 格式，排除最后一条用户消息，因为我们要用插件处理后的内容替换）
	for i, msg := range historyMessages {
		// 跳过最后一条用户消息（刚保存的），因为我们要用插件处理后的内容替换
		if i == len(historyMessages)-1 && msg.Role == "user" {
			continue
		}
		llmMessages = append(llmMessages, llms.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// 5.5 添加当前用户消息（包含插件处理后的内容）
	llmMessages = append(llmMessages, llms.Message{
		Role:    "user",
		Content: userContent,
	})
	logger.Infof(ctx, "[FunctionGenChat] 用户消息已添加 - ContentLength: %d, TraceID: %s", len(userContent), traceId)

	// 7. 获取 LLM 配置并调用 LLM
	logger.Debugf(ctx, "[FunctionGenChat] 获取 LLM 配置 - LLMConfigID: %d, TraceID: %s", agent.LLMConfigID, traceId)
	var llmConfig *model.LLMConfig
	if agent.LLMConfigID > 0 {
		llmConfig, err = s.llmRepo.GetByID(agent.LLMConfigID)
		if err != nil {
			logger.Errorf(ctx, "[FunctionGenChat] 获取LLM配置失败 - LLMConfigID: %d, TraceID: %s, Error: %v", agent.LLMConfigID, traceId, err)
			return nil, fmt.Errorf("获取LLM配置失败: %w", err)
		}
		logger.Infof(ctx, "[FunctionGenChat] 使用智能体绑定的LLM - LLMConfigID: %d, Provider: %s, Model: %s, TraceID: %s",
			llmConfig.ID, llmConfig.Provider, llmConfig.Model, traceId)
	} else {
		llmConfig, err = s.llmRepo.GetDefault()
		if err != nil {
			logger.Errorf(ctx, "[FunctionGenChat] 获取默认LLM配置失败 - TraceID: %s, Error: %v", traceId, err)
			return nil, fmt.Errorf("获取默认LLM配置失败: %w", err)
		}
		logger.Infof(ctx, "[FunctionGenChat] 使用默认LLM - LLMConfigID: %d, Provider: %s, Model: %s, TraceID: %s",
			llmConfig.ID, llmConfig.Provider, llmConfig.Model, traceId)
	}

	// 创建 LLM 客户端
	provider := llms.Provider(llmConfig.Provider)
	options := llms.DefaultClientOptions()
	if llmConfig.Model != "" {
		options = options.WithModel(llmConfig.Model)
	}
	if llmConfig.Timeout > 0 {
		options = options.WithTimeout(time.Duration(llmConfig.Timeout) * time.Second)
	}
	if llmConfig.APIBase != "" {
		options = options.WithBaseURL(llmConfig.APIBase)
	}

	client, err := llms.NewLLMClientWithOptions(provider, llmConfig.APIKey, options)
	if err != nil {
		return nil, fmt.Errorf("创建LLM客户端失败: %w", err)
	}

	// 解析额外配置
	var extraConfig map[string]interface{}
	if llmConfig.ExtraConfig != nil && *llmConfig.ExtraConfig != "" {
		json.Unmarshal([]byte(*llmConfig.ExtraConfig), &extraConfig)
	}

	// 构建聊天请求
	chatReq := &llms.ChatRequest{
		Messages: llmMessages,
		Model:    llmConfig.Model,
	}
	if maxTokens, ok := extraConfig["max_tokens"].(float64); ok && maxTokens > 0 {
		chatReq.MaxTokens = int(maxTokens)
	} else if llmConfig.MaxTokens > 0 {
		chatReq.MaxTokens = llmConfig.MaxTokens
	}
	if temperature, ok := extraConfig["temperature"].(float64); ok {
		chatReq.Temperature = temperature
	}

	// 8. 创建 function_gen 记录（异步处理）
	logger.Infof(ctx, "[FunctionGenChat] 创建生成记录 - SessionID: %s, AgentID: %d, TreeID: %d, TraceID: %s",
		sessionID, req.AgentID, req.TreeID, traceId)
	record := &model.FunctionGenRecord{
		SessionID: sessionID,
		AgentID:   req.AgentID,
		TreeID:    req.TreeID,
		Status:    model.FunctionGenStatusGenerating,
		User:      user,
	}
	record.CreatedBy = user
	record.UpdatedBy = user
	if err := s.functionGenRepo.Create(record); err != nil {
		logger.Errorf(ctx, "[FunctionGenChat] 创建生成记录失败 - SessionID: %s, TraceID: %s, Error: %v", sessionID, traceId, err)
		return nil, fmt.Errorf("创建生成记录失败: %w", err)
	}
	logger.Infof(ctx, "[FunctionGenChat] 生成记录创建成功 - RecordID: %d, SessionID: %s, TraceID: %s", record.ID, sessionID, traceId)

	// 9. 调用 LLM（异步处理，先返回）
	logger.Infof(ctx, "[FunctionGenChat] 启动异步 LLM 调用 - RecordID: %d, MessagesCount: %d, TraceID: %s",
		record.ID, len(llmMessages), traceId)
	go func() {
		// 获取 trace_id 和 user（用于后续 NATS 消息 header）
		traceId := contextx.GetTraceId(ctx)
		requestUser := contextx.GetRequestUser(ctx)
		
		// 创建带超时的子 context（用于 LLM 调用）
		llmTimeout := time.Duration(llmConfig.Timeout) * time.Second
		if llmTimeout <= 0 {
			llmTimeout = 600 * time.Second // 默认 600 秒
		}
		asyncCtx, cancel := context.WithTimeout(context.Background(), llmTimeout)
		defer cancel()
		
		logger.Infof(asyncCtx, "[FunctionGenChat] 开始调用 LLM - RecordID: %d, Provider: %s, Model: %s, Timeout: %v, MessagesCount: %d, TraceID: %s",
			record.ID, llmConfig.Provider, llmConfig.Model, llmTimeout, len(chatReq.Messages), traceId)
		
		// 在 goroutine 中调用 LLM
		resp, err := client.Chat(asyncCtx, chatReq)
		
		if err != nil {
			// 记录错误日志（包含 trace_id）
			logger.Errorf(asyncCtx, "[FunctionGen] LLM调用失败: %v, RecordID: %d, AgentID: %d, TraceID: %s", 
				err, record.ID, req.AgentID, traceId)
			// 更新记录为失败
			s.functionGenRepo.UpdateStatus(record.ID, model.FunctionGenStatusFailed, err.Error())
			return
		}

		// 保存 assistant 消息
		assistantMsg := &model.AgentChatMessage{
			SessionID: sessionID,
			Role:      "assistant",
			Content:   resp.Content,
			User:      user,
		}
		assistantMsg.CreatedBy = user
		assistantMsg.UpdatedBy = user
		if err := s.messageRepo.Create(assistantMsg); err != nil {
			logger.Errorf(asyncCtx, "[FunctionGen] 保存assistant消息失败: %v, RecordID: %d, TraceID: %s", 
				err, record.ID, traceId)
			// 继续执行，不中断流程
		}

		// 更新记录
		if err := s.functionGenRepo.UpdateCode(record.ID, resp.Content, model.FunctionGenStatusCompleted); err != nil {
			logger.Errorf(asyncCtx, "[FunctionGen] 更新记录失败: %v, RecordID: %d, TraceID: %s", 
				err, record.ID, traceId)
			// 继续执行，不中断流程
		}

		// 记录成功日志
		logger.Infof(asyncCtx, "[FunctionGen] 代码生成成功, RecordID: %d, AgentID: %d, TreeID: %d, TraceID: %s", 
			record.ID, req.AgentID, req.TreeID, traceId)

		// 10. 将结果写入 NATS 队列（供 app-server 消费）
		resultData := &dto.FunctionGenResult{
			RecordID: record.ID,
			AgentID:  req.AgentID,
			TreeID:   req.TreeID,
			User:     user,
			Code:     resp.Content,
		}

		// 发布结果到 NATS
		logger.Infof(asyncCtx, "[FunctionGenChat] 发布结果到 NATS - RecordID: %d, CodeLength: %d, TraceID: %s",
			record.ID, len(resultData.Code), traceId)
		if err := s.functionGenService.PublishResult(asyncCtx, resultData, traceId, requestUser); err != nil {
			logger.Errorf(asyncCtx, "[FunctionGenChat] 发布结果到 NATS 失败 - RecordID: %d, TraceID: %s, Error: %v",
				record.ID, traceId, err)
			// 更新记录状态为失败
			_ = s.functionGenRepo.UpdateStatus(record.ID, model.FunctionGenStatusFailed, err.Error())
			return
		}
		logger.Infof(asyncCtx, "[FunctionGenChat] 结果已发布到 NATS - RecordID: %d, TraceID: %s", record.ID, traceId)
	}()

	// 立即返回响应
	logger.Infof(ctx, "[FunctionGenChat] 返回响应 - SessionID: %s, RecordID: %d, Status: %s, TraceID: %s",
		sessionID, record.ID, model.FunctionGenStatusGenerating, traceId)
	return &dto.FunctionGenAgentChatResp{
		SessionID: sessionID,
		Content:   "正在生成代码，请稍候...",
		RecordID:  record.ID,
		Status:    model.FunctionGenStatusGenerating,
	}, nil
}
