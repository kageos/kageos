package repository

import (
	"context"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestGetFormOperateLogsFiltersByTraceID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&model.FormOperateLog{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	repo := NewOperateLogRepository(db)
	logs := []*model.FormOperateLog{
		{
			TenantUser:   "tenant",
			RequestUser:  "alice",
			Action:       "form_submit",
			App:          "demo",
			FullCodePath: "/demo/app/form",
			Source:       "scheduled_task",
			Code:         0,
			TraceID:      "trace-1",
		},
		{
			TenantUser:   "tenant",
			RequestUser:  "alice",
			Action:       "form_submit",
			App:          "demo",
			FullCodePath: "/demo/app/form",
			Source:       "scheduled_task",
			Code:         0,
			TraceID:      "trace-2",
		},
	}
	for _, log := range logs {
		if err := repo.CreateFormOperateLog(log); err != nil {
			t.Fatalf("create form operate log failed: %v", err)
		}
	}

	result, total, err := repo.GetFormOperateLogs(context.Background(), &dto.GetFormOperateLogsReq{
		FullCodePath: "/demo/app/form",
		Action:       "form_submit",
		Source:       "scheduled_task",
		TraceID:      "trace-2",
		Keyword:      "trace",
		Page:         1,
		PageSize:     20,
	})
	if err != nil {
		t.Fatalf("get form operate logs failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("unexpected total: got %d, want 1", total)
	}
	if len(result) != 1 {
		t.Fatalf("unexpected log count: got %d, want 1", len(result))
	}
	if result[0].TraceID != "trace-2" {
		t.Fatalf("unexpected trace id: got %q, want %q", result[0].TraceID, "trace-2")
	}
}
