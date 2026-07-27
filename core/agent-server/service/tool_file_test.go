package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
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

func TestRunEditFileToolUpdatesDocsNameAndRemovesResolvedEntry(t *testing.T) {
	const (
		directory = "/liubeiluo/work/ticket_system"
		fileName  = "important_issues.docs"
		docName   = "重要问题"
		docID     = int64(2001)
	)
	source := "# 重要问题\n\n## 支付回调重复\n仍需排查。\n\n## 已解决：历史导入失败\n已于昨天修复。\n"
	resolvedEntry := "\n## 已解决：历史导入失败\n已于昨天修复。\n"

	var updateReq dto.UpdateDocsReq
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/health":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == "/workspace/api/v1/docs/info/liubeiluo/work/ticket_system/important_issues.docs":
			writeToolAPIResponse(t, w, map[string]interface{}{
				"code": 0,
				"msg":  "ok",
				"data": dto.DocItem{
					ID:           docID,
					Name:         "Important Issues",
					Content:      source,
					Format:       "markdown",
					FullCodePath: directory + "/" + fileName,
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/workspace/api/v1/service_tree/detail":
			if got := r.URL.Query().Get("full_code_path"); got != directory+"/"+fileName {
				t.Fatalf("full_code_path = %q, want %q", got, directory+"/"+fileName)
			}
			writeToolAPIResponse(t, w, map[string]interface{}{
				"code": 0,
				"msg":  "ok",
				"data": dto.GetServiceTreeDetailResp{
					ID:           docID,
					Name:         "Important Issues",
					Code:         fileName,
					Type:         "docs",
					FullCodePath: directory + "/" + fileName,
				},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/workspace/api/v1/docs/crud/2001":
			if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
				t.Fatalf("decode update docs request: %v", err)
			}
			writeToolAPIResponse(t, w, map[string]interface{}{
				"code": 0,
				"msg":  "ok",
				"data": map[string]interface{}{},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	t.Setenv("GATEWAY_URL", server.URL)

	ctx := contextx.WithWorkspaceSession(context.Background(), "session-1", "无人值守巡检", WorkspaceRoleAppOperator)
	result := runEditFileTool(ctx, editFileArgs{
		Directory: directory,
		FileName:  fileName,
		Name:      docName,
		BaseSHA:   fileContentSHA(source),
		SearchEdits: []editFileSearchEditArgs{{
			OldText: resolvedEntry,
			NewText: "",
		}},
	}, "")
	if result.IsError {
		t.Fatalf("runEditFileTool returned error: %s", result.Content)
	}
	if updateReq.Name == nil || *updateReq.Name != docName {
		t.Fatalf("Name = %#v, want %q", updateReq.Name, docName)
	}
	if updateReq.Content == nil {
		t.Fatal("Content should be updated")
	}
	if strings.Contains(*updateReq.Content, "已解决：历史导入失败") {
		t.Fatalf("resolved entry should be deleted, content=%q", *updateReq.Content)
	}
	if !strings.Contains(*updateReq.Content, "支付回调重复") {
		t.Fatalf("unresolved entry should remain, content=%q", *updateReq.Content)
	}
}

func TestRunEditFileToolAppOperatorRejectsGoCode(t *testing.T) {
	ctx := contextx.WithWorkspaceSession(context.Background(), "session-1", "无人值守巡检", WorkspaceRoleAppOperator)
	result := runEditFileTool(ctx, editFileArgs{
		Directory: "/liubeiluo/work/ticket_system",
		FileName:  "main.go",
		BaseSHA:   "sha256:any",
		SearchEdits: []editFileSearchEditArgs{{
			OldText: "old",
			NewText: "new",
		}},
	}, "")
	if !result.IsError || !strings.Contains(result.Content, "只允许修改 .docs 文档") {
		t.Fatalf("app_operator should be blocked from editing Go code, result=%#v", result)
	}
}

func TestRunWriteFileToolCreatesDocsWithEnglishCodeAndChineseName(t *testing.T) {
	const (
		directory = "/liubeiluo/work/ticket_system"
		docCode   = "important_issues"
		docName   = "重要问题"
	)

	var createReq dto.CreateDocsReq
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/health":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == "/workspace/api/v1/docs/info/liubeiluo/work/ticket_system/important_issues.docs":
			writeToolAPIResponse(t, w, map[string]interface{}{
				"code": 7,
				"msg":  "record not found",
				"data": nil,
			})
		case r.Method == http.MethodGet && r.URL.Path == "/workspace/api/v1/service_tree/detail":
			fullCodePath := r.URL.Query().Get("full_code_path")
			switch fullCodePath {
			case directory + "/" + docCode + ".docs", directory + "/" + docCode:
				writeToolAPIResponse(t, w, map[string]interface{}{
					"code": 7,
					"msg":  "record not found",
					"data": nil,
				})
			case directory:
				writeToolAPIResponse(t, w, map[string]interface{}{
					"code": 0,
					"msg":  "ok",
					"data": dto.GetServiceTreeDetailResp{
						ID:           1001,
						Type:         "package",
						FullCodePath: directory,
					},
				})
			default:
				t.Fatalf("unexpected detail lookup: %s", fullCodePath)
			}
		case r.Method == http.MethodPost && r.URL.Path == "/workspace/api/v1/docs/crud":
			if err := json.NewDecoder(r.Body).Decode(&createReq); err != nil {
				t.Fatalf("decode create docs request: %v", err)
			}
			writeToolAPIResponse(t, w, map[string]interface{}{
				"code": 0,
				"msg":  "ok",
				"data": dto.CreateDocsResp{
					ID:           2001,
					Name:         docName,
					Code:         docCode + ".docs",
					Type:         "docs",
					FullCodePath: directory + "/" + docCode + ".docs",
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	t.Setenv("GATEWAY_URL", server.URL)

	ctx := contextx.WithWorkspaceSession(context.Background(), "session-1", "无人值守巡检", WorkspaceRoleAppOperator)
	result := runWriteFileTool(ctx, writeFileArgs{
		Directory: directory,
		FileName:  docCode + ".docs",
		Name:      docName,
		Content:   "# 重要问题\n\n## 支付回调重复\n仍需排查。\n",
	}, "")
	if result.IsError {
		t.Fatalf("runWriteFileTool returned error: %s", result.Content)
	}
	if createReq.Code != docCode {
		t.Fatalf("Code = %q, want %q", createReq.Code, docCode)
	}
	if createReq.Name != docName {
		t.Fatalf("Name = %q, want %q", createReq.Name, docName)
	}
}
