package roles

import (
	"strings"
)

const (
	Router              = "router"
	ProductManager      = "product_manager"
	AppDeveloper        = "app_developer"
	BuildEngineer       = "build_engineer"
	QAEngineer          = "qa_engineer"
	AppOperator         = "app_operator"
	AutomationOperator  = "automation_operator"
	MaintenanceEngineer = "maintenance_engineer"
	PlatformEngineer    = "platform_engineer"
	DataOperator        = "data_operator"
	Reviewer            = "reviewer"
)

const RoutingMarker = "{{WORKSPACE_ROLE_ROUTING}}"

type Spec struct {
	ID               string
	DisplayName      string
	Docs             []string
	Optional         []string
	AllowedTools     []string
	ForbiddenTools   []string
	Runtime          RuntimeContract
	NextRoles        []NextRole
	Action           string
	RouteDescription string
}

type RuntimeContract struct {
	EntryConditions     []string        `json:"entry_conditions,omitempty" schema_desc:"进入该角色的条件"`
	ForbiddenConditions []string        `json:"forbidden_conditions,omitempty" schema_desc:"不应进入该角色的条件"`
	SOP                 []string        `json:"sop,omitempty" schema_desc:"角色标准作业流程"`
	DoneWhen            []string        `json:"done_when,omitempty" schema_desc:"该角色何时算完成"`
	HandoffRequired     []string        `json:"handoff_required,omitempty" schema_desc:"切换到该角色时必须携带的标准交接字段"`
	Hooks               []LifecycleHook `json:"hooks,omitempty" schema_desc:"角色生命周期 Hook 声明"`
}

type LifecycleHook struct {
	ID                   string   `json:"id" schema_desc:"Hook 标识" schema_required:"true"`
	Stage                string   `json:"stage" schema_desc:"触发阶段，例如 before_enter/after_tool/before_handoff" schema_required:"true"`
	Purpose              string   `json:"purpose" schema_desc:"Hook 目的" schema_required:"true"`
	Reads                []string `json:"reads,omitempty" schema_desc:"Hook 读取的输入或资料"`
	Produces             []string `json:"produces,omitempty" schema_desc:"Hook 产出的上下文或产物"`
	ImplementationStatus string   `json:"implementation_status,omitempty" schema_desc:"Hook 运行时实现状态：implemented/planned"`
}

type NextRole struct {
	RoleID string `json:"role_id" schema_desc:"后续角色 ID" schema_required:"true"`
	When   string `json:"when" schema_desc:"触发条件" schema_required:"true"`
}

func Specs() map[string]Spec {
	specs := map[string]Spec{
		Router: {
			ID:          Router,
			DisplayName: "执行路由手册",
			Docs:        []string{"/system/prompt/roles/router"},
			AllowedTools: []string{
				"change_role", "summarize_task_state", "read_doc", "read_dir", "read_file", "read_app_log", "search", "web_search",
			},
			ForbiddenTools: []string{
				"write_prd", "create_directory", "write_doc", "write_file", "edit_file", "delete_file", "build_workspace",
				"run_table_search", "run_table_create", "run_table_update", "run_table_delete",
				"run_form_submit", "run_chart_query", "run_on_select_fuzzy", "run_python",
				"create_scheduled_function_task", "create_scheduled_agent_task", "manage_scheduled_task", "send_notification",
			},
			Runtime: runtimeContract(
				[]string{"当前角色无法继续、可见工具不足、门禁提示需要交接但目标角色不确定", "用户最新需求横跨多个阶段或意图模糊", "连续一次角色判断失败后需要重新读路由手册"},
				[]string{"已经明确需要某个专业角色直接执行且该角色可见工具足够"},
				[]string{"读取 /system/prompt/roles/router 完整路由手册", "按 3 步急救流程先判任务形态，再只读补一个最小证据", "按立即决策流程和门禁错误处理公式选择下一专业角色", "只用只读工具收集必要证据，不写入、不构建、不执行业务副作用", "同一轮内调用 change_role 交接到能实际干活的角色，并携带 execute_directory/task_context/key_information/references"},
				[]string{"已通过 change_role 明确进入下一专业角色", "或已说明无法判断时所需的最小补充信息"},
				nil,
			),
			Action:           "执行路由手册是兜底换挡角色：读取完整路由手册，按 3 步急救流程和立即决策流程收敛当前场景，并在同一轮内交接到真正能执行的专业角色；不执行写入、构建或业务运行副作用。",
			RouteDescription: "兜底换挡入口。当当前角色不知道下一步该切谁、可见工具不足、门禁提示需要交接、用户需求横跨开发/测试/执行/自动化/平台/数据处理，或模型连续一次误判角色时进入。执行路由手册必须读取 `/system/prompt/roles/router`，按 3 步急救流程、立即决策流程和门禁错误处理公式判断，必要时只读当前目录、函数 schema、源码或日志，随后同一轮内用 `change_role` 切到具体专业角色。它不是执行角色，不写 PRD、不写代码、不 build、不 run 业务函数、不创建定时任务。",
			NextRoles: []NextRole{
				{RoleID: AppOperator, When: "用户是在已有应用里完成真实业务操作，或只是要处理轻量一次性文件/数据任务"},
				{RoleID: AutomationOperator, When: "用户要配置未来、周期或无人值守自动执行"},
				{RoleID: ProductManager, When: "用户要新建长期系统或重新设计 PRD"},
				{RoleID: AppDeveloper, When: "用户已确认 PRD，进入开发实现"},
				{RoleID: MaintenanceEngineer, When: "需要修改已有应用能力、字段、搜索、回调或业务逻辑"},
				{RoleID: QAEngineer, When: "需要验证刚构建或刚修改的应用功能"},
				{RoleID: BuildEngineer, When: "问题指向 build/schema/widget/router/SDK API"},
				{RoleID: DataOperator, When: "用户要复杂、专项或多步骤的一次性文件、媒体或数据处理"},
				{RoleID: PlatformEngineer, When: "目标涉及平台 OpenAPI、权限、审计、组织或平台集成"},
				{RoleID: Reviewer, When: "只读解释、分析、review、产品介绍或能力边界说明"},
			},
		},
		ProductManager: {
			ID:           ProductManager,
			DisplayName:  "产品经理",
			Docs:         []string{"/system/prompt/roles/product-manager"},
			Optional:     []string{"/system/prompt/case_catalog"},
			AllowedTools: []string{"change_role", "read_doc", "read_dir", "write_prd"},
			ForbiddenTools: []string{
				"create_directory", "write_doc", "write_file", "edit_file", "delete_file", "build_workspace",
				"run_table_search", "run_table_create", "run_table_update", "run_table_delete",
				"run_form_submit", "run_chart_query", "run_on_select_fuzzy",
			},
			Runtime: runtimeContract(
				[]string{"用户要新建长期业务系统、应用目录、Form/Table/Chart 或管理后台", "尚未确认结构化 PRD"},
				[]string{"当前目录已有应用且运行函数能完成用户目标", "用户只是要操作现有软件完成业务结果"},
				[]string{"先结合当前目录判断是新建系统还是使用现有应用", "优先使用 file_profile、用户文件内容和样例数据提炼业务对象、字段、搜索、提交入口、统计和规则", "调用 write_prd 输出 PRD v2", "等待用户确认，不创建目录、不写代码、不 build"},
				[]string{"write_prd 已生成可确认 PRD artifact", "用户确认前保持 pending_confirmation"},
				[]LifecycleHook{
					hook("product_manager.prd_ready", "after_tool", "write_prd 后固化 PRD artifact，供前端确认和后续 handoff 使用。", []string{"write_prd structured data"}, []string{"agent_app_prd artifact", "pending_confirmation interaction"}),
					hook("product_manager.to_app_developer", "before_handoff", "用户确认 PRD 后生成开发执行视图和交接摘要。", []string{"agent_app_prd JSON", "source session notes"}, []string{"PRD_EXECUTION_MARKDOWN", "artifact_digest", "handoff_context", "executed_hooks"}),
				},
			),
			Action:           "产品经理只负责新建长期业务系统的需求分析、PRD v2 结构化输出和确认；调用 write_prd 后等待用户确认，不创建目录、不写代码、不 build。",
			RouteDescription: "用户明确要新建长期业务系统、应用目录、新 Form/Table/Chart 或管理后台，但还没有确认 PRD 时进入。上传 Excel/CSV/JSON/Markdown/文本/代码并表达“做成系统/后台/应用”时，如果消息里已有 `file_profile` 或内容清楚，直接进入产品经理基于字段和样例生成 PRD，不要为了读取同一文件先绕到 `data_operator`。当前目录已有应用且运行函数能完成用户目标时不要进入产品经理；这类请求是用户在使用软件完成业务结果，应进入 `app_operator`。例如在 `/system/x_world/vote` 里“创建一个投票”是业务操作，不是写 PRD。产品经理只负责把新系统需求拆成可确认的 PRD artifact：`project/tables/forms/charts/rules`；调用 `write_prd` 后等待用户确认，不创建目录、不写代码、不 build。",
			NextRoles: []NextRole{
				{RoleID: AppDeveloper, When: "用户确认 PRD 后进入应用开发"},
			},
		},
		AppDeveloper: {
			ID:          AppDeveloper,
			DisplayName: "应用开发工程师",
			Docs:        []string{"/system/prompt/roles/app-developer", "/system/prompt/sdk/agent-app-sdk-readme"},
			Optional:    []string{"/system/prompt/case_catalog"},
			AllowedTools: []string{
				"change_role", "summarize_task_state", "read_doc", "read_dir", "read_file",
				"create_directory", "write_doc", "write_file", "edit_file", "read_app_log", "build_workspace",
			},
			ForbiddenTools: []string{"write_prd"},
			Runtime: runtimeContract(
				[]string{"用户已确认 PRD", "handoff 会话携带完整 agent_app_prd JSON", "用户明确要求新增或改变软件能力"},
				[]string{"用户在已有应用里使用软件完成业务结果", "需求尚未确认 PRD 且不是小范围维护"},
				[]string{"调用 change_role 并固定 execute_directory", "优先阅读 PRD_EXECUTION_MARKDOWN，细节以 PRD JSON 为准", "读取 SDK 主文档和匹配案例", "创建或修改目标目录代码", "build_workspace 前必须先读回相关源码做模型 CR，并在 build 参数提交 pre_build_review/review_passed", "build 成功后不等待用户确认，立即交接 qa_engineer 并自动测试"},
				[]string{"build_workspace 成功、已交接 qa_engineer 并完成核心函数测试", "或 build/schema 失败并带完整错误交接给 build_engineer"},
				[]LifecycleHook{
					hook("app_developer.before_enter_prd", "before_enter", "整理确认后的 PRD、执行目录和开发参考资料。", []string{"agent_app_prd JSON", "PRD_EXECUTION_MARKDOWN", "reference docs"}, []string{"developer_context_packet"}),
					hook("app_developer.after_build", "after_tool", "build_workspace 后根据成功或失败决定 QA 或构建修复交接。", []string{"build_workspace result"}, []string{"agent_app_build artifact", "build_diagnostics"}),
				},
			),
			Action:           "应用开发工程师只按已确认 PRD v2 开发执行；区分 tables.fields 模型字段和 tables.search_fields 查询字段，读取 SDK 和案例，创建目录、写 Go 文件；build 前必须模型 CR，确认无伪实现和范围外功能后再 build；build 成功后立即交接 QA 自动测试，不重新输出 PRD。",
			RouteDescription: "用户已确认 PRD，或确认按钮开启的新会话携带完整 PRD JSON 时进入。只按 PRD v2 直接开发：读取匹配案例，创建目录，生成 Go struct 和函数代码，注册路由；`tables.fields` 是模型字段，`tables.search_fields` 是查询请求字段；build_workspace 前必须先读回相关源码做模型 CR，并在 build 参数提交 pre_build_review/review_passed，确认无“开发中/未实现/占位”伪功能和 PRD 外入口；build 成功后必须立即进入 `qa_engineer` 自动测试，不等待用户确认。不要重新输出 PRD，不要再次询问确认。用户是在已有应用里使用软件完成业务结果，而不是要求新增或改变软件能力时，不要进入开发工程师，应进入 `app_operator`。",
			NextRoles: []NextRole{
				{RoleID: QAEngineer, When: "build 成功后验证核心函数"},
				{RoleID: BuildEngineer, When: "build 失败或 schema compile failed"},
			},
		},
		MaintenanceEngineer: {
			ID:          MaintenanceEngineer,
			DisplayName: "应用维护工程师",
			Docs:        []string{"/system/prompt/roles/maintenance-engineer"},
			AllowedTools: []string{
				"change_role", "summarize_task_state", "read_doc", "read_dir", "read_file",
				"create_directory", "write_doc", "write_file", "edit_file", "delete_file", "read_app_log", "build_workspace",
			},
			ForbiddenTools: []string{"write_prd"},
			Runtime: runtimeContract(
				[]string{"用户要修改已有应用、字段、搜索、选项、回调、图表或业务 bug", "用户要创建或更新当前目录文档、运行手册 runbook.docs、SOP 或业务说明", "测试或操作发现业务实现问题"},
				[]string{"用户要重新设计新系统需求", "只是业务数据操作且当前应用可直接完成"},
				[]string{"固定目标应用 execute_directory", "读取相关目录和源码；纯文档/runbook 任务读取当前目录、函数 schema、已有文档和定时任务摘要", "用户要求当前目录运行手册时用 write_doc 写入当前目录 code=runbook、name=运行手册", "纯文档/runbook 修改完成后不 build、不交接 QA", "代码或 schema 修改时小改局部替换，大改再写完整文件", "代码修改后 build_workspace 前必须先读回相关源码做模型 CR，并在 build 参数提交 pre_build_review/review_passed", "build 成功后不等待用户确认，立即交接 qa_engineer 并自动测试", "失败时按错误类型补读文档或交接 build_engineer"},
				[]string{"纯文档/runbook 修改已用 write_doc 创建或更新并返回路径", "或目标代码修改已落盘、build 成功并完成 QA 测试", "或构建问题已交接 build_engineer"},
				[]LifecycleHook{
					hook("maintenance.before_enter_scope", "before_enter", "收敛维护范围，避免扫描或修改无关应用。", []string{"execute_directory", "changed files", "bug report"}, []string{"maintenance_scope"}),
					hook("maintenance.after_build", "after_tool", "构建后决定进入 QA 或构建修复。", []string{"build_workspace result"}, []string{"verification_focus", "build_diagnostics"}),
				},
			),
			Action:           "应用维护工程师负责修改已有应用、字段、搜索、选项、回调、图表、业务 bug 和当前目录文档/runbook；纯文档修改用 write_doc 完成，不 build；代码修改需读取相关源码后修改，build 前模型 CR，确认无伪实现和范围外功能后再 build。",
			RouteDescription: "用户要改已有应用、字段、选项、组件、回调、搜索、消息、跳转、图表、业务逻辑，或要为当前目录创建/更新文档、运行手册 runbook.docs、SOP、业务说明时进入。文档/runbook 任务先读取当前目录、函数和已有文档，使用 `write_doc` 写入；当前目录运行手册固定用 code=runbook、name=运行手册，生成 `<当前目录>/runbook.docs`，纯文档修改不 build、不交接 QA。代码修改先识别修改类型和影响范围，读取当前目录与相关源码，只改用户目标和必要依赖。新增或修改搜索时区分业务字段和系统字段；小改优先局部替换，大改或新增能力再写完整文件；build_workspace 前必须先读回相关源码做模型 CR，并在 build 参数提交 pre_build_review/review_passed；构建成功后必须立即进入 `qa_engineer` 自动测试，不等待用户确认。",
			NextRoles: []NextRole{
				{RoleID: QAEngineer, When: "修改 build 成功后验证功能"},
				{RoleID: BuildEngineer, When: "构建或 schema 校验失败"},
			},
		},
		QAEngineer: {
			ID:          QAEngineer,
			DisplayName: "测试工程师",
			Docs:        []string{"/system/prompt/roles/qa-engineer"},
			AllowedTools: []string{
				"change_role", "summarize_task_state", "read_doc", "read_dir", "search",
				"run_table_search", "run_table_create", "run_table_update", "run_table_delete",
				"run_form_submit", "run_chart_query", "run_on_select_fuzzy", "send_notification",
			},
			ForbiddenTools: []string{"write_prd", "create_directory", "write_doc", "write_file", "edit_file", "delete_file", "build_workspace"},
			Runtime: runtimeContract(
				[]string{"需要测试刚构建的应用", "需要验证已有函数是否符合预期"},
				[]string{"需要修改代码", "用户只是要完成真实业务操作且不是测试验证"},
				[]string{"固定 execute_directory", "用 search 获取目标函数 schema", "按业务链路准备测试数据", "调用 run_* 工具验证", "把失败归类为参数、数据、schema、业务 bug 或环境问题"},
				[]string{"核心 Table/Form/Chart 已验证并给出结论", "失败已归类并交接维护或构建角色"},
				[]LifecycleHook{
					hook("qa.before_enter_schema", "before_enter", "收集目标应用函数和 schema 摘要，防止测试整个工作区。", []string{"execute_directory", "agent_app_build artifact"}, []string{"test_capability_snapshot", "verification_plan"}),
					hook("qa.after_run", "after_tool", "运行工具失败后归因并准备修复交接。", []string{"run_* result"}, []string{"failure_classification", "handoff_recommendation"}),
				},
			),
			Action:           "测试工程师确认 schema 后按实际功能顺序验证 Table/Form/Chart，覆盖时间范围和用户筛选，并调用 run_* 工具；不直接改代码。",
			RouteDescription: "用户要查数据、提交表单、查图表、测试刚生成的应用或验证已有函数时进入。先确认目标函数、schema、必填字段、枚举、文件字段、搜索字段和写入能力，再调用运行工具。重点验证 Table 查询、时间范围筛选、用户筛选、Form 写入目标表和 Chart 统计。测试失败时判断是业务数据问题、代码问题还是构建/schema 问题，并建议切换到 `maintenance_engineer` 或 `build_engineer`。",
			NextRoles: []NextRole{
				{RoleID: MaintenanceEngineer, When: "测试发现业务 bug"},
				{RoleID: BuildEngineer, When: "测试失败指向构建、schema 或路由问题"},
			},
		},
		AppOperator: {
			ID:          AppOperator,
			DisplayName: "应用执行",
			Docs:        []string{"/system/prompt/roles/app-operator"},
			AllowedTools: []string{
				"change_role", "summarize_task_state", "read_doc", "read_dir", "search",
				"run_table_search", "run_table_create", "run_table_update", "run_table_delete",
				"run_form_submit", "run_chart_query", "run_on_select_fuzzy", "run_python", "send_notification",
				"list_scheduled_tasks", "list_scheduled_task_executions",
			},
			ForbiddenTools: []string{"write_prd", "create_directory", "write_doc", "write_file", "edit_file", "delete_file", "build_workspace"},
			Runtime: runtimeContract(
				[]string{"当前目录已有应用且用户要查询、新增、更新、删除、提交表单或查看图表", "用户意图是使用软件完成业务结果", "用户只是要轻量一次性文件/数据处理，例如简单转换、压缩、清洗、加水印、解析附件或整理临时结果"},
				[]string{"用户要新增或改变软件能力", "用户要测试刚构建应用而不是真实业务操作", "用户要复杂、专项或多步骤文件处理，需要数据/文件处理工程师的完整 SOP"},
				[]string{"固定目标应用 execute_directory", "先获取函数 schema、必填项、枚举、文件字段和写入能力", "必要时先查询关联数据或调用 OnSelectFuzzy", "需要轻量计算、解析附件、清洗、转换、压缩、加水印或整理中间数据时可调用 run_python，但真实业务写入仍走 run_* 函数", "调用 run_* 完成业务操作，或用 run_python 完成轻量一次性文件/数据处理", "失败时区分参数/数据问题、应用 bug 或是否需要交接 data_operator"},
				[]string{"业务操作完成并返回结果", "轻量文件/数据处理完成并返回产物或简洁结果", "或失败原因已分类并交接维护角色"},
				[]LifecycleHook{
					hook("app_operator.before_enter_capabilities", "before_enter", "进入操作角色前生成当前应用能力快照。", []string{"execute_directory", "registered functions"}, []string{"available_capabilities", "operation_schema_summary", "app_capabilities", "executed_hooks"}),
					hook("app_operator.after_run", "after_tool", "业务运行后判断是否需要继续查询、补参数或交接维护。", []string{"run_* result"}, []string{"operation_result", "failure_classification"}),
				},
			),
			Action:           "应用执行负责在已有应用里执行业务操作：查询、新增、更新、删除记录、提交表单、查看图表；也负责轻量一次性文件/数据处理。真实业务写入必须走 run_* 工具，不改 PRD、不改代码。",
			RouteDescription: "当前目录已是目标应用，或目录下已有 Table/Form/Chart 能完成用户目标时，只要用户是在使用软件完成业务结果且目的不是测试验证，就进入应用执行。用户只是要处理一个轻量一次性文件/数据任务，例如简单转换、压缩、清洗、改尺寸、加水印、解析附件或整理临时结果时，也默认进入应用执行，用 `run_python` 完成，不必为了这类短任务切到 `data_operator`。它优先于 `product_manager` 和 `app_developer` 处理真实业务数据操作，不依赖某个固定动词；例如在投票应用目录中“创建一个四大古都投票，北京南京西安洛阳单选”就是新增投票主题和选项。先确认目标应用和函数 schema；写入类操作要复述关键字段并避免误写；工具失败时判断是参数/数据问题、应用 bug 或文件处理复杂度超出轻量范围，必要时交接给 `maintenance_engineer` 或 `data_operator`。",
			NextRoles: []NextRole{
				{RoleID: MaintenanceEngineer, When: "业务操作失败且判断为应用 bug 或字段实现问题"},
				{RoleID: AutomationOperator, When: "用户要把当前业务操作保存为未来或周期自动执行"},
				{RoleID: DataOperator, When: "一次性文件/数据处理变成复杂、专项或多步骤任务"},
			},
		},
		AutomationOperator: {
			ID:          AutomationOperator,
			DisplayName: "自动执行配置",
			Docs:        []string{"/system/prompt/roles/automation-operator"},
			AllowedTools: []string{
				"change_role", "summarize_task_state", "read_doc", "read_dir", "search",
				"create_scheduled_function_task", "create_scheduled_agent_task",
				"list_scheduled_tasks", "manage_scheduled_task", "list_scheduled_task_executions", "send_notification",
			},
			ForbiddenTools: []string{
				"write_prd", "create_directory", "write_doc", "write_file", "edit_file", "delete_file", "build_workspace",
				"run_table_search", "run_table_create", "run_table_update", "run_table_delete",
				"run_form_submit", "run_chart_query", "run_on_select_fuzzy",
			},
			Runtime: runtimeContract(
				[]string{"用户要把已有应用函数、已有业务操作或已有工作台目录配置成指定时间、周期、提醒、巡检或 Agent 任务", "用户要查询、暂停、恢复、取消或立即运行已有定时任务"},
				[]string{"用户只是要立即查询、提交、更新或删除真实业务数据", "用户想定时执行但目标能力尚不存在、函数未确认或需要新增/修改软件能力", "用户要测试刚构建应用"},
				[]string{"固定目标应用 execute_directory", "先区分任务类型：固定 Form/Table/Chart 调用用函数任务，需要 Agent 判断/巡检/分析/总结/维护长期数据或多步骤用 Agent 任务", "先用 search 确认目标函数或目录，不搜索整个工作区", "Agent 任务可以编排当前目录、本空间其他目录、其他空间函数、系统工具和连接器函数；message 必须按无人值守 SOP 写清场景、长期目标、可用资源/函数、预期使用工具清单、执行步骤、按业务场景裁剪的质量控制、失败处理、输出格式和通知规则；不要把示例规则机械套到所有任务", "把用户自然语言计划转换为 atime、cron 或 every，并复述关键时间、频率和最多次数", "用户只是问能不能或怎么做时只说明方案，不创建任务", "创建任务前确认执行参数来自用户输入或已验证 schema，不猜必填字段、枚举或记录 ID；周期性写入任务必须等用户明确确认", "调用定时任务工具创建或管理任务", "返回 task_id、下次执行时间、执行来源和取消方式"},
				[]string{"函数任务或 Agent 任务已创建并返回 task_id/next_run_at", "或任务已暂停、恢复、取消、立即运行、执行记录已查询", "失败原因已区分为时间表达式、权限、参数/schema 或调度服务问题"},
				[]LifecycleHook{
					hook("automation.before_enter_scope", "before_enter", "进入自动化角色前收敛目标应用、候选函数和计划类型。", []string{"execute_directory", "user schedule intent", "function schema"}, []string{"automation_scope", "schedule_plan"}),
				},
			),
			Action:           "自动执行配置负责把已有业务操作配置成未来或周期自动执行，并管理函数任务、Agent 任务和执行记录；不直接修改代码，不直接执行真实业务写入。",
			RouteDescription: "用户要“定时、每天、每周、周期、提醒、自动跑、定期巡检、到点提交、Agent 任务”且目标是已有应用函数、已有业务操作或已有工作台目录时进入。它负责把已有能力配置成 timer-scheduler 任务，并管理暂停、恢复、取消、立即运行和执行记录。先区分两类任务：目标是一个明确 Form/Table/Chart 和固定参数时，用函数任务；目标需要 Agent 到点后判断、巡检、分析、总结、维护长期数据、选择多个工具或临场决策时，用 Agent 任务。Agent 任务可以编排当前目录、本空间其他目录、其他空间函数、系统工具和连接器函数；message 必须写成无人值守 SOP：场景/长期目标、绑定目录、可用资源/函数、预期使用工具清单、执行步骤、按业务场景裁剪的质量控制、失败处理、输出格式、通知规则；不要把示例规则机械套到所有任务。它不同于 `app_operator`：应用执行负责现在执行一次真实业务操作；自动执行配置负责以后自动执行。它也不同于 `product_manager/app_developer/maintenance_engineer`：如果用户想定时执行的能力还不存在、函数 schema 还不确定，或需要新增/修改软件能力，先进入产品、开发或维护，不要直接进入自动化。用户只是问“能不能/怎么做”时只说明方案和风险；周期性写入任务必须等用户明确确认后再创建。创建任务前必须确认目标函数/目录、计划时间、执行参数和权限边界；不设计 PRD、不写代码、不 build、不直接调用 run_* 完成真实业务写入。",
			NextRoles: []NextRole{
				{RoleID: AppOperator, When: "用户要求先立即执行一次业务操作验证参数"},
				{RoleID: MaintenanceEngineer, When: "定时执行失败且判断为应用 bug 或字段实现问题"},
			},
		},
		BuildEngineer: {
			ID:          BuildEngineer,
			DisplayName: "构建修复工程师",
			Docs:        []string{"/system/prompt/roles/build-engineer"},
			AllowedTools: []string{
				"change_role", "summarize_task_state", "read_doc", "read_file",
				"edit_file", "write_file", "read_app_log", "build_workspace",
			},
			ForbiddenTools: []string{"write_prd", "write_doc", "run_table_search", "run_table_create", "run_table_update", "run_table_delete", "run_form_submit", "run_chart_query"},
			Runtime: runtimeContract(
				[]string{"build_workspace 失败", "启动、schema、widget、路由后缀或 SDK API 相关错误"},
				[]string{"业务功能本身需要重新设计", "只是测试数据或参数问题"},
				[]string{"固定目标应用目录", "读取完整错误和相关源码", "按 router/字段/文件归类同类问题", "不确定先读 build-validation、SDK 主文档或案例", "批量修复后、build_workspace 前必须先读回相关源码做模型 CR，并在 build 参数提交 pre_build_review/review_passed", "build 成功后不等待用户确认，立即交接 qa_engineer 并自动测试"},
				[]string{"build_workspace 成功、已交接 QA 并完成测试", "或确认问题属于业务设计缺陷并交接维护角色"},
				[]LifecycleHook{
					hook("build_engineer.before_enter_diagnostics", "before_enter", "解析构建错误并匹配修复策略和必读文档。", []string{"build error", "workspace path"}, []string{"build_diagnostics", "required_docs", "repair_policy", "executed_hooks"}),
					hook("build_engineer.after_build", "after_tool", "重新构建后决定 QA 或继续修复。", []string{"build_workspace result"}, []string{"agent_app_build artifact", "remaining_errors"}),
				},
			),
			Action:           "构建修复工程师负责构建、启动、schema、widget、搜索字段和 SDK API 排错；按错误类型批量修复，build 前模型 CR 后再重新 build。",
			RouteDescription: "构建失败、启动失败、schema 编译失败、widget 校验失败、路由后缀错误、SDK API 不确定时进入。读取完整错误，按错误类型归类，同类问题批量修复。重新 build_workspace 前必须先读回修复相关源码做模型 CR，并在 build 参数提交 pre_build_review/review_passed；遇到搜索字段错误时先判断是系统查询字段还是业务模型字段；不要猜不存在的 SDK API；不确定时回到完整 SDK 文档或源码确认。",
			NextRoles: []NextRole{
				{RoleID: QAEngineer, When: "build 修复成功后验证功能"},
				{RoleID: MaintenanceEngineer, When: "错误指向业务逻辑或字段设计缺陷"},
			},
		},
		DataOperator: {
			ID:           DataOperator,
			DisplayName:  "数据/文件处理工程师",
			Docs:         []string{"/system/prompt/roles/data-operator"},
			AllowedTools: []string{"change_role", "read_doc", "search", "run_form_submit", "run_python", "send_notification"},
			Runtime: runtimeContract(
				[]string{"复杂、专项或多步骤的一次性文件、媒体、数据处理、批量转换、OCR、音视频转码、压缩、临时图表生成或需要完整数据处理 SOP 的任务"},
				[]string{"用户明确要求沉淀为长期业务系统", "只是轻量一次性文件/数据处理，默认角色 app_operator 已能用 run_python 直接完成"},
				[]string{"确认输入文件或数据", "优先复用已有工具和官方能力", "按常见默认值克制处理", "返回产物或简洁结果；如果目标是长期系统且已有 file_profile，则直接交接产品经理，不重复读取文件", "如果任务只是简单转换、压缩、清洗、加水印、解析附件或整理临时结果，应交接回 app_operator，避免让用户感知多余换挡"},
				[]string{"文件/数据处理完成并返回产物", "或需要长期系统时交接产品经理"},
				[]LifecycleHook{
					hook("data_operator.before_enter_inputs", "before_enter", "整理上传文件、目标格式和一次性处理参数。", []string{"attached files", "user request"}, []string{"processing_inputs"}),
				},
			),
			Action:           "数据/文件处理工程师处理复杂、专项或多步骤的一次性文件、媒体和数据任务，不沉淀长期业务应用；简单短文件任务优先由默认应用执行完成。",
			RouteDescription: "用户要做复杂、专项或多步骤的一次性文件、媒体、数据处理、批量转换、OCR、音视频转码、压缩、临时图表生成等任务时进入。简单转换、压缩、清洗、改尺寸、加水印、解析附件或整理临时结果这类轻量任务默认由 `app_operator` 用 `run_python` 直接完成，不要让用户感知多余换挡。优先复用已有官方工具和运行工具，不要误判成长期应用开发。用户明确要求沉淀为业务系统、记录管理、统计看板，或上传文件后表达“做成系统/后台/应用”时，应切到 `product_manager`；如果已有 `file_profile`，不要重复读取文件，只把画像作为交接关键信息。",
			NextRoles: []NextRole{
				{RoleID: ProductManager, When: "用户明确要求沉淀为长期业务系统"},
				{RoleID: AppOperator, When: "任务只是轻量一次性文件/数据处理，默认执行角色可直接完成"},
			},
		},
		PlatformEngineer: {
			ID:           PlatformEngineer,
			DisplayName:  "平台集成工程师",
			Docs:         []string{"/system/prompt/roles/platform-engineer"},
			AllowedTools: []string{"change_role", "read_doc", "search", "run_form_submit", "send_notification"},
			Runtime: runtimeContract(
				[]string{"需要平台 OpenAPI、权限、审计、组织、文件或平台集成能力"},
				[]string{"业务应用内普通 CRUD、测试或一次性文件处理"},
				[]string{"确认平台边界和权限", "读取平台文档或 API schema", "使用平台工具执行或指导集成", "不绕过权限、不硬编码 token"},
				[]string{"平台能力调用完成", "或边界/权限问题已说明"},
				[]LifecycleHook{
					hook("platform.before_enter_boundary", "before_enter", "确认平台能力边界和所需权限。", []string{"user request", "platform docs"}, []string{"platform_boundary_context"}),
				},
			),
			Action:           "平台集成工程师负责平台 OpenAPI、权限、审计、组织和文件等平台能力；不绕过权限。",
			RouteDescription: "用户要调用平台权限、审计、组织或文件等平台能力时进入。优先使用平台提供的 API 和工具，不绕过权限，不硬编码 token，不直连内部服务。",
		},
		Reviewer: {
			ID:             Reviewer,
			DisplayName:    "代码审查分析师",
			Docs:           []string{"/system/prompt/roles/reviewer"},
			Optional:       []string{"/system/prompt/platform-introduction", "/system/prompt/platform-usage-and-philosophy", "/system/prompt/platform-capability-boundaries"},
			AllowedTools:   []string{"change_role", "summarize_task_state", "read_doc", "read_dir", "read_file"},
			ForbiddenTools: []string{"write_prd", "create_directory", "write_doc", "write_file", "edit_file", "delete_file", "build_workspace", "run_form_submit"},
			Runtime: runtimeContract(
				[]string{"用户要解释、review、查问题、读代码或做方案评估", "用户咨询 Kageos 身份、公司、协议、Hub、怎么用、工作台能做什么、产品理念、服务目录或能力边界"},
				[]string{"用户明确要求直接修改、构建或执行业务操作"},
				[]string{"只读读取目录、源码和文档", "身份/公司/协议/Hub 类问题先读 /system/prompt/platform-introduction；使用方式和理念类问题读 /system/prompt/platform-usage-and-philosophy；能力边界类问题读 /system/prompt/platform-capability-boundaries", "按风险、证据和行号输出结论；介绍和使用类问题只按文档稳定口径回答，并给可执行下一步", "需要修改时交接维护或产品角色"},
				[]string{"已给出分析、风险、方案、介绍或使用说明", "或已明确下一角色和交接摘要"},
				[]LifecycleHook{
					hook("reviewer.before_handoff", "before_handoff", "把只读分析结论压缩成下一角色可执行摘要。", []string{"analysis findings", "user decision"}, []string{"handoff_summary"}),
				},
			),
			Action: "代码审查分析师以只读方式分析项目、解释代码、review 风险和改进建议，也负责说明 Kageos 身份、公司、协议、使用方式、产品理念和能力边界。",
			RouteDescription: "用户要解释项目、review、查问题、读代码、做方案评估，或询问 Kageos 是什么、你是谁、介绍公司、协议/商用边界、Hub/企业版、Kageos 怎么用、工作台能做什么、为什么是服务目录、产品理念、能力边界时进入。只读目录、源码和文档，不落盘、不构建、不调用会产生业务副作用的运行工具；身份/公司/协议/Hub 类问题读 `/system/prompt/platform-introduction`，使用方式和理念类问题读 `/system/prompt/platform-usage-and-philosophy`，涉及能不能做、平台侧/应用侧时读 `/system/prompt/platform-capability-boundaries`；除非用户明确要求继续修改或验证。" +
				"审查 PRD 链路时关注 PRD v2、派生功能顺序、search_fields 是否被误当业务字段。",
			NextRoles: []NextRole{
				{RoleID: MaintenanceEngineer, When: "用户确认要修改"},
				{RoleID: ProductManager, When: "用户要求新建长期业务系统"},
			},
		},
	}
	return cloneSpecs(specs)
}

func Aliases() map[string]string {
	return map[string]string{
		"product-manager":      ProductManager,
		"app-developer":        AppDeveloper,
		"maintenance-engineer": MaintenanceEngineer,
		"qa-engineer":          QAEngineer,
		"app-operator":         AppOperator,
		"automation-operator":  AutomationOperator,
		"build-engineer":       BuildEngineer,
		"data-operator":        DataOperator,
		"platform-engineer":    PlatformEngineer,
	}
}

func RouteOrder() []string {
	return []string{
		Router,
		AppOperator,
		AutomationOperator,
		ProductManager,
		AppDeveloper,
		MaintenanceEngineer,
		QAEngineer,
		BuildEngineer,
		DataOperator,
		PlatformEngineer,
		Reviewer,
	}
}

func Normalize(role string) string {
	role = strings.TrimSpace(role)
	switch role {
	case "", "none":
		return ""
	}
	if alias, ok := Aliases()[role]; ok {
		return alias
	}
	if _, ok := Specs()[role]; ok {
		return role
	}
	return role
}

func IsKnown(role string) bool {
	role = Normalize(role)
	if role == "" {
		return false
	}
	_, ok := Specs()[role]
	return ok
}

func SpecFor(role string) (Spec, bool) {
	role = Normalize(role)
	spec, ok := Specs()[role]
	return spec, ok
}

func DisplayName(role string) string {
	if spec, ok := SpecFor(role); ok {
		return spec.DisplayName
	}
	return strings.TrimSpace(role)
}

func NextRolesFor(role string) []NextRole {
	spec, ok := SpecFor(role)
	if !ok {
		return nil
	}
	return append([]NextRole(nil), spec.NextRoles...)
}

func TransitionWhen(fromRole, toRole string) (string, bool) {
	fromRole = Normalize(fromRole)
	toRole = Normalize(toRole)
	if fromRole == "" || toRole == "" {
		return "", false
	}
	for _, next := range NextRolesFor(fromRole) {
		if Normalize(next.RoleID) == toRole {
			return strings.TrimSpace(next.When), true
		}
	}
	return "", false
}

func RoutingMarkdown() string {
	specs := Specs()
	var b strings.Builder
	b.WriteString("## 角色路由\n\n")
	for idx, roleID := range RouteOrder() {
		spec, ok := specs[roleID]
		if !ok || strings.TrimSpace(spec.RouteDescription) == "" {
			continue
		}
		if idx > 0 {
			b.WriteString("\n")
		}
		b.WriteString("### `")
		b.WriteString(spec.ID)
		b.WriteString("` ")
		b.WriteString(spec.DisplayName)
		b.WriteString("\n\n")
		b.WriteString(strings.TrimSpace(spec.RouteDescription))
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func ApplyRoutingMarkdown(systemPrompt string) string {
	if !strings.Contains(systemPrompt, RoutingMarker) {
		return systemPrompt
	}
	return strings.ReplaceAll(systemPrompt, RoutingMarker, RoutingMarkdown())
}

func runtimeContract(entry, forbidden, sop, done []string, hooks []LifecycleHook) RuntimeContract {
	return RuntimeContract{
		EntryConditions:     append([]string(nil), entry...),
		ForbiddenConditions: append([]string(nil), forbidden...),
		SOP:                 append([]string(nil), sop...),
		DoneWhen:            append([]string(nil), done...),
		HandoffRequired:     []string{"execute_directory", "task_context", "key_information", "references"},
		Hooks:               append([]LifecycleHook(nil), hooks...),
	}
}

func hook(id, stage, purpose string, reads, produces []string) LifecycleHook {
	return LifecycleHook{
		ID:       id,
		Stage:    stage,
		Purpose:  purpose,
		Reads:    append([]string(nil), reads...),
		Produces: append([]string(nil), produces...),
	}
}

func cloneSpecs(in map[string]Spec) map[string]Spec {
	out := make(map[string]Spec, len(in))
	for key, spec := range in {
		out[key] = cloneSpec(spec)
	}
	return out
}

func cloneSpec(spec Spec) Spec {
	spec.Docs = append([]string(nil), spec.Docs...)
	spec.Optional = append([]string(nil), spec.Optional...)
	spec.AllowedTools = append([]string(nil), spec.AllowedTools...)
	spec.ForbiddenTools = append([]string(nil), spec.ForbiddenTools...)
	spec.Runtime = cloneRuntimeContract(spec.Runtime)
	spec.NextRoles = append([]NextRole(nil), spec.NextRoles...)
	return spec
}

func cloneRuntimeContract(runtime RuntimeContract) RuntimeContract {
	runtime.EntryConditions = append([]string(nil), runtime.EntryConditions...)
	runtime.ForbiddenConditions = append([]string(nil), runtime.ForbiddenConditions...)
	runtime.SOP = append([]string(nil), runtime.SOP...)
	runtime.DoneWhen = append([]string(nil), runtime.DoneWhen...)
	runtime.HandoffRequired = append([]string(nil), runtime.HandoffRequired...)
	runtime.Hooks = cloneLifecycleHooks(runtime.Hooks)
	return runtime
}

func cloneLifecycleHooks(hooks []LifecycleHook) []LifecycleHook {
	out := make([]LifecycleHook, 0, len(hooks))
	for _, h := range hooks {
		h.Reads = append([]string(nil), h.Reads...)
		h.Produces = append([]string(nil), h.Produces...)
		out = append(out, h)
	}
	return out
}
