package scheduledsdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

type recordingExecutionAdapter struct {
	Adapter
	started         atomic.Int32
	finished        atomic.Int32
	heartbeats      atomic.Int32
	mu              sync.Mutex
	statuses        []ExecutionStatus
	startedRunIDs   []string
	finishedRunIDs  []string
	heartbeatRunIDs []string
}

func (a *recordingExecutionAdapter) MarkExecutionStarted(_ context.Context, req MarkExecutionStartedRequest) error {
	a.started.Add(1)
	a.mu.Lock()
	a.startedRunIDs = append(a.startedRunIDs, req.ExecutorRunID)
	a.mu.Unlock()
	return nil
}

func (a *recordingExecutionAdapter) MarkExecutionHeartbeat(_ context.Context, req MarkExecutionHeartbeatRequest) error {
	a.heartbeats.Add(1)
	a.mu.Lock()
	a.heartbeatRunIDs = append(a.heartbeatRunIDs, req.ExecutorRunID)
	a.mu.Unlock()
	return nil
}

func (a *recordingExecutionAdapter) MarkExecutionFinished(_ context.Context, req MarkExecutionFinishedRequest) error {
	a.finished.Add(1)
	a.mu.Lock()
	a.statuses = append(a.statuses, req.Status)
	a.finishedRunIDs = append(a.finishedRunIDs, req.ExecutorRunID)
	a.mu.Unlock()
	return nil
}

func TestWorkerRejectsForgedExecutionRequestAndReplay(t *testing.T) {
	secret := strings.Repeat("s", 32)
	workerAuth, err := NewWorkerNATSAuth(secret)
	if err != nil {
		t.Fatal(err)
	}
	schedulerAuth, err := NewSchedulerNATSAuth(secret)
	if err != nil {
		t.Fatal(err)
	}
	adapter := &recordingExecutionAdapter{}
	var handlerCalls atomic.Int32
	var capabilityInjections atomic.Int32
	var reported error
	worker := &Worker{
		client:      NewClient(Options{Adapter: adapter}),
		executorKey: "agent.session",
		workerID:    "agent-worker-test",
		handler: func(ctx context.Context, _ ExecutionRequestedEvent) (*ExecutionResult, error) {
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://gateway.internal/workspace", nil)
			delegated, err := controlauth.ApplyDelegatedHTTPRequestSignature(req, nil)
			if err != nil || !delegated {
				return nil, fmt.Errorf("verified handler missing delegation capability: delegated=%v err=%v", delegated, err)
			}
			handlerCalls.Add(1)
			return &ExecutionResult{}, nil
		},
		verifiedContext: func(ctx context.Context) context.Context {
			capabilityInjections.Add(1)
			return controlauth.WithDelegatedHTTPRequestSigner(ctx, func(_ *http.Request, _ []byte) (bool, error) {
				return true, nil
			})
		},
		heartbeatInterval: -1,
		messageVerifier:   workerAuth.MessageVerifier,
		onError:           func(_ context.Context, err error) { reported = err },
	}
	eventData, err := json.Marshal(ExecutionRequestedEvent{TaskID: 1, ExecutionID: 2, ExecutorKey: "agent.session"})
	if err != nil {
		t.Fatal(err)
	}

	unsigned := nats.NewMsg(subjects.TimerExecutionRequestedSubject("agent.session"))
	unsigned.Data = eventData
	worker.handleMessage(unsigned)
	if handlerCalls.Load() != 0 || adapter.started.Load() != 0 {
		t.Fatalf("unsigned request executed: handler=%d started=%d", handlerCalls.Load(), adapter.started.Load())
	}
	if capabilityInjections.Load() != 0 {
		t.Fatalf("unsigned request obtained delegation capability %d times", capabilityInjections.Load())
	}
	if !errors.Is(reported, controlauth.ErrMissingMetadata) {
		t.Fatalf("unsigned request error = %v, want ErrMissingMetadata", reported)
	}

	tampered := nats.NewMsg(subjects.TimerExecutionRequestedSubject("agent.session"))
	tampered.Data = eventData
	if err := controlauth.SignNATSMessage(tampered, schedulerAuth.MessageSigner); err != nil {
		t.Fatal(err)
	}
	tampered.Data = []byte(`{"task_id":1,"execution_id":999,"executor_key":"agent.session"}`)
	reported = nil
	worker.handleMessage(tampered)
	if handlerCalls.Load() != 0 || adapter.started.Load() != 0 {
		t.Fatalf("tampered request executed: handler=%d started=%d", handlerCalls.Load(), adapter.started.Load())
	}
	if capabilityInjections.Load() != 0 {
		t.Fatalf("tampered request obtained delegation capability %d times", capabilityInjections.Load())
	}
	if !errors.Is(reported, controlauth.ErrInvalidSignature) {
		t.Fatalf("tampered request error = %v, want ErrInvalidSignature", reported)
	}

	reported = nil
	valid := nats.NewMsg(subjects.TimerExecutionRequestedSubject("agent.session"))
	valid.Data = eventData
	if err := controlauth.SignNATSMessage(valid, schedulerAuth.MessageSigner); err != nil {
		t.Fatal(err)
	}
	worker.handleMessage(valid)
	waitForWorkerCondition(t, func() bool {
		return handlerCalls.Load() == 1 && adapter.started.Load() == 1 && adapter.finished.Load() == 1
	})
	if reported != nil {
		t.Fatalf("valid request reported error: %v", reported)
	}
	if capabilityInjections.Load() != 1 {
		t.Fatalf("valid request capability injections = %d, want 1", capabilityInjections.Load())
	}
	adapter.mu.Lock()
	if len(adapter.startedRunIDs) != 1 || len(adapter.finishedRunIDs) != 1 || adapter.startedRunIDs[0] == "" || adapter.startedRunIDs[0] != adapter.finishedRunIDs[0] {
		t.Fatalf("executor run id binding: started=%v finished=%v", adapter.startedRunIDs, adapter.finishedRunIDs)
	}
	adapter.mu.Unlock()

	worker.handleMessage(valid)
	if handlerCalls.Load() != 1 || adapter.started.Load() != 1 {
		t.Fatalf("replayed request executed: handler=%d started=%d", handlerCalls.Load(), adapter.started.Load())
	}
	if capabilityInjections.Load() != 1 {
		t.Fatalf("replayed request obtained another capability; injections=%d", capabilityInjections.Load())
	}
	if !errors.Is(reported, controlauth.ErrReplay) {
		t.Fatalf("replayed request error = %v, want ErrReplay", reported)
	}
}

func TestWorkerAuthenticatesQueuedMessageBeforeLongHandlerCompletes(t *testing.T) {
	secret := strings.Repeat("q", 32)
	schedulerAuth, err := NewSchedulerNATSAuth(secret)
	if err != nil {
		t.Fatal(err)
	}
	workerAuth, err := NewWorkerNATSAuth(secret)
	if err != nil {
		t.Fatal(err)
	}
	adapter := &recordingExecutionAdapter{}
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var orderMu sync.Mutex
	var order []int64
	worker := &Worker{
		client:                  NewClient(Options{Adapter: adapter}),
		executorKey:             "agent.session",
		workerID:                "agent-worker-queue-test",
		heartbeatInterval:       -1,
		messageVerifier:         workerAuth.MessageVerifier,
		maxConcurrentExecutions: 1,
		handler: func(_ context.Context, event ExecutionRequestedEvent) (*ExecutionResult, error) {
			if event.ExecutionID == 1 {
				close(firstStarted)
				<-releaseFirst
			}
			orderMu.Lock()
			order = append(order, event.ExecutionID)
			orderMu.Unlock()
			return &ExecutionResult{}, nil
		},
	}
	message := func(executionID int64) *nats.Msg {
		t.Helper()
		data, err := json.Marshal(ExecutionRequestedEvent{
			TaskID:      executionID,
			ExecutionID: executionID,
			ExecutorKey: "agent.session",
		})
		if err != nil {
			t.Fatal(err)
		}
		msg := nats.NewMsg(subjects.TimerExecutionRequestedSubject("agent.session"))
		msg.Data = data
		if err := controlauth.SignNATSMessage(msg, schedulerAuth.MessageSigner); err != nil {
			t.Fatal(err)
		}
		return msg
	}
	first := message(1)
	second := message(2)
	callbacksReturned := make(chan struct{})
	go func() {
		worker.handleMessage(first)
		worker.handleMessage(second)
		close(callbacksReturned)
	}()

	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first handler did not start")
	}
	select {
	case <-callbacksReturned:
		// Both short-lived signatures were consumed before the long handler
		// released the single execution slot.
	case <-time.After(200 * time.Millisecond):
		t.Fatal("subscription callback remained blocked behind long handler")
	}
	close(releaseFirst)
	waitForWorkerCondition(t, func() bool {
		orderMu.Lock()
		defer orderMu.Unlock()
		return len(order) == 2
	})
	orderMu.Lock()
	defer orderMu.Unlock()
	if order[0] != 1 || order[1] != 2 {
		t.Fatalf("execution order = %v, want [1 2]", order)
	}
}

func TestWorkerClaimsAndHeartbeatsWhileWaitingForExecutionSlot(t *testing.T) {
	secret := strings.Repeat("c", 32)
	schedulerAuth, err := NewSchedulerNATSAuth(secret)
	if err != nil {
		t.Fatal(err)
	}
	workerAuth, err := NewWorkerNATSAuth(secret)
	if err != nil {
		t.Fatal(err)
	}
	adapter := &recordingExecutionAdapter{}
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var secondHandlerCalls atomic.Int32
	worker := &Worker{
		client:                  NewClient(Options{Adapter: adapter}),
		executorKey:             "agent.session",
		workerID:                "queued-claim-worker-test",
		heartbeatInterval:       time.Millisecond,
		messageVerifier:         workerAuth.MessageVerifier,
		maxConcurrentExecutions: 1,
		maxQueuedExecutions:     2,
		handler: func(_ context.Context, event ExecutionRequestedEvent) (*ExecutionResult, error) {
			if event.ExecutionID == 1 {
				close(firstStarted)
				<-releaseFirst
			} else {
				secondHandlerCalls.Add(1)
			}
			return &ExecutionResult{}, nil
		},
	}
	message := func(executionID int64) *nats.Msg {
		t.Helper()
		data, marshalErr := json.Marshal(ExecutionRequestedEvent{
			TaskID:      executionID,
			ExecutionID: executionID,
			ExecutorKey: "agent.session",
		})
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		msg := nats.NewMsg(subjects.TimerExecutionRequestedSubject("agent.session"))
		msg.Data = data
		if signErr := controlauth.SignNATSMessage(msg, schedulerAuth.MessageSigner); signErr != nil {
			t.Fatal(signErr)
		}
		return msg
	}

	worker.handleMessage(message(1))
	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first handler did not start")
	}
	worker.handleMessage(message(2))
	waitForWorkerCondition(t, func() bool { return adapter.started.Load() == 2 })
	if secondHandlerCalls.Load() != 0 {
		t.Fatal("second handler ran before the only execution slot was released")
	}
	adapter.mu.Lock()
	queuedRunID := adapter.startedRunIDs[1]
	adapter.mu.Unlock()
	waitForWorkerCondition(t, func() bool {
		adapter.mu.Lock()
		defer adapter.mu.Unlock()
		for _, runID := range adapter.heartbeatRunIDs {
			if runID == queuedRunID {
				return true
			}
		}
		return false
	})

	close(releaseFirst)
	waitForWorkerCondition(t, func() bool { return secondHandlerCalls.Load() == 1 })
}

func TestWorkerRejectsOversizedSignedEventBeforeDecodeOrQueue(t *testing.T) {
	secret := strings.Repeat("o", 32)
	schedulerAuth, err := NewSchedulerNATSAuth(secret)
	if err != nil {
		t.Fatal(err)
	}
	workerAuth, err := NewWorkerNATSAuth(secret)
	if err != nil {
		t.Fatal(err)
	}
	var reported error
	worker := &Worker{
		executorKey:     "agent.session",
		messageVerifier: workerAuth.MessageVerifier,
		handler: func(context.Context, ExecutionRequestedEvent) (*ExecutionResult, error) {
			t.Fatal("oversized event reached handler")
			return nil, nil
		},
		onError: func(_ context.Context, err error) { reported = err },
	}
	msg := nats.NewMsg(subjects.TimerExecutionRequestedSubject("agent.session"))
	msg.Data = make([]byte, maxExecutionMessageBytes+1)
	if err := controlauth.SignNATSMessage(msg, schedulerAuth.MessageSigner); err != nil {
		t.Fatal(err)
	}
	worker.handleMessage(msg)
	if !errors.Is(reported, ErrExecutionMessageTooLarge) {
		t.Fatalf("oversized event error = %v, want ErrExecutionMessageTooLarge", reported)
	}
	worker.executionMu.Lock()
	defer worker.executionMu.Unlock()
	if len(worker.executionQueue) != 0 || len(worker.executionKnown) != 0 {
		t.Fatalf("oversized event entered queue: queued=%d known=%d", len(worker.executionQueue), len(worker.executionKnown))
	}
}

func TestWorkerAcceptsMaximumExecutorPayloadWithEnvelope(t *testing.T) {
	secret := strings.Repeat("m", 32)
	schedulerAuth, err := NewSchedulerNATSAuth(secret)
	if err != nil {
		t.Fatal(err)
	}
	workerAuth, err := NewWorkerNATSAuth(secret)
	if err != nil {
		t.Fatal(err)
	}
	adapter := &recordingExecutionAdapter{}
	handled := make(chan struct{}, 1)
	worker := &Worker{
		client:              NewClient(Options{Adapter: adapter}),
		executorKey:         "agent.session",
		workerID:            "payload-boundary-worker",
		heartbeatInterval:   -1,
		messageVerifier:     workerAuth.MessageVerifier,
		maxQueuedExecutions: 1,
		handler: func(context.Context, ExecutionRequestedEvent) (*ExecutionResult, error) {
			handled <- struct{}{}
			return &ExecutionResult{}, nil
		},
	}
	payload := make([]byte, MaxExecutorPayloadBytes)
	payload[0] = '"'
	for i := 1; i < len(payload)-1; i++ {
		payload[i] = 'a'
	}
	payload[len(payload)-1] = '"'
	data, err := json.Marshal(ExecutionRequestedEvent{
		TaskID:          1,
		ExecutionID:     1,
		ExecutorKey:     "agent.session",
		ExecutorPayload: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > maxExecutionMessageBytes {
		t.Fatalf("boundary event envelope unexpectedly exceeds message cap: size=%d cap=%d", len(data), maxExecutionMessageBytes)
	}
	msg := nats.NewMsg(subjects.TimerExecutionRequestedSubject("agent.session"))
	msg.Data = data
	if err := controlauth.SignNATSMessage(msg, schedulerAuth.MessageSigner); err != nil {
		t.Fatal(err)
	}
	worker.handleMessage(msg)
	select {
	case <-handled:
	case <-time.After(time.Second):
		t.Fatal("maximum legal executor payload was not handled")
	}
}

func TestWorkerBoundsQueueAndDeduplicatesExecution(t *testing.T) {
	secret := strings.Repeat("b", 32)
	schedulerAuth, err := NewSchedulerNATSAuth(secret)
	if err != nil {
		t.Fatal(err)
	}
	workerAuth, err := NewWorkerNATSAuth(secret)
	if err != nil {
		t.Fatal(err)
	}
	adapter := &recordingExecutionAdapter{}
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	reported := make(chan error, 1)
	var orderMu sync.Mutex
	var order []int64
	worker := &Worker{
		client:                  NewClient(Options{Adapter: adapter}),
		executorKey:             "agent.session",
		workerID:                "bounded-worker-test",
		heartbeatInterval:       -1,
		messageVerifier:         workerAuth.MessageVerifier,
		maxConcurrentExecutions: 1,
		maxQueuedExecutions:     1,
		onError: func(_ context.Context, err error) {
			select {
			case reported <- err:
			default:
			}
		},
		handler: func(_ context.Context, event ExecutionRequestedEvent) (*ExecutionResult, error) {
			if event.ExecutionID == 1 {
				close(firstStarted)
				<-releaseFirst
			}
			orderMu.Lock()
			order = append(order, event.ExecutionID)
			orderMu.Unlock()
			return &ExecutionResult{}, nil
		},
	}
	message := func(taskID, executionID int64) *nats.Msg {
		t.Helper()
		data, err := json.Marshal(ExecutionRequestedEvent{
			TaskID:      taskID,
			ExecutionID: executionID,
			ExecutorKey: "agent.session",
		})
		if err != nil {
			t.Fatal(err)
		}
		msg := nats.NewMsg(subjects.TimerExecutionRequestedSubject("agent.session"))
		msg.Data = data
		if err := controlauth.SignNATSMessage(msg, schedulerAuth.MessageSigner); err != nil {
			t.Fatal(err)
		}
		return msg
	}

	worker.handleMessage(message(1, 1))
	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first handler did not start")
	}
	worker.handleMessage(message(1, 1)) // different signature, same execution
	worker.handleMessage(message(2, 2)) // fills the one waiting slot
	worker.handleMessage(message(3, 3)) // rejected so Timer may retry later
	select {
	case err := <-reported:
		if !errors.Is(err, ErrExecutionQueueFull) {
			t.Fatalf("queue error = %v, want ErrExecutionQueueFull", err)
		}
	case <-time.After(time.Second):
		t.Fatal("full execution queue did not report an error")
	}
	worker.executionMu.Lock()
	queued := len(worker.executionQueue)
	worker.executionMu.Unlock()
	if queued != 1 {
		t.Fatalf("queued executions = %d, want hard bound 1", queued)
	}

	close(releaseFirst)
	waitForWorkerCondition(t, func() bool {
		orderMu.Lock()
		defer orderMu.Unlock()
		return len(order) == 2
	})
	orderMu.Lock()
	defer orderMu.Unlock()
	if order[0] != 1 || order[1] != 2 {
		t.Fatalf("execution order = %v, want [1 2]", order)
	}
}

func TestWorkerPanicFinishesFailedStopsHeartbeatAndContinuesQueue(t *testing.T) {
	secret := strings.Repeat("p", 32)
	schedulerAuth, err := NewSchedulerNATSAuth(secret)
	if err != nil {
		t.Fatal(err)
	}
	workerAuth, err := NewWorkerNATSAuth(secret)
	if err != nil {
		t.Fatal(err)
	}
	adapter := &recordingExecutionAdapter{}
	worker := &Worker{
		client:                  NewClient(Options{Adapter: adapter}),
		executorKey:             "agent.session",
		workerID:                "panic-worker-test",
		heartbeatInterval:       time.Millisecond,
		messageVerifier:         workerAuth.MessageVerifier,
		maxConcurrentExecutions: 1,
		maxQueuedExecutions:     2,
		handler: func(_ context.Context, event ExecutionRequestedEvent) (*ExecutionResult, error) {
			if event.ExecutionID == 1 {
				panic("boom")
			}
			return &ExecutionResult{}, nil
		},
	}
	message := func(executionID int64) *nats.Msg {
		t.Helper()
		data, err := json.Marshal(ExecutionRequestedEvent{
			TaskID:      executionID,
			ExecutionID: executionID,
			ExecutorKey: "agent.session",
		})
		if err != nil {
			t.Fatal(err)
		}
		msg := nats.NewMsg(subjects.TimerExecutionRequestedSubject("agent.session"))
		msg.Data = data
		if err := controlauth.SignNATSMessage(msg, schedulerAuth.MessageSigner); err != nil {
			t.Fatal(err)
		}
		return msg
	}
	worker.handleMessage(message(1))
	worker.handleMessage(message(2))
	waitForWorkerCondition(t, func() bool { return adapter.finished.Load() == 2 })
	adapter.mu.Lock()
	statuses := append([]ExecutionStatus(nil), adapter.statuses...)
	adapter.mu.Unlock()
	if len(statuses) != 2 || statuses[0] != ExecutionStatusFailed || statuses[1] != ExecutionStatusSuccess {
		t.Fatalf("finished statuses = %v, want [failed success]", statuses)
	}
	heartbeats := adapter.heartbeats.Load()
	time.Sleep(10 * time.Millisecond)
	if got := adapter.heartbeats.Load(); got != heartbeats {
		t.Fatalf("heartbeat continued after panic completion: before=%d after=%d", heartbeats, got)
	}
}

func waitForWorkerCondition(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("timed out waiting for worker condition")
}
