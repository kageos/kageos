package repository

import (
	"context"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
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
