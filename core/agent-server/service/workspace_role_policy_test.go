package service

import "testing"

func TestWorkspaceRoleAliasesResolveToCanonicalRoles(t *testing.T) {
	cases := map[string]string{
		"product-manager":      WorkspaceRoleProductManager,
		"app-developer":        WorkspaceRoleAppDeveloper,
		"qa-engineer":          WorkspaceRoleQAEngineer,
		"build-engineer":       WorkspaceRoleBuildEngineer,
		"maintenance-engineer": WorkspaceRoleMaintenanceEngineer,
		"data-operator":        WorkspaceRoleDataOperator,
		"scheduler-engineer":   WorkspaceRoleSchedulerEngineer,
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
	if res, blocked := workspaceRoleToolGateResult(WorkspaceRoleQAEngineer, "write_go_file"); !blocked || !res.IsError {
		t.Fatalf("qa_engineer should block write_go_file, blocked=%v res=%#v", blocked, res)
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
