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

	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

type ExecutionHandler func(ctx context.Context, event ExecutionRequestedEvent) (*ExecutionResult, error)

type WorkerOptions struct {
	Client            *Client
	NATSConn          *nats.Conn
	ExecutorKey       string
	WorkerID          string
	QueueGroup        string
	Handler           ExecutionHandler
	HeartbeatInterval time.Duration
	Concurrency       int
	OnError           func(context.Context, error)
}

type Worker struct {
	client            *Client
	natsConn          *nats.Conn
	executorKey       string
	workerID          string
	queueGroup        string
	handler           ExecutionHandler
	heartbeatInterval time.Duration
	concurrency       int
	onError           func(context.Context, error)

	mu   sync.Mutex
	subs []*nats.Subscription
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
	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}
	return &Worker{
		client:            opts.Client,
		natsConn:          opts.NATSConn,
		executorKey:       executorKey,
		workerID:          workerID,
		queueGroup:        queueGroup,
		handler:           opts.Handler,
		heartbeatInterval: heartbeatInterval,
		concurrency:       concurrency,
		onError:           opts.OnError,
	}, nil
}

func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.subs) > 0 {
		return fmt.Errorf("scheduledsdk: worker already started")
	}
	subject := subjects.TimerExecutionRequestedSubject(w.executorKey)
	for i := 0; i < w.concurrency; i++ {
		sub, err := w.natsConn.QueueSubscribe(subject, w.queueGroup, w.handleMessage)
		if err != nil {
			for _, started := range w.subs {
				_ = started.Unsubscribe()
			}
			w.subs = nil
			return err
		}
		w.subs = append(w.subs, sub)
	}
	go func() {
		<-ctx.Done()
		_ = w.Stop()
	}()
	return nil
}

func (w *Worker) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.subs) == 0 {
		return nil
	}
	var stopErr error
	for _, sub := range w.subs {
		stopErr = errors.Join(stopErr, sub.Unsubscribe())
	}
	w.subs = nil
	return stopErr
}

func (w *Worker) handleMessage(msg *nats.Msg) {
	ctx := context.Background()
	var event ExecutionRequestedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		w.reportError(ctx, err)
		return
	}
	if strings.TrimSpace(event.ExecutorKey) != w.executorKey {
		w.reportError(ctx, fmt.Errorf("scheduledsdk: received executor_key %q on worker %q", event.ExecutorKey, w.executorKey))
		return
	}
	ctx = event.WithAuditContext(ctx)

	startedAt := time.Now()
	if err := w.client.MarkExecutionStarted(ctx, MarkExecutionStartedRequest{
		TaskID:      event.TaskID,
		ExecutionID: event.ExecutionID,
		WorkerID:    w.workerID,
		StartedAt:   startedAt,
	}); err != nil {
		w.reportError(ctx, err)
		return
	}

	handlerCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	if w.heartbeatInterval > 0 {
		go w.heartbeatLoop(handlerCtx, event, done)
	}

	result, err := w.handler(handlerCtx, event)
	finishedAt := time.Now()
	cancel()
	close(done)

	if result == nil {
		result = &ExecutionResult{}
	}
	if result.DurationMillis <= 0 {
		result.DurationMillis = finishedAt.Sub(startedAt).Milliseconds()
	}
	if err != nil {
		result.Status = ExecutionStatusFailed
		result.ErrorMessage = err.Error()
	} else if result.Status == "" {
		result.Status = ExecutionStatusSuccess
	}
	if finishErr := w.client.MarkExecutionFinished(ctx, MarkExecutionFinishedRequest{
		TaskID:         event.TaskID,
		ExecutionID:    event.ExecutionID,
		Status:         result.Status,
		ExecutorRunID:  result.ExecutorRunID,
		FinishedAt:     finishedAt,
		DurationMillis: result.DurationMillis,
		OutputSummary:  result.OutputSummary,
		ResultPayload:  result.ResultPayload,
		ErrorMessage:   result.ErrorMessage,
	}); finishErr != nil {
		w.reportError(ctx, finishErr)
	}
}

func (w *Worker) heartbeatLoop(ctx context.Context, event ExecutionRequestedEvent, done <-chan struct{}) {
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
				TaskID:      event.TaskID,
				ExecutionID: event.ExecutionID,
				WorkerID:    w.workerID,
				HeartbeatAt: t,
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
