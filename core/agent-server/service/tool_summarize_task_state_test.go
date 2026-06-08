package service

import (
	"strings"
	"testing"
)

func TestBuildTaskStateSummaryCompactsState(t *testing.T) {
	got := buildTaskStateSummary(summarizeTaskStateArgs{
		RoleID:              WorkspaceRoleAppDeveloper,
		Directory:           "/u/app/nps",
		Outcome:             "已生成 NPS 系统并 build 通过",
		UserGoal:            "门店经理快速提交 NPS 回访",
		ConfirmedScope:      []string{"提交评分 Form", "满意度记录只读表", "NPS 趋势图"},
		KeyDecisions:        []string{"记录表由 Form 产生，不开放手工新增"},
		Constraints:         []string{"创建开始时间/创建结束时间映射系统创建时间"},
		NonGoals:            []string{"暂不做复杂审批"},
		ChangedFiles:        []string{"nps_questionnaire_list.go", "nps_submit.go"},
		Verified:            []string{"build_workspace", "run_form_submit"},
		ImplementationNotes: []string{"Form 必须写入满意度记录表"},
		VerificationNotes:   []string{"验证门店筛选和日期趋势"},
		ArtifactRefs:        []string{"/liubeiluo/nps/submit.form"},
		ReferenceDocs:       []string{"/system/prompt/roles/qa-engineer", "/system/prompt/sdk/agent-app-sdk-readme"},
		ReferenceFiles:      []string{"nps_submit.go", "nps_chart.go"},
		NextRoleID:          WorkspaceRoleQAEngineer,
		NextGoal:            "验证核心函数",
	})
	if !strings.Contains(got.Summary, "角色=app_developer") {
		t.Fatalf("summary should include role, got %q", got.Summary)
	}
	if !strings.Contains(got.Summary, "用户目标=门店经理快速提交 NPS 回访") {
		t.Fatalf("summary should include user goal, got %q", got.Summary)
	}
	if !strings.Contains(got.Summary, "关键决策=记录表由 Form 产生") {
		t.Fatalf("summary should include key decisions, got %q", got.Summary)
	}
	if !strings.Contains(got.Summary, "不做=暂不做复杂审批") {
		t.Fatalf("summary should include non-goals, got %q", got.Summary)
	}
	if !strings.Contains(got.Summary, "下一角色=qa_engineer") {
		t.Fatalf("summary should include next role, got %q", got.Summary)
	}
	if !strings.Contains(got.Summary, "参考文档=/system/prompt/roles/qa-engineer") || !strings.Contains(got.Summary, "参考文件=nps_submit.go") {
		t.Fatalf("summary should include references, got %q", got.Summary)
	}
	if len(got.ConfirmedScope) != 3 || len(got.ImplementationNotes) != 1 || len(got.ArtifactRefs) != 1 || len(got.ReferenceDocs) != 2 || len(got.ReferenceFiles) != 2 {
		t.Fatalf("structured summary fields not preserved: %#v", got)
	}
	if got.Handoff.ExecuteDirectory != "/u/app/nps" {
		t.Fatalf("handoff should include execute directory, got %#v", got.Handoff)
	}
	if !strings.Contains(strings.Join(got.Handoff.TaskContext, "；"), "门店经理快速提交 NPS 回访") || !strings.Contains(strings.Join(got.Handoff.TaskContext, "；"), "暂不做复杂审批") {
		t.Fatalf("handoff task context should preserve user goal and special cases, got %#v", got.Handoff)
	}
	if !strings.Contains(strings.Join(got.Handoff.KeyInformation, "；"), "nps_submit.go") || !strings.Contains(strings.Join(got.Handoff.KeyInformation, "；"), "run_form_submit") {
		t.Fatalf("handoff key info should preserve changed files and verification, got %#v", got.Handoff)
	}
	if !containsWorkspaceRoleString(got.Handoff.References, "/system/prompt/roles/qa-engineer") || !containsWorkspaceRoleString(got.Handoff.References, "nps_chart.go") {
		t.Fatalf("handoff references should preserve docs and files, got %#v", got.Handoff)
	}
}

func TestBuildTaskStateSummaryNarrowsExecuteDirectoryFromArtifactRefs(t *testing.T) {
	got := buildTaskStateSummary(summarizeTaskStateArgs{
		RoleID:         WorkspaceRoleAppDeveloper,
		Directory:      "/system/x_world",
		Outcome:        "工单管理系统已生成并 build 通过",
		ChangedFiles:   []string{"/system/x_world/ticket_management/ticket.go"},
		ArtifactRefs:   []string{"/system/x_world/ticket_management/ticket_list.table"},
		ReferenceFiles: []string{"/system/x_world/ticket_management"},
		NextRoleID:     WorkspaceRoleQAEngineer,
		NextGoal:       "验证工单列表、筛选和 CRUD",
	})
	if got.Directory != "/system/x_world/ticket_management" || got.Handoff.ExecuteDirectory != "/system/x_world/ticket_management" {
		t.Fatalf("execute directory should narrow to target app dir, got directory=%q handoff=%#v", got.Directory, got.Handoff)
	}
	keyInfo := strings.Join(got.Handoff.KeyInformation, "；")
	if !strings.Contains(keyInfo, "工作区构建/来源目录：/system/x_world") ||
		!strings.Contains(keyInfo, "下一阶段目标应用目录：/system/x_world/ticket_management") {
		t.Fatalf("handoff should preserve source and target directories, got %#v", got.Handoff)
	}
}
