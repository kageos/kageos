package service

import (
	"context"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
)

type CancelScheduledTaskTool struct{}

type cancelScheduledTaskArgs struct {
	TaskID int64 `json:"task_id" schema_desc:"任务 ID" schema_required:"true"`
}

var cancelScheduledTaskToolDef = toolDefinition[cancelScheduledTaskArgs](
	"cancel_scheduled_task",
	"取消普通函数/表格/表单定时 task（create_scheduled_task 创建的任务，仅创建人可取消）。如果要取消定时会话任务/session task，应使用 cancel_scheduled_session_task。",
)

func (t *CancelScheduledTaskTool) Definition() dto.ToolDef {
	return cancelScheduledTaskToolDef
}

func (t *CancelScheduledTaskTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[cancelScheduledTaskArgs](call.Args)
	if err != nil {
		return toolResult("cancel_scheduled_task 参数解析失败: "+err.Error(), true)
	}
	content, isError := runCancelScheduledTaskTool(ctx, args)
	return toolResult(content, isError)
}

// runCancelScheduledTaskTool 取消定时任务（工作台工具）
func runCancelScheduledTaskTool(ctx context.Context, args cancelScheduledTaskArgs) (string, bool) {
	if args.TaskID <= 0 {
		return "cancel_scheduled_task 需传 task_id（正整数）。", true
	}
	if err := apicall.CancelScheduledTask(ctx, args.TaskID); err != nil {
		msg := "cancel_scheduled_task 调用失败: " + err.Error()
		if strings.Contains(strings.ToLower(err.Error()), "record not found") {
			msg += "。如果这个 ID 来自 list_scheduled_agent_tasks，它是定时会话任务/session task，请改用 cancel_scheduled_session_task。"
		}
		return msg, true
	}
	return "已取消定时任务。", false
}
