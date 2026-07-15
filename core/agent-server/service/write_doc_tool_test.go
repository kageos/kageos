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
	var detailLookups []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/health":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == "/workspace/api/v1/service_tree/detail":
			fullCodePath := r.URL.Query().Get("full_code_path")
			detailLookups = append(detailLookups, fullCodePath)
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
	if len(detailLookups) == 0 || detailLookups[0] != directory+"/"+docCode+".docs" {
		t.Fatalf("first detail lookup = %#v, want canonical .docs path first", detailLookups)
	}
}

func TestRunWriteDocToolUpdatesExistingDocWithCanonicalDocsSuffix(t *testing.T) {
	const (
		directory = "/liubeiluo/work/ticket_system"
		docCode   = "ticket_analysis_report"
		docName   = "工单管理系统数据分析报告"
		docID     = int64(2001)
	)

	var updateReq dto.UpdateDocsReq
	var createCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/health":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == "/workspace/api/v1/service_tree/detail":
			fullCodePath := r.URL.Query().Get("full_code_path")
			if fullCodePath != directory+"/"+docCode+".docs" {
				t.Fatalf("unexpected detail lookup: %s", fullCodePath)
			}
			writeToolAPIResponse(t, w, map[string]interface{}{
				"code": 0,
				"msg":  "ok",
				"data": dto.GetServiceTreeDetailResp{
					ID:           docID,
					Name:         docName,
					Code:         docCode + ".docs",
					Type:         "docs",
					FullCodePath: directory + "/" + docCode + ".docs",
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
		case r.Method == http.MethodPost && r.URL.Path == "/workspace/api/v1/docs/crud":
			createCalled = true
			t.Fatalf("write_doc should update existing docs node instead of creating")
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
		Content:   "# updated report",
		Format:    "markdown",
	}, "")
	if isError {
		t.Fatalf("runWriteDocTool returned error: %s", content)
	}
	if createCalled {
		t.Fatal("create should not be called")
	}
	if updateReq.Content == nil || *updateReq.Content != "# updated report" {
		t.Fatalf("Content = %#v, want updated report", updateReq.Content)
	}
	if updateReq.Format == nil || *updateReq.Format != "markdown" {
		t.Fatalf("Format = %#v, want markdown", updateReq.Format)
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
