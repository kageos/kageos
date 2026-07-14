package scheduledsdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

type ExecutionHandler func(ctx context.Context, event ExecutionRequestedEvent) (*ExecutionResult, error)

var (
	ErrExecutionQueueFull        = errors.New("scheduledsdk: verified execution queue is full")
	ErrExecutionMessageTooLarge  = errors.New("scheduledsdk: execution request message is too large")
	ErrExecutorPayloadTooLarge   = errors.New("scheduledsdk: executor payload is too large")
	ErrExecutionMetadataTooLarge = errors.New("scheduledsdk: execution metadata is too large")
)

const (
	defaultMaxQueuedExecutions = 128
	hardMaxQueuedExecutions    = 1024
	maxExecutionMessageBytes   = 512 << 10
	maxPendingExecutionBytes   = 256 << 20
)

type WorkerOptions struct {
	Client            *Client
	NATSConn          *nats.Conn
	ExecutorKey       string
	WorkerID          string
	QueueGroup        string
	Handler           ExecutionHandler
	HeartbeatInterval time.Duration
	OnError           func(context.Context, error)
	MessageVerifier   *controlauth.Verifier
	VerifiedContext   func(context.Context) context.Context
	// MaxConcurrentExecutions limits handler concurrency after messages have
	// already passed authentication and decoding. The default is one, matching
	// the historical per-subscription execution semantics.
	MaxConcurrentExecutions int
	// MaxQueuedExecutions bounds authenticated, decoded events waiting for an
	// execution slot. Full queues drop the local copy so Timer can lease/retry.
	MaxQueuedExecutions int
}

type Worker struct {
	client                  *Client
	natsConn                *nats.Conn
	executorKey             string
	workerID                string
	queueGroup              string
	handler                 ExecutionHandler
	heartbeatInterval       time.Duration
	onError                 func(context.Context, error)
	messageVerifier         *controlauth.Verifier
	verifiedContext         func(context.Context) context.Context
	maxConcurrentExecutions int
	maxQueuedExecutions     int

	mu  sync.Mutex
	sub *nats.Subscription

	executionMu       sync.Mutex
	executionQueue    []*queuedExecution
	executionKnown    map[executionIdentity]struct{}
	executionClaiming int
	executionRunning  int
}

type executionIdentity struct {
	taskID      int64
	executionID int64
}

type queuedExecution struct {
	event         ExecutionRequestedEvent
	executorRunID string
	startedAt     time.Time
	commandCtx    context.Context
	handlerCtx    context.Context
	cancel        context.CancelFunc
	done          chan struct{}
	stopOnce      sync.Once
}

func (q *queuedExecution) stopHeartbeat() {
	q.stopOnce.Do(func() {
		q.cancel()
		close(q.done)
	})
}

func NewWorker(opts WorkerOptions) (*Worker, error) {
	if opts.Client == nil {
		return nil, fmt.Errorf("scheduledsdk: worker client is required")
	}
	if opts.NATSConn == nil {
		return nil, fmt.Errorf("scheduledsdk: worker nats connection is required")
	}
	executorKey := strings.TrimSpace(opts.ExecutorKey)
	if executorKey == "" {
		return nil, fmt.Errorf("scheduledsdk: worker executor_key is required")
	}
	if opts.Handler == nil {
		return nil, fmt.Errorf("scheduledsdk: worker handler is required")
	}
	if opts.MessageVerifier == nil {
		return nil, fmt.Errorf("scheduledsdk: worker message verifier is required")
	}
	workerID := strings.TrimSpace(opts.WorkerID)
	if workerID == "" {
		workerID = defaultWorkerID(executorKey)
	}
	queueGroup := strings.TrimSpace(opts.QueueGroup)
	if queueGroup == "" {
		queueGroup = subjects.TimerWorkerQueueGroup(executorKey)
	}
	heartbeatInterval := opts.HeartbeatInterval
	if heartbeatInterval == 0 {
		heartbeatInterval = 30 * time.Second
	}
	maxConcurrentExecutions := opts.MaxConcurrentExecutions
	if maxConcurrentExecutions <= 0 {
		maxConcurrentExecutions = 1
	}
	maxQueuedExecutions := opts.MaxQueuedExecutions
	if maxQueuedExecutions <= 0 {
		maxQueuedExecutions = defaultMaxQueuedExecutions
	}
	if maxQueuedExecutions > hardMaxQueuedExecutions {
		maxQueuedExecutions = hardMaxQueuedExecutions
	}
	return &Worker{
		client:                  opts.Client,
		natsConn:                opts.NATSConn,
		executorKey:             executorKey,
		workerID:                workerID,
		queueGroup:              queueGroup,
		handler:                 opts.Handler,
		heartbeatInterval:       heartbeatInterval,
		onError:                 opts.OnError,
		messageVerifier:         opts.MessageVerifier,
		verifiedContext:         opts.VerifiedContext,
		maxConcurrentExecutions: maxConcurrentExecutions,
		maxQueuedExecutions:     maxQueuedExecutions,
	}, nil
}

func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.sub != nil {
		return fmt.Errorf("scheduledsdk: worker already started")
	}
	subject := subjects.TimerExecutionRequestedSubject(w.executorKey)
	sub, err := w.natsConn.QueueSubscribe(subject, w.queueGroup, w.handleMessage)
	if err != nil {
		return err
	}
	pendingMessages := w.maxQueuedExecutions + w.maxConcurrentExecutions
	if pendingMessages <= 0 {
		pendingMessages = defaultMaxQueuedExecutions + 1
	}
	pendingBytes := pendingMessages * (maxExecutionMessageBytes + 4096)
	if pendingBytes > maxPendingExecutionBytes {
		pendingBytes = maxPendingExecutionBytes
	}
	if err := sub.SetPendingLimits(pendingMessages, pendingBytes); err != nil {
		_ = sub.Unsubscribe()
		return fmt.Errorf("scheduledsdk: set worker subscription pending limits: %w", err)
	}
	w.sub = sub
	go func() {
		<-ctx.Done()
		_ = w.Stop()
	}()
	return nil
}

func (w *Worker) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.sub == nil {
		return nil
	}
	err := w.sub.Unsubscribe()
	w.sub = nil
	return err
}

func (w *Worker) handleMessage(msg *nats.Msg) {
	ctx := context.Background()
	if err := controlauth.VerifyNATSMessage(msg, w.messageVerifier); err != nil {
		w.reportError(ctx, fmt.Errorf("scheduledsdk: authenticate execution request: %w", err))
		return
	}
	ctx = contextx.NatsTraceContext(msg)
	if len(msg.Data) > maxExecutionMessageBytes {
		w.reportError(ctx, fmt.Errorf("%w: size=%d limit=%d", ErrExecutionMessageTooLarge, len(msg.Data), maxExecutionMessageBytes))
		return
	}
	var event ExecutionRequestedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		w.reportError(ctx, err)
		return
	}
	if len(event.ExecutorPayload) > MaxExecutorPayloadBytes {
		w.reportError(ctx, fmt.Errorf("%w: size=%d limit=%d", ErrExecutorPayloadTooLarge, len(event.ExecutorPayload), MaxExecutorPayloadBytes))
		return
	}
	if err := ValidateExecutionMetadata(event.Metadata); err != nil {
		w.reportError(ctx, fmt.Errorf("%w: %v", ErrExecutionMetadataTooLarge, err))
		return
	}
	if strings.TrimSpace(event.ExecutorKey) != w.executorKey {
		w.reportError(ctx, fmt.Errorf("scheduledsdk: received executor_key %q on worker %q", event.ExecutorKey, w.executorKey))
		return
	}
	// NATS invokes an async subscription callback serially. Authentication must
	// complete inside that callback, but long business execution must not block
	// the next message until its short-lived signature has expired.
	w.enqueueVerifiedExecution(ctx, event)
}

func (w *Worker) enqueueVerifiedExecution(ctx context.Context, event ExecutionRequestedEvent) {
	identity := executionIdentity{taskID: event.TaskID, executionID: event.ExecutionID}
	w.executionMu.Lock()
	if w.executionKnown == nil {
		w.executionKnown = make(map[executionIdentity]struct{})
	}
	if _, exists := w.executionKnown[identity]; exists {
		w.executionMu.Unlock()
		return
	}
	maxQueued := w.maxQueuedExecutions
	if maxQueued <= 0 {
		maxQueued = defaultMaxQueuedExecutions
	}
	if len(w.executionQueue)+w.executionClaiming >= maxQueued {
		w.executionMu.Unlock()
		w.reportError(ctx, fmt.Errorf("%w: limit=%d", ErrExecutionQueueFull, maxQueued))
		return
	}
	w.executionKnown[identity] = struct{}{}
	w.executionClaiming++
	w.executionMu.Unlock()

	if w.verifiedContext != nil {
		ctx = w.verifiedContext(ctx)
	}
	ctx = event.WithAuditContext(ctx)
	executorRunID := uuid.NewString()
	startedAt := time.Now()
	if err := w.client.MarkExecutionStarted(ctx, MarkExecutionStartedRequest{
		TaskID:        event.TaskID,
		ExecutionID:   event.ExecutionID,
		WorkerID:      w.workerID,
		StartedAt:     startedAt,
		ExecutorRunID: executorRunID,
	}); err != nil {
		w.executionMu.Lock()
		w.executionClaiming--
		delete(w.executionKnown, identity)
		w.executionMu.Unlock()
		w.reportError(ctx, err)
		return
	}

	handlerCtx, cancel := context.WithCancel(ctx)
	queued := &queuedExecution{
		event:         event,
		executorRunID: executorRunID,
		startedAt:     startedAt,
		commandCtx:    ctx,
		handlerCtx:    handlerCtx,
		cancel:        cancel,
		done:          make(chan struct{}),
	}
	if w.heartbeatInterval > 0 {
		go w.heartbeatLoop(handlerCtx, event, executorRunID, queued.done)
	}

	w.executionMu.Lock()
	w.executionClaiming--
	w.executionQueue = append(w.executionQueue, queued)
	limit := w.maxConcurrentExecutions
	if limit <= 0 {
		limit = 1
	}
	if w.executionRunning < limit {
		w.executionRunning++
		go w.executionLoop()
	}
	w.executionMu.Unlock()
}

func (w *Worker) executionLoop() {
	for {
		w.executionMu.Lock()
		if len(w.executionQueue) == 0 {
			w.executionRunning--
			w.executionMu.Unlock()
			return
		}
		queued := w.executionQueue[0]
		w.executionQueue[0] = nil
		w.executionQueue = w.executionQueue[1:]
		w.executionMu.Unlock()

		w.runQueuedExecution(queued)
	}
}

func (w *Worker) runQueuedExecution(queued *queuedExecution) {
	defer func() {
		w.executionMu.Lock()
		delete(w.executionKnown, executionIdentity{taskID: queued.event.TaskID, executionID: queued.event.ExecutionID})
		w.executionMu.Unlock()
	}()
	w.executeClaimedEvent(queued)
}

func (w *Worker) executeClaimedEvent(queued *queuedExecution) {
	defer queued.stopHeartbeat()
	result, err := w.callExecutionHandler(queued.handlerCtx, queued.event)
	finishedAt := time.Now()
	queued.stopHeartbeat()

	if result == nil {
		result = &ExecutionResult{}
	}
	result.ExecutorRunID = queued.executorRunID
	if result.DurationMillis <= 0 {
		result.DurationMillis = finishedAt.Sub(queued.startedAt).Milliseconds()
	}
	if err != nil {
		result.Status = ExecutionStatusFailed
		result.ErrorMessage = err.Error()
	} else if result.Status == "" {
		result.Status = ExecutionStatusSuccess
	}
	if finishErr := w.client.MarkExecutionFinished(queued.commandCtx, MarkExecutionFinishedRequest{
		TaskID:         queued.event.TaskID,
		ExecutionID:    queued.event.ExecutionID,
		Status:         result.Status,
		ExecutorRunID:  result.ExecutorRunID,
		FinishedAt:     finishedAt,
		DurationMillis: result.DurationMillis,
		OutputSummary:  result.OutputSummary,
		ResultPayload:  result.ResultPayload,
		ErrorMessage:   result.ErrorMessage,
	}); finishErr != nil {
		w.reportError(queued.commandCtx, finishErr)
	}
}

func (w *Worker) callExecutionHandler(ctx context.Context, event ExecutionRequestedEvent) (result *ExecutionResult, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = nil
			err = fmt.Errorf("scheduledsdk: execution handler panic: %v", recovered)
		}
	}()
	return w.handler(ctx, event)
}

func (w *Worker) heartbeatLoop(ctx context.Context, event ExecutionRequestedEvent, executorRunID string, done <-chan struct{}) {
	ticker := time.NewTicker(w.heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			if err := w.client.MarkExecutionHeartbeat(ctx, MarkExecutionHeartbeatRequest{
				TaskID:        event.TaskID,
				ExecutionID:   event.ExecutionID,
				WorkerID:      w.workerID,
				ExecutorRunID: executorRunID,
				HeartbeatAt:   t,
			}); err != nil {
				w.reportError(ctx, err)
			}
		}
	}
}

func (w *Worker) reportError(ctx context.Context, err error) {
	if w.onError != nil {
		w.onError(ctx, err)
	}
}

func defaultWorkerID(executorKey string) string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "unknown-host"
	}
	return fmt.Sprintf("%s-%s-%d", subjects.NormalizeTimerSubjectSuffix(executorKey), host, time.Now().UnixNano())
}
