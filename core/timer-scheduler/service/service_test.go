package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/kageos/kageos/core/timer-scheduler/model"
	"github.com/kageos/kageos/pkg/contextx"
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
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		CompanyCode: "acme",
		CompanyName: "Acme",
	})
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
	var requestedEvent model.TimerOutboxEvent
	if err := db.Where("event_type = ?", eventTypeRequested).First(&requestedEvent).Error; err != nil {
		t.Fatal(err)
	}
	var event scheduledsdk.ExecutionRequestedEvent
	if err := json.Unmarshal(requestedEvent.Payload, &event); err != nil {
		t.Fatal(err)
	}
	if event.Metadata[scheduledsdk.MetadataCompanyCode] != "acme" || event.Metadata[scheduledsdk.MetadataCompanyName] != "Acme" {
		t.Fatalf("company metadata was not propagated: %+v", event.Metadata)
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

func TestRecoverStaleExecutionsHandlesOrphansAndContinues(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, &now)
	ctx := context.Background()
	staleLease := now.Add(-2 * time.Minute)

	orphanRequeue := &model.TimerExecution{
		TaskID:           999001,
		ExecutorKey:      "test.executor",
		Status:           string(scheduledsdk.ExecutionStatusQueued),
		TriggerType:      triggerScheduled,
		ScheduledAt:      staleLease,
		LeaseUntil:       &staleLease,
		Attempt:          1,
		LastDispatchedAt: &staleLease,
		TraceID:          "orphan-requeue",
	}
	orphanTimeout := &model.TimerExecution{
		TaskID:           999002,
		ExecutorKey:      "test.executor",
		Status:           string(scheduledsdk.ExecutionStatusQueued),
		TriggerType:      triggerScheduled,
		ScheduledAt:      staleLease,
		LeaseUntil:       &staleLease,
		Attempt:          svc.opts.MaxDispatchAttempts,
		LastDispatchedAt: &staleLease,
		TraceID:          "orphan-timeout",
	}
	if err := db.Create(orphanRequeue).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(orphanTimeout).Error; err != nil {
		t.Fatal(err)
	}

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
	if recovered != 3 {
		t.Fatalf("recovered = %d, want 3", recovered)
	}

	var gotOrphanRequeue model.TimerExecution
	if err := db.First(&gotOrphanRequeue, orphanRequeue.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotOrphanRequeue.Status != string(scheduledsdk.ExecutionStatusTimeout) || gotOrphanRequeue.LeaseUntil != nil {
		t.Fatalf("orphan requeue was not timed out: %+v", gotOrphanRequeue)
	}
	if gotOrphanRequeue.ErrorMessage != "timer-scheduler task not found during recovery" {
		t.Fatalf("orphan requeue error = %q", gotOrphanRequeue.ErrorMessage)
	}

	var gotOrphanTimeout model.TimerExecution
	if err := db.First(&gotOrphanTimeout, orphanTimeout.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotOrphanTimeout.Status != string(scheduledsdk.ExecutionStatusTimeout) || gotOrphanTimeout.LeaseUntil != nil {
		t.Fatalf("orphan timeout was not timed out: %+v", gotOrphanTimeout)
	}
	if gotOrphanTimeout.ErrorMessage != "timer-scheduler execution was not picked up before timeout" {
		t.Fatalf("orphan timeout error = %q", gotOrphanTimeout.ErrorMessage)
	}

	gotExec, err := svc.GetExecution(ctx, task.ID, execs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotExec.Status != scheduledsdk.ExecutionStatusQueued || gotExec.Attempt != 2 {
		t.Fatalf("valid stale execution was not requeued: %+v", gotExec)
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

func TestCreateTaskReusesActiveIdempotencyKey(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, &now)
	ctx := context.Background()

	first, err := svc.CreateTask(ctx, scheduledsdk.CreateTaskRequest{
		Title:          "first",
		IdempotencyKey: "same-live-task",
		ExecutorKey:    "test.executor",
		Schedule:       scheduledsdk.Every(60),
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.CreateTask(ctx, scheduledsdk.CreateTaskRequest{
		Title:          "second",
		IdempotencyKey: "same-live-task",
		ExecutorKey:    "test.executor",
		Schedule:       scheduledsdk.Every(120),
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID || second.Title != "first" {
		t.Fatalf("active idempotency should return first task, first=%+v second=%+v", first, second)
	}
}

func TestCreateTaskRecreatesAfterCancelledIdempotencyKey(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, &now)
	ctx := context.Background()
	key := "same-after-cancel"

	first, err := svc.CreateTask(ctx, scheduledsdk.CreateTaskRequest{
		Title:          "first",
		IdempotencyKey: key,
		ExecutorKey:    "test.executor",
		Schedule:       scheduledsdk.Every(60),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.CancelTask(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	second, err := svc.CreateTask(ctx, scheduledsdk.CreateTaskRequest{
		Title:          "second",
		IdempotencyKey: key,
		ExecutorKey:    "test.executor",
		Schedule:       scheduledsdk.Every(120),
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.ID == first.ID {
		t.Fatalf("cancelled idempotency should create a new task, got same id %d", second.ID)
	}
	if second.Status != scheduledsdk.TaskStatusPending || second.IdempotencyKey != key {
		t.Fatalf("unexpected recreated task: %+v", second)
	}

	var oldTask model.TimerTask
	if err := db.First(&oldTask, first.ID).Error; err != nil {
		t.Fatal(err)
	}
	if oldTask.IdempotencyKey == nil || *oldTask.IdempotencyKey == key {
		t.Fatalf("old terminal task should release original idempotency key, got %#v", oldTask.IdempotencyKey)
	}
}

func TestDeleteTaskSoftDeletesAndReleasesIdempotencyKey(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, &now)
	ctx := context.Background()
	key := "same-after-delete"

	first, err := svc.CreateTask(ctx, scheduledsdk.CreateTaskRequest{
		Title:          "first",
		IdempotencyKey: key,
		ExecutorKey:    "test.executor",
		Schedule:       scheduledsdk.Every(60),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteTask(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetTask(ctx, first.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("deleted task should be hidden, got err=%v", err)
	}
	resp, err := svc.ListTasks(ctx, scheduledsdk.ListTasksRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Total != 0 || len(resp.List) != 0 {
		t.Fatalf("deleted task should not be listed: %+v", resp)
	}

	second, err := svc.CreateTask(ctx, scheduledsdk.CreateTaskRequest{
		Title:          "second",
		IdempotencyKey: key,
		ExecutorKey:    "test.executor",
		Schedule:       scheduledsdk.Every(120),
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.ID == first.ID || second.IdempotencyKey != key {
		t.Fatalf("deleted idempotency should create new task with original key, first=%+v second=%+v", first, second)
	}

	var deleted model.TimerTask
	if err := db.Unscoped().First(&deleted, first.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !deleted.DeletedAt.Valid {
		t.Fatalf("task should be soft deleted: %+v", deleted)
	}
	if deleted.IdempotencyKey == nil || *deleted.IdempotencyKey == key {
		t.Fatalf("deleted task should release original idempotency key, got %#v", deleted.IdempotencyKey)
	}
}

func TestDeleteTaskRejectsInflightExecution(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, &now)
	ctx := context.Background()

	task, err := svc.CreateTask(ctx, scheduledsdk.CreateTaskRequest{
		ExecutorKey: "test.executor",
		Schedule:    scheduledsdk.Every(60),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RunNow(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteTask(ctx, task.ID); !errors.Is(err, ErrTaskBusy) {
		t.Fatalf("delete inflight task err=%v, want ErrTaskBusy", err)
	}
	if _, err := svc.GetTask(ctx, task.ID); err != nil {
		t.Fatalf("busy task should still exist: %v", err)
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
