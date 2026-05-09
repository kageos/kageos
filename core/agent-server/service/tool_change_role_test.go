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
	if !containsWorkspaceRoleString(got.RequiredDocs, "/system/prompt/roles/qa-engineer") {
		t.Fatalf("required docs should include qa engineer doc: %v", got.RequiredDocs)
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
