package service

import (
	"strings"
	"testing"
)

func TestNormalizeAddFunctionsGoFileNameRejectsTestFiles(t *testing.T) {
	for _, raw := range []string{"demo_test.go", "demo_test", "demo_test.go.go"} {
		t.Run(raw, func(t *testing.T) {
			_, err := normalizeAddFunctionsGoFileName(raw, "fallback")
			if err == nil {
				t.Fatalf("expected _test.go file_name to fail")
			}
			if !strings.Contains(err.Error(), "_test.go") || !strings.Contains(err.Error(), "API 注册") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNormalizeAddFunctionsGoFileNameUsesFallbackAndTrimsGoSuffix(t *testing.T) {
	got, err := normalizeAddFunctionsGoFileName("demo.go.go", "fallback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "demo" {
		t.Fatalf("got %q, want demo", got)
	}

	got, err = normalizeAddFunctionsGoFileName("", "fallback.go")
	if err != nil {
		t.Fatalf("unexpected fallback error: %v", err)
	}
	if got != "fallback" {
		t.Fatalf("got %q, want fallback", got)
	}
}
