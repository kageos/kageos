package service

import (
	"testing"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/prompt"
)

func TestWorkspaceToolNamesForQAMode(t *testing.T) {
	provider := prompt.GetModeProvider("qa")
	if provider == nil {
		t.Fatal("expected qa mode provider")
	}

	tools := workspaceToolNamesForMode(provider, nil)
	if !containsServiceToolName(tools, "read_doc") {
		t.Fatalf("qa mode missing read_doc: %v", tools)
	}
	for _, expected := range []string{"search_skills", "read_skill"} {
		if !containsServiceToolName(tools, expected) {
			t.Fatalf("qa mode should expose %s: %v", expected, tools)
		}
	}
	for _, blocked := range []string{"write_go_file", "build_workspace", "run_form_submit", "run_table_create"} {
		if containsServiceToolName(tools, blocked) {
			t.Fatalf("qa mode should not expose %s: %v", blocked, tools)
		}
	}
}

func TestWorkspaceToolNamesForModeAppendsSkills(t *testing.T) {
	provider := prompt.GetModeProvider("qa")
	if provider == nil {
		t.Fatal("expected qa mode provider")
	}

	tools := workspaceToolNamesForMode(provider, nil)
	for _, expected := range []string{"search_skills", "read_skill"} {
		if !containsServiceToolName(tools, expected) {
			t.Fatalf("skills mode should expose %s: %v", expected, tools)
		}
	}
}

func TestWorkspaceModeToolGateResult(t *testing.T) {
	allowed := []string{"read_doc", "read_dir"}
	if res, blocked := workspaceModeToolGateResult("read_doc", allowed); blocked || res.IsError {
		t.Fatalf("read_doc should be allowed, blocked=%v res=%#v", blocked, res)
	}
	res, blocked := workspaceModeToolGateResult("write_go_file", allowed)
	if !blocked || !res.IsError {
		t.Fatalf("write_go_file should be blocked, blocked=%v res=%#v", blocked, res)
	}
	if res.Content == "" {
		t.Fatal("blocked result should include content")
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
