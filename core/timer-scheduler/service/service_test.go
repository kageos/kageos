package service

import (
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

func TestScheduledExecutionTokenSkipsTaskWithoutRequestUserID(t *testing.T) {
	task := &model.TimerTask{
		RequestUser: "system",
		CreatedBy:   "system",
	}
	if token := scheduledExecutionToken(task); token != "" {
		t.Fatalf("task without request_user_id should not receive delegated token, got %q", token)
	}
}

func TestScheduledExecutionTokenSupportsSystemUser(t *testing.T) {
	task := &model.TimerTask{
		RequestUser:     "system",
		CreatedBy:       "system",
		RequestUserDept: "/system",
		MetadataJSON:    json.RawMessage(`{"request_user_id":"1","request_email":"system@example.com","company_code":"kageos"}`),
	}
	token := scheduledExecutionToken(task)
	if token == "" {
		t.Fatal("system task with request_user_id should receive delegated token")
	}
	claims, err := auth.NewJWTService().ValidateToken(token)
	if err != nil {
		t.Fatalf("system delegated token should validate: %v", err)
	}
	if claims.UserID != 1 || claims.Username != "system" || claims.Email != "system@example.com" {
		t.Fatalf("unexpected claims: user_id=%d username=%q email=%q", claims.UserID, claims.Username, claims.Email)
	}
	if claims.DepartmentFullPath == nil || *claims.DepartmentFullPath != "/system" {
		t.Fatalf("expected department in system token, claims=%+v", claims)
	}
	if claims.CompanyCode != "kageos" {
		t.Fatalf("company code = %q, want kageos", claims.CompanyCode)
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

	task, err := svc.CreateTask(ctx, scheduledsdk.CreateTaskRequest{
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
	if strings.TrimSpace(event.Token) == "" {
		t.Fatal("scheduled execution event should include delegated token")
	}
	claims, err := auth.NewJWTService().ValidateToken(event.Token)
	if err != nil {
		t.Fatalf("scheduled execution token should validate: %v", err)
	}
	if claims.UserID != 42 || claims.Username != "alice" || claims.Email != "alice@example.com" || claims.Subject != "access_42" {
		t.Fatalf("unexpected scheduled token claims: user_id=%d username=%q email=%q subject=%q", claims.UserID, claims.Username, claims.Email, claims.Subject)
	}
	if claims.DepartmentFullPath == nil || *claims.DepartmentFullPath != "/org/dev" {
		t.Fatalf("scheduled token should include department, claims=%+v", claims)
	}
	if claims.CompanyCode != "acme" || claims.CompanyName != "Acme" {
		t.Fatalf("scheduled token should include company metadata, claims=%+v", claims)
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

func TestRecoverStaleRunningExecutionExtendsLeaseBeforeTimeout(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, &now)
	svc.opts.ExecutionLeaseDuration = time.Minute
	svc.opts.MaxHeartbeatMisses = 3
	ctx := context.Background()

	task, err := svc.CreateTask(ctx, scheduledsdk.CreateTaskRequest{
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
		TaskID:      task.ID,
		ExecutionID: exec.ID,
		WorkerID:    "worker-1",
		StartedAt:   now,
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
		TaskID:      task.ID,
		ExecutionID: exec.ID,
		WorkerID:    "worker-1",
		HeartbeatAt: now,
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

	task, err := svc.CreateTask(ctx, scheduledsdk.CreateTaskRequest{
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
		TaskID:      task.ID,
		ExecutionID: exec.ID,
		WorkerID:    "worker-1",
		StartedAt:   now,
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

func TestRecoverStaleExecutionsClearsMissingInflightReference(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, &now)
	ctx := context.Background()

	task, err := svc.CreateTask(ctx, scheduledsdk.CreateTaskRequest{
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

	task, err := svc.CreateTask(ctx, scheduledsdk.CreateTaskRequest{
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

func TestRunNowRecoversMissingInflightReference(t *testing.T) {
	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	svc, db := newTestService(t, &now)
	ctx := context.Background()

	task, err := svc.CreateTask(ctx, scheduledsdk.CreateTaskRequest{
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

func TestRunNowRejectsValidInflightExecution(t *testing.T) {
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
	first, err := svc.RunNow(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RunNow(ctx, task.ID); !errors.Is(err, ErrTaskBusy) {
		t.Fatalf("second run now err=%v, want ErrTaskBusy", err)
	}
	gotTask, err := svc.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotTask.InflightExecutionID != first.ID {
		t.Fatalf("task inflight = %d, want first execution %d", gotTask.InflightExecutionID, first.ID)
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
