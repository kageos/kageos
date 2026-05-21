package streamloop

import (
	"testing"

	"github.com/kageos/kageos/pkg/llms"
)

func TestAppendToolCallArgsPreservesStringSpaceAcrossChunks(t *testing.T) {
	got := appendToolCallArgs(`{"deadline":"2026-04-25 `, `18:00:00"}`)
	want := `{"deadline":"2026-04-25 18:00:00"}`
	if got != want {
		t.Fatalf("appendToolCallArgs() = %q, want %q", got, want)
	}
}

func TestAppendToolCallArgsIgnoresDeltaWhenCurrentIsValidJSON(t *testing.T) {
	got := appendToolCallArgs(`{"id":1}`, `,"extra":2`)
	want := `{"id":1}`
	if got != want {
		t.Fatalf("appendToolCallArgs() = %q, want %q", got, want)
	}
}

func TestMergeToolCallsUsesOpenAIStreamIndex(t *testing.T) {
	idx0 := 0
	idx1 := 1
	makeCall := func(index *int, id, name, args string) llms.ToolCall {
		tc := llms.ToolCall{ID: id, Type: "function", Index: index}
		tc.Function.Name = name
		tc.Function.Arguments = args
		return tc
	}

	var all []llms.ToolCall
	indexByID := map[string]int{}
	all, indexByID = mergeToolCalls([]llms.ToolCall{
		makeCall(&idx0, "call_a", "first_tool", `{"a":`),
		makeCall(&idx1, "call_b", "second_tool", `{"b":`),
	}, all, indexByID)
	all, indexByID = mergeToolCalls([]llms.ToolCall{
		makeCall(&idx0, "", "", `1}`),
		makeCall(&idx1, "", "", `2}`),
	}, all, indexByID)

	if len(all) != 2 {
		t.Fatalf("len(all) = %d, want 2", len(all))
	}
	if all[0].ID != "call_a" || all[0].Function.Name != "first_tool" || all[0].Function.Arguments != `{"a":1}` {
		t.Fatalf("first tool call not merged by index: %#v", all[0])
	}
	if all[1].ID != "call_b" || all[1].Function.Name != "second_tool" || all[1].Function.Arguments != `{"b":2}` {
		t.Fatalf("second tool call not merged by index: %#v", all[1])
	}
	if indexByID["call_a"] != 0 || indexByID["call_b"] != 1 {
		t.Fatalf("indexByID = %#v, want call_a=0 call_b=1", indexByID)
	}
}
