package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/msgx"
	"github.com/kageos/kageos/pkg/subjects"
)

type SendNotificationTool struct {
	publisher toolMessagePublisher
}

type sendNotificationArgs struct {
	ToUsers     string `json:"to_users" schema_desc:"接收用户 username，多个用逗号分隔，例如 alice,bob。当前上下文有请求用户或会话创建人时，通知当前用户/创建人/我可省略；没有默认用户时才必须显式填写。不要为了给当前用户发通知而追问用户是谁。"`
	Title       string `json:"title" schema_desc:"通知标题，简短说明发生了什么" schema_required:"true"`
	Message     string `json:"message" schema_desc:"通知正文。支持 Markdown；content_type=html 时可以传已生成的 HTML 片段" schema_required:"true"`
	ContentType string `json:"content_type" schema_desc:"正文格式：markdown、html 或 text；默认 markdown" schema_enum:"markdown,html,text"`
	Level       string `json:"level" schema_desc:"通知级别：info=普通提醒，warning=需要注意，critical=高优先级；默认 info" schema_enum:"info,warning,critical"`
}

type sendNotificationResultData struct {
	Status             string `json:"status" schema_desc:"提交状态" schema_required:"true"`
	ToUsers            string `json:"to_users" schema_desc:"接收用户" schema_required:"true"`
	RecipientCount     int    `json:"recipient_count" schema_desc:"接收用户数量" schema_required:"true"`
	Title              string `json:"title" schema_desc:"最终通知标题" schema_required:"true"`
	ContentType        string `json:"content_type" schema_desc:"正文格式" schema_required:"true"`
	Level              string `json:"level" schema_desc:"通知级别" schema_required:"true"`
	ClientSource       string `json:"client_source,omitempty" schema_desc:"来源入口"`
	SourceType         string `json:"source_type,omitempty" schema_desc:"消息来源类型"`
	SourceRef          string `json:"source_ref,omitempty" schema_desc:"消息来源引用"`
	SourcePath         string `json:"source_path,omitempty" schema_desc:"来源目录或函数路径"`
	SourceTitle        string `json:"source_title,omitempty" schema_desc:"来源展示名"`
	ThreadKey          string `json:"thread_key,omitempty" schema_desc:"站内信聚合线程键"`
	WorkspaceSessionID string `json:"workspace_session_id,omitempty" schema_desc:"关联工作台会话 ID"`
}

var sendNotificationToolDef = toolDefinitionWithOutput[sendNotificationArgs, structuredToolResultSchema[sendNotificationResultData]](
	"send_notification",
	"发送一条单向通知给用户，不等待回复。适合 Agent 任务或无人值守 Agent 在发现高优先级情报、风险、异常，或任务明确要求通知时主动提醒用户；不要用它询问用户并等待回复。当前上下文有请求用户或会话创建人时，通知当前用户/创建人/我可省略 to_users；没有默认用户时才必须显式传 to_users。首次基准记录、无变化结果、普通状态报告默认不通知。多个 username 用逗号分隔。通知来源会继承当前工作台/定时任务上下文，不会归到某个通知函数目录。content_type=html 时站内信会按安全清洗后的 HTML 渲染。",
)

func (t *SendNotificationTool) Definition() dto.ToolDef {
	return sendNotificationToolDef
}

func (t *SendNotificationTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[sendNotificationArgs](call.Args)
	if err != nil {
		return toolResult("send_notification 参数解析失败: "+err.Error(), true)
	}
	return runSendNotificationTool(ctx, t.publisher, args, call.FullCodePath)
}

func runSendNotificationTool(ctx context.Context, publisher toolMessagePublisher, args sendNotificationArgs, currentFullCodePath string) ToolResult {
	if publisher == nil {
		return toolResult("send_notification 当前不可用：message-service NATS 发送器未初始化。", true)
	}
	title := normalizeNotifyTitle(args.Title, args.Level)
	message := strings.TrimSpace(args.Message)
	if title == "" {
		return toolResult("send_notification 需传 title。", true)
	}
	if message == "" {
		return toolResult("send_notification 需传 message。", true)
	}
	contentType, err := normalizeNotifyContentType(args.ContentType)
	if err != nil {
		return toolResult(err.Error(), true)
	}
	level := normalizeNotifyLevel(args.Level)
	toUsers, recipientCount, err := resolveNotifyUsers(ctx, args.ToUsers)
	if err != nil {
		return toolResult(err.Error(), true)
	}

	meta := buildNotifyMessageMeta(ctx, title, currentFullCodePath)
	envelope := &dto.MessageSendEnvelope{
		Meta: meta,
		Message: dto.MessageSendPayload{
			ToUsers:     toUsers,
			Title:       title,
			Content:     message,
			ContentType: contentType,
		},
	}
	msg, err := msgx.BuildJSONRequest(ctx, subjects.MessageSendCommandSubject, envelope)
	if err != nil {
		return toolResult("send_notification 构建消息失败: "+err.Error(), true)
	}
	if err := publisher.PublishMsg(msg); err != nil {
		return toolResult("send_notification 提交消息失败: "+err.Error(), true)
	}

	data := sendNotificationResultData{
		Status:             "submitted",
		ToUsers:            toUsers,
		RecipientCount:     recipientCount,
		Title:              title,
		ContentType:        contentType,
		Level:              level,
		ClientSource:       meta.ClientSource,
		SourceType:         meta.SourceType,
		SourceRef:          meta.SourceRef,
		SourcePath:         meta.SourcePath,
		SourceTitle:        meta.SourceTitle,
		ThreadKey:          meta.ThreadKey,
		WorkspaceSessionID: meta.WorkspaceSessionID,
	}
	logger.Infof(ctx, "[SendNotificationTool] submitted to_users=%s title=%s source_type=%s source_ref=%s thread_key=%s",
		toUsers, title, meta.SourceType, meta.SourceRef, meta.ThreadKey)
	return toolResultWithStructuredData(data, false, "通知已提交给 message-service。")
}

func normalizeNotifyContentType(contentType string) (string, error) {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if contentType == "" {
		return "markdown", nil
	}
	switch contentType {
	case "markdown", "html", "text":
		return contentType, nil
	default:
		return "", fmt.Errorf("send_notification content_type 只支持 markdown、html、text。")
	}
}

func normalizeNotifyLevel(level string) string {
	level = strings.ToLower(strings.TrimSpace(level))
	switch level {
	case "critical", "warning", "info":
		return level
	default:
		return "info"
	}
}

func normalizeNotifyTitle(title string, level string) string {
	title = strings.TrimSpace(title)
	switch normalizeNotifyLevel(level) {
	case "critical":
		if !strings.HasPrefix(title, "【高优先级】") && !strings.HasPrefix(title, "【紧急】") {
			title = "【高优先级】" + title
		}
	case "warning":
		if !strings.HasPrefix(title, "【提醒】") && !strings.HasPrefix(title, "【注意】") {
			title = "【提醒】" + title
		}
	}
	return title
}

func resolveNotifyUsers(ctx context.Context, toUsers string) (string, int, error) {
	users := normalizeNotifyUsers(toUsers)
	if users != "" {
		return users, countNotifyUsers(users), nil
	}
	requestUser := strings.TrimSpace(contextx.GetRequestUser(ctx))
	if !hasWorkspaceRequestUser(requestUser) {
		return "", 0, fmt.Errorf("send_notification 无法默认接收人：当前上下文没有请求用户或会话创建人。请显式传 to_users，多个用户用逗号分隔。%s", notifyRecipientContextHint(ctx, requestUser))
	}
	users = normalizeNotifyUsers(requestUser)
	return users, countNotifyUsers(users), nil
}

func notifyRecipientContextHint(ctx context.Context, requestUser string) string {
	parts := []string{
		"context_user=" + defaultNotifyHintValue(requestUser),
		"client_source=" + defaultNotifyHintValue(contextx.GetAuditClientSource(ctx)),
		"source_type=" + defaultNotifyHintValue(contextx.GetSourceType(ctx)),
		"source_ref=" + defaultNotifyHintValue(contextx.GetSourceRef(ctx)),
		"workspace_session_id=" + defaultNotifyHintValue(contextx.GetWorkspaceSessionID(ctx)),
	}
	if title := strings.TrimSpace(contextx.GetWorkspaceSessionTitle(ctx)); title != "" {
		parts = append(parts, "workspace_session_title="+title)
	}
	return "当前上下文：" + strings.Join(parts, "，") + "。"
}

func defaultNotifyHintValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "<empty>"
	}
	return value
}

func normalizeNotifyUsers(toUsers string) string {
	toUsers = strings.NewReplacer(
		"，", ",",
		"、", ",",
		";", ",",
		"；", ",",
		"\n", ",",
		"\t", ",",
	).Replace(strings.TrimSpace(toUsers))
	parts := strings.Split(toUsers, ",")
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		user := strings.TrimSpace(part)
		if user == "" {
			continue
		}
		if _, ok := seen[user]; ok {
			continue
		}
		seen[user] = struct{}{}
		out = append(out, user)
	}
	return strings.Join(out, ",")
}

func countNotifyUsers(toUsers string) int {
	if toUsers = strings.TrimSpace(toUsers); toUsers == "" {
		return 0
	}
	return len(strings.Split(toUsers, ","))
}

func buildNotifyMessageMeta(ctx context.Context, title string, currentFullCodePath string) dto.MessageSendMeta {
	requestUser := strings.TrimSpace(contextx.GetRequestUser(ctx))
	from := requestUser
	if from == "" {
		from = "system"
	}
	sourceType := strings.TrimSpace(contextx.GetSourceType(ctx))
	sourceRef := strings.TrimSpace(contextx.GetSourceRef(ctx))
	workspaceSessionID := strings.TrimSpace(contextx.GetWorkspaceSessionID(ctx))
	if sourceType == "" && sourceRef == "" && workspaceSessionID != "" {
		sourceType = contextx.SourceTypeAgentTool
		sourceRef = workspaceSessionID
	}
	effectiveFullCodePath := firstNonEmptyString(currentFullCodePath, contextx.GetSourcePath(ctx), contextx.GetSourceParentPath(ctx))
	sourcePath := firstNonEmptyString(contextx.GetSourcePath(ctx), effectiveFullCodePath)
	sourceTitle := firstNonEmptyString(
		contextx.GetSourceTitle(ctx),
		contextx.GetWorkspaceSessionTitle(ctx),
		defaultNotifySourceTitle(sourceType, title),
	)
	return dto.MessageSendMeta{
		From:                  from,
		RequestUser:           requestUser,
		DepartmentFullPath:    strings.TrimSpace(contextx.GetRequestDepartmentFullPath(ctx)),
		FullCodePath:          strings.TrimSpace(effectiveFullCodePath),
		TraceID:               strings.TrimSpace(contextx.GetTraceId(ctx)),
		ClientSource:          strings.TrimSpace(contextx.GetAuditClientSource(ctx)),
		SourceType:            sourceType,
		SourceRef:             sourceRef,
		SourcePath:            strings.TrimSpace(sourcePath),
		SourceTitle:           strings.TrimSpace(sourceTitle),
		SourceParentPath:      strings.TrimSpace(contextx.GetSourceParentPath(ctx)),
		SourceParentTitle:     strings.TrimSpace(contextx.GetSourceParentTitle(ctx)),
		SourceTemplateType:    strings.TrimSpace(contextx.GetSourceTemplateType(ctx)),
		WorkspaceSessionID:    workspaceSessionID,
		WorkspaceSessionTitle: strings.TrimSpace(contextx.GetWorkspaceSessionTitle(ctx)),
		WorkspaceRole:         strings.TrimSpace(contextx.GetWorkspaceRole(ctx)),
		ThreadKey:             notifyMessageThreadKey(sourceType, sourceRef, workspaceSessionID, sourcePath),
	}
}

func defaultNotifySourceTitle(sourceType string, title string) string {
	switch strings.TrimSpace(sourceType) {
	case contextx.SourceTypeScheduledTask:
		return "Agent 任务通知"
	case contextx.SourceTypeAgentTool:
		return "工作台会话通知"
	default:
		return strings.TrimSpace(title)
	}
}

func notifyMessageThreadKey(sourceType string, sourceRef string, workspaceSessionID string, sourcePath string) string {
	sourceType = strings.TrimSpace(sourceType)
	sourceRef = strings.TrimSpace(sourceRef)
	if sourceType == contextx.SourceTypeScheduledTask && sourceRef != "" {
		return "scheduled_task:" + scheduledTaskThreadRef(sourceRef)
	}
	if workspaceSessionID = strings.TrimSpace(workspaceSessionID); workspaceSessionID != "" {
		return "workspace_session:" + workspaceSessionID
	}
	if sourceRef != "" {
		return sourceType + ":" + sourceRef
	}
	if sourcePath = strings.TrimSpace(sourcePath); sourcePath != "" {
		return "source:" + sourcePath
	}
	return ""
}

func scheduledTaskThreadRef(sourceRef string) string {
	parts := strings.Split(strings.TrimSpace(sourceRef), ":")
	if len(parts) >= 2 && parts[0] == "timer_task" && strings.TrimSpace(parts[1]) != "" {
		return "timer_task:" + strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(sourceRef)
}

func decodeNotifyEnvelope(data []byte) (*dto.MessageSendEnvelope, error) {
	var envelope dto.MessageSendEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	return &envelope, nil
}
