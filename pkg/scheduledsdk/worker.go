package scheduledsdk

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type ExecutionHandler func(ctx context.Context, event ExecutionRequestedEvent) (*ExecutionResult, error)

type ExecutionResult struct {
	Status         ExecutionStatus `json:"status,omitempty"`
	ExecutorRunID  string          `json:"executor_run_id,omitempty"`
	OutputSummary  string          `json:"output_summary,omitempty"`
	ResultPayload  json.RawMessage `json:"result_payload,omitempty"`
	ErrorMessage   string          `json:"error_message,omitempty"`
	DurationMillis int64           `json:"duration_millis,omitempty"`
}

type WorkerOptions struct {
	Client      *Client
	NATSConn    *nats.Conn
	ExecutorKey string
	WorkerID    string
	QueueGroup  string
	Handler     ExecutionHandler
	OnError     func(context.Context, error)
}

type Worker struct {
	client      *Client
	natsConn    *nats.Conn
	executorKey string
	workerID    string
	queueGroup  string
	handler     ExecutionHandler
	onError     func(context.Context, error)

	mu  sync.Mutex
	sub *nats.Subscription
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
		queueGroup = "timer-worker." + sanitizeQueueToken(executorKey)
	}
	return &Worker{
		client:      opts.Client,
		natsConn:    opts.NATSConn,
		executorKey: executorKey,
		workerID:    workerID,
		queueGroup:  queueGroup,
		handler:     opts.Handler,
		onError:     opts.OnError,
	}, nil
}

func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.sub != nil {
		return fmt.Errorf("scheduledsdk: worker already started")
	}
	sub, err := w.natsConn.QueueSubscribe(SubjectExecutionRequested, w.queueGroup, w.handleMessage)
	if err != nil {
		return err
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
	var event ExecutionRequestedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		w.reportError(ctx, err)
		return
	}
	if event.ExecutorKey != w.executorKey {
		return
	}

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

	result, err := w.handler(ctx, event)
	finishedAt := time.Now()
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

func (w *Worker) reportError(ctx context.Context, err error) {
	if w.onError != nil {
		w.onError(ctx, err)
	}
}

func defaultWorkerID(executorKey string) string {
	return sanitizeQueueToken(executorKey) + "-" + fmt.Sprint(time.Now().UnixNano())
}

func sanitizeQueueToken(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.ReplaceAll(value, "/", "_")
	return value
}
