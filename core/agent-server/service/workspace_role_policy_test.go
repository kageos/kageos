package service

import (
	"strings"
	"testing"
)

func TestWorkspaceRoleAliasesResolveToCanonicalRoles(t *testing.T) {
	cases := map[string]string{
		"product-manager":      WorkspaceRoleProductManager,
		"app-developer":        WorkspaceRoleAppDeveloper,
		"qa-engineer":          WorkspaceRoleQAEngineer,
		"app-operator":         WorkspaceRoleAppOperator,
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

func TestWorkspaceRoleToolGateBlocksWrongRoleTools(t *testing.T) {
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleProductManager, "write_prd"); blocked || res.IsError {
		t.Fatalf("product_manager should allow write_prd, blocked=%v res=%#v", blocked, res)
	}
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleProductManager, "write_go_file"); !blocked || !res.IsError {
		t.Fatalf("product_manager should block write_go_file, blocked=%v res=%#v", blocked, res)
	}
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleAppDeveloper, "write_go_file"); blocked || res.IsError {
		t.Fatalf("app_developer should allow write_go_file, blocked=%v res=%#v", blocked, res)
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
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleAppOperator, "write_go_file"); !blocked || !res.IsError {
		t.Fatalf("app_operator should block write_go_file, blocked=%v res=%#v", blocked, res)
	}
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleQAEngineer, "write_go_file"); !blocked || !res.IsError {
		t.Fatalf("qa_engineer should block write_go_file, blocked=%v res=%#v", blocked, res)
	}
}

func TestWorkspaceRoleToolGateAllowsReadOnlyToolsAcrossRoles(t *testing.T) {
	cases := []struct {
		role string
		tool string
	}{
		{WorkspaceRoleProductManager, "read_go_file"},
		{WorkspaceRoleQAEngineer, "read_go_file"},
		{WorkspaceRolePlatformEngineer, "read_dir"},
		{WorkspaceRolePlatformEngineer, "read_go_file_lines"},
		{WorkspaceRoleDataOperator, "read_app_log"},
		{WorkspaceRoleReviewer, "search_tools"},
		{WorkspaceRoleReviewer, "search_resources"},
		{"", "read_go_file"},
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
	if res, blocked := workspaceRoleToolGateResult("", "write_go_file"); !blocked || !res.IsError {
		t.Fatalf("empty role should block write_go_file, blocked=%v res=%#v", blocked, res)
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

func TestWorkspaceToolScopeGateBlocksPathsOutsideExecuteDirectory(t *testing.T) {
	res, blocked := workspaceToolScopeGateResult(WorkspaceRoleQAEngineer, "run_table_search", map[string]interface{}{
		"full_code_path": "/system/x_world/ticket/ticket_list.table",
	}, "/system/x_world/vote")
	if !blocked || !res.IsError {
		t.Fatalf("expected scope gate to block sibling app path, blocked=%v res=%#v", blocked, res)
	}
	if !strings.Contains(res.Content, "execute_directory /system/x_world/vote") {
		t.Fatalf("scope error should mention execute directory, got %q", res.Content)
	}

	res, blocked = workspaceToolScopeGateResult(WorkspaceRoleQAEngineer, "run_table_search", map[string]interface{}{
		"full_code_path": "/system/x_world/vote/vote_topic_list.table",
	}, "/system/x_world/vote")
	if blocked || res.IsError {
		t.Fatalf("expected scope gate to allow path inside execute directory, blocked=%v res=%#v", blocked, res)
	}
}

func TestWorkspaceToolScopeGateAllowsPromptDocsOutsideExecuteDirectory(t *testing.T) {
	res, blocked := workspaceToolScopeGateResult(WorkspaceRoleAppDeveloper, "read_doc", map[string]interface{}{
		"directory": "/system/prompt/roles/app-developer,/system/prompt/sdk/agent-app-sdk-readme",
	}, "/system/x_world/vote")
	if blocked || res.IsError {
		t.Fatalf("expected prompt docs to bypass workspace scope, blocked=%v res=%#v", blocked, res)
	}
}

func TestWorkspaceToolScopeGateRequiresScopedSearchForOperatorAndQA(t *testing.T) {
	res, blocked := workspaceToolScopeGateResult(WorkspaceRoleAppOperator, "search_tools", map[string]interface{}{
		"keyword": "投票",
	}, "/system/x_world/vote")
	if !blocked || !res.IsError {
		t.Fatalf("expected app_operator search_tools without directory to fail, blocked=%v res=%#v", blocked, res)
	}

	res, blocked = workspaceToolScopeGateResult(WorkspaceRoleAppOperator, "search_tools", map[string]interface{}{
		"directory": "/system/x_world/vote",
	}, "/system/x_world/vote")
	if blocked || res.IsError {
		t.Fatalf("expected app_operator scoped search_tools to pass, blocked=%v res=%#v", blocked, res)
	}

	res, blocked = workspaceToolScopeGateResult(WorkspaceRoleQAEngineer, "search_resources", map[string]interface{}{
		"scope": "current_app",
	}, "/system/x_world/vote")
	if blocked || res.IsError {
		t.Fatalf("expected qa current_app search_resources to pass, blocked=%v res=%#v", blocked, res)
	}
}
