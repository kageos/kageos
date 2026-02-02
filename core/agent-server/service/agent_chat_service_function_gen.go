package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/llms"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getFilesCount 获取文件数量（辅助函数）
func getFilesCount(files *types.Files) int {
	if files == nil {
		return 0
	}
	return len(files.Files)
}

// FunctionGenChat 函数生成智能体聊天（完整的对话流程）
func (s *AgentChatService) FunctionGenChat(ctx context.Context, req *dto.FunctionGenAgentChatReq) (*dto.FunctionGenAgentChatResp, error) {
	user := contextx.GetRequestUser(ctx)
	traceId := contextx.GetTraceId(ctx)

	logger.Infof(ctx, "[FunctionGenChat] 开始处理 - AgentID: %d, FullCodePath: %s, SessionID: %s, User: %s, TraceID: %s",
		req.AgentID, req.FullCodePath, req.SessionID, user, traceId)

	// 1. 验证并获取智能体
	agent, err := s.validateAndGetAgent(ctx, req.AgentID, traceId)
	if err != nil {
		return nil, err
	}

	// 2. 会话管理：创建或获取会话
	sessionID, _, userMessage, err := s.manageSession(ctx, req, user, traceId)
	if err != nil {
		return nil, err
	}

	// 3. 加载历史消息
	historyMessages, err := s.messageRepo.ListBySessionID(sessionID)
	if err != nil {
		logger.Errorf(ctx, "[FunctionGenChat] 加载历史消息失败 - SessionID: %s, TraceID: %s, Error: %v", sessionID, traceId, err)
		return nil, fmt.Errorf("加载历史消息失败: %w", err)
	}
	logger.Infof(ctx, "[FunctionGenChat] 历史消息数量 - SessionID: %s, Count: %d, TraceID: %s", sessionID, len(historyMessages), traceId)

	// 4. 构建 LLM 消息列表
	llmMessages, pluginResp, err := s.buildLLMMessages(ctx, req, agent, historyMessages, traceId)
	if err != nil {
		return nil, err
	}

	// 5. 获取 LLM 配置和客户端
	llmConfig, client, chatReq, err := s.prepareLLMRequest(ctx, agent, llmMessages, traceId)
	if err != nil {
		return nil, err
	}

	// 6. 创建函数生成记录
	record, err := s.createFunctionGenRecord(ctx, req, sessionID, userMessage.ID, user, pluginResp, traceId)
	if err != nil {
		return nil, err
	}

	// 7. 异步更新智能体生成次数
	go func() {
		if err := s.agentRepo.IncrementGenerationCount(req.AgentID); err != nil {
			logger.Errorf(ctx, "[FunctionGenChat] 更新智能体生成次数失败 - AgentID: %d, TraceID: %s, Error: %v", req.AgentID, traceId, err)
		} else {
			logger.Infof(ctx, "[FunctionGenChat] 智能体生成次数已更新 - AgentID: %d, TraceID: %s", req.AgentID, traceId)
		}
	}()

	// 8. 异步调用 LLM
	s.asyncCallLLM(ctx, req, sessionID, record, user, traceId, llmConfig, client, chatReq)

	// 9. 立即返回响应
	return &dto.FunctionGenAgentChatResp{
		SessionID:   sessionID,
		Content:     "正在生成代码，请稍候...",
		RecordID:    record.ID,
		Status:      model.FunctionGenStatusGenerating,
		CanContinue: false,
	}, nil
}

// validateAndGetAgent 验证并获取智能体信息
func (s *AgentChatService) validateAndGetAgent(ctx context.Context, agentID int64, traceId string) (*model.Agent, error) {
	logger.Debugf(ctx, "[FunctionGenChat] 获取智能体信息 - AgentID: %d", agentID)
	agent, err := s.agentRepo.GetByID(agentID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Errorf(ctx, "[FunctionGenChat] 智能体不存在 - AgentID: %d, TraceID: %s", agentID, traceId)
			return nil, fmt.Errorf("智能体不存在")
		}
		logger.Errorf(ctx, "[FunctionGenChat] 获取智能体失败 - AgentID: %d, TraceID: %s, Error: %v", agentID, traceId, err)
		return nil, fmt.Errorf("获取智能体失败: %w", err)
	}

	logger.Infof(ctx, "[FunctionGenChat] 智能体信息 - AgentID: %d, Name: %s, Type: %s, ChatType: %s, Enabled: %v, DocsPaths: %s, LLMConfigID: %d",
		agent.ID, agent.Name, agent.AgentType, agent.ChatType, agent.Enabled, agent.DocsPaths, agent.LLMConfigID)

	if !agent.Enabled {
		logger.Warnf(ctx, "[FunctionGenChat] 智能体已禁用 - AgentID: %d, TraceID: %s", agentID, traceId)
		return nil, fmt.Errorf("智能体已禁用")
	}

	return agent, nil
}

// manageSession 管理会话：创建或获取会话，保存用户消息
func (s *AgentChatService) manageSession(ctx context.Context, req *dto.FunctionGenAgentChatReq, user, traceId string) (string, *model.AgentChatSession, *model.AgentChatMessage, error) {
	var sessionID string
	var currentSession *model.AgentChatSession

	if req.SessionID == "" {
		// 创建新会话
		session, err := s.createNewSession(ctx, req, user, traceId)
		if err != nil {
			return "", nil, nil, err
		}
		sessionID = session.SessionID
		currentSession = session
	} else {
		// 使用现有会话
		session, err := s.getAndValidateSession(ctx, req.SessionID, user, traceId)
		if err != nil {
			return "", nil, nil, err
		}
		sessionID = req.SessionID
		currentSession = session
	}

	// 保存用户消息
	userMessage, err := s.saveUserMessage(ctx, req, sessionID, user, traceId)
	if err != nil {
		return "", nil, nil, err
	}

	// 如果是新会话，生成标题
	if req.SessionID == "" {
		s.generateSessionTitle(ctx, sessionID, req.Message.Content, user, traceId)
	}

	return sessionID, currentSession, userMessage, nil
}

// createNewSession 创建新会话
func (s *AgentChatService) createNewSession(ctx context.Context, req *dto.FunctionGenAgentChatReq, user, traceId string) (*model.AgentChatSession, error) {
	sessionID := uuid.New().String()

	// 根据 full_code_path 获取 TreeID（用于数据库存储）
	var treeID int64
	if req.FullCodePath != "" {
		detail, err := apicall.GetServiceTreeDetailByFullCodePath(ctx, req.FullCodePath)
		if err != nil || detail == nil {
			return nil, fmt.Errorf("无效的 full_code_path: %s", req.FullCodePath)
		}
		treeID = detail.ID
	} else {
		return nil, fmt.Errorf("full_code_path 必填")
	}

	logger.Infof(ctx, "[FunctionGenChat] 创建新会话 - SessionID: %s, FullCodePath: %s, TreeID: %d, TraceID: %s",
		sessionID, req.FullCodePath, treeID, traceId)

	aid := req.AgentID
	session := &model.AgentChatSession{
		TreeID:       treeID,
		FullCodePath: req.FullCodePath,
		SessionID:    sessionID,
		AgentID:      &aid,
		Title:        "",
		Status:       model.ChatSessionStatusGenerating,
		User:         user,
	}
	session.CreatedBy = user
	session.UpdatedBy = user

	if err := s.sessionRepo.Create(session); err != nil {
		logger.Errorf(ctx, "[FunctionGenChat] 创建会话失败 - SessionID: %s, TraceID: %s, Error: %v", sessionID, traceId, err)
		return nil, fmt.Errorf("创建会话失败: %w", err)
	}

	logger.Infof(ctx, "[FunctionGenChat] 会话创建成功 - SessionID: %s, AgentID: %d, TraceID: %s", sessionID, req.AgentID, traceId)
	return session, nil
}

// getAndValidateSession 获取并验证现有会话
func (s *AgentChatService) getAndValidateSession(ctx context.Context, sessionID, user, traceId string) (*model.AgentChatSession, error) {
	logger.Debugf(ctx, "[FunctionGenChat] 使用现有会话 - SessionID: %s, TraceID: %s", sessionID, traceId)

	session, err := s.sessionRepo.GetBySessionID(sessionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Errorf(ctx, "[FunctionGenChat] 会话不存在 - SessionID: %s, TraceID: %s", sessionID, traceId)
			return nil, fmt.Errorf("会话不存在")
		}
		logger.Errorf(ctx, "[FunctionGenChat] 获取会话失败 - SessionID: %s, TraceID: %s, Error: %v", sessionID, traceId, err)
		return nil, fmt.Errorf("获取会话失败: %w", err)
	}

	// 检查会话状态
	if session.Status == model.ChatSessionStatusDone {
		logger.Warnf(ctx, "[FunctionGenChat] 会话已结束，不能再输入 - SessionID: %s, Status: %s, TraceID: %s", sessionID, session.Status, traceId)
		return nil, fmt.Errorf("会话已结束，不能再输入。请新建会话继续生成")
	}
	if session.Status == model.ChatSessionStatusGenerating {
		logger.Warnf(ctx, "[FunctionGenChat] 会话正在生成中，请等待完成 - SessionID: %s, Status: %s, TraceID: %s", sessionID, session.Status, traceId)
		return nil, fmt.Errorf("会话正在生成中，请等待完成后再试")
	}

	// 设置会话状态为 generating
	session.Status = model.ChatSessionStatusGenerating
	session.UpdatedBy = user
	if err := s.sessionRepo.Update(session); err != nil {
		logger.Errorf(ctx, "[FunctionGenChat] 更新会话状态失败 - SessionID: %s, TraceID: %s, Error: %v", sessionID, traceId, err)
		return nil, fmt.Errorf("更新会话状态失败: %w", err)
	}

	logger.Infof(ctx, "[FunctionGenChat] 会话状态已更新为 generating - SessionID: %s, TraceID: %s", sessionID, traceId)
	return session, nil
}

// saveUserMessage 保存用户消息
func (s *AgentChatService) saveUserMessage(ctx context.Context, req *dto.FunctionGenAgentChatReq, sessionID, user, traceId string) (*model.AgentChatMessage, error) {
	logger.Debugf(ctx, "[FunctionGenChat] 保存用户消息 - SessionID: %s, AgentID: %d, ContentLength: %d, FilesCount: %d, TraceID: %s",
		sessionID, req.AgentID, len(req.Message.Content), getFilesCount(req.Message.Files), traceId)

	aid := req.AgentID
	userMessage := &model.AgentChatMessage{
		SessionID: sessionID,
		AgentID:   &aid,
		Role:      "user",
		Content:   req.Message.Content,
		User:      user,
	}

	// 处理文件列表
	if req.Message.Files != nil && len(req.Message.Files.Files) > 0 {
		filesJSON, err := json.Marshal(req.Message.Files)
		if err != nil {
			logger.Errorf(ctx, "[FunctionGenChat] 序列化文件列表失败 - SessionID: %s, TraceID: %s, Error: %v", sessionID, traceId, err)
			return nil, fmt.Errorf("序列化文件列表失败: %w", err)
		}
		filesJSONStr := string(filesJSON)
		userMessage.Files = &filesJSONStr
		logger.Infof(ctx, "[FunctionGenChat] 用户消息包含文件 - SessionID: %s, Files: %s, TraceID: %s", sessionID, filesJSONStr, traceId)
	}

	userMessage.CreatedBy = user
	userMessage.UpdatedBy = user

	if err := s.messageRepo.Create(userMessage); err != nil {
		logger.Errorf(ctx, "[FunctionGenChat] 保存用户消息失败 - SessionID: %s, TraceID: %s, Error: %v", sessionID, traceId, err)
		return nil, fmt.Errorf("保存用户消息失败: %w", err)
	}

	return userMessage, nil
}

// generateSessionTitle 生成会话标题
func (s *AgentChatService) generateSessionTitle(ctx context.Context, sessionID, content, user, traceId string) {
	session, err := s.sessionRepo.GetBySessionID(sessionID)
	if err == nil && session != nil && session.Title == "" {
		title := content
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		title = strings.TrimSpace(strings.ReplaceAll(title, "\n", " "))
		if title == "" {
			title = "新会话"
		}
		session.Title = title
		session.UpdatedBy = user
		if err := s.sessionRepo.Update(session); err != nil {
			logger.Warnf(ctx, "[FunctionGenChat] 更新会话标题失败 - SessionID: %s, Title: %s, TraceID: %s, Error: %v", sessionID, title, traceId, err)
		} else {
			logger.Infof(ctx, "[FunctionGenChat] 会话标题已生成 - SessionID: %s, Title: %s, TraceID: %s", sessionID, title, traceId)
		}
	}
}

// buildLLMMessages 构建 LLM 消息列表
func (s *AgentChatService) buildLLMMessages(ctx context.Context, req *dto.FunctionGenAgentChatReq, agent *model.Agent, historyMessages []*model.AgentChatMessage, traceId string) ([]llms.Message, *dto.PluginRunResp, error) {
	llmMessages := make([]llms.Message, 0)

	// 1. 构建系统消息
	systemMessage, err := s.buildSystemMessage(ctx, req, agent, traceId)
	if err != nil {
		return nil, nil, err
	}
	llmMessages = append(llmMessages, systemMessage)

	// 2. 处理插件（如果是 plugin 类型智能体）
	userContent, pluginResp, err := s.processPlugin(ctx, req, agent, traceId)
	if err != nil {
		return nil, nil, err
	}

	// 3. 添加历史消息（排除最后一条用户消息）
	for i, msg := range historyMessages {
		if i == len(historyMessages)-1 && msg.Role == "user" {
			continue
		}
		llmMessages = append(llmMessages, llms.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// 4. 添加当前用户消息（包含插件处理后的内容）
	llmMessages = append(llmMessages, llms.Message{
		Role:    "user",
		Content: userContent,
	})
	logger.Infof(ctx, "[FunctionGenChat] 用户消息已添加 - ContentLength: %d, TraceID: %s", len(userContent), traceId)

	return llmMessages, pluginResp, nil
}

// buildSystemMessage 构建系统消息
func (s *AgentChatService) buildSystemMessage(ctx context.Context, req *dto.FunctionGenAgentChatReq, agent *model.Agent, traceId string) (llms.Message, error) {
	// 1. 获取文档路径数组
	docsPaths := agent.GetDocsPaths()
	if len(docsPaths) == 0 {
		// 使用默认路径（标准库）
		docsPaths = []string{"/system/official/sdk"}
		logger.Warnf(ctx, "[FunctionGenChat] agent.docs_paths 为空，使用默认路径: %v, TraceID: %s", docsPaths, traceId)
	}

	logger.Infof(ctx, "[FunctionGenChat] 开始批量加载文档 - Paths: %v, TraceID: %s", docsPaths, traceId)

	// 2. 批量获取文档（一次 API 调用）
	docsResp, err := s.getDocsByPaths(ctx, docsPaths, traceId)
	if err != nil {
		logger.Errorf(ctx, "[FunctionGenChat] 批量获取文档失败 - Paths: %v, TraceID: %s, Error: %v", docsPaths, traceId, err)
		// 不中断流程，继续使用空知识库
		docsResp = &dto.GetDocsByPathsResp{Docs: []*dto.DocItem{}}
	}

	// 3. 构建知识库内容
	var knowledgeContent strings.Builder
	for _, doc := range docsResp.Docs {
		knowledgeContent.WriteString(fmt.Sprintf("\n## %s\n%s\n", doc.Name, doc.Content))
	}
	logger.Infof(ctx, "[FunctionGenChat] 知识库内容构建完成 - TotalDocs: %d, ContentLength: %d, TraceID: %s",
		len(docsResp.Docs), knowledgeContent.Len(), traceId)

	// 2. 构建系统提示词
	var systemPromptContent strings.Builder
	template := agent.SystemPromptTemplate
	if template == "" {
		template = "你是一个专业的代码生成助手。"
	}

	systemPromptContent.WriteString(template)
	if knowledgeContent.Len() > 0 {
		systemPromptContent.WriteString("\n\n")
		systemPromptContent.WriteString(knowledgeContent.String())
	}

	// 3. 添加已存在的子目录上下文（用于告诉模型已存在的目录）
	// ⭐ 不再传递 Package 上下文，因为 package 名称应该由元数据中的 directory_code 决定
	// ⭐ 改为传递已存在的子目录列表（由前端传递），格式如：ticket:工单管理
	if len(req.ExistingDirectories) > 0 {
		systemPromptContent.WriteString("\n\n## 当前目录下已存在的子目录\n当前目录下已存在以下子目录（格式：目录代码:目录名称）：\n")
		for _, dir := range req.ExistingDirectories {
			systemPromptContent.WriteString(fmt.Sprintf("- %s:%s\n", dir.Code, dir.Name))
		}
		systemPromptContent.WriteString("\n**重要**：生成代码时，请确保 `directory_code` 唯一，不要与已存在的子目录重名。如果生成的 `directory_code` 与已存在的子目录冲突，请修改 `directory_code`（例如：添加后缀或使用不同的名称）。\n")
		logger.Infof(ctx, "[FunctionGenChat] 已存在子目录上下文已添加 - DirectoriesCount: %d, Directories: %v, TraceID: %s", len(req.ExistingDirectories), req.ExistingDirectories, traceId)
	} else {
		logger.Infof(ctx, "[FunctionGenChat] 当前目录下没有子目录 - FullCodePath: %s, TraceID: %s", req.FullCodePath, traceId)
	}

	logger.Infof(ctx, "[FunctionGenChat] System message 构建完成 - SystemPromptLength: %d, KnowledgeLength: %d, TotalLength: %d, TraceID: %s",
		len(template), knowledgeContent.Len(), systemPromptContent.Len(), traceId)

	return llms.Message{
		Role:    "system",
		Content: systemPromptContent.String(),
	}, nil
}

// processPlugin 处理插件
func (s *AgentChatService) processPlugin(ctx context.Context, req *dto.FunctionGenAgentChatReq, agent *model.Agent, traceId string) (string, *dto.PluginRunResp, error) {
	if agent.AgentType != "plugin" {
		return req.Message.Content, nil, nil
	}

	logger.Infof(ctx, "[FunctionGenChat] 调用 Plugin - AgentID: %d, MessageLength: %d, FilesCount: %d, TraceID: %s",
		agent.ID, len(req.Message.Content), getFilesCount(req.Message.Files), traceId)

	// 构建 plugin 请求（直接使用 types.Files）
	pluginReq := &dto.PluginRunReq{
		Content: req.Message.Content,
		Files:   req.Message.Files, // 直接传递 types.Files
	}

	// 调用 plugin
	pluginResp, err := s.functionGenService.RunPlugin(ctx, agent, pluginReq)
	if err != nil {
		logger.Errorf(ctx, "[FunctionGenChat] Plugin 调用失败 - AgentID: %d, TraceID: %s, Error: %v", agent.ID, traceId, err)
		return "", nil, err
	}

	if pluginResp.Error != "" {
		logger.Errorf(ctx, "[FunctionGenChat] Plugin 处理失败 - AgentID: %d, Error: %s, TraceID: %s", agent.ID, pluginResp.Error, traceId)
		return "", nil, fmt.Errorf("插件处理失败: %s", pluginResp.Error)
	}

	logger.Infof(ctx, "[FunctionGenChat] Plugin 调用成功 - AgentID: %d, DataLength: %d, TraceID: %s", agent.ID, len(pluginResp.Data), traceId)

	// 构建用户消息
	userContent := req.Message.Content
	if pluginResp.Data != "" {
		userContent = fmt.Sprintf("%s\n\n%s", req.Message.Content, pluginResp.Data)
	}

	logger.Debugf(ctx, "[FunctionGenChat] 用户消息构建完成（包含插件处理结果） - OriginalLength: %d, PluginDataLength: %d, FinalLength: %d, TraceID: %s",
		len(req.Message.Content), len(pluginResp.Data), len(userContent), traceId)

	return userContent, pluginResp, nil
}

// prepareLLMRequest 准备 LLM 请求
func (s *AgentChatService) prepareLLMRequest(ctx context.Context, agent *model.Agent, llmMessages []llms.Message, traceId string) (*model.LLMConfig, llms.LLMClient, *llms.ChatRequest, error) {
	// 1. 获取 LLM 配置
	logger.Debugf(ctx, "[FunctionGenChat] 获取 LLM 配置 - LLMConfigID: %d, TraceID: %s", agent.LLMConfigID, traceId)
	var llmConfig *model.LLMConfig
	var err error

	if agent.LLMConfigID > 0 {
		llmConfig, err = s.llmRepo.GetByID(agent.LLMConfigID)
		if err != nil {
			logger.Errorf(ctx, "[FunctionGenChat] 获取LLM配置失败 - LLMConfigID: %d, TraceID: %s, Error: %v", agent.LLMConfigID, traceId, err)
			return nil, nil, nil, fmt.Errorf("获取LLM配置失败: %w", err)
		}
		logger.Infof(ctx, "[FunctionGenChat] 使用智能体绑定的LLM - LLMConfigID: %d, Provider: %s, Model: %s, TraceID: %s",
			llmConfig.ID, llmConfig.Provider, llmConfig.Model, traceId)
	} else {
		llmConfig, err = s.llmRepo.GetDefault()
		if err != nil {
			logger.Errorf(ctx, "[FunctionGenChat] 获取默认LLM配置失败 - TraceID: %s, Error: %v", traceId, err)
			return nil, nil, nil, fmt.Errorf("获取默认LLM配置失败: %w", err)
		}
		logger.Infof(ctx, "[FunctionGenChat] 使用默认LLM - LLMConfigID: %d, Provider: %s, Model: %s, TraceID: %s",
			llmConfig.ID, llmConfig.Provider, llmConfig.Model, traceId)
	}

	// 2. 创建 LLM 客户端
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
		return nil, nil, nil, fmt.Errorf("创建LLM客户端失败: %w", err)
	}

	// 3. 解析额外配置
	var extraConfig map[string]interface{}
	if llmConfig.ExtraConfig != nil && *llmConfig.ExtraConfig != "" {
		json.Unmarshal([]byte(*llmConfig.ExtraConfig), &extraConfig)
	}

	// 4. 构建聊天请求
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
	if llmConfig.UseThinking {
		useThinking := true
		chatReq.UseThinking = &useThinking
	}

	return llmConfig, client, chatReq, nil
}

// createFunctionGenRecord 创建函数生成记录
func (s *AgentChatService) createFunctionGenRecord(ctx context.Context, req *dto.FunctionGenAgentChatReq, sessionID string, messageID int64, user string, pluginResp *dto.PluginRunResp, traceId string) (*model.FunctionGenRecord, error) {
	// 根据 full_code_path 获取 TreeID（用于数据库存储）
	var treeID int64
	if req.FullCodePath != "" {
		detail, err := apicall.GetServiceTreeDetailByFullCodePath(ctx, req.FullCodePath)
		if err != nil || detail == nil {
			return nil, fmt.Errorf("无效的 full_code_path: %s", req.FullCodePath)
		}
		treeID = detail.ID
	} else {
		return nil, fmt.Errorf("full_code_path 必填")
	}

	logger.Infof(ctx, "[FunctionGenChat] 创建生成记录 - SessionID: %s, MessageID: %d, AgentID: %d, FullCodePath: %s, TreeID: %d, TraceID: %s",
		sessionID, messageID, req.AgentID, req.FullCodePath, treeID, traceId)

	record := &model.FunctionGenRecord{
		SessionID: sessionID,
		MessageID: messageID,
		AgentID:   req.AgentID,
		TreeID:    treeID,
		Status:    model.FunctionGenStatusGenerating,
		User:      user,
	}
	record.CreatedBy = user
	record.UpdatedBy = user

	// 设置元数据
	metadata := map[string]interface{}{
		"user_message": req.Message.Content,
		"files":        req.Message.Files,
	}
	if pluginResp != nil {
		metadata["plugin_data"] = pluginResp.Data
	}
	if err := record.SetMetadata(metadata); err != nil {
		logger.Errorf(ctx, "[FunctionGenChat] 设置元数据失败 - SessionID: %s, TraceID: %s, Error: %v", sessionID, traceId, err)
		return nil, fmt.Errorf("设置元数据失败: %w", err)
	}

	if err := s.functionGenRepo.Create(record); err != nil {
		logger.Errorf(ctx, "[FunctionGenChat] 创建生成记录失败 - SessionID: %s, TraceID: %s, Error: %v", sessionID, traceId, err)
		return nil, fmt.Errorf("创建生成记录失败: %w", err)
	}

	logger.Infof(ctx, "[FunctionGenChat] 生成记录创建成功 - RecordID: %d, MessageID: %d, SessionID: %s, TraceID: %s", record.ID, messageID, sessionID, traceId)
	return record, nil
}

// asyncCallLLM 异步调用 LLM
func (s *AgentChatService) asyncCallLLM(ctx context.Context, req *dto.FunctionGenAgentChatReq, sessionID string, record *model.FunctionGenRecord, user, traceId string, llmConfig *model.LLMConfig, client llms.LLMClient, chatReq *llms.ChatRequest) {
	logger.Infof(ctx, "[FunctionGenChat] 启动异步 LLM 调用 - RecordID: %d, MessagesCount: %d, TraceID: %s",
		record.ID, len(chatReq.Messages), traceId)

	go func() {
		// 创建带超时的子 context
		//llmTimeout := time.Duration(llmConfig.Timeout) * time.Second
		//if llmTimeout <= 0 {
		//	llmTimeout = 600 * time.Second
		//}
		//asyncCtx, cancel := context.WithTimeout(context.Background(), llmTimeout)
		//defer cancel()

		//logger.Infof(ctx, "[FunctionGenChat] 开始调用 LLM - RecordID: %d, Provider: %s, Model: %s, Timeout: %v, MessagesCount: %d, TraceID: %s",
		//	record.ID, llmConfig.Provider, llmConfig.Model, llmTimeout, len(chatReq.Messages), traceId)

		// 调用 LLM
		resp, err := client.Chat(ctx, chatReq)
		if err != nil {
			logger.Errorf(ctx, "[FunctionGen] LLM调用失败: %v, RecordID: %d, AgentID: %d, TraceID: %s",
				err, record.ID, req.AgentID, traceId)
			s.functionGenRepo.UpdateStatus(record.ID, model.FunctionGenStatusFailed, err.Error())
			return
		}

		// 保存 assistant 消息
		s.saveAssistantMessage(ctx, sessionID, req.AgentID, resp.Content, user, record.ID, traceId)

		// 提取代码
		extractedCode := s.extractCodeFromLLMResponse(resp.Content)
		logger.Infof(ctx, "[FunctionGen] 代码提取完成 - 原始长度: %d, 提取后长度: %d, RecordID: %d, TraceID: %s",
			len(resp.Content), len(extractedCode), record.ID, traceId)

		// 更新记录
		if err := s.functionGenRepo.UpdateCode(record.ID, extractedCode); err != nil {
			logger.Errorf(ctx, "[FunctionGen] 更新代码失败: %v, RecordID: %d, TraceID: %s", err, record.ID, traceId)
		}

		// ⭐ 直接传递代码和父目录（full_code_path）；租户由 app-server 从 full_code_path 解析，不传 User
		// 发布结果到 app-server
		resultData := &dto.AddFunctionsReq{
			RecordID:     record.ID,
			MessageID:    record.MessageID,
			AgentID:      req.AgentID,
			FullCodePath: req.FullCodePath, // 必填
			SourceCode:   extractedCode,    // 传递完整代码，app-server 会解析元数据
		}

		logger.Infof(ctx, "[FunctionGenChat] 提交生成的代码到 app-server - RecordID: %d, SourceCodeLength: %d, TraceID: %s",
			record.ID, len(resultData.SourceCode), traceId)
		if err := s.functionGenService.SubmitGeneratedCodeTask(ctx, resultData); err != nil {
			logger.Errorf(ctx, "[FunctionGenChat] 提交代码失败 - RecordID: %d, TraceID: %s, Error: %v", record.ID, traceId, err)
			s.functionGenRepo.UpdateStatus(record.ID, model.FunctionGenStatusFailed, err.Error())
			return
		}
		logger.Infof(ctx, "[FunctionGenChat] 代码已提交 - RecordID: %d, TraceID: %s", record.ID, traceId)
	}()
}

// saveAssistantMessage 保存 assistant 消息
func (s *AgentChatService) saveAssistantMessage(ctx context.Context, sessionID string, agentID int64, content, user string, recordID int64, traceId string) {
	aid := agentID
	assistantMsg := &model.AgentChatMessage{
		SessionID: sessionID,
		AgentID:   &aid,
		Role:      "assistant",
		Content:   content,
		User:      user,
	}
	assistantMsg.CreatedBy = user
	assistantMsg.UpdatedBy = user

	if err := s.messageRepo.Create(assistantMsg); err != nil {
		logger.Errorf(ctx, "[FunctionGen] 保存assistant消息失败: %v, RecordID: %d, TraceID: %s", err, recordID, traceId)
	}
}

// extractCodeFromLLMResponse 从 LLM 响应中提取代码（Markdown 代码块）
func (s *AgentChatService) extractCodeFromLLMResponse(content string) string {
	// 查找 ```go 或 ``` 开头的代码块
	lines := strings.Split(content, "\n")
	var codeBlocks []string
	inCodeBlock := false
	codeBlockStart := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if inCodeBlock {
				// 代码块结束，提取内容
				if i > codeBlockStart {
					codeBlock := strings.Join(lines[codeBlockStart+1:i], "\n")
					codeBlocks = append(codeBlocks, codeBlock)
				}
				inCodeBlock = false
			} else {
				// 代码块开始
				inCodeBlock = true
				codeBlockStart = i
			}
			continue
		}
	}

	// 如果代码块没有正确关闭，也提取已收集的内容
	if inCodeBlock && codeBlockStart < len(lines)-1 {
		codeBlock := strings.Join(lines[codeBlockStart+1:], "\n")
		codeBlocks = append(codeBlocks, codeBlock)
	}

	// 如果找到代码块，返回第一个（通常是最主要的）
	if len(codeBlocks) > 0 {
		return strings.TrimSpace(codeBlocks[0])
	}

	// 如果没有找到代码块，返回原始内容
	return content
}

// getDocsByPaths 批量获取文档（通过 apicall 调用 app-server）
func (s *AgentChatService) getDocsByPaths(ctx context.Context, paths []string, traceId string) (*dto.GetDocsByPathsResp, error) {
	logger.Infof(ctx, "[FunctionGenChat] 调用批量获取文档 API - Paths: %v, TraceID: %s", paths, traceId)

	// 通过 apicall 调用 app-server 的批量获取文档接口
	resp, err := apicall.GetDocsByPaths(ctx, paths)
	if err != nil {
		logger.Errorf(ctx, "[FunctionGenChat] 批量获取文档失败 - Paths: %v, TraceID: %s, Error: %v", paths, traceId, err)
		return nil, err
	}

	logger.Infof(ctx, "[FunctionGenChat] 批量获取文档成功 - Paths: %v, DocsCount: %d, TraceID: %s", paths, len(resp.Docs), traceId)

	return resp, nil
}
