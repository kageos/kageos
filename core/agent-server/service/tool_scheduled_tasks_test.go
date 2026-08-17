package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/scheduledsdk"
)

func TestBuildScheduledTaskScheduleParsesLocalAtime(t *testing.T) {
	got, err := buildScheduledTaskSchedule(scheduledTaskScheduleArgs{
		ScheduleType: "atime",
		RunAt:        "2026-06-10 09:30:00",
		Timezone:     "Asia/Shanghai",
		MaxRuns:      1,
	})
	if err != nil {
		t.Fatalf("buildScheduledTaskSchedule returned error: %v", err)
	}
	if got.Type != scheduledsdk.ScheduleAt || got.MaxRuns != 1 {
		t.Fatalf("unexpected schedule: %#v", got)
	}
	if got.RunAt.Format(time.RFC3339) != "2026-06-10T09:30:00+08:00" {
		t.Fatalf("run_at = %s, want 2026-06-10T09:30:00+08:00", got.RunAt.Format(time.RFC3339))
	}
}

func TestBuildScheduledTaskScheduleInfersCron(t *testing.T) {
	got, err := buildScheduledTaskSchedule(scheduledTaskScheduleArgs{
		CronExpr: "* * * * *",
		Timezone: "Asia/Shanghai",
	})
	if err != nil {
		t.Fatalf("buildScheduledTaskSchedule returned error: %v", err)
	}
	if got.Type != scheduledsdk.ScheduleCron || got.CronExpr != "* * * * *" {
		t.Fatalf("unexpected schedule: %#v", got)
	}
}

func TestBuildScheduledTaskScheduleCronUsesRuntimeTimezoneByDefault(t *testing.T) {
	got, err := buildScheduledTaskSchedule(scheduledTaskScheduleArgs{
		CronExpr: "* * * * *",
	})
	if err != nil {
		t.Fatalf("buildScheduledTaskSchedule returned error: %v", err)
	}
	if got.Type != scheduledsdk.ScheduleCron || got.Timezone != "" {
		t.Fatalf("cron without timezone should use timer runtime local timezone, got %#v", got)
	}
}

func TestBuildScheduledTaskScheduleValidatesEvery(t *testing.T) {
	_, err := buildScheduledTaskSchedule(scheduledTaskScheduleArgs{
		ScheduleType:    "every",
		IntervalSeconds: 0,
	})
	if err == nil || !strings.Contains(err.Error(), "interval_seconds") {
		t.Fatalf("expected interval validation error, got %v", err)
	}
}

func TestNormalizeScheduledFunctionTaskArgsAcceptsCompatibilityFields(t *testing.T) {
	got := normalizeCreateScheduledFunctionTaskArgs(createScheduledFunctionTaskArgs{
		FunctionPath:  "/system/test22/ticket_management/ticket_submit.form",
		Cron:          "* * * * *",
		TaskName:      "每分钟自动提交工单",
		MaxExecutions: "100",
		InvokeParams:  `{"body":"{\"title\":\"自动测试工单\",\"priority\":\"中\"}"}`,
	})
	if got.FullCodePath != "/system/test22/ticket_management/ticket_submit.form" {
		t.Fatalf("FullCodePath=%q", got.FullCodePath)
	}
	if got.CronExpr != "* * * * *" || got.ScheduleType != "cron" {
		t.Fatalf("schedule fields not normalized: %#v", got)
	}
	if got.Title != "每分钟自动提交工单" || got.MaxRuns != 100 {
		t.Fatalf("title/max runs not normalized: %#v", got)
	}
	payload, err := parseScheduledPayload(got.Body)
	if err != nil {
		t.Fatalf("normalized body should be valid JSON: %v", err)
	}
	body, ok := payload.(map[string]interface{})
	if !ok || body["title"] != "自动测试工单" || body["priority"] != "中" {
		t.Fatalf("unexpected normalized body: %#v", payload)
	}
}

func TestCreateScheduledFunctionTaskRejectsMissingFormBody(t *testing.T) {
	res := runCreateScheduledFunctionTask(context.Background(), createScheduledFunctionTaskArgs{
		FullCodePath: "/system/test22/ticket_management/ticket_submit.form",
		CronExpr:     "* * * * *",
	}, "")
	if !res.IsError || !strings.Contains(res.Content, "必须传 body") {
		t.Fatalf("expected missing body error, got %#v", res)
	}
}

func TestCreateScheduledFunctionTaskSchemaAllowsLegacyAliases(t *testing.T) {
	schema := (&CreateScheduledFunctionTaskTool{}).Definition().InputSchema
	err := validateToolArguments(schema, map[string]interface{}{
		"function_path":  "/system/test22/ticket_management/ticket_submit.form",
		"cron":           "* * * * *",
		"task_name":      "每分钟自动提交工单",
		"max_executions": "100",
		"invoke_params":  `{"body":"{\"title\":\"自动测试工单\"}"}`,
	})
	if err != nil {
		t.Fatalf("legacy aliases should not be rejected by schema validation: %v", err)
	}
}

func TestCreateScheduledAgentTaskSchemaAcceptsMessage(t *testing.T) {
	schema := (&CreateScheduledAgentTaskTool{}).Definition().InputSchema
	err := validateToolArguments(schema, map[string]interface{}{
		"full_code_path": "/system/test22/hot_news",
		"title":          "热点情报定时推送",
		"message":        "搜索热点，阅读文章，分析总结后发送企业微信群。",
		"schedule_type":  "cron",
		"cron_expr":      "*/15 * * * *",
	})
	if err != nil {
		t.Fatalf("message should be accepted by schema validation: %v", err)
	}
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("schema properties missing: %#v", schema)
	}
	messageSchema, ok := properties["message"].(map[string]interface{})
	if !ok {
		t.Fatalf("message schema missing: %#v", properties)
	}
	desc, _ := messageSchema["description"].(string)
	for _, want := range []string{"注入任务创建人/请求用户", "可省略 to_users", "首次基准记录"} {
		if !strings.Contains(desc, want) {
			t.Fatalf("message schema should contain %q, got %q", want, desc)
		}
	}
}

func TestCreateScheduledAgentTaskRejectsUnknownArgs(t *testing.T) {
	tool := &CreateScheduledAgentTaskTool{}
	for _, args := range []map[string]interface{}{
		{"body": `{"title":"热点信息自动追踪与推送","message":"搜索热点并推送"}`},
		{"payload": map[string]interface{}{"message": "搜索热点并推送"}},
		{"content": "搜索热点并推送"},
		{"every": 300},
		{"cron": "*/5 * * * *"},
		{"name": "热点信息自动追踪与推送"},
		{"task_name": "热点信息自动追踪与推送"},
		{"max_executions": 10},
	} {
		res := tool.Execute(context.Background(), ToolCall{Args: args})
		if !res.IsError || !strings.Contains(res.Content, "不支持参数") {
			t.Fatalf("unknown arg should be rejected: args=%#v res=%#v", args, res)
		}
	}
}

func TestCreateScheduledAgentTaskAllowsDirectoryAlias(t *testing.T) {
	tool := &CreateScheduledAgentTaskTool{}
	res := tool.Execute(context.Background(), ToolCall{Args: map[string]interface{}{
		"directory": "/system/test22/hot_news",
	}})
	if !res.IsError {
		t.Fatalf("expected missing message error, got %#v", res)
	}
	if strings.Contains(res.Content, "不支持参数") || strings.Contains(res.Content, "需传 full_code_path") {
		t.Fatalf("directory should be accepted as full_code_path alias, got %#v", res)
	}

	got := normalizeCreateScheduledAgentTaskArgs(createScheduledAgentTaskArgs{
		Directory:       "/system/test22/hot_news",
		IntervalSeconds: 300,
	})
	if got.FullCodePath != "/system/test22/hot_news" || got.ScheduleType != "every" {
		t.Fatalf("directory alias not normalized: %#v", got)
	}
}

func TestCreateScheduledAgentTaskAcceptsOverlapConfig(t *testing.T) {
	tool := &CreateScheduledAgentTaskTool{}
	res := tool.Execute(context.Background(), ToolCall{Args: map[string]interface{}{
		"full_code_path":  "/system/test22/hot_news",
		"overlap_policy":  "allow",
		"max_parallelism": 2,
	}})
	if !res.IsError {
		t.Fatalf("expected missing message error, got %#v", res)
	}
	if strings.Contains(res.Content, "不支持参数") {
		t.Fatalf("overlap config should be accepted, got %#v", res)
	}
}

func TestNormalizeScheduledAgentTaskArgsInfersEverySchedule(t *testing.T) {
	got := normalizeCreateScheduledAgentTaskArgs(createScheduledAgentTaskArgs{
		Title:           "热点信息自动追踪与推送",
		Message:         "搜索热点并推送",
		IntervalSeconds: 300,
	})
	if got.Title != "热点信息自动追踪与推送" || got.Message != "搜索热点并推送" {
		t.Fatalf("title/message not normalized: %#v", got)
	}
	if got.IntervalSeconds != 300 || got.ScheduleType != "every" {
		t.Fatalf("interval schedule not normalized: %#v", got)
	}
}

func TestScheduledTaskSchemasAcceptNumericStringsAfterNormalization(t *testing.T) {
	listSchema := (&ListScheduledTasksTool{}).Definition().InputSchema
	listArgs := normalizeToolArgumentsForSchema(listSchema, map[string]interface{}{
		"page":      "1",
		"page_size": "20",
	})
	if err := validateToolArguments(listSchema, listArgs); err != nil {
		t.Fatalf("normalized list args should validate: %v", err)
	}

	manageSchema := (&ManageScheduledTaskTool{}).Definition().InputSchema
	manageArgs := normalizeToolArgumentsForSchema(manageSchema, map[string]interface{}{
		"task_id": "3",
		"action":  "delete",
	})
	if err := validateToolArguments(manageSchema, manageArgs); err != nil {
		t.Fatalf("normalized manage args should validate: %v", err)
	}
	if manageArgs["task_id"] != int64(3) {
		t.Fatalf("task_id=%#v, want int64(3)", manageArgs["task_id"])
	}
}

func TestListScheduledTasksRequestUsesDirectoryPrefixWithoutCreatedBy(t *testing.T) {
	req := listScheduledTasksRequest("all", "/system/democase/gold_watch", "pending", 2, 50)
	if req.CreatedBy != "" {
		t.Fatalf("list_scheduled_tasks should not filter by created_by, got %q", req.CreatedBy)
	}
	if req.ResourceKey != "" {
		t.Fatalf("directory listing should not use exact resource_key, got %q", req.ResourceKey)
	}
	if req.ResourceKeyPrefix != "/system/democase/gold_watch" {
		t.Fatalf("resource_key_prefix=%q", req.ResourceKeyPrefix)
	}
	if req.ExecutorKey != "" || req.ResourceScope != "" {
		t.Fatalf("kind=all should not narrow executor/scope, got %#v", req)
	}
	if req.Status != "pending" {
		t.Fatalf("status=%q", req.Status)
	}
	if req.Page != 2 || req.PageSize != 50 {
		t.Fatalf("page hints should be preserved before all-page collection, got %#v", req)
	}
}

func TestWorkspaceScheduledTaskSummarySkipsExecutorPayloadContent(t *testing.T) {
	task := &scheduledsdk.Task{
		ID:              25,
		Title:           "黄金盯盘日报",
		Description:     "每天生成观察日报。",
		ExecutorKey:     "agent.session",
		ExecutorPayload: []byte(`{"message":"这是一大段无人值守运行手册，不应该进入模型环境摘要","display_content":"也不要注入"}`),
		Metadata:        map[string]string{"kind": "scheduled_agent_session"},
		Status:          scheduledsdk.TaskStatusPending,
		Schedule:        scheduledsdk.Cron("0 8 * * *"),
		ResourceKey:     "/system/democase/gold_watch",
		RunCount:        1,
		CreatedBy:       "system",
	}
	got := formatWorkspaceScheduledTaskSummary(task)
	for _, want := range []string{"id=25", "类型=数字员工", "标题=黄金盯盘日报", "描述=每天生成观察日报"} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary should contain %q, got %s", want, got)
		}
	}
	for _, forbidden := range []string{"无人值守运行手册", "display_content", "executor_payload"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("summary should not include payload content %q, got %s", forbidden, got)
		}
	}
}

func TestScheduledTaskToolDescriptionsDistinguishFunctionAndAgentTasks(t *testing.T) {
	functionDesc := (&CreateScheduledFunctionTaskTool{}).Definition().Description
	for _, want := range []string{
		"直接调用一个已确认的 Form/Table/Chart 函数",
		"固定函数路径 + 固定 body",
		"不要用于需要 Agent 到时候判断",
		"运行时用户不在线",
	} {
		if !strings.Contains(functionDesc, want) {
			t.Fatalf("function task description should contain %q, got %q", want, functionDesc)
		}
	}

	agentDesc := (&CreateScheduledAgentTaskTool{}).Definition().Description
	for _, want := range []string{
		"数字员工（Agent 任务）",
		"用户说“给这个目录创建一个数字员工”时使用本工具",
		"不要创建用户、角色、应用目录或普通函数任务",
		"启动一个 Agent 工作台会话",
		"message 当作执行说明交给工作台 Agent",
		"核心参数是 title + message",
		"interval_seconds=300",
		"不要把这些参数包进 body",
		"需要 Agent 判断、查询、总结、维护长期数据或组合多个动作",
		"运行时用户无法回答问题",
		"全部写进 message",
		"预期使用工具清单",
		"change_role、read_dir/search、web_search",
		"Agent 任务可以编排当前目录、同一能力包其他目录、用户明确授权的其他工作空间",
		"资源引用遵循相对路径优先",
		"当前目录写成 <./xxx.form>",
		"同一能力包兄弟目录写成 <../other/xxx.table>",
		"只有用户明确要求跨工作空间绑定时才使用 </完整路径>",
		"不可移植、复制后需重新绑定",
		"质量规则要结合业务",
		"不要用于已明确的单个 Form/Table/Chart 函数调用",
		"不能静默替换用户明确要求的数字员工",
	} {
		if !strings.Contains(agentDesc, want) {
			t.Fatalf("agent task description should contain %q, got %q", want, agentDesc)
		}
	}

	manageDesc := (&ManageScheduledTaskTool{}).Definition().Description
	for _, want := range []string{
		"cancel 取消但保留记录",
		"delete 删除并从列表移除",
		"delete 只删除任务配置",
		"已有 inflight 执行也会继续提交",
	} {
		if !strings.Contains(manageDesc, want) {
			t.Fatalf("manage task description should contain %q, got %q", want, manageDesc)
		}
	}
}

func TestScheduledFunctionRequiredAction(t *testing.T) {
	cases := []struct {
		path   string
		action string
		want   access.Action
	}{
		{path: "/system/x/vote/vote_chart.chart", action: "execute", want: access.ActionRead},
		{path: "/system/x/vote/vote_form.form", action: "execute", want: access.ActionWrite},
		{path: "/system/x/vote/vote_table.table", action: "table_update", want: access.ActionUpdate},
		{path: "/system/x/vote/vote_table.table", action: "table_delete", want: access.ActionDelete},
	}
	for _, tc := range cases {
		if got := scheduledFunctionRequiredAction(tc.path, tc.action); got != tc.want {
			t.Fatalf("scheduledFunctionRequiredAction(%q, %q)=%s want %s", tc.path, tc.action, got, tc.want)
		}
	}
}
