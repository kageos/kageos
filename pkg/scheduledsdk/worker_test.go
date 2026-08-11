package scheduledsdk

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

type workerLifecycleTestAdapter struct {
	Adapter
	started  chan struct{}
	finished chan error
}

func (a *workerLifecycleTestAdapter) MarkExecutionStarted(context.Context, MarkExecutionStartedRequest) error {
	a.started <- struct{}{}
	return nil
}

func (a *workerLifecycleTestAdapter) MarkExecutionFinished(ctx context.Context, _ MarkExecutionFinishedRequest) error {
	a.finished <- ctx.Err()
	return nil
}

func TestWorkerMessagePreservesLifecycleCancellation(t *testing.T) {
	adapter := &workerLifecycleTestAdapter{
		started:  make(chan struct{}, 1),
		finished: make(chan error, 1),
	}
	handlerCanceled := make(chan struct{}, 1)
	worker, err := NewWorker(WorkerOptions{
		Client:      NewClient(Options{Adapter: adapter}),
		NATSConn:    &nats.Conn{},
		ExecutorKey: "test.executor",
		Handler: func(ctx context.Context, _ ExecutionRequestedEvent) (*ExecutionResult, error) {
			<-ctx.Done()
			handlerCanceled <- struct{}{}
			return nil, ctx.Err()
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	eventData, err := json.Marshal(ExecutionRequestedEvent{TaskID: 1, ExecutionID: 2, ExecutorKey: "test.executor"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		worker.handleMessage(ctx, &nats.Msg{Data: eventData})
	}()

	select {
	case <-adapter.started:
	case <-time.After(time.Second):
		t.Fatal("worker did not mark execution started")
	}
	cancel()

	select {
	case <-handlerCanceled:
	case <-time.After(time.Second):
		t.Fatal("worker handler did not receive lifecycle cancellation")
	}
	select {
	case finishCtxErr := <-adapter.finished:
		if finishCtxErr != nil {
			t.Fatalf("finish context error = %v, want detached finalization context", finishCtxErr)
		}
	case <-time.After(time.Second):
		t.Fatal("worker did not report the final execution state")
	}
	<-done
}

func TestWorkerStartRejectsNilContext(t *testing.T) {
	worker := &Worker{}
	if err := worker.Start(nil); err == nil {
		t.Fatal("Start(nil) error = nil, want required context error")
	}
}

func TestNewWorkerDefaultsAndAcceptsBoundedConcurrency(t *testing.T) {
	newWorker := func(concurrency int) *Worker {
		t.Helper()
		conn := &nats.Conn{}
		worker, err := NewWorker(WorkerOptions{
			Client:      NewClient(Options{Adapter: NewNATSAdapter(conn, NATSAdapterOptions{})}),
			NATSConn:    conn,
			ExecutorKey: "test.executor",
			Concurrency: concurrency,
			Handler: func(context.Context, ExecutionRequestedEvent) (*ExecutionResult, error) {
				return &ExecutionResult{}, nil
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		return worker
	}

	if got := newWorker(0).concurrency; got != 1 {
		t.Fatalf("default concurrency = %d, want 1", got)
	}
	if got := newWorker(4).concurrency; got != 4 {
		t.Fatalf("configured concurrency = %d, want 4", got)
	}
}
