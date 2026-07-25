package scheduledsdk

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"
)

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
