package service

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/kageos/kageos/core/timer-scheduler/model"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestService(t *testing.T, now *time.Time) (*Service, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "timer.db")), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := model.InitTables(db); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, Options{
		QueueAckTimeout:        time.Minute,
		ExecutionLeaseDuration: 10 * time.Minute,
		MaxDispatchAttempts:    3,
		Now: func() time.Time {
			return *now
		},
	})
	return svc, db
}

func TestServiceDispatchAndFinishAtimeTask(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, &now)
	ctx := context.Background()
	payload := json.RawMessage(`{"hello":"timer"}`)

	task, err := svc.CreateTask(ctx, scheduledsdk.CreateTaskRequest{
		Title:           "demo",
		ExecutorKey:     "test.executor",
		ExecutorPayload: payload,
		Schedule:        scheduledsdk.At(now.Add(-time.Minute)),
		SourceType:      "test",
		ResourceScope:   "workspace",
		ResourceKey:     "root",
		RequestUser:     "system",
	})
	if err != nil {
		t.Fatal(err)
	}

	execs, err := svc.DispatchDue(ctx, "owner-1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(execs) != 1 {
		t.Fatalf("dispatched executions = %d, want 1", len(execs))
	}
	exec := execs[0]
	if exec.ExecutorKey != "test.executor" || exec.TraceID == "" {
		t.Fatalf("unexpected execution: %+v", exec)
	}

	var outboxCount int64
	if err := db.Model(&model.TimerOutboxEvent{}).Where("event_type = ?", eventTypeRequested).Count(&outboxCount).Error; err != nil {
		t.Fatal(err)
	}
	if outboxCount != 1 {
		t.Fatalf("requested outbox count = %d, want 1", outboxCount)
	}

	if err := svc.MarkExecutionStarted(ctx, scheduledsdk.MarkExecutionStartedRequest{
		TaskID:      task.ID,
		ExecutionID: exec.ID,
		WorkerID:    "worker-1",
		StartedAt:   now,
	}); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	if err := svc.MarkExecutionHeartbeat(ctx, scheduledsdk.MarkExecutionHeartbeatRequest{
		TaskID:      task.ID,
		ExecutionID: exec.ID,
		WorkerID:    "worker-1",
		HeartbeatAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.MarkExecutionFinished(ctx, scheduledsdk.MarkExecutionFinishedRequest{
		TaskID:        task.ID,
		ExecutionID:   exec.ID,
		Status:        scheduledsdk.ExecutionStatusSuccess,
		ExecutorRunID: "run-1",
		FinishedAt:    now,
	}); err != nil {
		t.Fatal(err)
	}

	gotTask, err := svc.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotTask.Status != scheduledsdk.TaskStatusDone {
		t.Fatalf("task status = %s, want done", gotTask.Status)
	}
	if gotTask.RunCount != 1 {
		t.Fatalf("task run_count = %d, want 1", gotTask.RunCount)
	}
	if gotTask.InflightExecutionID != 0 {
		t.Fatalf("task inflight = %d, want 0", gotTask.InflightExecutionID)
	}
}

func TestServiceRequeuesQueuedExecutionBeforeTimeout(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, &now)
	ctx := context.Background()

	task, err := svc.CreateTask(ctx, scheduledsdk.CreateTaskRequest{
		ExecutorKey: "test.executor",
		Schedule:    scheduledsdk.At(now.Add(-time.Second)),
	})
	if err != nil {
		t.Fatal(err)
	}
	execs, err := svc.DispatchDue(ctx, "owner-1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(execs) != 1 {
		t.Fatalf("dispatched executions = %d, want 1", len(execs))
	}

	now = now.Add(2 * time.Minute)
	recovered, err := svc.RecoverStaleExecutions(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if recovered != 1 {
		t.Fatalf("recovered = %d, want 1", recovered)
	}

	gotExec, err := svc.GetExecution(ctx, task.ID, execs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotExec.Status != scheduledsdk.ExecutionStatusQueued {
		t.Fatalf("execution status = %s, want queued", gotExec.Status)
	}
	if gotExec.Attempt != 2 {
		t.Fatalf("execution attempt = %d, want 2", gotExec.Attempt)
	}
	var outboxCount int64
	if err := db.Model(&model.TimerOutboxEvent{}).Where("event_type = ?", eventTypeRequested).Count(&outboxCount).Error; err != nil {
		t.Fatal(err)
	}
	if outboxCount != 2 {
		t.Fatalf("requested outbox count = %d, want 2", outboxCount)
	}
}

func TestRunNowDoesNotChangeScheduledCadence(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, &now)
	ctx := context.Background()

	task, err := svc.CreateTask(ctx, scheduledsdk.CreateTaskRequest{
		ExecutorKey: "test.executor",
		Schedule:    scheduledsdk.Every(3600),
	})
	if err != nil {
		t.Fatal(err)
	}
	originalNextRunAt := *task.NextRunAt

	exec, err := svc.RunNow(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if exec.TriggerType != triggerManual {
		t.Fatalf("trigger_type = %q, want manual", exec.TriggerType)
	}
	if err := svc.MarkExecutionStarted(ctx, scheduledsdk.MarkExecutionStartedRequest{
		TaskID:      task.ID,
		ExecutionID: exec.ID,
		WorkerID:    "worker-1",
		StartedAt:   now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.MarkExecutionFinished(ctx, scheduledsdk.MarkExecutionFinishedRequest{
		TaskID:      task.ID,
		ExecutionID: exec.ID,
		Status:      scheduledsdk.ExecutionStatusSuccess,
		FinishedAt:  now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}

	gotTask, err := svc.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotTask.RunCount != 0 {
		t.Fatalf("run_now changed run_count to %d, want 0", gotTask.RunCount)
	}
	if gotTask.NextRunAt == nil || !gotTask.NextRunAt.Equal(originalNextRunAt) {
		t.Fatalf("run_now changed next_run_at to %v, want %v", gotTask.NextRunAt, originalNextRunAt)
	}
	if gotTask.Status != scheduledsdk.TaskStatusPending {
		t.Fatalf("task status = %s, want pending", gotTask.Status)
	}
}

func TestPublishPendingOutboxRetriesBeforePublished(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, &now)
	ctx := context.Background()

	if _, err := svc.CreateTask(ctx, scheduledsdk.CreateTaskRequest{
		ExecutorKey: "test.executor",
		Schedule:    scheduledsdk.At(now.Add(-time.Second)),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DispatchDue(ctx, "owner-1", 10); err != nil {
		t.Fatal(err)
	}

	publisher := &testPublisher{fail: true}
	published, err := svc.PublishPendingOutbox(ctx, publisher, 10)
	if err != nil {
		t.Fatal(err)
	}
	if published != 0 {
		t.Fatalf("published = %d, want 0", published)
	}
	var event model.TimerOutboxEvent
	if err := db.Where("event_type = ?", eventTypeRequested).First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Status != "retry" || event.Attempts != 1 || event.NextAttemptAt == nil {
		t.Fatalf("unexpected retry event: %+v", event)
	}

	now = event.NextAttemptAt.Add(time.Millisecond)
	publisher.fail = false
	published, err = svc.PublishPendingOutbox(ctx, publisher, 10)
	if err != nil {
		t.Fatal(err)
	}
	if published != 1 {
		t.Fatalf("published = %d, want 1", published)
	}
	if err := db.First(&event, event.ID).Error; err != nil {
		t.Fatal(err)
	}
	if event.Status != "published" || publisher.subject == "" || len(publisher.payload) == 0 {
		t.Fatalf("unexpected published event=%+v publisher=%+v", event, publisher)
	}
}

type testPublisher struct {
	fail    bool
	subject string
	payload []byte
}

func (p *testPublisher) Publish(ctx context.Context, subject string, payload []byte) error {
	if p.fail {
		return fmt.Errorf("publish failed")
	}
	p.subject = subject
	p.payload = append([]byte(nil), payload...)
	return nil
}
