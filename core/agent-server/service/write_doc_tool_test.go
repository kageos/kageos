package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kageos/kageos/dto"
)

func TestRunWriteDocToolUsesDirectoryAsParentPathWhenCreatingDoc(t *testing.T) {
	const (
		directory = "/liubeiluo/work/ticket_system"
		docCode   = "ticket_analysis_report"
		docName   = "工单管理系统数据分析报告"
	)

	var createReq dto.CreateDocsReq
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/health":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == "/workspace/api/v1/service_tree/detail":
			fullCodePath := r.URL.Query().Get("full_code_path")
			switch fullCodePath {
			case directory + "/" + docCode:
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

	content, isError := runWriteDocTool(context.Background(), writeDocArgs{
		Directory: directory,
		Name:      docName,
		Code:      docCode,
		Content:   "# report",
	}, "")
	if isError {
		t.Fatalf("runWriteDocTool returned error: %s", content)
	}
	if createReq.ParentFullCodePath != directory {
		t.Fatalf("ParentFullCodePath = %q, want %q", createReq.ParentFullCodePath, directory)
	}
	if createReq.Code != docCode {
		t.Fatalf("Code = %q, want %q", createReq.Code, docCode)
	}
	if createReq.Name != docName {
		t.Fatalf("Name = %q, want %q", createReq.Name, docName)
	}
}

func TestRunCreateDirectoryCommandRejectsInvalidDirectoryCode(t *testing.T) {
	t.Parallel()

	content, isError := runCreateDirectoryCommand(context.Background(), createDirectoryCommand{
		Directory: "/liubeiluo/work",
		Name:      "用户中心",
		Code:      "user-center",
	}, "")
	if !isError {
		t.Fatalf("expected invalid directory code to be rejected, got content=%s", content)
	}
	if !strings.Contains(content, "目录英文标识不符合要求") {
		t.Fatalf("unexpected error content: %s", content)
	}
}

func writeToolAPIResponse(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode api response: %v", err)
	}
}
