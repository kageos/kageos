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
	if got := normalizeOperateLogOrderBy("created_at asc"); got != "created_at ASC" {
		t.Fatalf("created_at asc = %q", got)
	}
	if got := normalizeOperateLogOrderBy("created_at DESC; DROP TABLE operate_logs"); got != "created_at DESC" {
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

func TestGetOperateLogsFiltersByCompanyCode(t *testing.T) {
	db := newOperateLogRepositoryTestDB(t)
	repo := NewOperateLogRepository(db)
	ctx := context.Background()

	for _, item := range []struct {
		companyCode string
		actorUser   string
	}{
		{companyCode: "acme", actorUser: "alice"},
		{companyCode: "other", actorUser: "bob"},
	} {
		if err := repo.CreateOperateLog(ctx, &model.OperateLog{
			TenantUser:   "owner",
			CompanyCode:  item.companyCode,
			App:          "ops",
			ActorUser:    item.actorUser,
			Action:       "team.role.assigned",
			ResourceType: "team_access",
			ResourcePath: "/owner/ops",
			Status:       "success",
		}); err != nil {
			t.Fatalf("create log: %v", err)
		}
	}

	logs, total, err := repo.GetOperateLogs(ctx, &dto.GetOperateLogsReq{
		CompanyCode: "acme",
		Page:        1,
		PageSize:    20,
	})
	if err != nil {
		t.Fatalf("query logs: %v", err)
	}
	if total != 1 || len(logs) != 1 || logs[0].CompanyCode != "acme" {
		t.Fatalf("expected only acme logs, total=%d logs=%+v", total, logs)
	}
}

func TestGetOperateLogsIncludesLegacyEmptyCompanyLogs(t *testing.T) {
	db := newOperateLogRepositoryTestDB(t)
	repo := NewOperateLogRepository(db)
	ctx := context.Background()

	for _, item := range []struct {
		companyCode string
		source      string
		trace       string
	}{
		{companyCode: "acme", source: "browser", trace: "trace-acme"},
		{companyCode: "", source: "scheduled_task", trace: "trace-legacy-scheduled"},
		{companyCode: "other", source: "scheduled_task", trace: "trace-other"},
	} {
		if err := repo.CreateOperateLog(ctx, &model.OperateLog{
			TenantUser:   "owner",
			CompanyCode:  item.companyCode,
			App:          "ops",
			ActorUser:    "alice",
			Action:       "scheduled_function_execute",
			ResourceType: "function",
			ResourcePath: "/owner/ops/report.chart",
			Status:       "success",
			Source:       item.source,
			TraceID:      item.trace,
		}); err != nil {
			t.Fatalf("create log: %v", err)
		}
	}

	logs, total, err := repo.GetOperateLogs(ctx, &dto.GetOperateLogsReq{
		CompanyCode:  "acme",
		ResourcePath: "/owner/ops/report.chart",
		Page:         1,
		PageSize:     20,
	})
	if err != nil {
		t.Fatalf("query logs: %v", err)
	}
	if total != 2 || len(logs) != 2 {
		t.Fatalf("expected acme + legacy empty company logs, total=%d logs=%+v", total, logs)
	}
	for _, log := range logs {
		if log.CompanyCode == "other" {
			t.Fatalf("other company log should not be returned: %+v", log)
		}
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
