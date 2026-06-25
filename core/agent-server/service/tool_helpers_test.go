package service

import (
	"strings"
	"testing"
)

func TestResolveTypedFunctionFullCodePathArg(t *testing.T) {
	t.Run("accepts matching suffix", func(t *testing.T) {
		got, notice := resolveTypedFunctionFullCodePathArg("/u/app/demo.form", "", ".form")
		if notice != "" {
			t.Fatalf("unexpected notice: %s", notice)
		}
		if got != "/u/app/demo.form" {
			t.Fatalf("unexpected path: %s", got)
		}
	})

	t.Run("auto appends missing suffix", func(t *testing.T) {
		got, notice := resolveTypedFunctionFullCodePathArg("/u/app/demo", "", ".chart")
		if got != "/u/app/demo.chart" {
			t.Fatalf("unexpected path: %s", got)
		}
		if notice == "" || !containsAll(notice, "/u/app/demo", "/u/app/demo.chart", "已自动改为") {
			t.Fatalf("unexpected notice: %s", notice)
		}
	})

	t.Run("auto corrects wrong suffix", func(t *testing.T) {
		got, notice := resolveTypedFunctionFullCodePathArg("/u/app/demo.form", "", ".table")
		if got != "/u/app/demo.table" {
			t.Fatalf("unexpected path: %s", got)
		}
		if notice == "" || !containsAll(notice, ".form", ".table", "/u/app/demo.table") {
			t.Fatalf("unexpected notice: %s", notice)
		}
	})

	t.Run("resolves current package relative form path", func(t *testing.T) {
		got, notice := resolveTypedFunctionFullCodePathArg("./sweep.form", "/system/democase/gold_watch", ".form")
		if notice != "" {
			t.Fatalf("unexpected notice: %s", notice)
		}
		if got != "/system/democase/gold_watch/sweep.form" {
			t.Fatalf("unexpected path: %s", got)
		}
	})

	t.Run("resolves relative table path from function default", func(t *testing.T) {
		got, notice := resolveTypedFunctionFullCodePathArg("./signals.table", "/system/democase/gold_watch/sweep.form", ".table")
		if notice != "" {
			t.Fatalf("unexpected notice: %s", notice)
		}
		if got != "/system/democase/gold_watch/signals.table" {
			t.Fatalf("unexpected path: %s", got)
		}
	})

	t.Run("preserves query on relative path", func(t *testing.T) {
		got, notice := resolveTypedFunctionFullCodePathArg("./snapshots.table?page=1&page_size=20", "/system/democase/gold_watch", ".table")
		if notice != "" {
			t.Fatalf("unexpected notice: %s", notice)
		}
		if got != "/system/democase/gold_watch/snapshots.table?page=1&page_size=20" {
			t.Fatalf("unexpected path: %s", got)
		}
	})
}

func TestToolResultWithStructuredData(t *testing.T) {
	t.Run("stores raw structured data for frontend consumption", func(t *testing.T) {
		payload := map[string]interface{}{
			"output_files": "kageos/workspace/output/clip.mp4",
		}

		result := toolResultWithStructuredData(payload, false)

		if result.IsError {
			t.Fatal("expected success result")
		}
		if !containsAll(result.Content, "output_files", "kageos/workspace/output/clip.mp4") {
			t.Fatalf("unexpected content: %s", result.Content)
		}
		got, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		if _, ok := got["output_files"]; !ok {
			t.Fatalf("expected output_files in data: %#v", got)
		}
	})

	t.Run("keeps notice in content without polluting structured data", func(t *testing.T) {
		payload := map[string]interface{}{"status": "ok"}
		result := toolResultWithStructuredData(payload, false, "注意：已自动修正路径")

		if !strings.HasPrefix(result.Content, "注意：已自动修正路径\n\n") {
			t.Fatalf("unexpected content prefix: %s", result.Content)
		}
		got, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map data, got %T", result.Data)
		}
		if len(got) != 1 || got["status"] != "ok" {
			t.Fatalf("unexpected structured data: %#v", got)
		}
	})
}

func containsAll(s string, parts ...string) bool {
	for _, part := range parts {
		if part == "" {
			continue
		}
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}
