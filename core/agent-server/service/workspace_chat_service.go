package service

import (
	"bytes"
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
	"github.com/google/uuid"
	"gorm.io/gorm"
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
	EventSession         = "session"
	EventToolCall        = "tool_call"
	EventToolCallsStream = "tool_calls_stream" // LLM 流式输出的 tool_calls 当前列表（name+arguments），前端实时展示
	EventContent         = "content"
	EventDone            = "done"
	EventError           = "error"
)

const filesInstruction = "以上 <files> 标签中的 JSON 为本轮用户上传的文件引用。files 字段的新标准是 bucket/object_key 字符串；提交表单或表格时，直接把 refs 字符串填到对应 files 字段。"

// buildUserMessageContentWithFiles 当用户上传了文件时，将消息内容格式化为：
// <files>\n{JSON}\n</files> + 说明 + 用户需求，便于 Agent 把 <files> 内的 refs 当作 run_form_submit 的 input_files 使用。
// 仅用于拼装发给 LLM 的完整内容，不入库。
func buildUserMessageContentWithFiles(files string, userContent string) string {
	files = strings.TrimSpace(files)
	if files == "" {
		return userContent
	}
	demand := strings.TrimSpace(userContent)
	if demand == "" {
		demand = "用户需求：请处理上述文件"
	}
	return "<files>\n" + filesPayloadForLLM(files) + "\n</files>\n\n" + filesInstruction + "\n\n" + demand
}

// userContentForStorage 入库用：只保留用户文字到 Content，文件引用字符串单独到 Files。
// 返回 (content, filesRefs)；无文件时 filesRefs 为 nil。
func userContentForStorage(files string, userContent string) (content string, filesRefs *string) {
	demand := strings.TrimSpace(userContent)
	files = strings.TrimSpace(files)
	if files == "" {
		return demand, nil
	}
	if demand == "" {
		demand = "用户需求：请处理上述文件"
	}
	return demand, &files
}

// userContentForLLM 从库中取出的 user 消息：若有 Files 则拼出完整内容（<files>+说明+content）供 LLM 使用。
func userContentForLLM(content string, filesRefs *string) string {
	if filesRefs == nil || *filesRefs == "" {
		return content
	}
	demand := strings.TrimSpace(content)
	if demand == "" {
		demand = "用户需求：请处理上述文件"
	}
	return "<files>\n" + filesPayloadForLLM(*filesRefs) + "\n</files>\n\n" + filesInstruction + "\n\n" + demand
}

func filesPayloadForLLM(files string) string {
	files = strings.TrimSpace(files)
	if files == "" {
		return "{}"
	}
	payload := map[string]interface{}{
		"refs": files,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf(`{"refs":%q}`, files)
	}
	return string(raw)
}

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
	Event string      `json:"event"` // session|agent_id|tool_call|content|done|error
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
	Name       string                  `json:"name"`
	Status     string                  `json:"status"`                // ok / error / running / streaming
	Arguments  string                  `json:"arguments"`             // 流式或最终参数（streaming 时逐段推送，供前端实时展示）
	Result     string                  `json:"result"`                // 工具返回结果（status=ok 时可选）
	ResultData interface{}             `json:"result_data,omitempty"` // 结构化工具结果（供前端直接消费）
	Metadata   *dto.ToolResultMetadata `json:"metadata,omitempty"`    // 工具结果元数据（供前端按字段渲染）
	Error      string                  `json:"error"`                 // 错误信息（status=error 时可选）
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
	SessionID     string                             `json:"session_id"`
	ToolCalls     []dto.WorkspaceChatToolCallSummary `json:"tool_calls"`
	LLMConfigID   int64                              `json:"llm_config_id,omitempty"`
	LLMConfigName string                             `json:"llm_config_name,omitempty"`
	LLMProvider   string                             `json:"llm_provider,omitempty"`
	LLMModel      string                             `json:"llm_model,omitempty"`
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
			TreeID:        workspaceCtx.Directory.ID,
			FullCodePath:  fullCodePath,
			Source:        SourceWorkspace,
			SessionID:     uuid.New().String(),
			AgentID:       nil,
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
			SessionID: sessionID, AgentID: nil, Role: RoleUser,
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
	s.deleteWorkspaceRuntimeState(ctx, sessionID)
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
	return s.buildWorkspaceSessionItems(ctx, sessions), nil
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
	return s.buildWorkspaceSessionItems(ctx, sessions), nil
}

func (s *WorkspaceChatService) buildWorkspaceSessionItems(ctx context.Context, sessions []*model.AgentChatSession) []*dto.WorkspaceSessionItem {
	directoryNames := s.resolveWorkspaceSessionDirectoryNames(ctx, sessions)
	items := make([]*dto.WorkspaceSessionItem, 0, len(sessions))
	for _, session := range sessions {
		fullCodePath := strings.TrimSpace(session.FullCodePath)
		items = append(items, &dto.WorkspaceSessionItem{
			SessionID:         session.SessionID,
			Title:             session.Title,
			User:              session.User,
			AgentID:           session.AgentID,
			ModeCode:          normalizeWorkspaceModeCode(session.ModeCode),
			Status:            session.Status,
			RoleID:            workspaceSessionRoleID(session),
			RoleDisplayName:   workspaceSessionRoleDisplayName(session),
			FullCodePath:      session.FullCodePath,
			DirectoryName:     directoryNames[fullCodePath],
			ParentSessionID:   session.ParentSessionID,
			HandoffKind:       session.HandoffKind,
			HandoffTargetRole: session.HandoffTargetRole,
			ContextPolicy:     session.ContextPolicy,
			ArchivedForModel:  session.ArchivedForModel,
			ArchiveReason:     session.ArchiveReason,
			CreatedAt:         session.CreatedAt,
			UpdatedAt:         session.UpdatedAt,
		})
	}
	return items
}

func (s *WorkspaceChatService) resolveWorkspaceSessionDirectoryNames(ctx context.Context, sessions []*model.AgentChatSession) map[string]string {
	if s == nil || s.sessionRepo == nil || len(sessions) == 0 {
		return map[string]string{}
	}

	paths := make([]string, 0, len(sessions))
	for _, session := range sessions {
		path := strings.TrimSpace(session.FullCodePath)
		if path != "" {
			paths = append(paths, path)
		}
	}

	directoryNames, err := s.sessionRepo.GetServiceTreeNamesByFullCodePaths(paths)
	if err != nil {
		logger.Warnf(ctx, "[WorkspaceChat] 查询会话目录名称失败: %v", err)
		return map[string]string{}
	}
	return directoryNames
}

func (s *WorkspaceChatService) persistWorkspaceSessionInteractionStatus(ctx context.Context, sessionID string, summaries []streamloop.ToolCallSummary, user string) {
	nextStatus := workspaceSessionStatusFromToolSummaries(summaries)
	if nextStatus == "" || s == nil || s.sessionRepo == nil {
		return
	}

	session, err := s.sessionRepo.GetBySessionID(sessionID)
	if err != nil || session == nil {
		logger.Warnf(ctx, "[WorkspaceChat] 读取待交互会话失败 session_id=%s err=%v", sessionID, err)
		return
	}
	if session.Status == model.ChatSessionStatusCancelled || session.Status == model.ChatSessionStatusDone {
		return
	}
	session.Status = nextStatus
	session.UpdatedBy = user
	if err := s.sessionRepo.Update(session); err != nil {
		logger.Warnf(ctx, "[WorkspaceChat] 持久化待交互状态失败 session_id=%s status=%s err=%v", sessionID, nextStatus, err)
	}
}

func workspaceSessionStatusFromToolSummaries(summaries []streamloop.ToolCallSummary) string {
	if status := workspaceInteractionSessionStatusFromToolSummaries(summaries); status != "" {
		return status
	}
	if workspaceToolSummariesHaveGeneratedOutput(summaries) {
		return model.ChatSessionStatusOutput
	}
	return ""
}

func workspaceInteractionSessionStatusFromToolSummaries(summaries []streamloop.ToolCallSummary) string {
	for i := len(summaries) - 1; i >= 0; i-- {
		if summaries[i].Status != ToolCallStatusOK {
			continue
		}
		if status := workspaceInteractionSessionStatusFromResultData(summaries[i].ResultData); status != "" {
			return status
		}
	}
	return ""
}

func workspaceToolSummariesHaveGeneratedOutput(summaries []streamloop.ToolCallSummary) bool {
	for i := len(summaries) - 1; i >= 0; i-- {
		summary := summaries[i]
		if summary.Status != ToolCallStatusOK {
			continue
		}
		if workspaceToolCallHasGeneratedOutput(summary) {
			return true
		}
	}
	return false
}

func workspaceToolCallHasGeneratedOutput(summary streamloop.ToolCallSummary) bool {
	switch strings.TrimSpace(summary.Name) {
	case "write_prd",
		"build_workspace",
		"write_go_file",
		"write_doc",
		"create_directory",
		"copy_directory":
		return true
	}
	if summary.Metadata != nil && len(summary.Metadata.DisplayFileFields) > 0 {
		return true
	}
	return workspaceResultDataLooksLikeArtifact(summary.ResultData)
}

func workspaceResultDataLooksLikeArtifact(resultData interface{}) bool {
	if resultData == nil {
		return false
	}
	raw, err := json.Marshal(resultData)
	if err != nil {
		return false
	}
	var payload struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false
	}
	kind := strings.TrimSpace(payload.Kind)
	return strings.HasPrefix(kind, "agent_app_") || strings.HasPrefix(kind, "workspace_")
}

func workspaceInteractionSessionStatusFromResultData(resultData interface{}) string {
	if resultData == nil {
		return ""
	}
	raw, err := json.Marshal(resultData)
	if err != nil {
		return ""
	}
	var payload struct {
		Interaction *struct {
			Status string `json:"status"`
		} `json:"interaction"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil || payload.Interaction == nil {
		return ""
	}
	return normalizeWorkspacePendingInteractionStatus(payload.Interaction.Status)
}

func normalizeWorkspacePendingInteractionStatus(status string) string {
	switch strings.TrimSpace(status) {
	case model.ChatSessionStatusPendingConfirmation:
		return model.ChatSessionStatusPendingConfirmation
	case model.ChatSessionStatusPendingTest:
		return model.ChatSessionStatusPendingTest
	default:
		return ""
	}
}

func (s *WorkspaceChatService) ResolveWorkspacePendingInteraction(ctx context.Context, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session_id 必填")
	}
	session, err := s.sessionRepo.GetBySessionID(sessionID)
	if err != nil || session == nil {
		return fmt.Errorf("会话不存在: %s", sessionID)
	}
	user := contextx.GetRequestUser(ctx)
	if user != "" && session.User != "" && session.User != user {
		return fmt.Errorf("不能操作其他用户的会话")
	}
	switch session.Status {
	case model.ChatSessionStatusPendingConfirmation, model.ChatSessionStatusPendingTest:
		session.Status = model.ChatSessionStatusActive
		session.UpdatedBy = user
		if err := s.sessionRepo.Update(session); err != nil {
			return fmt.Errorf("更新会话状态失败: %w", err)
		}
	}
	return nil
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
	client := llms.NewOpenAIClientWithOptions(llmConfig.APIKey, opts)

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
	return llmConfig, client, chatReq, nil
}

func buildMessageLLMMetadata(llmConfig *model.LLMConfig, client llms.LLMClient) messageLLMMetadata {
	meta := messageLLMMetadata{}
	if llmConfig != nil {
		meta.ConfigID = llmConfig.ID
		meta.ConfigName = llmConfig.Name
		meta.Provider = llmConfig.Provider
		meta.Model = llmConfig.Model
	}
	if client != nil {
		if modelName := strings.TrimSpace(client.GetModelName()); modelName != "" {
			meta.Model = modelName
		}
	}
	return meta
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
			Schema:       n.Schema,
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
	if modeProvider != nil {
		systemPromptFragment = "" // 在 env 填好后由 provider.SystemPrompt(data) 产出
	} else {
		systemPromptFragment = fallbackSystemPrompt
	}
	toolNames = workspaceToolNamesForMode(modeProvider, fallbackToolNames)
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

	msgs := []llms.Message{{Role: "system", Content: system}}
	for _, m := range list {
		if normalizeMessageContextUsage(m.ContextUsage) == MessageContextDisplayOnly {
			continue
		}
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

	items := s.buildWorkspaceSessionItems(ctx, sessions)

	return items, total, nil
}

// ListMessages 根据 sessionID 获取消息列表
func (s *WorkspaceChatService) ListMessages(ctx context.Context, sessionID string) ([]*model.AgentChatMessage, error) {
	_ = ctx
	return s.messageRepo.ListBySessionID(sessionID)
}

func (s *WorkspaceChatService) ensureWorkspaceSessionHasRunnableMessage(sessionID string) error {
	if s == nil || s.messageRepo == nil {
		return fmt.Errorf("消息仓储未初始化，无法续跑会话")
	}
	messages, err := s.messageRepo.ListBySessionID(sessionID)
	if err != nil {
		return fmt.Errorf("读取会话消息失败: %w", err)
	}
	for _, msg := range messages {
		if msg == nil || msg.Role != RoleUser {
			continue
		}
		if normalizeMessageContextUsage(msg.ContextUsage) != MessageContextDisplayOnly {
			return nil
		}
	}
	return fmt.Errorf("会话没有可进入模型上下文的用户消息，无法续跑")
}

// CreateWorkspaceHandoff freezes the source conversation for model context and
// creates a clean target session that starts from one structured artifact.
func (s *WorkspaceChatService) CreateWorkspaceHandoff(ctx context.Context, req *dto.WorkspaceHandoffReq) (*dto.WorkspaceHandoffResp, error) {
	if req == nil {
		return nil, fmt.Errorf("handoff 请求不能为空")
	}
	sourceSessionID := strings.TrimSpace(req.SourceSessionID)
	if sourceSessionID == "" {
		return nil, fmt.Errorf("source_session_id 必填")
	}
	source, err := s.sessionRepo.GetBySessionID(sourceSessionID)
	if err != nil || source == nil {
		return nil, fmt.Errorf("来源会话不存在: %s", sourceSessionID)
	}
	user := contextx.GetRequestUser(ctx)
	if user != "" && source.User != "" && source.User != user {
		return nil, fmt.Errorf("不能交接其他用户的会话")
	}
	targetRole := normalizeWorkspaceRole(req.TargetRole)
	if targetRole == "" || !isKnownWorkspaceRole(targetRole) {
		return nil, fmt.Errorf("target_role 不支持: %s", strings.TrimSpace(req.TargetRole))
	}
	artifactKind := strings.TrimSpace(req.ArtifactKind)
	if artifactKind == "" {
		return nil, fmt.Errorf("artifact_kind 必填")
	}
	artifactJSON := prettyWorkspaceHandoffArtifact(req.Artifact)
	if artifactJSON == "" {
		return nil, fmt.Errorf("artifact 不能为空")
	}
	fullCodePath := strings.TrimSpace(req.FullCodePath)
	if fullCodePath == "" {
		fullCodePath = source.FullCodePath
	}
	if fullCodePath == "" {
		return nil, fmt.Errorf("full_code_path 必填")
	}
	contextPolicy := normalizeWorkspaceHandoffContextPolicy(req.ContextPolicy)
	modeCode := normalizeWorkspaceModeCode(source.ModeCode)
	if modeCode == "" {
		modeCode = "dev"
	}

	source.ArchivedForModel = true
	source.ContextPolicy = ContextPolicyDisplayOnly
	source.ArchiveReason = fmt.Sprintf("已交接到%s，会话仅保留展示历史", workspaceRoleDisplayName(targetRole))
	source.Status = model.ChatSessionStatusDone
	source.UpdatedBy = user

	targetSessionID := uuid.New().String()
	displayContent := strings.TrimSpace(req.DisplayContent)
	if displayContent == "" {
		displayContent = defaultWorkspaceHandoffDisplayContent(artifactKind, targetRole, req.Remark)
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = displayContent
	}
	if len([]rune(title)) > 50 {
		runes := []rune(title)
		title = string(runes[:50]) + "..."
	}
	target := &model.AgentChatSession{
		TreeID:            source.TreeID,
		FullCodePath:      fullCodePath,
		Source:            SourceWorkspace,
		SessionID:         targetSessionID,
		AgentID:           nil,
		Title:             title,
		ModeCode:          modeCode,
		Status:            model.ChatSessionStatusActive,
		RoleID:            targetRole,
		RoleDisplayName:   workspaceRoleDisplayName(targetRole),
		ParentSessionID:   source.SessionID,
		HandoffKind:       artifactKind,
		HandoffTargetRole: targetRole,
		ContextPolicy:     contextPolicy,
		User:              user,
	}
	if target.User == "" {
		target.User = source.User
	}
	target.CreatedBy = user
	target.UpdatedBy = user
	content := buildWorkspaceHandoffContent(workspaceHandoffContentInput{
		ArtifactKind:  artifactKind,
		ArtifactJSON:  artifactJSON,
		TargetRole:    targetRole,
		Remark:        req.Remark,
		ContextPolicy: contextPolicy,
	})
	initialMessage := &model.AgentChatMessage{
		SessionID:      targetSessionID,
		AgentID:        nil,
		Role:           RoleUser,
		Content:        content,
		DisplayContent: displayContent,
		ContextUsage:   MessageContextArtifact,
		ArtifactKind:   artifactKind,
		User:           target.User,
	}
	initialMessage.CreatedBy = user
	initialMessage.UpdatedBy = user
	handoffPacket := &model.WorkspaceHandoffPacket{
		SourceSessionID: source.SessionID,
		TargetSessionID: targetSessionID,
		FullCodePath:    fullCodePath,
		TargetRole:      targetRole,
		ArtifactKind:    artifactKind,
		ArtifactJSON:    artifactJSON,
		Remark:          strings.TrimSpace(req.Remark),
		ContextPolicy:   contextPolicy,
		User:            target.User,
	}
	handoffPacket.CreatedBy = user
	handoffPacket.UpdatedBy = user
	if err := s.sessionRepo.TransactionWithMessagesAndHandoffPackets(func(sessionTx *repository.ChatSessionRepository, messageTx *repository.ChatMessageRepository, handoffTx *repository.WorkspaceHandoffPacketRepository) error {
		if err := sessionTx.Update(source); err != nil {
			return fmt.Errorf("归档来源会话失败: %w", err)
		}
		if err := sessionTx.Create(target); err != nil {
			return fmt.Errorf("创建交接会话失败: %w", err)
		}
		if err := messageTx.Create(initialMessage); err != nil {
			return fmt.Errorf("创建交接消息失败: %w", err)
		}
		handoffPacket.InitialMessageID = initialMessage.ID
		if err := handoffTx.Create(handoffPacket); err != nil {
			return fmt.Errorf("创建交接包失败: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &dto.WorkspaceHandoffResp{
		SessionID:       targetSessionID,
		SourceSessionID: source.SessionID,
		TargetRole:      targetRole,
		ArtifactKind:    artifactKind,
		ContextPolicy:   contextPolicy,
		HandoffPacketID: handoffPacket.ID,
		MessageID:       initialMessage.ID,
		Content:         content,
		DisplayContent:  displayContent,
	}, nil
}

type workspaceHandoffContentInput struct {
	ArtifactKind  string
	ArtifactJSON  string
	TargetRole    string
	Remark        string
	ContextPolicy string
}

func prettyWorkspaceHandoffArtifact(raw json.RawMessage) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return ""
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(trimmed), "", "  "); err == nil {
		return buf.String()
	}
	return trimmed
}

func normalizeWorkspaceHandoffContextPolicy(policy string) string {
	switch strings.TrimSpace(policy) {
	case ContextPolicyFull:
		return ContextPolicyFull
	case ContextPolicyDisplayOnly:
		return ContextPolicyDisplayOnly
	default:
		return ContextPolicyArtifactOnly
	}
}

func normalizeMessageContextUsage(usage string) string {
	switch strings.TrimSpace(usage) {
	case MessageContextDisplayOnly:
		return MessageContextDisplayOnly
	case MessageContextArtifact:
		return MessageContextArtifact
	default:
		return MessageContextInclude
	}
}

func defaultWorkspaceHandoffDisplayContent(artifactKind, targetRole, remark string) string {
	switch artifactKind {
	case "agent_app_prd":
		if strings.TrimSpace(remark) != "" {
			return "已确认 PRD，开始创建目录和生成代码。\n\n补充备注：\n" + strings.TrimSpace(remark)
		}
		return "已确认 PRD，开始创建目录和生成代码。"
	case workspaceBuildArtifactKind:
		if strings.TrimSpace(remark) != "" {
			return "已构建成功，开始测试验证。\n\n补充备注：\n" + strings.TrimSpace(remark)
		}
		return "已构建成功，开始测试验证。"
	default:
		label := strings.TrimSpace(artifactKind)
		if label == "" {
			label = "阶段产物"
		}
		return fmt.Sprintf("已确认 %s，进入 %s 阶段。", label, workspaceRoleDisplayName(targetRole))
	}
}

func buildWorkspaceHandoffContent(input workspaceHandoffContentInput) string {
	artifactLabel := input.ArtifactKind
	if artifactLabel == "" {
		artifactLabel = "artifact"
	}
	lines := []string{
		"已确认阶段交接产物，进入下一阶段。",
		"",
		fmt.Sprintf("这是阶段交接后的执行会话。请先调用 change_role，target_role 固定为 %s。", input.TargetRole),
		fmt.Sprintf("上下文策略：%s。只以本消息中的结构化产物 JSON 和补充备注为准，不要依赖来源会话的历史讨论。", input.ContextPolicy),
		"不要重复产出已确认的设计文档；除非产物本身缺失关键字段，否则直接执行目标阶段任务。",
	}
	if input.ArtifactKind == "agent_app_prd" && normalizeWorkspaceRole(input.TargetRole) == WorkspaceRoleAppDeveloper {
		lines = append(lines,
			"生成阶段要求：不要重新输出 PRD，不要再次询问确认；先读取 1 到多个匹配案例，再根据 PRD tables/forms/charts/workflow/rules 创建目录、写代码文件、注册路由并 build。tables.fields 是业务模型字段，tables.search_fields 是查询请求字段；创建开始时间/创建结束时间/创建人等系统搜索字段不要生成业务列。route、method、widget tag、列表列和预览数据均从 PRD 派生。非常简单的需求才可跳过额外案例。",
		)
	}
	if input.ArtifactKind == workspaceBuildArtifactKind && normalizeWorkspaceRole(input.TargetRole) == WorkspaceRoleQAEngineer {
		lines = append(lines,
			"测试阶段要求：不要修改代码，不要重新 build；先调用 change_role 进入 qa_engineer，再用 search_tools/read_dir 确认当前工作空间函数清单和 schema。按业务操作顺序验证：先主数据/配置表，再 Form 提交，再目标记录表，再 Chart；重点覆盖创建开始时间/创建结束时间和用户筛选。测试失败时判断是测试数据问题、业务 bug 还是构建/schema 问题，并交接给 maintenance_engineer 或 build_engineer。",
		)
	}
	lines = append(lines,
		"",
		strings.ToUpper(artifactLabel)+" JSON:",
		"```json",
		input.ArtifactJSON,
		"```",
	)
	if remark := strings.TrimSpace(input.Remark); remark != "" {
		lines = append(lines, "", "补充备注：", remark)
	}
	return strings.Join(lines, "\n")
}

// executeToolCalls 执行工具调用并保存消息。若 tool 消息保存失败则返回 error，不再进入下一轮，避免 400 insufficient tool messages。
func (s *WorkspaceChatService) executeToolCalls(
	ctx context.Context,
	allToolCalls []llms.ToolCall,
	_ string, // currentAssistantContent 保留参数以兼容调用方，已不再用于 <var> 解析
	sessionID, fullCodePath string,
	agentIDPtr *int64,
	user string,
	files string,
	_ []string,
	sendEvent func(string, interface{}),
) ([]dto.WorkspaceChatToolCallSummary, error) {
	ctx = withAgentToolExecutionContext(ctx, sessionID)
	toolSummaries := make([]dto.WorkspaceChatToolCallSummary, 0, len(allToolCalls))
	logger.Infof(ctx, "[WorkspaceChatStream] 开始执行工具调用 - 工具数量: %d, SessionID: %s", len(allToolCalls), sessionID)
	loadedGuideDocs := s.loadedGuideDocsForSession(ctx, sessionID)
	activeRoleID := s.currentWorkspaceRoleForSession(ctx, sessionID)

	for i, tc := range allToolCalls {
		logger.Infof(ctx, "[WorkspaceChatStream] [%d/%d] 执行工具调用 - ToolCallID: %s, ToolName: %s, Arguments: %q",
			i+1, len(allToolCalls), tc.ID, tc.Function.Name, tc.Function.Arguments)

		sendEvent(EventToolCall, StreamEventToolCall{Name: tc.Function.Name, Status: ToolCallStatusRunning, Arguments: tc.Function.Arguments})

		args := s.parseToolCallArgs(ctx, tc)
		var toolRes ToolResult
		var st string
		if blockedRes, blocked := workspaceRoleToolGateResult(activeRoleID, tc.Function.Name); blocked {
			toolRes = blockedRes
			st = ToolCallStatusError
			logger.Warnf(ctx, "[WorkspaceChatStream] [%d/%d] 角色工具门禁阻断 - RoleID: %s, ToolName: %s, Error: %s", i+1, len(allToolCalls), activeRoleID, tc.Function.Name, toolRes.Content)
		} else {
			toolRes, st = s.callOtherTool(ctx, tc.Function.Name, args, fullCodePath, files, i+1, len(allToolCalls))
		}
		if st == ToolCallStatusOK && tc.Function.Name == "change_role" {
			if nextRoleID := workspaceRoleFromToolResult(toolRes); nextRoleID != "" {
				activeRoleID = nextRoleID
				s.updateWorkspaceSessionRole(ctx, sessionID, nextRoleID, user)
			}
		}

		resultStr, errStr := "", ""
		var resultData interface{}
		if st == ToolCallStatusOK {
			resultStr = toolRes.Content
			resultData = toolRes.Data
		} else {
			errStr = toolRes.Content
		}
		toolSummaries = append(toolSummaries, dto.WorkspaceChatToolCallSummary{
			Name: tc.Function.Name, Status: st, Arguments: tc.Function.Arguments, Result: resultStr, ResultData: resultData, Metadata: toolRes.Metadata, Error: errStr,
		})
		sendEvent(EventToolCall, StreamEventToolCall{
			Name: tc.Function.Name, Status: st, Arguments: tc.Function.Arguments, Result: resultStr, ResultData: resultData, Metadata: toolRes.Metadata, Error: errStr,
		})
		if err := s.saveToolMessage(ctx, sessionID, agentIDPtr, tc.ID, tc.Function.Name, st, toolRes, user); err != nil {
			logger.Warnf(ctx, "[WorkspaceChatStream] 保存 tool 消息失败 ToolCallID=%s: %v（若为 Error 1366 请将表转为 utf8mb4）", tc.ID, err)
			return toolSummaries, fmt.Errorf("保存 tool 消息失败: %w", err)
		}
		updateLoadedGuideDocsAfterToolCall(loadedGuideDocs, tc.Function.Name, args, st)
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

func withAgentToolExecutionContext(ctx context.Context, sessionID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	// 注入 session_id 供 record_workspace_event 等工具追溯，同时统一标记 agent 工具入口来源。
	ctx = context.WithValue(ctx, WorkspaceSessionIDKey, sessionID)
	return withAgentToolClientSource(ctx)
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
func (s *WorkspaceChatService) callOtherTool(ctx context.Context, name string, args map[string]interface{}, fullCodePath string, files string, idx, total int) (res ToolResult, st string) {
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

func hasLoadedGuideDoc(loadedGuideDocs map[string]struct{}, docPath string) bool {
	required := normalizeGuideDocPath(docPath)
	for loaded := range loadedGuideDocs {
		if loaded == required || strings.HasPrefix(required, loaded+"/") {
			return true
		}
	}
	return false
}

func (s *WorkspaceChatService) loadedGuideDocsForSession(ctx context.Context, sessionID string) map[string]struct{} {
	loaded := make(map[string]struct{})
	if s == nil || s.messageRepo == nil || strings.TrimSpace(sessionID) == "" {
		return loaded
	}
	messages, err := s.messageRepo.ListBySessionID(sessionID)
	if err != nil {
		logger.Warnf(ctx, "[WorkspaceChatStream] 查询会话已读 SOP 失败 SessionID=%s: %v", sessionID, err)
		return loaded
	}
	return loadedGuideDocsFromMessages(ctx, messages)
}

func loadedGuideDocsFromMessages(ctx context.Context, messages []*model.AgentChatMessage) map[string]struct{} {
	loaded := make(map[string]struct{})
	readDocCalls := make(map[string][]string)
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		switch msg.Role {
		case RoleAssistant:
			if msg.ToolCalls == nil || strings.TrimSpace(*msg.ToolCalls) == "" {
				continue
			}
			var toolCalls []llms.ToolCall
			if err := json.Unmarshal([]byte(*msg.ToolCalls), &toolCalls); err != nil {
				logger.Warnf(ctx, "[WorkspaceChatStream] 解析历史 tool_calls 失败 MessageID=%d: %v", msg.ID, err)
				continue
			}
			for _, tc := range toolCalls {
				if strings.TrimSpace(tc.ID) == "" {
					continue
				}
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					logger.Warnf(ctx, "[WorkspaceChatStream] 解析历史 %s 参数失败 ToolCallID=%s: %v", tc.Function.Name, tc.ID, err)
					continue
				}
				switch tc.Function.Name {
				case "read_doc":
					readDocCalls[tc.ID] = guideDocPathsFromReadDocArgs(args)
				}
			}
		case RoleTool:
			if msg.ToolStatus != ToolCallStatusOK {
				continue
			}
			for _, docPath := range readDocCalls[msg.ToolCallID] {
				loaded[docPath] = struct{}{}
			}
		}
	}
	return loaded
}

func guideDocPathsFromReadDocArgs(args map[string]interface{}) []string {
	dirArg, _ := args["directory"].(string)
	if strings.TrimSpace(dirArg) == "" {
		dirArg, _ = args["full_code_path"].(string)
	}
	rawPaths := splitDirectoryPaths(dirArg)
	paths := make([]string, 0, len(rawPaths))
	for _, rawPath := range rawPaths {
		path := normalizeGuideDocPath(rawPath)
		if path == "" {
			continue
		}
		paths = append(paths, path)
	}
	return paths
}

func updateLoadedGuideDocsAfterToolCall(loaded map[string]struct{}, toolName string, args map[string]interface{}, status string) {
	if status != ToolCallStatusOK || loaded == nil {
		return
	}
	switch strings.TrimSpace(toolName) {
	case "read_doc":
		for _, docPath := range guideDocPathsFromReadDocArgs(args) {
			loaded[docPath] = struct{}{}
		}
	}
}

func normalizeGuideDocPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(path, "/")
}

func shouldSuggestExecuteGuide(toolName string) bool {
	switch strings.TrimSpace(toolName) {
	case "run_table_search", "run_table_create", "run_table_update", "run_table_delete", "run_form_submit", "run_chart_query", "run_on_select_fuzzy":
		return true
	default:
		return false
	}
}

func appendExecuteGuideHint(toolName, message string) string {
	return message
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

func marshalToolResultField(ctx context.Context, toolCallID, fieldName string, value any) *string {
	if value == nil {
		return nil
	}
	b, err := json.Marshal(value)
	if err != nil {
		logger.Warnf(ctx, "[WorkspaceChatStream] 保存 tool %s 失败 ToolCallID=%s: %v", fieldName, toolCallID, err)
		return nil
	}
	out := string(b)
	return &out
}

// saveToolMessage 保存一条 role=tool 的消息。失败时返回 error，调用方应中止下一轮以免 400 insufficient tool messages。
func (s *WorkspaceChatService) saveToolMessage(ctx context.Context, sessionID string, agentIDPtr *int64, toolCallID, toolName, status string, result ToolResult, user string) error {
	toolMsg := &model.AgentChatMessage{
		SessionID:      sessionID,
		AgentID:        agentIDPtr,
		Role:           RoleTool,
		Content:        sanitizeContentForMySQLUtf8(result.Content),
		ToolCallID:     toolCallID,
		ToolStatus:     status,
		ResultData:     marshalToolResultField(ctx, toolCallID, "result_data", result.Data),
		ResultMetadata: marshalToolResultField(ctx, toolCallID, "result_metadata", result.Metadata),
		User:           user,
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
	llmMeta messageLLMMetadata,
) error {
	toolCallsJSON, _ := json.Marshal(allToolCalls)
	toolCallsStr := string(toolCallsJSON)
	asstMsg := &model.AgentChatMessage{
		SessionID:     sessionID,
		AgentID:       agentIDPtr,
		Role:          RoleAssistant,
		Content:       content,
		ToolCalls:     &toolCallsStr,
		LLMConfigID:   llmMeta.ConfigID,
		LLMConfigName: llmMeta.ConfigName,
		LLMProvider:   llmMeta.Provider,
		LLMModel:      llmMeta.Model,
		User:          user,
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
	llmMeta messageLLMMetadata,
) error {
	asstMsg := &model.AgentChatMessage{
		SessionID:     sessionID,
		AgentID:       agentIDPtr,
		Role:          RoleAssistant,
		Content:       content,
		LLMConfigID:   llmMeta.ConfigID,
		LLMConfigName: llmMeta.ConfigName,
		LLMProvider:   llmMeta.Provider,
		LLMModel:      llmMeta.Model,
		User:          user,
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
