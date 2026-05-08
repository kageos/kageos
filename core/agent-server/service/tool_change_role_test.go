package service

import (
	"context"
	"testing"
)

func TestBuildChangeRoleLoadsPlanDocs(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		TargetRole: "app.plan",
		UserInput:  "帮我搞个 NPS 管理系统",
		Directory:  "/u/app/nps",
	})
	if got.CurrentRole != "app.plan" {
		t.Fatalf("current role=%s want app.plan", got.CurrentRole)
	}
	if !containsIntentString(got.RequiredDocs, "/system/prompt/intents/app-plan") {
		t.Fatalf("required docs should include plan intent doc: %v", got.RequiredDocs)
	}
	if !containsIntentString(got.RequiredDocs, "/system/prompt/case_catalog") {
		t.Fatalf("app.plan should include case catalog index: %v", got.RequiredDocs)
	}
	for _, heavyCase := range []string{
		"/system/prompt/case_catalog/formandtable/vote",
		"/system/prompt/case_catalog/form_table_chart/cashier",
		"/system/prompt/case_catalog/table/ticket",
	} {
		if containsIntentString(got.RequiredDocs, heavyCase) {
			t.Fatalf("app.plan should not auto-inject full case %s before PRD confirmation: %v", heavyCase, got.RequiredDocs)
		}
	}
	for _, removed := range []string{
		"/system/prompt/sdk/widget-reference",
		"/system/prompt/sdk/build-validation-reference",
		"/system/prompt/sdk/workbench-tools-sdk-relationship",
		"/system/prompt/platform-overview",
		"/system/prompt/platform-capability-boundaries",
	} {
		if containsIntentString(got.RequiredDocs, removed) {
			t.Fatalf("app.plan should not inject heavy doc %s: %v", removed, got.RequiredDocs)
		}
	}
	if len(got.LoadedDocs) == 0 {
		t.Fatal("expected loaded docs")
	}
}

func TestBuildChangeRoleLoadsCreateDocs(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		TargetRole: "app.create",
		UserInput:  "已确认 PRD，开始创建目录和生成代码",
		Directory:  "/u/app/nps",
	})
	if got.CurrentRole != "app.create" {
		t.Fatalf("current role=%s want app.create", got.CurrentRole)
	}
	if !containsIntentString(got.RequiredDocs, "/system/prompt/intents/app-create") {
		t.Fatalf("required docs should include create intent doc: %v", got.RequiredDocs)
	}
	if !containsIntentString(got.RequiredDocs, "/system/prompt/sdk/agent-app-sdk-readme") {
		t.Fatalf("required docs should include SDK readme: %v", got.RequiredDocs)
	}
	if containsIntentString(got.AllowedNextTools, "write_prd") {
		t.Fatalf("app.create should not plan PRD again, tools=%v", got.AllowedNextTools)
	}
}

func TestBuildChangeRoleDoesNotInferFromUserInput(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		UserInput: "帮我搞个 NPS 管理系统",
	})
	if got.CurrentRole != "app.explain_review" {
		t.Fatalf("current role=%s want app.explain_review", got.CurrentRole)
	}
	if containsIntentString(got.RequiredDocs, "/system/prompt/intents/app-plan") ||
		containsIntentString(got.RequiredDocs, "/system/prompt/intents/app-create") {
		t.Fatalf("change_role should not infer app plan/create from user_input: %v", got.RequiredDocs)
	}
}

func TestBuildChangeRoleKeepsCurrentWhenTargetMissing(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		CurrentRole: "app.create",
		UserInput:   "build_workspace 报错",
	})
	if got.CurrentRole != "app.create" {
		t.Fatalf("current role=%s want app.create", got.CurrentRole)
	}
}

func TestBuildChangeRoleSwitchesAndCarriesSummary(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		CurrentRole:  "app.create",
		TargetRole:   "app.operate_test",
		TaskSummary:  "NPS 系统 build 已通过",
		ResetContext: true,
	})
	if !got.Switched {
		t.Fatal("expected role switch")
	}
	if got.CurrentRole != "app.operate_test" {
		t.Fatalf("current role=%s want app.operate_test", got.CurrentRole)
	}
	if got.ContextPolicy == "" {
		t.Fatal("expected context policy")
	}
	if !containsIntentString(got.RequiredDocs, "/system/prompt/intents/app-operate-test") {
		t.Fatalf("required docs should include operate test doc: %v", got.RequiredDocs)
	}
}
