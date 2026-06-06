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
	ID       string   `json:"id" schema_desc:"Hook 标识" schema_required:"true"`
	Stage    string   `json:"stage" schema_desc:"触发阶段，例如 before_enter/after_tool/before_handoff" schema_required:"true"`
	Purpose  string   `json:"purpose" schema_desc:"Hook 目的" schema_required:"true"`
	Reads    []string `json:"reads,omitempty" schema_desc:"Hook 读取的输入或资料"`
	Produces []string `json:"produces,omitempty" schema_desc:"Hook 产出的上下文或产物"`
}

type NextRole struct {
	RoleID string `json:"role_id" schema_desc:"后续角色 ID" schema_required:"true"`
	When   string `json:"when" schema_desc:"触发条件" schema_required:"true"`
}

func Specs() map[string]Spec {
	specs := map[string]Spec{
		Router: {
			ID:           Router,
			DisplayName:  "工作台调度员",
			Docs:         []string{"/system/prompt/roles/reviewer"},
			AllowedTools: []string{"change_role", "read_doc", "read_dir"},
			Runtime: runtimeContract(
				[]string{"需要识别用户意图、解释代码或做只读分析"},
				[]string{"已经明确需要某个专业角色直接执行"},
				[]string{"结合当前目录理解用户意图", "选择或沿用一个明确角色", "不执行写入、构建或业务运行副作用"},
				[]string{"已通过 change_role 明确目标角色", "已说明无法判断时所需的最小补充信息"},
				nil,
			),
			Action: "识别用户最新需求并调度到合适角色；不执行写入、构建或业务运行副作用。",
		},
		ProductManager: {
			ID:           ProductManager,
			DisplayName:  "产品经理",
			Docs:         []string{"/system/prompt/roles/product-manager"},
			Optional:     []string{"/system/prompt/case_catalog"},
			AllowedTools: []string{"change_role", "read_doc", "read_dir", "write_prd"},
			ForbiddenTools: []string{
				"create_directory", "write_go_file", "search_replace_file", "delete_file", "build_workspace",
				"run_table_search", "run_table_create", "run_table_update", "run_table_delete",
				"run_form_submit", "run_chart_query", "run_on_select_fuzzy",
			},
			Runtime: runtimeContract(
				[]string{"用户要新建长期业务系统、应用目录、Form/Table/Chart 或管理后台", "尚未确认结构化 PRD"},
				[]string{"当前目录已有应用且运行函数能完成用户目标", "用户只是要操作现有软件完成业务结果"},
				[]string{"先结合当前目录判断是新建系统还是使用现有应用", "读取必要案例并提炼业务对象、字段、搜索、提交入口、统计和规则", "调用 write_prd 输出 PRD v2", "等待用户确认，不创建目录、不写代码、不 build"},
				[]string{"write_prd 已生成可确认 PRD artifact", "用户确认前保持 pending_confirmation"},
				[]LifecycleHook{
					hook("product_manager.prd_ready", "after_tool", "write_prd 后固化 PRD artifact，供前端确认和后续 handoff 使用。", []string{"write_prd structured data"}, []string{"agent_app_prd artifact", "pending_confirmation interaction"}),
					hook("product_manager.to_app_developer", "before_handoff", "用户确认 PRD 后生成开发执行视图和交接摘要。", []string{"agent_app_prd JSON", "source session notes"}, []string{"PRD_EXECUTION_MARKDOWN", "artifact_digest", "handoff_context"}),
				},
			),
			Action:           "产品经理只负责新建长期业务系统的需求分析、PRD v2 结构化输出和确认；调用 write_prd 后等待用户确认，不创建目录、不写代码、不 build。",
			RouteDescription: "用户明确要新建长期业务系统、应用目录、新 Form/Table/Chart 或管理后台，但还没有确认 PRD 时进入。当前目录已有应用且运行函数能完成用户目标时不要进入产品经理；这类请求是用户在使用软件完成业务结果，应进入 `app_operator`。例如在 `/system/x_world/vote` 里“创建一个投票”是业务操作，不是写 PRD。产品经理只负责把新系统需求拆成可确认的 PRD artifact：`project/tables/forms/charts/rules`；调用 `write_prd` 后等待用户确认，不创建目录、不写代码、不 build。",
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
				"change_role", "summarize_task_state", "read_doc", "read_dir", "read_go_file", "read_go_file_lines",
				"create_directory", "write_go_file", "search_replace_file", "read_app_log", "build_workspace",
			},
			ForbiddenTools: []string{"write_prd"},
			Runtime: runtimeContract(
				[]string{"用户已确认 PRD", "handoff 会话携带完整 agent_app_prd JSON", "用户明确要求新增或改变软件能力"},
				[]string{"用户在已有应用里使用软件完成业务结果", "需求尚未确认 PRD 且不是小范围维护"},
				[]string{"调用 change_role 并固定 execute_directory", "优先阅读 PRD_EXECUTION_MARKDOWN，细节以 PRD JSON 为准", "读取 SDK 主文档和匹配案例", "创建或修改目标目录代码", "统一调用 build_workspace"},
				[]string{"build_workspace 成功并产生 agent_app_build artifact", "或 build/schema 失败并带完整错误交接给 build_engineer"},
				[]LifecycleHook{
					hook("app_developer.before_enter_prd", "before_enter", "整理确认后的 PRD、执行目录和开发参考资料。", []string{"agent_app_prd JSON", "PRD_EXECUTION_MARKDOWN", "reference docs"}, []string{"developer_context_packet"}),
					hook("app_developer.after_build", "after_tool", "build_workspace 后根据成功或失败决定 QA 或构建修复交接。", []string{"build_workspace result"}, []string{"agent_app_build artifact", "build_diagnostics"}),
				},
			),
			Action:           "应用开发工程师只按已确认 PRD v2 开发执行；区分 tables.fields 模型字段和 tables.search_fields 查询字段，读取 SDK 和案例，创建目录、写 Go 文件并统一 build，不重新输出 PRD。",
			RouteDescription: "用户已确认 PRD，或确认按钮开启的新会话携带完整 PRD JSON 时进入。只按 PRD v2 直接开发：读取匹配案例，创建目录，生成 Go struct 和函数代码，注册路由并统一 build；`tables.fields` 是模型字段，`tables.search_fields` 是查询请求字段；不要重新输出 PRD，不要再次询问确认。用户是在已有应用里使用软件完成业务结果，而不是要求新增或改变软件能力时，不要进入开发工程师，应进入 `app_operator`。",
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
				"change_role", "summarize_task_state", "read_doc", "read_dir", "read_go_file", "read_go_file_lines",
				"create_directory", "write_go_file", "search_replace_file", "delete_file", "read_app_log", "build_workspace",
			},
			ForbiddenTools: []string{"write_prd"},
			Runtime: runtimeContract(
				[]string{"用户要修改已有应用、字段、搜索、选项、回调、图表或业务 bug", "测试或操作发现业务实现问题"},
				[]string{"用户要重新设计新系统需求", "只是业务数据操作且当前应用可直接完成"},
				[]string{"固定目标应用 execute_directory", "读取相关目录和源码", "小改局部替换，大改再写完整文件", "修改后 build_workspace", "失败时按错误类型补读文档或交接 build_engineer"},
				[]string{"目标修改已落盘并 build 成功", "或构建问题已交接 build_engineer"},
				[]LifecycleHook{
					hook("maintenance.before_enter_scope", "before_enter", "收敛维护范围，避免扫描或修改无关应用。", []string{"execute_directory", "changed files", "bug report"}, []string{"maintenance_scope"}),
					hook("maintenance.after_build", "after_tool", "构建后决定进入 QA 或构建修复。", []string{"build_workspace result"}, []string{"verification_focus", "build_diagnostics"}),
				},
			),
			Action:           "应用维护工程师负责修改已有应用、字段、搜索、选项、回调、图表和业务 bug；区分业务字段和系统搜索字段，读取相关源码后修改并 build。",
			RouteDescription: "用户要改已有应用、字段、选项、组件、回调、搜索、消息、跳转、图表或业务逻辑时进入。先识别修改类型和影响范围，读取当前目录与相关源码，只改用户目标和必要依赖。新增或修改搜索时区分业务字段和系统字段；小改优先局部替换，大改或新增能力再写完整文件；构建成功后建议进入 `qa_engineer`。",
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
				"change_role", "summarize_task_state", "read_doc", "read_dir", "search_tools",
				"run_table_search", "run_table_create", "run_table_update", "run_table_delete",
				"run_form_submit", "run_chart_query", "run_on_select_fuzzy",
			},
			ForbiddenTools: []string{"write_prd", "create_directory", "write_go_file", "search_replace_file", "delete_file", "build_workspace"},
			Runtime: runtimeContract(
				[]string{"需要测试刚构建的应用", "需要验证已有函数是否符合预期"},
				[]string{"需要修改代码", "用户只是要完成真实业务操作且不是测试验证"},
				[]string{"固定 execute_directory", "用 search_tools/search_resources 获取目标函数 schema", "按业务链路准备测试数据", "调用 run_* 工具验证", "把失败归类为参数、数据、schema、业务 bug 或环境问题"},
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
			DisplayName: "应用操作员",
			Docs:        []string{"/system/prompt/roles/app-operator"},
			AllowedTools: []string{
				"change_role", "summarize_task_state", "read_doc", "read_dir", "search_tools",
				"run_table_search", "run_table_create", "run_table_update", "run_table_delete",
				"run_form_submit", "run_chart_query", "run_on_select_fuzzy",
			},
			ForbiddenTools: []string{"write_prd", "create_directory", "write_go_file", "search_replace_file", "delete_file", "build_workspace"},
			Runtime: runtimeContract(
				[]string{"当前目录已有应用且用户要查询、新增、更新、删除、提交表单或查看图表", "用户意图是使用软件完成业务结果"},
				[]string{"用户要新增或改变软件能力", "用户要测试刚构建应用而不是真实业务操作"},
				[]string{"固定目标应用 execute_directory", "先获取函数 schema、必填项、枚举、文件字段和写入能力", "必要时先查询关联数据或调用 OnSelectFuzzy", "调用 run_* 完成业务操作", "失败时区分参数/数据问题和应用 bug"},
				[]string{"业务操作完成并返回结果", "或失败原因已分类并交接维护角色"},
				[]LifecycleHook{
					hook("app_operator.before_enter_capabilities", "before_enter", "进入操作角色前生成当前应用能力快照。", []string{"execute_directory", "registered functions"}, []string{"available_capabilities", "operation_schema_summary"}),
					hook("app_operator.after_run", "after_tool", "业务运行后判断是否需要继续查询、补参数或交接维护。", []string{"run_* result"}, []string{"operation_result", "failure_classification"}),
				},
			),
			Action:           "应用操作员负责在已有应用里执行业务操作：查询、新增、更新、删除记录、提交表单、查看图表；操作前确认目标函数和关键字段，不改 PRD、不改代码。",
			RouteDescription: "当前目录已是目标应用，或目录下已有 Table/Form/Chart 能完成用户目标时，只要用户是在使用软件完成业务结果且目的不是测试验证，就进入应用操作员。它优先于 `product_manager` 和 `app_developer` 处理真实业务数据操作，不依赖某个固定动词；例如在投票应用目录中“创建一个四大古都投票，北京南京西安洛阳单选”就是新增投票主题和选项。先确认目标应用和函数 schema；写入类操作要复述关键字段并避免误写；工具失败时判断是参数/数据问题还是应用 bug，必要时交接给 `maintenance_engineer`。",
			NextRoles: []NextRole{
				{RoleID: MaintenanceEngineer, When: "业务操作失败且判断为应用 bug 或字段实现问题"},
			},
		},
		BuildEngineer: {
			ID:          BuildEngineer,
			DisplayName: "构建修复工程师",
			Docs:        []string{"/system/prompt/roles/build-engineer"},
			AllowedTools: []string{
				"change_role", "summarize_task_state", "read_doc", "read_go_file", "read_go_file_lines",
				"search_replace_file", "write_go_file", "read_app_log", "build_workspace",
			},
			ForbiddenTools: []string{"write_prd", "run_table_search", "run_table_create", "run_table_update", "run_table_delete", "run_form_submit", "run_chart_query"},
			Runtime: runtimeContract(
				[]string{"build_workspace 失败", "启动、schema、widget、路由后缀或 SDK API 相关错误"},
				[]string{"业务功能本身需要重新设计", "只是测试数据或参数问题"},
				[]string{"固定目标应用目录", "读取完整错误和相关源码", "按 router/字段/文件归类同类问题", "不确定先读 build-validation、SDK 主文档或案例", "批量修复后重新 build"},
				[]string{"build_workspace 成功并可交接 QA", "或确认问题属于业务设计缺陷并交接维护角色"},
				[]LifecycleHook{
					hook("build_engineer.before_enter_diagnostics", "before_enter", "解析构建错误并匹配修复策略和必读文档。", []string{"build error", "workspace path"}, []string{"build_diagnostics", "required_docs", "repair_policy"}),
					hook("build_engineer.after_build", "after_tool", "重新构建后决定 QA 或继续修复。", []string{"build_workspace result"}, []string{"agent_app_build artifact", "remaining_errors"}),
				},
			),
			Action:           "构建修复工程师负责构建、启动、schema、widget、搜索字段和 SDK API 排错；按错误类型批量修复并重新 build。",
			RouteDescription: "构建失败、启动失败、schema 编译失败、widget 校验失败、路由后缀错误、SDK API 不确定时进入。读取完整错误，按错误类型归类，同类问题批量修复。遇到搜索字段错误时先判断是系统查询字段还是业务模型字段；不要猜不存在的 SDK API；不确定时回到完整 SDK 文档或源码确认。",
			NextRoles: []NextRole{
				{RoleID: QAEngineer, When: "build 修复成功后验证功能"},
				{RoleID: MaintenanceEngineer, When: "错误指向业务逻辑或字段设计缺陷"},
			},
		},
		DataOperator: {
			ID:           DataOperator,
			DisplayName:  "数据/文件处理工程师",
			Docs:         []string{"/system/prompt/roles/data-operator"},
			AllowedTools: []string{"change_role", "read_doc", "search_tools", "run_form_submit", "run_python"},
			Runtime: runtimeContract(
				[]string{"一次性文件、媒体、数据处理、格式转换、OCR、压缩、转码或临时图表生成"},
				[]string{"用户明确要求沉淀为长期业务系统"},
				[]string{"确认输入文件或数据", "优先复用已有工具和官方能力", "按常见默认值克制处理", "返回产物或简洁结果"},
				[]string{"文件/数据处理完成并返回产物", "或需要长期系统时交接产品经理"},
				[]LifecycleHook{
					hook("data_operator.before_enter_inputs", "before_enter", "整理上传文件、目标格式和一次性处理参数。", []string{"attached files", "user request"}, []string{"processing_inputs"}),
				},
			),
			Action:           "数据/文件处理工程师处理一次性文件、媒体和数据任务，不沉淀长期业务应用。",
			RouteDescription: "用户要做一次性文件、媒体、数据处理、图表生成、格式转换、OCR、压缩、转码等杂活时进入。优先复用已有官方工具和运行工具，不要误判成长期应用开发。只有用户明确要求沉淀为业务系统、记录管理或统计看板时，才切到 `product_manager`。",
			NextRoles: []NextRole{
				{RoleID: ProductManager, When: "用户明确要求沉淀为长期业务系统"},
			},
		},
		PlatformEngineer: {
			ID:           PlatformEngineer,
			DisplayName:  "平台集成工程师",
			Docs:         []string{"/system/prompt/roles/platform-engineer"},
			AllowedTools: []string{"change_role", "read_doc", "search_tools", "run_form_submit"},
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
			AllowedTools:   []string{"change_role", "summarize_task_state", "read_doc", "read_dir", "read_go_file", "read_go_file_lines"},
			ForbiddenTools: []string{"write_prd", "create_directory", "write_go_file", "search_replace_file", "delete_file", "build_workspace", "run_form_submit"},
			Runtime: runtimeContract(
				[]string{"用户要解释、review、查问题、读代码或做方案评估"},
				[]string{"用户明确要求直接修改、构建或执行业务操作"},
				[]string{"只读读取目录、源码和文档", "按风险、证据和行号输出结论", "需要修改时交接维护或产品角色"},
				[]string{"已给出分析、风险或方案", "或已明确下一角色和交接摘要"},
				[]LifecycleHook{
					hook("reviewer.before_handoff", "before_handoff", "把只读分析结论压缩成下一角色可执行摘要。", []string{"analysis findings", "user decision"}, []string{"handoff_summary"}),
				},
			),
			Action: "代码审查分析师以只读方式分析项目、解释代码、review 风险和改进建议。",
			RouteDescription: "用户要解释项目、review、查问题、读代码或做方案评估时进入。只读目录、源码和文档，不落盘、不构建、不调用会产生业务副作用的运行工具，除非用户明确要求继续修改或验证。" +
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
		"build-engineer":       BuildEngineer,
		"data-operator":        DataOperator,
		"platform-engineer":    PlatformEngineer,
	}
}

func RouteOrder() []string {
	return []string{
		AppOperator,
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
