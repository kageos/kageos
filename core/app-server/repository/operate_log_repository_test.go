package repository

import (
	"context"
	"testing"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newOperateLogRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.OperateLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestNormalizeOperateLogOrderByWhitelist(t *testing.T) {
	if got := normalizeOperateLogOrderBy("created_at asc"); got != "created_at ASC, id ASC" {
		t.Fatalf("created_at asc = %q", got)
	}
	if got := normalizeOperateLogOrderBy("created_at DESC; DROP TABLE operate_logs"); got != "created_at DESC, id DESC" {
		t.Fatalf("unsafe order should fall back to desc, got %q", got)
	}
}

func TestGetOperateLogsFiltersByID(t *testing.T) {
	db := newOperateLogRepositoryTestDB(t)
	repo := NewOperateLogRepository(db)
	ctx := context.Background()

	first := &model.OperateLog{
		TenantUser:   "alice",
		App:          "ops",
		ActorUser:    "alice",
		Action:       "form_submit",
		ResourceType: "form",
		ResourcePath: "/alice/ops/export.form",
		Status:       "success",
	}
	if err := repo.CreateOperateLog(ctx, first); err != nil {
		t.Fatalf("create first log: %v", err)
	}
	if err := repo.CreateOperateLog(ctx, &model.OperateLog{
		TenantUser:   "alice",
		App:          "ops",
		ActorUser:    "bob",
		Action:       "form_submit",
		ResourceType: "form",
		ResourcePath: "/alice/ops/export.form",
		Status:       "success",
	}); err != nil {
		t.Fatalf("create second log: %v", err)
	}

	logs, total, err := repo.GetOperateLogs(ctx, &dto.GetOperateLogsReq{
		ID:       first.ID,
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("query logs: %v", err)
	}
	if total != 1 || len(logs) != 1 || logs[0].ID != first.ID {
		t.Fatalf("expected only id=%d, total=%d logs=%+v", first.ID, total, logs)
	}
}

func TestGetOperateLogsFiltersRowIDByTargetID(t *testing.T) {
	db := newOperateLogRepositoryTestDB(t)
	repo := NewOperateLogRepository(db)
	ctx := context.Background()

	if err := repo.CreateOperateLog(ctx, &model.OperateLog{
		TenantUser:   "alice",
		App:          "ops",
		ActorUser:    "alice",
		Action:       "OnTableUpdateRow",
		ResourceType: "table",
		ResourcePath: "/alice/ops/tickets.table",
		TargetID:     "42",
		DetailsJSON:  []byte(`{"row_id":7}`),
		Status:       "success",
	}); err != nil {
		t.Fatalf("create target log: %v", err)
	}

	if err := repo.CreateOperateLog(ctx, &model.OperateLog{
		TenantUser:   "alice",
		App:          "ops",
		ActorUser:    "alice",
		Action:       "OnTableUpdateRow",
		ResourceType: "table",
		ResourcePath: "/alice/ops/tickets.table",
		TargetID:     "7",
		Status:       "success",
	}); err != nil {
		t.Fatalf("create other log: %v", err)
	}

	logs, total, err := repo.GetOperateLogs(ctx, &dto.GetOperateLogsReq{
		ResourcePath: "/alice/ops/tickets.table",
		RowID:        42,
		Page:         1,
		PageSize:     20,
	})
	if err != nil {
		t.Fatalf("query logs: %v", err)
	}
	if total != 1 || len(logs) != 1 || logs[0].TargetID != "42" {
		t.Fatalf("expected only target_id=42, total=%d logs=%+v", total, logs)
	}
}

func TestGetOperateLogsFiltersBySource(t *testing.T) {
	db := newOperateLogRepositoryTestDB(t)
	repo := NewOperateLogRepository(db)
	ctx := context.Background()

	for _, item := range []struct {
		source string
		trace  string
	}{
		{source: "agent", trace: "trace-agent"},
		{source: "openapi", trace: "trace-openapi"},
	} {
		if err := repo.CreateOperateLog(ctx, &model.OperateLog{
			TenantUser:   "owner",
			App:          "ops",
			ActorUser:    "alice",
			Action:       "form_submit",
			ResourceType: "form",
			ResourcePath: "/owner/ops/export.form",
			Status:       "success",
			Source:       item.source,
			TraceID:      item.trace,
		}); err != nil {
			t.Fatalf("create log: %v", err)
		}
	}

	logs, total, err := repo.GetOperateLogs(ctx, &dto.GetOperateLogsReq{
		Source:   "openapi",
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("query logs: %v", err)
	}
	if total != 1 || len(logs) != 1 || logs[0].Source != "openapi" {
		t.Fatalf("expected only openapi logs, total=%d logs=%+v", total, logs)
	}
}

func TestGetOperateLogsExcludesScheduledTasksByDefaultFlag(t *testing.T) {
	db := newOperateLogRepositoryTestDB(t)
	repo := NewOperateLogRepository(db)
	ctx := context.Background()

	logs := []*model.OperateLog{
		{TenantUser: "alice", App: "ops", ActorUser: "alice", Action: "OnTableUpdateRow", ResourceType: "table", ResourcePath: "/alice/ops/tickets.table", Status: "success", Source: "browser"},
		{TenantUser: "alice", App: "ops", ActorUser: "scheduler", Action: "OnTableUpdateRow", ResourceType: "table", ResourcePath: "/alice/ops/tickets.table", Status: "success", Source: "scheduled_task"},
		{TenantUser: "alice", App: "ops", ActorUser: "scheduler", Action: "scheduled_function_execute", ResourceType: "table", ResourcePath: "/alice/ops/tickets.table", Status: "success", ExecutorType: "scheduled_function"},
	}
	for _, log := range logs {
		if err := repo.CreateOperateLog(ctx, log); err != nil {
			t.Fatalf("create log: %v", err)
		}
	}

	got, total, err := repo.GetOperateLogs(ctx, &dto.GetOperateLogsReq{
		ResourcePath:          "/alice/ops/tickets.table",
		ExcludeScheduledTasks: true,
		Page:                  1,
		PageSize:              20,
	})
	if err != nil {
		t.Fatalf("query logs: %v", err)
	}
	if total != 1 || len(got) != 1 || got[0].Source != "browser" {
		t.Fatalf("logs = %+v, total = %d; want only browser log", got, total)
	}
}

func TestGetOperateLogsFiltersByWorkspaceSessionID(t *testing.T) {
	db := newOperateLogRepositoryTestDB(t)
	repo := NewOperateLogRepository(db)
	ctx := context.Background()

	for _, item := range []struct {
		sessionID    string
		executorType string
		trace        string
	}{
		{sessionID: "session-target", executorType: "agent", trace: "trace-target"},
		{sessionID: "session-other", executorType: "agent", trace: "trace-other"},
	} {
		if err := repo.CreateOperateLog(ctx, &model.OperateLog{
			TenantUser:         "owner",
			App:                "ops",
			ActorUser:          "alice",
			Action:             "form_submit",
			ResourceType:       "form",
			ResourcePath:       "/owner/ops/export.form",
			Status:             "success",
			Source:             "agent",
			ExecutorType:       item.executorType,
			WorkspaceSessionID: item.sessionID,
			TraceID:            item.trace,
		}); err != nil {
			t.Fatalf("create log: %v", err)
		}
	}

	logs, total, err := repo.GetOperateLogs(ctx, &dto.GetOperateLogsReq{
		WorkspaceSessionID: "session-target",
		Page:               1,
		PageSize:           20,
	})
	if err != nil {
		t.Fatalf("query logs: %v", err)
	}
	if total != 1 || len(logs) != 1 || logs[0].WorkspaceSessionID != "session-target" {
		t.Fatalf("expected only session-target logs, total=%d logs=%+v", total, logs)
	}
}

func TestGetOperateLogsFiltersByTraceID(t *testing.T) {
	db := newOperateLogRepositoryTestDB(t)
	repo := NewOperateLogRepository(db)
	ctx := context.Background()

	for _, item := range []struct {
		traceID string
		action  string
	}{
		{traceID: "trace-target", action: "form_submit"},
		{traceID: "trace-other", action: "OnTableUpdateRow"},
	} {
		if err := repo.CreateOperateLog(ctx, &model.OperateLog{
			TenantUser:   "owner",
			App:          "ops",
			ActorUser:    "alice",
			Action:       item.action,
			ResourceType: "form",
			ResourcePath: "/owner/ops/export.form",
			Status:       "success",
			TraceID:      item.traceID,
		}); err != nil {
			t.Fatalf("create log: %v", err)
		}
	}

	logs, total, err := repo.GetOperateLogs(ctx, &dto.GetOperateLogsReq{
		TraceID:  "trace-target",
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("query logs: %v", err)
	}
	if total != 1 || len(logs) != 1 || logs[0].TraceID != "trace-target" {
		t.Fatalf("expected only trace-target log, total=%d logs=%+v", total, logs)
	}
}
