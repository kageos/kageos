package service

import (
	"strings"
	"testing"
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
	if !strings.Contains(msg, "创建失败：字段缺失") {
		t.Fatalf("expected original error message, got %q", msg)
	}
	if !strings.Contains(msg, "`read_doc(\""+executeGuideDocPath+"\")`") {
		t.Fatalf("expected guide hint, got %q", msg)
	}

	plain := appendExecuteGuideHint("read_doc", "读取失败")
	if plain != "读取失败" {
		t.Fatalf("expected non execute tool error unchanged, got %q", plain)
	}
}
