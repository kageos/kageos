package scheduledsdk

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

var _ Adapter = (*NATSAdapter)(nil)

const defaultNATSAdapterTimeout = 5 * time.Second

type NATSAdapterOptions struct {
	Timeout time.Duration
}

type NATSAdapter struct {
	conn    *nats.Conn
	timeout time.Duration
}

func NewNATSAdapter(conn *nats.Conn, opts NATSAdapterOptions) *NATSAdapter {
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = defaultNATSAdapterTimeout
	}
	return &NATSAdapter{
		conn:    conn,
		timeout: timeout,
	}
}

func (a *NATSAdapter) CreateTask(context.Context, CreateTaskRequest) (*Task, error) {
	return nil, unsupportedNATSAdapterOperation("CreateTask")
}

func (a *NATSAdapter) UpdateTask(context.Context, int64, UpdateTaskRequest) (*Task, error) {
	return nil, unsupportedNATSAdapterOperation("UpdateTask")
}

func (a *NATSAdapter) PauseTask(context.Context, int64) error {
	return unsupportedNATSAdapterOperation("PauseTask")
}

func (a *NATSAdapter) ResumeTask(context.Context, int64) error {
	return unsupportedNATSAdapterOperation("ResumeTask")
}

func (a *NATSAdapter) CancelTask(context.Context, int64) error {
	return unsupportedNATSAdapterOperation("CancelTask")
}

func (a *NATSAdapter) DeleteTask(context.Context, int64) error {
	return unsupportedNATSAdapterOperation("DeleteTask")
}

func (a *NATSAdapter) RunNow(context.Context, int64) (*Execution, error) {
	return nil, unsupportedNATSAdapterOperation("RunNow")
}

func (a *NATSAdapter) GetTask(context.Context, int64) (*Task, error) {
	return nil, unsupportedNATSAdapterOperation("GetTask")
}

func (a *NATSAdapter) ListTasks(context.Context, ListTasksRequest) (*ListTasksResponse, error) {
	return nil, unsupportedNATSAdapterOperation("ListTasks")
}

func (a *NATSAdapter) GetExecution(context.Context, int64, int64) (*Execution, error) {
	return nil, unsupportedNATSAdapterOperation("GetExecution")
}

func (a *NATSAdapter) ListExecutions(context.Context, int64, ListExecutionsRequest) (*ListExecutionsResponse, error) {
	return nil, unsupportedNATSAdapterOperation("ListExecutions")
}

func (a *NATSAdapter) MarkExecutionStarted(ctx context.Context, req MarkExecutionStartedRequest) error {
	return a.request(ctx, subjects.TimerExecutionStartedCommandSubject, req)
}

func (a *NATSAdapter) MarkExecutionHeartbeat(ctx context.Context, req MarkExecutionHeartbeatRequest) error {
	return a.request(ctx, subjects.TimerExecutionHeartbeatCommandSubject, req)
}

func (a *NATSAdapter) MarkExecutionFinished(ctx context.Context, req MarkExecutionFinishedRequest) error {
	return a.request(ctx, subjects.TimerExecutionFinishedCommandSubject, req)
}

func (a *NATSAdapter) request(ctx context.Context, subject string, req interface{}) error {
	if a == nil || a.conn == nil {
		return fmt.Errorf("scheduledsdk: nats connection is required")
	}
	if strings.TrimSpace(subject) == "" {
		return fmt.Errorf("scheduledsdk: nats subject is required")
	}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	msg := nats.NewMsg(subject)
	msg.Data = data
	applyNATSContextHeaders(msg, ctx)

	requestCtx, cancel := natsAdapterRequestContext(ctx, a.timeout)
	defer cancel()

	resp, err := a.conn.RequestMsgWithContext(requestCtx, msg)
	if err != nil {
		return err
	}
	if len(resp.Data) == 0 {
		return nil
	}
	var out natsCommandResponse
	if err := json.Unmarshal(resp.Data, &out); err != nil {
		return err
	}
	if !out.OK {
		if strings.TrimSpace(out.Error) != "" {
			return fmt.Errorf("scheduledsdk: %s", out.Error)
		}
		return fmt.Errorf("scheduledsdk: nats command failed")
	}
	return nil
}

type natsCommandResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func unsupportedNATSAdapterOperation(name string) error {
	return fmt.Errorf("%w: %s is not supported by NATSAdapter", ErrUnsupported, name)
}

func natsAdapterRequestContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	if timeout <= 0 {
		timeout = defaultNATSAdapterTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

func applyNATSContextHeaders(msg *nats.Msg, ctx context.Context) {
	if msg == nil || ctx == nil {
		return
	}
	header := nats.Header{}
	if token := contextx.GetToken(ctx); token != "" {
		header.Set(contextx.TokenHeader, token)
	}
	if traceID := contextx.GetTraceId(ctx); traceID != "" {
		header.Set(contextx.TraceIdHeader, traceID)
	}
	if requestUser := contextx.GetRequestUser(ctx); requestUser != "" {
		header.Set(contextx.RequestUserHeader, requestUser)
	}
	if departmentFullPath := contextx.GetRequestDepartmentFullPath(ctx); departmentFullPath != "" {
		header.Set(contextx.DepartmentFullPathHeader, departmentFullPath)
	}
	if clientSource := contextx.GetClientSource(ctx); clientSource != "" {
		header.Set(contextx.ClientSourceHeader, clientSource)
	}
	if sourceType := contextx.GetSourceType(ctx); sourceType != "" {
		header.Set(contextx.SourceTypeHeader, sourceType)
	}
	if sourceRef := contextx.GetSourceRef(ctx); sourceRef != "" {
		header.Set(contextx.SourceRefHeader, sourceRef)
	}
	if len(header) > 0 {
		msg.Header = header
	}
}
