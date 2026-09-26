package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/gormx/models"
	"testing"
	"time"
)

func TestScheduledStorageMigrationAndTaskIsolation(t *testing.T) {
	db := newOperateLogRepositoryTestDB(t)
	if err := db.AutoMigrate(&model.ScheduledExecutionLog{}, &model.LogArchiveBatch{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	repo := NewOperateLogRepository(db)
	old := model.OperateLog{TenantUser: "a", App: "b", ResourcePath: "/a/b/f", ResourceType: "form", Source: "scheduled_task", SourceRef: "timer_task:1:execution:10", Status: "failed", NewValuesJSON: json.RawMessage(`{"error":"timeout"}`)}
	old.CreatedAt = models.Time(time.Now().AddDate(0, 0, -10))
	if err := db.Create(&old).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&old, old.ID).Error; err != nil {
		t.Fatal(err)
	}
	// New writes can already have IDs overlapping the old table.
	fresh := old
	fresh.ID = 0
	fresh.SourceRef = "timer_task:10:execution:11"
	if err := repo.CreateOperateLog(ctx, &fresh); err != nil {
		t.Fatal(err)
	}
	archive := NewLogArchiveRepository(db)
	for range 2 {
		if err := archive.MoveLegacyScheduledLogs(ctx); err != nil {
			t.Fatal(err)
		}
	}
	logs, total, err := repo.GetOperateLogs(ctx, &dto.GetOperateLogsReq{LogKind: "scheduled", TaskID: 1})
	if err != nil || total != 1 || len(logs) != 1 {
		t.Fatalf("task boundary: %d %v", total, err)
	}
	if logs[0].OriginalLogID == nil || *logs[0].OriginalLogID != old.ID || string(logs[0].NewValuesJSON) != string(old.NewValuesJSON) || !time.Time(logs[0].CreatedAt).Equal(time.Time(old.CreatedAt)) {
		t.Fatalf("migration lost original fields: %+v", logs[0])
	}
	human := model.OperateLog{ResourceType: "scheduled_task", Action: "timer.task.paused", Source: "scheduled_task"}
	if err := repo.CreateOperateLog(ctx, &human); err != nil {
		t.Fatal(err)
	}
	_, total, err = repo.GetOperateLogs(ctx, &dto.GetOperateLogsReq{})
	if err != nil || total != 1 {
		t.Fatalf("management action missing: %d %v", total, err)
	}
}

func TestScheduledStorageMigrationMovesMultipleBatches(t *testing.T) {
	db := newOperateLogRepositoryTestDB(t)
	if err := db.AutoMigrate(&model.ScheduledExecutionLog{}, &model.LogArchiveBatch{}); err != nil {
		t.Fatal(err)
	}
	rows := make([]model.OperateLog, 501)
	for i := range rows {
		rows[i] = model.OperateLog{
			TenantUser:   "a",
			App:          "b",
			ResourceType: "form",
			Source:       "scheduled_task",
			SourceRef:    fmt.Sprintf("timer_task:1:execution:%d", i+1),
		}
	}
	if err := db.CreateInBatches(&rows, 100).Error; err != nil {
		t.Fatal(err)
	}
	if err := NewLogArchiveRepository(db).MoveLegacyScheduledLogs(context.Background()); err != nil {
		t.Fatal(err)
	}
	var legacyCount, migratedCount, distinctOriginalIDs int64
	if err := db.Model(&model.OperateLog{}).Count(&legacyCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.ScheduledExecutionLog{}).Count(&migratedCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.ScheduledExecutionLog{}).Distinct("original_log_id").Count(&distinctOriginalIDs).Error; err != nil {
		t.Fatal(err)
	}
	if legacyCount != 0 || migratedCount != 501 || distinctOriginalIDs != 501 {
		t.Fatalf("unexpected migration counts: legacy=%d migrated=%d original_ids=%d", legacyCount, migratedCount, distinctOriginalIDs)
	}
}

func TestArchiveThresholdDoesNotAbsorbLowVolumePaths(t *testing.T) {
	db := newOperateLogRepositoryTestDB(t)
	if err := db.AutoMigrate(&model.ScheduledExecutionLog{}, &model.LogArchiveBatch{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Now()
	for _, path := range []string{"/a/b/busy", "/a/b/busy", "/a/b/busy/quiet", "/a/b/recent", "/a/b/recent"} {
		at := now.AddDate(0, 0, -100)
		if path == "/a/b/recent" {
			at = now
		}
		row := model.OperateLog{ResourcePath: path, TenantUser: "a", App: "b", Status: "success"}
		row.CreatedAt = models.Time(at)
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	repo := NewLogArchiveRepository(db).ForType("operate_log")
	_, _, ids, err := repo.SelectArchiveIDs(ctx, now.AddDate(0, 0, -90), 2, 100)
	if err != nil || len(ids) != 2 {
		t.Fatalf("selection: %v %v", ids, err)
	}
	selected, _ := json.Marshal(ids)
	if _, err := repo.DeleteRange(ctx, &model.LogArchiveBatch{ArchiveType: "operate_log", SelectedIDsJSON: selected}, 100); err != nil {
		t.Fatal(err)
	}
	_, _, _, err = repo.SelectArchiveIDs(ctx, now.AddDate(0, 0, -90), 2, 100)
	if !IsArchiveNotFound(err) {
		t.Fatalf("low volume/recent logs selected: %v", err)
	}
	for _, status := range []string{"success", "failed"} {
		row := model.ScheduledExecutionLog{TenantUser: "a", App: "b", ResourcePath: "/a/b/task", Status: status}
		row.CreatedAt = models.Time(now.AddDate(0, 0, -8))
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	_, _, ids, err = repo.ForType("scheduled_execution").SelectArchiveIDs(ctx, now.AddDate(0, 0, -7), 1, 100)
	if err != nil || len(ids) != 2 {
		t.Fatalf("scheduled failures must also archive: %v %v", ids, err)
	}
}

func TestMigrationLeavesInFlightArchiveIDsUntouched(t *testing.T) {
	db := newOperateLogRepositoryTestDB(t)
	if err := db.AutoMigrate(&model.ScheduledExecutionLog{}, &model.LogArchiveBatch{}); err != nil {
		t.Fatal(err)
	}
	old := model.OperateLog{TenantUser: "a", App: "b", ResourceType: "form", Source: "scheduled_task"}
	if err := db.Create(&old).Error; err != nil {
		t.Fatal(err)
	}
	batch := model.LogArchiveBatch{ArchiveType: "operate_log", ArchiveKey: "in-flight", TenantUser: "a", App: "b", MinLogID: old.ID, MaxLogID: old.ID, Status: "exporting", SelectedIDsJSON: json.RawMessage(`[1]`)}
	if err := db.Create(&batch).Error; err != nil {
		t.Fatal(err)
	}
	if err := NewLogArchiveRepository(db).MoveLegacyScheduledLogs(context.Background()); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&model.OperateLog{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("claimed log moved: %d %v", count, err)
	}
}
