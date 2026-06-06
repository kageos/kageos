package service

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/core/agent-server/prompt"
	"github.com/kageos/kageos/core/agent-server/repository"
	"github.com/kageos/kageos/core/agent-server/streamloop"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
)

const (
	SourceWorkspace = "workspace"
	MaxToolRounds   = 100 // 与 streamloop.MaxToolRounds 保持一致，仅作注释/文档用，实际以 streamloop 为准
)

// 工作台系统提示词与内置文档统一来自本地内嵌的 /system/prompt。

// 消息角色常量
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

const (
	ContextPolicyFull         = "full"
	ContextPolicyArtifactOnly = "artifact_only"
	ContextPolicyDisplayOnly  = "display_only"

	MessageContextInclude     = "include"
	MessageContextDisplayOnly = "display_only"
	MessageContextArtifact    = "artifact"
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
	EventSession  = "session"
	EventToolCall = "tool_call"
	EventContent  = "content"
	EventDone     = "done"
	EventError    = "error"
)

// WorkspaceChatService 工作台对话编排：会话、历史、LLM、Tool 循环；只认 LLM + 单模式（dev）
type WorkspaceChatService struct {
	toolReg      *ToolRegistry
	sessionRepo  *repository.ChatSessionRepository
	messageRepo  *repository.ChatMessageRepository
	llmRepo      *repository.LLMRepository
	runtimeState RuntimeStateStore

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
	runtimeState ...RuntimeStateStore,
) *WorkspaceChatService {
	var stateStore RuntimeStateStore
	if len(runtimeState) > 0 {
		stateStore = runtimeState[0]
	}
	return &WorkspaceChatService{
		toolReg:      toolReg,
		sessionRepo:  sessionRepo,
		messageRepo:  messageRepo,
		llmRepo:      llmRepo,
		runtimeState: stateStore,
	}
}

// StreamEvent 流式事件：用于 SSE 传输
type StreamEvent struct {
	Event string      `json:"event"` // session|tool_call|content|done|error
	Data  interface{} `json:"data"`  // 对应负载（具体类型见下方各事件结构体）
}

// WorkspaceChatEventSink 接收工作台执行事件。SSE 与后台任务共用同一执行链路。
type WorkspaceChatEventSink interface {
	Send(event string, data interface{})
}

type workspaceChatEventSinkFunc func(event string, data interface{})

func (f workspaceChatEventSinkFunc) Send(event string, data interface{}) {
	f(event, data)
}

// StreamEventSession session 事件数据
type StreamEventSession struct {
	SessionID string `json:"session_id"`
}

// StreamEventToolCall tool_call 事件数据
type StreamEventToolCall struct {
	ID         string                  `json:"id,omitempty"` // tool_call_id（用于稳定合并同一次调用）
	Index      int                     `json:"index"`        // 当前工具轮次内的调用序号
	Round      int                     `json:"round"`        // 工具调用轮次，从 0 开始
	Name       string                  `json:"name"`
	Status     string                  `json:"status"`                // ok / error / running / streaming
	Arguments  string                  `json:"arguments"`             // 流式或最终参数（streaming 时逐段推送，供前端实时展示）
	Result     string                  `json:"result"`                // 工具返回结果（status=ok 时可选）
	ResultData interface{}             `json:"result_data,omitempty"` // 结构化工具结果（供前端直接消费）
	Metadata   *dto.ToolResultMetadata `json:"metadata,omitempty"`    // 工具结果元数据（供前端按字段渲染）
	Error      string                  `json:"error"`                 // 错误信息（status=error 时可选）
}

// StreamEventContent content 事件数据
type StreamEventContent struct {
	Content string `json:"content"`
}

// StreamEventDone done 事件数据
type StreamEventDone struct {
	SessionID     string                             `json:"session_id"`
	ToolCalls     []dto.WorkspaceChatToolCallSummary `json:"tool_calls"`
	LLMConfigID   int64                              `json:"llm_config_id,omitempty"`
	LLMConfigName string                             `json:"llm_config_name,omitempty"`
	LLMProvider   string                             `json:"llm_provider,omitempty"`
	LLMModel      string                             `json:"llm_model,omitempty"`
	LLMUsage      *dto.LLMUsageInfo                  `json:"llm_usage,omitempty"`
}

type messageLLMMetadata struct {
	ConfigID   int64
	ConfigName string
	Provider   string
	Model      string
}

// StreamEventError error 事件数据
type StreamEventError struct {
	Message string `json:"message"`
}

// WorkspaceChatStream 工作台对话流式入口：通过 eventChan 发送 SSE 事件（session、tool_call、content、done、error）
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
	var runtimeStateKey string
	var runtimeStateBase dto.RuntimeStateItem
	// 非阻塞发送事件：写不进 eventChan 时直接丢弃（刷新后无人读也不会卡死 goroutine）
	sendEvent := func(event string, data interface{}) {
		s.updateWorkspaceRuntimeStateFromEvent(ctx, runtimeStateKey, runtimeStateBase, event, data)
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

	// 1) 解析或创建 session。已有会话优先使用会话自身目录，避免阶段交接后被前端旧面板路径带回来源目录。
	var session *model.AgentChatSession
	if req.SessionID != "" {
		var e error
		session, e = s.sessionRepo.GetBySessionID(req.SessionID)
		if e != nil || session == nil {
			return s.handleError(sendEvent, fmt.Sprintf("会话不存在: %s", req.SessionID), e)
		}
		if e := ensureWorkspaceSessionOwner(ctx, session); e != nil {
			return s.handleError(sendEvent, e.Error(), e)
		}
		if sessionPath := strings.TrimSpace(session.FullCodePath); sessionPath != "" {
			fullCodePath = sessionPath
		}
	}

	// 2) 获取工作台环境信息（包含目录详情、子节点等，一次性获取，避免重复调用）
	workspaceCtx, e := apicall.GetWorkspaceContext(ctx, fullCodePath, "")
	if e != nil || workspaceCtx == nil {
		return s.handleError(sendEvent, "无效的 full_code_path，无法解析目录", e)
	}
	directoryName := workspaceCtx.Directory.Name
	if directoryName == "" {
		directoryName = workspaceCtx.Directory.Code
	}

	// 3) 创建新 session
	if req.SessionID == "" {
		session = &model.AgentChatSession{
			TreeID:        workspaceCtx.Directory.ID,
			FullCodePath:  fullCodePath,
			Source:        SourceWorkspace,
			SessionID:     uuid.New().String(),
			Title:         "",
			ModeCode:      requestedModeCode,
			Status:        model.ChatSessionStatusActive,
			ContextPolicy: ContextPolicyFull,
			User:          user,
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
	if session.ArchivedForModel {
		return s.handleError(sendEvent, "该会话已归档为展示历史，不再进入模型上下文；请从新的阶段交接会话继续。", nil)
	}
	if session.Status == model.ChatSessionStatusGenerating {
		return s.handleError(sendEvent, "该会话正在执行中，请等待当前任务完成，或先取消后再继续。", nil)
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
	resumeExistingMessage := req.Resume
	if resumeExistingMessage {
		if req.SessionID == "" {
			return s.handleError(sendEvent, "resume 必须指定 session_id", nil)
		}
		if e := s.ensureWorkspaceSessionHasRunnableMessage(sessionID); e != nil {
			return s.handleError(sendEvent, e.Error(), e)
		}
	}
	s.sseConnections.Store(sessionID, struct{}{})

	// ⭐ 标记会话为 generating（后台执行中）
	markedGenerating, e := s.sessionRepo.TryMarkGenerating(sessionID, user, modeCode)
	if e != nil {
		logger.Warnf(ctx, "[WorkspaceChatStream] 标记 generating 失败: %v", e)
		s.sseConnections.Delete(sessionID)
		return s.handleError(sendEvent, "标记会话执行中失败", e)
	} else if !markedGenerating {
		s.sseConnections.Delete(sessionID)
		return s.handleError(sendEvent, "该会话正在执行中，请等待当前任务完成，或先取消后再继续。", nil)
	}
	session.Status = model.ChatSessionStatusGenerating
	session.ModeCode = modeCode
	session.UpdatedBy = user

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
		finalStatus := ""
		if e == nil && latest != nil && latest.Status == model.ChatSessionStatusCancelled {
			finalStatus = RuntimeStateStatusCancelled
		}
		s.finishWorkspaceRuntimeState(context.Background(), runtimeStateKey, runtimeStateBase, err, finalStatus)
	}()

	llmConfigID := req.LLMConfigID

	// 3) 保存 user 消息。handoff 原子化场景下，首条 artifact 消息已在 handoff 事务内落库，此处只续跑。
	if !resumeExistingMessage {
		storageContent, storageFiles := userContentForStorage(req.Message.Files, req.Message.Content)
		userMsg := &model.AgentChatMessage{
			SessionID: sessionID, Role: RoleUser,
			Content: storageContent, DisplayContent: strings.TrimSpace(req.Message.DisplayContent), Files: storageFiles,
			ContextUsage: normalizeMessageContextUsage(req.Message.ContextUsage),
			ArtifactKind: strings.TrimSpace(req.Message.ArtifactKind),
			User:         user,
		}
		userMsg.CreatedBy = user
		userMsg.UpdatedBy = user
		if e := s.messageRepo.Create(userMsg); e != nil {
			return s.handleError(sendEvent, "保存用户消息失败", e)
		}
	}

	// 3.1) 新会话自动生成标题
	if !resumeExistingMessage && session.Title == "" {
		title := strings.TrimSpace(req.Message.DisplayContent)
		if title == "" {
			title = strings.TrimSpace(req.Message.Content)
		}
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

	runtimeStateKey, runtimeStateBase = s.startWorkspaceRuntimeState(ctx, session, fullCodePath, modeCode, user)
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

// RunWorkspaceChat 后台执行工作台对话；用于定时 Agent 会话等无浏览器连接场景。
func (s *WorkspaceChatService) RunWorkspaceChat(ctx context.Context, req *dto.WorkspaceChatReq, sink WorkspaceChatEventSink) error {
	if sink == nil {
		sink = workspaceChatEventSinkFunc(func(string, interface{}) {})
	}
	eventChan := make(chan StreamEvent, 100)
	done := make(chan error, 1)
	go func() {
		done <- s.WorkspaceChatStream(ctx, req, eventChan)
		close(eventChan)
	}()
	for ev := range eventChan {
		sink.Send(ev.Event, ev.Data)
	}
	return <-done
}

// handleError 统一错误处理：发送错误事件并返回错误
func (s *WorkspaceChatService) handleError(sendEvent func(string, interface{}), message string, err error) error {
	sendEvent(EventError, StreamEventError{Message: message})
	if err != nil {
		return fmt.Errorf("%s: %w", message, err)
	}
	return fmt.Errorf("%s", message)
}
