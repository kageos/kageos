package service

import (
	"context"
	"testing"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
)

func TestFillMessageMetaFromContext(t *testing.T) {
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		TraceId:               "trace-1",
		RequestUser:           "alice",
		DepartmentFullPath:    "/org/dev",
		ClientSource:          "scheduled_task",
		SourceType:            "timer_execution",
		SourceRef:             "42",
		SourcePath:            "/system/demos/meeting/meeting_room_notify_soon.form",
		SourceTitle:           "会议即将开始提醒",
		SourceParentPath:      "/system/demos/meeting",
		SourceParentTitle:     "智能会议室系统",
		SourceTemplateType:    "form",
		WorkspaceSessionID:    "session-1",
		WorkspaceSessionTitle: "定时会议巡检",
		WorkspaceRole:         "automation_operator",
	})
	meta := dto.MessageSendMeta{}
	fillMessageMetaFromContext(ctx, &meta)

	if meta.From != "alice" || meta.RequestUser != "alice" {
		t.Fatalf("meta user = %#v", meta)
	}
	if meta.TraceID != "trace-1" || meta.SourceType != "timer_execution" || meta.SourceRef != "42" {
		t.Fatalf("meta context fields = %#v", meta)
	}
	if meta.SourcePath != "/system/demos/meeting/meeting_room_notify_soon.form" ||
		meta.SourceTitle != "会议即将开始提醒" ||
		meta.SourceParentPath != "/system/demos/meeting" ||
		meta.SourceParentTitle != "智能会议室系统" ||
		meta.SourceTemplateType != "form" {
		t.Fatalf("meta source display = %#v", meta)
	}
	if meta.WorkspaceSessionID != "session-1" || meta.WorkspaceSessionTitle != "定时会议巡检" || meta.WorkspaceRole != "automation_operator" {
		t.Fatalf("meta workspace session = %#v", meta)
	}
}
