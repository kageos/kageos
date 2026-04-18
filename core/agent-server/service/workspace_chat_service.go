package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
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
	MaxToolRounds   = 100 // 与 streamloop.MaxToolRounds 保持一致，仅作注释/文档用，实际以 streamloop 为准
)

// 工作台系统提示词与文档统一来自 /system/prompt；运行时优先读树，缺失时回落到本地 seed。

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
	EventToolCall        = "tool_call"
	EventToolCallsStream = "tool_calls_stream" // LLM 流式输出的 tool_calls 当前列表（name+arguments），前端实时展示
	EventContent         = "content"
	EventDone            = "done"
	EventError           = "error"
)

const filesInstruction = "以上 <files> 标签中的 JSON 为本轮用户上传的文件数据，可作为表单函数的 files 参数使用。"

// buildUserMessageContentWithFiles 当用户上传了文件时，将消息内容格式化为：
// <files>\n{JSON}\n</files> + 说明 + 用户需求，便于 Agent 把 <files> 内的 JSON 当作 run_form_submit 的 input_files 使用。
// 仅用于拼装发给 LLM 的完整内容，不入库。
func buildUserMessageContentWithFiles(files *types.Files, userContent string) string {
	if files == nil || len(files.Files) == 0 {
		return userContent
	}
	raw, err := json.Marshal(files)
	if err != nil {
		return userContent
	}
	demand := strings.TrimSpace(userContent)
	if demand == "" {
		demand = "用户需求：请处理上述文件"
	}
	return "<files>\n" + string(raw) + "\n</files>\n\n" + filesInstruction + "\n\n" + demand
}

// userContentForStorage 入库用：只保留用户文字到 Content，文件单独到 Files（JSON）。
// 返回 (content, filesJSON)；无文件时 filesJSON 为 nil。
func userContentForStorage(files *types.Files, userContent string) (content string, filesJSON *string) {
	demand := strings.TrimSpace(userContent)
	if files == nil || len(files.Files) == 0 {
		return demand, nil
	}
	raw, err := json.Marshal(files)
	if err != nil {
		return demand, nil
	}
	if demand == "" {
		demand = "用户需求：请处理上述文件"
	}
	s := string(raw)
	return demand, &s
}

// userContentForLLM 从库中取出的 user 消息：若有 Files 则拼出完整内容（<files>+说明+content）供 LLM 使用。
func userContentForLLM(content string, filesJSON *string) string {
	if filesJSON == nil || *filesJSON == "" {
		return content
	}
	demand := strings.TrimSpace(content)
	if demand == "" {
		demand = "用户需求：请处理上述文件"
	}
	return "<files>\n" + *filesJSON + "\n</files>\n\n" + filesInstruction + "\n\n" + demand
}

// WorkspaceChatService 工作台对话编排：会话、历史、LLM、Tool 循环；只认 LLM + 单模式（dev）
type WorkspaceChatService struct {
	toolReg     *ToolRegistry
	sessionRepo *repository.ChatSessionRepository
	messageRepo *repository.ChatMessageRepository
	llmRepo     *repository.LLMRepository

	// runningCancels 维护「正在执行的 session → cancelFunc」映射，供手动取消使用
	runningCancels sync.Map // key: sessionID (string), value: context.CancelFunc
	// sseConnections 维护「有活跃 SSE 连接的 session」，供前端存活检测，避免无谓轮询大消息列表
	sseConnections sync.Map // key: sessionID (string), value: struct{}
}

// NewWorkspaceChatService 创建 WorkspaceChatService
func NewWorkspaceChatService(
	toolReg *ToolRegistry,
	sessionRepo *repository.ChatSessionRepository,
	messageRepo *repository.ChatMessageRepository,
	llmRepo *repository.LLMRepository,
) *WorkspaceChatService {
	return &WorkspaceChatService{
		toolReg:     toolReg,
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
		llmRepo:     llmRepo,
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

// StreamEventToolCall tool_call 事件数据
type StreamEventToolCall struct {
	Name       string      `json:"name"`
	Status     string      `json:"status"`                // ok / error / running / streaming
	Arguments  string      `json:"arguments"`             // 流式或最终参数（streaming 时逐段推送，供前端实时展示）
	Result     string      `json:"result"`                // 工具返回结果（status=ok 时可选）
	ResultData interface{} `json:"result_data,omitempty"` // 结构化工具结果（供前端直接消费）
	Error      string      `json:"error"`                 // 错误信息（status=error 时可选）
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
			default:
			}
		}
	}()
	// 非阻塞发送事件：写不进 eventChan 时直接丢弃（刷新后无人读也不会卡死 goroutine）
	sendEvent := func(event string, data interface{}) {
		select {
		case eventChan <- StreamEvent{Event: event, Data: data}:
		default:
			// channel 满或无人读，丢弃该事件（事件已存 DB，前端可通过轮询消息恢复）
		}
	}
	user := contextx.GetRequestUser(ctx)
	fullCodePath := strings.TrimSpace(req.FullCodePath)
	if fullCodePath == "" {
		return s.handleError(sendEvent, "full_code_path 必填", nil)
	}
	requestedModeCode := normalizeWorkspaceModeCode(req.ModeCode)
	requestedModeProvider := prompt.ResolveModeProvider(ctx, requestedModeCode)
	if req.SessionID == "" && requestedModeProvider == nil {
		return s.handleError(sendEvent, fmt.Sprintf("不支持的 mode_code: %s", requestedModeCode), nil)
	}

	// 1) 获取工作台环境信息（包含目录详情、子节点等，一次性获取，避免重复调用）
	workspaceCtx, e := apicall.GetWorkspaceContext(ctx, fullCodePath, "")
	if e != nil || workspaceCtx == nil {
		return s.handleError(sendEvent, "无效的 full_code_path，无法解析目录", e)
	}
	directoryName := workspaceCtx.Directory.Name
	if directoryName == "" {
		directoryName = workspaceCtx.Directory.Code
	}

	// 2) 解析或创建 session
	var session *model.AgentChatSession
	if req.SessionID != "" {
		var e error
		session, e = s.sessionRepo.GetBySessionID(req.SessionID)
		if e != nil || session == nil {
			return s.handleError(sendEvent, fmt.Sprintf("会话不存在: %s", req.SessionID), e)
		}
	} else {
		session = &model.AgentChatSession{
			TreeID:       workspaceCtx.Directory.ID,
			FullCodePath: fullCodePath,
			Source:       SourceWorkspace,
			SessionID:    uuid.New().String(),
			AgentID:      nil,
			Title:        "",
			ModeCode:     requestedModeCode,
			Status:       model.ChatSessionStatusActive,
			User:         user,
		}
		session.CreatedBy = user
		session.UpdatedBy = user
		if e := s.sessionRepo.Create(session); e != nil {
			return s.handleError(sendEvent, "创建会话失败", e)
		}
	}
	modeCode := requestedModeCode
	if req.SessionID != "" && strings.TrimSpace(req.ModeCode) == "" {
		modeCode = normalizeWorkspaceModeCode(session.ModeCode)
	}
	if session.ModeCode != modeCode {
		session.ModeCode = modeCode
	}
	modeProvider := requestedModeProvider
	if modeCode != requestedModeCode || modeProvider == nil {
		modeProvider = prompt.ResolveModeProvider(ctx, modeCode)
	}
	var toolNames []string
	var systemPromptFragment string
	if modeProvider == nil {
		return s.handleError(sendEvent, fmt.Sprintf("不支持的 mode_code: %s", modeCode), nil)
	}

	sessionID := session.SessionID
	s.sseConnections.Store(sessionID, struct{}{})

	// ⭐ 标记会话为 generating（后台执行中）
	session.Status = model.ChatSessionStatusGenerating
	session.UpdatedBy = user
	if e := s.sessionRepo.Update(session); e != nil {
		logger.Warnf(ctx, "[WorkspaceChatStream] 标记 generating 失败: %v", e)
	}

	// ⭐ 创建可取消的 context 并注册到 runningCancels，供 CancelSession 使用
	runCtx, runCancel := context.WithCancel(ctx)
	s.runningCancels.Store(sessionID, runCancel)
	// 无论正常结束还是异常，都要恢复状态并清理
	defer func() {
		runCancel()
		s.runningCancels.Delete(sessionID)
		s.sseConnections.Delete(sessionID)
		// ⭐ 恢复会话状态：已取消的保持 cancelled，否则标回 active
		latest, e := s.sessionRepo.GetBySessionID(sessionID)
		if e == nil && latest != nil && latest.Status == model.ChatSessionStatusGenerating {
			latest.Status = model.ChatSessionStatusActive
			latest.UpdatedBy = user
			if e := s.sessionRepo.Update(latest); e != nil {
				logger.Warnf(ctx, "[WorkspaceChatStream] 恢复 active 失败: %v", e)
			}
		}
	}()

	llmConfigID := req.LLMConfigID

	// 3) 保存 user 消息
	storageContent, storageFiles := userContentForStorage(req.Message.Files, req.Message.Content)
	userMsg := &model.AgentChatMessage{
		SessionID: sessionID, AgentID: nil, Role: RoleUser,
		Content: storageContent, Files: storageFiles, User: user,
	}
	userMsg.CreatedBy = user
	userMsg.UpdatedBy = user
	if e := s.messageRepo.Create(userMsg); e != nil {
		return s.handleError(sendEvent, "保存用户消息失败", e)
	}

	// 3.1) 新会话自动生成标题
	if session.Title == "" {
		title := strings.TrimSpace(req.Message.Content)
		title = strings.ReplaceAll(title, "\n", " ")
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

	deps := &workspaceStreamLoopDeps{
		ctx:                  runCtx,
		sendEvent:            sendEvent,
		sessionID:            sessionID,
		fullCodePath:         fullCodePath,
		llmConfigID:          llmConfigID,
		user:                 user,
		modeProvider:         modeProvider,
		toolNames:            toolNames,
		systemPromptFragment: systemPromptFragment,
		files:                req.Message.Files,
		service:              s,
	}
	return streamloop.RunStreamLoop(runCtx, deps)
}

// CancelSession 手动取消正在执行的会话
func (s *WorkspaceChatService) CancelSession(ctx context.Context, sessionID string) error {
	session, err := s.sessionRepo.GetBySessionID(sessionID)
	if err != nil {
		return fmt.Errorf("会话不存在: %w", err)
	}
	if session.Status != model.ChatSessionStatusGenerating {
		return fmt.Errorf("会话未在执行中（当前状态: %s）", session.Status)
	}

	// 标记为已取消
	session.Status = model.ChatSessionStatusCancelled
	user := contextx.GetRequestUser(ctx)
	session.UpdatedBy = user
	if err := s.sessionRepo.Update(session); err != nil {
		return fmt.Errorf("更新会话状态失败: %w", err)
	}

	// 触发 cancelFunc，让 streamloop 尽快退出
	if cancelFn, ok := s.runningCancels.LoadAndDelete(sessionID); ok {
		cancelFn.(context.CancelFunc)()
		logger.Infof(ctx, "[WorkspaceChatStream] 会话已取消 - SessionID: %s", sessionID)
	}
	return nil
}

// IsSSEConnected 检查该 session 是否仍有活跃的 SSE 连接（供前端存活检测，SSE 存活则不轮询大消息列表）
func (s *WorkspaceChatService) IsSSEConnected(sessionID string) bool {
	_, ok := s.sseConnections.Load(sessionID)
	return ok
}

// ListRunningSessions 查询当前用户所有正在执行的工作台会话
func (s *WorkspaceChatService) ListRunningSessions(ctx context.Context) ([]*dto.WorkspaceSessionItem, error) {
	user := contextx.GetRequestUser(ctx)
	sessions, err := s.sessionRepo.ListRunningByUser(user)
	if err != nil {
		return nil, fmt.Errorf("查询执行中会话失败: %w", err)
	}
	items := make([]*dto.WorkspaceSessionItem, 0, len(sessions))
	for _, session := range sessions {
		items = append(items, &dto.WorkspaceSessionItem{
			SessionID:    session.SessionID,
			Title:        session.Title,
			User:         session.User,
			AgentID:      session.AgentID,
			ModeCode:     normalizeWorkspaceModeCode(session.ModeCode),
			Status:       session.Status,
			FullCodePath: session.FullCodePath,
			CreatedAt:    session.CreatedAt,
			UpdatedAt:    session.UpdatedAt,
		})
	}
	return items, nil
}

// ListFinishedSessions 查询当前用户最近已结束的工作台会话
func (s *WorkspaceChatService) ListFinishedSessions(ctx context.Context, limit int) ([]*dto.WorkspaceSessionItem, error) {
	if limit <= 0 {
		limit = 20
	}
	user := contextx.GetRequestUser(ctx)
	sessions, err := s.sessionRepo.ListFinishedByUser(user, limit)
	if err != nil {
		return nil, fmt.Errorf("查询已结束会话失败: %w", err)
	}
	items := make([]*dto.WorkspaceSessionItem, 0, len(sessions))
	for _, session := range sessions {
		items = append(items, &dto.WorkspaceSessionItem{
			SessionID:    session.SessionID,
			Title:        session.Title,
			User:         session.User,
			AgentID:      session.AgentID,
			Status:       session.Status,
			FullCodePath: session.FullCodePath,
			CreatedAt:    session.CreatedAt,
			UpdatedAt:    session.UpdatedAt,
		})
	}
	return items, nil
}

// prepareLLMRequest 工作台只认 LLM：llmConfigID > 0 用该配置，否则用默认
func (s *WorkspaceChatService) prepareLLMRequest(ctx context.Context, llmConfigID int64, msgs []llms.Message, tools []llms.ToolDef) (*model.LLMConfig, llms.LLMClient, *llms.ChatRequest, error) {
	var llmConfig *model.LLMConfig
	var err error

	if llmConfigID > 0 {
		llmConfig, err = s.llmRepo.GetByID(llmConfigID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("获取 LLM 配置失败: %w", err)
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
			Callbacks:    n.Callbacks,
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
		User:                   c.User,
		DepartmentFullPath:     c.DepartmentFullPath,
		DepartmentFullNamePath: c.DepartmentFullNamePath,
		DirName:                c.Directory.Name,
		DirCode:                c.Directory.Code,
		FullCodePath:           c.Directory.FullCodePath,
		DirType:                c.Directory.Type,
		DirDescription:         dirDesc,
		PublishedToHub:         c.Directory.PublishedToHub,
		HubFullCodePath:        c.Directory.HubFullCodePath,
		Children:               children,
		Files:                  files,
	}
}

func normalizeWorkspaceModeCode(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return "dev"
	}
	return code
}

func (s *WorkspaceChatService) buildLLMMessages(ctx context.Context, sessionID, fullCodePath, directoryName string, workspaceCtx *dto.GetWorkspaceContextResp, modeProvider prompt.WorkspaceModePromptProvider, fallbackToolNames []string, fallbackSystemPrompt string) ([]llms.Message, []llms.ToolDef, error) {
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

	// 环境数据与 env 块统一由 prompt 包构建
	now := time.Now()
	var envInput *prompt.WorkspaceEnvInput
	if workspaceCtx != nil {
		envInput = workspaceCtxToEnvInput(workspaceCtx)
	}
	catalog := prompt.LoadPromptDocCatalog(ctx)
	data := prompt.BuildWorkspaceEnvDataWithCatalog(envInput, directoryName, fullCodePath, now, catalog)
	envTemplate := prompt.LoadWorkspaceEnvTemplate(ctx)
	envBlock := prompt.BuildWorkspaceEnvBlockWithTemplate(envTemplate, data, workspaceCtx != nil, directoryName, fullCodePath)

	if modeProvider != nil {
		systemPromptFragment = modeProvider.SystemPrompt(data)
	}

	// 工作台固定系统提示词（不再依赖智能体）
	system := "你是智能工作台的助手。\n\n" + envBlock
	if systemPromptFragment != "" {
		system += "\n\n" + systemPromptFragment
	}
	// 额外操作提示词：仅 modeProvider 明确提供时才追加；默认所有规则都放在 system_prompt 与子文档里。
	var operationPrompt string
	if modeProvider != nil {
		operationPrompt = strings.TrimSpace(modeProvider.OperationPrompt())
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
			userContent := userContentForLLM(m.Content, m.Files)
			msgs = append(msgs, llms.Message{Role: RoleUser, Content: userContent})
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

	items := make([]*dto.WorkspaceSessionItem, 0, len(sessions))
	for _, session := range sessions {
		items = append(items, &dto.WorkspaceSessionItem{
			SessionID:    session.SessionID,
			Title:        session.Title,
			User:         session.User,
			AgentID:      session.AgentID,
			Status:       session.Status,
			FullCodePath: session.FullCodePath,
			CreatedAt:    session.CreatedAt,
			UpdatedAt:    session.UpdatedAt,
		})
	}

	return items, total, nil
}

// ListMessages 根据 sessionID 获取消息列表
func (s *WorkspaceChatService) ListMessages(ctx context.Context, sessionID string) ([]*model.AgentChatMessage, error) {
	return s.messageRepo.ListBySessionID(sessionID)
}

// executeToolCalls 执行工具调用并保存消息。若 tool 消息保存失败则返回 error，不再进入下一轮，避免 400 insufficient tool messages。
func (s *WorkspaceChatService) executeToolCalls(
	ctx context.Context,
	allToolCalls []llms.ToolCall,
	_ string, // currentAssistantContent 保留参数以兼容调用方，已不再用于 <var> 解析
	sessionID, fullCodePath string,
	agentIDPtr *int64,
	user string,
	files *types.Files,
	sendEvent func(string, interface{}),
) ([]dto.WorkspaceChatToolCallSummary, error) {
	// 注入 session_id 到 context，供 record_workspace_event 等工具追溯
	ctx = context.WithValue(ctx, WorkspaceSessionIDKey, sessionID)
	toolSummaries := make([]dto.WorkspaceChatToolCallSummary, 0, len(allToolCalls))
	logger.Infof(ctx, "[WorkspaceChatStream] 开始执行工具调用 - 工具数量: %d, SessionID: %s", len(allToolCalls), sessionID)

	for i, tc := range allToolCalls {
		logger.Infof(ctx, "[WorkspaceChatStream] [%d/%d] 执行工具调用 - ToolCallID: %s, ToolName: %s, Arguments: %q",
			i+1, len(allToolCalls), tc.ID, tc.Function.Name, tc.Function.Arguments)

		sendEvent(EventToolCall, StreamEventToolCall{Name: tc.Function.Name, Status: ToolCallStatusRunning, Arguments: tc.Function.Arguments})

		args := s.parseToolCallArgs(ctx, tc)
		toolRes, st := s.callOtherTool(ctx, tc.Function.Name, args, fullCodePath, files, i+1, len(allToolCalls))

		resultStr, errStr := "", ""
		var resultData interface{}
		if st == ToolCallStatusOK {
			resultStr = toolRes.Content
			resultData = toolRes.Data
		} else {
			errStr = toolRes.Content
		}
		toolSummaries = append(toolSummaries, dto.WorkspaceChatToolCallSummary{
			Name: tc.Function.Name, Status: st, Arguments: tc.Function.Arguments, Result: resultStr, ResultData: resultData, Error: errStr,
		})
		sendEvent(EventToolCall, StreamEventToolCall{
			Name: tc.Function.Name, Status: st, Arguments: tc.Function.Arguments, Result: resultStr, ResultData: resultData, Error: errStr,
		})
		if err := s.saveToolMessage(ctx, sessionID, agentIDPtr, tc.ID, tc.Function.Name, st, toolRes, user); err != nil {
			logger.Warnf(ctx, "[WorkspaceChatStream] 保存 tool 消息失败 ToolCallID=%s: %v（若为 Error 1366 请将表转为 utf8mb4）", tc.ID, err)
			return toolSummaries, fmt.Errorf("保存 tool 消息失败: %w", err)
		}
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
	return toolSummaries, nil
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
func (s *WorkspaceChatService) callOtherTool(ctx context.Context, name string, args map[string]interface{}, fullCodePath string, files *types.Files, idx, total int) (res ToolResult, st string) {
	logger.Infof(ctx, "[WorkspaceChatStream] [%d/%d] 调用工具 - ToolName: %s, FullCodePath: %s", idx, total, name, fullCodePath)
	result := s.toolReg.CallTool(ctx, name, args, fullCodePath, files)
	isErr := result.IsError
	st = ToolCallStatusOK
	if isErr {
		result.Content = appendExecuteGuideHint(name, result.Content)
		st = ToolCallStatusError
		logger.Warnf(ctx, "[WorkspaceChatStream] [%d/%d] 工具调用失败 - ToolName: %s, Error: %s", idx, total, name, result.Content)
	} else if len(result.Content) > 200 {
		logger.Infof(ctx, "[WorkspaceChatStream] [%d/%d] 工具调用成功 - ToolName: %s, ResultLength: %d", idx, total, name, len(result.Content))
	} else {
		logger.Infof(ctx, "[WorkspaceChatStream] [%d/%d] 工具调用成功 - ToolName: %s, Result: %s", idx, total, name, result.Content)
	}
	return result, st
}

const executeGuideDocPath = "/system/prompt/workspace/execute"

func shouldSuggestExecuteGuide(toolName string) bool {
	switch strings.TrimSpace(toolName) {
	case "run_table_search", "run_table_create", "run_table_update", "run_form_submit", "run_chart_query", "run_on_select_fuzzy":
		return true
	default:
		return false
	}
}

func appendExecuteGuideHint(toolName, message string) string {
	if !shouldSuggestExecuteGuide(toolName) || strings.Contains(message, executeGuideDocPath) {
		return message
	}
	hint := fmt.Sprintf("提示：如需该工具的 SOP、参数规范和易错点，可先 `read_doc(\"%s\")` 再重试。", executeGuideDocPath)
	message = strings.TrimSpace(message)
	if message == "" {
		return hint
	}
	return message + "\n\n" + hint
}

// sanitizeContentForMySQLUtf8 去掉 4 字节 UTF-8 字符（BMP 外），避免 MySQL utf8 列报 Error 1366；表为 utf8mb4 时无需此过滤。
func sanitizeContentForMySQLUtf8(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 0x10000 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// saveToolMessage 保存一条 role=tool 的消息。失败时返回 error，调用方应中止下一轮以免 400 insufficient tool messages。
func (s *WorkspaceChatService) saveToolMessage(ctx context.Context, sessionID string, agentIDPtr *int64, toolCallID, toolName, status string, result ToolResult, user string) error {
	var resultDataStr *string
	if result.Data != nil {
		if b, err := json.Marshal(result.Data); err == nil {
			s := string(b)
			resultDataStr = &s
		} else {
			logger.Warnf(ctx, "[WorkspaceChatStream] 保存 tool result_data 失败 ToolCallID=%s: %v", toolCallID, err)
		}
	}
	toolMsg := &model.AgentChatMessage{
		SessionID:  sessionID,
		AgentID:    agentIDPtr,
		Role:       RoleTool,
		Content:    sanitizeContentForMySQLUtf8(result.Content),
		ToolCallID: toolCallID,
		ToolStatus: status,
		ResultData: resultDataStr,
		User:       user,
	}
	toolMsg.CreatedBy = user
	toolMsg.UpdatedBy = user
	if err := s.messageRepo.Create(toolMsg); err != nil {
		return err
	}
	return nil
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

// handleError 统一错误处理：发送错误事件并返回错误
func (s *WorkspaceChatService) handleError(sendEvent func(string, interface{}), message string, err error) error {
	sendEvent(EventError, StreamEventError{Message: message})
	if err != nil {
		return fmt.Errorf("%s: %w", message, err)
	}
	return fmt.Errorf("%s", message)
}
