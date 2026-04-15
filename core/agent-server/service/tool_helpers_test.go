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
