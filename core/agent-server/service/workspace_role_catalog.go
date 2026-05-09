package service

import "strings"

const (
	WorkspaceRoleRouter              = "router"
	WorkspaceRoleProductManager      = "product_manager"
	WorkspaceRoleAppDeveloper        = "app_developer"
	WorkspaceRoleBuildEngineer       = "build_engineer"
	WorkspaceRoleQAEngineer          = "qa_engineer"
	WorkspaceRoleMaintenanceEngineer = "maintenance_engineer"
	WorkspaceRoleSchedulerEngineer   = "scheduler_engineer"
	WorkspaceRolePlatformEngineer    = "platform_engineer"
	WorkspaceRoleDataOperator        = "data_operator"
	WorkspaceRoleReviewer            = "reviewer"
)

type workspaceRoleSpec struct {
	ID             string
	DisplayName    string
	Docs           []string
	Optional       []string
	AllowedTools   []string
	ForbiddenTools []string
	NextRoles      []nextWorkspaceRole
	Action         string
}

type nextWorkspaceRole struct {
	RoleID string `json:"role_id" schema_desc:"后续角色 ID" schema_required:"true"`
	When   string `json:"when" schema_desc:"触发条件" schema_required:"true"`
}

func workspaceRoleSpecs() map[string]workspaceRoleSpec {
	return map[string]workspaceRoleSpec{
		WorkspaceRoleRouter: {
			ID:           WorkspaceRoleRouter,
			DisplayName:  "工作台调度员",
			Docs:         []string{"/system/prompt/roles/reviewer"},
			AllowedTools: []string{"change_role", "read_doc", "read_dir"},
			Action:       "识别用户最新需求并调度到合适角色；不执行写入、构建或业务运行副作用。",
		},
		WorkspaceRoleProductManager: {
			ID:           WorkspaceRoleProductManager,
			DisplayName:  "产品经理",
			Docs:         []string{"/system/prompt/roles/product-manager"},
			Optional:     []string{"/system/prompt/case_catalog"},
			AllowedTools: []string{"change_role", "read_doc", "read_dir", "write_prd"},
			ForbiddenTools: []string{
				"create_directory", "write_go_file", "search_replace_file", "delete_file", "build_workspace",
				"run_table_search", "run_table_create", "run_table_batch_create", "run_table_update", "run_table_delete",
				"run_form_submit", "run_chart_query", "run_on_select_fuzzy",
			},
			Action: "产品经理只负责需求分析、结构化 PRD 和确认；输出 write_prd 后等待用户确认，不创建目录、不写代码、不 build。",
			NextRoles: []nextWorkspaceRole{
				{RoleID: WorkspaceRoleAppDeveloper, When: "用户确认 PRD 后进入应用开发"},
			},
		},
		WorkspaceRoleAppDeveloper: {
			ID:          WorkspaceRoleAppDeveloper,
			DisplayName: "应用开发工程师",
			Docs:        []string{"/system/prompt/roles/app-developer", "/system/prompt/sdk/agent-app-sdk-readme"},
			Optional:    []string{"/system/prompt/case_catalog"},
			AllowedTools: []string{
				"change_role", "summarize_task_state", "read_doc", "read_dir", "read_go_file", "read_go_file_lines",
				"create_directory", "write_go_file", "search_replace_file", "read_app_log", "build_workspace",
			},
			ForbiddenTools: []string{"write_prd"},
			Action:         "应用开发工程师只按已确认 PRD 开发执行；读取 SDK 和案例，创建目录、写 Go 文件并统一 build，不重新输出 PRD。",
			NextRoles: []nextWorkspaceRole{
				{RoleID: WorkspaceRoleQAEngineer, When: "build 成功后验证核心函数"},
				{RoleID: WorkspaceRoleBuildEngineer, When: "build 失败或 schema compile failed"},
			},
		},
		WorkspaceRoleMaintenanceEngineer: {
			ID:          WorkspaceRoleMaintenanceEngineer,
			DisplayName: "应用维护工程师",
			Docs:        []string{"/system/prompt/roles/maintenance-engineer"},
			AllowedTools: []string{
				"change_role", "summarize_task_state", "read_doc", "read_dir", "read_go_file", "read_go_file_lines",
				"create_directory", "write_go_file", "search_replace_file", "delete_file", "read_app_log", "build_workspace",
			},
			ForbiddenTools: []string{"write_prd"},
			Action:         "应用维护工程师负责修改已有应用、字段、选项、回调、图表和业务 bug；读取相关源码后修改并 build。",
			NextRoles: []nextWorkspaceRole{
				{RoleID: WorkspaceRoleQAEngineer, When: "修改 build 成功后验证功能"},
				{RoleID: WorkspaceRoleBuildEngineer, When: "构建或 schema 校验失败"},
			},
		},
		WorkspaceRoleQAEngineer: {
			ID:          WorkspaceRoleQAEngineer,
			DisplayName: "测试工程师",
			Docs:        []string{"/system/prompt/roles/qa-engineer"},
			AllowedTools: []string{
				"change_role", "summarize_task_state", "read_doc", "read_dir", "search_tools",
				"run_table_search", "run_table_create", "run_table_batch_create", "run_table_update", "run_table_delete",
				"run_form_submit", "run_chart_query", "run_on_select_fuzzy",
			},
			ForbiddenTools: []string{"write_prd", "create_directory", "write_go_file", "search_replace_file", "delete_file", "build_workspace"},
			Action:         "测试工程师读取操作测试文档，确认 schema 后设计测试用例并调用 run_* 工具；不直接改代码。",
			NextRoles: []nextWorkspaceRole{
				{RoleID: WorkspaceRoleMaintenanceEngineer, When: "测试发现业务 bug"},
				{RoleID: WorkspaceRoleBuildEngineer, When: "测试失败指向构建、schema 或路由问题"},
				{RoleID: WorkspaceRoleSchedulerEngineer, When: "功能稳定后需要定时执行"},
			},
		},
		WorkspaceRoleBuildEngineer: {
			ID:          WorkspaceRoleBuildEngineer,
			DisplayName: "构建修复工程师",
			Docs:        []string{"/system/prompt/roles/build-engineer"},
			AllowedTools: []string{
				"change_role", "summarize_task_state", "read_doc", "read_go_file", "read_go_file_lines",
				"search_replace_file", "write_go_file", "read_app_log", "build_workspace",
			},
			ForbiddenTools: []string{"write_prd", "run_table_search", "run_table_create", "run_table_batch_create", "run_table_update", "run_table_delete", "run_form_submit", "run_chart_query"},
			Action:         "构建修复工程师负责构建、启动、schema、widget 和 SDK API 排错；按错误类型批量修复并重新 build。",
			NextRoles: []nextWorkspaceRole{
				{RoleID: WorkspaceRoleQAEngineer, When: "build 修复成功后验证功能"},
				{RoleID: WorkspaceRoleMaintenanceEngineer, When: "错误指向业务逻辑或字段设计缺陷"},
			},
		},
		WorkspaceRoleDataOperator: {
			ID:           WorkspaceRoleDataOperator,
			DisplayName:  "数据/文件处理工程师",
			Docs:         []string{"/system/prompt/roles/data-operator"},
			AllowedTools: []string{"change_role", "read_doc", "search_tools", "run_form_submit", "run_official_python", "fetch_url_content", "web_search"},
			Action:       "数据/文件处理工程师处理一次性文件、媒体和数据任务，不沉淀长期业务应用。",
			NextRoles: []nextWorkspaceRole{
				{RoleID: WorkspaceRoleProductManager, When: "用户明确要求沉淀为长期业务系统"},
			},
		},
		WorkspaceRoleSchedulerEngineer: {
			ID:          WorkspaceRoleSchedulerEngineer,
			DisplayName: "定时任务工程师",
			Docs:        []string{"/system/prompt/roles/scheduler-engineer"},
			AllowedTools: []string{
				"change_role", "read_doc", "search_tools", "run_form_submit",
				"create_scheduled_task", "list_scheduled_tasks", "cancel_scheduled_task", "list_scheduled_task_executions",
				"create_scheduled_agent_task", "list_scheduled_agent_tasks", "list_scheduled_agent_task_executions", "run_scheduled_agent_task_now",
			},
			Action: "定时任务工程师负责定时任务配置和执行记录；没有单次能力时交接给产品经理或应用维护工程师。",
			NextRoles: []nextWorkspaceRole{
				{RoleID: WorkspaceRoleProductManager, When: "没有单次执行能力，需要先设计新应用"},
				{RoleID: WorkspaceRoleMaintenanceEngineer, When: "需要改造已有单次执行能力"},
			},
		},
		WorkspaceRolePlatformEngineer: {
			ID:           WorkspaceRolePlatformEngineer,
			DisplayName:  "平台集成工程师",
			Docs:         []string{"/system/prompt/roles/platform-engineer"},
			AllowedTools: []string{"change_role", "read_doc", "search_tools", "run_form_submit", "fetch_url_content", "web_search"},
			Action:       "平台集成工程师负责平台 OpenAPI、消息、权限、审计和应用市场等平台能力；不绕过权限。",
		},
		WorkspaceRoleReviewer: {
			ID:             WorkspaceRoleReviewer,
			DisplayName:    "代码审查分析师",
			Docs:           []string{"/system/prompt/roles/reviewer"},
			AllowedTools:   []string{"change_role", "summarize_task_state", "read_doc", "read_dir", "read_go_file", "read_go_file_lines"},
			ForbiddenTools: []string{"write_prd", "create_directory", "write_go_file", "search_replace_file", "delete_file", "build_workspace", "run_form_submit"},
			Action:         "代码审查分析师以只读方式分析项目、解释代码、review 风险和改进建议。",
			NextRoles: []nextWorkspaceRole{
				{RoleID: WorkspaceRoleMaintenanceEngineer, When: "用户确认要修改"},
				{RoleID: WorkspaceRoleProductManager, When: "用户要求新建长期业务系统"},
			},
		},
	}
}

func workspaceRoleAliases() map[string]string {
	return map[string]string{
		"product-manager":      WorkspaceRoleProductManager,
		"app-developer":        WorkspaceRoleAppDeveloper,
		"maintenance-engineer": WorkspaceRoleMaintenanceEngineer,
		"qa-engineer":          WorkspaceRoleQAEngineer,
		"build-engineer":       WorkspaceRoleBuildEngineer,
		"data-operator":        WorkspaceRoleDataOperator,
		"scheduler-engineer":   WorkspaceRoleSchedulerEngineer,
		"platform-engineer":    WorkspaceRolePlatformEngineer,
	}
}

func normalizeWorkspaceRole(role string) string {
	role = strings.TrimSpace(role)
	switch role {
	case "", "none":
		return ""
	}
	if alias, ok := workspaceRoleAliases()[role]; ok {
		return alias
	}
	if _, ok := workspaceRoleSpecs()[role]; ok {
		return role
	}
	return role
}

func isKnownWorkspaceRole(role string) bool {
	role = normalizeWorkspaceRole(role)
	if role == "" {
		return false
	}
	_, ok := workspaceRoleSpecs()[role]
	return ok
}

func workspaceRoleSpecFor(role string) (workspaceRoleSpec, bool) {
	role = normalizeWorkspaceRole(role)
	spec, ok := workspaceRoleSpecs()[role]
	return spec, ok
}

func workspaceRoleDisplayName(role string) string {
	if spec, ok := workspaceRoleSpecFor(role); ok {
		return spec.DisplayName
	}
	return strings.TrimSpace(role)
}

func workspaceRoleAllowedTools(role string) []string {
	if spec, ok := workspaceRoleSpecFor(role); ok {
		return append([]string(nil), spec.AllowedTools...)
	}
	return nil
}

func workspaceRoleForbiddenTools(role string) []string {
	if spec, ok := workspaceRoleSpecFor(role); ok {
		return append([]string(nil), spec.ForbiddenTools...)
	}
	return nil
}

func containsWorkspaceRoleString(list []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, item := range list {
		if strings.TrimSpace(item) == target {
			return true
		}
	}
	return false
}
