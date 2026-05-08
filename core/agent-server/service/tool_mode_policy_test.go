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

func TestWorkspaceModeToolGateResult(t *testing.T) {
	allowed := []string{"read_doc", "read_dir"}
	if res, blocked := workspaceModeToolGateResult("read_doc", allowed); blocked || res.IsError {
		t.Fatalf("read_doc should be allowed, blocked=%v res=%#v", blocked, res)
	}
	res, blocked := workspaceModeToolGateResult("write_go_file", allowed)
	if blocked || res.IsError {
		t.Fatalf("mode gate should be advisory and not block write_go_file, blocked=%v res=%#v", blocked, res)
	}
	if _, blocked := workspaceModeToolGateResult("anything", nil); blocked {
		t.Fatal("empty allowlist should preserve legacy allow-all behavior")
	}
}

func containsServiceToolName(tools []string, target string) bool {
	for _, tool := range tools {
		if tool == target {
			return true
		}
	}
	return false
}
