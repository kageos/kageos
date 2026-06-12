package service

import (
	"context"
	"testing"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
)

func TestFillMessageMetaFromContext(t *testing.T) {
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		TraceId:            "trace-1",
		RequestUser:        "alice",
		DepartmentFullPath: "/org/dev",
		ClientSource:       "scheduled_task",
		SourceType:         "timer_execution",
		SourceRef:          "42",
	})
	meta := dto.MessageSendMeta{}
	fillMessageMetaFromContext(ctx, &meta)

	if meta.From != "alice" || meta.RequestUser != "alice" {
		t.Fatalf("meta user = %#v", meta)
	}
	if meta.TraceID != "trace-1" || meta.SourceType != "timer_execution" || meta.SourceRef != "42" {
		t.Fatalf("meta context fields = %#v", meta)
	}
}
