package service

import (
	"context"

	"github.com/kageos/kageos/dto"
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
	Message            string `json:"message" schema_desc:"到点后交给工作台 Agent 执行的完整说明。必须像无人值守运行手册一样写清：任务性质、长期目标、绑定目录/资源（路径可用 </full/code/path> 或同目录 <./xxx.form> 标记；内置工具可用 <tool:send_notification> 标记）、可用的其他目录/函数/连接器/内置工具、预期使用工具清单（如 change_role/search/read_dir/web_search/run_table_search/run_table_create/run_table_update/run_form_submit/run_chart_query/send_notification；写给人读的说明里可把通知工具标成 <tool:send_notification>）、执行步骤、按业务场景裁剪的质量控制策略、失败处理、输出格式、通知规则。Agent 任务运行时会注入任务创建人/请求用户作为默认通知对象；需要通知创建人、当前用户或“我”时，send_notification 可省略 to_users，也可显式传创建人 username；首次基准记录、无变化结果、普通状态报告默认不通知。运行时用户不在线，不能写“到时候问我/等待确认”。如果只是固定调用一个函数，请改用 create_scheduled_function_task"`
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
	OverlapPolicy      string `json:"overlap_policy" schema_desc:"上一次尚未完成时的处理策略：forbid 跳过本轮（默认、最安全）；queue_latest 最多合并保留一个待执行；allow 允许并行" schema_enum:"forbid,queue_latest,allow"`
	MaxParallelism     int    `json:"max_parallelism" schema_desc:"overlap_policy=allow 时同一任务最大并行数，默认 2，最大 16"`
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
	OverlapPolicy      string `json:"overlap_policy" schema_desc:"可选：重叠策略 forbid/queue_latest/allow" schema_enum:"forbid,queue_latest,allow"`
	MaxParallelism     int    `json:"max_parallelism" schema_desc:"可选：allow 策略的同任务最大并行数，最大 16"`
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
	Status   string `json:"status" schema_desc:"执行状态过滤" schema_enum:"waiting,queued,running,success,failed,timeout,cancelled,skipped"`
	Page     int    `json:"page" schema_desc:"页码，默认 1"`
	PageSize int    `json:"page_size" schema_desc:"每页条数，默认 20，最多 100"`
}

var createScheduledFunctionTaskToolDef = toolDefinition[createScheduledFunctionTaskArgs](
	"create_scheduled_function_task",
	"创建【函数任务】：到点后直接调用一个已确认的 Form/Table/Chart 函数，也就是固定函数路径 + 固定 body。适合“定时提交这个表单”“每天查这个图表”“每周新增这些表格记录”这类固定动作。只要目标已经能写成一次 run_form_submit/run_table_create/run_table_update/run_table_delete/run_chart_query，就优先用这个工具，不要创建 Agent 任务。不要用于需要 Agent 到时候判断、搜索多个资源、总结分析或临场决定步骤的任务，那种用 create_scheduled_agent_task。定时任务运行时用户不在线，创建前必须问清所有必要参数和确认项；不要创建会在执行时还需要用户回答的问题。创建前必须用 search 确认函数 schema、必填字段和枚举；body 直接传业务 JSON 字符串，不要传 invoke_params 包装。周期性写入任务必须先向用户复述执行对象、频率、参数、最大次数和取消方式，并等用户明确确认后再创建。工具只创建 timer-scheduler 任务，不会立即执行业务写入；后续执行会以 source=scheduled_task 进入操作日志。",
)

var createScheduledAgentTaskToolDef = toolDefinition[createScheduledAgentTaskArgs](
	"create_scheduled_agent_task",
	"创建【Agent 任务】：到点后启动一个 Agent 工作台会话，并把 message 当作执行说明交给工作台 Agent。核心参数是 title + message；其它参数只是目录、时间、附件、模型等配置。每 N 秒执行请传 interval_seconds，例如每 5 分钟传 interval_seconds=300；不要把这些参数包进 body。适合“模型库每 6 小时巡检全球厂商和新模型并可信写入”“每天整理新闻日报”“每周读取业务数据生成周报”“定期巡检工单/订单/库存异常”这类需要 Agent 判断、查询、总结、维护长期数据或组合多个动作的任务。Agent 任务可以编排当前目录、本空间其他目录、其他空间函数、系统工具、连接器函数和内置 Agent 工具；message 里的资源路径可用 </full/code/path> 或同目录 <./xxx.form> 轻量标记，内置工具可用 <tool:send_notification> 标记。Agent 任务是无人值守执行，运行时用户无法回答问题；创建前必须把范围、可用资源/函数、预期使用工具清单、必要参数、按业务场景裁剪的质量控制策略、输出格式、失败处理和风险确认问清楚，并全部写进 message，不能留下“到时候问用户/等待确认”。message 必须明确列出预计使用哪些工具以及何时使用，例如 change_role、read_dir/search、web_search、run_table_search、run_table_create/run_table_update、run_form_submit、run_chart_query、send_notification；写给人读的说明里可把通知工具标成 <tool:send_notification>，真实工具名仍是 send_notification。Agent 任务运行时会注入任务创建人/请求用户作为默认通知对象；send_notification 通知创建人、当前用户或“我”时可省略 to_users，也可显式传创建人 username；首次基准记录、无变化结果、普通状态报告默认不通知；质量规则要结合业务，不要把示例机械套到所有任务。不要用于已明确的单个 Form/Table/Chart 函数调用；如果目标能写成固定函数路径 + 固定 body，请用 create_scheduled_function_task，更稳定也更便宜。创建前必须确认 full_code_path 是目标工作空间/目录，不是具体函数路径；message 不能包含未授权的跨应用操作。",
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
	"查询某个定时任务的执行记录，用于回答“最近跑了没、成功没、失败原因是什么”。这是只读诊断查询，不按创建人过滤；可按 waiting/queued/running/success/failed/timeout/cancelled/skipped 过滤。",
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
