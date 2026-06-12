package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
)

func TestResolveRequestMessageMetaFromHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/message/api/v1/send", nil)
	req.Header.Set(contextx.RequestUserHeader, "alice")
	req.Header.Set(contextx.DepartmentFullPathHeader, "/org/engineering")
	req.Header.Set(contextx.TraceIdHeader, "trace-1")
	req.Header.Set(contextx.ClientSourceHeader, "browser")
	c.Request = req

	ctx, meta, err := resolveRequestMessageMeta(c, &dto.MessageSendMeta{FullCodePath: "/system/demo/send.form"})
	if err != nil {
		t.Fatalf("resolve meta: %v", err)
	}
	if meta.From != "alice" || meta.RequestUser != "alice" {
		t.Fatalf("sender meta = %#v, want alice", meta)
	}
	if got := contextx.GetRequestDepartmentFullPath(ctx); got != "/org/engineering" {
		t.Fatalf("department = %q", got)
	}
	if meta.FullCodePath != "/system/demo/send.form" {
		t.Fatalf("full_code_path = %q", meta.FullCodePath)
	}
}

func TestResolveRequestMessageMetaRequiresSender(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/message/api/v1/send", nil)

	if _, _, err := resolveRequestMessageMeta(c, nil); err == nil {
		t.Fatal("expected error when sender is missing")
	}
}
