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
	if containsWorkspaceRoleString(got.AllowedTools, "write_go_file") {
		t.Fatalf("app_operator should not write code, tools=%v", got.AllowedTools)
	}
	if !strings.Contains(got.RouteDescription, "目的不是测试验证") {
		t.Fatalf("app_operator route description should distinguish business operation from QA: %q", got.RouteDescription)
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

func TestWorkspaceRoleSpecUnknownIsNotKnown(t *testing.T) {
	if _, ok := workspaceRoleSpecFor("unknown.role"); ok {
		t.Fatal("unknown role should not resolve to a role spec")
	}
}
