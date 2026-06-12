package scheduledsdk

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos/pkg/contextx"
	"github.com/nats-io/nats.go"
)

func TestNATSAdapterUnsupportedControlPlaneOperations(t *testing.T) {
	adapter := NewNATSAdapter(nil, NATSAdapterOptions{})
	if _, err := adapter.CreateTask(context.Background(), CreateTaskRequest{}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("CreateTask error = %v, want ErrUnsupported", err)
	}
}

func TestNATSAdapterRequiresConnectionForExecutionUpdates(t *testing.T) {
	adapter := NewNATSAdapter(nil, NATSAdapterOptions{})
	err := adapter.MarkExecutionStarted(context.Background(), MarkExecutionStartedRequest{TaskID: 1, ExecutionID: 2})
	if err == nil || !strings.Contains(err.Error(), "nats connection") {
		t.Fatalf("MarkExecutionStarted error = %v, want nats connection error", err)
	}
}

func TestNATSAdapterRequestContextKeepsExistingDeadline(t *testing.T) {
	parent, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	got, gotCancel := natsAdapterRequestContext(parent, time.Second)
	defer gotCancel()
	if got != parent {
		t.Fatal("natsAdapterRequestContext should keep context with existing deadline")
	}
}

func TestApplyNATSContextHeaders(t *testing.T) {
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		TraceId:      "trace-1",
		RequestUser:  "alice",
		ClientSource: contextx.ClientSourceScheduledTask,
		SourceType:   contextx.SourceTypeScheduledTask,
		SourceRef:    "timer_task:1:execution:2",
	})
	msg := nats.NewMsg("timer.test")
	applyNATSContextHeaders(msg, ctx)

	if got := msg.Header.Get(contextx.TraceIdHeader); got != "trace-1" {
		t.Fatalf("trace header = %q", got)
	}
	if got := msg.Header.Get(contextx.RequestUserHeader); got != "alice" {
		t.Fatalf("request user header = %q", got)
	}
	if got := msg.Header.Get(contextx.SourceRefHeader); got != "timer_task:1:execution:2" {
		t.Fatalf("source ref header = %q", got)
	}
}
