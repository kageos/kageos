package service

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
	if err := db.AutoMigrate(&model.OperateLog{}, &model.ScheduledExecutionLog{}, &model.LogArchiveBatch{}); err != nil {
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
	t.Setenv("KAGEOS_LOG_ARCHIVE_MAX_BATCHES", "")
	t.Setenv("KAGEOS_LOG_ARCHIVE_BATCH_SIZE", "")
	t.Setenv("KAGEOS_LOG_ARCHIVE_RETENTION_DAYS", "30")
	t.Setenv("KAGEOS_LOG_ARCHIVE_CRON", "5 2 * * *")
	cfg := DefaultLogArchiveConfig()
	if !cfg.Enabled || cfg.RetentionDays != 30 || cfg.CronExpr != "5 2 * * *" || cfg.MaxBatches != 1000 || cfg.BatchSize != 10000 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func archiveFixture(t *testing.T) (*gorm.DB, *LogArchiveService, *model.LogArchiveBatch) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.OperateLog{}, &model.ScheduledExecutionLog{}, &model.LogArchiveBatch{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	selected, _ := json.Marshal([]int64{1, 2})
	batch := &model.LogArchiveBatch{ArchiveKey: t.Name(), ArchiveType: logArchiveTypeOperate, TenantUser: "alice", App: "crm", MinLogID: 1, MaxLogID: 2, RecordCount: 2, SelectedIDsJSON: selected, Status: model.LogArchiveStatusUploaded, ObjectVerifiedAt: &now, ObjectRef: "bucket/verified", SHA256: "verified", RangeStartedAt: now, RangeEndedAt: now}
	if err := db.Create(batch).Error; err != nil {
		t.Fatal(err)
	}
	for id := int64(1); id <= 3; id++ {
		if err := db.Create(&model.OperateLog{Base: models.Base{ID: id, CreatedAt: models.Time(now.AddDate(0, 0, -100))}, TenantUser: "alice", App: "crm", ActorUser: "alice", Action: "test"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db, NewLogArchiveService(repository.NewLogArchiveRepository(db), DefaultLogArchiveConfig()), batch
}

func TestArchiveCompletionWriteFailureResumesWithoutExportingDeletedLogs(t *testing.T) {
	db, svc, batch := archiveFixture(t)
	fail := true
	if err := db.Callback().Update().Before("gorm:update").Register("test:completion_failure", func(tx *gorm.DB) {
		if value, ok := tx.Statement.Dest.(*model.LogArchiveBatch); ok && value.Status == model.LogArchiveStatusCompleted && fail {
			fail = false
			tx.AddError(fmt.Errorf("injected completion write failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Retry(context.Background(), batch.ID); err == nil {
		t.Fatal("expected injected failure")
	}
	saved, err := svc.repo.Get(context.Background(), batch.ID)
	if err != nil || saved.Status != model.LogArchiveStatusUploaded || saved.NextRetryAt == nil {
		t.Fatalf("checkpoint lost: %+v %v", saved, err)
	}
	var count int64
	db.Model(&model.OperateLog{}).Count(&count)
	if count != 1 {
		t.Fatalf("source cleanup count = %d", count)
	}
	if err := svc.Retry(context.Background(), batch.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Retry(context.Background(), batch.ID); err != nil {
		t.Fatal("completed retry must be idempotent", err)
	}
	saved, _ = svc.repo.Get(context.Background(), batch.ID)
	if saved.Status != model.LogArchiveStatusCompleted || saved.Attempts != 2 {
		t.Fatalf("bad completion: %+v", saved)
	}
}

func TestArchiveFailedLegacyCheckpointAndPartialCleanupResume(t *testing.T) {
	db, svc, batch := archiveFixture(t)
	db.Unscoped().Delete(&model.OperateLog{}, 1)
	batch.Status = model.LogArchiveStatusFailed
	if err := svc.repo.Save(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	if err := svc.Retry(context.Background(), batch.ID); err != nil {
		t.Fatal(err)
	}
	var ids []int64
	db.Model(&model.OperateLog{}).Pluck("id", &ids)
	if len(ids) != 1 || ids[0] != 3 {
		t.Fatalf("unexpected remaining IDs: %v", ids)
	}
}

func TestFailedArchiveBackoffDoesNotClaimItsSourceAgain(t *testing.T) {
	db, svc, batch := archiveFixture(t)
	batch.ObjectRef, batch.SHA256, batch.ObjectVerifiedAt = "", "", nil
	svc.markFailed(context.Background(), batch, fmt.Errorf("network unavailable"))
	if _, err := svc.repo.GetResumable(context.Background()); !repository.IsArchiveNotFound(err) {
		t.Fatalf("backoff batch was selected: %v", err)
	}
	_, _, ids, err := svc.repo.ForType("operate_log").SelectArchiveIDs(context.Background(), time.Now(), 1, 100)
	if err != nil || len(ids) != 1 || ids[0] != 3 {
		t.Fatalf("claimed source was selected again: %v %v", ids, err)
	}
	var count int64
	db.Model(&model.OperateLog{}).Count(&count)
	if count != 3 {
		t.Fatalf("failure deleted source: %d", count)
	}
}

func TestVerifyArchiveRejectsSameSizeCorruptionAndTruncation(t *testing.T) {
	payload := "verified gzip bytes"
	hash := sha256.Sum256([]byte(payload))
	for _, body := range []string{payload, "corrupt! gzip bytes", "short"} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, body) }))
		svc := &LogArchiveService{httpClient: server.Client()}
		err := svc.verifyUploadedObject(context.Background(), server.URL, int64(len(payload)), hex.EncodeToString(hash[:]))
		server.Close()
		if (err == nil) != (body == payload) {
			t.Fatalf("verification body=%q error=%v", body, err)
		}
	}
}

func TestRunContinuesAfterOneBatchFails(t *testing.T) {
	db, svc, failed := archiveFixture(t)
	failed.Status, failed.ObjectRef, failed.SHA256, failed.ObjectVerifiedAt = model.LogArchiveStatusExporting, "", "", nil
	failed.RecordCount = 99 // export fails before any upload or deletion
	if err := svc.repo.Save(context.Background(), failed); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	selected, _ := json.Marshal([]int64{3})
	ready := &model.LogArchiveBatch{ArchiveKey: "ready", ArchiveType: logArchiveTypeOperate, TenantUser: "alice", App: "crm", MinLogID: 3, MaxLogID: 3, RecordCount: 1, SelectedIDsJSON: selected, Status: model.LogArchiveStatusUploaded, ObjectVerifiedAt: &now, ObjectRef: "bucket/ready", SHA256: "verified", RangeStartedAt: now, RangeEndedAt: now}
	if err := db.Create(ready).Error; err != nil {
		t.Fatal(err)
	}
	summary, err := svc.Run(context.Background())
	if err == nil || summary.Batches != 1 || summary.Records != 1 {
		t.Fatalf("other batches blocked: %+v %v", summary, err)
	}
	var count int64
	db.Model(&model.OperateLog{}).Count(&count)
	if count != 2 {
		t.Fatalf("failed batch source was removed: %d", count)
	}
}

func TestManualArchiveRetryCannotOverlapAnotherWorker(t *testing.T) {
	_, svc, batch := archiveFixture(t)
	err := svc.repo.Exclusive(context.Background(), func(_ *repository.LogArchiveRepository) error {
		if err := svc.Retry(context.Background(), batch.ID); err == nil {
			t.Fatal("concurrent retry should be rejected")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestArchiveRunContinuesBeyondTwentyBatchesAndRenewsContext(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.OperateLog{}, &model.ScheduledExecutionLog{}, &model.LogArchiveBatch{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for i := 0; i < 21; i++ {
		batch := &model.LogArchiveBatch{ArchiveKey: fmt.Sprintf("verified-%d", i), ArchiveType: logArchiveTypeOperate, TenantUser: "alice", App: "crm", Status: model.LogArchiveStatusUploaded, SelectedIDsJSON: json.RawMessage(`[]`), ObjectVerifiedAt: &now, ObjectRef: "bucket/verified", SHA256: "verified", RangeStartedAt: now, RangeEndedAt: now}
		if err := db.Create(batch).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc := NewLogArchiveService(repository.NewLogArchiveRepository(db), LogArchiveConfig{RetentionDays: 90, BatchSize: 10000, MaxBatches: 1000})
	renewals := 0
	result, err := svc.runWithContext(context.Background(), func(ctx context.Context) (context.Context, error) { renewals++; return ctx, nil })
	if err != nil || result.Batches != 21 || renewals != 22 || result.StopReason != "no_eligible_logs" {
		t.Fatalf("result=%+v renewals=%d error=%v", result, renewals, err)
	}
}

func TestArchiveProgressSurvivesServiceRecreationAndReportsDeferredRetry(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.OperateLog{}, &model.ScheduledExecutionLog{}, &model.LogArchiveBatch{}, &model.LogArchiveProgress{}); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewLogArchiveRepository(db)
	svc := NewLogArchiveService(repo, LogArchiveConfig{RetentionDays: 90, MaxBatches: 1})
	p := &model.LogArchiveProgress{ExecutionID: 123, StartedAt: time.Now()}
	ctx := context.WithValue(context.Background(), archiveProgressKey{}, p)
	svc.setPhase(ctx, "verifying", &model.LogArchiveBatch{Base: models.Base{ID: 42}, RecordCount: 10000})
	another := NewLogArchiveService(repo, svc.config)
	got, err := another.Progress(context.Background(), 123)
	if err != nil || got == nil || got.Phase != "verifying" || got.BatchID != 42 || got.BatchRecords != 10000 {
		t.Fatalf("snapshot not durable: %+v %v", got, err)
	}
	missing, err := another.Progress(context.Background(), 456)
	if err != nil || missing != nil {
		t.Fatalf("missing snapshot: %+v %v", missing, err)
	}
	retry := time.Now().Add(time.Hour)
	batch := &model.LogArchiveBatch{ArchiveKey: "deferred", TenantUser: "alice", App: "ops", RangeStartedAt: time.Now(), RangeEndedAt: time.Now(), Status: model.LogArchiveStatusFailed, NextRetryAt: &retry, SelectedIDsJSON: json.RawMessage(`[]`)}
	if err := db.Create(batch).Error; err != nil {
		t.Fatal(err)
	}
	result, err := svc.Run(context.Background())
	if err != nil || result.StopReason != "retry_pending" {
		t.Fatalf("deferred work reported drained: %+v %v", result, err)
	}
}

func TestScheduledArchiveRoundTripAndSourceTableIsolation(t *testing.T) {
	db, svc, _ := archiveFixture(t)
	// Remove the fixture checkpoint, not its human operation rows.
	if err := db.Unscoped().Where("1=1").Delete(&model.LogArchiveBatch{}).Error; err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{"success", "failed"} {
		row := model.ScheduledExecutionLog{TenantUser: "alice", App: "crm", ResourcePath: "/alice/crm/task.form", ResourceType: "form", Status: status, SourceRef: "timer_task:1:execution:2", NewValuesJSON: json.RawMessage(`{"result":{"n":0},"error":"timeout"}`)}
		row.CreatedAt = models.Time(time.Now().AddDate(0, 0, -8))
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	batch, err := svc.nextBatch(context.Background(), time.Now().AddDate(0, 0, -90))
	if err != nil {
		t.Fatal(err)
	}
	if batch.ArchiveType != "scheduled_execution" || batch.RecordCount != 2 {
		t.Fatalf("wrong batch: %+v", batch)
	}
	path, summary, err := svc.exportBatch(context.Background(), batch)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	decoder := json.NewDecoder(gz)
	for range 2 {
		var row model.ScheduledExecutionLog
		if err := decoder.Decode(&row); err != nil {
			t.Fatal(err)
		}
		if string(row.NewValuesJSON) != `{"result":{"n":0},"error":"timeout"}` {
			t.Fatalf("raw result changed: %s", row.NewValuesJSON)
		}
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		t.Fatalf("unexpected trailing content: %v", err)
	}
	if !strings.Contains(string(summary), `"failed":1`) {
		t.Fatalf("missing failure count: %s", summary)
	}
	if err := svc.deleteArchivedSource(context.Background(), batch); err == nil {
		t.Fatal("deleted before verification")
	}
	now := time.Now()
	batch.ObjectVerifiedAt = &now
	batch.ObjectRef = "verified"
	batch.SHA256 = "test"
	if err := svc.deleteArchivedSource(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	var human, scheduled int64
	if err := db.Model(&model.OperateLog{}).Count(&human).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.ScheduledExecutionLog{}).Count(&scheduled).Error; err != nil {
		t.Fatal(err)
	}
	if human != 3 || scheduled != 0 {
		t.Fatalf("cross-table deletion: human=%d scheduled=%d", human, scheduled)
	}
}

func TestLegacyArchiveRetryReleasesUnselectedScheduledRowsForMigration(t *testing.T) {
	db, svc, batch := archiveFixture(t)
	batch.MaxLogID = 3 // The old batch owns the range, but selected only IDs 1 and 2.
	if err := svc.repo.Save(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.OperateLog{}).Where("id = 3").Updates(map[string]any{"resource_type": "form", "source": "scheduled_task"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.MoveLegacyScheduledLogs(context.Background()); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&model.ScheduledExecutionLog{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("migrated an owned range: %d %v", count, err)
	}
	if err := svc.Retry(context.Background(), batch.ID); err != nil {
		t.Fatal(err)
	}
	var row model.ScheduledExecutionLog
	if err := db.First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.OriginalLogID == nil || *row.OriginalLogID != 3 {
		t.Fatalf("unselected row not recovered: %+v", row)
	}
}
