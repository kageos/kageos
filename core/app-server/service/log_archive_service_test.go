package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/pkg/gormx/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeleteArchivedSourceRequiresVerificationAndDeletesOnlySelectedIDs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:log-archive-delete?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.OperateLog{}, &model.LogArchiveBatch{}); err != nil {
		t.Fatal(err)
	}
	for id := int64(1); id <= 3; id++ {
		if err := db.Create(&model.OperateLog{Base: models.Base{ID: id}, TenantUser: "alice", App: "crm", ActorUser: "alice", Action: "test"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	selected, _ := json.Marshal([]int64{1, 3})
	batch := &model.LogArchiveBatch{
		ArchiveKey: "test", ArchiveType: logArchiveTypeOperate, TenantUser: "alice", App: "crm",
		MinLogID: 1, MaxLogID: 3, RecordCount: 2, SelectedIDsJSON: selected, Status: model.LogArchiveStatusExporting,
		RangeStartedAt: time.Now(), RangeEndedAt: time.Now(),
	}
	if err := db.Create(batch).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewLogArchiveService(repository.NewLogArchiveRepository(db), DefaultLogArchiveConfig())
	if err := svc.deleteArchivedSource(context.Background(), batch); err == nil {
		t.Fatal("unverified batch must not delete source logs")
	}
	var count int64
	if err := db.Model(&model.OperateLog{}).Count(&count).Error; err != nil || count != 3 {
		t.Fatalf("source count after refusal = %d, err=%v", count, err)
	}
	now := time.Now()
	batch.ObjectVerifiedAt, batch.ObjectRef, batch.SHA256 = &now, "bucket/key", "abc"
	if err := svc.deleteArchivedSource(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	var remaining []int64
	if err := db.Model(&model.OperateLog{}).Order("id").Pluck("id", &remaining).Error; err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 || remaining[0] != 2 {
		t.Fatalf("remaining IDs = %v, want [2]", remaining)
	}
	if batch.Status != model.LogArchiveStatusCompleted || batch.DeletedAtSource == nil {
		t.Fatalf("unexpected completed batch: %+v", batch)
	}
}

func TestDefaultLogArchiveConfig(t *testing.T) {
	t.Setenv("KAGEOS_LOG_ARCHIVE_RETENTION_DAYS", "30")
	t.Setenv("KAGEOS_LOG_ARCHIVE_CRON", "5 2 * * *")
	cfg := DefaultLogArchiveConfig()
	if !cfg.Enabled || cfg.RetentionDays != 30 || cfg.CronExpr != "5 2 * * *" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}
