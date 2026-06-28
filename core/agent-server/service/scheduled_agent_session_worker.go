package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/nats-io/nats.go"
)

const ScheduledAgentSessionExecutorKey = "agent.session"

const scheduledAgentUnattendedPrefix = "【Agent 任务执行约束】本次 Agent 任务由自动执行触发，但当前目标不是创建或管理定时任务，而是执行后面的任务说明。请先选择能完成该业务执行的角色；需要调用已有 Form/Table/Chart 或连接器时，通常应进入 app_operator。执行过程中用户不在线、无法回答问题或确认操作；不要向用户提问，不要等待用户补充信息，不要把下一步停在“请确认/请提供”。如果发现高优先级情报、异常、风险或任务说明明确要求通知用户，可调用 send_notification 主动通知；send_notification 只负责单向通知，不能作为等待用户回复的交互。首次基准记录、无变化结果、普通状态报告默认不通知，只在执行摘要中记录。若创建时的信息不足以安全执行，按已知上下文完成可安全完成的部分，并在结果中明确记录缺失信息、未执行的动作和原因；涉及高风险写入且缺少必要确认时应跳过该动作并说明原因。"

type scheduledAgentWorkspaceRootContextKey struct{}

type scheduledAgentSessionPayload struct {
	FullCodePath       string `json:"full_code_path"`
	Directory          string `json:"directory,omitempty"`
	Message            string `json:"message,omitempty"`
	ModeCode           string `json:"mode_code,omitempty"`
	Files              string `json:"files,omitempty"`
	LLMConfigID        int64  `json:"llm_config_id,omitempty"`
	MaxDurationSeconds int64  `json:"max_duration_seconds,omitempty"`
	MaxDurationSec     int64  `json:"max_duration_sec,omitempty"`
	SessionID          string `json:"session_id,omitempty"`
	Resume             bool   `json:"resume,omitempty"`
	DisplayContent     string `json:"display_content,omitempty"`
	ContextUsage       string `json:"context_usage,omitempty"`
	ArtifactKind       string `json:"artifact_kind,omitempty"`
	InteractionAction  string `json:"interaction_action,omitempty"`
}

// NewScheduledAgentSessionWorker wires timer-scheduler's agent.session executor
// to the existing background workspace chat runner.
func NewScheduledAgentSessionWorker(natsConn *nats.Conn, chatSvc *WorkspaceChatService) (*scheduledsdk.Worker, error) {
	if natsConn == nil {
		return nil, fmt.Errorf("scheduled agent session worker requires nats connection")
	}
	if chatSvc == nil {
		return nil, fmt.Errorf("scheduled agent session worker requires workspace chat service")
	}
	client := scheduledsdk.NewClient(scheduledsdk.Options{
		Adapter: scheduledsdk.NewNATSAdapter(natsConn, scheduledsdk.NATSAdapterOptions{}),
	})
	return scheduledsdk.NewWorker(scheduledsdk.WorkerOptions{
		Client:      client,
		NATSConn:    natsConn,
		ExecutorKey: ScheduledAgentSessionExecutorKey,
		Handler:     chatSvc.RunScheduledAgentSession,
		OnError: func(ctx context.Context, err error) {
			logger.Warnf(ctx, "[ScheduledAgentSessionWorker] %v", err)
		},
	})
}

// RunScheduledAgentSession executes one timer-scheduler agent.session event.
func (s *WorkspaceChatService) RunScheduledAgentSession(ctx context.Context, event scheduledsdk.ExecutionRequestedEvent) (*scheduledsdk.ExecutionResult, error) {
	ctx = event.WithAuditContext(ctx)
	req, payload, err := scheduledAgentSessionWorkspaceRequest(event)
	if err != nil {
		return nil, err
	}
	if payload.MaxDurationSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(payload.MaxDurationSeconds)*time.Second)
		defer cancel()
	}

	sink := &scheduledAgentSessionSink{}
	logger.Infof(ctx, "[ScheduledAgentSessionWorker] start task_id=%d execution_id=%d full_code_path=%s",
		event.TaskID, event.ExecutionID, req.FullCodePath)
	ctx = contextWithScheduledAgentWorkspaceRoot(ctx, req.FullCodePath)
	err = s.RunWorkspaceChat(ctx, req, sink)
	result := sink.ExecutionResult()
	if err != nil {
		err = scheduledAgentSessionRunError(req.FullCodePath, err)
		logger.Errorf(ctx, "[ScheduledAgentSessionWorker] failed task_id=%d execution_id=%d session_id=%s err=%v",
			event.TaskID, event.ExecutionID, sink.SessionID(), err)
		return result, err
	}
	logger.Infof(ctx, "[ScheduledAgentSessionWorker] done task_id=%d execution_id=%d session_id=%s",
		event.TaskID, event.ExecutionID, sink.SessionID())
	return result, nil
}

func scheduledAgentSessionRunError(fullCodePath string, err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	if strings.Contains(message, "服务目录不存在") || strings.Contains(message, "无效的 full_code_path") {
		path := strings.TrimSpace(fullCodePath)
		if path == "" {
			return fmt.Errorf("Agent 任务配置的工作台目录不存在。请编辑任务换成有效目录，或删除后在目标目录重新创建: %w", err)
		}
		return fmt.Errorf("Agent 任务配置的工作台目录不存在（full_code_path=%s）。请编辑任务换成有效目录，或删除后在目标目录重新创建: %w", path, err)
	}
	return err
}

func scheduledAgentSessionWorkspaceRequest(event scheduledsdk.ExecutionRequestedEvent) (*dto.WorkspaceChatReq, scheduledAgentSessionPayload, error) {
	payload, err := decodeScheduledAgentSessionPayload(event)
	if err != nil {
		return nil, payload, err
	}
	if payload.FullCodePath == "" {
		return nil, payload, fmt.Errorf("scheduled agent session payload requires full_code_path")
	}
	if !payload.Resume && payload.Message == "" {
		return nil, payload, fmt.Errorf("scheduled agent session payload requires message")
	}
	modeCode := strings.TrimSpace(payload.ModeCode)
	if modeCode == "" {
		modeCode = "dev"
	}
	content := scheduledAgentSessionMessageContent(event, payload)
	return &dto.WorkspaceChatReq{
		FullCodePath: payload.FullCodePath,
		SessionID:    strings.TrimSpace(payload.SessionID),
		ModeCode:     modeCode,
		LLMConfigID:  payload.LLMConfigID,
		Resume:       payload.Resume,
		Message: dto.WorkspaceMsg{
			Content:           content,
			DisplayContent:    scheduledAgentSessionDisplayContent(payload),
			Files:             strings.TrimSpace(payload.Files),
			ContextUsage:      strings.TrimSpace(payload.ContextUsage),
			ArtifactKind:      strings.TrimSpace(payload.ArtifactKind),
			InteractionAction: strings.TrimSpace(payload.InteractionAction),
		},
	}, payload, nil
}

func decodeScheduledAgentSessionPayload(event scheduledsdk.ExecutionRequestedEvent) (scheduledAgentSessionPayload, error) {
	var payload scheduledAgentSessionPayload
	if len(event.ExecutorPayload) == 0 {
		return payload, fmt.Errorf("scheduled agent session executor_payload is empty")
	}
	if err := json.Unmarshal(event.ExecutorPayload, &payload); err != nil {
		return payload, fmt.Errorf("decode scheduled agent session payload: %w", err)
	}
	payload.FullCodePath = strings.TrimSpace(firstNonEmptyString(payload.FullCodePath, payload.Directory, event.ResourceKey))
	payload.Message = strings.TrimSpace(payload.Message)
	payload.ModeCode = strings.TrimSpace(payload.ModeCode)
	payload.Files = strings.TrimSpace(payload.Files)
	payload.SessionID = strings.TrimSpace(payload.SessionID)
	payload.DisplayContent = strings.TrimSpace(payload.DisplayContent)
	payload.ContextUsage = strings.TrimSpace(payload.ContextUsage)
	payload.ArtifactKind = strings.TrimSpace(payload.ArtifactKind)
	payload.InteractionAction = strings.TrimSpace(payload.InteractionAction)
	if payload.MaxDurationSeconds <= 0 && payload.MaxDurationSec > 0 {
		payload.MaxDurationSeconds = payload.MaxDurationSec
	}
	return payload, nil
}

func scheduledAgentSessionMessageContent(event scheduledsdk.ExecutionRequestedEvent, payload scheduledAgentSessionPayload) string {
	content := strings.TrimSpace(payload.Message)
	if content == "" {
		return ""
	}
	notificationHint := scheduledAgentNotificationInstruction(event)
	directoryHint := ""
	if fullCodePath := normalizeWorkspacePath(payload.FullCodePath); fullCodePath != "" {
		directoryHint = "\n本次任务绑定工作台目录：" + fullCodePath + "；change_role.execute_directory 必须沿用这个完整路径，不要改写、缩短或根据标题猜目录。"
	}
	return scheduledAgentUnattendedPrefix + notificationHint + directoryHint + "\n\n" + content
}

func scheduledAgentNotificationInstruction(event scheduledsdk.ExecutionRequestedEvent) string {
	requestUser := strings.TrimSpace(event.RequestUser)
	if requestUser == "" {
		return "\n【Agent 任务通知规则】本次 Agent 任务没有创建人/请求用户可作为默认接收人。只有任务说明明确给出接收人 username，或已经从任务上下文可靠获得接收人时，才可调用 send_notification；调用时必须显式传 to_users。"
	}
	return fmt.Sprintf("\n【Agent 任务通知规则】本次 Agent 任务创建人/默认通知对象：%s。如果任务要求通知创建人、当前用户或“我”，调用 send_notification 时可省略 to_users；如果显式传，请使用 to_users: %q。", requestUser, requestUser)
}

func scheduledAgentSessionDisplayContent(payload scheduledAgentSessionPayload) string {
	return strings.TrimSpace(payload.Message)
}

func contextWithScheduledAgentWorkspaceRoot(ctx context.Context, fullCodePath string) context.Context {
	root := normalizeWorkspacePath(fullCodePath)
	if root == "" {
		return ctx
	}
	return context.WithValue(ctx, scheduledAgentWorkspaceRootContextKey{}, root)
}

func scheduledAgentWorkspaceRootFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	root, _ := ctx.Value(scheduledAgentWorkspaceRootContextKey{}).(string)
	return normalizeWorkspacePath(root)
}

type scheduledAgentSessionSink struct {
	sessionID string
	content   strings.Builder
	errors    []string
	toolCalls []dto.WorkspaceChatToolCallSummary
}

func (s *scheduledAgentSessionSink) Send(event string, data interface{}) {
	switch event {
	case EventSession:
		if session := workspaceStreamSessionData(data); session.SessionID != "" {
			s.sessionID = session.SessionID
		}
	case EventContent:
		content := workspaceStreamContentData(data).Content
		if content != "" {
			s.content.WriteString(content)
		}
	case EventToolCall:
		toolCall := workspaceStreamToolCallData(data)
		if toolCall.Status == ToolCallStatusOK || toolCall.Status == ToolCallStatusError {
			s.toolCalls = append(s.toolCalls, dto.WorkspaceChatToolCallSummary{
				ID:        toolCall.ID,
				Index:     toolCall.Index,
				Round:     toolCall.Round,
				Name:      toolCall.Name,
				Status:    toolCall.Status,
				Arguments: toolCall.Arguments,
				Result:    toolCall.Result,
				Error:     toolCall.Error,
			})
		}
	case EventDone:
		done := workspaceStreamDoneData(data)
		if done.SessionID != "" {
			s.sessionID = done.SessionID
		}
		if len(done.ToolCalls) > 0 {
			s.toolCalls = done.ToolCalls
		}
	case EventError:
		if message := strings.TrimSpace(workspaceStreamErrorData(data).Message); message != "" {
			s.errors = append(s.errors, message)
		}
	}
}

func (s *scheduledAgentSessionSink) SessionID() string {
	if s == nil {
		return ""
	}
	return s.sessionID
}

func (s *scheduledAgentSessionSink) ExecutionResult() *scheduledsdk.ExecutionResult {
	if s == nil {
		return &scheduledsdk.ExecutionResult{}
	}
	resultPayload, _ := json.Marshal(map[string]interface{}{
		"session_id": s.sessionID,
		"tool_calls": len(s.toolCalls),
	})
	return &scheduledsdk.ExecutionResult{
		ExecutorRunID: s.sessionID,
		OutputSummary: scheduledAgentSessionSummary(
			s.sessionID,
			s.content.String(),
			s.toolCalls,
			s.errors,
		),
		ResultPayload: resultPayload,
	}
}

func scheduledAgentSessionSummary(sessionID string, content string, toolCalls []dto.WorkspaceChatToolCallSummary, errors []string) string {
	parts := make([]string, 0, 4)
	if sessionID != "" {
		parts = append(parts, "session_id="+sessionID)
	}
	content = compactScheduledAgentSummary(content)
	if content != "" {
		parts = append(parts, content)
	}
	if len(toolCalls) > 0 {
		errorCount := 0
		for _, call := range toolCalls {
			if call.Status == ToolCallStatusError {
				errorCount++
			}
		}
		parts = append(parts, fmt.Sprintf("工具调用 %d 次，失败 %d 次", len(toolCalls), errorCount))
	}
	if len(errors) > 0 {
		parts = append(parts, "错误: "+strings.Join(errors, "；"))
	}
	if len(parts) == 0 {
		return "Agent 任务执行完成"
	}
	return strings.Join(parts, "；")
}

func compactScheduledAgentSummary(content string) string {
	content = strings.Join(strings.Fields(content), " ")
	const max = 240
	if len(content) <= max {
		return content
	}
	return content[:max] + "..."
}

func workspaceStreamSessionData(data interface{}) dto.WorkspaceStreamSession {
	switch v := data.(type) {
	case dto.WorkspaceStreamSession:
		return v
	case *dto.WorkspaceStreamSession:
		if v != nil {
			return *v
		}
	}
	return dto.WorkspaceStreamSession{}
}

func workspaceStreamContentData(data interface{}) dto.WorkspaceStreamContent {
	switch v := data.(type) {
	case dto.WorkspaceStreamContent:
		return v
	case *dto.WorkspaceStreamContent:
		if v != nil {
			return *v
		}
	}
	return dto.WorkspaceStreamContent{}
}

func workspaceStreamToolCallData(data interface{}) dto.WorkspaceStreamToolCall {
	switch v := data.(type) {
	case dto.WorkspaceStreamToolCall:
		return v
	case *dto.WorkspaceStreamToolCall:
		if v != nil {
			return *v
		}
	}
	return dto.WorkspaceStreamToolCall{}
}

func workspaceStreamDoneData(data interface{}) dto.WorkspaceStreamDone {
	switch v := data.(type) {
	case dto.WorkspaceStreamDone:
		return v
	case *dto.WorkspaceStreamDone:
		if v != nil {
			return *v
		}
	}
	return dto.WorkspaceStreamDone{}
}

func workspaceStreamErrorData(data interface{}) dto.WorkspaceStreamError {
	switch v := data.(type) {
	case dto.WorkspaceStreamError:
		return v
	case *dto.WorkspaceStreamError:
		if v != nil {
			return *v
		}
	}
	return dto.WorkspaceStreamError{}
}
