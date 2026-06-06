package service

import (
	"context"
	"testing"
)

func TestWorkspaceRoleDefinitionsExposeCompleteProtocol(t *testing.T) {
	for roleID := range workspaceRoleSpecs() {
		definition, ok := workspaceRoleDefinitionFor(roleID)
		if !ok {
			t.Fatalf("role %s should expose role definition", roleID)
		}
		if definition.ProtocolVersion != workspaceRoleDefinitionProtocolVersion {
			t.Fatalf("role %s protocol version=%q", roleID, definition.ProtocolVersion)
		}
		if definition.ID != roleID || definition.DisplayName == "" || definition.Responsibility == "" {
			t.Fatalf("role %s definition identity/responsibility incomplete: %#v", roleID, definition)
		}
		if len(definition.DocumentPackage) == 0 {
			t.Fatalf("role %s should expose document package", roleID)
		}
		if len(definition.AllowedTools) == 0 {
			t.Fatalf("role %s should expose allowed tools", roleID)
		}
		for _, field := range []string{"execute_directory", "task_context", "key_information", "references"} {
			if !containsWorkspaceRoleString(definition.HandoffRequired, field) {
				t.Fatalf("role %s handoff required should include %s: %#v", roleID, field, definition.HandoffRequired)
			}
		}
		if len(definition.RuntimeContract.SOP) == 0 || len(definition.RuntimeContract.DoneWhen) == 0 {
			t.Fatalf("role %s should expose SOP and done_when: %#v", roleID, definition.RuntimeContract)
		}
		for _, next := range definition.AllowedTransitions {
			if !isKnownWorkspaceRole(next.RoleID) {
				t.Fatalf("role %s has unknown transition target %#v", roleID, next)
			}
			if next.When == "" {
				t.Fatalf("role %s transition to %s should explain when", roleID, next.RoleID)
			}
		}
	}
}

func TestWorkspaceRoleDefinitionBuildEngineerIncludesRepairDocs(t *testing.T) {
	definition, ok := workspaceRoleDefinitionFor(WorkspaceRoleBuildEngineer)
	if !ok {
		t.Fatal("build engineer definition missing")
	}
	for _, doc := range []string{
		"/system/prompt/roles/build-engineer",
		"/system/prompt/sdk/agent-app-sdk-readme",
		"/system/prompt/sdk/reference/build-validation",
	} {
		if !containsWorkspaceRoleString(definition.DocumentPackage, doc) {
			t.Fatalf("build engineer document package should include %s, got %#v", doc, definition.DocumentPackage)
		}
	}
	if !containsWorkspaceRoleString(definition.AllowedTools, "build_workspace") ||
		!containsWorkspaceRoleString(definition.ForbiddenTools, "run_form_submit") {
		t.Fatalf("build engineer tool policy incomplete: allowed=%#v forbidden=%#v", definition.AllowedTools, definition.ForbiddenTools)
	}
}

func TestBuildChangeRoleDerivesLegacyFieldsFromRoleDefinition(t *testing.T) {
	got := buildChangeRole(context.Background(), changeRoleArgs{
		TargetRole:       WorkspaceRoleAppDeveloper,
		ExecuteDirectory: "/system/x_world",
		TaskContext:      []string{"PRD 已确认，进入开发"},
	}, "/system/x_world")
	if got.RoleDefinition.ID != WorkspaceRoleAppDeveloper {
		t.Fatalf("change_role should return role definition, got %#v", got.RoleDefinition)
	}
	if got.RuntimeContract.DoneWhen == nil || len(got.RoleDefinition.RuntimeContract.DoneWhen) == 0 {
		t.Fatalf("role definition should carry runtime contract: %#v", got.RoleDefinition.RuntimeContract)
	}
	if len(got.AllowedNextTools) != len(got.RoleDefinition.AllowedTools) {
		t.Fatalf("allowed_next_tools should derive from role_definition: legacy=%#v definition=%#v", got.AllowedNextTools, got.RoleDefinition.AllowedTools)
	}
	for _, doc := range got.RequiredDocs {
		if !containsWorkspaceRoleString(got.RoleDefinition.DocumentPackage, doc) {
			t.Fatalf("required doc %s should be in role definition document package %#v", doc, got.RoleDefinition.DocumentPackage)
		}
	}
	if got.NextAction != got.RoleDefinition.DefaultNextAction {
		t.Fatalf("next action should derive from role definition")
	}
}
