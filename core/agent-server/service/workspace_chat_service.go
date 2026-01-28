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
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/llms"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	SourceWorkspace = "workspace"
	MaxToolRounds   = 5 // 最大工具调用轮数，防止无限循环
)

// 消息角色常量
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// 工具调用状态常量
const (
	ToolCallStatusOK    = "ok"
	ToolCallStatusError = "error"
)

// 流式事件类型常量
const (
	EventSession  = "session"
	EventAgentID  = "agent_id"
	EventToolCall = "tool_call"
	EventContent  = "content"
	EventDone     = "done"
	EventError    = "error"
)

// WorkspaceChatService 工作台对话编排：会话、历史、LLM、Tool 循环；复用 prepareLLMRequest 逻辑与 pkg/llms ChatStream
type WorkspaceChatService struct {
	toolReg     *ToolRegistry
	sessionRepo *repository.ChatSessionRepository
	messageRepo *repository.ChatMessageRepository
	llmRepo     *repository.LLMRepository
	agentRepo   *repository.AgentRepository
}

// NewWorkspaceChatService 创建 WorkspaceChatService
func NewWorkspaceChatService(
	toolReg *ToolRegistry,
	sessionRepo *repository.ChatSessionRepository,
	messageRepo *repository.ChatMessageRepository,
	llmRepo *repository.LLMRepository,
	agentRepo *repository.AgentRepository,
) *WorkspaceChatService {
	return &WorkspaceChatService{
		toolReg:     toolReg,
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
		llmRepo:     llmRepo,
		agentRepo:   agentRepo,
	}
}

// StreamEvent 流式事件：用于 SSE 传输
type StreamEvent struct {
	Event string      `json:"event"` // session|agent_id|tool_call|content|done|error
	Data  interface{} `json:"data"`  // 对应负载（具体类型见下方各事件结构体）
}

// StreamEventSession session 事件数据
type StreamEventSession struct {
	SessionID string `json:"session_id"`
}

// StreamEventAgentID agent_id 事件数据
type StreamEventAgentID struct {
	AgentID int64 `json:"agent_id"`
}

// StreamEventToolCall tool_call 事件数据
type StreamEventToolCall struct {
	Name   string `json:"name"`
	Status string `json:"status"` // ok / error
}

// StreamEventContent content 事件数据
type StreamEventContent struct {
	Content string `json:"content"`
}

// StreamEventDone done 事件数据
type StreamEventDone struct {
	SessionID string                             `json:"session_id"`
	AgentID   int64                              `json:"agent_id"`
	ToolCalls []dto.WorkspaceChatToolCallSummary `json:"tool_calls"`
}

// StreamEventError error 事件数据
type StreamEventError struct {
	Message string `json:"message"`
}

// WorkspaceChatStream 工作台对话流式入口：通过 eventChan 发送 SSE 事件（session、agent_id、tool_call、content、done、error）
// eventChan 为只写 channel，由调用方在 goroutine 中读取并写 SSE，避免 Flush 阻塞 LLM stream 消费
func (s *WorkspaceChatService) WorkspaceChatStream(ctx context.Context, req *dto.WorkspaceChatReq, eventChan chan<- StreamEvent) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("WorkspaceChatStream panic: %v", e)
		select {
		case eventChan <- StreamEvent{Event: EventError, Data: StreamEventError{Message: fmt.Sprintf("%v", e)}}:
		case <-ctx.Done():
		}
		}
	}()
	// 辅助函数：发送事件到 channel（检查 ctx.Done()，避免在客户端断开时阻塞）
	sendEvent := func(event string, data interface{}) {
		select {
		case eventChan <- StreamEvent{Event: event, Data: data}:
		case <-ctx.Done():
			// 上下文已取消（客户端断开），不再发送
		}
	}
	user := contextx.GetRequestUser(ctx)
	fullCodePath := strings.TrimSpace(req.FullCodePath)
	if fullCodePath == "" {
		return s.handleError(sendEvent, "full_code_path 必填", nil)
	}

	// 1) 获取工作台环境信息（包含目录详情、子节点等，一次性获取，避免重复调用）
	workspaceCtx, e := apicall.GetWorkspaceContext(ctx, fullCodePath)
	if e != nil || workspaceCtx == nil {
		return s.handleError(sendEvent, "无效的 full_code_path，无法解析目录", e)
	}
	directoryName := workspaceCtx.Directory.Name // 目录的中文名称
	if directoryName == "" {
		directoryName = workspaceCtx.Directory.Code // 如果没有中文名称，使用 code 作为备选
	}

	// 2) 解析或创建 session
	var session *model.AgentChatSession
	if req.SessionID != "" {
		var e error
		session, e = s.sessionRepo.GetBySessionID(req.SessionID)
		if e != nil || session == nil {
			return s.handleError(sendEvent, fmt.Sprintf("会话不存在: %s", req.SessionID), e)
		}
		if req.AgentID > 0 {
			aid := req.AgentID
			session.AgentID = &aid
			session.UpdatedBy = user
			_ = s.sessionRepo.Update(session)
		}
	} else {
		var agentIDForSession *int64
		if req.AgentID > 0 {
			aid := req.AgentID
			agentIDForSession = &aid
		}
		session = &model.AgentChatSession{
			TreeID:       workspaceCtx.Directory.ID,
			FullCodePath: fullCodePath,
			Source:       SourceWorkspace,
			SessionID:    uuid.New().String(),
			AgentID:      agentIDForSession,
			Title:        "",
			Status:       model.ChatSessionStatusActive,
			User:         user,
		}
		session.CreatedBy = user
		session.UpdatedBy = user
		if e := s.sessionRepo.Create(session); e != nil {
			return s.handleError(sendEvent, "创建会话失败", e)
		}
	}

		sessionID := session.SessionID
		
		// 统一获取 agentID：优先使用 session.AgentID，如果为0则使用请求中的 req.AgentID
		agentID := s.getAgentID(session.AgentID, req.AgentID)
		agentIDPtr := s.getAgentIDPtr(agentID)

		// 3) 保存 user 消息
	userMsg := &model.AgentChatMessage{
		SessionID: sessionID, AgentID: agentIDPtr, Role: RoleUser,
		Content: req.Message.Content, User: user,
	}
	userMsg.CreatedBy = user
	userMsg.UpdatedBy = user
	if e := s.messageRepo.Create(userMsg); e != nil {
		return s.handleError(sendEvent, "保存用户消息失败", e)
	}

		// 3.1) 如果是新会话且标题为空，使用第一条用户消息作为标题
	if session.Title == "" {
		title := strings.TrimSpace(req.Message.Content)
		// 移除换行符，替换为空格
		title = strings.ReplaceAll(title, "\n", " ")
		// 截取前50个字符
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		if title == "" {
			title = "新会话"
		}
		session.Title = title
		session.UpdatedBy = user
		if e := s.sessionRepo.Update(session); e != nil {
			logger.Warnf(ctx, "[WorkspaceChatStream] 更新会话标题失败: %v", e)
		} else {
			logger.Infof(ctx, "[WorkspaceChatStream] 会话标题已生成 - SessionID: %s, Title: %s", sessionID, title)
		}
	}

	sendEvent(EventSession, StreamEventSession{SessionID: sessionID})
	sendEvent(EventAgentID, StreamEventAgentID{AgentID: agentID})

	// 构建 LLM 消息和工具定义（传入已获取的环境信息，避免重复调用）
	msgs, tools, e := s.buildLLMMessages(ctx, sessionID, fullCodePath, directoryName, agentID, workspaceCtx)
	if e != nil {
		return s.handleError(sendEvent, e.Error(), e)
	}
	_, client, chatReq, e := s.prepareLLMRequest(ctx, agentID, msgs, tools)
	if e != nil {
		return s.handleError(sendEvent, e.Error(), e)
	}
	stream, e := client.ChatStream(ctx, chatReq)
	if e != nil {
		return s.handleError(sendEvent, "LLM 调用失败", e)
	}

	// 流式处理：支持 tool_calls
	content, allToolCalls, err := s.processStreamChunks(ctx, stream, sendEvent)
	if err != nil {
		return err
	}

	// 处理 tool_calls（如果有）
	if len(allToolCalls) > 0 {
		// 保存 assistant 消息（包含 tool_calls）
		s.saveAssistantMessageWithToolCalls(ctx, sessionID, agentIDPtr, content, allToolCalls, user)

		// 执行工具调用
		toolSummaries := s.executeToolCalls(ctx, allToolCalls, sessionID, fullCodePath, agentIDPtr, user, req.Message.Files, sendEvent)

		// 继续下一轮对话：将 tool 结果发送给 LLM，让模型基于工具结果继续回复
		return s.continueToolCallLoop(ctx, sessionID, fullCodePath, agentID, agentIDPtr, user, 1, toolSummaries, sendEvent)
	}

	// 普通回复：保存 assistant 消息
	if err := s.saveAssistantMessage(ctx, sessionID, agentIDPtr, content, user); err != nil {
		logger.Warnf(ctx, "[WorkspaceChatStream] 保存 assistant 消息失败: %v", err)
	}

	sendEvent(EventDone, StreamEventDone{SessionID: sessionID, AgentID: agentID, ToolCalls: []dto.WorkspaceChatToolCallSummary{}})
	return nil
}

// continueToolCallLoop 继续工具调用循环：将 tool 结果发送给 LLM，让模型基于工具结果继续回复
func (s *WorkspaceChatService) continueToolCallLoop(
	ctx context.Context,
	sessionID, fullCodePath string,
	agentID int64,
	agentIDPtr *int64,
	user string,
	round int,
	previousToolSummaries []dto.WorkspaceChatToolCallSummary,
	sendEvent func(string, interface{}),
) error {
	// 防止无限循环
	if round >= MaxToolRounds {
		logger.Warnf(ctx, "[WorkspaceChatStream] 达到最大工具调用轮数 %d，停止循环", MaxToolRounds)
		sendEvent(EventDone, StreamEventDone{
			SessionID: sessionID,
			AgentID:   agentID,
			ToolCalls: previousToolSummaries,
		})
		return nil
	}

		// 获取工作台环境信息（包含目录详情、子节点等）
		workspaceCtx, e := apicall.GetWorkspaceContext(ctx, fullCodePath)
		if e != nil || workspaceCtx == nil {
			return s.handleError(sendEvent, "无效的 full_code_path，无法解析目录", e)
		}
		directoryName := workspaceCtx.Directory.Name
		if directoryName == "" {
			directoryName = workspaceCtx.Directory.Code
		}

		// 构建 LLM 消息（包含 tool 结果，传入已获取的环境信息）
		msgs, tools, e := s.buildLLMMessages(ctx, sessionID, fullCodePath, directoryName, agentID, workspaceCtx)
		if e != nil {
			return s.handleError(sendEvent, e.Error(), e)
		}

	_, client, chatReq, e := s.prepareLLMRequest(ctx, agentID, msgs, tools)
	if e != nil {
		return s.handleError(sendEvent, e.Error(), e)
	}

	stream, e := client.ChatStream(ctx, chatReq)
	if e != nil {
		return s.handleError(sendEvent, "LLM 调用失败", e)
	}

	// 流式处理：复用 processStreamChunks
	content, allToolCalls, err := s.processStreamChunks(ctx, stream, sendEvent)
	if err != nil {
		return err
	}

	// 如果有新的 tool_calls，继续执行工具并递归调用
	if len(allToolCalls) > 0 {
		// 保存 assistant 消息（包含 tool_calls）
		s.saveAssistantMessageWithToolCalls(ctx, sessionID, agentIDPtr, content, allToolCalls, user)

		// 执行工具调用
		toolSummaries := s.executeToolCalls(ctx, allToolCalls, sessionID, fullCodePath, agentIDPtr, user, nil, sendEvent)
		toolSummaries = append(previousToolSummaries, toolSummaries...)

		// 递归调用，继续下一轮
		return s.continueToolCallLoop(ctx, sessionID, fullCodePath, agentID, agentIDPtr, user, round+1, toolSummaries, sendEvent)
	}

	// 保存最终回复并结束
	if err := s.saveAssistantMessage(ctx, sessionID, agentIDPtr, content, user); err != nil {
		logger.Warnf(ctx, "[WorkspaceChatStream] 保存 assistant 消息失败: %v", err)
	}

	sendEvent(EventDone, StreamEventDone{
		SessionID: sessionID,
		AgentID:   agentID,
		ToolCalls: previousToolSummaries,
	})
	return nil
}

// prepareLLMRequest 复用与 function_gen/agent_chat 一致的 LLM 配置逻辑：agent.LLMConfigID -> 否则默认；ExtraConfig、MaxTokens、Temperature、UseThinking
func (s *WorkspaceChatService) prepareLLMRequest(ctx context.Context, agentID int64, msgs []llms.Message, tools []llms.ToolDef) (*model.LLMConfig, llms.LLMClient, *llms.ChatRequest, error) {
	var llmConfig *model.LLMConfig
	var err error

	if agentID > 0 {
		agent, aErr := s.agentRepo.GetByID(agentID)
		if aErr == nil && agent != nil && agent.LLMConfigID > 0 {
			llmConfig, err = s.llmRepo.GetByID(agent.LLMConfigID)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("获取智能体绑定的 LLM 配置失败: %w", err)
			}
		}
	}
	if llmConfig == nil {
		llmConfig, err = s.llmRepo.GetDefault()
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, nil, nil, fmt.Errorf("未设置默认 LLM，请在 LLM 管理中配置")
			}
			return nil, nil, nil, fmt.Errorf("获取 LLM 配置失败: %w", err)
		}
	}

	provider := llms.Provider(llmConfig.Provider)
	opts := llms.DefaultClientOptions()
	if llmConfig.Model != "" {
		opts = opts.WithModel(llmConfig.Model)
	}
	if llmConfig.Timeout > 0 {
		opts = opts.WithTimeout(time.Duration(llmConfig.Timeout) * time.Second)
	}
	if llmConfig.APIBase != "" {
		opts = opts.WithBaseURL(llmConfig.APIBase)
	}
	client, err := llms.NewLLMClientWithOptions(provider, llmConfig.APIKey, opts)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("创建 LLM 客户端失败: %w", err)
	}

	var extraConfig map[string]interface{}
	if llmConfig.ExtraConfig != nil && *llmConfig.ExtraConfig != "" {
		_ = json.Unmarshal([]byte(*llmConfig.ExtraConfig), &extraConfig)
	}
	chatReq := &llms.ChatRequest{
		Messages: msgs,
		Model:    llmConfig.Model,
		Tools:    tools, // 添加工具定义
	}
	if maxTokens, ok := extraConfig["max_tokens"].(float64); ok && maxTokens > 0 {
		chatReq.MaxTokens = int(maxTokens)
	} else if llmConfig.MaxTokens > 0 {
		chatReq.MaxTokens = llmConfig.MaxTokens
	} else {
		chatReq.MaxTokens = 4096
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

// convertToLLMTools 将 dto.ToolDef 转换为 llms.ToolDef（标准格式）
func convertToLLMTools(toolsDesc []dto.ToolDef) []llms.ToolDef {
	llmTools := make([]llms.ToolDef, 0, len(toolsDesc))
	for _, t := range toolsDesc {
		llmTools = append(llmTools, llms.ToolDef{
			Type: "function",
			Function: llms.ToolFunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema, // InputSchema 已经是 JSON Schema 格式
			},
		})
	}
	return llmTools
}

func (s *WorkspaceChatService) buildLLMMessages(ctx context.Context, sessionID, fullCodePath, directoryName string, agentID int64, workspaceCtx *dto.GetWorkspaceContextResp) ([]llms.Message, []llms.ToolDef, error) {
	list, err := s.messageRepo.ListBySessionID(sessionID)
	if err != nil {
		return nil, nil, err
	}
	toolsDesc, _ := s.toolReg.ListTools(ctx)
	llmTools := convertToLLMTools(toolsDesc)

	// 构建环境信息块（使用已传入的环境信息，避免重复调用）
	var workstationCtxBlock strings.Builder
	if workspaceCtx != nil {
		// 获取当前时间信息
		now := time.Now()
		currentTime := now.Format("2006-01-02 15:04:05")
		currentDate := now.Format("2006-01-02")
		timestamp := now.Unix()
		
		workstationCtxBlock.WriteString(fmt.Sprintf(`## 工作环境信息

### 用户信息
- 当前用户：%s

### 时间信息
- 当前时间：%s
- 当前日期：%s
- 时间戳：%d

### 当前工作目录
- 目录名称：%s
- 目录代码：%s
- 完整路径：%s
- 目录类型：%s`, 
			workspaceCtx.User,
			currentTime,
			currentDate,
			timestamp,
			workspaceCtx.Directory.Name,
			workspaceCtx.Directory.Code,
			workspaceCtx.Directory.FullCodePath,
			workspaceCtx.Directory.Type))

		if workspaceCtx.Directory.Description != "" {
			workstationCtxBlock.WriteString(fmt.Sprintf("\n- 目录介绍：%s", workspaceCtx.Directory.Description))
		}

		if len(workspaceCtx.Children) > 0 {
			workstationCtxBlock.WriteString("\n\n### 目录结构\n")
			// 按类型分组显示
			packages := make([]dto.WorkspaceContextNode, 0)
			functions := make([]dto.WorkspaceContextNode, 0)
			for _, child := range workspaceCtx.Children {
				if child.Type == "package" {
					packages = append(packages, child)
				} else if child.Type == "function" {
					functions = append(functions, child)
				}
			}

			if len(packages) > 0 {
				workstationCtxBlock.WriteString("\n**子目录：**\n")
				for _, pkg := range packages {
					workstationCtxBlock.WriteString(fmt.Sprintf("- %s（%s）", pkg.Name, pkg.Code))
					if pkg.Description != "" {
						workstationCtxBlock.WriteString(fmt.Sprintf("：%s", pkg.Description))
					}
					workstationCtxBlock.WriteString("\n")
				}
			}

			if len(functions) > 0 {
				workstationCtxBlock.WriteString("\n**函数/文件：**\n")
				for _, fn := range functions {
					workstationCtxBlock.WriteString(fmt.Sprintf("- %s（%s）", fn.Name, fn.Code))
					if fn.Description != "" {
						workstationCtxBlock.WriteString(fmt.Sprintf("：%s", fn.Description))
					}
					workstationCtxBlock.WriteString("\n")
				}
			}
		} else {
			workstationCtxBlock.WriteString("\n\n### 目录结构\n当前目录下没有子节点。")
		}

		// 添加文件列表（不含代码内容，节省 token）
		if len(workspaceCtx.Files) > 0 {
			workstationCtxBlock.WriteString("\n\n### 代码文件列表\n")
			for _, file := range workspaceCtx.Files {
				workstationCtxBlock.WriteString(fmt.Sprintf("- %s（%s，%d 行）\n", file.RelativePath, file.FileType, file.LineCount))
			}
		}

		workstationCtxBlock.WriteString("\n\n**重要提示：**")
		workstationCtxBlock.WriteString("\n- 以上信息已经包含了当前目录的完整结构（子目录、函数列表、文件列表）")
		workstationCtxBlock.WriteString("\n- 如果用户只是询问项目结构、功能概览等，**不需要**调用 `tree`、`read_dir` 等工具，直接基于上述信息回答即可")
		workstationCtxBlock.WriteString("\n- **只有在需要查看具体代码内容**时，才使用 `read_file` 工具读取文件内容")
		workstationCtxBlock.WriteString("\n- **只有在需要递归查看子目录**时，才使用 `tree` 工具")
		workstationCtxBlock.WriteString("\n\n你可以使用提供的工具来帮助用户完成任务。")
	} else {
		// 降级：使用简化版本
		workstationCtxBlock.WriteString(fmt.Sprintf(`当前工作目录：
- 目录名称：%s
- 完整路径：%s

你可以使用提供的工具来帮助用户完成任务。`, directoryName, fullCodePath))
	}

	var system string
	workstationCtxStr := workstationCtxBlock.String()
	if agentID == 0 {
		system = "你是智能工作台的助手。\n\n" + workstationCtxStr
	} else {
		agent, err := s.agentRepo.GetByID(agentID)
		if err != nil || agent == nil {
			logger.Warnf(ctx, "[WorkspaceChat] 获取智能体失败, agentID=%d: %v，退化为默认工作台助手", agentID, err)
			system = "你是智能工作台的助手。\n\n" + workstationCtxStr
		} else {
			// 知识库：仅当 agent.DocsPaths 解析出路径时拉取；空则无知识库（不默认 /system/official/sdk）
			var knowledgeContent string
			docsPaths := agent.GetDocsPaths()
			if len(docsPaths) > 0 {
				docsResp, dErr := apicall.GetDocsByPaths(ctx, docsPaths)
				if dErr != nil {
					logger.Warnf(ctx, "[WorkspaceChat] GetDocsByPaths 失败, agentID=%d, paths=%v: %v，按无知识库继续", agentID, docsPaths, dErr)
				} else if docsResp != nil && len(docsResp.Docs) > 0 {
					var b strings.Builder
					for _, doc := range docsResp.Docs {
						b.WriteString(fmt.Sprintf("## %s\n%s\n", doc.Name, doc.Content))
					}
					knowledgeContent = b.String()
				}
			}
			template := agent.SystemPromptTemplate
			if template == "" {
				template = "你是智能工作台的助手。"
			}
			system = template
			if knowledgeContent != "" {
				system += "\n\n" + knowledgeContent
			}
			system += "\n\n" + workstationCtxStr
		}
	}

	msgs := []llms.Message{{Role: "system", Content: system}}
	for _, m := range list {
		switch m.Role {
		case RoleUser:
			msgs = append(msgs, llms.Message{Role: RoleUser, Content: m.Content})
		case RoleAssistant:
			// 检查是否有 tool_calls（从 ToolCalls JSON 字段解析）
			msg := llms.Message{Role: RoleAssistant, Content: m.Content}
			if m.ToolCalls != nil && *m.ToolCalls != "" {
				// 解析 tool_calls JSON（如果存在）
				var toolCalls []llms.ToolCall
				if err := json.Unmarshal([]byte(*m.ToolCalls), &toolCalls); err == nil {
					msg.ToolCalls = toolCalls
				}
			}
			msgs = append(msgs, msg)
		case RoleTool:
			// 使用标准的 tool 角色消息
			msgs = append(msgs, llms.Message{
				Role:       RoleTool,
				ToolCallID: m.ToolCallID,
				Content:    m.Content,
			})
		}
	}
	return msgs, llmTools, nil
}

// ListSessions 根据 full_code_path 获取会话列表
func (s *WorkspaceChatService) ListSessions(ctx context.Context, fullCodePath string, page, pageSize int) ([]*dto.WorkspaceSessionItem, int64, error) {
	// 设置默认值
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100 // 限制最大每页数量
	}

	offset := (page - 1) * pageSize
	sessions, total, err := s.sessionRepo.ListByFullCodePath(fullCodePath, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("获取会话列表失败: %w", err)
	}

	// 转换为响应格式
	items := make([]*dto.WorkspaceSessionItem, 0, len(sessions))
	for _, session := range sessions {
		item := &dto.WorkspaceSessionItem{
			SessionID: session.SessionID,
			Title:     session.Title,
			AgentID:   session.AgentID,
			Status:    session.Status,
			CreatedAt: session.CreatedAt,
			UpdatedAt: session.UpdatedAt,
		}
		// 如果有关联的智能体，填充智能体名称
		if session.Agent != nil {
			item.AgentName = session.Agent.Name
		}
		items = append(items, item)
	}

	return items, total, nil
}

// ListMessages 根据 sessionID 获取消息列表
func (s *WorkspaceChatService) ListMessages(ctx context.Context, sessionID string) ([]*model.AgentChatMessage, error) {
	return s.messageRepo.ListBySessionID(sessionID)
}

// mergeToolCalls 合并流式 tool_calls（按 ID 合并分散的 chunk）
func mergeToolCalls(chunkToolCalls []llms.ToolCall, allToolCalls []llms.ToolCall, toolCallsIndex map[string]int) ([]llms.ToolCall, map[string]int) {
	for _, tc := range chunkToolCalls {
		// 情况1：有 ToolCallID，按 ID 匹配
		if tc.ID != "" {
			// 检查是否已存在（同一个 tool_call 可能分散在多个 chunk）
			if idx, ok := toolCallsIndex[tc.ID]; ok {
				// 合并：更新 function.name 和 function.arguments
				if tc.Function.Name != "" {
					allToolCalls[idx].Function.Name = tc.Function.Name
				}
				// arguments 是 JSON 字符串，需要累加（流式输出中可能分块传输）
				if tc.Function.Arguments != "" {
					allToolCalls[idx].Function.Arguments += tc.Function.Arguments
				}
			} else {
				// 新建：直接 append 到 slice（即使 arguments 为空也先创建，后续 chunk 会补充）
				allToolCalls = append(allToolCalls, tc)
				toolCallsIndex[tc.ID] = len(allToolCalls) - 1 // 记录索引
			}
		} else if tc.Function.Arguments != "" {
			// 情况2：ToolCallID 为空，但 Arguments 不为空
			// 这是流式输出中的增量 arguments chunk，应该追加到最后一个 tool_call
			// 注意：在流式输出中，DeepSeek 可能只在第一个 chunk 中发送完整的 tool_call（包括 ID），
			// 后续的 chunks 只发送 arguments 的增量部分，但 ToolCallID 为空
			if len(allToolCalls) > 0 {
				// 追加到最后一个 tool_call 的 arguments
				lastIdx := len(allToolCalls) - 1
				allToolCalls[lastIdx].Function.Arguments += tc.Function.Arguments
			}
			// 如果 allToolCalls 为空，说明这是第一个 chunk，但 ToolCallID 为空，这种情况不应该发生
			// 但为了健壮性，我们仍然处理：如果有 Name，创建一个新的 tool_call
			if len(allToolCalls) == 0 && tc.Function.Name != "" {
				// 这种情况不应该发生，但为了健壮性，我们仍然处理
				allToolCalls = append(allToolCalls, tc)
			}
		} else if tc.Function.Name != "" {
			// 情况3：ToolCallID 为空，Arguments 为空，但 Name 不为空
			// 这可能是新的 tool_call 的开始，但 ToolCallID 还没有发送
			// 这种情况不应该发生，但为了健壮性，我们仍然处理
			if len(allToolCalls) == 0 || allToolCalls[len(allToolCalls)-1].ID != "" {
				// 如果 allToolCalls 为空，或者最后一个 tool_call 已经有 ID，说明这是一个新的 tool_call
				allToolCalls = append(allToolCalls, tc)
			}
		}
		// 情况4：ToolCallID、Arguments、Name 都为空，跳过
	}
	return allToolCalls, toolCallsIndex
}

// executeToolCalls 执行工具调用并保存消息
func (s *WorkspaceChatService) executeToolCalls(
	ctx context.Context,
	allToolCalls []llms.ToolCall,
	sessionID, fullCodePath string,
	agentIDPtr *int64,
	user string,
	files *types.Files,
	sendEvent func(string, interface{}),
) []dto.WorkspaceChatToolCallSummary {
	toolSummaries := make([]dto.WorkspaceChatToolCallSummary, 0, len(allToolCalls))
	logger.Infof(ctx, "[WorkspaceChatStream] 开始执行工具调用 - 工具数量: %d, SessionID: %s", len(allToolCalls), sessionID)

	for i, tc := range allToolCalls {
		logger.Infof(ctx, "[WorkspaceChatStream] [%d/%d] 执行工具调用 - ToolCallID: %s, ToolName: %s, Arguments: %q",
			i+1, len(allToolCalls), tc.ID, tc.Function.Name, tc.Function.Arguments)

		args := s.parseToolCallArgs(ctx, tc)
		var res, st string

		switch tc.Function.Name {
		case "generate_code":
			res, st = s.handleGenerateCode(args)
		case "write_package_code":
			res, st = s.handleWriteFileCode(ctx, sessionID, fullCodePath, args)
		default:
			res, st = s.callOtherTool(ctx, tc.Function.Name, args, fullCodePath, files, i+1, len(allToolCalls))
		}

		toolSummaries = append(toolSummaries, dto.WorkspaceChatToolCallSummary{Name: tc.Function.Name, Status: st})
		sendEvent(EventToolCall, StreamEventToolCall{Name: tc.Function.Name, Status: st})
		s.saveToolMessage(ctx, sessionID, agentIDPtr, tc.ID, tc.Function.Name, res, user)
	}

	successCount, errorCount := 0, 0
	for _, ts := range toolSummaries {
		if ts.Status == ToolCallStatusOK {
			successCount++
		} else {
			errorCount++
		}
	}
	logger.Infof(ctx, "[WorkspaceChatStream] 工具调用执行完成 - 总数量: %d, 成功: %d, 失败: %d, SessionID: %s",
		len(allToolCalls), successCount, errorCount, sessionID)
	return toolSummaries
}

// parseToolCallArgs 解析 tool_call 的 arguments JSON，解析失败时返回空 map
func (s *WorkspaceChatService) parseToolCallArgs(ctx context.Context, tc llms.ToolCall) map[string]interface{} {
	argumentsStr := tc.Function.Arguments
	if argumentsStr == "" {
		argumentsStr = "{}"
		logger.Warnf(ctx, "[WorkspaceChatStream] tool_call arguments 为空，使用空对象，ToolCallID: %s, ToolName: %s", tc.ID, tc.Function.Name)
	}
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argumentsStr), &args); err != nil {
		logger.Warnf(ctx, "[WorkspaceChatStream] 解析 tool_call arguments 失败: %v, ToolCallID: %s, ToolName: %s", err, tc.ID, tc.Function.Name)
		args = make(map[string]interface{})
	}
	return args
}

// handleGenerateCode 处理 generate_code：只记录并提示「输出完成后调用 write_package_code 工具」
func (s *WorkspaceChatService) handleGenerateCode(args map[string]interface{}) (res string, st string) {
	fileName := GetStringArg(args, "file_name")
	if fileName == "" {
		return "generate_code 工具调用失败：缺少必需的参数 file_name。请确保在调用 generate_code 时提供 file_name 参数，例如：{\"file_name\": \"example\"}。", ToolCallStatusError
	}
	return fmt.Sprintf("已记录将生成 %s.go。请在本轮或下一轮输出 markdown 代码块（```go ... ```），输出完成后请调用 write_package_code 工具将代码写入文件（若代码含目录元数据将自动创建子目录并写入）。", fileName), ToolCallStatusOK
}

// handleWriteFileCode 从会话记录中查找「generate_code 之后」的 assistant 消息，解析代码块并写入（按元数据自动创建子目录时由 app-server AddFunctions 处理）
func (s *WorkspaceChatService) handleWriteFileCode(ctx context.Context, sessionID, fullCodePath string, args map[string]interface{}) (res string, st string) {
	fileName := GetStringArg(args, "file_name")
	if fileName == "" {
		return "write_package_code 工具调用失败：缺少必需的参数 file_name。", ToolCallStatusError
	}
	list, err := s.messageRepo.ListBySessionID(sessionID)
	if err != nil {
		return "write_package_code 失败：无法读取会话消息列表。" + err.Error(), ToolCallStatusError
	}
	codeContent := s.findCodeContentAfterGenerateCode(list)
	if codeContent == "" {
		return "write_package_code 失败：未在会话记录中找到「generate_code 之后」的 assistant 消息（含代码块）。请先输出 markdown 代码块再调用 write_package_code。", ToolCallStatusError
	}
	code, err := s.extractCodeFromMarkdown(codeContent)
	if err != nil || code == "" {
		return "write_package_code 失败：在该条 assistant 消息中未找到 markdown 代码块（```go ... ```）。", ToolCallStatusError
	}
	writeArgs := map[string]interface{}{"file_name": fileName, "source_code": code}
	res, isErr := RunAddFunctionsTool(ctx, writeArgs, fullCodePath)
	st = ToolCallStatusOK
	if isErr {
		st = ToolCallStatusError
		logger.Warnf(ctx, "[WorkspaceChatStream] write_package_code 写入文件失败 - FileName: %s, Error: %s", fileName, res)
	} else {
		logger.Infof(ctx, "[WorkspaceChatStream] write_package_code 写入文件成功 - FileName: %s", fileName)
	}
	return res, st
}

// findCodeContentAfterGenerateCode 在消息列表中找最后一条「已记录将生成」的 tool 消息，返回其下一条 assistant 的 content
func (s *WorkspaceChatService) findCodeContentAfterGenerateCode(list []*model.AgentChatMessage) string {
	for j := len(list) - 1; j >= 0; j-- {
		if list[j].Role == RoleTool && strings.Contains(list[j].Content, "已记录将生成") {
			for k := j + 1; k < len(list); k++ {
				if list[k].Role == RoleAssistant && list[k].Content != "" {
					return list[k].Content
				}
			}
			break
		}
	}
	return ""
}

// callOtherTool 调用 ToolRegistry（read_file、read_dir、get_current_time、插件等）
func (s *WorkspaceChatService) callOtherTool(ctx context.Context, name string, args map[string]interface{}, fullCodePath string, files *types.Files, idx, total int) (res string, st string) {
	logger.Infof(ctx, "[WorkspaceChatStream] [%d/%d] 调用工具 - ToolName: %s, FullCodePath: %s", idx, total, name, fullCodePath)
	res, isErr := s.toolReg.CallTool(ctx, name, args, fullCodePath, files)
	st = ToolCallStatusOK
	if isErr {
		st = ToolCallStatusError
		logger.Warnf(ctx, "[WorkspaceChatStream] [%d/%d] 工具调用失败 - ToolName: %s, Error: %s", idx, total, name, res)
	} else if len(res) > 200 {
		logger.Infof(ctx, "[WorkspaceChatStream] [%d/%d] 工具调用成功 - ToolName: %s, ResultLength: %d", idx, total, name, len(res))
	} else {
		logger.Infof(ctx, "[WorkspaceChatStream] [%d/%d] 工具调用成功 - ToolName: %s, Result: %s", idx, total, name, res)
	}
	return res, st
}

// saveToolMessage 保存一条 role=tool 的消息
func (s *WorkspaceChatService) saveToolMessage(ctx context.Context, sessionID string, agentIDPtr *int64, toolCallID, toolName, content, user string) {
	toolMsg := &model.AgentChatMessage{
		SessionID:  sessionID,
		AgentID:    agentIDPtr,
		Role:       RoleTool,
		Content:    content,
		ToolCallID: toolCallID,
		User:       user,
	}
	toolMsg.CreatedBy = user
	toolMsg.UpdatedBy = user
	_ = s.messageRepo.Create(toolMsg)
}

// saveAssistantMessageWithToolCalls 保存 assistant 消息（包含 tool_calls）
func (s *WorkspaceChatService) saveAssistantMessageWithToolCalls(
	ctx context.Context,
	sessionID string,
	agentIDPtr *int64,
	content string,
	allToolCalls []llms.ToolCall,
	user string,
) {
	toolCallsJSON, _ := json.Marshal(allToolCalls)
	toolCallsStr := string(toolCallsJSON)
	asstMsg := &model.AgentChatMessage{
		SessionID: sessionID,
		AgentID:   agentIDPtr,
		Role:      RoleAssistant,
		Content:   content,
		ToolCalls: &toolCallsStr,
		User:      user,
	}
	asstMsg.CreatedBy = user
	asstMsg.UpdatedBy = user
	if err := s.messageRepo.Create(asstMsg); err != nil {
		logger.Warnf(ctx, "[WorkspaceChatStream] 保存 assistant 消息失败: %v", err)
	}
}

// saveAssistantMessage 保存普通 assistant 消息
func (s *WorkspaceChatService) saveAssistantMessage(
	ctx context.Context,
	sessionID string,
	agentIDPtr *int64,
	content string,
	user string,
) error {
	asstMsg := &model.AgentChatMessage{
		SessionID: sessionID,
		AgentID:   agentIDPtr,
		Role:      RoleAssistant,
		Content:   content,
		User:      user,
	}
	asstMsg.CreatedBy = user
	asstMsg.UpdatedBy = user
	return s.messageRepo.Create(asstMsg)
}

// processStreamChunks 处理流式响应：收集 content 和 tool_calls
func (s *WorkspaceChatService) processStreamChunks(
	ctx context.Context,
	stream <-chan *llms.StreamChunk,
	sendEvent func(string, interface{}),
) (string, []llms.ToolCall, error) {
	var buf strings.Builder
	allToolCalls := make([]llms.ToolCall, 0)
	toolCallsIndex := make(map[string]int) // 记录已存在的 tool_call ID 在 slice 中的索引

	for ch := range stream {
		// 检查上下文是否已取消（客户端断开）
		select {
		case <-ctx.Done():
			logger.Infof(ctx, "[WorkspaceChatStream] 上下文已取消，停止处理")
			return "", nil, ctx.Err()
		default:
		}
		if ch.Error != "" {
			// processStreamChunks 内部不能直接使用 handleError（因为返回类型不同），但可以发送事件
			sendEvent(EventError, StreamEventError{Message: "LLM 流式错误: " + ch.Error})
			return "", nil, fmt.Errorf("LLM 流式错误: %s", ch.Error)
		}
		if ch.Content != "" {
			buf.WriteString(ch.Content)
			// 直接转发每个 chunk，保持流畅输出
			sendEvent(EventContent, StreamEventContent{Content: ch.Content})
		}
		// 收集 tool_calls（如果同一个 tool_call 分散在多个 chunk，需要按 ID 合并）
		if len(ch.ToolCalls) > 0 {
			// 调试日志：记录每个 chunk 中的所有 tool_calls
			for _, tc := range ch.ToolCalls {
				logger.Infof(ctx, "[WorkspaceChatStream] 收到 tool_call chunk - ToolCallID: %s, Name: %s, Arguments: %q (长度: %d)", 
					tc.ID, tc.Function.Name, tc.Function.Arguments, len(tc.Function.Arguments))
			}
			allToolCalls, toolCallsIndex = mergeToolCalls(ch.ToolCalls, allToolCalls, toolCallsIndex)
			
			// 调试日志：记录合并后的状态
			for _, tc := range allToolCalls {
				if tc.Function.Name == "generate_code" {
					logger.Infof(ctx, "[WorkspaceChatStream] 合并后 generate_code - ToolCallID: %s, Arguments: %q (长度: %d)", 
						tc.ID, tc.Function.Arguments, len(tc.Function.Arguments))
				}
			}
		}
		// 不在这里 break：generate_code 调用后，代码内容可能在后续 chunk 中才流式输出，
		// 必须等流式输出全部结束（channel 关闭）后再提取代码，否则会「未找到代码块」
		// if ch.Done { break } 已移除，依赖 for range stream 自然结束
	}

	content := strings.TrimSpace(buf.String())
	
	// 调试日志：检查 tool_calls 的 arguments 是否完整
	for _, tc := range allToolCalls {
		if tc.Function.Name == "generate_code" {
			logger.Infof(ctx, "[WorkspaceChatStream] 流式输出完成 - generate_code ToolCallID: %s, Arguments: %q (长度: %d), Content长度: %d", 
				tc.ID, tc.Function.Arguments, len(tc.Function.Arguments), len(content))
		} else if tc.Function.Arguments == "" {
			logger.Warnf(ctx, "[WorkspaceChatStream] tool_call arguments 为空，ToolCallID: %s, ToolName: %s", tc.ID, tc.Function.Name)
		}
	}
	
	return content, allToolCalls, nil
}

// getAgentID 统一获取 agentID：优先使用 session.AgentID，如果为0则使用请求中的 req.AgentID
func (s *WorkspaceChatService) getAgentID(sessionAgentID *int64, reqAgentID int64) int64 {
	if sessionAgentID != nil && *sessionAgentID > 0 {
		return *sessionAgentID
	}
	if reqAgentID > 0 {
		return reqAgentID
	}
	return 0
}

// getAgentIDPtr 根据 agentID 创建指针（用于保存消息时使用）
func (s *WorkspaceChatService) getAgentIDPtr(agentID int64) *int64 {
	if agentID == 0 {
		return nil
	}
	return &agentID
}

// handleError 统一错误处理：发送错误事件并返回错误
func (s *WorkspaceChatService) handleError(sendEvent func(string, interface{}), message string, err error) error {
	sendEvent(EventError, StreamEventError{Message: message})
	if err != nil {
		return fmt.Errorf("%s: %w", message, err)
	}
	return fmt.Errorf("%s", message)
}

// extractCodeFromMarkdown 从 markdown 格式的文本中提取代码块（复用 agent_chat_service 的逻辑）
func (s *WorkspaceChatService) extractCodeFromMarkdown(content string) (string, error) {
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
		return strings.TrimSpace(codeBlocks[0]), nil
	}

	// 如果没有找到代码块，返回错误
	return "", fmt.Errorf("未找到 markdown 代码块（```go ... ``` 或 ``` ... ```）")
}
