package scheduledsdk

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

var _ Adapter = (*NATSAdapter)(nil)

const defaultNATSAdapterTimeout = 5 * time.Second

type NATSAdapterOptions struct {
	Timeout          time.Duration
	CommandSigner    *controlauth.Signer
	ResponseVerifier *controlauth.Verifier
}

type NATSAdapter struct {
	conn     *nats.Conn
	timeout  time.Duration
	signer   *controlauth.Signer
	verifier *controlauth.Verifier
}

func NewNATSAdapter(conn *nats.Conn, opts NATSAdapterOptions) *NATSAdapter {
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = defaultNATSAdapterTimeout
	}
	return &NATSAdapter{
		conn:     conn,
		timeout:  timeout,
		signer:   opts.CommandSigner,
		verifier: opts.ResponseVerifier,
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
	msg.Reply = nats.NewInbox()
	applyNATSContextHeaders(msg, ctx)
	if err := controlauth.SignNATSMessage(msg, a.signer); err != nil {
		return fmt.Errorf("scheduledsdk: authenticate nats command: %w", err)
	}
	requestNonce := strings.TrimSpace(msg.Header.Get(controlauth.NATSNonceHeader))
	if requestNonce == "" {
		return fmt.Errorf("scheduledsdk: authenticated nats command nonce is missing")
	}

	requestCtx, cancel := natsAdapterRequestContext(ctx, a.timeout)
	defer cancel()

	sub, err := a.conn.SubscribeSync(msg.Reply)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()
	if err := a.conn.FlushWithContext(requestCtx); err != nil {
		return err
	}
	if err := a.conn.PublishMsg(msg); err != nil {
		return err
	}
	if err := a.conn.FlushWithContext(requestCtx); err != nil {
		return err
	}
	return a.waitForResponse(requestCtx, requestNonce, subject, sub.NextMsgWithContext)
}

type natsResponseNext func(context.Context) (*nats.Msg, error)

func (a *NATSAdapter) waitForResponse(ctx context.Context, requestNonce, requestSubject string, next natsResponseNext) error {
	for {
		resp, err := next(ctx)
		if err != nil {
			return err
		}
		matched, err := a.handleResponse(resp, requestNonce, requestSubject)
		if !matched {
			continue
		}
		return err
	}
}

func (a *NATSAdapter) handleResponse(resp *nats.Msg, requestNonce, requestSubject string) (bool, error) {
	if resp == nil {
		return false, nil
	}
	if err := controlauth.VerifyNATSMessage(resp, a.verifier); err != nil {
		return false, nil
	}
	if len(resp.Data) == 0 {
		return false, nil
	}
	var out natsCommandResponse
	if err := json.Unmarshal(resp.Data, &out); err != nil {
		return false, nil
	}
	if strings.TrimSpace(out.RequestNonce) != requestNonce || strings.TrimSpace(out.RequestSubject) != requestSubject {
		return false, nil
	}
	if !out.OK {
		if strings.TrimSpace(out.Error) != "" {
			return true, fmt.Errorf("scheduledsdk: %s", out.Error)
		}
		return true, fmt.Errorf("scheduledsdk: nats command failed")
	}
	return true, nil
}

type natsCommandResponse struct {
	OK             bool   `json:"ok"`
	Error          string `json:"error,omitempty"`
	RequestNonce   string `json:"request_nonce"`
	RequestSubject string `json:"request_subject"`
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
	traced := contextx.CtxToTraceNats(ctx, msg.Subject)
	for key, values := range traced.Header {
		// Execution control messages need identity/audit metadata but must never
		// carry a reusable bearer token.
		if strings.EqualFold(key, contextx.TokenHeader) {
			continue
		}
		msg.Header[key] = append([]string(nil), values...)
	}
}
