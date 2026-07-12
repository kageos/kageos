package contextx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/nats-io/nats.go"
)

func TestTrustedIdentityHeadersClearAndApply(t *testing.T) {
	header := make(http.Header)
	for _, name := range TrustedIdentityHeaderNames() {
		header.Add(name, "forged")
		header.Add(name, "second-forged-value")
	}
	header["x-request-user"] = []string{"non-canonical-forged"}
	header.Set(TokenHeader, "credential-must-survive")
	header.Set(TraceIdHeader, "trace-must-survive")

	captured := CaptureTrustedIdentityHeaders(header)
	if got := captured[RequestUserHeader]; got != "forged" {
		t.Fatalf("captured request user = %q, want forged", got)
	}

	ClearTrustedIdentityHeaders(header)
	for _, name := range TrustedIdentityHeaderNames() {
		if values := header.Values(name); len(values) != 0 {
			t.Fatalf("trusted header %s survived clear: %#v", name, values)
		}
	}
	if _, exists := header["x-request-user"]; exists {
		t.Fatal("non-canonical trusted header survived clear")
	}
	if got := header.Get(TokenHeader); got != "credential-must-survive" {
		t.Fatalf("token after clear = %q", got)
	}
	if got := header.Get(TraceIdHeader); got != "trace-must-survive" {
		t.Fatalf("trace after clear = %q", got)
	}

	ApplyTrustedIdentityHeaders(header, map[string]string{
		RequestUserHeader:        "alice",
		WorkspaceRoleHeader:      "app_developer",
		DepartmentFullPathHeader: " /org/engineering ",
		"X-Not-Trusted":          "must-not-be-applied",
	})
	if got := header.Get(RequestUserHeader); got != "alice" {
		t.Fatalf("request user after apply = %q", got)
	}
	if got := header.Get(DepartmentFullPathHeader); got != "/org/engineering" {
		t.Fatalf("department after apply = %q", got)
	}
	if got := header.Get(WorkspaceRoleHeader); got != "app_developer" {
		t.Fatalf("workspace role after apply = %q", got)
	}
	if got := header.Get("X-Not-Trusted"); got != "" {
		t.Fatalf("unknown identity header was applied: %q", got)
	}
}

func TestToContextPreservesClientSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/demo", nil)
	c.Request.Header.Set(ClientSourceHeader, "browser")

	ctx := ToContext(c)

	if got := GetClientSource(ctx); got != "browser" {
		t.Fatalf("GetClientSource(ctx) = %q, want browser", got)
	}
	if got := c.GetHeader(ClientSourceHeader); got != "browser" {
		t.Fatalf("gin header = %q, want browser", got)
	}
}

func TestToContextCopiesOnlyPrivateDelegationFromRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	parent := context.WithValue(context.Background(), RequestUserHeader, "forged-parent-user")
	called := false
	parent = controlauth.WithDelegatedHTTPRequestSigner(parent, func(_ *http.Request, _ []byte) (bool, error) {
		called = true
		return true, nil
	})
	c.Request = httptest.NewRequest(http.MethodGet, "/demo", nil).WithContext(parent)

	ctx := ToContext(c)
	if got := GetRequestUser(ctx); got != "" {
		t.Fatalf("parent string-key identity leaked into clean context: %q", got)
	}
	outbound := httptest.NewRequest(http.MethodGet, "http://gateway.internal/workspace", nil).WithContext(ctx)
	signed, err := controlauth.ApplyDelegatedHTTPRequestSignature(outbound, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !signed || !called {
		t.Fatal("private delegation capability was not propagated")
	}
}

func TestNatsTraceContextPreservesClientSource(t *testing.T) {
	msg := nats.NewMsg("demo")
	msg.Header.Set(ClientSourceHeader, "agent")

	ctx := NatsTraceContext(msg)

	if got := GetClientSource(ctx); got != "agent" {
		t.Fatalf("GetClientSource(ctx) = %q, want agent", got)
	}
}

func TestCtxToTraceNatsPreservesClientSource(t *testing.T) {
	ctx := WithClientSource(context.Background(), "agent")
	ctx = WithInitiatorUser(ctx, "bob")
	ctx = WithWorkspaceMessageID(ctx, 42)
	ctx = WithToolCallInfo(ctx, "call-1", "run_table_add")

	msg := CtxToTraceNats(ctx, "demo")

	if got := msg.Header.Get(ClientSourceHeader); got != "agent" {
		t.Fatalf("nats header = %q, want agent", got)
	}
	if got := msg.Header.Get(InitiatorUserHeader); got != "bob" {
		t.Fatalf("initiator header = %q, want bob", got)
	}
	if got := msg.Header.Get(WorkspaceMessageIDHeader); got != "42" {
		t.Fatalf("workspace message header = %q, want 42", got)
	}
	if got := msg.Header.Get(ToolCallIDHeader); got != "call-1" {
		t.Fatalf("tool call header = %q, want call-1", got)
	}
	if got := msg.Header.Get(ToolNameHeader); got != "run_table_add" {
		t.Fatalf("tool name header = %q, want run_table_add", got)
	}
}

func TestResolveClientSourceInfersOpenAPI(t *testing.T) {
	ctx := WithSourceInfo(context.Background(), SourceTypeOpenAPIToken, "alice")

	if got := ResolveClientSource(ctx); got != ClientSourceOpenAPI {
		t.Fatalf("ResolveClientSource(ctx) = %q, want openapi", got)
	}
}

func TestResolveClientSourceInfersScheduledTask(t *testing.T) {
	ctx := WithSourceInfo(context.Background(), SourceTypeScheduledTask, "timer_task:1:execution:2")

	if got := ResolveClientSource(ctx); got != ClientSourceScheduledTask {
		t.Fatalf("ResolveClientSource(ctx) = %q, want scheduled_task", got)
	}
}

func TestGetAuditClientSourceFallsBackToUnknown(t *testing.T) {
	if got := GetAuditClientSource(context.Background()); got != ClientSourceUnknown {
		t.Fatalf("GetAuditClientSource(ctx) = %q, want unknown", got)
	}
}

func TestCtxToTraceNatsPreservesDepartment(t *testing.T) {
	ctx := WithDepartmentFullPath(context.Background(), "/org/dev")

	msg := CtxToTraceNats(ctx, "demo")

	if got := msg.Header.Get(DepartmentFullPathHeader); got != "/org/dev" {
		t.Fatalf("nats department header = %q, want /org/dev", got)
	}
}
