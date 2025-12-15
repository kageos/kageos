package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/llms"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/msgx"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// FunctionGenService 函数生成服务
// 负责调用 plugin 处理输入，以及发布函数生成结果到 NATS
type FunctionGenService struct {
	natsConn *nats.Conn
	cfg      *config.AgentServerConfig
}

// NewFunctionGenService 创建函数生成服务
func NewFunctionGenService(natsConn *nats.Conn, cfg *config.AgentServerConfig) *FunctionGenService {
	return &FunctionGenService{
		natsConn: natsConn,
		cfg:      cfg,
	}
}

// RunPlugin 执行插件处理
// agent: 智能体信息（包含 Plugin 关联）
// req: 插件执行请求（包含用户消息和文件列表）
func (s *FunctionGenService) RunPlugin(ctx context.Context, agent *model.Agent, req *dto.PluginRunReq) (*dto.PluginRunResp, error) {
	traceId := contextx.GetTraceId(ctx)

	// 1. 验证插件类型
	if agent.AgentType != "plugin" {
		return nil, fmt.Errorf("智能体类型不是 plugin，无法调用插件")
	}

	// 2. 获取插件信息
	if agent.PluginID == nil || *agent.PluginID == 0 {
		// 向后兼容：如果没有关联插件，使用旧的逻辑
		pluginSubject := subjects.BuildAgentPluginRunSubject(agent.ChatType, agent.CreatedBy, agent.ID)
		logger.Warnf(ctx, "[FunctionGenService] 智能体未关联插件，使用旧的插件主题 - Subject: %s, AgentID: %d, TraceID: %s",
			pluginSubject, agent.ID, traceId)
		return s.callPlugin(ctx, pluginSubject, agent.ID, req, traceId)
	}

	// 3. 验证插件是否已预加载
	if agent.Plugin == nil {
		return nil, fmt.Errorf("插件信息未预加载，请确保 AgentRepository.GetByID 预加载了 Plugin")
	}

	plugin := agent.Plugin
	if !plugin.Enabled {
		return nil, fmt.Errorf("插件已禁用: PluginID=%d", plugin.ID)
	}

	// 4. 使用插件的主题
	pluginSubject := plugin.Subject
	if pluginSubject == "" {
		return nil, fmt.Errorf("插件主题为空: PluginID=%d", plugin.ID)
	}

	logger.Infof(ctx, "[FunctionGenService] 开始调用 Plugin - Subject: %s, PluginID: %d, AgentID: %d, MessageLength: %d, FilesCount: %d, TraceID: %s",
		pluginSubject, plugin.ID, agent.ID, len(req.Message), len(req.Files), traceId)

	return s.callPlugin(ctx, pluginSubject, agent.ID, req, traceId)
}

// callPlugin 调用插件的通用方法
func (s *FunctionGenService) callPlugin(ctx context.Context, pluginSubject string, agentID int64, req *dto.PluginRunReq, traceId string) (*dto.PluginRunResp, error) {

	// 调用插件（使用 NATS Request/Reply 模式）
	var pluginResp dto.PluginRunResp
	timeout := time.Duration(s.cfg.GetNatsTimeout()) * time.Second
	logger.Debugf(ctx, "[FunctionGenService] 发送 NATS 请求 - Subject: %s, Timeout: %v, TraceID: %s",
		pluginSubject, timeout, traceId)

	startTime := time.Now()
	_, err := msgx.RequestMsgWithTimeout(ctx, s.natsConn, pluginSubject, req, &pluginResp, timeout)
	duration := time.Since(startTime)

	if err != nil {
		logger.Errorf(ctx, "[FunctionGenService] 调用插件失败 - Subject: %s, AgentID: %d, Duration: %v, TraceID: %s, Error: %v",
			pluginSubject, agentID, duration, traceId, err)
		return nil, fmt.Errorf("调用 plugin 失败: %w", err)
	}

	logger.Infof(ctx, "[FunctionGenService] 插件执行成功 - Subject: %s, AgentID: %d, DataLength: %d, Duration: %v, TraceID: %s",
		pluginSubject, agentID, len(pluginResp.Data), duration, traceId)

	return &pluginResp, nil
}

// PublishResult 发布函数生成结果到 NATS
// result: 函数生成结果
// traceId: 追踪ID（用于设置 NATS header）
// requestUser: 请求用户（用于设置 NATS header）
func (s *FunctionGenService) PublishResult(ctx context.Context, result *dto.FunctionGenResult, traceId, requestUser string) error {
	// 1. 构建结果主题
	resultSubject := subjects.GetAgentServerFunctionGenSubject()
	logger.Infof(ctx, "[FunctionGenService] 开始发布结果到 NATS - RecordID: %d, AgentID: %d, TreeID: %d, Subject: %s, TraceID: %s, User: %s",
		result.RecordID, result.AgentID, result.TreeID, resultSubject, traceId, requestUser)

	// 2. 序列化结果
	resultJSON, err := json.Marshal(result)
	if err != nil {
		logger.Errorf(ctx, "[FunctionGenService] 序列化结果失败 - RecordID: %d, TraceID: %s, Error: %v",
			result.RecordID, traceId, err)
		return fmt.Errorf("序列化结果失败: %w", err)
	}
	logger.Debugf(ctx, "[FunctionGenService] 结果序列化成功 - RecordID: %d, JSONLength: %d, CodeLength: %d, TraceID: %s",
		result.RecordID, len(resultJSON), len(result.Code), traceId)

	// 3. 创建 NATS 消息，并在 header 中设置 trace_id 和 user 信息
	msg := nats.NewMsg(resultSubject)
	msg.Data = resultJSON

	// 设置 header，供下游（app-server）使用
	if traceId != "" {
		msg.Header.Set("X-Trace-Id", traceId)
	}
	if requestUser != "" {
		msg.Header.Set("X-Request-User", requestUser)
	}
	logger.Debugf(ctx, "[FunctionGenService] NATS 消息 Header 设置完成 - RecordID: %d, TraceID: %s, User: %s",
		result.RecordID, traceId, requestUser)

	// 4. 发布消息
	if err := s.natsConn.PublishMsg(msg); err != nil {
		logger.Errorf(ctx, "[FunctionGenService] 发布NATS消息失败 - RecordID: %d, Subject: %s, TraceID: %s, Error: %v",
			result.RecordID, resultSubject, traceId, err)
		return fmt.Errorf("发布NATS消息失败: %w", err)
	}

	logger.Infof(ctx, "[FunctionGenService] NATS消息发布成功 - RecordID: %d, Subject: %s, TraceID: %s, User: %s, CodeLength: %d",
		result.RecordID, resultSubject, traceId, requestUser, len(result.Code))

	return nil
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
	var currentSession *model.AgentChatSession
	if req.SessionID == "" {
		// 创建新会话
		sessionID = uuid.New().String()
		logger.Infof(ctx, "[FunctionGenChat] 创建新会话 - SessionID: %s, TreeID: %d, TraceID: %s",
			sessionID, req.TreeID, traceId)
		session := &model.AgentChatSession{
			TreeID:    req.TreeID,
			SessionID: sessionID,
			AgentID:   req.AgentID,                   // 关联智能体ID
			Title:     "",                            // 可以后续根据第一条消息自动生成
			Status:    model.ChatSessionStatusActive, // 默认状态为 active
			User:      user,
		}
		session.CreatedBy = user
		session.UpdatedBy = user
		if err := s.sessionRepo.Create(session); err != nil {
			logger.Errorf(ctx, "[FunctionGenChat] 创建会话失败 - SessionID: %s, TraceID: %s, Error: %v", sessionID, traceId, err)
			return nil, fmt.Errorf("创建会话失败: %w", err)
		}
		currentSession = session
		logger.Infof(ctx, "[FunctionGenChat] 会话创建成功 - SessionID: %s, AgentID: %d, TraceID: %s", sessionID, req.AgentID, traceId)

		// 创建新会话后，立即设置为 generating 状态（因为即将开始代码生成）
		session.Status = model.ChatSessionStatusGenerating
		session.UpdatedBy = user
		if err := s.sessionRepo.Update(session); err != nil {
			logger.Errorf(ctx, "[FunctionGenChat] 更新新会话状态失败 - SessionID: %s, TraceID: %s, Error: %v", sessionID, traceId, err)
			return nil, fmt.Errorf("更新会话状态失败: %w", err)
		}
		logger.Infof(ctx, "[FunctionGenChat] 新会话状态已更新为 generating - SessionID: %s, TraceID: %s", sessionID, traceId)
	} else {
		// 使用现有会话
		sessionID = req.SessionID
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
		currentSession = session

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
	}

	// 3. 保存用户消息
	logger.Debugf(ctx, "[FunctionGenChat] 保存用户消息 - SessionID: %s, AgentID: %d, ContentLength: %d, FilesCount: %d, TraceID: %s",
		sessionID, req.AgentID, len(req.Message.Content), len(req.Message.Files), traceId)
	userMessage := &model.AgentChatMessage{
		SessionID: sessionID,
		AgentID:   req.AgentID, // 记录处理该消息的智能体ID
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

	// 3.1 如果是新会话且标题为空，使用第一条用户消息作为标题
	if req.SessionID == "" {
		// 获取会话
		session, err := s.sessionRepo.GetBySessionID(sessionID)
		if err == nil && session != nil && session.Title == "" {
			// 生成标题：取用户消息的前50个字符
			title := req.Message.Content
			if len(title) > 50 {
				title = title[:50] + "..."
			}
			// 去除换行和多余空格
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

	// 5.2 构建 system message：系统提示词 + 换行 + 知识库内容 + Package 上下文
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

	// 添加 Package 上下文信息（从前端传递）
	if req.Package != "" {
		systemPromptContent.WriteString(fmt.Sprintf("\n\n当前 Package 上下文：%s", req.Package))
		logger.Infof(ctx, "[FunctionGenChat] Package 上下文已添加 - Package: %s, TraceID: %s", req.Package, traceId)
	} else {
		logger.Warnf(ctx, "[FunctionGenChat] Package 信息为空 - TreeID: %d, TraceID: %s", req.TreeID, traceId)
	}

	// 添加已存在文件的上下文（确保生成的文件名唯一）
	if len(req.ExistingFiles) > 0 {
		systemPromptContent.WriteString("\n\n## 已存在的文件\n当前 Package 下已存在以下文件（不含 .go 后缀）：\n")
		for _, fileName := range req.ExistingFiles {
			systemPromptContent.WriteString(fmt.Sprintf("- %s.go\n", fileName))
		}
		systemPromptContent.WriteString("\n**重要**：生成代码时，请确保文件名唯一，不要与已存在的文件重名。如果生成的文件名与已存在的文件冲突，请修改文件名（例如：添加后缀或使用不同的名称）。\n")
		logger.Infof(ctx, "[FunctionGenChat] 已存在文件上下文已添加 - FilesCount: %d, Files: %v, TraceID: %s", len(req.ExistingFiles), req.ExistingFiles, traceId)
	}

	// 添加 system message
	llmMessages = append(llmMessages, llms.Message{
		Role:    "system",
		Content: systemPromptContent.String(),
	})
	logger.Infof(ctx, "[FunctionGenChat] System message 构建完成 - SystemPromptLength: %d, KnowledgeLength: %d, PackageContext: %s, TotalLength: %d, TraceID: %s",
		len(template), knowledgeContent.Len(), req.Package, systemPromptContent.Len(), traceId)

	// 5.4 处理 plugin 类型智能体（在构建用户消息之前）
	var userContent string
	var pluginResp *dto.PluginRunResp // 保存插件响应，用于后续记录 Metadata
	if agent.AgentType == "plugin" {
		logger.Infof(ctx, "[FunctionGenChat] 调用 Plugin - AgentID: %d, MessageLength: %d, FilesCount: %d, TraceID: %s",
			agent.ID, len(req.Message.Content), len(req.Message.Files), traceId)
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
		var err error
		pluginResp, err = s.functionGenService.RunPlugin(ctx, agent, pluginReq)
		if err != nil {
			logger.Errorf(ctx, "[FunctionGenChat] Plugin 调用失败 - AgentID: %d, TraceID: %s, Error: %v",
				agent.ID, traceId, err)
			return nil, err
		}

		// 6.2.1 检查插件返回的错误（插件处理失败，不应调用 LLM）
		if pluginResp.Error != "" {
			logger.Errorf(ctx, "[FunctionGenChat] Plugin 处理失败 - AgentID: %d, Error: %s, TraceID: %s",
				agent.ID, pluginResp.Error, traceId)
			return nil, fmt.Errorf("插件处理失败: %s", pluginResp.Error)
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

	// 5.5 添加历史消息（转换为 LLM 格式，排除最后一条用户消息，因为我们要用插件处理后的内容替换）
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

	// 5.6 添加当前用户消息（包含插件处理后的内容）
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
	// 如果配置了 UseThinking，设置到请求中
	if llmConfig.UseThinking {
		useThinking := true
		chatReq.UseThinking = &useThinking
	}

	// 8. 创建 function_gen 记录（包含 MessageID 和 Metadata）
	logger.Infof(ctx, "[FunctionGenChat] 创建生成记录 - SessionID: %s, MessageID: %d, AgentID: %d, TreeID: %d, TraceID: %s",
		sessionID, userMessage.ID, req.AgentID, req.TreeID, traceId)
	record := &model.FunctionGenRecord{
		SessionID: sessionID,
		MessageID: userMessage.ID, // 关联到用户消息
		AgentID:   req.AgentID,
		TreeID:    req.TreeID,
		Status:    model.FunctionGenStatusGenerating,
		User:      user,
	}
	record.CreatedBy = user
	record.UpdatedBy = user

	// 设置元数据（记录生成过程）
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
	logger.Infof(ctx, "[FunctionGenChat] 生成记录创建成功 - RecordID: %d, MessageID: %d, SessionID: %s, TraceID: %s", record.ID, userMessage.ID, sessionID, traceId)

	// 更新智能体的生成次数统计（异步，不阻塞主流程）
	go func() {
		if err := s.agentRepo.IncrementGenerationCount(req.AgentID); err != nil {
			logger.Errorf(ctx, "[FunctionGenChat] 更新智能体生成次数失败 - AgentID: %d, TraceID: %s, Error: %v", req.AgentID, traceId, err)
			// 不中断主流程，仅记录错误
		} else {
			logger.Infof(ctx, "[FunctionGenChat] 智能体生成次数已更新 - AgentID: %d, TraceID: %s", req.AgentID, traceId)
		}
	}()

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
			AgentID:   req.AgentID, // 记录处理该消息的智能体ID
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

		// 从 LLM 响应中提取代码（可能包含 Markdown 代码块）
		extractedCode := extractCodeFromMarkdown(resp.Content)
		logger.Infof(asyncCtx, "[FunctionGen] 代码提取完成 - 原始长度: %d, 提取后长度: %d, RecordID: %d, TraceID: %s",
			len(resp.Content), len(extractedCode), record.ID, traceId)

		// 更新记录（只保存代码，不更新状态）
		// 状态和 full_group_codes 由 app-server 处理完代码后的回调更新
		if err := s.functionGenRepo.UpdateCode(record.ID, extractedCode); err != nil {
			logger.Errorf(asyncCtx, "[FunctionGen] 更新代码失败: %v, RecordID: %d, TraceID: %s",
				err, record.ID, traceId)
			// 继续执行，不中断流程
		}
		logger.Infof(asyncCtx, "[FunctionGen] 代码已保存，等待 app-server 处理 - RecordID: %d, TraceID: %s", record.ID, traceId)

		// 记录成功日志
		logger.Infof(asyncCtx, "[FunctionGen] 代码生成成功, RecordID: %d, AgentID: %d, TreeID: %d, TraceID: %s",
			record.ID, req.AgentID, req.TreeID, traceId)

		// 10. 将结果写入 NATS 队列（供 app-server 消费）
		resultData := &dto.FunctionGenResult{
			RecordID:  record.ID,
			MessageID: record.MessageID, // 消息ID
			AgentID:   req.AgentID,
			TreeID:    req.TreeID,
			User:      user,
			Code:      extractedCode, // 使用提取后的代码
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
	// 判断是否可以继续输入：此时会话状态已经是 generating，所以不能继续输入
	// 因为已经触发了代码生成，会话状态已设置为 generating，所以不能继续输入
	canContinue := false
	logger.Infof(ctx, "[FunctionGenChat] 返回响应 - SessionID: %s, RecordID: %d, Status: %s, SessionStatus: %s, CanContinue: %v, TraceID: %s",
		sessionID, record.ID, model.FunctionGenStatusGenerating, currentSession.Status, canContinue, traceId)
	return &dto.FunctionGenAgentChatResp{
		SessionID:   sessionID,
		Content:     "正在生成代码，请稍候...",
		RecordID:    record.ID,
		Status:      model.FunctionGenStatusGenerating,
		CanContinue: canContinue,
	}, nil
}

// extractCodeFromMarkdown 从 Markdown 代码块中提取代码
// 支持格式：
// 1. ```go\n代码\n```
// 2. ```\n代码\n```
// 3. 如果找不到代码块，返回原始内容
func extractCodeFromMarkdown(content string) string {
	// 查找 ```go 或 ``` 开头的代码块
	lines := strings.Split(content, "\n")

	var codeBlocks []string
	var inCodeBlock bool
	var codeBlockStart int

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 检查是否是代码块开始标记
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

	// 如果有代码块，返回第一个（通常只有一个）
	if len(codeBlocks) > 0 {
		extracted := strings.TrimSpace(codeBlocks[0])
		// 如果提取的代码不为空，返回它
		if extracted != "" {
			return extracted
		}
	}

	// 如果没有找到代码块或代码块为空，返回原始内容（作为 fallback）
	return content
}
