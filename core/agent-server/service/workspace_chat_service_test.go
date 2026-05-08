package service

import (
	"context"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
)

func TestShouldSuggestExecuteGuide(t *testing.T) {
	if !shouldSuggestExecuteGuide("run_form_submit") {
		t.Fatal("expected run_form_submit to suggest execute guide")
	}
	if !shouldSuggestExecuteGuide("run_table_search") {
		t.Fatal("expected run_table_search to suggest execute guide")
	}
	if !shouldSuggestExecuteGuide("run_table_batch_create") {
		t.Fatal("expected run_table_batch_create to suggest execute guide")
	}
	if !shouldSuggestExecuteGuide("run_table_delete") {
		t.Fatal("expected run_table_delete to suggest execute guide")
	}
	if shouldSuggestExecuteGuide("read_doc") {
		t.Fatal("did not expect read_doc to suggest execute guide")
	}
}

func TestAppendExecuteGuideHint(t *testing.T) {
	msg := appendExecuteGuideHint("run_table_create", "创建失败：字段缺失")
	if msg != "创建失败：字段缺失" {
		t.Fatalf("expected original error unchanged, got %q", msg)
	}

	plain := appendExecuteGuideHint("read_doc", "读取失败")
	if plain != "读取失败" {
		t.Fatalf("expected non execute tool error unchanged, got %q", plain)
	}
}

func TestGuideDocPathsFromReadDocArgsSupportsCommaSeparatedPaths(t *testing.T) {
	paths := guideDocPathsFromReadDocArgs(map[string]interface{}{
		"directory": "/system/prompt/platform-capability-boundaries,/system/prompt/sdk/agent-app-sdk-readme/",
	})
	loaded := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		loaded[path] = struct{}{}
	}

	if _, ok := loaded["/system/prompt/platform-capability-boundaries"]; !ok {
		t.Fatalf("expected platform capability boundaries to be marked loaded: %#v", loaded)
	}
	if _, ok := loaded["/system/prompt/sdk/agent-app-sdk-readme"]; !ok {
		t.Fatalf("expected sdk readme to be marked loaded: %#v", loaded)
	}
}

func TestWithAgentToolExecutionContextMarksSourceAndSession(t *testing.T) {
	base := contextx.WithClientSource(context.Background(), "browser")

	ctx := withAgentToolExecutionContext(base, "session-1")

	if got := contextx.GetClientSource(ctx); got != "agent" {
		t.Fatalf("client source = %q, want agent", got)
	}
	if got := getWorkspaceSessionID(ctx); got != "session-1" {
		t.Fatalf("session id = %q, want session-1", got)
	}
}
