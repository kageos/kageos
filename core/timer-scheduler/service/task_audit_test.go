package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos/core/timer-scheduler/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/scheduledsdk"
)

func auditTaskRequest(now time.Time) scheduledsdk.CreateTaskRequest {
	return scheduledsdk.CreateTaskRequest{Title: "Audit task", ExecutorKey: "test.executor", ResourceScope: "function", ResourceKey: "/alice/ops/job.form", ExecutorPayload: json.RawMessage(`{"api_key":"must-not-leak"}`), Schedule: scheduledsdk.At(now.Add(time.Hour))}
}

func TestTaskAuditLifecycleAndIdempotency(t *testing.T) {
	now := time.Now()
	svc, db := newTestService(t, &now)
	ctx := contextx.WithRequestUser(context.Background(), "bob")
	req := auditTaskRequest(now)
	req.IdempotencyKey = "stable"
	task, err := svc.CreateTask(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateTask(ctx, req); err != nil {
		t.Fatal(err)
	}
	if err := svc.PauseTask(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.ResumeTask(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	title := "Renamed"
	if _, err := svc.UpdateTask(ctx, task.ID, scheduledsdk.UpdateTaskRequest{Title: &title}); err != nil {
		t.Fatal(err)
	}
	execution, err := svc.RunNow(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteTask(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	var rows []model.TimerOutboxEvent
	if err := db.Where("subject = ?", dto.TaskAuditSubject).Order("id").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 6 {
		t.Fatalf("events=%d, want 6 (idempotent create must not duplicate)", len(rows))
	}
	for _, row := range rows {
		if strings.Contains(string(row.Payload), "must-not-leak") {
			t.Fatal("payload secret leaked")
		}
		var event dto.TaskAuditEvent
		if err := json.Unmarshal(row.Payload, &event); err != nil {
			t.Fatal(err)
		}
		if event.Actor != "bob" || event.ResourcePath != req.ResourceKey {
			t.Fatalf("audit identity: %+v", event)
		}
		if event.Action == "timer.task.run_now" && event.ExecutionID != execution.ID {
			t.Fatal("manual execution link missing")
		}
		if event.Action == "timer.task.deleted" && (!strings.Contains(string(event.Before), "Renamed") || len(event.After) != 0) {
			t.Fatal("deleted snapshot lost")
		}
	}
}

func TestTaskMutationRollsBackWhenAuditCannotBeStored(t *testing.T) {
	now := time.Now()
	svc, db := newTestService(t, &now)
	task, err := svc.CreateTask(context.Background(), auditTaskRequest(now))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrator().DropTable(&model.TimerOutboxEvent{}); err != nil {
		t.Fatal(err)
	}
	if err := svc.PauseTask(context.Background(), task.ID); err == nil {
		t.Fatal("mutation should fail when its audit cannot commit")
	}
	got, err := svc.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != scheduledsdk.TaskStatusPending {
		t.Fatalf("unaudited mutation committed: %s", got.Status)
	}
}

func TestTaskAuditRetriesPastNormalOutboxLimit(t *testing.T) {
	now := time.Now()
	svc, db := newTestService(t, &now)
	if _, err := svc.CreateTask(context.Background(), auditTaskRequest(now)); err != nil {
		t.Fatal(err)
	}
	publisher := &testPublisher{fail: true}
	for i := 0; i < svc.opts.MaxOutboxAttempts+2; i++ {
		if _, err := svc.PublishPendingOutbox(context.Background(), publisher, 10); err != nil {
			t.Fatal(err)
		}
		now = now.Add(6 * time.Minute)
	}
	var row model.TimerOutboxEvent
	if err := db.Where("subject = ?", dto.TaskAuditSubject).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != "retry" {
		t.Fatalf("audit stopped retrying: %s", row.Status)
	}
	publisher.fail = false
	if _, err := svc.PublishPendingOutbox(context.Background(), publisher, 10); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&row, row.ID).Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != "published" {
		t.Fatal("recovered audit was not delivered")
	}
}

func TestTaskAuditRecordsRejectedUpdateWithoutAfterValues(t *testing.T) {
	now := time.Now()
	svc, db := newTestService(t, &now)
	task, err := svc.CreateTask(context.Background(), auditTaskRequest(now))
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.CancelTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	title := "Must not be saved"
	if _, err := svc.UpdateTask(context.Background(), task.ID, scheduledsdk.UpdateTaskRequest{Title: &title}); err == nil {
		t.Fatal("expected rejection")
	}
	var row model.TimerOutboxEvent
	if err := db.Where("subject = ?", dto.TaskAuditSubject).Order("id DESC").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	var event dto.TaskAuditEvent
	if err := json.Unmarshal(row.Payload, &event); err != nil {
		t.Fatal(err)
	}
	if event.Status != "failed" || len(event.After) > 0 {
		t.Fatalf("rejected state recorded as applied: %+v", event)
	}
}
