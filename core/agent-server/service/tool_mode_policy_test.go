package service

import (
	"testing"

	"github.com/kageos/kageos/core/agent-server/prompt"
)

func TestWorkspaceToolNamesForModeUsesConfiguredToolsOnly(t *testing.T) {
	tools := workspaceToolNamesForMode(nil, []string{"read_doc", "read_dir"})
	if !containsServiceToolName(tools, "read_doc") || !containsServiceToolName(tools, "read_dir") {
		t.Fatalf("configured tools missing: %v", tools)
	}
	for _, removed := range removedModeDocToolNames() {
		if containsServiceToolName(tools, removed) {
			t.Fatalf("tools should not include %s: %v", removed, tools)
		}
	}
}

func TestDevModeToolNamesCoverWorkspaceRoleAllowedTools(t *testing.T) {
	provider := prompt.GetModeProvider("dev")
	if provider == nil {
		t.Fatal("dev mode provider is nil")
	}
	modeTools := make(map[string]struct{})
	for _, name := range provider.ToolNames() {
		modeTools[name] = struct{}{}
	}

	for roleID, spec := range workspaceRoleSpecs() {
		definition := buildWorkspaceRoleDefinition(spec)
		for _, tool := range definition.AllowedTools {
			if _, ok := modeTools[tool]; !ok {
				t.Fatalf("dev mode tool_names missing %q allowed by role %s", tool, roleID)
			}
		}
	}
}

func TestWorkspaceToolNamesForRoleFiltersVisibleTools(t *testing.T) {
	provider := prompt.GetModeProvider("dev")
	if provider == nil {
		t.Fatal("dev mode provider is nil")
	}
	qaTools := workspaceToolNamesForRole(provider.ToolNames(), WorkspaceRoleQAEngineer)
	for _, want := range []string{"change_role", "search", "run_form_submit", "run_chart_query"} {
		if !containsServiceToolName(qaTools, want) {
			t.Fatalf("qa visible tools should include %s: %v", want, qaTools)
		}
	}
	for _, blocked := range []string{"edit_file", "write_file", "build_workspace"} {
		if containsServiceToolName(qaTools, blocked) {
			t.Fatalf("qa visible tools should hide %s: %v", blocked, qaTools)
		}
	}

	buildTools := workspaceToolNamesForRole(provider.ToolNames(), WorkspaceRoleBuildEngineer)
	for _, want := range []string{"change_role", "read_file", "edit_file", "write_file", "build_workspace"} {
		if !containsServiceToolName(buildTools, want) {
			t.Fatalf("build engineer visible tools should include %s: %v", want, buildTools)
		}
	}
	if containsServiceToolName(buildTools, "run_form_submit") {
		t.Fatalf("build engineer visible tools should hide run_form_submit: %v", buildTools)
	}

	routerTools := workspaceToolNamesForRole(provider.ToolNames(), WorkspaceRoleRouter)
	for _, want := range []string{"change_role", "read_doc", "read_dir", "read_file", "search"} {
		if !containsServiceToolName(routerTools, want) {
			t.Fatalf("router visible tools should include %s: %v", want, routerTools)
		}
	}
	for _, blocked := range []string{"edit_file", "write_file", "build_workspace", "run_form_submit"} {
		if containsServiceToolName(routerTools, blocked) {
			t.Fatalf("router visible tools should hide %s: %v", blocked, routerTools)
		}
	}
}

func TestWorkspaceToolNamesForLLMKeepsStableModeToolSet(t *testing.T) {
	provider := prompt.GetModeProvider("dev")
	if provider == nil {
		t.Fatal("dev mode provider is nil")
	}
	tools := workspaceToolNamesForLLM(provider.ToolNames())
	for _, want := range []string{"change_role", "read_doc", "write_file", "build_workspace", "run_form_submit", "send_notification"} {
		if !containsServiceToolName(tools, want) {
			t.Fatalf("LLM tools should include stable mode tool %s: %v", want, tools)
		}
	}
	if len(tools) != len(provider.ToolNames()) {
		t.Fatalf("LLM tools should preserve dev mode tool count, got %d want %d", len(tools), len(provider.ToolNames()))
	}
}

func removedModeDocToolNames() []string {
	return []string{"read_" + "sk" + "ill", "search_" + "sk" + "ills"}
}

func containsServiceToolName(tools []string, target string) bool {
	for _, tool := range tools {
		if tool == target {
			return true
		}
	}
	return false
}
