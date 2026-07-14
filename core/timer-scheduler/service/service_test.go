package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos/core/timer-scheduler/model"
	"github.com/kageos/kageos/pkg/auth"
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

func createTestTask(svc *Service, ctx context.Context, req scheduledsdk.CreateTaskRequest) (*scheduledsdk.Task, error) {
	return svc.CreateTask(ctx, req)
}

func TestServiceCreateTaskSupportsPausedInitialStatus(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, &now)

	task, err := createTestTask(svc, context.Background(), scheduledsdk.CreateTaskRequest{
		Title:           "paused demo",
		ExecutorKey:     "test.executor",
		ExecutorPayload: json.RawMessage(`{"hello":"timer"}`),
		Status:          scheduledsdk.TaskStatusPaused,
		Schedule:        scheduledsdk.At(now.Add(-time.Minute)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != scheduledsdk.TaskStatusPaused {
		t.Fatalf("status = %q, want %q", task.Status, scheduledsdk.TaskStatusPaused)
	}
	execs, err := svc.DispatchDue(context.Background(), "owner-1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(execs) != 0 {
		t.Fatalf("paused task should not dispatch, got %d executions", len(execs))
	}
}

func TestTaskMetadataSerializedSizeIsEnforcedOnCreateAndUpdate(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, &now)
	overhead := len(`{"k":""}`)
	exact := map[string]string{"k": strings.Repeat("a", scheduledsdk.MaxExecutionMetadataJSONBytes-overhead)}
	encoded, err := json.Marshal(exact)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) != scheduledsdk.MaxExecutionMetadataJSONBytes {
		t.Fatalf("boundary metadata encoded size = %d", len(encoded))
	}
	task, err := createTestTask(svc, context.Background(), scheduledsdk.CreateTaskRequest{
		ExecutorKey: "test.executor",
		Schedule:    scheduledsdk.Every(60),
		Metadata:    exact,
	})
	if err != nil {
		t.Fatalf("exact metadata boundary rejected: %v", err)
	}

	tooLarge := map[string]string{"k": exact["k"] + "a"}
	if _, err := createTestTask(svc, context.Background(), scheduledsdk.CreateTaskRequest{
		ExecutorKey: "test.executor",
		Schedule:    scheduledsdk.Every(60),
		Metadata:    tooLarge,
	}); !errors.Is(err, scheduledsdk.ErrInvalidRequest) {
		t.Fatalf("oversized create metadata error = %v, want ErrInvalidRequest", err)
	}
	if _, err := svc.UpdateTask(context.Background(), task.ID, scheduledsdk.UpdateTaskRequest{Metadata: &tooLarge}); !errors.Is(err, scheduledsdk.ErrInvalidRequest) {
		t.Fatalf("oversized update metadata error = %v, want ErrInvalidRequest", err)
	}
	escaped := map[string]string{"k": strings.Repeat("\x00", scheduledsdk.MaxExecutionMetadataJSONBytes/6+1)}
	if _, err := createTestTask(svc, context.Background(), scheduledsdk.CreateTaskRequest{
		ExecutorKey: "test.executor",
		Schedule:    scheduledsdk.Every(60),
		Metadata:    escaped,
	}); !errors.Is(err, scheduledsdk.ErrInvalidRequest) {
		t.Fatalf("JSON-escaped oversized metadata error = %v, want ErrInvalidRequest", err)
	}
}

func TestMarkExecutionStartedRequiresActiveTaskAndIsIdempotent(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	t.Run("exact duplicate is idempotent", func(t *testing.T) {
		svc, _ := newTestService(t, &now)
		task, err := createTestTask(svc, context.Background(), scheduledsdk.CreateTaskRequest{
			ExecutorKey: "test.executor",
			Schedule:    scheduledsdk.At(now.Add(-time.Minute)),
		})
		if err != nil {
			t.Fatal(err)
		}
		executions, err := svc.DispatchDue(context.Background(), "owner", 1)
		if err != nil || len(executions) != 1 {
			t.Fatalf("DispatchDue executions=%d err=%v", len(executions), err)
		}
		req := scheduledsdk.MarkExecutionStartedRequest{
			TaskID:        task.ID,
			ExecutionID:   executions[0].ID,
			WorkerID:      "worker-1",
			ExecutorRunID: "run-1",
			StartedAt:     now,
		}
		if err := svc.MarkExecutionStarted(context.Background(), req); err != nil {
			t.Fatal(err)
		}
		if err := svc.MarkExecutionHeartbeat(context.Background(), scheduledsdk.MarkExecutionHeartbeatRequest{
			TaskID: task.ID, ExecutionID: executions[0].ID, WorkerID: "worker-1", ExecutorRunID: "wrong-run", HeartbeatAt: now.Add(time.Second),
		}); !errors.Is(err, ErrInvalidTaskStatus) {
			t.Fatalf("wrong-run heartbeat error = %v, want ErrInvalidTaskStatus", err)
		}
		if err := svc.MarkExecutionFinished(context.Background(), scheduledsdk.MarkExecutionFinishedRequest{
			TaskID: task.ID, ExecutionID: executions[0].ID, Status: scheduledsdk.ExecutionStatusSuccess, ExecutorRunID: "wrong-run", FinishedAt: now.Add(time.Second),
		}); !errors.Is(err, ErrInvalidTaskStatus) {
			t.Fatalf("wrong-run finish error = %v, want ErrInvalidTaskStatus", err)
		}
		if err := svc.MarkExecutionStarted(context.Background(), req); err != nil {
			t.Fatalf("duplicate start error = %v", err)
		}
		req.ExecutorRunID = "different-run"
		if err := svc.MarkExecutionStarted(context.Background(), req); !errors.Is(err, ErrInvalidTaskStatus) {
			t.Fatalf("different run start error = %v, want ErrInvalidTaskStatus", err)
		}
	})

	for _, action := range []string{"cancel", "delete"} {
		t.Run(action+" blocks queued start", func(t *testing.T) {
			svc, _ := newTestService(t, &now)
			task, err := createTestTask(svc, context.Background(), scheduledsdk.CreateTaskRequest{
				ExecutorKey: "test.executor",
				Schedule:    scheduledsdk.At(now.Add(-time.Minute)),
			})
			if err != nil {
				t.Fatal(err)
			}
			executions, err := svc.DispatchDue(context.Background(), "owner", 1)
			if err != nil || len(executions) != 1 {
				t.Fatalf("DispatchDue executions=%d err=%v", len(executions), err)
			}
			if action == "cancel" {
				err = svc.CancelTask(context.Background(), task.ID)
			} else {
				err = svc.DeleteTask(context.Background(), task.ID)
			}
			if err != nil {
				t.Fatal(err)
			}
			err = svc.MarkExecutionStarted(context.Background(), scheduledsdk.MarkExecutionStartedRequest{
				TaskID:        task.ID,
				ExecutionID:   executions[0].ID,
				WorkerID:      "worker-1",
				ExecutorRunID: "run-1",
				StartedAt:     now,
			})
			if !errors.Is(err, ErrInvalidTaskStatus) {
				t.Fatalf("start after %s error = %v, want ErrInvalidTaskStatus", action, err)
			}
			stored, err := svc.GetExecution(context.Background(), task.ID, executions[0].ID)
			if err != nil {
				t.Fatal(err)
			}
			if stored.Status != scheduledsdk.ExecutionStatusCancelled {
				t.Fatalf("execution after %s status = %s, want cancelled", action, stored.Status)
			}
		})
	}
}

func TestMarkExecutionFinishedPreservesPausedAndCancelledTaskState(t *testing.T) {
	for _, status := range []scheduledsdk.TaskStatus{scheduledsdk.TaskStatusPaused, scheduledsdk.TaskStatusCancelled} {
		t.Run(string(status), func(t *testing.T) {
			now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
			svc, _ := newTestService(t, &now)
			task, err := createTestTask(svc, context.Background(), scheduledsdk.CreateTaskRequest{
				ExecutorKey: "test.executor",
				Schedule:    scheduledsdk.At(now.Add(-time.Minute)),
			})
			if err != nil {
				t.Fatal(err)
			}
			executions, err := svc.DispatchDue(context.Background(), "owner", 1)
			if err != nil || len(executions) != 1 {
				t.Fatalf("DispatchDue executions=%d err=%v", len(executions), err)
			}
			if err := svc.MarkExecutionStarted(context.Background(), scheduledsdk.MarkExecutionStartedRequest{
				TaskID: task.ID, ExecutionID: executions[0].ID, WorkerID: "worker", ExecutorRunID: "run", StartedAt: now,
			}); err != nil {
				t.Fatal(err)
			}
			if status == scheduledsdk.TaskStatusPaused {
				err = svc.PauseTask(context.Background(), task.ID)
			} else {
				err = svc.CancelTask(context.Background(), task.ID)
			}
			if err != nil {
				t.Fatal(err)
			}
			finishErr := svc.MarkExecutionFinished(context.Background(), scheduledsdk.MarkExecutionFinishedRequest{
				TaskID: task.ID, ExecutionID: executions[0].ID, Status: scheduledsdk.ExecutionStatusSuccess, ExecutorRunID: "run", FinishedAt: now.Add(time.Second),
			})
			if status == scheduledsdk.TaskStatusPaused && finishErr != nil {
				t.Fatal(finishErr)
			}
			if status == scheduledsdk.TaskStatusCancelled && !errors.Is(finishErr, ErrInvalidTaskStatus) {
				t.Fatalf("finish after cancel error = %v, want ErrInvalidTaskStatus", finishErr)
			}
			got, err := svc.GetTask(context.Background(), task.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != status {
				t.Fatalf("task status = %s, want %s", got.Status, status)
			}
			if status == scheduledsdk.TaskStatusCancelled && got.NextRunAt != nil {
				t.Fatalf("cancelled task next_run_at = %v, want nil", got.NextRunAt)
			}
		})
	}
}

func TestScheduledCompletionSettlesWithoutClearingNewerManualInflight(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, &now)
	task, err := createTestTask(svc, context.Background(), scheduledsdk.CreateTaskRequest{
		ExecutorKey: "test.executor",
		Schedule:    scheduledsdk.At(now.Add(-time.Minute)),
	})
	if err != nil {
		t.Fatal(err)
	}
	scheduled, err := svc.DispatchDue(context.Background(), "owner", 1)
	if err != nil || len(scheduled) != 1 {
		t.Fatalf("DispatchDue executions=%d err=%v", len(scheduled), err)
	}
	manual, err := svc.RunNow(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.MarkExecutionStarted(context.Background(), scheduledsdk.MarkExecutionStartedRequest{
		TaskID: task.ID, ExecutionID: scheduled[0].ID, WorkerID: "worker", ExecutorRunID: "scheduled-run", StartedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.MarkExecutionFinished(context.Background(), scheduledsdk.MarkExecutionFinishedRequest{
		TaskID: task.ID, ExecutionID: scheduled[0].ID, Status: scheduledsdk.ExecutionStatusSuccess, ExecutorRunID: "scheduled-run", FinishedAt: now.Add(time.Second),
	}); err != nil {
		t.Fatal(err)
	}
	settled, err := svc.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if settled.Status != scheduledsdk.TaskStatusDone || settled.RunCount != 1 || settled.InflightExecutionID != manual.ID {
		t.Fatalf("scheduled settlement lost state or newer inflight: %#v", settled)
	}
	if err := svc.MarkExecutionStarted(context.Background(), scheduledsdk.MarkExecutionStartedRequest{
		TaskID: task.ID, ExecutionID: manual.ID, WorkerID: "worker", ExecutorRunID: "manual-run", StartedAt: now.Add(time.Second),
	}); err != nil {
		t.Fatalf("queued manual start after scheduled task became done: %v", err)
	}
	if err := svc.MarkExecutionFinished(context.Background(), scheduledsdk.MarkExecutionFinishedRequest{
		TaskID: task.ID, ExecutionID: manual.ID, Status: scheduledsdk.ExecutionStatusSuccess, ExecutorRunID: "manual-run", FinishedAt: now.Add(2 * time.Second),
	}); err != nil {
		t.Fatal(err)
	}
	finished, err := svc.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != scheduledsdk.TaskStatusDone || finished.RunCount != 1 || finished.InflightExecutionID != 0 {
		t.Fatalf("manual completion changed scheduled settlement: %#v", finished)
	}
}

func TestServiceDispatchAndFinishAtimeTask(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, &now)
	createToken, err := auth.NewJWTService().GenerateAccessTokenWithContext(auth.UserTokenContext{
		UserID:             42,
		Username:           "alice",
		Email:              "alice@example.com",
		CompanyCode:        "acme",
		CompanyName:        "Acme",
		DepartmentFullPath: "/org/dev",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		Token:              createToken,
		RequestUser:        "alice",
		DepartmentFullPath: "/org/dev",
		CompanyCode:        "acme",
		CompanyName:        "Acme",
	})
	payload := json.RawMessage(`{"hello":"timer"}`)

	task, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
		Title:           "demo",
		ExecutorKey:     "test.executor",
		ExecutorPayload: payload,
		Schedule:        scheduledsdk.At(now.Add(-time.Minute)),
		SourceType:      "test",
		ResourceScope:   "workspace",
		ResourceKey:     "root",
		RequestUser:     "alice",
		RequestUserDept: "/org/dev",
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
	if event.RequestUser != "alice" || event.RequestUserDept != "/org/dev" {
		t.Fatalf("request user was not propagated: user=%q dept=%q", event.RequestUser, event.RequestUserDept)
	}
	if event.Token != "" || bytes.Contains(requestedEvent.Payload, []byte(`"token"`)) || bytes.Contains(requestedEvent.Payload, []byte("eyJ")) {
		t.Fatalf("scheduled execution event leaked JWT material: %s", requestedEvent.Payload)
	}
	if event.Metadata[scheduledsdk.MetadataRequestEmail] != "alice@example.com" {
		t.Fatalf("signed execution identity metadata was not preserved: %+v", event.Metadata)
	}
	if event.Metadata[scheduledsdk.MetadataRequestUserID] != "42" {
		t.Fatalf("signed execution user id was not preserved: %+v", event.Metadata)
	}
	if event.Metadata["task_title"] != "demo" {
		t.Fatalf("task title metadata was not propagated: %+v", event.Metadata)
	}

	if err := svc.MarkExecutionStarted(ctx, scheduledsdk.MarkExecutionStartedRequest{
		TaskID:        task.ID,
		ExecutionID:   exec.ID,
		WorkerID:      "worker-1",
		ExecutorRunID: "run-1",
		StartedAt:     now,
	}); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	if err := svc.MarkExecutionHeartbeat(ctx, scheduledsdk.MarkExecutionHeartbeatRequest{
		TaskID:        task.ID,
		ExecutionID:   exec.ID,
		WorkerID:      "worker-1",
		ExecutorRunID: "run-1",
		HeartbeatAt:   now,
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

func TestDispatchDueContinuesAfterOneTaskDispatchError(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, &now)
	ctx := context.Background()

	first, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
		Title:       "broken first",
		ExecutorKey: "test.executor",
		Schedule:    scheduledsdk.At(now.Add(-time.Second)),
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
		Title:       "healthy second",
		ExecutorKey: "test.executor",
		Schedule:    scheduledsdk.At(now.Add(-time.Second)),
	})
	if err != nil {
		t.Fatal(err)
	}
	triggerName := fmt.Sprintf("fail_timer_execution_task_%d", first.ID)
	if err := db.Exec(fmt.Sprintf(
		`CREATE TRIGGER %s BEFORE INSERT ON timer_execution WHEN NEW.task_id = %d BEGIN SELECT RAISE(ABORT, 'synthetic insert failure'); END`,
		triggerName,
		first.ID,
	)).Error; err != nil {
		t.Fatal(err)
	}

	execs, err := svc.DispatchDue(ctx, "owner-1", 10)
	if err == nil || !strings.Contains(err.Error(), "synthetic insert failure") {
		t.Fatalf("DispatchDue err=%v, want synthetic task error", err)
	}
	if len(execs) != 1 {
		t.Fatalf("dispatched executions = %d, want 1", len(execs))
	}
	if execs[0].TaskID != second.ID {
		t.Fatalf("dispatched task_id = %d, want healthy second task %d", execs[0].TaskID, second.ID)
	}
	gotSecond, err := svc.GetTask(ctx, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotSecond.InflightExecutionID != execs[0].ID {
		t.Fatalf("second task inflight = %d, want execution %d", gotSecond.InflightExecutionID, execs[0].ID)
	}
}

func TestDispatchDueRunsMissedTaskEvenWithInflightExecution(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, &now)
	ctx := context.Background()

	task, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
		ExecutorKey: "test.executor",
		Schedule:    scheduledsdk.Every(60),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.TimerTask{}).Where("id = ?", task.ID).Update("next_run_at", now.Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	first, err := svc.DispatchDue(ctx, "owner-1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 {
		t.Fatalf("first dispatch executions = %d, want 1", len(first))
	}

	now = now.Add(2 * time.Minute)
	second, err := svc.DispatchDue(ctx, "owner-1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 1 {
		t.Fatalf("second dispatch executions = %d, want 1", len(second))
	}
	if second[0].ID == first[0].ID {
		t.Fatalf("missed task should create a new execution, got same id %d", second[0].ID)
	}
	gotTask, err := svc.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotTask.InflightExecutionID != second[0].ID {
		t.Fatalf("task inflight = %d, want latest scheduled execution %d", gotTask.InflightExecutionID, second[0].ID)
	}
	if gotTask.NextRunAt == nil || !gotTask.NextRunAt.After(now) {
		t.Fatalf("next_run_at = %v, want advanced after %v", gotTask.NextRunAt, now)
	}
}

func TestServiceRequeuesQueuedExecutionBeforeTimeout(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, &now)
	ctx := context.Background()

	task, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
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

func TestRecoverStaleExecutionsTimesOutDetachedQueuedExecution(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, &now)
	svc.opts.MaxDispatchAttempts = 1
	ctx := context.Background()

	task, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
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
	exec := execs[0]
	if err := db.Model(&model.TimerTask{}).Where("id = ?", task.ID).Updates(map[string]interface{}{
		"inflight_execution_id": 0,
		"lease_owner":           "",
		"lease_until":           nil,
	}).Error; err != nil {
		t.Fatal(err)
	}

	now = now.Add(2 * time.Minute)
	recovered, err := svc.RecoverStaleExecutions(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if recovered != 1 {
		t.Fatalf("recovered = %d, want 1", recovered)
	}

	gotExec, err := svc.GetExecution(ctx, task.ID, exec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotExec.Status != scheduledsdk.ExecutionStatusTimeout {
		t.Fatalf("detached execution status = %s, want timeout", gotExec.Status)
	}
	gotTask, err := svc.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotTask.InflightExecutionID != 0 {
		t.Fatalf("task inflight = %d, want 0", gotTask.InflightExecutionID)
	}
}

func TestRecoverStaleRunningExecutionExtendsLeaseBeforeTimeout(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, &now)
	svc.opts.ExecutionLeaseDuration = time.Minute
	svc.opts.MaxHeartbeatMisses = 3
	ctx := context.Background()

	task, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
		ExecutorKey: "test.executor",
		Schedule:    scheduledsdk.Every(3600),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.TimerTask{}).Where("id = ?", task.ID).Update("next_run_at", now.Add(-time.Second)).Error; err != nil {
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
	if err := svc.MarkExecutionStarted(ctx, scheduledsdk.MarkExecutionStartedRequest{
		TaskID:        task.ID,
		ExecutionID:   exec.ID,
		WorkerID:      "worker-1",
		ExecutorRunID: "run-1",
		StartedAt:     now,
	}); err != nil {
		t.Fatal(err)
	}

	now = now.Add(2 * time.Minute)
	recovered, err := svc.RecoverStaleExecutions(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if recovered != 1 {
		t.Fatalf("recovered = %d, want 1", recovered)
	}

	var gotExec model.TimerExecution
	if err := db.First(&gotExec, exec.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotExec.Status != string(scheduledsdk.ExecutionStatusRunning) {
		t.Fatalf("execution status = %s, want running", gotExec.Status)
	}
	if gotExec.HeartbeatMisses != 1 {
		t.Fatalf("heartbeat_misses = %d, want 1", gotExec.HeartbeatMisses)
	}
	if gotExec.LeaseUntil == nil || !gotExec.LeaseUntil.Equal(now.Add(time.Minute)) {
		t.Fatalf("lease_until = %v, want %v", gotExec.LeaseUntil, now.Add(time.Minute))
	}

	gotTask, err := svc.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotTask.InflightExecutionID != exec.ID {
		t.Fatalf("task inflight = %d, want %d", gotTask.InflightExecutionID, exec.ID)
	}

	now = now.Add(30 * time.Second)
	if err := svc.MarkExecutionHeartbeat(ctx, scheduledsdk.MarkExecutionHeartbeatRequest{
		TaskID:        task.ID,
		ExecutionID:   exec.ID,
		WorkerID:      "worker-1",
		ExecutorRunID: "run-1",
		HeartbeatAt:   now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&gotExec, exec.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotExec.HeartbeatMisses != 0 {
		t.Fatalf("heartbeat_misses after heartbeat = %d, want 0", gotExec.HeartbeatMisses)
	}
}

func TestRecoverStaleRunningExecutionTimesOutAfterHeartbeatMissLimit(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, &now)
	svc.opts.ExecutionLeaseDuration = time.Minute
	svc.opts.MaxHeartbeatMisses = 2
	ctx := context.Background()

	task, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
		ExecutorKey: "test.executor",
		Schedule:    scheduledsdk.Every(3600),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.TimerTask{}).Where("id = ?", task.ID).Update("next_run_at", now.Add(-time.Second)).Error; err != nil {
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
	if err := svc.MarkExecutionStarted(ctx, scheduledsdk.MarkExecutionStartedRequest{
		TaskID:        task.ID,
		ExecutionID:   exec.ID,
		WorkerID:      "worker-1",
		ExecutorRunID: "run-1",
		StartedAt:     now,
	}); err != nil {
		t.Fatal(err)
	}

	now = now.Add(2 * time.Minute)
	recovered, err := svc.RecoverStaleExecutions(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if recovered != 1 {
		t.Fatalf("first recovered = %d, want 1", recovered)
	}

	var gotExec model.TimerExecution
	if err := db.First(&gotExec, exec.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotExec.Status != string(scheduledsdk.ExecutionStatusRunning) || gotExec.HeartbeatMisses != 1 {
		t.Fatalf("execution after first miss = status %s misses %d, want running/1", gotExec.Status, gotExec.HeartbeatMisses)
	}

	now = now.Add(2 * time.Minute)
	recovered, err = svc.RecoverStaleExecutions(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if recovered != 1 {
		t.Fatalf("second recovered = %d, want 1", recovered)
	}
	gotSDKExec, err := svc.GetExecution(ctx, task.ID, exec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotSDKExec.Status != scheduledsdk.ExecutionStatusTimeout {
		t.Fatalf("execution status = %s, want timeout", gotSDKExec.Status)
	}
	if gotSDKExec.ErrorMessage != "timer-scheduler execution heartbeat expired" {
		t.Fatalf("error_message = %q", gotSDKExec.ErrorMessage)
	}
	gotTask, err := svc.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotTask.InflightExecutionID != 0 {
		t.Fatalf("task inflight = %d, want 0", gotTask.InflightExecutionID)
	}
	if gotTask.Status != scheduledsdk.TaskStatusPending {
		t.Fatalf("task status = %s, want pending", gotTask.Status)
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

	task, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
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

func TestRecoverStaleExecutionsClearsMissingInflightReference(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, &now)
	ctx := context.Background()

	task, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
		ExecutorKey: "test.executor",
		Schedule:    scheduledsdk.Every(3600),
	})
	if err != nil {
		t.Fatal(err)
	}
	const missingExecutionID int64 = 12345
	if err := db.Model(&model.TimerTask{}).
		Where("id = ?", task.ID).
		Updates(map[string]interface{}{
			"inflight_execution_id": missingExecutionID,
			"last_execution_id":     missingExecutionID,
		}).Error; err != nil {
		t.Fatal(err)
	}

	recovered, err := svc.RecoverStaleExecutions(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if recovered != 1 {
		t.Fatalf("recovered = %d, want 1", recovered)
	}
	gotTask, err := svc.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotTask.InflightExecutionID != 0 {
		t.Fatalf("task inflight = %d, want 0", gotTask.InflightExecutionID)
	}
	if !strings.Contains(gotTask.LastErrorMessage, "stale inflight execution reference") {
		t.Fatalf("last error = %q, want stale inflight message", gotTask.LastErrorMessage)
	}
}

func TestRecoverStaleExecutionsClearsTerminalInflightReference(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, &now)
	ctx := context.Background()

	task, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
		ExecutorKey: "test.executor",
		Schedule:    scheduledsdk.Every(3600),
	})
	if err != nil {
		t.Fatal(err)
	}
	exec := &model.TimerExecution{
		TaskID:      task.ID,
		ExecutorKey: "test.executor",
		Status:      string(scheduledsdk.ExecutionStatusFailed),
		TriggerType: triggerManual,
		ScheduledAt: now,
		FinishedAt:  &now,
		Attempt:     1,
		TraceID:     "terminal-inflight",
	}
	if err := db.Create(exec).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.TimerTask{}).
		Where("id = ?", task.ID).
		Updates(map[string]interface{}{
			"inflight_execution_id": exec.ID,
			"last_execution_id":     exec.ID,
		}).Error; err != nil {
		t.Fatal(err)
	}

	recovered, err := svc.RecoverStaleExecutions(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if recovered != 1 {
		t.Fatalf("recovered = %d, want 1", recovered)
	}
	gotTask, err := svc.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotTask.InflightExecutionID != 0 {
		t.Fatalf("task inflight = %d, want 0", gotTask.InflightExecutionID)
	}
}

func TestRunNowDoesNotChangeScheduledCadence(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, &now)
	ctx := context.Background()

	task, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
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
		TaskID:        task.ID,
		ExecutionID:   exec.ID,
		WorkerID:      "worker-1",
		ExecutorRunID: "run-1",
		StartedAt:     now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.MarkExecutionFinished(ctx, scheduledsdk.MarkExecutionFinishedRequest{
		TaskID:        task.ID,
		ExecutionID:   exec.ID,
		Status:        scheduledsdk.ExecutionStatusSuccess,
		ExecutorRunID: "run-1",
		FinishedAt:    now.Add(time.Minute),
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

func TestRunNowRecoversMissingInflightReference(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, &now)
	ctx := context.Background()

	task, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
		ExecutorKey: "test.executor",
		Schedule:    scheduledsdk.Every(3600),
	})
	if err != nil {
		t.Fatal(err)
	}
	const missingExecutionID int64 = 4453
	if err := db.Model(&model.TimerTask{}).
		Where("id = ?", task.ID).
		Updates(map[string]interface{}{
			"inflight_execution_id": missingExecutionID,
			"last_execution_id":     missingExecutionID,
		}).Error; err != nil {
		t.Fatal(err)
	}

	exec, err := svc.RunNow(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if exec.TriggerType != triggerManual {
		t.Fatalf("trigger_type = %q, want manual", exec.TriggerType)
	}
	gotTask, err := svc.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotTask.InflightExecutionID != exec.ID {
		t.Fatalf("task inflight = %d, want new execution %d", gotTask.InflightExecutionID, exec.ID)
	}
}

func TestRunNowIgnoresStaleInflightExecution(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, &now)
	ctx := context.Background()

	task, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
		ExecutorKey: "test.executor",
		Schedule:    scheduledsdk.Every(3600),
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := svc.RunNow(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}

	now = now.Add(2 * time.Minute)
	second, err := svc.RunNow(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if second.ID == first.ID {
		t.Fatalf("stale inflight should create a new execution, got same id %d", second.ID)
	}

	oldExec, err := svc.GetExecution(ctx, task.ID, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if oldExec.Status != scheduledsdk.ExecutionStatusQueued {
		t.Fatalf("old execution status = %s, want queued", oldExec.Status)
	}
	gotTask, err := svc.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotTask.InflightExecutionID != second.ID {
		t.Fatalf("task inflight = %d, want new execution %d", gotTask.InflightExecutionID, second.ID)
	}
}

func TestRunNowAllowsOverlappingInflightExecution(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, &now)
	ctx := context.Background()

	task, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
		ExecutorKey: "test.executor",
		Schedule:    scheduledsdk.Every(3600),
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := svc.RunNow(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.RunNow(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if second.ID == first.ID {
		t.Fatalf("overlapping run_now should create a new execution, got same id %d", second.ID)
	}
	gotTask, err := svc.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotTask.InflightExecutionID != second.ID {
		t.Fatalf("task inflight = %d, want latest execution %d", gotTask.InflightExecutionID, second.ID)
	}
	if gotTask.LastExecutionID != second.ID {
		t.Fatalf("task last_execution = %d, want overlapping execution %d", gotTask.LastExecutionID, second.ID)
	}

	if err := svc.MarkExecutionStarted(ctx, scheduledsdk.MarkExecutionStartedRequest{
		TaskID:        task.ID,
		ExecutionID:   second.ID,
		WorkerID:      "worker-2",
		ExecutorRunID: "run-2",
		StartedAt:     now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.MarkExecutionFinished(ctx, scheduledsdk.MarkExecutionFinishedRequest{
		TaskID:        task.ID,
		ExecutionID:   second.ID,
		Status:        scheduledsdk.ExecutionStatusSuccess,
		ExecutorRunID: "run-2",
		FinishedAt:    now.Add(time.Second),
	}); err != nil {
		t.Fatal(err)
	}

	gotTask, err = svc.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotTask.InflightExecutionID != 0 {
		t.Fatalf("finishing overlapping execution should clear observational inflight, got %d", gotTask.InflightExecutionID)
	}
}

func TestCreateTaskReusesActiveIdempotencyKey(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, &now)
	ctx := context.Background()

	first, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
		Title:          "first",
		IdempotencyKey: "same-live-task",
		ExecutorKey:    "test.executor",
		Schedule:       scheduledsdk.Every(60),
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
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

	first, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
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
	second, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
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

	first, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
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

	second, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
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

func TestDeleteTaskAllowsInflightExecution(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, &now)
	ctx := context.Background()

	task, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
		ExecutorKey: "test.executor",
		Schedule:    scheduledsdk.Every(60),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RunNow(ctx, task.ID); err != nil {
		t.Fatal(err)
	}

	if err := svc.DeleteTask(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetTask(ctx, task.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("deleted task should be hidden, got err=%v", err)
	}
}

func TestListTasksResourceKeyPrefixIncludesDirectoryDescendantsOnly(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, &now)
	ctx := context.Background()
	for _, req := range []scheduledsdk.CreateTaskRequest{
		{
			Title:         "daily report",
			ExecutorKey:   "agent.session",
			Schedule:      scheduledsdk.Every(60),
			ResourceScope: "workspace_directory",
			ResourceKey:   "/system/app",
		},
		{
			Title:         "collect",
			ExecutorKey:   "app.function",
			Schedule:      scheduledsdk.Every(60),
			ResourceScope: "function",
			ResourceKey:   "/system/app/collect.form",
		},
		{
			Title:         "other app",
			ExecutorKey:   "app.function",
			Schedule:      scheduledsdk.Every(60),
			ResourceScope: "function",
			ResourceKey:   "/system/app2/collect.form",
		},
	} {
		if _, err := createTestTask(svc, ctx, req); err != nil {
			t.Fatal(err)
		}
	}

	resp, err := svc.ListTasks(ctx, scheduledsdk.ListTasksRequest{
		ResourceKeyPrefix: "/system/app",
		PageSize:          100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Total != 2 || len(resp.List) != 2 {
		t.Fatalf("prefix list should include exact directory and descendants only, got total=%d list=%+v", resp.Total, resp.List)
	}
	for _, task := range resp.List {
		if task.ResourceKey == "/system/app2/collect.form" {
			t.Fatalf("prefix should not include sibling path: %+v", task)
		}
	}
}

func TestDeleteTaskDoesNotCheckInflightExecution(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, &now)
	ctx := context.Background()

	task, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
		ExecutorKey: "test.executor",
		Schedule:    scheduledsdk.Every(60),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RunNow(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteTask(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetTask(ctx, task.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("deleted task should be hidden, got err=%v", err)
	}
}

func TestFinishExecutionAfterTaskDeleteIsRejected(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, _ := newTestService(t, &now)
	ctx := context.Background()

	task, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
		ExecutorKey: "test.executor",
		Schedule:    scheduledsdk.Every(60),
	})
	if err != nil {
		t.Fatal(err)
	}
	exec, err := svc.RunNow(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteTask(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.MarkExecutionFinished(ctx, scheduledsdk.MarkExecutionFinishedRequest{
		TaskID:        task.ID,
		ExecutionID:   exec.ID,
		Status:        scheduledsdk.ExecutionStatusSuccess,
		ExecutorRunID: "run-delete",
		FinishedAt:    now.Add(time.Second),
	}); !errors.Is(err, ErrInvalidTaskStatus) {
		t.Fatalf("finish deleted execution error = %v, want ErrInvalidTaskStatus", err)
	}
	gotExec, err := svc.GetExecution(ctx, task.ID, exec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotExec.Status != scheduledsdk.ExecutionStatusCancelled {
		t.Fatalf("execution status = %s, want cancelled", gotExec.Status)
	}
}

func TestPublishPendingOutboxRetriesBeforePublished(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, &now)
	ctx := context.Background()

	if _, err := createTestTask(svc, ctx, scheduledsdk.CreateTaskRequest{
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
