package streamloop

import "testing"

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
