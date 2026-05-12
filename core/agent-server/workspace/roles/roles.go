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
	SchedulerEngineer   = "scheduler_engineer"
	PlatformEngineer    = "platform_engineer"
	DataOperator        = "data_operator"
	WorkflowEngineer    = "workflow_engineer"
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
	NextRoles        []NextRole
	Action           string
	RouteDescription string
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
			Action:       "识别用户最新需求并调度到合适角色；不执行写入、构建或业务运行副作用。",
		},
		ProductManager: {
			ID:           ProductManager,
			DisplayName:  "产品经理",
			Docs:         []string{"/system/prompt/roles/product-manager"},
			Optional:     []string{"/system/prompt/case_catalog"},
			AllowedTools: []string{"change_role", "read_doc", "read_dir", "write_prd"},
			ForbiddenTools: []string{
				"create_directory", "write_go_file", "search_replace_file", "delete_file", "build_workspace",
				"run_table_search", "run_table_create", "run_table_batch_create", "run_table_update", "run_table_delete",
				"run_form_submit", "run_chart_query", "run_on_select_fuzzy",
			},
			Action:           "产品经理只负责需求分析、PRD v2 结构化输出和确认；调用 write_prd 后等待用户确认，不创建目录、不写代码、不 build。",
			RouteDescription: "用户要新建系统、目录、Form、Table、Chart 或管理后台，但还没有确认 PRD 时进入。只负责把需求拆成可确认的 PRD artifact：`project/tables/forms/charts/workflow/rules`；调用 `write_prd` 后等待用户确认，不创建目录、不写代码、不 build。",
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
			ForbiddenTools:   []string{"write_prd"},
			Action:           "应用开发工程师只按已确认 PRD v2 开发执行；区分 tables.fields 模型字段和 tables.search_fields 查询字段，读取 SDK 和案例，创建目录、写 Go 文件并统一 build，不重新输出 PRD。",
			RouteDescription: "用户已确认 PRD，或确认按钮开启的新会话携带完整 PRD JSON 时进入。只按 PRD v2 直接开发：读取匹配案例，创建目录，生成 Go struct 和函数代码，注册路由并统一 build；`tables.fields` 是模型字段，`tables.search_fields` 是查询请求字段；不要重新输出 PRD，不要再次询问确认。",
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
			ForbiddenTools:   []string{"write_prd"},
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
				"run_table_search", "run_table_create", "run_table_batch_create", "run_table_update", "run_table_delete",
				"run_form_submit", "run_chart_query", "run_on_select_fuzzy",
			},
			ForbiddenTools:   []string{"write_prd", "create_directory", "write_go_file", "search_replace_file", "delete_file", "build_workspace"},
			Action:           "测试工程师确认 schema 后按 workflow 验证 Table/Form/Chart，覆盖时间范围和用户筛选，并调用 run_* 工具；不直接改代码。",
			RouteDescription: "用户要查数据、提交表单、查图表、测试刚生成的应用或验证已有函数时进入。先确认目标函数、schema、必填字段、枚举、文件字段、搜索字段和写入能力，再调用运行工具。重点验证 Table 查询、时间范围筛选、用户筛选、Form 写入目标表和 Chart 统计。测试失败时判断是业务数据问题、代码问题还是构建/schema 问题，并建议切换到 `maintenance_engineer` 或 `build_engineer`。",
			NextRoles: []NextRole{
				{RoleID: MaintenanceEngineer, When: "测试发现业务 bug"},
				{RoleID: BuildEngineer, When: "测试失败指向构建、schema 或路由问题"},
				{RoleID: SchedulerEngineer, When: "功能稳定后需要定时执行"},
			},
		},
		AppOperator: {
			ID:          AppOperator,
			DisplayName: "应用操作员",
			Docs:        []string{"/system/prompt/roles/app-operator"},
			AllowedTools: []string{
				"change_role", "summarize_task_state", "read_doc", "read_dir", "search_tools",
				"run_table_search", "run_table_create", "run_table_batch_create", "run_table_update", "run_table_delete",
				"run_form_submit", "run_chart_query", "run_on_select_fuzzy",
			},
			ForbiddenTools:   []string{"write_prd", "create_directory", "write_go_file", "search_replace_file", "delete_file", "build_workspace"},
			Action:           "应用操作员负责在已有应用里执行业务操作：查询、新增、更新、删除记录、提交表单、查看图表；操作前确认目标函数和关键字段，不改 PRD、不改代码。",
			RouteDescription: "用户要在已有应用中创建业务数据、修改记录、删除记录、提交表单、查询列表或查看图表，但目的不是测试验证时进入。例如创建投票、录入工单、提交评分、更新状态、查询统计。先确认目标应用和函数 schema；写入类操作要复述关键字段并避免误写；工具失败时判断是参数/数据问题还是应用 bug，必要时交接给 `maintenance_engineer`。",
			NextRoles: []NextRole{
				{RoleID: MaintenanceEngineer, When: "业务操作失败且判断为应用 bug 或字段实现问题"},
				{RoleID: SchedulerEngineer, When: "用户要求把操作改成周期性执行"},
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
			ForbiddenTools:   []string{"write_prd", "run_table_search", "run_table_create", "run_table_batch_create", "run_table_update", "run_table_delete", "run_form_submit", "run_chart_query"},
			Action:           "构建修复工程师负责构建、启动、schema、widget、搜索字段和 SDK API 排错；按错误类型批量修复并重新 build。",
			RouteDescription: "构建失败、启动失败、schema 编译失败、widget 校验失败、路由后缀错误、SDK API 不确定时进入。读取完整错误，按错误类型归类，同类问题批量修复。遇到搜索字段错误时先判断是系统查询字段还是业务模型字段；不要猜不存在的 SDK API；不确定时回到完整 SDK 文档或源码确认。",
			NextRoles: []NextRole{
				{RoleID: QAEngineer, When: "build 修复成功后验证功能"},
				{RoleID: MaintenanceEngineer, When: "错误指向业务逻辑或字段设计缺陷"},
			},
		},
		DataOperator: {
			ID:               DataOperator,
			DisplayName:      "数据/文件处理工程师",
			Docs:             []string{"/system/prompt/roles/data-operator"},
			AllowedTools:     []string{"change_role", "read_doc", "search_tools", "run_form_submit", "run_python", "fetch_url_content", "web_search"},
			Action:           "数据/文件处理工程师处理一次性文件、媒体和数据任务，不沉淀长期业务应用。",
			RouteDescription: "用户要做一次性文件、媒体、数据处理、图表生成、格式转换、OCR、压缩、转码等杂活时进入。优先复用已有官方工具和运行工具，不要误判成长期应用开发。只有用户明确要求沉淀为业务系统、记录管理或统计看板时，才切到 `product_manager`。",
			NextRoles: []NextRole{
				{RoleID: ProductManager, When: "用户明确要求沉淀为长期业务系统"},
			},
		},
		WorkflowEngineer: {
			ID:          WorkflowEngineer,
			DisplayName: "工作流编排工程师",
			Docs:        []string{"/system/prompt/roles/workflow-engineer"},
			Optional:    []string{"/system/prompt/case_catalog/workflow"},
			AllowedTools: []string{
				"change_role", "summarize_task_state", "read_doc", "read_dir", "search_tools", "search_resources",
				"run_form_submit", "write_doc", "create_workflow",
			},
			ForbiddenTools: []string{
				"write_prd", "create_directory", "write_go_file", "search_replace_file", "delete_file", "build_workspace",
				"run_table_create", "run_table_batch_create", "run_table_update", "run_table_delete",
			},
			Action:           "工作流编排工程师负责把已存在的 Form/Table/Chart/Agent 能力梳理成 workflow.v1 定义；先搜索资源和 schema，再调用 create_workflow 落库，不写业务代码。",
			RouteDescription: "用户要把多个 Form 或工具串起来、让上一步输出作为下一步输入、创建或修改 workflow、生成 workflow JSON、设计工作流模板、做画布编排或希望 Agent 自动编排已有能力时进入。先用 `search_resources/search_tools` 找真实函数和字段摘要，MVP 使用 `workflow.v1` 的 `form.submit` 顺序图定义；需要创建时调用 `create_workflow`，不要让用户手动复制粘贴 JSON；不要凭函数名猜 `full_code_path` 或字段名。缺少底层能力时交接给 `product_manager`、`app_developer` 或 `maintenance_engineer`。",
			NextRoles: []NextRole{
				{RoleID: ProductManager, When: "需要新增长期业务应用或能力包"},
				{RoleID: AppDeveloper, When: "已有 PRD 且需要开发缺失的 Form/Table/Chart 能力"},
				{RoleID: MaintenanceEngineer, When: "需要改造已有函数的入参、出参或副作用边界"},
				{RoleID: QAEngineer, When: "workflow 草稿发布后需要验证运行链路"},
				{RoleID: SchedulerEngineer, When: "workflow 稳定后需要定时触发"},
			},
		},
		SchedulerEngineer: {
			ID:          SchedulerEngineer,
			DisplayName: "定时任务工程师",
			Docs:        []string{"/system/prompt/roles/scheduler-engineer"},
			AllowedTools: []string{
				"change_role", "read_doc", "search_tools", "run_form_submit",
				"create_scheduled_task", "list_scheduled_tasks", "cancel_scheduled_task", "list_scheduled_task_executions",
				"create_scheduled_agent_task", "list_scheduled_agent_tasks", "list_scheduled_agent_task_executions", "cancel_scheduled_session_task", "run_scheduled_agent_task_now",
			},
			Action:           "定时任务工程师负责定时任务配置和执行记录；没有单次能力时交接给产品经理或应用维护工程师。",
			RouteDescription: "用户要求每天、每周、每月、固定时间或周期执行时进入。先判断是否已有可重入的单次函数；没有就先切到 `product_manager` 或 `maintenance_engineer` 创建单次能力。业务代码只做单次执行，不写后台循环，不自建 cron。",
			NextRoles: []NextRole{
				{RoleID: ProductManager, When: "没有单次执行能力，需要先设计新应用"},
				{RoleID: MaintenanceEngineer, When: "需要改造已有单次执行能力"},
			},
		},
		PlatformEngineer: {
			ID:               PlatformEngineer,
			DisplayName:      "平台集成工程师",
			Docs:             []string{"/system/prompt/roles/platform-engineer"},
			AllowedTools:     []string{"change_role", "read_doc", "search_tools", "run_form_submit", "fetch_url_content", "web_search"},
			Action:           "平台集成工程师负责平台 OpenAPI、消息、权限、审计、组织、文件和定时任务等平台能力；不绕过权限。",
			RouteDescription: "用户要调用平台消息、权限、审计、组织、文件或定时任务等平台能力时进入。优先使用平台提供的 API 和工具，不绕过权限，不硬编码 token，不直连内部服务。Hub 相关 OpenAPI 暂不暴露。",
		},
		Reviewer: {
			ID:             Reviewer,
			DisplayName:    "代码审查分析师",
			Docs:           []string{"/system/prompt/roles/reviewer"},
			AllowedTools:   []string{"change_role", "summarize_task_state", "read_doc", "read_dir", "read_go_file", "read_go_file_lines"},
			ForbiddenTools: []string{"write_prd", "create_directory", "write_go_file", "search_replace_file", "delete_file", "build_workspace", "run_form_submit"},
			Action:         "代码审查分析师以只读方式分析项目、解释代码、review 风险和改进建议。",
			RouteDescription: "用户要解释项目、review、查问题、读代码或做方案评估时进入。只读目录、源码和文档，不落盘、不构建、不调用会产生业务副作用的运行工具，除非用户明确要求继续修改或验证。" +
				"审查 PRD 链路时关注 PRD v2、workflow 顺序、search_fields 是否被误当业务字段。",
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
		"workflow-engineer":    WorkflowEngineer,
		"scheduler-engineer":   SchedulerEngineer,
		"platform-engineer":    PlatformEngineer,
	}
}

func RouteOrder() []string {
	return []string{
		ProductManager,
		AppDeveloper,
		MaintenanceEngineer,
		AppOperator,
		QAEngineer,
		BuildEngineer,
		DataOperator,
		WorkflowEngineer,
		SchedulerEngineer,
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
	spec.NextRoles = append([]NextRole(nil), spec.NextRoles...)
	return spec
}
