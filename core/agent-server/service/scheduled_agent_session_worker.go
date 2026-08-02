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

const scheduledAgentWorkerConcurrency = 4

const (
	scheduledAgentDefaultTokenTTL = 24 * time.Hour
	scheduledAgentTokenGrace      = 30 * time.Minute
)

const scheduledAgentUnattendedPrefix = `【Agent 任务执行约束】
本次任务由自动执行触发，当前是在执行无人值守任务，不是在创建或管理定时任务，也不是与在线用户讨论方案。先选择能完成业务执行的角色；需要调用已有 Form/Table/Chart 或连接器时，通常进入 app_operator。

【离线与人工接管】
用户当前不在线，不能实时回答问题或确认操作。不要向用户提问、等待补充信息，或把本轮停在“请确认/请提供”。任务说明指定的负责人优先；未指定时，按动态工作环境中的目录管理员、目录创建人、任务创建人/当前用户顺序选择人工接管人，不得编造联系人。需要人工决定时可以发通知征求后续决定，但本轮不得等待回复、自动重试高风险动作或据此修改代码；应停止未获授权的动作，由人工在后续会话接管。

【通知：重要事项优先不漏发】
通知治理以不漏掉重要事项为优先，不能为了减少噪音压掉可能造成资损、安全/隐私/合规问题、数据损坏或丢失、大范围业务影响、关键任务持续失败/阻塞、权限异常，或明确需要人工决定的事项。这些情况必须调用 send_notification；影响暂时不能完全确认但存在上述合理可能时，也应至少发送 warning 通知。任务明确要求的通知同样必须发送。首次基准记录、无变化、无待处理对象、普通成功状态和普通执行摘要默认静默，只写执行记录；同一问题没有新进展、影响扩大或任务约定的提醒周期时不要重复通知。send_notification 是单向通知，不是等待回复的在线对话。

通知必须让另一个 Agent 或人工在新会话中直接接管，正文简要包含：任务与目录背景、检查对象/时间范围、已确认事实和证据、影响或风险、已经完成/尝试过的动作、明确没有执行的动作及原因、需要接管人决定或处理的具体事项。来源任务、目录和会话元数据由平台自动附带。

【数据读取】
读取范围由任务目标决定：要求“全部”“某个完整时间窗口”或需要总体统计时，必须分页读到覆盖目标范围并核对数量；只要求最近 N 条或抽样时应按该范围停止。不能只读第一页却声称已完成全量分析，也不要把所有任务机械扩大成全量读取。

【禁止自修改】
无人值守运行中禁止修改 Go 代码、普通文本、构建应用、创建/删除目录或删除文件。发现应用 bug、schema/权限问题或需要代码修复时，只收集证据、记录真正重要且仍未解决的问题、通知负责人并停止相关高风险动作。文件工具在 app_operator 下只允许维护当前目录的 .docs 运行记忆。

【运行记忆】
确需维护长期运行记忆时，只能用 read_file/edit_file/write_file 读写当前目录的 .docs：file_name/code 使用有意义的英文标识，中文项目的工作台 name 使用清楚的中文名称；只保留当前仍存在、确实重要的问题，问题解决后直接删除对应内容，不维护“已解决”列表，也不要把普通无变化结果写成长久噪音。

若创建时的信息不足以安全执行，按已知上下文完成可安全部分，并在执行记录中明确缺失信息、未执行动作和原因；涉及高风险写入且缺少必要确认时必须跳过。`

type scheduledAgentWorkspaceRootContextKey struct{}
type scheduledAgentSessionIdentityContextKey struct{}

type scheduledAgentSessionIdentity struct {
	TaskID    int64
	TaskCode  string
	TaskTitle string
}

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
		Concurrency: scheduledAgentWorkerConcurrency,
		Handler:     chatSvc.RunScheduledAgentSession,
		OnError: func(ctx context.Context, err error) {
			logger.Warnf(ctx, "[ScheduledAgentSessionWorker] %v", err)
		},
	})
}

// RunScheduledAgentSession executes one timer-scheduler agent.session event.
func (s *WorkspaceChatService) RunScheduledAgentSession(ctx context.Context, event scheduledsdk.ExecutionRequestedEvent) (*scheduledsdk.ExecutionResult, error) {
	req, payload, err := scheduledAgentSessionWorkspaceRequest(event)
	if err != nil {
		return nil, err
	}
	if err := event.IssueToken(scheduledAgentExecutionTokenTTL(payload.MaxDurationSeconds)); err != nil {
		return nil, fmt.Errorf("签发 Agent 任务执行令牌失败: %w", err)
	}
	ctx = event.WithAuditContext(ctx)
	if payload.MaxDurationSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(payload.MaxDurationSeconds)*time.Second)
		defer cancel()
	}

	sink := &scheduledAgentSessionSink{}
	logger.Infof(ctx, "[ScheduledAgentSessionWorker] start task_id=%d execution_id=%d full_code_path=%s",
		event.TaskID, event.ExecutionID, req.FullCodePath)
	ctx = contextWithScheduledAgentWorkspaceRoot(ctx, req.FullCodePath)
	ctx = contextWithScheduledAgentSessionIdentity(ctx, event)
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

func scheduledAgentExecutionTokenTTL(maxDurationSeconds int64) time.Duration {
	if maxDurationSeconds <= 0 {
		return scheduledAgentDefaultTokenTTL
	}
	return time.Duration(maxDurationSeconds)*time.Second + scheduledAgentTokenGrace
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
		return "\n【Agent 任务通知对象】本次 Agent 任务没有创建人/请求用户可作为默认接收人。优先使用任务说明明确的负责人；未指定时使用动态工作环境中的目录管理员，其次目录创建人。调用 send_notification 时必须显式传 to_users。若这些信息也未配置，不得编造接收人；在执行记录中明确写出重要通知未能送达及原因。"
	}
	return fmt.Sprintf("\n【Agent 任务通知对象】本次 Agent 任务创建人/默认通知对象：%s。任务说明明确负责人时以任务说明为准；否则需要目录问题人工接管时优先通知动态工作环境中的目录管理员，其次目录创建人，再次使用本默认通知对象。通知默认对象时可省略 to_users；如果显式传，请使用 to_users: %q。", requestUser, requestUser)
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

func contextWithScheduledAgentSessionIdentity(ctx context.Context, event scheduledsdk.ExecutionRequestedEvent) context.Context {
	metadata := event.Metadata
	title := strings.TrimSpace(metadata["task_title"])
	if title == "" && event.TaskID > 0 {
		title = fmt.Sprintf("自动化 Agent #%d", event.TaskID)
	}
	identity := scheduledAgentSessionIdentity{
		TaskID:    event.TaskID,
		TaskCode:  strings.TrimSpace(firstNonEmptyString(metadata["schedule_code"], metadata["bundle_task_code"])),
		TaskTitle: title,
	}
	return context.WithValue(ctx, scheduledAgentSessionIdentityContextKey{}, identity)
}

func scheduledAgentSessionIdentityFromContext(ctx context.Context) scheduledAgentSessionIdentity {
	if ctx == nil {
		return scheduledAgentSessionIdentity{}
	}
	identity, _ := ctx.Value(scheduledAgentSessionIdentityContextKey{}).(scheduledAgentSessionIdentity)
	return identity
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
