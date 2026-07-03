package service

import (
	"strings"
	"testing"
)

func TestWorkspaceRoleSpecProductManager(t *testing.T) {
	got, ok := workspaceRoleSpecFor(WorkspaceRoleProductManager)
	if !ok || got.ID != WorkspaceRoleProductManager {
		t.Fatalf("spec ID=%s ok=%v want %s", got.ID, ok, WorkspaceRoleProductManager)
	}
	if !containsWorkspaceRoleString(got.Docs, "/system/prompt/roles/product-manager") {
		t.Fatalf("product_manager should require role SOP, docs=%v", got.Docs)
	}
	if !containsWorkspaceRoleString(got.AllowedTools, "write_prd") {
		t.Fatalf("product_manager should allow write_prd, tools=%v", got.AllowedTools)
	}
	if strings.TrimSpace(got.RouteDescription) == "" {
		t.Fatalf("product_manager should expose route description")
	}
}

func TestWorkspaceRoleSpecRouterIsReadOnlyFallback(t *testing.T) {
	got, ok := workspaceRoleSpecFor(WorkspaceRoleRouter)
	if !ok || got.ID != WorkspaceRoleRouter {
		t.Fatalf("spec ID=%s ok=%v want %s", got.ID, ok, WorkspaceRoleRouter)
	}
	if !containsWorkspaceRoleString(got.Docs, "/system/prompt/roles/router") {
		t.Fatalf("router should require router handbook, docs=%v", got.Docs)
	}
	if !containsWorkspaceRoleString(got.AllowedTools, "change_role") ||
		!containsWorkspaceRoleString(got.AllowedTools, "read_doc") {
		t.Fatalf("router should allow read and change_role tools, tools=%v", got.AllowedTools)
	}
	for _, blocked := range []string{"write_prd", "edit_file", "build_workspace", "run_form_submit"} {
		if !containsWorkspaceRoleString(got.ForbiddenTools, blocked) {
			t.Fatalf("router should forbid %s, forbidden=%v", blocked, got.ForbiddenTools)
		}
	}
	if !strings.Contains(got.RouteDescription, "兜底") ||
		!strings.Contains(got.RouteDescription, "/system/prompt/roles/router") ||
		!strings.Contains(got.RouteDescription, "3 步急救流程") ||
		!strings.Contains(got.RouteDescription, "同一轮内") {
		t.Fatalf("router route description should explain fallback handbook: %q", got.RouteDescription)
	}
	if !containsWorkspaceRoleString(workspaceStandardRoleIDs(), WorkspaceRoleRouter) {
		t.Fatalf("router should be included in standard role ids: %v", workspaceStandardRoleIDs())
	}
}

func TestWorkspaceRoleSpecAppDeveloper(t *testing.T) {
	got, ok := workspaceRoleSpecFor(WorkspaceRoleAppDeveloper)
	if !ok || got.ID != WorkspaceRoleAppDeveloper {
		t.Fatalf("spec ID=%s ok=%v want %s", got.ID, ok, WorkspaceRoleAppDeveloper)
	}
	if !containsWorkspaceRoleString(got.Docs, "/system/prompt/sdk/agent-app-sdk-readme") {
		t.Fatalf("app_developer should require full SDK readme, docs=%v", got.Docs)
	}
	if containsWorkspaceRoleString(got.AllowedTools, "write_prd") {
		t.Fatalf("app_developer should not output PRD again, tools=%v", got.AllowedTools)
	}
	if !strings.Contains(got.RouteDescription, "`tables.fields` 是模型字段，`tables.search_fields` 是查询请求字段") {
		t.Fatalf("app_developer route description should document PRD v2 field split: %q", got.RouteDescription)
	}
}

func TestWorkspaceRoleSpecAppOperator(t *testing.T) {
	got, ok := workspaceRoleSpecFor(WorkspaceRoleAppOperator)
	if !ok {
		t.Fatal("app_operator spec missing")
	}
	if !containsWorkspaceRoleString(got.Docs, "/system/prompt/roles/app-operator") {
		t.Fatalf("app_operator should require role SOP, docs=%v", got.Docs)
	}
	if !containsWorkspaceRoleString(got.AllowedTools, "run_table_create") ||
		!containsWorkspaceRoleString(got.AllowedTools, "run_form_submit") {
		t.Fatalf("app_operator should allow business run tools, tools=%v", got.AllowedTools)
	}
	if !containsWorkspaceRoleString(got.AllowedTools, "run_python") {
		t.Fatalf("app_operator should allow run_python for lightweight calculation and file processing, tools=%v", got.AllowedTools)
	}
	if !containsWorkspaceRoleString(got.AllowedTools, "list_scheduled_tasks") {
		t.Fatalf("app_operator should allow read-only scheduled task listing, tools=%v", got.AllowedTools)
	}
	if !containsWorkspaceRoleString(got.AllowedTools, "list_scheduled_task_executions") {
		t.Fatalf("app_operator should allow read-only scheduled execution listing, tools=%v", got.AllowedTools)
	}
	if containsWorkspaceRoleString(got.AllowedTools, "write_file") {
		t.Fatalf("app_operator should not write code, tools=%v", got.AllowedTools)
	}
	if !strings.Contains(got.RouteDescription, "目的不是测试验证") {
		t.Fatalf("app_operator route description should distinguish business operation from QA: %q", got.RouteDescription)
	}
}

func TestWorkspaceRoleSpecAutomationOperator(t *testing.T) {
	got, ok := workspaceRoleSpecFor(WorkspaceRoleAutomationOperator)
	if !ok {
		t.Fatal("automation_operator spec missing")
	}
	if !containsWorkspaceRoleString(got.Docs, "/system/prompt/roles/automation-operator") {
		t.Fatalf("automation_operator should require role SOP, docs=%v", got.Docs)
	}
	for _, tool := range []string{
		"create_scheduled_function_task",
		"create_scheduled_agent_task",
		"list_scheduled_tasks",
		"manage_scheduled_task",
		"list_scheduled_task_executions",
	} {
		if !containsWorkspaceRoleString(got.AllowedTools, tool) {
			t.Fatalf("automation_operator should allow %s, tools=%v", tool, got.AllowedTools)
		}
	}
	if containsWorkspaceRoleString(got.AllowedTools, "run_form_submit") {
		t.Fatalf("automation_operator should not directly run business tools, tools=%v", got.AllowedTools)
	}
	if !strings.Contains(got.RouteDescription, "以后自动执行") {
		t.Fatalf("automation_operator route description should distinguish scheduled work from immediate operations: %q", got.RouteDescription)
	}
	if !strings.Contains(got.RouteDescription, "当前目录、本空间其他目录、其他空间函数、系统工具和连接器函数") ||
		!strings.Contains(got.RouteDescription, "不要把示例规则机械套到所有任务") {
		t.Fatalf("automation_operator route description should explain scheduled session resource orchestration and scenario-specific quality control: %q", got.RouteDescription)
	}
}

func TestWorkspaceRoleSpecBuildEngineer(t *testing.T) {
	got, ok := workspaceRoleSpecFor(WorkspaceRoleBuildEngineer)
	if !ok || got.ID != WorkspaceRoleBuildEngineer {
		t.Fatalf("spec ID=%s ok=%v want %s", got.ID, ok, WorkspaceRoleBuildEngineer)
	}
	if !containsWorkspaceRoleString(got.Docs, "/system/prompt/roles/build-engineer") {
		t.Fatalf("build engineer should include role SOP, docs=%v", got.Docs)
	}
	if containsWorkspaceRoleString(got.Docs, "/system/prompt/sdk/build-validation-reference") {
		t.Fatalf("build engineer should not auto-inject retired build reference, docs=%v", got.Docs)
	}
}

func TestWorkspaceRoleSpecReviewerCoversIntroductionUsageAndPhilosophy(t *testing.T) {
	got, ok := workspaceRoleSpecFor(WorkspaceRoleReviewer)
	if !ok || got.ID != WorkspaceRoleReviewer {
		t.Fatalf("spec ID=%s ok=%v want %s", got.ID, ok, WorkspaceRoleReviewer)
	}
	for _, doc := range []string{
		"/system/prompt/platform-introduction",
		"/system/prompt/platform-usage-and-philosophy",
		"/system/prompt/platform-capability-boundaries",
	} {
		if !containsWorkspaceRoleString(got.Optional, doc) {
			t.Fatalf("reviewer should expose optional doc %s, optional=%v", doc, got.Optional)
		}
	}
	for _, want := range []string{
		"Kageos 是什么",
		"Hub/企业版",
		"/system/prompt/platform-introduction",
		"Kageos 怎么用",
		"产品理念",
		"/system/prompt/platform-usage-and-philosophy",
	} {
		if !strings.Contains(got.RouteDescription, want) {
			t.Fatalf("reviewer route description should contain %q, got %q", want, got.RouteDescription)
		}
	}
}

func TestWorkspaceRoleSpecUnknownIsNotKnown(t *testing.T) {
	if _, ok := workspaceRoleSpecFor("unknown.role"); ok {
		t.Fatal("unknown role should not resolve to a role spec")
	}
}
