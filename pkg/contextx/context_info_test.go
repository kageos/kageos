package contextx

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

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
