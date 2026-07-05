package service

import (
	"strings"
	"testing"

	"github.com/kageos/kageos/dto"
)

func TestApplyLineEditsToContentChecksExpectedOldText(t *testing.T) {
	source := strings.Join([]string{
		"package demo",
		"",
		"func run() {",
		"\tprintln(\"old\")",
		"}",
		"",
	}, "\n")

	got, edits, err := applyLineEditsToContent(source, []editFileLineEditArgs{{
		StartLine:       4,
		EndLine:         4,
		ExpectedOldText: "\tprintln(\"old\")",
		Replacement:     "\tprintln(\"new\")",
	}})
	if err != nil {
		t.Fatalf("applyLineEditsToContent returned error: %v", err)
	}
	if edits != 1 {
		t.Fatalf("applied edits = %d, want 1", edits)
	}
	if !strings.Contains(got, `println("new")`) {
		t.Fatalf("line edit did not apply:\n%s", got)
	}

	_, _, err = applyLineEditsToContent(source, []editFileLineEditArgs{{
		StartLine:       4,
		EndLine:         4,
		ExpectedOldText: "\tprintln(\"different\")",
		Replacement:     "\tprintln(\"new\")",
	}})
	if err == nil || !strings.Contains(err.Error(), "expected_old_text") {
		t.Fatalf("expected old text mismatch, got %v", err)
	}
}

func TestApplyLineEditsToContentRejectsOverlappingRanges(t *testing.T) {
	source := strings.Join([]string{
		"package demo",
		"",
		"func run() {",
		"\tprintln(\"old\")",
		"}",
		"",
	}, "\n")

	_, _, err := applyLineEditsToContent(source, []editFileLineEditArgs{
		{StartLine: 3, EndLine: 4, Replacement: "\tprintln(\"new\")"},
		{StartLine: 4, EndLine: 5, Replacement: ""},
	})
	if err == nil || !strings.Contains(err.Error(), "重叠") {
		t.Fatalf("expected overlapping range error, got %v", err)
	}
}

func TestApplySearchEditsToContentPreservesWhitespace(t *testing.T) {
	source := strings.Join([]string{
		"package demo",
		"",
		"func run() {",
		"\tprintln(\"old\")",
		"}",
		"",
	}, "\n")

	got, edits, err := applySearchEditsToContent(source, []editFileSearchEditArgs{{
		OldText: "\tprintln(\"old\")",
		NewText: "\tprintln(\"new\")",
	}})
	if err != nil {
		t.Fatalf("applySearchEditsToContent returned error: %v", err)
	}
	if edits != 1 {
		t.Fatalf("applied edits = %d, want 1", edits)
	}
	if !strings.Contains(got, "\tprintln(\"new\")") {
		t.Fatalf("search edit did not preserve tab indentation:\n%s", got)
	}
}

func TestApplySearchEditsToContentChecksExpectedCount(t *testing.T) {
	source := "one\none\n"

	_, _, err := applySearchEditsToContent(source, []editFileSearchEditArgs{{
		OldText:       "one",
		NewText:       "two",
		ExpectedCount: 1,
	}})
	if err == nil || !strings.Contains(err.Error(), "匹配次数") {
		t.Fatalf("expected count mismatch error, got %v", err)
	}
}

func TestApplySearchEditsToContentValidatesSequentially(t *testing.T) {
	source := "alpha\n"

	got, edits, err := applySearchEditsToContent(source, []editFileSearchEditArgs{
		{OldText: "alpha", NewText: "beta"},
		{OldText: "beta", NewText: "gamma"},
	})
	if err != nil {
		t.Fatalf("applySearchEditsToContent returned error: %v", err)
	}
	if edits != 2 || got != "gamma\n" {
		t.Fatalf("got edits=%d content=%q, want 2 and gamma", edits, got)
	}
}

func TestIsGeneratedInitGoFile(t *testing.T) {
	for _, name := range []string{"init_", "init_.go", "dir/init_.go"} {
		if !isGeneratedInitGoFile(name) {
			t.Fatalf("isGeneratedInitGoFile(%q) = false, want true", name)
		}
	}
	if isGeneratedInitGoFile("customer.go") {
		t.Fatal("customer.go should not be treated as generated init file")
	}
}

func TestNormalizeWriteFileNameUsesExplicitFileType(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		fileType string
		want     string
	}{
		{name: "go default", fileName: "handler", want: "handler.go"},
		{name: "json type", fileName: "config", fileType: "json", want: "config.json"},
		{name: "extension wins", fileName: "template.md", fileType: "json", want: "template.md"},
		{name: "dotted type", fileName: "settings", fileType: ".yaml", want: "settings.yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeWriteFileName(tt.fileName, tt.fileType); got != tt.want {
				t.Fatalf("normalizeWriteFileName(%q, %q) = %q, want %q", tt.fileName, tt.fileType, got, tt.want)
			}
		})
	}
}

func TestBlockingGoWriteDiagnosticsBlocksSyntaxError(t *testing.T) {
	source := `package demo

type PurchaseInbound struct{}

// TableName 获取表名
	return "purchase_inbound"
}
`

	msg := blockingGoWriteDiagnostics("/system/demo", "purchase_inbound.go", source)
	if msg == "" {
		t.Fatal("expected blocking go syntax diagnostics")
	}
	if !strings.Contains(msg, "go_syntax") || !strings.Contains(msg, "expected declaration") {
		t.Fatalf("unexpected diagnostics: %q", msg)
	}
}

func TestBlockingGoWriteDiagnosticsAllowsWarnings(t *testing.T) {
	source := `package demo

import "fmt"

func run() {}
`

	msg := blockingGoWriteDiagnostics("/system/demo", "demo.go", source)
	if msg != "" {
		t.Fatalf("warnings should not block writes, got %q", msg)
	}
}

func TestBuildReadFileResultDataIncludesSHAAndNumberedContent(t *testing.T) {
	file := &dto.WorkspaceContextFile{
		FileName:      "demo",
		RelativePath:  "demo.go",
		FileType:      "go",
		Content:       "package demo\n\nfunc run() {}\n",
		ContentLength: len("package demo\n\nfunc run() {}\n"),
		LineCount:     3,
	}
	data := buildReadFileResultData("/system/demo/app", file, "1-2")
	if data.ContentSHA == "" || !strings.HasPrefix(data.ContentSHA, "sha256:") {
		t.Fatalf("content sha missing: %#v", data)
	}
	if data.Content != "package demo\n" {
		t.Fatalf("content = %q, want first two lines", data.Content)
	}
	if !strings.Contains(data.NumberedContent, "1 | package demo") {
		t.Fatalf("numbered content missing line numbers: %q", data.NumberedContent)
	}
}
