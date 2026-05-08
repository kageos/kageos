package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/scheduledsdk"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeScheduledTaskAppClient struct {
	function *model.Function
}

func (f *fakeScheduledTaskAppClient) RequestApp(ctx context.Context, req *dto.RequestAppReq) (*dto.RequestAppResp, error) {
	return &dto.RequestAppResp{}, nil
}

func (f *fakeScheduledTaskAppClient) GetFunctionByFullCodePath(ctx context.Context, fullCodePath string) (*model.Function, error) {
	return f.function, nil
}

func (f *fakeScheduledTaskAppClient) IncrementFunctionRunCount(ctx context.Context, fullCodePath string) {
}

func (f *fakeScheduledTaskAppClient) RecordFormOperateLog(ctx context.Context, req *dto.RecordFormOperateLogReq) error {
	return nil
}

func (f *fakeScheduledTaskAppClient) RecordTableOperateLog(ctx context.Context, req *dto.RecordTableOperateLogReq) error {
	return nil
}

type fakeScheduledTaskTimerAdapter struct {
	createdReq CreateScheduledTaskReqForTimerTest
	task       *scheduledsdk.Task
	canceledID int64
}

type CreateScheduledTaskReqForTimerTest struct {
	Title           string
	ExecutorKey     string
	ExecutorPayload string
	Metadata        map[string]string
	Schedule        scheduledsdk.Schedule
	NotifyUsers     []string
}

func (f *fakeScheduledTaskTimerAdapter) CreateTask(_ context.Context, req scheduledsdk.CreateTaskRequest) (*scheduledsdk.Task, error) {
	f.createdReq = CreateScheduledTaskReqForTimerTest{
		Title:           req.Title,
		ExecutorKey:     req.ExecutorKey,
		ExecutorPayload: string(req.ExecutorPayload),
		Metadata:        req.Metadata,
		Schedule:        req.Schedule,
		NotifyUsers:     req.NotifyUsers,
	}
	if f.task != nil {
		return f.task, nil
	}
	return &scheduledsdk.Task{ID: 123, ExecutorKey: req.ExecutorKey}, nil
}

func (f *fakeScheduledTaskTimerAdapter) UpdateTask(context.Context, int64, scheduledsdk.UpdateTaskRequest) (*scheduledsdk.Task, error) {
	return f.task, nil
}

func (f *fakeScheduledTaskTimerAdapter) PauseTask(context.Context, int64) error {
	return nil
}

func (f *fakeScheduledTaskTimerAdapter) ResumeTask(context.Context, int64) error {
	return nil
}

func (f *fakeScheduledTaskTimerAdapter) CancelTask(_ context.Context, taskID int64) error {
	f.canceledID = taskID
	return nil
}

func (f *fakeScheduledTaskTimerAdapter) RunNow(context.Context, int64) (*scheduledsdk.Execution, error) {
	return nil, nil
}

func (f *fakeScheduledTaskTimerAdapter) GetTask(context.Context, int64) (*scheduledsdk.Task, error) {
	return f.task, nil
}

func (f *fakeScheduledTaskTimerAdapter) ListTasks(context.Context, scheduledsdk.ListTasksRequest) (*scheduledsdk.ListTasksResponse, error) {
	return &scheduledsdk.ListTasksResponse{}, nil
}

func (f *fakeScheduledTaskTimerAdapter) GetExecution(context.Context, int64, int64) (*scheduledsdk.Execution, error) {
	return nil, nil
}

func (f *fakeScheduledTaskTimerAdapter) ListExecutions(context.Context, int64, scheduledsdk.ListExecutionsRequest) (*scheduledsdk.ListExecutionsResponse, error) {
	return &scheduledsdk.ListExecutionsResponse{}, nil
}

func scheduledTaskSchema(t *testing.T, templateType string, callbacks ...string) json.RawMessage {
	t.Helper()
	raw := map[string]interface{}{
		"version":   1,
		"type":      templateType,
		"callbacks": callbacks,
	}
	switch templateType {
	case "table":
		raw["table"] = map[string]interface{}{"request": []interface{}{}, "fields": []interface{}{}}
	case "form":
		raw["form"] = map[string]interface{}{"request": []interface{}{}, "response": []interface{}{}}
	case "chart":
		raw["chart"] = map[string]interface{}{"request": []interface{}{}, "response": []interface{}{}}
	}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	return data
}

func TestResolveScheduledTaskRunAt(t *testing.T) {
	now := time.Date(2026, 4, 15, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60))

	cronRunAt, err := resolveScheduledTaskRunAt("cron", "", now)
	if err != nil {
		t.Fatalf("cron resolve run_at returned error: %v", err)
	}
	if !cronRunAt.Equal(now) {
		t.Fatalf("cron run_at = %s, want %s", cronRunAt.Format(time.RFC3339), now.Format(time.RFC3339))
	}

	everyRunAt, err := resolveScheduledTaskRunAt("every", "", now)
	if err != nil {
		t.Fatalf("every resolve run_at returned error: %v", err)
	}
	if !everyRunAt.Equal(now) {
		t.Fatalf("every run_at = %s, want %s", everyRunAt.Format(time.RFC3339), now.Format(time.RFC3339))
	}

	if _, err := resolveScheduledTaskRunAt("atime", "", now); err == nil {
		t.Fatalf("atime resolve run_at should reject empty run_at")
	}
}

func TestNextCronRunAfterUsesNextMatch(t *testing.T) {
	base := time.Date(2026, 4, 15, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	want := time.Date(2026, 4, 15, 10, 1, 0, 0, time.FixedZone("CST", 8*60*60))

	next, err := nextCronRunAfter("*/1 * * * *", base)
	if err != nil {
		t.Fatalf("nextCronRunAfter returned error: %v", err)
	}
	if !next.Equal(want) {
		t.Fatalf("next = %s, want %s", next.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

func TestComputeTaskNextStateUsesScheduledTimeForCatchUp(t *testing.T) {
	scheduledAt := time.Date(2026, 4, 15, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60))

	cronTask := &model.ScheduledTask{
		ScheduleType: "cron",
		CronExpr:     "*/1 * * * *",
	}
	cronStatus, cronNext, cronErr := computeTaskNextState(cronTask, scheduledAt, true)
	if cronStatus != "pending" || cronNext == nil || !cronNext.Equal(scheduledAt.Add(time.Minute)) || cronErr != "" {
		t.Fatalf("cron next state = (%s, %v, %q), want pending, %s, empty error", cronStatus, cronNext, cronErr, scheduledAt.Add(time.Minute).Format(time.RFC3339))
	}

	everyTask := &model.ScheduledTask{
		ScheduleType:    "every",
		IntervalSeconds: 60,
		ErrorMessage:    "temporary failure",
	}
	everyStatus, everyNext, everyErr := computeTaskNextState(everyTask, scheduledAt, false)
	if everyStatus != "pending" || everyNext == nil || !everyNext.Equal(scheduledAt.Add(time.Minute)) || everyErr != "temporary failure" {
		t.Fatalf("every next state = (%s, %v, %q), want pending, %s, original error", everyStatus, everyNext, everyErr, scheduledAt.Add(time.Minute).Format(time.RFC3339))
	}
}

func TestCreateEveryScheduledTaskDoesNotRunImmediately(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ScheduledTask{}, &model.ScheduledTaskExecution{}); err != nil {
		t.Fatalf("migrate scheduled task tables: %v", err)
	}
	fn := &model.Function{
		Method:       "POST",
		TemplateType: "form",
		Schema:       scheduledTaskSchema(t, "form"),
	}
	svc := NewScheduledTaskService(
		&fakeScheduledTaskAppClient{function: fn},
		nil,
		repository.NewScheduledTaskRepository(db),
		repository.NewScheduledTaskExecutionRepository(db),
		ScheduledTaskServiceOptions{
			TimerClient: scheduledsdk.NewClient(scheduledsdk.Options{Adapter: &fakeScheduledTaskTimerAdapter{}}),
		},
	)

	task, err := svc.Create(context.Background(), &dto.CreateScheduledTaskReq{
		Name:            "every task",
		FullCodePath:    "/alice/demo/report.form",
		Action:          ScheduledTaskActionExecute,
		Method:          "POST",
		Payload:         json.RawMessage(`{}`),
		ScheduleType:    "every",
		IntervalSeconds: 60,
	}, "alice")
	if err != nil {
		t.Fatalf("create every task: %v", err)
	}
	want := task.RunAt.Add(time.Minute)
	if task.NextRunAt == nil || !task.NextRunAt.Equal(want) {
		t.Fatalf("every next_run_at = %v, want %s", task.NextRunAt, want.Format(time.RFC3339))
	}
}

func TestCreateScheduledTaskRegistersTimerTask(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ScheduledTask{}, &model.ScheduledTaskExecution{}); err != nil {
		t.Fatalf("migrate scheduled task tables: %v", err)
	}
	fn := &model.Function{
		Method:       "POST",
		TemplateType: "form",
		Schema:       scheduledTaskSchema(t, "form"),
	}
	timerAdapter := &fakeScheduledTaskTimerAdapter{
		task: &scheduledsdk.Task{ID: 456, ExecutorKey: scheduledTaskExecutorKey},
	}
	svc := NewScheduledTaskService(
		&fakeScheduledTaskAppClient{function: fn},
		nil,
		repository.NewScheduledTaskRepository(db),
		repository.NewScheduledTaskExecutionRepository(db),
		ScheduledTaskServiceOptions{
			TimerClient: scheduledsdk.NewClient(scheduledsdk.Options{Adapter: timerAdapter}),
		},
	)
	runAt := time.Now().Add(time.Hour).UTC().Truncate(time.Second)

	task, err := svc.Create(context.Background(), &dto.CreateScheduledTaskReq{
		Name:         "daily report",
		FullCodePath: "/alice/demo/report.form",
		Action:       ScheduledTaskActionExecute,
		Method:       "POST",
		Payload:      json.RawMessage(`{"foo":"bar"}`),
		RequestUser:  "alice",
		ScheduleType: "atime",
		RunAt:        runAt.Format(time.RFC3339),
		NotifyUsers:  []string{"alice", "bob", "alice"},
		NotifyOn:     ScheduledTaskNotifySuccess,
	}, "alice")
	if err != nil {
		t.Fatalf("create scheduled task: %v", err)
	}
	if task.TimerTaskID != 456 {
		t.Fatalf("timer_task_id = %d, want 456", task.TimerTaskID)
	}
	if timerAdapter.createdReq.ExecutorKey != scheduledTaskExecutorKey {
		t.Fatalf("timer executor_key = %q, want %q", timerAdapter.createdReq.ExecutorKey, scheduledTaskExecutorKey)
	}
	if timerAdapter.createdReq.Metadata["full_code_path"] != "/alice/demo/report.form" {
		t.Fatalf("timer metadata full_code_path = %q", timerAdapter.createdReq.Metadata["full_code_path"])
	}
	if timerAdapter.createdReq.Schedule.Type != scheduledsdk.ScheduleAt || !timerAdapter.createdReq.Schedule.RunAt.Equal(runAt) {
		t.Fatalf("timer schedule = %#v, want atime %s", timerAdapter.createdReq.Schedule, runAt.Format(time.RFC3339))
	}
	if timerAdapter.createdReq.ExecutorPayload == "" || !strings.Contains(timerAdapter.createdReq.ExecutorPayload, `"task_id":`) {
		t.Fatalf("timer executor_payload = %q, want task_id", timerAdapter.createdReq.ExecutorPayload)
	}

	stored, err := repository.NewScheduledTaskRepository(db).GetByID(task.ID)
	if err != nil {
		t.Fatalf("load stored task: %v", err)
	}
	if stored.TimerTaskID != 456 {
		t.Fatalf("stored timer_task_id = %d, want 456", stored.TimerTaskID)
	}
}

func TestScheduledTaskHandleTimerExecutionRunsBusinessTask(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ScheduledTask{}, &model.ScheduledTaskExecution{}); err != nil {
		t.Fatalf("migrate scheduled task tables: %v", err)
	}
	fn := &model.Function{
		Method:       "POST",
		TemplateType: "form",
		Schema:       scheduledTaskSchema(t, "form"),
	}
	svc := NewScheduledTaskService(
		&fakeScheduledTaskAppClient{function: fn},
		nil,
		repository.NewScheduledTaskRepository(db),
		repository.NewScheduledTaskExecutionRepository(db),
		ScheduledTaskServiceOptions{
			TimerClient: scheduledsdk.NewClient(scheduledsdk.Options{Adapter: &fakeScheduledTaskTimerAdapter{}}),
		},
	)
	runAt := time.Now().Add(-time.Minute).UTC().Truncate(time.Second)
	task, err := svc.Create(context.Background(), &dto.CreateScheduledTaskReq{
		Name:         "once",
		FullCodePath: "/alice/demo/report.form",
		Action:       ScheduledTaskActionExecute,
		Method:       "POST",
		Payload:      json.RawMessage(`{"foo":"bar"}`),
		RequestUser:  "alice",
		ScheduleType: "atime",
		RunAt:        runAt.Format(time.RFC3339),
	}, "alice")
	if err != nil {
		t.Fatalf("create scheduled task: %v", err)
	}
	payload, _ := json.Marshal(scheduledTaskTimerPayload{TaskID: task.ID})
	result, err := svc.HandleTimerExecution(context.Background(), scheduledsdk.ExecutionRequestedEvent{
		TaskID:          987,
		ExecutionID:     654,
		ExecutorKey:     scheduledTaskExecutorKey,
		ScheduledAt:     runAt,
		ExecutorPayload: payload,
	})
	if err != nil {
		t.Fatalf("HandleTimerExecution returned error: %v", err)
	}
	if result.Status != scheduledsdk.ExecutionStatusSuccess {
		t.Fatalf("execution status = %q, want success", result.Status)
	}
	if result.ExecutorRunID == "" {
		t.Fatalf("executor_run_id should contain local trace id")
	}
	stored, err := repository.NewScheduledTaskRepository(db).GetByID(task.ID)
	if err != nil {
		t.Fatalf("load stored task: %v", err)
	}
	if stored.Status != "done" || stored.RunCount != 1 {
		t.Fatalf("stored status/run_count = %s/%d, want done/1", stored.Status, stored.RunCount)
	}
	executions, total, err := repository.NewScheduledTaskExecutionRepository(db).ListByTaskID(task.ID, "", 0, 10)
	if err != nil {
		t.Fatalf("list executions: %v", err)
	}
	if total != 1 || len(executions) != 1 {
		t.Fatalf("executions total/list = %d/%d, want 1/1", total, len(executions))
	}
	if executions[0].Status != "success" {
		t.Fatalf("local execution status = %q, want success", executions[0].Status)
	}
}

func TestNormalizeScheduledTaskNotifyOnDefaults(t *testing.T) {
	withRecipients, err := normalizeScheduledTaskNotifyOn("", true)
	if err != nil {
		t.Fatalf("normalize notify_on with recipients returned error: %v", err)
	}
	if withRecipients != ScheduledTaskNotifyAll {
		t.Fatalf("notify_on with recipients = %q, want %q", withRecipients, ScheduledTaskNotifyAll)
	}

	withoutRecipients, err := normalizeScheduledTaskNotifyOn("", false)
	if err != nil {
		t.Fatalf("normalize notify_on without recipients returned error: %v", err)
	}
	if withoutRecipients != ScheduledTaskNotifyNone {
		t.Fatalf("notify_on without recipients = %q, want %q", withoutRecipients, ScheduledTaskNotifyNone)
	}

	if _, err := normalizeScheduledTaskNotifyOn("bad", true); err == nil {
		t.Fatalf("normalize notify_on should reject unsupported value")
	}
}

func TestNormalizeScheduledTaskActionRejectsUnsupportedFormAlias(t *testing.T) {
	if _, err := normalizeScheduledTaskAction("form"); err == nil {
		t.Fatalf("unsupported form action should be rejected")
	}
}

func TestValidateScheduledTaskTargetUsesSchemaCallbacksAndMethodMapping(t *testing.T) {
	fn := &model.Function{
		Method:       "GET",
		TemplateType: "table",
		Schema:       scheduledTaskSchema(t, "table", "OnTableUpdateRow"),
	}
	svc := &ScheduledTaskService{appClient: &fakeScheduledTaskAppClient{function: fn}}

	_, method, err := svc.validateScheduledTaskTarget(context.Background(), "/alice/demo/tickets.table", ScheduledTaskActionTableUpdate, "POST", "alice")
	if err != nil {
		t.Fatalf("validate table update returned error: %v", err)
	}
	if method != "PUT" {
		t.Fatalf("table update method = %q, want PUT", method)
	}
}

func TestValidateScheduledTaskTargetRejectsMissingTableCallback(t *testing.T) {
	fn := &model.Function{
		Method:       "GET",
		TemplateType: "table",
		Schema:       scheduledTaskSchema(t, "table", "OnTableAddRow"),
	}
	svc := &ScheduledTaskService{appClient: &fakeScheduledTaskAppClient{function: fn}}

	if _, _, err := svc.validateScheduledTaskTarget(context.Background(), "/alice/demo/tickets.table", ScheduledTaskActionTableDelete, "", "alice"); err == nil {
		t.Fatalf("table delete without OnTableDeleteRows should be rejected")
	}
}

func TestValidateScheduledTaskTargetRejectsExecuteTableWrite(t *testing.T) {
	fn := &model.Function{
		Method:       "POST",
		TemplateType: "table",
		Schema:       scheduledTaskSchema(t, "table"),
	}
	svc := &ScheduledTaskService{appClient: &fakeScheduledTaskAppClient{function: fn}}

	if _, _, err := svc.validateScheduledTaskTarget(context.Background(), "/alice/demo/tickets.table", ScheduledTaskActionExecute, "POST", "alice"); err == nil {
		t.Fatalf("execute must not be allowed to write table functions")
	}
}

func TestShouldNotifyScheduledTaskMatchesCondition(t *testing.T) {
	task := &model.ScheduledTask{NotifyUsers: "alice", NotifyOn: ScheduledTaskNotifySuccess}
	if !shouldNotifyScheduledTask(task, true) {
		t.Fatalf("success notification should fire on success")
	}
	if shouldNotifyScheduledTask(task, false) {
		t.Fatalf("success notification should not fire on failure")
	}

	task.NotifyOn = ScheduledTaskNotifyFailed
	if shouldNotifyScheduledTask(task, true) {
		t.Fatalf("failed notification should not fire on success")
	}
	if !shouldNotifyScheduledTask(task, false) {
		t.Fatalf("failed notification should fire on failure")
	}

	task.NotifyUsers = ""
	task.NotifyDepartments = ""
	task.NotifyOn = ScheduledTaskNotifyAll
	if shouldNotifyScheduledTask(task, true) {
		t.Fatalf("notification without recipients should not fire")
	}
}

func TestScheduledTaskExecutionResultURL(t *testing.T) {
	svc := &ScheduledTaskService{
		options: ScheduledTaskServiceOptions{
			NotificationBaseURL: "https://example.com/",
		},
	}
	task := &model.ScheduledTask{
		ID:           7,
		FullCodePath: "/alice/demo/report.form",
	}
	exec := &model.ScheduledTaskExecution{ID: 11}

	got := svc.buildExecutionResultURL(task, exec)
	want := "https://example.com/workspace/alice/demo/report.form?_panel=scheduledTask&_scheduled_execution_id=11&_scheduled_task_id=7"
	if got != want {
		t.Fatalf("execution result url = %q, want %q", got, want)
	}
}

func TestScheduledTaskExecutionReplayURL(t *testing.T) {
	svc := &ScheduledTaskService{
		options: ScheduledTaskServiceOptions{
			NotificationBaseURL: "https://example.com/",
		},
	}
	task := &model.ScheduledTask{
		ID:           7,
		FullCodePath: "/alice/demo/report.form",
	}
	exec := &model.ScheduledTaskExecution{ID: 11}

	got := svc.buildExecutionReplayURL(task, exec)
	want := "https://example.com/workspace/alice/demo/report.form?_replay=scheduled_execution&_scheduled_execution_id=11&_scheduled_task_id=7"
	if got != want {
		t.Fatalf("execution replay url = %q, want %q", got, want)
	}
}

func TestScheduledTaskFormFunctionDetection(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "form path", path: "/alice/demo/report.form", want: true},
		{name: "form path with spaces", path: " /alice/demo/report.form ", want: true},
		{name: "table path", path: "/alice/demo/report.table", want: false},
		{name: "chart path", path: "/alice/demo/report.chart", want: false},
		{name: "empty path", path: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isScheduledTaskFormFunction(&model.ScheduledTask{FullCodePath: tt.path})
			if got != tt.want {
				t.Fatalf("isScheduledTaskFormFunction(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}

	if isScheduledTaskFormFunction(nil) {
		t.Fatalf("nil task must not be treated as form")
	}
}

func TestScheduledTaskExecutionTraceURL(t *testing.T) {
	svc := &ScheduledTaskService{
		options: ScheduledTaskServiceOptions{
			NotificationBaseURL: "https://example.com/",
		},
	}
	task := &model.ScheduledTask{FullCodePath: "/alice/demo/report.form"}
	exec := &model.ScheduledTaskExecution{TraceID: "trace-1"}

	got := svc.buildExecutionTraceURL(task, exec)
	want := "https://example.com/workspace/alice/demo/report.form?_panel=operateLog&_trace_id=trace-1"
	if got != want {
		t.Fatalf("execution trace url = %q, want %q", got, want)
	}
}
