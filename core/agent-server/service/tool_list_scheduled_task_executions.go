package service

import (
	"context"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
)

type ListScheduledTaskExecutionsTool struct{}

type listScheduledTaskExecutionsArgs struct {
	TaskID   int64  `json:"task_id" schema_desc:"任务 ID" schema_required:"true"`
	Status   string `json:"status" schema_desc:"执行状态" schema_enum:"success,failed"`
	Page     *int   `json:"page" schema_desc:"页码"`
	PageSize *int   `json:"page_size" schema_desc:"每页条数"`
}

var listScheduledTaskExecutionsToolDef = toolDefinition[listScheduledTaskExecutionsArgs](
	"list_scheduled_task_executions",
	"查询某个定时任务的执行记录。",
)

func (t *ListScheduledTaskExecutionsTool) Definition() dto.ToolDef {
	return listScheduledTaskExecutionsToolDef
}

func (t *ListScheduledTaskExecutionsTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[listScheduledTaskExecutionsArgs](call.Args)
	if err != nil {
		return toolResult("list_scheduled_task_executions 参数解析失败: "+err.Error(), true)
	}
	content, isError := runListScheduledTaskExecutionsTool(ctx, args)
	return toolResult(content, isError)
}

// runListScheduledTaskExecutionsTool 查询任务执行记录（工作台工具）
func runListScheduledTaskExecutionsTool(ctx context.Context, args listScheduledTaskExecutionsArgs) (string, bool) {
	if args.TaskID <= 0 {
		return "list_scheduled_task_executions 需传 task_id（正整数）。", true
	}
	status := strings.TrimSpace(args.Status)
	page := 1
	if args.Page != nil && *args.Page > 0 {
		page = *args.Page
	}
	pageSize := 20
	if args.PageSize != nil && *args.PageSize > 0 {
		pageSize = *args.PageSize
	}

	resp, err := apicall.ListScheduledTaskExecutions(ctx, args.TaskID, status, page, pageSize)
	if err != nil {
		return "list_scheduled_task_executions 调用失败: " + err.Error(), true
	}
	out := map[string]interface{}{
		"task_id": args.TaskID,
		"total":   resp.Total,
		"list":    resp.List,
	}
	return formatJSONResult(out)
}
