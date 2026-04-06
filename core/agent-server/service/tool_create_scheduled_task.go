package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
)

type CreateScheduledTaskTool struct{}

type createScheduledTaskArgs struct {
	Name            string `json:"name" schema_desc:"任务名称" schema_required:"true"`
	FullCodePath    string `json:"full_code_path" schema_desc:"函数完整路径，不传默认当前目录"`
	Action          string `json:"action" schema_desc:"任务动作" schema_enum:"execute,form,table_create,table_update,table_delete"`
	Method          string `json:"method" schema_desc:"请求方法"`
	Payload         string `json:"payload" schema_desc:"JSON 对象字符串"`
	ScheduleType    string `json:"schedule_type" schema_desc:"调度类型" schema_required:"true" schema_enum:"atime,cron,every"`
	RunAt           string `json:"run_at" schema_desc:"首次执行时间" schema_required:"true"`
	CronExpr        string `json:"cron_expr" schema_desc:"cron 表达式"`
	IntervalSeconds *int   `json:"interval_seconds" schema_desc:"间隔秒数"`
	MaxRuns         *int   `json:"max_runs" schema_desc:"最多执行次数"`
	Timezone        string `json:"timezone" schema_desc:"时区"`
}

var createScheduledTaskToolDef = toolDefinition[createScheduledTaskArgs](
	"create_scheduled_task",
	"创建定时任务。支持 execute/form（普通函数，form 会自动映射为 execute）、table_create（表格新增）、table_update（表格更新）、table_delete（表格删除）。full_code_path 可不传（默认当前目录）。table_update 的 payload 需包含 id 与 updates，执行时会自动补 old_values。run_at 建议用本地日期时间字符串（无 Z），与前端一致；也可用带时区偏移的 RFC3339。",
)

func (t *CreateScheduledTaskTool) Definition() dto.ToolDef {
	return createScheduledTaskToolDef
}

func (t *CreateScheduledTaskTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[createScheduledTaskArgs](call.Args)
	if err != nil {
		return toolResult("create_scheduled_task 参数解析失败: "+err.Error(), true)
	}
	content, isError := runCreateScheduledTaskTool(ctx, args, call.FullCodePath)
	return toolResult(content, isError)
}

// runCreateScheduledTaskTool 创建定时任务
func runCreateScheduledTaskTool(ctx context.Context, args createScheduledTaskArgs, currentFullCodePath string) (string, bool) {
	name := strings.TrimSpace(args.Name)
	if name == "" {
		return "create_scheduled_task 需传 name（任务名称）。", true
	}
	fullCodePath := resolveFullCodePathArg(args.FullCodePath, currentFullCodePath)
	if fullCodePath == "" {
		return "create_scheduled_task 需传 full_code_path，或在可推导当前目录的上下文中调用。", true
	}

	scheduleType := strings.TrimSpace(args.ScheduleType)
	if scheduleType == "" {
		return "create_scheduled_task 需传 schedule_type（atime/cron/every）。", true
	}
	runAt := strings.TrimSpace(args.RunAt)
	if runAt == "" {
		return "create_scheduled_task 需传 run_at（本地时间如 2006-01-02 15:04:05，或带偏移的 RFC3339）。", true
	}

	action := strings.TrimSpace(args.Action)
	if action == "" {
		action = "execute"
	}
	if strings.EqualFold(action, "form") {
		action = "execute"
	}
	method := strings.ToUpper(strings.TrimSpace(args.Method))
	if method == "" {
		method = "POST"
	}

	payloadStr := strings.TrimSpace(args.Payload)
	if payloadStr == "" {
		payloadStr = "{}"
	}
	var payloadRaw json.RawMessage
	if err := json.Unmarshal([]byte(payloadStr), &payloadRaw); err != nil {
		return "create_scheduled_task 的 payload 必须是合法 JSON 对象字符串: " + err.Error(), true
	}

	req := &dto.CreateScheduledTaskReq{
		Name:         name,
		FullCodePath: fullCodePath,
		Action:       action,
		Method:       method,
		Payload:      payloadRaw,
		ScheduleType: scheduleType,
		RunAt:        runAt,
		CronExpr:     strings.TrimSpace(args.CronExpr),
		Timezone:     strings.TrimSpace(args.Timezone),
	}
	if args.IntervalSeconds != nil {
		req.IntervalSeconds = int64(*args.IntervalSeconds)
	}
	if args.MaxRuns != nil {
		req.MaxRuns = *args.MaxRuns
	}

	item, err := apicall.CreateScheduledTask(ctx, req)
	if err != nil {
		return "create_scheduled_task 调用失败: " + err.Error(), true
	}
	out := map[string]interface{}{
		"id":             item.ID,
		"name":           item.Name,
		"full_code_path": item.FullCodePath,
		"action":         item.Action,
		"schedule_type":  item.ScheduleType,
		"status":         item.Status,
		"run_at":         item.RunAt,
		"next_run_at":    item.NextRunAt,
	}
	return formatJSONResult(out)
}
