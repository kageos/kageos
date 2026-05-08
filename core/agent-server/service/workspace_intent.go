package service

type intentNextIntent struct {
	Intent string `json:"intent" schema_desc:"后续身份 ID" schema_required:"true"`
	When   string `json:"when" schema_desc:"触发条件" schema_required:"true"`
}

type intentSpec struct {
	ID          string
	Docs        []string
	Optional    []string
	NextTools   []string
	NextIntents []intentNextIntent
	Action      string
}

func workspaceIntentSpec(intent string) intentSpec {
	specs := map[string]intentSpec{
		"app.plan": {
			ID: "app.plan",
			Docs: []string{
				"/system/prompt/intents/app-plan",
			},
			Optional:  []string{"/system/prompt/case_catalog"},
			NextTools: []string{"write_prd", "read_doc", "read_dir"},
			Action:    "使用 app.plan 文档包，先做结构化 PRD 设计并调用 write_prd 输出预览，等待用户确认；不创建目录、不写代码、不 build。",
			NextIntents: []intentNextIntent{
				{Intent: "app.create", When: "用户确认 PRD 后进入开发执行"},
			},
		},
		"app.create": {
			ID: "app.create",
			Docs: []string{
				"/system/prompt/intents/app-create",
				"/system/prompt/sdk/agent-app-sdk-readme",
			},
			Optional:  []string{"/system/prompt/case_catalog"},
			NextTools: []string{"read_doc", "read_dir", "create_directory", "write_go_file", "build_workspace"},
			Action:    "使用 app.create 文档包，只按已确认 PRD JSON 开发执行；不重新输出 PRD，不二次确认，完整落盘后统一 build。",
			NextIntents: []intentNextIntent{
				{Intent: "app.operate_test", When: "build 成功后验证核心函数"},
				{Intent: "app.build_fix", When: "build 失败或 schema compile failed"},
			},
		},
		"app.modify": {
			ID:        "app.modify",
			Docs:      []string{"/system/prompt/intents/app-modify", "/system/prompt/intents/modify/index"},
			NextTools: []string{"read_doc", "read_dir", "read_go_file", "search_replace_file", "write_go_file", "build_workspace"},
			Action:    "先判断修改二级类型并读取专项文档，再读取相关文件，修改后统一 build 和验证。",
			NextIntents: []intentNextIntent{
				{Intent: "app.operate_test", When: "修改 build 成功后验证功能"},
				{Intent: "app.build_fix", When: "构建或 schema 校验失败"},
			},
		},
		"app.operate_test": {
			ID:        "app.operate_test",
			Docs:      []string{"/system/prompt/intents/app-operate-test"},
			NextTools: []string{"read_dir", "run_table_search", "run_form_submit", "run_chart_query", "run_on_select_fuzzy"},
			Action:    "读取操作测试文档，确认 schema 后设计测试用例并调用 run_* 工具。",
			NextIntents: []intentNextIntent{
				{Intent: "app.modify", When: "测试发现业务 bug"},
				{Intent: "schedule.task", When: "功能稳定后需要定时执行"},
			},
		},
		"app.build_fix": {
			ID:        "app.build_fix",
			Docs:      []string{"/system/prompt/intents/app-build-fix"},
			NextTools: []string{"read_go_file", "search_replace_file", "build_workspace"},
			Action:    "读取构建排错文档，按错误类型批量修复，再重新 build。",
			NextIntents: []intentNextIntent{
				{Intent: "app.operate_test", When: "build 修复成功后验证功能"},
				{Intent: "app.modify", When: "错误指向业务逻辑或字段设计缺陷"},
			},
		},
		"temp.task": {
			ID:        "temp.task",
			Docs:      []string{"/system/prompt/intents/temp-task"},
			NextTools: []string{"search_tools", "run_form_submit", "run_official_python"},
			Action:    "优先搜索 system tools，确认 schema 后执行一次性任务，不创建长期业务应用。",
		},
		"schedule.task": {
			ID:        "schedule.task",
			Docs:      []string{"/system/prompt/intents/schedule-task"},
			NextTools: []string{"create_scheduled_task", "create_scheduled_agent_task", "list_scheduled_tasks"},
			Action:    "判断是否已有单次执行函数；没有则先切到创建/修改意图实现，再配置平台定时任务。",
		},
		"platform.openapi": {
			ID:        "platform.openapi",
			Docs:      []string{"/system/prompt/intents/platform-openapi"},
			NextTools: []string{"search_tools", "run_form_submit"},
			Action:    "读取平台 OpenAPI 文档，确认接口 schema、权限和副作用后执行。",
		},
		"app.explain_review": {
			ID:        "app.explain_review",
			Docs:      []string{"/system/prompt/intents/app-explain-review"},
			NextTools: []string{"read_dir", "read_go_file"},
			Action:    "读取解释/审查文档，以只读方式分析项目、风险和改进建议。",
		},
	}
	if spec, ok := specs[intent]; ok {
		return spec
	}
	return specs["app.explain_review"]
}
