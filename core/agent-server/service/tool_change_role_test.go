package service

import (
	"context"
	"strings"
	"testing"
)

func TestBuildChangeRoleLoadsPlanDocs(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		TargetRole: WorkspaceRoleProductManager,
		UserInput:  "帮我搞个 NPS 管理系统",
		Directory:  "/u/app/nps",
	})
	if got.CurrentRole != WorkspaceRoleProductManager || got.RoleID != WorkspaceRoleProductManager {
		t.Fatalf("current role=%s role_id=%s want %s", got.CurrentRole, got.RoleID, WorkspaceRoleProductManager)
	}
	if got.DisplayName != "产品经理" {
		t.Fatalf("display name=%s want 产品经理", got.DisplayName)
	}
	if !containsWorkspaceRoleString(got.RequiredDocs, "/system/prompt/roles/product-manager") {
		t.Fatalf("required docs should include product manager role doc: %v", got.RequiredDocs)
	}
	if !containsWorkspaceRoleString(got.RequiredDocs, "/system/prompt/case_catalog") {
		t.Fatalf("product_manager should include case catalog index: %v", got.RequiredDocs)
	}
	for _, heavyCase := range []string{
		"/system/prompt/case_catalog/formandtable/vote",
		"/system/prompt/case_catalog/form_table_chart/cashier",
		"/system/prompt/case_catalog/table/ticket",
	} {
		if containsWorkspaceRoleString(got.RequiredDocs, heavyCase) {
			t.Fatalf("product_manager should not auto-inject full case %s before PRD confirmation: %v", heavyCase, got.RequiredDocs)
		}
	}
	for _, removed := range []string{
		"/system/prompt/sdk/widget-reference",
		"/system/prompt/sdk/build-validation-reference",
		"/system/prompt/sdk/workbench-tools-sdk-relationship",
		"/system/prompt/platform-overview",
		"/system/prompt/platform-capability-boundaries",
	} {
		if containsWorkspaceRoleString(got.RequiredDocs, removed) {
			t.Fatalf("product_manager should not inject heavy doc %s: %v", removed, got.RequiredDocs)
		}
	}
	if len(got.LoadedDocs) == 0 {
		t.Fatal("expected loaded docs")
	}
}

func TestBuildChangeRoleLoadsCreateDocs(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		TargetRole: WorkspaceRoleAppDeveloper,
		UserInput:  "已确认 PRD，开始创建目录和生成代码",
		Directory:  "/u/app/nps",
	})
	if got.CurrentRole != WorkspaceRoleAppDeveloper || got.RoleID != WorkspaceRoleAppDeveloper {
		t.Fatalf("current role=%s role_id=%s want %s", got.CurrentRole, got.RoleID, WorkspaceRoleAppDeveloper)
	}
	if got.DisplayName != "应用开发工程师" {
		t.Fatalf("display name=%s want 应用开发工程师", got.DisplayName)
	}
	if !containsWorkspaceRoleString(got.RequiredDocs, "/system/prompt/roles/app-developer") {
		t.Fatalf("required docs should include app developer role doc: %v", got.RequiredDocs)
	}
	if !containsWorkspaceRoleString(got.RequiredDocs, "/system/prompt/sdk/agent-app-sdk-readme") {
		t.Fatalf("required docs should include SDK readme: %v", got.RequiredDocs)
	}
	if containsWorkspaceRoleString(got.AllowedNextTools, "write_prd") {
		t.Fatalf("app_developer should not plan PRD again, tools=%v", got.AllowedNextTools)
	}
}

func TestBuildChangeRoleDoesNotInferFromUserInput(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		UserInput: "帮我搞个 NPS 管理系统",
	})
	if got.CurrentRole != WorkspaceRoleReviewer {
		t.Fatalf("current role=%s want %s", got.CurrentRole, WorkspaceRoleReviewer)
	}
	if containsWorkspaceRoleString(got.RequiredDocs, "/system/prompt/roles/product-manager") ||
		containsWorkspaceRoleString(got.RequiredDocs, "/system/prompt/roles/app-developer") {
		t.Fatalf("change_role should not infer product manager/developer from user_input: %v", got.RequiredDocs)
	}
}

func TestBuildChangeRoleKeepsCurrentWhenTargetMissing(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		CurrentRole: WorkspaceRoleAppDeveloper,
		UserInput:   "build_workspace 报错",
	})
	if got.CurrentRole != WorkspaceRoleAppDeveloper {
		t.Fatalf("current role=%s want %s", got.CurrentRole, WorkspaceRoleAppDeveloper)
	}
}

func TestBuildChangeRoleSwitchesAndCarriesSummary(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		CurrentRole:  WorkspaceRoleAppDeveloper,
		TargetRole:   WorkspaceRoleQAEngineer,
		TaskSummary:  "NPS 系统 build 已通过",
		ResetContext: true,
	})
	if !got.Switched {
		t.Fatal("expected role switch")
	}
	if got.CurrentRole != WorkspaceRoleQAEngineer {
		t.Fatalf("current role=%s want %s", got.CurrentRole, WorkspaceRoleQAEngineer)
	}
	if got.ContextPolicy == "" {
		t.Fatal("expected context policy")
	}
	if !strings.Contains(got.Reason, "符合推荐流转") || !strings.Contains(got.Reason, "build 成功") {
		t.Fatalf("expected recommended transition reason, got %q", got.Reason)
	}
	if !containsWorkspaceRoleString(got.RequiredDocs, "/system/prompt/roles/qa-engineer") {
		t.Fatalf("required docs should include qa engineer doc: %v", got.RequiredDocs)
	}
}

func TestBuildChangeRoleReportsBaseReadOnlyToolsForEveryRole(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		TargetRole: WorkspaceRolePlatformEngineer,
		UserInput:  "全链路测试 openapi",
	})
	for _, tool := range []string{"read_dir", "read_go_file", "read_go_file_lines", "read_app_log", "search_tools"} {
		if !containsWorkspaceRoleString(got.AllowedNextTools, tool) {
			t.Fatalf("platform engineer should report base read-only tool %s, tools=%v", tool, got.AllowedNextTools)
		}
	}
	if containsWorkspaceRoleString(got.AllowedNextTools, "write_go_file") {
		t.Fatalf("platform engineer should not report write_go_file, tools=%v", got.AllowedNextTools)
	}
}

func TestBuildChangeRoleLoadsWorkflowEngineerDocs(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		TargetRole: WorkspaceRoleWorkflowEngineer,
		UserInput:  "把多个 Form 串成工作流",
		Directory:  "/u/app/automation",
	})
	if got.CurrentRole != WorkspaceRoleWorkflowEngineer {
		t.Fatalf("current role=%s want %s", got.CurrentRole, WorkspaceRoleWorkflowEngineer)
	}
	if got.DisplayName != "工作流编排工程师" {
		t.Fatalf("display name=%s want 工作流编排工程师", got.DisplayName)
	}
	if !containsWorkspaceRoleString(got.RequiredDocs, "/system/prompt/roles/workflow-engineer") {
		t.Fatalf("required docs should include workflow engineer role doc: %v", got.RequiredDocs)
	}
	if !containsWorkspaceRoleString(got.RequiredDocs, "/system/prompt/case_catalog/workflow") {
		t.Fatalf("required docs should include workflow case catalog index: %v", got.RequiredDocs)
	}
	if !containsWorkspaceRoleString(got.AllowedNextTools, "search_tools") ||
		!containsWorkspaceRoleString(got.AllowedNextTools, "write_doc") ||
		!containsWorkspaceRoleString(got.AllowedNextTools, "create_workflow") {
		t.Fatalf("workflow_engineer should expose orchestration tools, tools=%v", got.AllowedNextTools)
	}
}

func TestChangeRoleRejectsRetiredOrMalformedRoleID(t *testing.T) {
	res := (&ChangeRoleTool{}).Execute(context.Background(), ToolCall{
		Args: map[string]interface{}{
			"target_role": "app_operate_test",
		},
	})
	if !res.IsError {
		t.Fatalf("expected invalid target_role to fail, got %#v", res)
	}
	if !strings.Contains(res.Content, WorkspaceRoleQAEngineer) {
		t.Fatalf("error should list canonical role ids, got %q", res.Content)
	}
}
