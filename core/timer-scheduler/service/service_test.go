package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/timer-scheduler/model"
	"github.com/ai-agent-os/ai-agent-os/pkg/scheduledsdk"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestService(t *testing.T, now time.Time) (*Service, *gorm.DB) {
	t.Helper()
	return newTestServiceWithOptions(t, Options{Now: func() time.Time { return now }})
}

func newTestServiceWithOptions(t *testing.T, opts Options) (*Service, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := model.InitTables(db); err != nil {
		t.Fatalf("init tables: %v", err)
	}
	return NewService(db, opts), db
}

func TestCreateTaskUsesGenericExecutorPayload(t *testing.T) {
	now := time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, now)
	payload := json.RawMessage(`{"business_ref":"agent_task:123"}`)

	task, err := svc.CreateTask(context.Background(), scheduledsdk.CreateTaskRequest{
		Title:             "daily inspection",
		Category:          "inspection",
		Tags:              []string{"daily", "agent"},
		ExecutorKey:       "agent.session",
		ExecutorPayload:   payload,
		Metadata:          map[string]string{"team": "platform"},
		Schedule:          scheduledsdk.Every(3600),
		RequestUser:       "alice",
		NotifyUsers:       []string{"alice", " bob ", "alice"},
		NotifyDepartments: []string{" /org/platform ", "/org/platform/ops"},
	})
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	if task.ExecutorKey != "agent.session" {
		t.Fatalf("executor_key = %q, want agent.session", task.ExecutorKey)
	}
	if string(task.ExecutorPayload) != string(payload) {
		t.Fatalf("executor_payload = %s, want %s", task.ExecutorPayload, payload)
	}
	if got := task.Metadata["team"]; got != "platform" {
		t.Fatalf("metadata team = %q, want platform", got)
	}
	wantNext := now.Add(time.Hour)
	if task.NextRunAt == nil || !task.NextRunAt.Equal(wantNext) {
		t.Fatalf("next_run_at = %v, want %s", task.NextRunAt, wantNext)
	}

	var stored model.TimerTask
	if err := db.First(&stored, task.ID).Error; err != nil {
		t.Fatalf("load stored task: %v", err)
	}
	if stored.NotifyUsers != "alice,bob" {
		t.Fatalf("stored notify_users = %q, want alice,bob", stored.NotifyUsers)
	}
	if stored.NotifyDepartments != "/org/platform,/org/platform/ops" {
		t.Fatalf("stored notify_departments = %q", stored.NotifyDepartments)
	}
	if got := task.NotifyUsers; len(got) != 2 || got[0] != "alice" || got[1] != "bob" {
		t.Fatalf("task notify_users = %#v, want alice,bob", got)
	}
}

func TestDispatchDueCreatesQueuedExecutionAndOutbox(t *testing.T) {
	now := time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, now)
	task, err := svc.CreateTask(context.Background(), scheduledsdk.CreateTaskRequest{
		Title:           "once",
		ExecutorKey:     "app.function",
		ExecutorPayload: json.RawMessage(`{"business_ref":"form_task:7"}`),
		Schedule:        scheduledsdk.At(now),
		Metadata:        map[string]string{"app_code": "demo"},
	})
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}

	execs, err := svc.DispatchDue(context.Background(), "scheduler-1", 10)
	if err != nil {
		t.Fatalf("DispatchDue returned error: %v", err)
	}
	if len(execs) != 1 {
		t.Fatalf("dispatched executions = %d, want 1", len(execs))
	}
	if execs[0].Status != scheduledsdk.ExecutionStatusQueued {
		t.Fatalf("execution status = %q, want queued", execs[0].Status)
	}

	updated, err := svc.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("GetTask returned error: %v", err)
	}
	if updated.InflightExecutionID != execs[0].ID {
		t.Fatalf("inflight_execution_id = %d, want %d", updated.InflightExecutionID, execs[0].ID)
	}

	events, err := svc.ListPendingOutbox(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListPendingOutbox returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("outbox events = %d, want 1", len(events))
	}
	if events[0].EventType != scheduledsdk.SubjectExecutionRequested {
		t.Fatalf("event type = %q, want %q", events[0].EventType, scheduledsdk.SubjectExecutionRequested)
	}

	var event scheduledsdk.ExecutionRequestedEvent
	if err := json.Unmarshal(events[0].Payload, &event); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if event.ExecutorKey != "app.function" {
		t.Fatalf("event executor_key = %q, want app.function", event.ExecutorKey)
	}
	if event.Metadata["app_code"] != "demo" {
		t.Fatalf("event metadata app_code = %q, want demo", event.Metadata["app_code"])
	}
}

func TestMarkExecutionFinishedClearsInflightAndCompletesAtimeTask(t *testing.T) {
	now := time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, now)
	task, err := svc.CreateTask(context.Background(), scheduledsdk.CreateTaskRequest{
		Title:       "once",
		ExecutorKey: "app.function",
		Schedule:    scheduledsdk.At(now),
	})
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	execs, err := svc.DispatchDue(context.Background(), "scheduler-1", 10)
	if err != nil {
		t.Fatalf("DispatchDue returned error: %v", err)
	}
	exec := execs[0]

	if err := svc.MarkExecutionStarted(context.Background(), scheduledsdk.MarkExecutionStartedRequest{
		TaskID:        task.ID,
		ExecutionID:   exec.ID,
		WorkerID:      "worker-1",
		ExecutorRunID: "run-1",
		StartedAt:     now.Add(time.Second),
	}); err != nil {
		t.Fatalf("MarkExecutionStarted returned error: %v", err)
	}
	if err := svc.MarkExecutionFinished(context.Background(), scheduledsdk.MarkExecutionFinishedRequest{
		TaskID:         task.ID,
		ExecutionID:    exec.ID,
		Status:         scheduledsdk.ExecutionStatusSuccess,
		FinishedAt:     now.Add(2 * time.Second),
		DurationMillis: 1000,
		OutputSummary:  "ok",
	}); err != nil {
		t.Fatalf("MarkExecutionFinished returned error: %v", err)
	}

	updated, err := svc.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("GetTask returned error: %v", err)
	}
	if updated.Status != scheduledsdk.TaskStatusDone {
		t.Fatalf("task status = %q, want done", updated.Status)
	}
	if updated.InflightExecutionID != 0 {
		t.Fatalf("inflight_execution_id = %d, want 0", updated.InflightExecutionID)
	}
	if updated.RunCount != 1 {
		t.Fatalf("run_count = %d, want 1", updated.RunCount)
	}
}

func TestRecoverStaleQueuedExecutionClearsInflight(t *testing.T) {
	current := time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)
	svc, _ := newTestServiceWithOptions(t, Options{
		Now:                    func() time.Time { return current },
		ExecutionLeaseDuration: time.Minute,
	})
	task, err := svc.CreateTask(context.Background(), scheduledsdk.CreateTaskRequest{
		Title:       "stale queued",
		ExecutorKey: "agent.session",
		Schedule:    scheduledsdk.Every(60),
	})
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	current = current.Add(time.Minute)
	execs, err := svc.DispatchDue(context.Background(), "scheduler-1", 10)
	if err != nil {
		t.Fatalf("DispatchDue returned error: %v", err)
	}
	if len(execs) != 1 {
		t.Fatalf("dispatched executions = %d, want 1", len(execs))
	}

	current = current.Add(2 * time.Minute)
	recovered, err := svc.RecoverStaleExecutions(context.Background(), 10)
	if err != nil {
		t.Fatalf("RecoverStaleExecutions returned error: %v", err)
	}
	if recovered != 1 {
		t.Fatalf("recovered = %d, want 1", recovered)
	}
	updated, err := svc.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("GetTask returned error: %v", err)
	}
	if updated.InflightExecutionID != 0 {
		t.Fatalf("inflight_execution_id = %d, want 0", updated.InflightExecutionID)
	}
	if updated.Status != scheduledsdk.TaskStatusPending {
		t.Fatalf("task status = %q, want pending", updated.Status)
	}
	if updated.RunCount != 1 {
		t.Fatalf("run_count = %d, want 1", updated.RunCount)
	}
	exec, err := svc.GetExecution(context.Background(), task.ID, execs[0].ID)
	if err != nil {
		t.Fatalf("GetExecution returned error: %v", err)
	}
	if exec.Status != scheduledsdk.ExecutionStatusTimeout {
		t.Fatalf("execution status = %q, want timeout", exec.Status)
	}
}

func TestRecoverStaleRunningExecutionClearsInflight(t *testing.T) {
	current := time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)
	svc, _ := newTestServiceWithOptions(t, Options{
		Now:                    func() time.Time { return current },
		ExecutionLeaseDuration: time.Minute,
	})
	task, err := svc.CreateTask(context.Background(), scheduledsdk.CreateTaskRequest{
		Title:       "stale running",
		ExecutorKey: "app.function",
		Schedule:    scheduledsdk.Every(60),
	})
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	current = current.Add(time.Minute)
	execs, err := svc.DispatchDue(context.Background(), "scheduler-1", 10)
	if err != nil {
		t.Fatalf("DispatchDue returned error: %v", err)
	}
	exec := execs[0]
	if err := svc.MarkExecutionStarted(context.Background(), scheduledsdk.MarkExecutionStartedRequest{
		TaskID:      task.ID,
		ExecutionID: exec.ID,
		WorkerID:    "worker-1",
		StartedAt:   current,
	}); err != nil {
		t.Fatalf("MarkExecutionStarted returned error: %v", err)
	}

	current = current.Add(2 * time.Minute)
	recovered, err := svc.RecoverStaleExecutions(context.Background(), 10)
	if err != nil {
		t.Fatalf("RecoverStaleExecutions returned error: %v", err)
	}
	if recovered != 1 {
		t.Fatalf("recovered = %d, want 1", recovered)
	}
	updated, err := svc.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("GetTask returned error: %v", err)
	}
	if updated.InflightExecutionID != 0 || updated.RunCount != 1 {
		t.Fatalf("task inflight/run_count = %d/%d, want 0/1", updated.InflightExecutionID, updated.RunCount)
	}
	timedOut, err := svc.GetExecution(context.Background(), task.ID, exec.ID)
	if err != nil {
		t.Fatalf("GetExecution returned error: %v", err)
	}
	if timedOut.Status != scheduledsdk.ExecutionStatusTimeout {
		t.Fatalf("execution status = %q, want timeout", timedOut.Status)
	}
}

type fakeOutboxPublisher struct {
	fail     bool
	subjects []string
	payloads [][]byte
}

func (p *fakeOutboxPublisher) Publish(_ context.Context, subject string, payload []byte) error {
	if p.fail {
		return fmt.Errorf("publish failed")
	}
	p.subjects = append(p.subjects, subject)
	p.payloads = append(p.payloads, append([]byte(nil), payload...))
	return nil
}

func TestPublishPendingOutboxMarksPublished(t *testing.T) {
	now := time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, now)
	if _, err := svc.CreateTask(context.Background(), scheduledsdk.CreateTaskRequest{
		Title:       "once",
		ExecutorKey: "app.function",
		Schedule:    scheduledsdk.At(now),
	}); err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	execs, err := svc.DispatchDue(context.Background(), "scheduler-1", 10)
	if err != nil {
		t.Fatalf("DispatchDue returned error: %v", err)
	}
	if len(execs) != 1 {
		t.Fatalf("dispatched executions = %d, want 1", len(execs))
	}

	publisher := &fakeOutboxPublisher{}
	count, err := svc.PublishPendingOutbox(context.Background(), publisher, 10)
	if err != nil {
		t.Fatalf("PublishPendingOutbox returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("published count = %d, want 1", count)
	}
	if len(publisher.subjects) != 1 || publisher.subjects[0] != scheduledsdk.SubjectExecutionRequested {
		t.Fatalf("published subjects = %#v", publisher.subjects)
	}

	var events []model.TimerOutboxEvent
	if err := db.Find(&events).Error; err != nil {
		t.Fatalf("load outbox: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("outbox events = %d, want 1", len(events))
	}
	if events[0].Status != "published" || events[0].PublishedAt == nil {
		t.Fatalf("outbox status=%q published_at=%v, want published with timestamp", events[0].Status, events[0].PublishedAt)
	}
	if events[0].AggregateID != execs[0].ID {
		t.Fatalf("aggregate_id = %d, want execution id %d", events[0].AggregateID, execs[0].ID)
	}
}
