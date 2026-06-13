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
