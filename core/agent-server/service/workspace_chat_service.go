package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/prompt"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/streamloop"
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
	MaxToolRounds   = 30 // 与 streamloop.MaxToolRounds 保持一致，仅作注释/文档用，实际以 streamloop 为准
)

// 工作台操作提示词在 core/agent-server/prompt/content/doc/ 下，由 //go:embed content 嵌入，通过 prompt 包加载（见 prompt.ReadContent / init）

// 消息角色常量
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// 工具调用状态常量
const (
	ToolCallStatusOK        = "ok"
	ToolCallStatusError     = "error"
	ToolCallStatusRunning   = "running"   // 工具正在执行，用于流式反馈到前端
	ToolCallStatusStreaming = "streaming" // LLM 流式输出 tool_call（name/arguments 逐段到达），推送到前端实时展示
)

// 流式事件类型常量
const (
	EventSession         = "session"
	EventAgentID         = "agent_id"
	EventToolCall        = "tool_call"
	EventToolCallsStream = "tool_calls_stream" // LLM 流式输出的 tool_calls 当前列表（name+arguments），前端实时展示
	EventContent         = "content"
	EventDone            = "done"
	EventError           = "error"
)

// WorkspaceChatService 工作台对话编排：会话、历史、LLM、Tool 循环；复用 prepareLLMRequest 逻辑与 pkg/llms ChatStream
type WorkspaceChatService struct {
	toolReg     *ToolRegistry
	modeRepo    *repository.WorkspaceModeRepository
	sessionRepo *repository.ChatSessionRepository
	messageRepo *repository.ChatMessageRepository
	llmRepo     *repository.LLMRepository
	agentRepo   *repository.AgentRepository
}

// NewWorkspaceChatService 创建 WorkspaceChatService
func NewWorkspaceChatService(
	toolReg *ToolRegistry,
	modeRepo *repository.WorkspaceModeRepository,
	sessionRepo *repository.ChatSessionRepository,
	messageRepo *repository.ChatMessageRepository,
	llmRepo *repository.LLMRepository,
	agentRepo *repository.AgentRepository,
) *WorkspaceChatService {
	return &WorkspaceChatService{
		toolReg:     toolReg,
		modeRepo:    modeRepo,
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
	Name      string `json:"name"`
	Status    string `json:"status"`    // ok / error / running / streaming
	Arguments string `json:"arguments"` // 流式或最终参数（streaming 时逐段推送，供前端实时展示）
	Result    string `json:"result"`    // 工具返回结果（status=ok 时可选）
	Error     string `json:"error"`    // 错误信息（status=error 时可选）
}

// StreamEventToolCallsStream 流式 tool_calls 列表（当前已合并的全部 tool_call，供前端实时展示）
type StreamEventToolCallsStream struct {
	ToolCalls []StreamEventToolCallItem `json:"tool_calls"`
}

// StreamEventToolCallItem 流式单项（仅 name + arguments）
type StreamEventToolCallItem struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
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
	workspaceCtx, e := apicall.GetWorkspaceContext(ctx, fullCodePath, "")
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

	// 解析模式：按 req.Mode（code）查库，未传或查不到则用 dev
	modeCode := strings.TrimSpace(req.Mode)
	if modeCode == "" {
		modeCode = "dev"
	}
	mode, _ := s.modeRepo.GetByCode(modeCode)
	if mode == nil {
		mode, _ = s.modeRepo.GetByCode("dev")
	}
	var toolNames []string
	var systemPromptFragment string
	var modeProvider prompt.WorkspaceModePromptProvider
	if mode != nil {
		modeProvider = prompt.GetModeProvider(mode.Code)
		if modeProvider == nil {
			toolNames = mode.GetToolNames()
			systemPromptFragment = strings.TrimSpace(mode.SystemPromptFragment)
		}
	}

	// 有效 agentID：模式绑定了智能体则用模式的，否则用 session/req
	agentID := s.getAgentID(session.AgentID, req.AgentID)
	if mode != nil && mode.AgentID != nil && *mode.AgentID > 0 {
		agentID = *mode.AgentID
	}
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

	deps := &workspaceStreamLoopDeps{
		ctx:                  ctx,
		sendEvent:            sendEvent,
		sessionID:            sessionID,
		fullCodePath:         fullCodePath,
		agentID:              agentID,
		agentIDPtr:           agentIDPtr,
		user:                 user,
		modeProvider:         modeProvider,
		toolNames:            toolNames,
		systemPromptFragment: systemPromptFragment,
		files:                req.Message.Files,
		service:              s,
	}
	return streamloop.RunStreamLoop(ctx, deps)
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

// workspaceCtxToEnvInput 将 dto 工作台上下文转为 prompt 包的环境输入，供 BuildWorkspaceEnvData 使用
func workspaceCtxToEnvInput(c *dto.GetWorkspaceContextResp) *prompt.WorkspaceEnvInput {
	if c == nil {
		return nil
	}
	dirDesc := ""
	if c.Directory.Description != "" {
		dirDesc = "\n- 目录介绍：" + c.Directory.Description
	}
	children := make([]prompt.WorkspaceEnvNode, 0, len(c.Children))
	for _, n := range c.Children {
		children = append(children, prompt.WorkspaceEnvNode{
			Name:         n.Name,
			Code:         n.Code,
			Description:  n.Description,
			Type:         n.Type,
			FullCodePath: n.FullCodePath,
			TemplateType: n.TemplateType,
		})
	}
	files := make([]prompt.WorkspaceEnvFile, 0, len(c.Files))
	for _, f := range c.Files {
		files = append(files, prompt.WorkspaceEnvFile{
			RelativePath: f.RelativePath,
			FileType:     f.FileType,
			LineCount:    f.LineCount,
		})
	}
	return &prompt.WorkspaceEnvInput{
		User:           c.User,
		DirName:        c.Directory.Name,
		DirCode:        c.Directory.Code,
		FullCodePath:   c.Directory.FullCodePath,
		DirType:        c.Directory.Type,
		DirDescription: dirDesc,
		Children:       children,
		Files:          files,
	}
}

func (s *WorkspaceChatService) buildLLMMessages(ctx context.Context, sessionID, fullCodePath, directoryName string, agentID int64, workspaceCtx *dto.GetWorkspaceContextResp, modeProvider prompt.WorkspaceModePromptProvider, fallbackToolNames []string, fallbackSystemPrompt string) ([]llms.Message, []llms.ToolDef, error) {
	list, err := s.messageRepo.ListBySessionID(sessionID)
	if err != nil {
		return nil, nil, err
	}
	var toolNames []string
	var systemPromptFragment string
	var firstAssistantContent string
	if modeProvider != nil {
		toolNames = modeProvider.ToolNames()
		systemPromptFragment = "" // 在 env 填好后由 provider.SystemPrompt(data) 产出
		firstAssistantContent = modeProvider.FirstAssistantContent()
	} else {
		toolNames = fallbackToolNames
		systemPromptFragment = fallbackSystemPrompt
	}
	toolsDesc, _ := s.toolReg.ListTools(ctx, toolNames)
	llmTools := convertToLLMTools(toolsDesc)

	// 环境数据与 env 块统一由 prompt 包构建，避免在此处手写 ChildrenSection/FilesSection/DirectoryList
	now := time.Now()
	var envInput *prompt.WorkspaceEnvInput
	if workspaceCtx != nil {
		envInput = workspaceCtxToEnvInput(workspaceCtx)
	}
	data := prompt.BuildWorkspaceEnvData(envInput, directoryName, fullCodePath, now)
	envBlock := prompt.BuildWorkspaceEnvBlock(data, workspaceCtx != nil, directoryName, fullCodePath)

	// 模式专属提示：多态时由 provider 产出，否则用 fallback 的 systemPromptFragment
	if modeProvider != nil {
		systemPromptFragment = modeProvider.SystemPrompt(data)
	}

	var system string
	if agentID == 0 {
		system = "你是智能工作台的助手。\n\n" + envBlock
	} else {
		agent, err := s.agentRepo.GetByID(agentID)
		if err != nil || agent == nil {
			logger.Warnf(ctx, "[WorkspaceChat] 获取智能体失败, agentID=%d: %v，退化为默认工作台助手", agentID, err)
			system = "你是智能工作台的助手。\n\n" + envBlock
		} else {
			template := agent.SystemPromptTemplate
			if template == "" {
				template = "你是智能工作台的助手。"
			}
			system = template + "\n\n" + envBlock
		}
	}
	// 模式专属补充提示（拼在 Agent 提示后）
	if systemPromptFragment != "" {
		system += "\n\n" + systemPromptFragment
	}
	// 操作提示词：有 modeProvider 时用其 OperationPrompt（可为空）；无 modeProvider 时用公用 WorkspacePrompt（doc/工作台操作提示词.md）。mode 未提供 OperationPrompt 时不再追加（规则已在 system_prompt 中）
	var operationPrompt string
	if modeProvider != nil {
		operationPrompt = strings.TrimSpace(modeProvider.OperationPrompt())
	} else {
		operationPrompt = strings.TrimSpace(prompt.WorkspacePrompt)
	}
	if operationPrompt != "" {
		system += "\n\n" + operationPrompt
	}

	msgs := []llms.Message{{Role: "system", Content: system}}
	// 首条 assistant：多态时由 provider 提供，会话开始时注入
	if firstAssistantContent != "" {
		msgs = append(msgs, llms.Message{Role: RoleAssistant, Content: firstAssistantContent})
	}
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

// executeToolCalls 执行工具调用并保存消息。
func (s *WorkspaceChatService) executeToolCalls(
	ctx context.Context,
	allToolCalls []llms.ToolCall,
	_ string, // currentAssistantContent 保留参数以兼容调用方，已不再用于 <var> 解析
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

		sendEvent(EventToolCall, StreamEventToolCall{Name: tc.Function.Name, Status: ToolCallStatusRunning, Arguments: tc.Function.Arguments})

		args := s.parseToolCallArgs(ctx, tc)
		res, st := s.callOtherTool(ctx, tc.Function.Name, args, fullCodePath, files, i+1, len(allToolCalls))

		resultStr, errStr := "", ""
		if st == ToolCallStatusOK {
			resultStr = res
		} else {
			errStr = res
		}
		toolSummaries = append(toolSummaries, dto.WorkspaceChatToolCallSummary{
			Name: tc.Function.Name, Status: st, Arguments: tc.Function.Arguments, Result: resultStr, Error: errStr,
		})
		sendEvent(EventToolCall, StreamEventToolCall{
			Name: tc.Function.Name, Status: st, Arguments: tc.Function.Arguments, Result: resultStr, Error: errStr,
		})
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

// callOtherTool 调用 ToolRegistry（read_go_file、read_doc、read_dir、write_doc、write_go_file、build_workspace、create_directory、插件等）
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
) error {
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
		return err
	}
	return nil
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
