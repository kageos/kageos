package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/scheduledsdk"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeScheduledAgentChatRunner struct {
	sessionID string
	err       error
	calls     int32
}

func (r *fakeScheduledAgentChatRunner) RunWorkspaceChat(ctx context.Context, req *dto.WorkspaceChatReq, sink WorkspaceChatEventSink) error {
	atomic.AddInt32(&r.calls, 1)
	if r.sessionID != "" && sink != nil {
		sink.Send(EventSession, StreamEventSession{SessionID: r.sessionID})
	}
	return r.err
}

func (r *fakeScheduledAgentChatRunner) CallCount() int32 {
	return atomic.LoadInt32(&r.calls)
}

type fakeScheduledAgentTimerAdapter struct {
	createdReq scheduledsdk.CreateTaskRequest
	pausedID   int64
	resumedID  int64
	canceledID int64
	runNowID   int64
	pauseErr   error
	cancelErr  error
	task       *scheduledsdk.Task
}

func (f *fakeScheduledAgentTimerAdapter) CreateTask(_ context.Context, req scheduledsdk.CreateTaskRequest) (*scheduledsdk.Task, error) {
	f.createdReq = req
	if f.task != nil {
		return f.task, nil
	}
	return &scheduledsdk.Task{ID: 99, ExecutorKey: req.ExecutorKey}, nil
}

func (f *fakeScheduledAgentTimerAdapter) UpdateTask(context.Context, int64, scheduledsdk.UpdateTaskRequest) (*scheduledsdk.Task, error) {
	return f.task, nil
}

func (f *fakeScheduledAgentTimerAdapter) PauseTask(_ context.Context, id int64) error {
	f.pausedID = id
	return f.pauseErr
}
func (f *fakeScheduledAgentTimerAdapter) ResumeTask(_ context.Context, id int64) error {
	f.resumedID = id
	return nil
}
func (f *fakeScheduledAgentTimerAdapter) CancelTask(_ context.Context, id int64) error {
	f.canceledID = id
	return f.cancelErr
}
func (f *fakeScheduledAgentTimerAdapter) RunNow(_ context.Context, id int64) (*scheduledsdk.Execution, error) {
	f.runNowID = id
	return &scheduledsdk.Execution{
		ID:          456,
		TaskID:      id,
		Status:      scheduledsdk.ExecutionStatusQueued,
		ScheduledAt: time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC),
	}, nil
}
func (f *fakeScheduledAgentTimerAdapter) GetTask(context.Context, int64) (*scheduledsdk.Task, error) {
	return nil, nil
}
func (f *fakeScheduledAgentTimerAdapter) ListTasks(context.Context, scheduledsdk.ListTasksRequest) (*scheduledsdk.ListTasksResponse, error) {
	return nil, nil
}
func (f *fakeScheduledAgentTimerAdapter) GetExecution(context.Context, int64, int64) (*scheduledsdk.Execution, error) {
	return nil, nil
}
func (f *fakeScheduledAgentTimerAdapter) ListExecutions(context.Context, int64, scheduledsdk.ListExecutionsRequest) (*scheduledsdk.ListExecutionsResponse, error) {
	return nil, nil
}

func newScheduledAgentTaskTestService(t *testing.T, runner scheduledAgentWorkspaceChatRunner) (*ScheduledAgentTaskService, *repository.ScheduledAgentTaskRepository, *repository.ScheduledAgentExecutionRepository) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ScheduledAgentTask{}, &model.ScheduledAgentExecution{}); err != nil {
		t.Fatalf("migrate scheduled agent models: %v", err)
	}

	taskRepo := repository.NewScheduledAgentTaskRepository(db)
	executionRepo := repository.NewScheduledAgentExecutionRepository(db)
	svc := NewScheduledAgentTaskService(nil, taskRepo, executionRepo, nil, ScheduledAgentTaskServiceOptions{
		MaxConcurrency: 1,
		DefaultTimeout: time.Minute,
		TimerClient:    scheduledsdk.NewClient(scheduledsdk.Options{Adapter: &fakeScheduledAgentTimerAdapter{}}),
	})
	svc.chatRunner = runner
	return svc, taskRepo, executionRepo
}

func createScheduledAgentTestTask(t *testing.T, repo *repository.ScheduledAgentTaskRepository, task *model.ScheduledAgentTask) *model.ScheduledAgentTask {
	t.Helper()
	if task.Name == "" {
		task.Name = "test scheduled agent task"
	}
	if task.FullCodePath == "" {
		task.FullCodePath = "/system/demo"
	}
	if task.Goal == "" {
		task.Goal = "检查项目状态"
	}
	if task.ModeCode == "" {
		task.ModeCode = "dev"
	}
	if task.ScheduleType == "" {
		task.ScheduleType = ScheduledAgentScheduleEvery
	}
	if task.Status == "" {
		task.Status = model.ScheduledAgentTaskStatusPending
	}
	if task.RequestUser == "" {
		task.RequestUser = "alice"
	}
	if task.NotifyOn == "" {
		task.NotifyOn = ScheduledAgentNotifyNone
	}
	if task.CreatedBy == "" {
		task.CreatedBy = task.RequestUser
	}
	if task.UpdatedBy == "" {
		task.UpdatedBy = task.RequestUser
	}
	if err := repo.Create(task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	return task
}

func TestParseScheduledAgentRunAt(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	got, err := parseScheduledAgentRunAt("2026-04-24 09:30:00", loc)
	if err != nil {
		t.Fatalf("parseScheduledAgentRunAt returned error: %v", err)
	}
	want := time.Date(2026, 4, 24, 9, 30, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("run_at = %s, want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
	}

	if _, err := parseScheduledAgentRunAt("", loc); err == nil {
		t.Fatal("empty run_at should be rejected")
	}
}

func TestComputeScheduledAgentNextRun(t *testing.T) {
	scheduledAt := time.Date(2026, 4, 24, 9, 0, 0, 0, time.FixedZone("CST", 8*60*60))

	cronTask := &model.ScheduledAgentTask{
		ScheduleType: ScheduledAgentScheduleCron,
		CronExpr:     "*/5 * * * *",
		Status:       model.ScheduledAgentTaskStatusPending,
	}
	status, next, stateErr := computeScheduledAgentNextRunAt(cronTask, scheduledAt, scheduledAt, true)
	if status != model.ScheduledAgentTaskStatusPending || next == nil || !next.Equal(scheduledAt.Add(5*time.Minute)) || stateErr != "" {
		t.Fatalf("cron next state = (%s, %v, %q), want pending, %s, empty error", status, next, stateErr, scheduledAt.Add(5*time.Minute).Format(time.RFC3339))
	}

	oneShot := &model.ScheduledAgentTask{ScheduleType: ScheduledAgentScheduleAtime}
	status, next, _ = computeScheduledAgentNextRunAt(oneShot, scheduledAt, scheduledAt, true)
	if status != model.ScheduledAgentTaskStatusDone || next != nil {
		t.Fatalf("atime success next state = (%s, %v), want done, nil", status, next)
	}
}

func TestComputeScheduledAgentNextRunAvoidsCatchUpBurst(t *testing.T) {
	scheduledAt := time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)
	completedAt := scheduledAt.Add(time.Hour)

	everyTask := &model.ScheduledAgentTask{
		ScheduleType:    ScheduledAgentScheduleEvery,
		IntervalSeconds: 60,
	}
	status, next, stateErr := computeScheduledAgentNextRunAt(everyTask, scheduledAt, completedAt, true)
	if status != model.ScheduledAgentTaskStatusPending || next == nil || !next.Equal(completedAt.Add(time.Minute)) || stateErr != "" {
		t.Fatalf("every delayed next state = (%s, %v, %q), want pending, %s, empty error", status, next, stateErr, completedAt.Add(time.Minute).Format(time.RFC3339))
	}

	cronTask := &model.ScheduledAgentTask{
		ScheduleType: ScheduledAgentScheduleCron,
		CronExpr:     "*/5 * * * *",
	}
	status, next, stateErr = computeScheduledAgentNextRunAt(cronTask, scheduledAt, completedAt, true)
	if status != model.ScheduledAgentTaskStatusPending || next == nil || !next.Equal(completedAt.Add(5*time.Minute)) || stateErr != "" {
		t.Fatalf("cron delayed next state = (%s, %v, %q), want pending, %s, empty error", status, next, stateErr, completedAt.Add(5*time.Minute).Format(time.RFC3339))
	}
}

func TestScheduledAgentInitialNextRunAtDoesNotRunEveryImmediately(t *testing.T) {
	base := time.Date(2026, 4, 24, 9, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	task := &model.ScheduledAgentTask{
		ScheduleType:    ScheduledAgentScheduleEvery,
		IntervalSeconds: 60,
		RunAt:           base,
	}

	next, err := (&ScheduledAgentTaskService{}).initialNextRunAt(task, base)
	if err != nil {
		t.Fatalf("initialNextRunAt returned error: %v", err)
	}
	want := base.Add(time.Minute)
	if next == nil || !next.Equal(want) {
		t.Fatalf("every initial next = %v, want %s", next, want.Format(time.RFC3339))
	}
}

func TestScheduledAgentInitialNextRunAtRejectsPastAtime(t *testing.T) {
	now := time.Date(2026, 4, 24, 9, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	task := &model.ScheduledAgentTask{
		ScheduleType: ScheduledAgentScheduleAtime,
		RunAt:        now.Add(-time.Minute),
	}

	if _, err := (&ScheduledAgentTaskService{}).initialNextRunAt(task, now); err == nil {
		t.Fatalf("past atime should be rejected")
	}
}

func TestScheduledAgentUpdateDoesNotRecalculateNextRunForNonScheduleFields(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ScheduledAgentTask{}); err != nil {
		t.Fatalf("migrate scheduled agent task: %v", err)
	}
	repo := repository.NewScheduledAgentTaskRepository(db)
	svc := &ScheduledAgentTaskService{
		taskRepo: repo,
		options: ScheduledAgentTaskServiceOptions{
			TimerClient: scheduledsdk.NewClient(scheduledsdk.Options{Adapter: &fakeScheduledAgentTimerAdapter{}}),
		},
	}

	nextRunAt := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	task := &model.ScheduledAgentTask{
		Name:              "old",
		FullCodePath:      "/system/demo",
		Goal:              "old goal",
		ModeCode:          "dev",
		ScheduleType:      ScheduledAgentScheduleEvery,
		RunAt:             time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC),
		NextRunAt:         &nextRunAt,
		IntervalSeconds:   300,
		Status:            model.ScheduledAgentTaskStatusPending,
		RequestUser:       "alice",
		NotifyOn:          ScheduledAgentNotifyNone,
		NotifyUsers:       "",
		NotifyDepartments: "",
		TimerTaskID:       88,
	}
	task.CreatedBy = "alice"
	task.UpdatedBy = "alice"
	if err := repo.Create(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	updated, err := svc.Update(context.Background(), task.ID, &dto.UpdateScheduledAgentTaskReq{
		Name: "new",
		Goal: "new goal",
	}, "alice")
	if err != nil {
		t.Fatalf("update task: %v", err)
	}
	if updated.NextRunAt == nil || !updated.NextRunAt.Equal(nextRunAt) {
		t.Fatalf("next_run_at changed to %v, want %s", updated.NextRunAt, nextRunAt.Format(time.RFC3339))
	}
}

func TestScheduledAgentCreateEveryDoesNotRunImmediately(t *testing.T) {
	runner := &fakeScheduledAgentChatRunner{sessionID: "session-create"}
	svc, taskRepo, executionRepo := newScheduledAgentTaskTestService(t, runner)

	task, err := svc.Create(context.Background(), &dto.CreateScheduledAgentTaskReq{
		Name:            "every task",
		FullCodePath:    "/system/demo",
		Goal:            "检查项目状态",
		ScheduleType:    ScheduledAgentScheduleEvery,
		IntervalSeconds: 3600,
	}, "alice")
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if task.NextRunAt == nil || !task.NextRunAt.After(time.Now()) {
		t.Fatalf("created next_run_at = %v, want future", task.NextRunAt)
	}

	if runner.CallCount() != 0 {
		t.Fatalf("created every task ran immediately, calls=%d", runner.CallCount())
	}
	if _, total, err := executionRepo.ListByTaskID(task.ID, "", 0, 20); err != nil || total != 0 {
		t.Fatalf("execution total = %d, err=%v; want 0, nil", total, err)
	}
	stored, err := taskRepo.GetByID(task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if stored.RunCount != 0 || stored.Status != model.ScheduledAgentTaskStatusPending {
		t.Fatalf("stored state = run_count:%d status:%s, want 0 pending", stored.RunCount, stored.Status)
	}
}

func TestScheduledAgentRefs(t *testing.T) {
	if got := scheduledAgentTaskRef(42); got != "scheduled_agent_task:42" {
		t.Fatalf("task ref = %q", got)
	}
	if got := scheduledAgentExecutionRef(7); got != "scheduled_agent_execution:7" {
		t.Fatalf("execution ref = %q", got)
	}
}

func TestScheduledAgentCreateRegistersTimerTask(t *testing.T) {
	svc, taskRepo, _ := newScheduledAgentTaskTestService(t, &fakeScheduledAgentChatRunner{})
	timerAdapter := &fakeScheduledAgentTimerAdapter{task: &scheduledsdk.Task{ID: 123, ExecutorKey: scheduledAgentExecutorKey}}
	svc.options.TimerClient = scheduledsdk.NewClient(scheduledsdk.Options{Adapter: timerAdapter})

	task, err := svc.Create(context.Background(), &dto.CreateScheduledAgentTaskReq{
		Name:            "timer backed agent",
		FullCodePath:    "/system/demo",
		Goal:            "巡检",
		ScheduleType:    ScheduledAgentScheduleEvery,
		IntervalSeconds: 60,
		RequestUser:     "alice",
		NotifyUsers:     []string{"alice"},
	}, "alice")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if task.TimerTaskID != 123 {
		t.Fatalf("timer_task_id = %d, want 123", task.TimerTaskID)
	}
	if timerAdapter.createdReq.ExecutorKey != scheduledAgentExecutorKey {
		t.Fatalf("executor_key = %q, want %q", timerAdapter.createdReq.ExecutorKey, scheduledAgentExecutorKey)
	}
	if timerAdapter.createdReq.Metadata["full_code_path"] != "/system/demo" {
		t.Fatalf("metadata full_code_path = %q", timerAdapter.createdReq.Metadata["full_code_path"])
	}
	var payload scheduledAgentTimerPayload
	if err := json.Unmarshal(timerAdapter.createdReq.ExecutorPayload, &payload); err != nil {
		t.Fatalf("unmarshal executor payload: %v", err)
	}
	if payload.TaskID != task.ID || payload.BusinessRef != scheduledAgentTaskRef(task.ID) {
		t.Fatalf("payload = %#v, want task id/ref", payload)
	}
	stored, err := taskRepo.GetByID(task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if stored.TimerTaskID != 123 {
		t.Fatalf("stored timer_task_id = %d, want 123", stored.TimerTaskID)
	}
}

func TestScheduledAgentRunNowUsesTimerScheduler(t *testing.T) {
	runner := &fakeScheduledAgentChatRunner{sessionID: "session-direct-run"}
	svc, taskRepo, executionRepo := newScheduledAgentTaskTestService(t, runner)
	timerAdapter := &fakeScheduledAgentTimerAdapter{}
	svc.options.TimerClient = scheduledsdk.NewClient(scheduledsdk.Options{Adapter: timerAdapter})
	task := createScheduledAgentTestTask(t, taskRepo, &model.ScheduledAgentTask{
		TimerTaskID:     321,
		IntervalSeconds: 60,
	})

	exec, err := svc.RunNow(context.Background(), task.ID, "alice")
	if err != nil {
		t.Fatalf("RunNow returned error: %v", err)
	}
	if timerAdapter.runNowID != 321 {
		t.Fatalf("timer RunNow id = %d, want 321", timerAdapter.runNowID)
	}
	if exec == nil || exec.ID != 456 || exec.TaskID != 321 || exec.Status != scheduledsdk.ExecutionStatusQueued {
		t.Fatalf("timer execution = %#v", exec)
	}
	if runner.CallCount() != 0 {
		t.Fatalf("RunNow should not execute locally, calls=%d", runner.CallCount())
	}
	if _, total, err := executionRepo.ListByTaskID(task.ID, "", 0, 10); err != nil || total != 0 {
		t.Fatalf("local executions total=%d err=%v, want 0 nil", total, err)
	}
}

func TestScheduledAgentPauseDoesNotChangeLocalWhenTimerPauseFails(t *testing.T) {
	svc, taskRepo, _ := newScheduledAgentTaskTestService(t, &fakeScheduledAgentChatRunner{})
	timerAdapter := &fakeScheduledAgentTimerAdapter{pauseErr: fmt.Errorf("timer unavailable")}
	svc.options.TimerClient = scheduledsdk.NewClient(scheduledsdk.Options{Adapter: timerAdapter})
	task := createScheduledAgentTestTask(t, taskRepo, &model.ScheduledAgentTask{
		TimerTaskID:     654,
		IntervalSeconds: 60,
	})

	if err := svc.Pause(context.Background(), task.ID, "alice"); err == nil {
		t.Fatalf("Pause should fail when timer-scheduler pause fails")
	}
	stored, err := taskRepo.GetByID(task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if stored.Status != model.ScheduledAgentTaskStatusPending {
		t.Fatalf("local status = %q, want pending", stored.Status)
	}
}

func TestCancelScheduledSessionTaskToolCancelsSessionTask(t *testing.T) {
	svc, taskRepo, _ := newScheduledAgentTaskTestService(t, &fakeScheduledAgentChatRunner{})
	timerAdapter := &fakeScheduledAgentTimerAdapter{}
	svc.options.TimerClient = scheduledsdk.NewClient(scheduledsdk.Options{Adapter: timerAdapter})
	task := createScheduledAgentTestTask(t, taskRepo, &model.ScheduledAgentTask{
		TimerTaskID:     987,
		IntervalSeconds: 60,
	})
	reg := NewToolRegistry(nil)
	reg.SetScheduledAgentTaskService(svc)
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{RequestUser: "alice"})

	out, isErr := runCancelScheduledSessionTaskTool(ctx, reg, cancelScheduledSessionTaskArgs{TaskID: task.ID})
	if isErr {
		t.Fatalf("runCancelScheduledSessionTaskTool returned error: %s", out)
	}
	if timerAdapter.canceledID != 987 {
		t.Fatalf("timer canceled id = %d, want 987", timerAdapter.canceledID)
	}
	stored, err := taskRepo.GetByID(task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if stored.Status != model.ScheduledAgentTaskStatusCancelled || stored.NextRunAt != nil {
		t.Fatalf("stored status/next_run_at = %q/%v, want cancelled/nil", stored.Status, stored.NextRunAt)
	}
	if !strings.Contains(out, "已取消定时会话任务/session task") {
		t.Fatalf("output should include cancel message, got:\n%s", out)
	}
}

func TestScheduledAgentHandleTimerExecution(t *testing.T) {
	runner := &fakeScheduledAgentChatRunner{sessionID: "session-timer"}
	svc, taskRepo, executionRepo := newScheduledAgentTaskTestService(t, runner)
	task := createScheduledAgentTestTask(t, taskRepo, &model.ScheduledAgentTask{
		IntervalSeconds: 60,
		NextRunAt:       nil,
	})
	payload, err := json.Marshal(scheduledAgentTimerPayload{TaskID: task.ID, BusinessRef: scheduledAgentTaskRef(task.ID)})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	result, err := svc.HandleTimerExecution(context.Background(), scheduledsdk.ExecutionRequestedEvent{
		TaskID:          100,
		ExecutionID:     200,
		ExecutorKey:     scheduledAgentExecutorKey,
		ScheduledAt:     time.Now(),
		ExecutorPayload: payload,
	})
	if err != nil {
		t.Fatalf("HandleTimerExecution returned error: %v", err)
	}
	if result.Status != scheduledsdk.ExecutionStatusSuccess || result.ExecutorRunID != "session-timer" {
		t.Fatalf("result = %#v", result)
	}
	list, total, err := executionRepo.ListByTaskID(task.ID, "", 0, 10)
	if err != nil {
		t.Fatalf("list executions: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("executions total/list = %d/%d, want 1/1", total, len(list))
	}
	if list[0].SessionID != "session-timer" || list[0].Status != model.ScheduledAgentExecutionStatusSuccess {
		t.Fatalf("execution = %#v", list[0])
	}
}

func TestResolveScheduledAgentSource(t *testing.T) {
	sourceType, sourceRef := resolveScheduledAgentSource(context.Background())
	if sourceType != ScheduledAgentSourceType || sourceRef != "" {
		t.Fatalf("default source = (%q, %q), want (%q, empty)", sourceType, sourceRef, ScheduledAgentSourceType)
	}

	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		SourceType: "function",
		SourceRef:  "function:/system/reminder/send",
	})
	sourceType, sourceRef = resolveScheduledAgentSource(ctx)
	if sourceType != "function" || sourceRef != "function:/system/reminder/send" {
		t.Fatalf("context source = (%q, %q)", sourceType, sourceRef)
	}
}
