package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/agent-server/service"
	"github.com/kageos/kageos/dto"
)

func TestBatchToolDetailsReturnsLightweightMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/workspace/tools/batch_detail", NewWorkspace(service.NewToolRegistry(), nil).BatchToolDetails)

	body := bytes.NewBufferString(`{
		"names": ["<tool:send_notification>", "tool:send_notification", "missing_tool"],
		"include_schema": false
	}`)
	req := httptest.NewRequest(http.MethodPost, "/workspace/tools/batch_detail", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code int                               `json:"code"`
		Data dto.BatchWorkspaceToolDetailsResp `json:"data"`
		Msg  string                            `json:"msg"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; body = %s", err, w.Body.String())
	}
	if resp.Code != 0 {
		t.Fatalf("response code = %d msg = %q", resp.Code, resp.Msg)
	}
	if len(resp.Data.Tools) != 1 {
		t.Fatalf("tools len = %d, want 1; tools = %#v", len(resp.Data.Tools), resp.Data.Tools)
	}
	tool := resp.Data.Tools[0]
	if tool.Name != "send_notification" || tool.Token != "<tool:send_notification>" || tool.TypeLabel != "内置工具" {
		t.Fatalf("tool metadata mismatch: %#v", tool)
	}
	if tool.Description == "" {
		t.Fatalf("tool description should not be empty")
	}
	if len(tool.InputFields) == 0 {
		t.Fatalf("tool input fields should be summarized")
	}
	if tool.InputSchema != nil {
		t.Fatalf("input schema should be omitted by default")
	}
	if len(resp.Data.Missing) != 1 || resp.Data.Missing[0] != "missing_tool" {
		t.Fatalf("missing = %#v, want [missing_tool]", resp.Data.Missing)
	}
}

func TestBatchToolDetailsCanIncludeSchema(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/workspace/tools/batch_detail", NewWorkspace(service.NewToolRegistry(), nil).BatchToolDetails)

	body := bytes.NewBufferString(`{"names":["search"],"include_schema":true}`)
	req := httptest.NewRequest(http.MethodPost, "/workspace/tools/batch_detail", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code int                               `json:"code"`
		Data dto.BatchWorkspaceToolDetailsResp `json:"data"`
		Msg  string                            `json:"msg"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; body = %s", err, w.Body.String())
	}
	if resp.Code != 0 {
		t.Fatalf("response code = %d msg = %q", resp.Code, resp.Msg)
	}
	if len(resp.Data.Tools) != 1 {
		t.Fatalf("tools len = %d, want 1", len(resp.Data.Tools))
	}
	if len(resp.Data.Tools[0].InputSchema) == 0 {
		t.Fatalf("input schema should be included when include_schema=true")
	}
}
