package service

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
)

type CancelScheduledTaskTool struct{}

type cancelScheduledTaskArgs struct {
	TaskID int64 `json:"task_id" schema_desc:"任务 ID" schema_required:"true"`
}

var cancelScheduledTaskToolDef = toolDefinition[cancelScheduledTaskArgs](
	"cancel_scheduled_task",
	"取消定时任务（仅创建人可取消）。",
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
		return "cancel_scheduled_task 调用失败: " + err.Error(), true
	}
	return "已取消定时任务。", false
}
