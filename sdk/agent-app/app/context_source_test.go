package app

import (
	"context"
	"testing"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
)

func TestNewContextCarriesClientSource(t *testing.T) {
	a := &App{}

	ctx, err := a.NewContext(context.Background(), &dto.RequestAppReq{
		Method:       "POST",
		Router:       "/demo",
		ClientSource: "agent",
	})
	if err != nil {
		t.Fatalf("NewContext returned error: %v", err)
	}

	if got := ctx.GetClientSource(); got != "agent" {
		t.Fatalf("ctx.GetClientSource() = %q, want agent", got)
	}
	if got := contextx.GetClientSource(ctx); got != "agent" {
		t.Fatalf("contextx.GetClientSource(ctx) = %q, want agent", got)
	}
}

func TestNewContextCarriesRequestInfo(t *testing.T) {
	a := &App{}

	ctx, err := a.NewContext(context.Background(), &dto.RequestAppReq{
		TraceId:         "trace-1",
		RequestUser:     "alice",
		RequestUserDept: "/org/dev",
		Token:           "token-1",
		ClientSource:    "agent",
	})
	if err != nil {
		t.Fatalf("NewContext returned error: %v", err)
	}

	if got := contextx.GetTraceId(ctx); got != "trace-1" {
		t.Fatalf("trace id = %q, want trace-1", got)
	}
	if got := contextx.GetRequestUser(ctx); got != "alice" {
		t.Fatalf("request user = %q, want alice", got)
	}
	if got := contextx.GetToken(ctx); got != "token-1" {
		t.Fatalf("token = %q, want token-1", got)
	}
	if got := contextx.GetRequestDepartmentFullPath(ctx); got != "/org/dev" {
		t.Fatalf("dept = %q, want /org/dev", got)
	}
}
