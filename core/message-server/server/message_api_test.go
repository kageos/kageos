package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/pkg/auth"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/gin-gonic/gin"
)

func TestResolveRequestSenderFromToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	deptPath := "/org/engineering"
	token, err := auth.NewJWTService().GenerateAccessTokenWithHR(1, "alice", "alice@example.com", deptPath, "")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/message/api/v1/send", nil)
	req.Header.Set(contextx.TokenHeader, token)
	c.Request = req

	ctx, sender, err := resolveRequestSender(c)
	if err != nil {
		t.Fatalf("resolve sender: %v", err)
	}
	if sender != "alice" {
		t.Fatalf("sender = %q, want alice", sender)
	}
	if got := c.Request.Header.Get(contextx.RequestUserHeader); got != "alice" {
		t.Fatalf("%s = %q, want alice", contextx.RequestUserHeader, got)
	}
	if got := contextx.GetRequestDepartmentFullPath(ctx); got != deptPath {
		t.Fatalf("department path = %q, want %q", got, deptPath)
	}
}

func TestResolveRequestSenderRequiresToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/message/api/v1/send", nil)
	req.Header.Set(contextx.RequestUserHeader, "spoofed-user")
	c.Request = req

	if _, _, err := resolveRequestSender(c); err == nil {
		t.Fatal("expected error when token is missing")
	}
}
