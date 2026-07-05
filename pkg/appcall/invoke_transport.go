package appcall

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/appinvoke"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// appInvokeTransport 管理 RequestApp 这条 publish + reply + waiter 链路。
type appInvokeTransport struct {
	connProvider ConnProvider
	waiter       Waiter
	timeout      time.Duration
	subs         []*nats.Subscription
}

func newAppInvokeTransport(connProvider ConnProvider, waiter Waiter, timeout time.Duration) *appInvokeTransport {
	return &appInvokeTransport{
		connProvider: connProvider,
		waiter:       waiter,
		timeout:      timeout,
		subs:         make([]*nats.Subscription, 0),
	}
}

// RequestApp 请求应用（Publish 到 runtime.v1.cmd.app.invoke.{user}.{app}.{version}，通过 waiter 等响应）。
func (c *Client) RequestApp(ctx context.Context, natsID int64, req *dto.RequestAppReq) (*dto.RequestAppResp, error) {
	return c.invoke.requestApp(ctx, natsID, req)
}

func (t *appInvokeTransport) requestApp(ctx context.Context, natsID int64, req *dto.RequestAppReq) (*dto.RequestAppResp, error) {
	start := time.Now()
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	invokeReq := *req
	if invokeReq.ClientSource == "" {
		invokeReq.ClientSource = contextx.GetClientSource(ctx)
	}
	if invokeReq.SourceType == "" {
		invokeReq.SourceType = contextx.GetSourceType(ctx)
	}
	if invokeReq.SourceRef == "" {
		invokeReq.SourceRef = contextx.GetSourceRef(ctx)
	}
	if invokeReq.SourcePath == "" {
		invokeReq.SourcePath = contextx.GetSourcePath(ctx)
	}
	if invokeReq.SourceTitle == "" {
		invokeReq.SourceTitle = contextx.GetSourceTitle(ctx)
	}
	if invokeReq.SourceParentPath == "" {
		invokeReq.SourceParentPath = contextx.GetSourceParentPath(ctx)
	}
	if invokeReq.SourceParentTitle == "" {
		invokeReq.SourceParentTitle = contextx.GetSourceParentTitle(ctx)
	}
	if invokeReq.SourceTemplateType == "" {
		invokeReq.SourceTemplateType = contextx.GetSourceTemplateType(ctx)
	}
	if invokeReq.WorkspaceSessionID == "" {
		invokeReq.WorkspaceSessionID = contextx.GetWorkspaceSessionID(ctx)
	}
	if invokeReq.WorkspaceSessionTitle == "" {
		invokeReq.WorkspaceSessionTitle = contextx.GetWorkspaceSessionTitle(ctx)
	}
	if invokeReq.WorkspaceRole == "" {
		invokeReq.WorkspaceRole = contextx.GetWorkspaceRole(ctx)
	}
	if invokeReq.InitiatorUser == "" {
		invokeReq.InitiatorUser = contextx.GetInitiatorUser(ctx)
	}
	if invokeReq.WorkspaceMessageID == 0 {
		invokeReq.WorkspaceMessageID = parsePositiveInt64(contextx.GetWorkspaceMessageID(ctx))
	}
	if invokeReq.ToolCallID == "" {
		invokeReq.ToolCallID = contextx.GetToolCallID(ctx)
	}
	if invokeReq.ToolName == "" {
		invokeReq.ToolName = contextx.GetToolName(ctx)
	}
	msg, err := appinvoke.BuildRuntimeRequestMsg(&invokeReq)
	if err != nil {
		return nil, err
	}
	subject := msg.Subject
	logger.Debugf(ctx, "[appcall:RequestApp] start: traceId=%s, subject=%s, method=%s, router=%s, user=%s, bodyLen=%d",
		req.TraceId, subject, req.Method, req.Router, req.RequestUser, len(req.Body))

	conn, err := t.connProvider.GetConnByNATSID(natsID)
	if err != nil {
		logger.Errorf(ctx, "[appcall:RequestApp] GetConnByNATSID failed: traceId=%s, natsId=%d, err=%v, elapsed=%s",
			req.TraceId, natsID, err, time.Since(start).Truncate(time.Millisecond))
		return nil, err
	}

	registration := t.waiter.Register(req.TraceId)
	if err := conn.PublishMsg(msg); err != nil {
		registration.Cancel()
		logger.Errorf(ctx, "[appcall:RequestApp] PublishMsg failed: traceId=%s, subject=%s, err=%v, elapsed=%s",
			req.TraceId, subject, err, time.Since(start).Truncate(time.Millisecond))
		return nil, fmt.Errorf("publish request failed: %w", err)
	}

	publishElapsed := time.Since(start)
	logger.Debugf(ctx, "[appcall:RequestApp] published, waiting response: traceId=%s, publishElapsed=%s, waitTimeout=%s",
		req.TraceId, publishElapsed.Truncate(time.Millisecond), t.timeout)

	resp, err := registration.Wait(ctx, t.timeout)
	totalElapsed := time.Since(start)
	if err != nil {
		logger.Errorf(ctx, "[appcall:RequestApp] Wait failed: traceId=%s, err=%v, totalElapsed=%s (publish=%s, wait=%s)",
			req.TraceId, err, totalElapsed.Truncate(time.Millisecond), publishElapsed.Truncate(time.Millisecond),
			(totalElapsed - publishElapsed).Truncate(time.Millisecond))
		return nil, fmt.Errorf("wait response timeout: %w", err)
	}

	logger.Debugf(ctx, "[appcall:RequestApp] done: traceId=%s, hasError=%v, totalElapsed=%s",
		req.TraceId, resp.Error != "", totalElapsed.Truncate(time.Millisecond))
	return resp, nil
}

func (t *appInvokeTransport) initSubscriptions() {
	for _, hostID := range t.connProvider.HostIDs() {
		conn, err := t.connProvider.GetConnByHost(hostID)
		if err != nil {
			continue
		}

		sub, err := conn.Subscribe(subjects.AppServerAppInvokeReplySubjectPattern, t.handleInvokeReply)
		if err != nil {
			logger.Warnf(context.Background(), "[appcall] Failed to subscribe to response subject on host %d: %v", hostID, err)
			continue
		}
		t.subs = append(t.subs, sub)
	}
}

func (t *appInvokeTransport) handleInvokeReply(msg *nats.Msg) {
	var resp dto.RequestAppResp
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return
	}
	if traceID := msg.Header.Get(contextx.TraceIdHeader); traceID != "" {
		resp.TraceId = traceID
	}
	if !t.waiter.Notify(resp.TraceId, &resp) {
		logger.Warnf(context.Background(), "[appcall] No waiting request found for traceId: %s", resp.TraceId)
	}
}

func (t *appInvokeTransport) close() error {
	for _, sub := range t.subs {
		if sub != nil {
			_ = sub.Unsubscribe()
		}
	}
	t.subs = nil
	return nil
}

func parsePositiveInt64(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0
	}
	return value
}
