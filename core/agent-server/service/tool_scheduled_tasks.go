package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/kageos/kageos/pkg/serviceconfig"
)

type CreateScheduledFunctionTaskTool struct{}
type CreateScheduledAgentTaskTool struct{}
type UpdateScheduledAgentTaskTool struct{}
type ListScheduledTasksTool struct{}
type ManageScheduledTaskTool struct{}
type ListScheduledTaskExecutionsTool struct{}

type scheduledTaskScheduleArgs struct {
	ScheduleType    string `json:"schedule_type" schema_desc:"计划类型，可不传。只填 run_at 会自动当一次性任务；只填 cron_expr 会自动当 cron 周期任务；只填 interval_seconds 会自动当每隔 N 秒任务" schema_enum:"atime,cron,every"`
	RunAt           string `json:"run_at" schema_desc:"一次性任务的执行时间，例如 2026-06-10 09:30:00 或 RFC3339；无时区时按 timezone 解析"`
	CronExpr        string `json:"cron_expr" schema_desc:"cron 周期表达式，例如每分钟 * * * * *，每天 9 点 0 9 * * *"`
	IntervalSeconds int64  `json:"interval_seconds" schema_desc:"每隔多少秒执行一次，例如 60 表示每分钟"`
	Timezone        string `json:"timezone" schema_desc:"可选 IANA 时区；不传则使用平台部署时区。cron 和无时区 run_at 会使用它"`
	MaxRuns         int    `json:"max_runs" schema_desc:"最多执行次数；0 表示不限制，atime 通常为 1"`
}

type createScheduledFunctionTaskArgs struct {
	FullCodePath    string `json:"full_code_path" schema_desc:"要定时执行的具体函数路径，必须是已确认的 .form/.table/.chart，例如 /system/app/ticket_submit.form；不是目录路径"`
	Action          string `json:"action" schema_desc:"通常不传，默认 execute。Form 提交、Chart 查询、普通函数执行用 execute；只有 Table 写入才传 table_create/table_update/table_delete" schema_enum:"execute,table_create,table_update,table_delete"`
	Body            string `json:"body" schema_desc:"函数执行参数 JSON 字符串，直接放业务字段，不要包一层 body/invoke_params/payload。Form 示例：{\"title\":\"测试工单\",\"priority\":\"中\"}；表格写入按 run_table_create/update/delete 的 body 语义传"`
	ScheduleType    string `json:"schedule_type" schema_desc:"计划类型，可不传。只填 run_at 会自动当一次性任务；只填 cron_expr 会自动当 cron 周期任务；只填 interval_seconds 会自动当每隔 N 秒任务" schema_enum:"atime,cron,every"`
	RunAt           string `json:"run_at" schema_desc:"一次性任务的执行时间，例如 2026-06-10 09:30:00 或 RFC3339；无时区时按 timezone 解析"`
	CronExpr        string `json:"cron_expr" schema_desc:"cron 周期表达式，例如每分钟 * * * * *，每天 9 点 0 9 * * *"`
	IntervalSeconds int64  `json:"interval_seconds" schema_desc:"每隔多少秒执行一次，例如 60 表示每分钟"`
	Timezone        string `json:"timezone" schema_desc:"可选 IANA 时区；不传则使用平台部署时区。cron 和无时区 run_at 会使用它"`
	MaxRuns         int    `json:"max_runs" schema_desc:"最多执行次数；0 表示不限制，atime 通常为 1"`
	Title           string `json:"title" schema_desc:"任务名称，不传则自动生成"`
	Description     string `json:"description" schema_desc:"任务说明"`
	IdempotencyKey  string `json:"idempotency_key" schema_desc:"可选幂等 key；不传则根据目标、计划和参数生成"`

	FunctionPath  string      `json:"function_path" schema_ignore:"true"`
	Cron          string      `json:"cron" schema_ignore:"true"`
	TaskName      string      `json:"task_name" schema_ignore:"true"`
	InvokeParams  interface{} `json:"invoke_params" schema_ignore:"true"`
	Payload       interface{} `json:"payload" schema_ignore:"true"`
	MaxExecutions interface{} `json:"max_executions" schema_ignore:"true"`
}

type createScheduledAgentTaskArgs struct {
	FullCodePath       string `json:"full_code_path" schema_desc:"Agent 任务运行的工作台目录或应用路径，例如 /system/app；通常就是当前 execute_directory，不是具体 .form/.table/.chart 函数路径"`
	Title              string `json:"title" schema_desc:"Agent 任务名称，用来在列表里识别这条任务，例如 每日热点推送"`
	Message            string `json:"message" schema_desc:"到点后交给工作台 Agent 执行的完整说明。必须像无人值守运行手册一样写清：任务性质、长期目标、绑定目录/资源（路径可用 </full/code/path> 标记）、可用的其他目录/函数/连接器、预期使用工具清单（如 change_role/search/read_dir/web_search/run_table_search/run_table_create/run_table_update/run_form_submit/run_chart_query/send_notification）、执行步骤、按业务场景裁剪的质量控制策略、失败处理、输出格式、通知规则。Agent 任务运行时会注入任务创建人/请求用户作为默认通知对象；需要通知创建人、当前用户或“我”时，send_notification 可省略 to_users，也可显式传创建人 username；首次基准记录、无变化结果、普通状态报告默认不通知。运行时用户不在线，不能写“到时候问我/等待确认”。如果只是固定调用一个函数，请改用 create_scheduled_function_task"`
	ScheduleType       string `json:"schedule_type" schema_desc:"计划类型，可不传。只填 run_at 会自动当一次性任务；只填 cron_expr 会自动当 cron 周期任务；只填 interval_seconds 会自动当每隔 N 秒任务" schema_enum:"atime,cron,every"`
	RunAt              string `json:"run_at" schema_desc:"一次性任务的执行时间，例如 2026-06-10 09:30:00 或 RFC3339；无时区时按 timezone 解析"`
	CronExpr           string `json:"cron_expr" schema_desc:"cron 周期表达式，例如每分钟 * * * * *，每天 9 点 0 9 * * *"`
	IntervalSeconds    int64  `json:"interval_seconds" schema_desc:"每隔多少秒执行一次，例如 60 表示每分钟"`
	Timezone           string `json:"timezone" schema_desc:"可选 IANA 时区；不传则使用平台部署时区。cron 和无时区 run_at 会使用它"`
	MaxRuns            int    `json:"max_runs" schema_desc:"最多执行次数；0 表示不限制，atime 通常为 1"`
	ModeCode           string `json:"mode_code" schema_desc:"工作台模式，通常不用传，默认 dev"`
	Files              string `json:"files" schema_desc:"Agent 任务需要带上的附件 refs，多个用逗号分隔；没有附件就不传"`
	LLMConfigID        int64  `json:"llm_config_id" schema_desc:"可选 LLM 配置 ID"`
	MaxDurationSeconds int64  `json:"max_duration_seconds" schema_desc:"可选最大执行时长秒数"`
	Description        string `json:"description" schema_desc:"任务说明"`
	IdempotencyKey     string `json:"idempotency_key" schema_desc:"可选幂等 key；不传则根据目标、计划和参数生成"`
	Directory          string `json:"directory" schema_ignore:"true"`
}

type updateScheduledAgentTaskArgs struct {
	TaskID             int64  `json:"task_id" schema_desc:"需要更新的定时任务 ID" schema_required:"true"`
	Title              string `json:"title" schema_desc:"可选：更新任务名称，不传则不更新"`
	Message            string `json:"message" schema_desc:"可选：更新交给工作台 Agent 执行的完整说明。请提供完整的新内容"`
	ScheduleType       string `json:"schedule_type" schema_desc:"可选：更新计划类型。如果不更新计划，此项及以下计划参数请勿传" schema_enum:"atime,cron,every"`
	RunAt              string `json:"run_at" schema_desc:"可选：一次性任务的执行时间"`
	CronExpr           string `json:"cron_expr" schema_desc:"可选：cron 周期表达式"`
	IntervalSeconds    int64  `json:"interval_seconds" schema_desc:"可选：每隔多少秒执行一次"`
	Timezone           string `json:"timezone" schema_desc:"可选：IANA 时区"`
	MaxRuns            int    `json:"max_runs" schema_desc:"可选：最多执行次数"`
	ModeCode           string `json:"mode_code" schema_desc:"可选：工作台模式"`
	Files              string `json:"files" schema_desc:"可选：附件 refs"`
	LLMConfigID        int64  `json:"llm_config_id" schema_desc:"可选：LLM 配置 ID"`
	MaxDurationSeconds int64  `json:"max_duration_seconds" schema_desc:"可选：最大执行时长秒数"`
	Description        string `json:"description" schema_desc:"可选：任务说明"`
}

func (args updateScheduledAgentTaskArgs) scheduleArgs() scheduledTaskScheduleArgs {
	return scheduledTaskScheduleArgs{
		ScheduleType:    args.ScheduleType,
		RunAt:           args.RunAt,
		CronExpr:        args.CronExpr,
		IntervalSeconds: args.IntervalSeconds,
		Timezone:        args.Timezone,
		MaxRuns:         args.MaxRuns,
	}
}

type listScheduledTasksArgs struct {
	Kind         string `json:"kind" schema_desc:"任务类型：function=函数任务，agent_session=Agent 任务，all=全部" schema_enum:"function,agent_session,all"`
	ResourcePath string `json:"resource_path" schema_desc:"资源完整路径；不传则使用当前 execute_directory"`
	Status       string `json:"status" schema_desc:"任务状态过滤" schema_enum:"pending,paused,done,failed,cancelled"`
	Page         int    `json:"page" schema_desc:"页码，默认 1"`
	PageSize     int    `json:"page_size" schema_desc:"每页条数，默认 20，最多 100"`
}

type manageScheduledTaskArgs struct {
	TaskID       int64  `json:"task_id" schema_desc:"定时任务 ID" schema_required:"true"`
	Action       string `json:"action" schema_desc:"管理动作：pause 暂停、resume 开启、cancel 取消但保留记录、delete 删除并从列表移除、run_now 立即运行一次" schema_enum:"pause,resume,cancel,delete,run_now" schema_required:"true"`
	ResourcePath string `json:"resource_path" schema_desc:"可选资源路径；传入后会校验任务归属，避免误操作"`
}

type listScheduledTaskExecutionsArgs struct {
	TaskID   int64  `json:"task_id" schema_desc:"定时任务 ID" schema_required:"true"`
	Status   string `json:"status" schema_desc:"执行状态过滤" schema_enum:"queued,running,success,failed,timeout,cancelled"`
	Page     int    `json:"page" schema_desc:"页码，默认 1"`
	PageSize int    `json:"page_size" schema_desc:"每页条数，默认 20，最多 100"`
}

var createScheduledFunctionTaskToolDef = toolDefinition[createScheduledFunctionTaskArgs](
	"create_scheduled_function_task",
	"创建【函数任务】：到点后直接调用一个已确认的 Form/Table/Chart 函数，也就是固定函数路径 + 固定 body。适合“定时提交这个表单”“每天查这个图表”“每周新增这些表格记录”这类固定动作。只要目标已经能写成一次 run_form_submit/run_table_create/run_table_update/run_table_delete/run_chart_query，就优先用这个工具，不要创建 Agent 任务。不要用于需要 Agent 到时候判断、搜索多个资源、总结分析或临场决定步骤的任务，那种用 create_scheduled_agent_task。定时任务运行时用户不在线，创建前必须问清所有必要参数和确认项；不要创建会在执行时还需要用户回答的问题。创建前必须用 search 确认函数 schema、必填字段和枚举；body 直接传业务 JSON 字符串，不要传 invoke_params 包装。周期性写入任务必须先向用户复述执行对象、频率、参数、最大次数和取消方式，并等用户明确确认后再创建。工具只创建 timer-scheduler 任务，不会立即执行业务写入；后续执行会以 source=scheduled_task 进入操作日志。",
)

var createScheduledAgentTaskToolDef = toolDefinition[createScheduledAgentTaskArgs](
	"create_scheduled_agent_task",
	"创建【Agent 任务】：到点后启动一个 Agent 工作台会话，并把 message 当作执行说明交给工作台 Agent。核心参数是 title + message；其它参数只是目录、时间、附件、模型等配置。每 N 秒执行请传 interval_seconds，例如每 5 分钟传 interval_seconds=300；不要把这些参数包进 body。适合“模型库每 6 小时巡检全球厂商和新模型并可信写入”“每天整理新闻日报”“每周读取业务数据生成周报”“定期巡检工单/订单/库存异常”这类需要 Agent 判断、查询、总结、维护长期数据或组合多个动作的任务。Agent 任务可以编排当前目录、本空间其他目录、其他空间函数、系统工具和连接器函数，message 里的资源路径可用 </full/code/path> 轻量标记。Agent 任务是无人值守执行，运行时用户无法回答问题；创建前必须把范围、可用资源/函数、预期使用工具清单、必要参数、按业务场景裁剪的质量控制策略、输出格式、失败处理和风险确认问清楚，并全部写进 message，不能留下“到时候问用户/等待确认”。message 必须明确列出预计使用哪些工具以及何时使用，例如 change_role、read_dir/search、web_search、run_table_search、run_table_create/run_table_update、run_form_submit、run_chart_query、send_notification；Agent 任务运行时会注入任务创建人/请求用户作为默认通知对象；send_notification 通知创建人、当前用户或“我”时可省略 to_users，也可显式传创建人 username；首次基准记录、无变化结果、普通状态报告默认不通知；质量规则要结合业务，不要把示例机械套到所有任务。不要用于已明确的单个 Form/Table/Chart 函数调用；如果目标能写成固定函数路径 + 固定 body，请用 create_scheduled_function_task，更稳定也更便宜。创建前必须确认 full_code_path 是目标工作空间/目录，不是具体函数路径；message 不能包含未授权的跨应用操作。",
)

var updateScheduledAgentTaskToolDef = toolDefinition[updateScheduledAgentTaskArgs](
	"update_scheduled_agent_task",
	"更新【Agent 任务】配置：修改名称、执行说明（message）或定时计划。如果不修改计划，请不要传 schedule_type 及相关的 cron_expr 等参数。更新执行说明时，必须提供完整的新内容，不能只写追加的部分。",
)

var listScheduledTasksToolDef = toolDefinition[listScheduledTasksArgs](
	"list_scheduled_tasks",
	"查询指定目录下全部定时任务，不按创建人过滤。kind=function 查函数任务，kind=agent_session 查 Agent 任务，kind=all 查全部；默认按当前 execute_directory 及其子资源查询，也可按 resource_path 和 status 过滤。目录路径会返回绑定到该目录的 Agent 任务，以及目录下函数的函数任务。",
)

var manageScheduledTaskToolDef = toolDefinition[manageScheduledTaskArgs](
	"manage_scheduled_task",
	"管理当前用户创建的定时任务：pause 暂停、resume 开启、cancel 取消但保留记录、delete 删除并从列表移除、run_now 立即强制运行一次。操作前先用 list_scheduled_tasks 确认 task_id 和资源归属；run_now 会创建一次新的手动执行记录，即使已有 inflight 执行也会继续提交；delete 只删除任务配置，不等待旧执行完成。",
)

var listScheduledTaskExecutionsToolDef = toolDefinition[listScheduledTaskExecutionsArgs](
	"list_scheduled_task_executions",
	"查询某个定时任务的执行记录，用于回答“最近跑了没、成功没、失败原因是什么”。这是只读诊断查询，不按创建人过滤；可按 queued/running/success/failed/timeout/cancelled 过滤。",
)

func (t *CreateScheduledFunctionTaskTool) Definition() dto.ToolDef {
	return createScheduledFunctionTaskToolDef
}
func (t *CreateScheduledAgentTaskTool) Definition() dto.ToolDef {
	return createScheduledAgentTaskToolDef
}
func (t *UpdateScheduledAgentTaskTool) Definition() dto.ToolDef {
	return updateScheduledAgentTaskToolDef
}
func (t *ListScheduledTasksTool) Definition() dto.ToolDef  { return listScheduledTasksToolDef }
func (t *ManageScheduledTaskTool) Definition() dto.ToolDef { return manageScheduledTaskToolDef }
func (t *ListScheduledTaskExecutionsTool) Definition() dto.ToolDef {
	return listScheduledTaskExecutionsToolDef
}

func (t *CreateScheduledFunctionTaskTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[createScheduledFunctionTaskArgs](call.Args)
	if err != nil {
		return toolResult("create_scheduled_function_task 参数解析失败: "+err.Error(), true)
	}
	return runCreateScheduledFunctionTask(ctx, args, call.FullCodePath)
}

func (t *CreateScheduledAgentTaskTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	if err := rejectCreateScheduledAgentTaskUnknownArgs(call.Args); err != nil {
		return toolResult("create_scheduled_agent_task 参数错误: "+err.Error(), true)
	}
	args, err := decodeToolArgs[createScheduledAgentTaskArgs](call.Args)
	if err != nil {
		return toolResult("create_scheduled_agent_task 参数解析失败: "+err.Error(), true)
	}
	return runCreateScheduledAgentTask(ctx, args, call.FullCodePath)
}

func (t *UpdateScheduledAgentTaskTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[updateScheduledAgentTaskArgs](call.Args)
	if err != nil {
		return toolResult("update_scheduled_agent_task 参数解析失败: "+err.Error(), true)
	}
	return runUpdateScheduledAgentTask(ctx, args)
}

func (t *ListScheduledTasksTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[listScheduledTasksArgs](call.Args)
	if err != nil {
		return toolResult("list_scheduled_tasks 参数解析失败: "+err.Error(), true)
	}
	return runListScheduledTasks(ctx, args, call.FullCodePath)
}

func (t *ManageScheduledTaskTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[manageScheduledTaskArgs](call.Args)
	if err != nil {
		return toolResult("manage_scheduled_task 参数解析失败: "+err.Error(), true)
	}
	return runManageScheduledTask(ctx, args)
}

func (t *ListScheduledTaskExecutionsTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[listScheduledTaskExecutionsArgs](call.Args)
	if err != nil {
		return toolResult("list_scheduled_task_executions 参数解析失败: "+err.Error(), true)
	}
	return runListScheduledTaskExecutions(ctx, args)
}

func (args createScheduledFunctionTaskArgs) scheduleArgs() scheduledTaskScheduleArgs {
	return scheduledTaskScheduleArgs{
		ScheduleType:    args.ScheduleType,
		RunAt:           args.RunAt,
		CronExpr:        args.CronExpr,
		IntervalSeconds: args.IntervalSeconds,
		Timezone:        args.Timezone,
		MaxRuns:         args.MaxRuns,
	}
}

func (args createScheduledAgentTaskArgs) scheduleArgs() scheduledTaskScheduleArgs {
	return scheduledTaskScheduleArgs{
		ScheduleType:    args.ScheduleType,
		RunAt:           args.RunAt,
		CronExpr:        args.CronExpr,
		IntervalSeconds: args.IntervalSeconds,
		Timezone:        args.Timezone,
		MaxRuns:         args.MaxRuns,
	}
}

func runCreateScheduledFunctionTask(ctx context.Context, args createScheduledFunctionTaskArgs, currentFullCodePath string) ToolResult {
	ctx = withAgentToolClientSource(ctx)
	args = normalizeCreateScheduledFunctionTaskArgs(args)
	fullCodePath := resolveFullCodePathArg(args.FullCodePath, currentFullCodePath)
	if fullCodePath == "" {
		return toolResult("create_scheduled_function_task 需传 full_code_path。", true)
	}
	action := strings.TrimSpace(args.Action)
	if action == "" {
		action = "execute"
	}
	if !isScheduledFunctionAction(action) {
		return toolResult("create_scheduled_function_task action 仅支持 execute/table_create/table_update/table_delete。", true)
	}
	if requiresScheduledFunctionBody(fullCodePath, action) && strings.TrimSpace(args.Body) == "" {
		return toolResult("create_scheduled_function_task 创建写入类函数定时任务时必须传 body，且 body 要直接包含已确认的业务字段；不要传空参数或 invoke_params 包装。", true)
	}
	payload, err := parseScheduledPayload(args.Body)
	if err != nil {
		return toolResult("create_scheduled_function_task body 需为合法 JSON: "+err.Error(), true)
	}
	schedule, err := buildScheduledTaskSchedule(args.scheduleArgs())
	if err != nil {
		return toolResult("create_scheduled_function_task 计划参数错误: "+err.Error(), true)
	}
	if err := requireScheduledTaskPermission(ctx, fullCodePath, scheduledFunctionRequiredAction(fullCodePath, action)); err != nil {
		return toolResult("create_scheduled_function_task 权限校验失败: "+err.Error(), true)
	}
	user, err := scheduledTaskCurrentUser(ctx)
	if err != nil {
		return toolResult(err.Error(), true)
	}
	executorPayload := map[string]interface{}{
		"full_code_path": fullCodePath,
		"template_type":  scheduledFunctionTemplateType(fullCodePath),
		"action":         action,
		"method":         methodForScheduledFunctionAction(action),
		"payload":        payload,
	}
	req := scheduledsdk.CreateTaskRequest{
		Title:           firstNonEmptyString(args.Title, defaultScheduledFunctionTitle(fullCodePath)),
		Description:     strings.TrimSpace(args.Description),
		Category:        "scheduled_function",
		Tags:            []string{"function", action},
		IdempotencyKey:  scheduledIdempotencyKey(args.IdempotencyKey, "function", fullCodePath, action, schedule, payload),
		ExecutorKey:     "app.function",
		ExecutorPayload: mustRawJSON(executorPayload),
		Metadata: map[string]string{
			"kind":          "scheduled_function",
			"action":        action,
			"method":        methodForScheduledFunctionAction(action),
			"template_type": scheduledFunctionTemplateType(fullCodePath),
		},
		Schedule:        schedule,
		SourceType:      "function",
		SourceRef:       fullCodePath,
		ResourceScope:   "function",
		ResourceKey:     fullCodePath,
		RequestUser:     user,
		RequestUserDept: contextx.GetRequestDepartmentFullPath(ctx),
		CreatedBy:       user,
	}
	task, err := scheduledTaskClient().CreateTask(ctx, req)
	if err != nil {
		return toolResult("create_scheduled_function_task 调用失败: "+err.Error(), true)
	}
	return toolResultWithStructuredData(map[string]interface{}{
		"task":        task,
		"next_run_at": task.NextRunAt,
		"message":     "函数任务已创建",
	}, false)
}

func runCreateScheduledAgentTask(ctx context.Context, args createScheduledAgentTaskArgs, currentFullCodePath string) ToolResult {
	ctx = withAgentToolClientSource(ctx)
	args = normalizeCreateScheduledAgentTaskArgs(args)
	fullCodePath := resolveFullCodePathArg(args.FullCodePath, currentFullCodePath)
	if fullCodePath == "" {
		return toolResult("create_scheduled_agent_task 需传 full_code_path。", true)
	}
	message := strings.TrimSpace(args.Message)
	if message == "" {
		return toolResult("create_scheduled_agent_task 需传 message。message 是到点后交给 Agent 的执行说明。", true)
	}
	schedule, err := buildScheduledTaskSchedule(args.scheduleArgs())
	if err != nil {
		return toolResult("create_scheduled_agent_task 计划参数错误: "+err.Error(), true)
	}
	if err := requireScheduledTaskPermission(ctx, fullCodePath, access.ActionWrite); err != nil {
		return toolResult("create_scheduled_agent_task 权限校验失败: "+err.Error(), true)
	}
	user, err := scheduledTaskCurrentUser(ctx)
	if err != nil {
		return toolResult(err.Error(), true)
	}
	modeCode := strings.TrimSpace(args.ModeCode)
	if modeCode == "" {
		modeCode = "dev"
	}
	executorPayload := map[string]interface{}{
		"full_code_path":  fullCodePath,
		"message":         message,
		"display_content": message,
	}
	if modeCode != "" && modeCode != "dev" {
		executorPayload["mode_code"] = modeCode
	}
	if files := strings.TrimSpace(args.Files); files != "" {
		executorPayload["files"] = files
	}
	if args.LLMConfigID > 0 {
		executorPayload["llm_config_id"] = args.LLMConfigID
	}
	if args.MaxDurationSeconds > 0 {
		executorPayload["max_duration_seconds"] = args.MaxDurationSeconds
	}
	req := scheduledsdk.CreateTaskRequest{
		Title:           firstNonEmptyString(args.Title, defaultScheduledAgentTitle(fullCodePath, message)),
		Description:     strings.TrimSpace(args.Description),
		Category:        "scheduled_agent_session",
		Tags:            []string{"agent", "session"},
		IdempotencyKey:  scheduledIdempotencyKey(args.IdempotencyKey, "agent_session", fullCodePath, modeCode, schedule, message),
		ExecutorKey:     "agent.session",
		ExecutorPayload: mustRawJSON(executorPayload),
		Metadata: map[string]string{
			"kind":      "scheduled_agent_session",
			"mode_code": modeCode,
		},
		Status:          scheduledsdk.TaskStatusPaused,
		Schedule:        schedule,
		SourceType:      "agent_session",
		SourceRef:       fullCodePath,
		ResourceScope:   "workspace_directory",
		ResourceKey:     fullCodePath,
		RequestUser:     user,
		RequestUserDept: contextx.GetRequestDepartmentFullPath(ctx),
		CreatedBy:       user,
	}
	task, err := scheduledTaskClient().CreateTask(ctx, req)
	if err != nil {
		return toolResult("create_scheduled_agent_task 调用失败: "+err.Error(), true)
	}
	return toolResultWithStructuredData(map[string]interface{}{
		"task":        task,
		"next_run_at": task.NextRunAt,
		"message":     "Agent 任务已创建，默认暂停；需要启用后才会无人值守执行。",
	}, false)
}

func runListScheduledTasks(ctx context.Context, args listScheduledTasksArgs, currentFullCodePath string) ToolResult {
	ctx = withAgentToolClientSource(ctx)
	kind := normalizeScheduledTaskKind(args.Kind)
	resourcePath := resolveFullCodePathArg(args.ResourcePath, currentFullCodePath)
	req := listScheduledTasksRequest(kind, resourcePath, strings.TrimSpace(args.Status), args.Page, args.PageSize)
	resp, err := listScheduledTasksAllPages(ctx, scheduledTaskClient(), req)
	if err != nil {
		return toolResult("list_scheduled_tasks 调用失败: "+err.Error(), true)
	}
	return toolResultWithStructuredData(resp, false)
}

func listScheduledTasksRequest(kind string, resourcePath string, status string, page int, pageSize int) scheduledsdk.ListTasksRequest {
	req := scheduledsdk.ListTasksRequest{
		Status:   strings.TrimSpace(status),
		Page:     page,
		PageSize: pageSize,
	}
	if kind != "all" {
		req.ExecutorKey = scheduledExecutorKeyForKind(kind)
	}
	if resourcePath != "" {
		req.ResourceKeyPrefix = resourcePath
	}
	if kind != "all" {
		req.ResourceScope = scheduledResourceScopeForKind(kind)
	}
	return req
}

func listScheduledTasksAllPages(ctx context.Context, client *scheduledsdk.Client, req scheduledsdk.ListTasksRequest) (*scheduledsdk.ListTasksResponse, error) {
	if client == nil {
		client = scheduledTaskClient()
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 100
	}
	var out scheduledsdk.ListTasksResponse
	for page := 1; ; page++ {
		req.Page = page
		resp, err := client.ListTasks(ctx, req)
		if err != nil {
			return nil, err
		}
		if resp == nil || len(resp.List) == 0 {
			break
		}
		out.List = append(out.List, resp.List...)
		out.Total = resp.Total
		if resp.Total > 0 && int64(len(out.List)) >= resp.Total {
			break
		}
		if len(resp.List) < req.PageSize {
			break
		}
	}
	if out.Total == 0 {
		out.Total = int64(len(out.List))
	}
	return &out, nil
}

func runManageScheduledTask(ctx context.Context, args manageScheduledTaskArgs) ToolResult {
	ctx = withAgentToolClientSource(ctx)
	if args.TaskID <= 0 {
		return toolResult("manage_scheduled_task 需传合法 task_id。", true)
	}
	client := scheduledTaskClient()
	task, err := client.GetTask(ctx, args.TaskID)
	if err != nil {
		return toolResult("manage_scheduled_task 查询任务失败: "+err.Error(), true)
	}
	if err := ensureScheduledTaskOwnedByCurrentUser(ctx, task); err != nil {
		return toolResult("manage_scheduled_task 权限校验失败: "+err.Error(), true)
	}
	if resourcePath := resolveFullCodePathArg(args.ResourcePath, ""); resourcePath != "" && task.ResourceKey != resourcePath {
		return toolResult(fmt.Sprintf("manage_scheduled_task 任务资源不匹配：任务属于 %s，不是 %s。", task.ResourceKey, resourcePath), true)
	}
	switch strings.TrimSpace(args.Action) {
	case "pause":
		err = client.PauseTask(ctx, args.TaskID)
	case "resume":
		err = client.ResumeTask(ctx, args.TaskID)
	case "cancel":
		err = client.CancelTask(ctx, args.TaskID)
	case "delete":
		err = client.DeleteTask(ctx, args.TaskID)
	case "run_now":
		exec, runErr := client.RunNow(ctx, args.TaskID)
		if runErr != nil {
			return toolResult("manage_scheduled_task 立即运行失败: "+runErr.Error(), true)
		}
		return toolResultWithStructuredData(map[string]interface{}{
			"task_id":   args.TaskID,
			"execution": exec,
			"message":   "已提交立即运行",
		}, false)
	default:
		return toolResult("manage_scheduled_task action 仅支持 pause/resume/cancel/delete/run_now。", true)
	}
	if err != nil {
		return toolResult("manage_scheduled_task 调用失败: "+err.Error(), true)
	}
	if strings.TrimSpace(args.Action) == "delete" {
		return toolResultWithStructuredData(map[string]interface{}{
			"task_id": args.TaskID,
			"action":  args.Action,
			"message": "定时任务已删除",
		}, false)
	}
	updated, getErr := client.GetTask(ctx, args.TaskID)
	if getErr != nil {
		return toolResultWithStructuredData(map[string]interface{}{
			"task_id": args.TaskID,
			"action":  args.Action,
			"message": "操作已提交，但刷新任务详情失败: " + getErr.Error(),
		}, false)
	}
	return toolResultWithStructuredData(map[string]interface{}{
		"task":    updated,
		"message": "定时任务已" + scheduledManageActionLabel(args.Action),
	}, false)
}

func runListScheduledTaskExecutions(ctx context.Context, args listScheduledTaskExecutionsArgs) ToolResult {
	ctx = withAgentToolClientSource(ctx)
	if args.TaskID <= 0 {
		return toolResult("list_scheduled_task_executions 需传合法 task_id。", true)
	}
	client := scheduledTaskClient()
	if _, err := client.GetTask(ctx, args.TaskID); err != nil {
		return toolResult("list_scheduled_task_executions 查询任务失败: "+err.Error(), true)
	}
	resp, err := client.ListExecutions(ctx, args.TaskID, scheduledsdk.ListExecutionsRequest{
		Status:   strings.TrimSpace(args.Status),
		Page:     args.Page,
		PageSize: args.PageSize,
	})
	if err != nil {
		return toolResult("list_scheduled_task_executions 调用失败: "+err.Error(), true)
	}
	return toolResultWithStructuredData(resp, false)
}

func scheduledTaskClient() *scheduledsdk.Client {
	return scheduledsdk.NewClient(scheduledsdk.Options{
		BaseURL: serviceconfig.BuildGatewayURL("/timer/api/v1"),
	})
}

func buildScheduledTaskSchedule(args scheduledTaskScheduleArgs) (scheduledsdk.Schedule, error) {
	args = normalizeScheduledTaskScheduleArgs(args)
	scheduleType := scheduledsdk.ScheduleType(strings.TrimSpace(args.ScheduleType))
	schedule := scheduledsdk.Schedule{
		Type:    scheduleType,
		MaxRuns: args.MaxRuns,
	}
	switch scheduleType {
	case scheduledsdk.ScheduleAt:
		runAt, err := parseScheduledRunAt(args.RunAt, args.Timezone)
		if err != nil {
			return schedule, err
		}
		schedule.RunAt = runAt
	case scheduledsdk.ScheduleCron:
		schedule.CronExpr = strings.TrimSpace(args.CronExpr)
		schedule.Timezone = scheduledTaskTimezone(args.Timezone)
	case scheduledsdk.ScheduleEvery:
		schedule.IntervalSeconds = args.IntervalSeconds
	default:
		return schedule, fmt.Errorf("schedule_type is required unless exactly one of run_at, cron_expr or interval_seconds is provided")
	}
	return schedule, schedule.Validate()
}

func normalizeCreateScheduledFunctionTaskArgs(args createScheduledFunctionTaskArgs) createScheduledFunctionTaskArgs {
	if strings.TrimSpace(args.FullCodePath) == "" {
		args.FullCodePath = strings.TrimSpace(args.FunctionPath)
	}
	if strings.TrimSpace(args.CronExpr) == "" {
		args.CronExpr = strings.TrimSpace(args.Cron)
	}
	if strings.TrimSpace(args.Title) == "" {
		args.Title = strings.TrimSpace(args.TaskName)
	}
	if args.MaxRuns == 0 {
		args.MaxRuns = parseScheduledCompatInt(args.MaxExecutions)
	}
	if strings.TrimSpace(args.Body) == "" {
		args.Body = scheduledBodyFromCompatValue(args.InvokeParams)
	}
	if strings.TrimSpace(args.Body) == "" {
		args.Body = scheduledBodyFromCompatValue(args.Payload)
	}
	args.ScheduleType = normalizeScheduledTaskScheduleArgs(args.scheduleArgs()).ScheduleType
	return args
}

func normalizeCreateScheduledAgentTaskArgs(args createScheduledAgentTaskArgs) createScheduledAgentTaskArgs {
	if strings.TrimSpace(args.FullCodePath) == "" {
		args.FullCodePath = strings.TrimSpace(args.Directory)
	}
	args.ScheduleType = normalizeScheduledTaskScheduleArgs(args.scheduleArgs()).ScheduleType
	return args
}

func rejectCreateScheduledAgentTaskUnknownArgs(args map[string]interface{}) error {
	allowed := map[string]struct{}{
		"full_code_path":       {},
		"directory":            {},
		"title":                {},
		"message":              {},
		"schedule_type":        {},
		"run_at":               {},
		"cron_expr":            {},
		"interval_seconds":     {},
		"timezone":             {},
		"max_runs":             {},
		"mode_code":            {},
		"files":                {},
		"llm_config_id":        {},
		"max_duration_seconds": {},
		"description":          {},
		"idempotency_key":      {},
	}
	for key := range args {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("不支持参数 %q；Agent 任务只接收顶层 title、message、full_code_path（directory 仅兼容旧别名）和计划配置，间隔执行请用 interval_seconds", key)
		}
	}
	return nil
}

func normalizeScheduledTaskScheduleArgs(args scheduledTaskScheduleArgs) scheduledTaskScheduleArgs {
	if strings.TrimSpace(args.ScheduleType) != "" {
		return args
	}
	hasRunAt := strings.TrimSpace(args.RunAt) != ""
	hasCron := strings.TrimSpace(args.CronExpr) != ""
	hasEvery := args.IntervalSeconds > 0
	switch {
	case hasRunAt && !hasCron && !hasEvery:
		args.ScheduleType = string(scheduledsdk.ScheduleAt)
	case hasCron && !hasRunAt && !hasEvery:
		args.ScheduleType = string(scheduledsdk.ScheduleCron)
	case hasEvery && !hasRunAt && !hasCron:
		args.ScheduleType = string(scheduledsdk.ScheduleEvery)
	}
	return args
}

func parseScheduledRunAt(raw string, timezone string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, fmt.Errorf("run_at is required")
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	location := time.Local
	if tz := scheduledTaskTimezone(timezone); tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			location = loc
		}
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02T15:04:05", "2006-01-02T15:04"} {
		if t, err := time.ParseInLocation(layout, value, location); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("run_at must be RFC3339 or YYYY-MM-DD HH:mm:ss")
}

func scheduledTaskTimezone(raw string) string {
	tz := strings.TrimSpace(raw)
	if tz == "" {
		return ""
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return ""
	}
	return tz
}

func parseScheduledPayload(raw string) (interface{}, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return map[string]interface{}{}, nil
	}
	var out interface{}
	if err := json.Unmarshal([]byte(value), &out); err != nil {
		return nil, err
	}
	if out == nil {
		return map[string]interface{}{}, nil
	}
	return out, nil
}

func scheduledBodyFromCompatValue(value interface{}) string {
	raw := scheduledRawJSONFromCompatValue(value)
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	return unwrapScheduledInvokeParamsBody(raw)
}

func scheduledRawJSONFromCompatValue(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(data))
	}
}

func unwrapScheduledInvokeParamsBody(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	var wrapper map[string]interface{}
	if err := json.Unmarshal([]byte(value), &wrapper); err != nil {
		return value
	}
	for _, key := range []string{"body", "payload"} {
		if nested, ok := wrapper[key]; ok {
			nestedRaw := scheduledRawJSONFromCompatValue(nested)
			if strings.TrimSpace(nestedRaw) != "" {
				return nestedRaw
			}
		}
	}
	return value
}

func parseScheduledCompatInt(value interface{}) int {
	normalize := func(n int) int {
		if n > 0 {
			return n
		}
		return 0
	}
	switch v := value.(type) {
	case nil:
		return 0
	case int:
		return normalize(v)
	case int64:
		return normalize(int(v))
	case float64:
		return normalize(int(v))
	case json.Number:
		n, _ := v.Int64()
		return normalize(int(n))
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil {
			return normalize(n)
		}
	}
	return 0
}

func mustRawJSON(value interface{}) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return data
}

func scheduledIdempotencyKey(explicit string, parts ...interface{}) string {
	if key := strings.TrimSpace(explicit); key != "" {
		return key
	}
	data, _ := json.Marshal(parts)
	sum := sha1.Sum(data)
	return "agent-scheduled-" + hex.EncodeToString(sum[:])[:24]
}

func isScheduledFunctionAction(action string) bool {
	switch strings.TrimSpace(action) {
	case "execute", "table_create", "table_update", "table_delete":
		return true
	default:
		return false
	}
}

func requiresScheduledFunctionBody(fullCodePath, action string) bool {
	switch strings.TrimSpace(action) {
	case "table_create", "table_update", "table_delete":
		return true
	case "execute":
		return strings.HasSuffix(strings.TrimSpace(fullCodePath), ".form")
	default:
		return false
	}
}

func methodForScheduledFunctionAction(action string) string {
	switch strings.TrimSpace(action) {
	case "table_update":
		return "PUT"
	case "table_delete":
		return "DELETE"
	default:
		return "POST"
	}
}

func scheduledFunctionRequiredAction(fullCodePath string, action string) access.Action {
	switch strings.TrimSpace(action) {
	case "table_update":
		return access.ActionUpdate
	case "table_delete":
		return access.ActionDelete
	case "execute":
		if scheduledFunctionTemplateType(fullCodePath) == "chart" {
			return access.ActionRead
		}
		return access.ActionWrite
	default:
		return access.ActionWrite
	}
}

func scheduledFunctionTemplateType(fullCodePath string) string {
	path := strings.ToLower(strings.TrimSpace(fullCodePath))
	switch {
	case strings.Contains(path, ".form"):
		return "form"
	case strings.Contains(path, ".table"):
		return "table"
	case strings.Contains(path, ".chart"):
		return "chart"
	default:
		return ""
	}
}

func requireScheduledTaskPermission(ctx context.Context, resourcePath string, action access.Action) error {
	resourcePath = access.NormalizeResourcePath(resourcePath)
	if resourcePath == "" {
		return fmt.Errorf("resource_path is required")
	}
	resp, err := apicall.MyPermissions(ctx, resourcePath)
	if err != nil {
		return err
	}
	if resp == nil || !access.HasPermission(resp.Permissions, action) {
		return fmt.Errorf("当前用户缺少 %s 权限: %s", action, resourcePath)
	}
	return nil
}

func scheduledTaskCurrentUser(ctx context.Context) (string, error) {
	user := strings.TrimSpace(contextx.GetRequestUser(ctx))
	if user == "" {
		return "", fmt.Errorf("无法获取当前用户，不能创建或管理定时任务")
	}
	return user, nil
}

func ensureScheduledTaskOwnedByCurrentUser(ctx context.Context, task *scheduledsdk.Task) error {
	if task == nil {
		return fmt.Errorf("任务不存在")
	}
	user, err := scheduledTaskCurrentUser(ctx)
	if err != nil {
		return err
	}
	if task.CreatedBy == user || task.RequestUser == user {
		return nil
	}
	return fmt.Errorf("只能管理当前用户创建或代执行的定时任务")
}

func normalizeScheduledTaskKind(raw string) string {
	switch strings.TrimSpace(raw) {
	case "function", "agent_session":
		return strings.TrimSpace(raw)
	default:
		return "all"
	}
}

func scheduledExecutorKeyForKind(kind string) string {
	switch kind {
	case "function":
		return "app.function"
	case "agent_session":
		return "agent.session"
	default:
		return ""
	}
}

func scheduledResourceScopeForKind(kind string) string {
	switch kind {
	case "function":
		return "function"
	case "agent_session":
		return "workspace_directory"
	default:
		return ""
	}
}

func defaultScheduledFunctionTitle(fullCodePath string) string {
	name := strings.Trim(strings.TrimSpace(fullCodePath), "/")
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	if name == "" {
		name = "函数"
	}
	return name + " 定时任务"
}

func defaultScheduledAgentTitle(fullCodePath string, message string) string {
	message = strings.TrimSpace(message)
	if message != "" {
		runes := []rune(message)
		if len(runes) > 18 {
			return string(runes[:18]) + "..."
		}
		return message
	}
	name := strings.Trim(strings.TrimSpace(fullCodePath), "/")
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	if name == "" {
		name = "工作台"
	}
	return name + " Agent 任务"
}

func defaultScheduledAgentTitleFromMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	lines := strings.Split(message, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(strings.Trim(line, "#*- 　\t"))
		if line == "" {
			continue
		}
		runes := []rune(line)
		if len(runes) > 40 {
			return string(runes[:40]) + "..."
		}
		return line
	}
	runes := []rune(message)
	if len(runes) > 40 {
		return string(runes[:40]) + "..."
	}
	return message
}

func scheduledManageActionLabel(action string) string {
	switch strings.TrimSpace(action) {
	case "pause":
		return "暂停"
	case "resume":
		return "开启"
	case "cancel":
		return "取消"
	case "delete":
		return "删除"
	default:
		return "更新"
	}
}

func runUpdateScheduledAgentTask(ctx context.Context, args updateScheduledAgentTaskArgs) ToolResult {
	ctx = withAgentToolClientSource(ctx)
	if args.TaskID <= 0 {
		return toolResult("update_scheduled_agent_task 需传合法 task_id。", true)
	}
	client := scheduledTaskClient()
	task, err := client.GetTask(ctx, args.TaskID)
	if err != nil {
		return toolResult("update_scheduled_agent_task 查询任务失败: "+err.Error(), true)
	}
	if err := ensureScheduledTaskOwnedByCurrentUser(ctx, task); err != nil {
		return toolResult("update_scheduled_agent_task 权限校验失败: "+err.Error(), true)
	}
	if task.ExecutorKey != "agent.session" {
		return toolResult("该任务不是 Agent 任务，无法使用此工具更新", true)
	}
	req := scheduledsdk.UpdateTaskRequest{}
	if title := strings.TrimSpace(args.Title); title != "" {
		req.Title = &title
	}
	if desc := strings.TrimSpace(args.Description); desc != "" {
		req.Description = &desc
	}

	hasScheduleArgs := strings.TrimSpace(args.ScheduleType) != "" || strings.TrimSpace(args.RunAt) != "" || strings.TrimSpace(args.CronExpr) != "" || args.IntervalSeconds > 0
	if hasScheduleArgs {
		schedule, err := buildScheduledTaskSchedule(args.scheduleArgs())
		if err != nil {
			return toolResult("update_scheduled_agent_task 计划参数错误: "+err.Error(), true)
		}
		req.Schedule = &schedule
	}

	message := strings.TrimSpace(args.Message)
	modeCode := strings.TrimSpace(args.ModeCode)
	files := strings.TrimSpace(args.Files)

	hasPayloadChanges := message != "" || modeCode != "" || files != "" || args.LLMConfigID > 0 || args.MaxDurationSeconds > 0
	if hasPayloadChanges {
		var payload map[string]interface{}
		if len(task.ExecutorPayload) > 0 {
			if err := json.Unmarshal(task.ExecutorPayload, &payload); err != nil {
				payload = make(map[string]interface{})
			}
		} else {
			payload = make(map[string]interface{})
		}

		if message != "" {
			payload["message"] = message
			payload["display_content"] = message
		}
		if modeCode != "" {
			payload["mode_code"] = modeCode
		}
		if files != "" {
			payload["files"] = files
		}
		if args.LLMConfigID > 0 {
			payload["llm_config_id"] = args.LLMConfigID
		}
		if args.MaxDurationSeconds > 0 {
			payload["max_duration_seconds"] = args.MaxDurationSeconds
		}
		req.ExecutorPayload = mustRawJSON(payload)
	}

	updated, err := client.UpdateTask(ctx, args.TaskID, req)
	if err != nil {
		return toolResult("update_scheduled_agent_task 更新失败: "+err.Error(), true)
	}
	return toolResultWithStructuredData(map[string]interface{}{
		"task":        updated,
		"next_run_at": updated.NextRunAt,
		"message":     "Agent 任务已更新",
	}, false)
}
