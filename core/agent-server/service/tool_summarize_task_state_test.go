package service

import (
	"strings"
	"testing"
)

func TestBuildTaskStateSummaryCompactsState(t *testing.T) {
	got := buildTaskStateSummary(summarizeTaskStateArgs{
		Intent:       "app.create",
		Directory:    "/u/app/nps",
		Outcome:      "已生成 NPS 系统并 build 通过",
		ChangedFiles: []string{"nps_questionnaire_list.go", "nps_submit.go"},
		Verified:     []string{"build_workspace", "run_form_submit"},
		NextIntent:   "app.operate_test",
		NextGoal:     "验证核心函数",
	})
	if !strings.Contains(got.Summary, "身份=app.create") {
		t.Fatalf("summary should include role, got %q", got.Summary)
	}
	if !strings.Contains(got.Summary, "下一身份=app.operate_test") {
		t.Fatalf("summary should include next role, got %q", got.Summary)
	}
}
