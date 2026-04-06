package service

import (
	"context"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
)

type ListScheduledTasksTool struct{}

type listScheduledTasksArgs struct {
	FullCodePath string `json:"full_code_path" schema_desc:"工作台路径前缀"`
	Status       string `json:"status" schema_desc:"任务状态" schema_enum:"pending,done,failed,cancelled"`
	Page         *int   `json:"page" schema_desc:"页码"`
	PageSize     *int   `json:"page_size" schema_desc:"每页条数"`
}

var listScheduledTasksToolDef = toolDefinition[listScheduledTasksArgs](
	"list_scheduled_tasks",
	"查询定时任务列表。full_code_path 可不传（默认当前工作台路径）。传入路径时返回该路径本身及所有子路径下的任务（例如在目录节点也能看到子目录/子表单上挂的定时任务）。可按 status 过滤。",
)

func (t *ListScheduledTasksTool) Definition() dto.ToolDef {
	return listScheduledTasksToolDef
}

func (t *ListScheduledTasksTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[listScheduledTasksArgs](call.Args)
	if err != nil {
		return toolResult("list_scheduled_tasks 参数解析失败: "+err.Error(), true)
	}
	content, isError := runListScheduledTasksTool(ctx, args, call.FullCodePath)
	return toolResult(content, isError)
}

// runListScheduledTasksTool 查询定时任务列表（工作台工具）
func runListScheduledTasksTool(ctx context.Context, args listScheduledTasksArgs, currentFullCodePath string) (string, bool) {
	fullCodePath := resolveFullCodePathArg(args.FullCodePath, currentFullCodePath)
	status := strings.TrimSpace(args.Status)
	page := 1
	if args.Page != nil && *args.Page > 0 {
		page = *args.Page
	}
	pageSize := 20
	if args.PageSize != nil && *args.PageSize > 0 {
		pageSize = *args.PageSize
	}

	resp, err := apicall.ListScheduledTasks(ctx, fullCodePath, status, page, pageSize)
	if err != nil {
		return "list_scheduled_tasks 调用失败: " + err.Error(), true
	}
	out := map[string]interface{}{
		"total": resp.Total,
		"list":  resp.List,
	}
	return formatJSONResult(out)
}
