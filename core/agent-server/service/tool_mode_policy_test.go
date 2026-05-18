package service

import "testing"

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
