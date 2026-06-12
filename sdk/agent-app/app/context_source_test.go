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

func TestNewContextCarriesSourceDisplayAndWorkspaceSession(t *testing.T) {
	a := &App{}

	ctx, err := a.NewContext(context.Background(), &dto.RequestAppReq{
		SourcePath:            "/system/demos/meeting/meeting_room_notify_soon.form",
		SourceTitle:           "会议即将开始提醒",
		SourceParentPath:      "/system/demos/meeting",
		SourceParentTitle:     "智能会议室系统",
		SourceTemplateType:    "form",
		WorkspaceSessionID:    "session-1",
		WorkspaceSessionTitle: "定时会议巡检",
		WorkspaceRole:         "automation_operator",
	})
	if err != nil {
		t.Fatalf("NewContext returned error: %v", err)
	}

	if got := contextx.GetSourcePath(ctx); got != "/system/demos/meeting/meeting_room_notify_soon.form" {
		t.Fatalf("source path = %q", got)
	}
	if got := contextx.GetSourceParentTitle(ctx); got != "智能会议室系统" {
		t.Fatalf("source parent title = %q", got)
	}
	if got := ctx.GetWorkspaceSessionID(); got != "session-1" {
		t.Fatalf("workspace session id = %q", got)
	}
	if got := ctx.GetWorkspaceSessionTitle(); got != "定时会议巡检" {
		t.Fatalf("workspace session title = %q", got)
	}
	if got := ctx.GetWorkspaceRole(); got != "automation_operator" {
		t.Fatalf("workspace role = %q", got)
	}
}

func TestNewContextCarriesPublicShareContext(t *testing.T) {
	a := &App{}

	ctx, err := a.NewContext(context.Background(), &dto.RequestAppReq{
		TraceId:        "trace-1",
		RequestUser:    "guest_anon_1",
		Token:          "legacy-anon-token",
		AnonymousToken: "anon-token",
		ClientSource:   "public_share",
		SourceType:     "public_share",
		SourceRef:      "ps_123",
	})
	if err != nil {
		t.Fatalf("NewContext returned error: %v", err)
	}

	if got := ctx.token; got != "" {
		t.Fatalf("ctx.token = %q, want empty for public share", got)
	}
	if got := ctx.anonymousToken; got != "anon-token" {
		t.Fatalf("ctx.anonymousToken = %q, want anon-token", got)
	}
	if got := contextx.GetSourceType(ctx); got != "public_share" {
		t.Fatalf("source type = %q, want public_share", got)
	}
	if got := contextx.GetSourceRef(ctx); got != "ps_123" {
		t.Fatalf("source ref = %q, want ps_123", got)
	}
}
