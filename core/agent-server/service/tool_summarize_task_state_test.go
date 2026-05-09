package service

import (
	"strings"
	"testing"
)

func TestBuildTaskStateSummaryCompactsState(t *testing.T) {
	got := buildTaskStateSummary(summarizeTaskStateArgs{
		RoleID:       WorkspaceRoleAppDeveloper,
		Directory:    "/u/app/nps",
		Outcome:      "已生成 NPS 系统并 build 通过",
		ChangedFiles: []string{"nps_questionnaire_list.go", "nps_submit.go"},
		Verified:     []string{"build_workspace", "run_form_submit"},
		NextRoleID:   WorkspaceRoleQAEngineer,
		NextGoal:     "验证核心函数",
	})
	if !strings.Contains(got.Summary, "角色=app_developer") {
		t.Fatalf("summary should include role, got %q", got.Summary)
	}
	if !strings.Contains(got.Summary, "下一角色=qa_engineer") {
		t.Fatalf("summary should include next role, got %q", got.Summary)
	}
}
