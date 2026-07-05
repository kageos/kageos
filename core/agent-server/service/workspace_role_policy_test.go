package service

import (
	"strings"
	"testing"

	"github.com/kageos/kageos/core/agent-server/model"
)

func TestWorkspaceRoleAliasesResolveToCanonicalRoles(t *testing.T) {
	cases := map[string]string{
		"product-manager":      WorkspaceRoleProductManager,
		"app-developer":        WorkspaceRoleAppDeveloper,
		"qa-engineer":          WorkspaceRoleQAEngineer,
		"app-operator":         WorkspaceRoleAppOperator,
		"automation-operator":  WorkspaceRoleAutomationOperator,
		"build-engineer":       WorkspaceRoleBuildEngineer,
		"maintenance-engineer": WorkspaceRoleMaintenanceEngineer,
		"data-operator":        WorkspaceRoleDataOperator,
		"platform-engineer":    WorkspaceRolePlatformEngineer,
	}
	for input, want := range cases {
		if got := normalizeWorkspaceRole(input); got != want {
			t.Fatalf("normalizeWorkspaceRole(%q)=%q want %q", input, got, want)
		}
	}
}

func TestWorkspaceRoleDoesNotAcceptRetiredRoleIDs(t *testing.T) {
	for _, input := range []string{"app.plan", "app.create", "app.operate_test", "app_operate_test"} {
		if isKnownWorkspaceRole(input) {
			t.Fatalf("retired role id %q should not be a known workspace role", input)
		}
	}
}

func TestApplyDefaultWorkspaceSessionRoleUsesAppOperator(t *testing.T) {
	session := &model.AgentChatSession{}

	applyDefaultWorkspaceSessionRole(session)

	if session.RoleID != WorkspaceRoleAppOperator || session.RoleDisplayName != "应用执行" {
		t.Fatalf("default role = %q/%q, want app_operator/应用执行", session.RoleID, session.RoleDisplayName)
	}
}

func TestApplyDefaultWorkspaceSessionRolePreservesExistingRole(t *testing.T) {
	session := &model.AgentChatSession{
		RoleID:          WorkspaceRoleQAEngineer,
		RoleDisplayName: "测试工程师",
	}

	applyDefaultWorkspaceSessionRole(session)

	if session.RoleID != WorkspaceRoleQAEngineer || session.RoleDisplayName != "测试工程师" {
		t.Fatalf("existing role should be preserved, got %#v", session)
	}

	handoffSession := &model.AgentChatSession{HandoffTargetRole: WorkspaceRoleAppDeveloper}
	applyDefaultWorkspaceSessionRole(handoffSession)
	if handoffSession.RoleID != "" || handoffSession.HandoffTargetRole != WorkspaceRoleAppDeveloper {
		t.Fatalf("handoff target role should be preserved without injecting default, got %#v", handoffSession)
	}
}

func TestWorkspaceRoleToolGateBlocksWrongRoleTools(t *testing.T) {
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleProductManager, "write_prd"); blocked || res.IsError {
		t.Fatalf("product_manager should allow write_prd, blocked=%v res=%#v", blocked, res)
	}
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleProductManager, "write_file"); !blocked || !res.IsError {
		t.Fatalf("product_manager should block write_file, blocked=%v res=%#v", blocked, res)
	}
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleAppDeveloper, "write_file"); blocked || res.IsError {
		t.Fatalf("app_developer should allow write_file, blocked=%v res=%#v", blocked, res)
	}
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleAppDeveloper, "edit_file"); blocked || res.IsError {
		t.Fatalf("app_developer should allow edit_file, blocked=%v res=%#v", blocked, res)
	}
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleAppDeveloper, "write_doc"); !blocked || !res.IsError {
		t.Fatalf("app_developer should block default-hidden write_doc, blocked=%v res=%#v", blocked, res)
	}
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleMaintenanceEngineer, "write_doc"); !blocked || !res.IsError {
		t.Fatalf("maintenance_engineer should block default-hidden write_doc, blocked=%v res=%#v", blocked, res)
	}
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleAppDeveloper, "write_prd"); !blocked || !res.IsError {
		t.Fatalf("app_developer should block write_prd, blocked=%v res=%#v", blocked, res)
	}
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleQAEngineer, "run_form_submit"); blocked || res.IsError {
		t.Fatalf("qa_engineer should allow run_form_submit, blocked=%v res=%#v", blocked, res)
	}
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleAppOperator, "run_table_create"); blocked || res.IsError {
		t.Fatalf("app_operator should allow run_table_create, blocked=%v res=%#v", blocked, res)
	}
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleAppOperator, "run_python"); blocked || res.IsError {
		t.Fatalf("app_operator should allow run_python, blocked=%v res=%#v", blocked, res)
	}
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleAppOperator, "list_scheduled_tasks"); blocked || res.IsError {
		t.Fatalf("app_operator should allow read-only scheduled task listing, blocked=%v res=%#v", blocked, res)
	}
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleAppOperator, "list_scheduled_task_executions"); blocked || res.IsError {
		t.Fatalf("app_operator should allow read-only scheduled execution listing, blocked=%v res=%#v", blocked, res)
	}
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleAppOperator, "write_file"); !blocked || !res.IsError {
		t.Fatalf("app_operator should block write_file, blocked=%v res=%#v", blocked, res)
	}
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleAppOperator, "write_doc"); !blocked || !res.IsError {
		t.Fatalf("app_operator should block write_doc, blocked=%v res=%#v", blocked, res)
	}
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleAutomationOperator, "create_scheduled_function_task"); blocked || res.IsError {
		t.Fatalf("automation_operator should allow create_scheduled_function_task, blocked=%v res=%#v", blocked, res)
	}
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleAutomationOperator, "run_form_submit"); !blocked || !res.IsError {
		t.Fatalf("automation_operator should block direct run_form_submit, blocked=%v res=%#v", blocked, res)
	}
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleAppOperator, "create_scheduled_function_task"); !blocked || !res.IsError {
		t.Fatalf("app_operator should block scheduled task creation, blocked=%v res=%#v", blocked, res)
	}
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleQAEngineer, "write_file"); !blocked || !res.IsError {
		t.Fatalf("qa_engineer should block write_file, blocked=%v res=%#v", blocked, res)
	}
}

func TestWorkspaceRoleToolGateRunPythonGuidanceIsActionable(t *testing.T) {
	res, blocked := workspaceRoleToolGateResult(WorkspaceRoleMaintenanceEngineer, "run_python")
	if !blocked || !res.IsError {
		t.Fatalf("maintenance_engineer should block run_python, blocked=%v res=%#v", blocked, res)
	}
	for _, want := range []string{
		"应用执行",
		"数据/文件处理工程师",
		"def kageos_entry(args, output_dir):",
		"read_file/search/read_doc",
	} {
		if !strings.Contains(res.Content, want) {
			t.Fatalf("run_python gate guidance should contain %q, got: %s", want, res.Content)
		}
	}
}

func TestWorkspaceRoleToolGateAllowsReadOnlyToolsAcrossRoles(t *testing.T) {
	cases := []struct {
		role string
		tool string
	}{
		{WorkspaceRoleProductManager, "read_file"},
		{WorkspaceRoleQAEngineer, "read_file"},
		{WorkspaceRolePlatformEngineer, "read_dir"},
		{WorkspaceRolePlatformEngineer, "read_file"},
		{WorkspaceRoleDataOperator, "read_app_log"},
		{WorkspaceRoleReviewer, "search"},
		{WorkspaceRoleProductManager, "web_search"},
		{WorkspaceRoleAppDeveloper, "web_search"},
		{WorkspaceRoleAppOperator, "web_search"},
		{WorkspaceRoleAutomationOperator, "web_search"},
		{WorkspaceRoleReviewer, "web_search"},
		{WorkspaceRoleProductManager, "list_scheduled_tasks"},
		{WorkspaceRoleQAEngineer, "list_scheduled_task_executions"},
		{"", "read_file"},
	}
	for _, tc := range cases {
		if res, blocked := workspaceRoleToolGateResult(tc.role, tc.tool); blocked || res.IsError {
			t.Fatalf("%s should allow read-only tool %s, blocked=%v res=%#v", tc.role, tc.tool, blocked, res)
		}
	}
}

func TestWorkspaceRoleToolGateRequiresRoleBeforeMutations(t *testing.T) {
	if res, blocked := workspaceRoleToolGateResult("", "change_role"); blocked || res.IsError {
		t.Fatalf("empty role should allow change_role, blocked=%v res=%#v", blocked, res)
	}
	if res, blocked := workspaceRoleToolGateResult("", "write_file"); !blocked || !res.IsError {
		t.Fatalf("empty role should block write_file, blocked=%v res=%#v", blocked, res)
	}
}

func TestWorkspaceRoleHookDeclarationsAreRegisteredOrPlanned(t *testing.T) {
	for roleID, spec := range workspaceRoleSpecs() {
		for _, hook := range spec.Runtime.Hooks {
			status := workspaceRoleHookImplementationStatus(hook.ID)
			switch status {
			case workspaceRoleHookImplementationImplemented, workspaceRoleHookImplementationPlanned:
			default:
				t.Fatalf("role %s hook %s should be registered or explicitly planned, status=%q", roleID, hook.ID, status)
			}
		}
	}
}
