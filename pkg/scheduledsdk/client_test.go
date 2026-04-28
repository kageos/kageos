package scheduledsdk

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

type fakeAdapter struct {
	createdReq CreateTaskRequest
	task       *Task
	execution  *Execution
}

func (f *fakeAdapter) CreateTask(_ context.Context, req CreateTaskRequest) (*Task, error) {
	f.createdReq = req
	return f.task, nil
}

func (f *fakeAdapter) UpdateTask(context.Context, int64, UpdateTaskRequest) (*Task, error) {
	return f.task, nil
}

func (f *fakeAdapter) PauseTask(context.Context, int64) error {
	return nil
}

func (f *fakeAdapter) ResumeTask(context.Context, int64) error {
	return nil
}

func (f *fakeAdapter) CancelTask(context.Context, int64) error {
	return nil
}

func (f *fakeAdapter) RunNow(context.Context, int64) (*Execution, error) {
	return f.execution, nil
}

func (f *fakeAdapter) GetTask(context.Context, int64) (*Task, error) {
	return f.task, nil
}

func (f *fakeAdapter) ListTasks(context.Context, ListTasksRequest) (*ListTasksResponse, error) {
	return &ListTasksResponse{List: []*Task{f.task}, Total: 1}, nil
}

func (f *fakeAdapter) GetExecution(context.Context, int64, int64) (*Execution, error) {
	return f.execution, nil
}

func (f *fakeAdapter) ListExecutions(context.Context, int64, ListExecutionsRequest) (*ListExecutionsResponse, error) {
	return &ListExecutionsResponse{List: []*Execution{f.execution}, Total: 1}, nil
}

func TestClientRequiresAdapter(t *testing.T) {
	client := NewClient(Options{})
	if _, err := client.GetTask(context.Background(), 1); !errors.Is(err, ErrNilAdapter) {
		t.Fatalf("GetTask err = %v, want ErrNilAdapter", err)
	}
}

func TestClientDelegatesCreateTask(t *testing.T) {
	adapter := &fakeAdapter{task: &Task{ID: 7, ExecutorKey: "agent.session"}}
	client := NewClient(Options{Adapter: adapter})
	runAt := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	payload := json.RawMessage(`{"business_ref":"agent_task:7"}`)

	task, err := client.CreateTask(context.Background(), CreateTaskRequest{
		Title:           "daily check",
		Category:        "inspection",
		ExecutorKey:     "agent.session",
		ExecutorPayload: payload,
		Schedule:        At(runAt),
		SourceRef:       "/system/project",
	})
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	if task.ID != 7 {
		t.Fatalf("task ID = %d, want 7", task.ID)
	}
	if adapter.createdReq.ExecutorKey != "agent.session" {
		t.Fatalf("delegated executor_key = %q, want agent.session", adapter.createdReq.ExecutorKey)
	}
	if string(adapter.createdReq.ExecutorPayload) != string(payload) {
		t.Fatalf("delegated executor_payload = %s, want %s", adapter.createdReq.ExecutorPayload, payload)
	}
	if adapter.createdReq.Schedule.Type != ScheduleAt || !adapter.createdReq.Schedule.RunAt.Equal(runAt) {
		t.Fatalf("delegated schedule = %#v, want atime %s", adapter.createdReq.Schedule, runAt.Format(time.RFC3339))
	}
}

func TestClientExecutionEventsRequireEventAdapter(t *testing.T) {
	client := NewClient(Options{Adapter: &fakeAdapter{}})
	if err := client.MarkExecutionStarted(context.Background(), MarkExecutionStartedRequest{ExecutionID: 1}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("MarkExecutionStarted err = %v, want ErrUnsupported", err)
	}
}

type fakeEventAdapter struct {
	fakeAdapter
	startedReq  MarkExecutionStartedRequest
	finishedReq MarkExecutionFinishedRequest
}

func (f *fakeEventAdapter) PublishExecutionRequested(context.Context, ExecutionRequestedEvent) error {
	return nil
}

func (f *fakeEventAdapter) MarkExecutionStarted(_ context.Context, req MarkExecutionStartedRequest) error {
	f.startedReq = req
	return nil
}

func (f *fakeEventAdapter) MarkExecutionFinished(_ context.Context, req MarkExecutionFinishedRequest) error {
	f.finishedReq = req
	return nil
}

func TestWorkerHandlesMatchingExecutorEvent(t *testing.T) {
	adapter := &fakeEventAdapter{}
	client := NewClient(Options{Adapter: adapter})
	called := false
	worker := &Worker{
		client:      client,
		executorKey: "agent.session",
		workerID:    "worker-1",
		handler: func(_ context.Context, event ExecutionRequestedEvent) (*ExecutionResult, error) {
			called = true
			if string(event.ExecutorPayload) != `{"business_ref":"agent_task:7"}` {
				t.Fatalf("executor_payload = %s", event.ExecutorPayload)
			}
			return &ExecutionResult{
				Status:        ExecutionStatusSuccess,
				ExecutorRunID: "session-1",
				OutputSummary: "ok",
			}, nil
		},
	}
	event := ExecutionRequestedEvent{
		TaskID:          11,
		ExecutionID:     22,
		ExecutorKey:     "agent.session",
		ExecutorPayload: json.RawMessage(`{"business_ref":"agent_task:7"}`),
	}
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	worker.handleMessage(&nats.Msg{Data: data})
	if !called {
		t.Fatal("handler was not called")
	}
	if adapter.startedReq.WorkerID != "worker-1" || adapter.startedReq.ExecutionID != 22 {
		t.Fatalf("started req = %#v", adapter.startedReq)
	}
	if adapter.finishedReq.Status != ExecutionStatusSuccess || adapter.finishedReq.ExecutorRunID != "session-1" {
		t.Fatalf("finished req = %#v", adapter.finishedReq)
	}
}

func TestScheduleValidate(t *testing.T) {
	if err := Every(0).Validate(); err == nil {
		t.Fatal("Every(0).Validate returned nil, want error")
	}
	if err := Cron("  0 * * * *  ").Validate(); err != nil {
		t.Fatalf("Cron Validate returned error: %v", err)
	}
}
