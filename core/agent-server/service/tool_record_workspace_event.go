package service

import (
	"context"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type workspaceContextKey struct{}

// WorkspaceSessionIDKey 工作台会话 ID 的 context key（在 executeToolCalls 中注入，record_workspace_event 中读取）
var WorkspaceSessionIDKey = workspaceContextKey{}

func getWorkspaceSessionID(ctx context.Context) string {
	if v := ctx.Value(WorkspaceSessionIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

type RecordWorkspaceEventTool struct {
	eventRepo *repository.WorkspaceEventRepository
}

type recordWorkspaceEventArgs struct {
	EventType   string `json:"event_type" schema_desc:"事件类型" schema_required:"true"`
	Description string `json:"description" schema_desc:"一句话描述" schema_required:"true"`
	Context     string `json:"context" schema_desc:"上下文摘要"`
	Extra       string `json:"extra" schema_desc:"额外 JSON 字符串"`
}

var recordWorkspaceEventToolDef = toolDefinition[recordWorkspaceEventArgs](
	"record_workspace_event",
	"记录工作台内事件，用于产品分析与改进。当判断需求无法实现或需求不明确时，在回复用户前调用。event_type 必填：unsupported_demand（平台无法实现）、unclear_requirement（需求不明确需澄清）、task_failed（执行失败）等；description 必填（一句话说明）；context、extra 可选。",
)

func (t *RecordWorkspaceEventTool) Definition() dto.ToolDef {
	return recordWorkspaceEventToolDef
}

func (t *RecordWorkspaceEventTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[recordWorkspaceEventArgs](call.Args)
	if err != nil {
		return toolResult("record_workspace_event 参数解析失败: "+err.Error(), true)
	}
	content, isError := runRecordWorkspaceEventTool(ctx, t.eventRepo, args, call.FullCodePath)
	return toolResult(content, isError)
}

func runRecordWorkspaceEventTool(ctx context.Context, eventRepo *repository.WorkspaceEventRepository, args recordWorkspaceEventArgs, fullCodePath string) (string, bool) {
	eventType := strings.TrimSpace(args.EventType)
	description := strings.TrimSpace(args.Description)
	if eventType == "" || description == "" {
		return "record_workspace_event 必填 event_type 和 description。", true
	}
	contextStr := args.Context
	extra := args.Extra
	sessionID := getWorkspaceSessionID(ctx)
	user := contextx.GetRequestUser(ctx)

	e := &model.WorkspaceEvent{
		SessionID:    sessionID,
		FullCodePath: fullCodePath,
		User:         user,
		EventType:    eventType,
		Description:  description,
		Context:      contextStr,
		Extra:        extra,
	}
	if eventRepo != nil {
		if err := eventRepo.Create(ctx, e); err != nil {
			logger.Warnf(ctx, "[workspace_event] 落库失败: %v", err)
		}
	}
	logger.Infof(ctx, "[workspace_event] event_type=%s session_id=%s full_code_path=%s description=%s",
		eventType, sessionID, fullCodePath, description)
	return "已记录。", false
}
