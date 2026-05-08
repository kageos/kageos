package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
)

const scheduledAgentToolSourceType = "agent_tool"

type CreateScheduledAgentTaskTool struct {
	registry *ToolRegistry
}

type createScheduledAgentTaskArgs struct {
	Name              string      `json:"name" schema_desc:"任务名称" schema_required:"true"`
	FullCodePath      string      `json:"full_code_path" schema_desc:"工作台目录完整路径，不传默认当前目录"`
	Goal              string      `json:"goal" schema_desc:"每次执行时要交给智能体完成的目标/提示词" schema_required:"true"`
	ModeCode          string      `json:"mode_code" schema_desc:"工作台模式，默认 dev"`
	Files             string      `json:"files" schema_desc:"文件引用，bucket/object_key 多文件逗号分隔；不传则自动使用当前用户消息附件"`
	LLMConfigID       *int64      `json:"llm_config_id" schema_desc:"LLM 配置 ID，不传使用默认配置"`
	ContextPolicy     interface{} `json:"context_policy" schema_desc:"上下文策略 JSON 对象，可选"`
	ToolPolicy        interface{} `json:"tool_policy" schema_desc:"工具策略 JSON 对象，可选"`
	ApprovalPolicy    interface{} `json:"approval_policy" schema_desc:"审批策略 JSON 对象，可选"`
	BudgetPolicy      interface{} `json:"budget_policy" schema_desc:"预算策略 JSON 对象，可选；不传或 max_duration_seconds 为 0 时使用服务端默认值，例如 {\"max_duration_seconds\":1800}"`
	ScheduleType      string      `json:"schedule_type" schema_desc:"调度类型" schema_required:"true" schema_enum:"atime,cron,every"`
	RunAt             string      `json:"run_at" schema_desc:"仅 atime 需要：首次执行时间，支持 2006-01-02 15:04:05 或 RFC3339"`
	CronExpr          string      `json:"cron_expr" schema_desc:"cron 表达式，cron 类型必填"`
	IntervalSeconds   *int64      `json:"interval_seconds" schema_desc:"间隔秒数，every 类型必填；建议不要小于 60 秒"`
	MaxRuns           *int        `json:"max_runs" schema_desc:"最多执行次数，0 或不传表示不限"`
	Timezone          string      `json:"timezone" schema_desc:"时区，例如 Asia/Shanghai；不传使用服务端本地时区"`
	NotifyUsers       []string    `json:"notify_users" schema_desc:"执行完成后通知的用户名列表"`
	NotifyDepartments []string    `json:"notify_departments" schema_desc:"执行完成后通知的部门 full_code_path 列表"`
	NotifyOn          string      `json:"notify_on" schema_desc:"通知条件" schema_enum:"none,all,success,failed"`
}

type ListScheduledAgentTasksTool struct {
	registry *ToolRegistry
}

type listScheduledAgentTasksArgs struct {
	FullCodePath string `json:"full_code_path" schema_desc:"工作台路径前缀，不传默认当前工作台路径；当前路径为空时查询自己创建的任务"`
	Status       string `json:"status" schema_desc:"任务状态" schema_enum:"pending,paused,done,failed,cancelled"`
	Page         *int   `json:"page" schema_desc:"页码"`
	PageSize     *int   `json:"page_size" schema_desc:"每页条数"`
}

type ListScheduledAgentTaskExecutionsTool struct {
	registry *ToolRegistry
}

type listScheduledAgentTaskExecutionsArgs struct {
	TaskID   int64  `json:"task_id" schema_desc:"定时智能体任务 ID" schema_required:"true"`
	Status   string `json:"status" schema_desc:"执行状态" schema_enum:"pending,running,success,failed,timeout,cancelled"`
	Page     *int   `json:"page" schema_desc:"页码"`
	PageSize *int   `json:"page_size" schema_desc:"每页条数"`
}

type RunScheduledAgentTaskNowTool struct {
	registry *ToolRegistry
}

type runScheduledAgentTaskNowArgs struct {
	TaskID int64 `json:"task_id" schema_desc:"定时智能体任务 ID" schema_required:"true"`
}

var createScheduledAgentTaskToolDef = toolDefinition[createScheduledAgentTaskArgs](
	"create_scheduled_agent_task",
	"创建定时智能体会话任务：到点后会在指定工作台目录下按 goal 自动发起一轮智能体会话。它用于“定时让工作台/智能体做事”；如果目标是定时执行表格新增/更新/删除、表单提交或图表查询，应使用 create_scheduled_task。full_code_path 可不传（默认当前目录）；files 可不传（默认使用当前消息附件）。",
)

var listScheduledAgentTasksToolDef = toolDefinition[listScheduledAgentTasksArgs](
	"list_scheduled_agent_tasks",
	"查询定时智能体会话任务列表。full_code_path 可不传（默认当前工作台路径），可按 status 过滤。",
)

var listScheduledAgentTaskExecutionsToolDef = toolDefinition[listScheduledAgentTaskExecutionsArgs](
	"list_scheduled_agent_task_executions",
	"查询某个定时智能体会话任务的执行记录，可用于找到自动创建的工作台 session_id。",
)

var runScheduledAgentTaskNowToolDef = toolDefinition[runScheduledAgentTaskNowArgs](
	"run_scheduled_agent_task_now",
	"立即触发某个定时智能体会话任务执行一次。该工具会异步启动执行并返回执行记录；session_id 通常会在后台执行开始后写入，可稍后查询执行记录获取。",
)

func (t *CreateScheduledAgentTaskTool) Definition() dto.ToolDef {
	return createScheduledAgentTaskToolDef
}

func (t *CreateScheduledAgentTaskTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[createScheduledAgentTaskArgs](call.Args)
	if err != nil {
		return toolResult("create_scheduled_agent_task 参数解析失败: "+err.Error(), true)
	}
	content, isError := runCreateScheduledAgentTaskTool(ctx, t.registry, args, call.FullCodePath, call.Files)
	return toolResult(content, isError)
}

func (t *ListScheduledAgentTasksTool) Definition() dto.ToolDef {
	return listScheduledAgentTasksToolDef
}

func (t *ListScheduledAgentTasksTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[listScheduledAgentTasksArgs](call.Args)
	if err != nil {
		return toolResult("list_scheduled_agent_tasks 参数解析失败: "+err.Error(), true)
	}
	content, isError := runListScheduledAgentTasksTool(ctx, t.registry, args, call.FullCodePath)
	return toolResult(content, isError)
}

func (t *ListScheduledAgentTaskExecutionsTool) Definition() dto.ToolDef {
	return listScheduledAgentTaskExecutionsToolDef
}

func (t *ListScheduledAgentTaskExecutionsTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[listScheduledAgentTaskExecutionsArgs](call.Args)
	if err != nil {
		return toolResult("list_scheduled_agent_task_executions 参数解析失败: "+err.Error(), true)
	}
	content, isError := runListScheduledAgentTaskExecutionsTool(ctx, t.registry, args)
	return toolResult(content, isError)
}

func (t *RunScheduledAgentTaskNowTool) Definition() dto.ToolDef {
	return runScheduledAgentTaskNowToolDef
}

func (t *RunScheduledAgentTaskNowTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[runScheduledAgentTaskNowArgs](call.Args)
	if err != nil {
		return toolResult("run_scheduled_agent_task_now 参数解析失败: "+err.Error(), true)
	}
	content, isError := runScheduledAgentTaskNowTool(ctx, t.registry, args)
	return toolResult(content, isError)
}

func runCreateScheduledAgentTaskTool(ctx context.Context, registry *ToolRegistry, args createScheduledAgentTaskArgs, currentFullCodePath string, attachedFiles string) (string, bool) {
	svc, errMsg := scheduledAgentTaskToolService(registry, "create_scheduled_agent_task")
	if errMsg != "" {
		return errMsg, true
	}
	requestUser, requestDept, errMsg := scheduledAgentTaskToolRequestUser(ctx, "create_scheduled_agent_task")
	if errMsg != "" {
		return errMsg, true
	}

	name := strings.TrimSpace(args.Name)
	if name == "" {
		return "create_scheduled_agent_task 需传 name（任务名称）。", true
	}
	fullCodePath := resolveFullCodePathArg(args.FullCodePath, currentFullCodePath)
	if fullCodePath == "" {
		return "create_scheduled_agent_task 需传 full_code_path，或在可推导当前目录的上下文中调用。", true
	}
	goal := strings.TrimSpace(args.Goal)
	if goal == "" {
		return "create_scheduled_agent_task 需传 goal（每次执行目标/提示词）。", true
	}
	scheduleType := strings.ToLower(strings.TrimSpace(args.ScheduleType))
	if scheduleType == "" {
		return "create_scheduled_agent_task 需传 schedule_type（atime/cron/every）。", true
	}
	runAt := strings.TrimSpace(args.RunAt)
	if scheduleType == ScheduledAgentScheduleAtime && runAt == "" {
		return "create_scheduled_agent_task 需传 run_at（如 2006-01-02 15:04:05，或带偏移的 RFC3339）。", true
	}

	contextPolicy, err := scheduledAgentTaskToolJSONObject("context_policy", args.ContextPolicy)
	if err != nil {
		return "create_scheduled_agent_task 参数错误: " + err.Error(), true
	}
	toolPolicy, err := scheduledAgentTaskToolJSONObject("tool_policy", args.ToolPolicy)
	if err != nil {
		return "create_scheduled_agent_task 参数错误: " + err.Error(), true
	}
	approvalPolicy, err := scheduledAgentTaskToolJSONObject("approval_policy", args.ApprovalPolicy)
	if err != nil {
		return "create_scheduled_agent_task 参数错误: " + err.Error(), true
	}
	budgetPolicy, err := scheduledAgentTaskToolJSONObject("budget_policy", args.BudgetPolicy)
	if err != nil {
		return "create_scheduled_agent_task 参数错误: " + err.Error(), true
	}

	files := strings.TrimSpace(args.Files)
	if files == "" {
		files = strings.TrimSpace(attachedFiles)
	}

	req := &dto.CreateScheduledAgentTaskReq{
		Name:              name,
		FullCodePath:      fullCodePath,
		Goal:              goal,
		ModeCode:          strings.TrimSpace(args.ModeCode),
		Files:             files,
		ContextPolicy:     contextPolicy,
		ToolPolicy:        toolPolicy,
		ApprovalPolicy:    approvalPolicy,
		BudgetPolicy:      budgetPolicy,
		ScheduleType:      scheduleType,
		RunAt:             runAt,
		CronExpr:          strings.TrimSpace(args.CronExpr),
		Timezone:          strings.TrimSpace(args.Timezone),
		RequestUser:       requestUser,
		RequestUserDept:   requestDept,
		NotifyUsers:       args.NotifyUsers,
		NotifyDepartments: args.NotifyDepartments,
		NotifyOn:          strings.TrimSpace(args.NotifyOn),
	}
	if args.LLMConfigID != nil {
		req.LLMConfigID = *args.LLMConfigID
	}
	if args.IntervalSeconds != nil {
		req.IntervalSeconds = *args.IntervalSeconds
	}
	if args.MaxRuns != nil {
		req.MaxRuns = *args.MaxRuns
	}

	ctx = withScheduledAgentToolSource(ctx)
	task, err := svc.Create(ctx, req, requestUser)
	if err != nil {
		return "create_scheduled_agent_task 调用失败: " + err.Error(), true
	}
	out := map[string]interface{}{
		"id":               task.ID,
		"timer_task_id":    task.TimerTaskID,
		"name":             task.Name,
		"full_code_path":   task.FullCodePath,
		"goal":             task.Goal,
		"mode_code":        task.ModeCode,
		"schedule_type":    task.ScheduleType,
		"status":           task.Status,
		"next_run_at":      task.NextRunAt,
		"run_at":           task.RunAt,
		"cron_expr":        task.CronExpr,
		"interval_seconds": task.IntervalSeconds,
		"max_runs":         task.MaxRuns,
		"source_type":      task.SourceType,
		"source_ref":       task.SourceRef,
	}
	return formatJSONResult(out)
}

func runListScheduledAgentTasksTool(ctx context.Context, registry *ToolRegistry, args listScheduledAgentTasksArgs, currentFullCodePath string) (string, bool) {
	svc, errMsg := scheduledAgentTaskToolService(registry, "list_scheduled_agent_tasks")
	if errMsg != "" {
		return errMsg, true
	}
	requestUser, _, errMsg := scheduledAgentTaskToolRequestUser(ctx, "list_scheduled_agent_tasks")
	if errMsg != "" {
		return errMsg, true
	}
	page, pageSize := scheduledAgentTaskToolPage(args.Page, args.PageSize)
	fullCodePath := resolveFullCodePathArg(args.FullCodePath, currentFullCodePath)
	list, total, err := svc.List(ctx, requestUser, strings.TrimSpace(args.Status), fullCodePath, page, pageSize)
	if err != nil {
		return "list_scheduled_agent_tasks 调用失败: " + err.Error(), true
	}
	items := make([]dto.ScheduledAgentTaskItem, 0, len(list))
	for _, task := range list {
		items = append(items, scheduledAgentTaskToolTaskItem(task))
	}
	out := map[string]interface{}{
		"total": total,
		"list":  items,
	}
	return formatJSONResult(out)
}

func runListScheduledAgentTaskExecutionsTool(ctx context.Context, registry *ToolRegistry, args listScheduledAgentTaskExecutionsArgs) (string, bool) {
	svc, errMsg := scheduledAgentTaskToolService(registry, "list_scheduled_agent_task_executions")
	if errMsg != "" {
		return errMsg, true
	}
	requestUser, _, errMsg := scheduledAgentTaskToolRequestUser(ctx, "list_scheduled_agent_task_executions")
	if errMsg != "" {
		return errMsg, true
	}
	if args.TaskID <= 0 {
		return "list_scheduled_agent_task_executions 需传 task_id（正整数）。", true
	}
	if _, errMsg := scheduledAgentTaskToolRequireTaskAccess(ctx, svc, args.TaskID, requestUser, "list_scheduled_agent_task_executions"); errMsg != "" {
		return errMsg, true
	}
	page, pageSize := scheduledAgentTaskToolPage(args.Page, args.PageSize)
	list, total, err := svc.ListExecutions(ctx, args.TaskID, requestUser, strings.TrimSpace(args.Status), page, pageSize)
	if err != nil {
		return "list_scheduled_agent_task_executions 调用失败: " + err.Error(), true
	}
	items := make([]dto.ScheduledAgentExecutionItem, 0, len(list))
	for _, exec := range list {
		items = append(items, scheduledAgentTaskToolExecutionItem(exec))
	}
	out := map[string]interface{}{
		"task_id": args.TaskID,
		"total":   total,
		"list":    items,
	}
	return formatJSONResult(out)
}

func runScheduledAgentTaskNowTool(ctx context.Context, registry *ToolRegistry, args runScheduledAgentTaskNowArgs) (string, bool) {
	svc, errMsg := scheduledAgentTaskToolService(registry, "run_scheduled_agent_task_now")
	if errMsg != "" {
		return errMsg, true
	}
	requestUser, _, errMsg := scheduledAgentTaskToolRequestUser(ctx, "run_scheduled_agent_task_now")
	if errMsg != "" {
		return errMsg, true
	}
	if args.TaskID <= 0 {
		return "run_scheduled_agent_task_now 需传 task_id（正整数）。", true
	}
	if _, errMsg := scheduledAgentTaskToolRequireTaskAccess(ctx, svc, args.TaskID, requestUser, "run_scheduled_agent_task_now"); errMsg != "" {
		return errMsg, true
	}
	exec, err := svc.RunNow(ctx, args.TaskID, requestUser)
	if err != nil {
		return "run_scheduled_agent_task_now 调用失败: " + err.Error(), true
	}
	out := map[string]interface{}{
		"timer_execution": exec,
		"message":         "已提交到 timer-scheduler 立即触发，可稍后用 list_scheduled_agent_task_executions 查看业务 session_id 和结果。",
	}
	return formatJSONResult(out)
}

func scheduledAgentTaskToolService(registry *ToolRegistry, toolName string) (*ScheduledAgentTaskService, string) {
	if registry == nil || registry.scheduledAgentTasks == nil {
		return nil, toolName + " 当前不可用：ScheduledAgentTaskService 未初始化。"
	}
	return registry.scheduledAgentTasks, ""
}

func scheduledAgentTaskToolRequestUser(ctx context.Context, toolName string) (string, string, string) {
	requestUser := strings.TrimSpace(contextx.GetRequestUser(ctx))
	if requestUser == "" {
		return "", "", toolName + " 需登录后调用，当前上下文缺少请求用户。"
	}
	return requestUser, strings.TrimSpace(contextx.GetRequestDepartmentFullPath(ctx)), ""
}

func scheduledAgentTaskToolRequireTaskAccess(ctx context.Context, svc *ScheduledAgentTaskService, taskID int64, requestUser string, toolName string) (*model.ScheduledAgentTask, string) {
	task, err := svc.Get(ctx, taskID, requestUser)
	if err != nil {
		return nil, toolName + " 获取任务失败: " + err.Error()
	}
	if task.CreatedBy == requestUser || task.RequestUser == requestUser {
		return task, ""
	}
	return nil, toolName + " 无权限操作该定时智能体任务。"
}

func scheduledAgentTaskToolJSONObject(fieldName string, value interface{}) (json.RawMessage, error) {
	if value == nil {
		return nil, nil
	}
	var raw []byte
	switch v := value.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return nil, nil
		}
		raw = []byte(v)
	default:
		var err error
		raw, err = json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("%s 必须是合法 JSON 对象: %w", fieldName, err)
		}
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		if err == nil {
			err = fmt.Errorf("不是 JSON 对象")
		}
		return nil, fmt.Errorf("%s 必须是合法 JSON 对象: %w", fieldName, err)
	}
	return json.RawMessage(raw), nil
}

func scheduledAgentTaskToolPage(pageArg *int, pageSizeArg *int) (int, int) {
	page := 1
	if pageArg != nil && *pageArg > 0 {
		page = *pageArg
	}
	pageSize := 20
	if pageSizeArg != nil && *pageSizeArg > 0 {
		pageSize = *pageSizeArg
	}
	return page, pageSize
}

func withScheduledAgentToolSource(ctx context.Context) context.Context {
	if strings.TrimSpace(contextx.GetSourceType(ctx)) != "" {
		return ctx
	}
	return contextx.WithRequestInfo(ctx, contextx.RequestInfo{SourceType: scheduledAgentToolSourceType})
}

func scheduledAgentTaskToolTaskItem(t *model.ScheduledAgentTask) dto.ScheduledAgentTaskItem {
	if t == nil {
		return dto.ScheduledAgentTaskItem{}
	}
	return dto.ScheduledAgentTaskItem{
		ID:                t.ID,
		Name:              t.Name,
		FullCodePath:      t.FullCodePath,
		Goal:              t.Goal,
		ModeCode:          t.ModeCode,
		Files:             t.Files,
		LLMConfigID:       t.LLMConfigID,
		ContextPolicy:     t.ContextPolicy,
		ToolPolicy:        t.ToolPolicy,
		ApprovalPolicy:    t.ApprovalPolicy,
		BudgetPolicy:      t.BudgetPolicy,
		ScheduleType:      t.ScheduleType,
		RunAt:             t.RunAt,
		NextRunAt:         t.NextRunAt,
		CronExpr:          t.CronExpr,
		IntervalSeconds:   t.IntervalSeconds,
		MaxRuns:           t.MaxRuns,
		Timezone:          t.Timezone,
		Status:            t.Status,
		RunCount:          t.RunCount,
		LastSessionID:     t.LastSessionID,
		LastExecutionID:   t.LastExecutionID,
		LastErrorMessage:  t.LastErrorMessage,
		RequestUser:       t.RequestUser,
		RequestUserDept:   t.RequestUserDept,
		NotifyUsers:       SplitScheduledAgentRecipientsForAPI(t.NotifyUsers),
		NotifyDepartments: SplitScheduledAgentRecipientsForAPI(t.NotifyDepartments),
		NotifyOn:          t.NotifyOn,
		SourceType:        t.SourceType,
		SourceRef:         t.SourceRef,
		CreatedBy:         t.CreatedBy,
		CreatedAt:         time.Time(t.CreatedAt),
		UpdatedAt:         time.Time(t.UpdatedAt),
	}
}

func scheduledAgentTaskToolExecutionItem(e *model.ScheduledAgentExecution) dto.ScheduledAgentExecutionItem {
	if e == nil {
		return dto.ScheduledAgentExecutionItem{}
	}
	return dto.ScheduledAgentExecutionItem{
		ID:             e.ID,
		TaskID:         e.TaskID,
		SessionID:      e.SessionID,
		ScheduledAt:    e.ScheduledAt,
		StartedAt:      e.StartedAt,
		FinishedAt:     e.FinishedAt,
		Status:         e.Status,
		WorkerID:       e.WorkerID,
		DurationMillis: e.DurationMillis,
		InputGoal:      e.InputGoal,
		OutputSummary:  e.OutputSummary,
		ToolCallCount:  e.ToolCallCount,
		TokenUsage:     e.TokenUsage,
		ErrorMessage:   e.ErrorMessage,
		TraceID:        e.TraceID,
		SourceType:     e.SourceType,
		SourceRef:      e.SourceRef,
		CreatedAt:      time.Time(e.CreatedAt),
		UpdatedAt:      time.Time(e.UpdatedAt),
	}
}
