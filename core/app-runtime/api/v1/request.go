package v1

import (
	"context"
	"time"

	"github.com/kageos/kageos/core/app-runtime/service"
	"github.com/kageos/kageos/pkg/appinvoke"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/nats-io/nats.go"
)

// InvokeTransport 由 server 层实现，用于 invoke 请求的转发与运行兜底。
type InvokeTransport interface {
	ForwardToApp(msg *nats.Msg) error
	IsAppVersionRunning(user, app, version string) bool
	EnsureAppVersionRunning(ctx context.Context, user, app, version string) error
}

// RequestHandler 处理来自 app-server 的 invoke 请求（转发给应用）。
type RequestHandler struct {
	appManageService *service.AppManageService
	transport        InvokeTransport
}

// NewRequestHandler 创建 Request 处理器（依赖注入）
func NewRequestHandler(appManageService *service.AppManageService, transport InvokeTransport) *RequestHandler {
	return &RequestHandler{appManageService: appManageService, transport: transport}
}

// HandleAppServerInvokeRequest 处理来自 app-server 的调用请求，转发给应用。
func (h *RequestHandler) HandleAppServerInvokeRequest(msg *nats.Msg) {
	start := time.Now()
	ctx := handlerContext(msg)
	req, err := appinvoke.ParseRuntimeRequest(msg)
	if err != nil {
		logger.Errorf(ctx, "[HandleAppServerInvokeRequest] Invalid invoke request: %v", err)
		return
	}

	logger.Debugf(ctx, "[HandleAppServerInvokeRequest] received: traceId=%s, %s/%s/%s, method=%s, router=%s, dataLen=%d",
		req.TraceID, req.User, req.App, req.Version, req.Method, req.Router, len(msg.Data))

	h.appManageService.QPSTracker.RecordRequest(req.User, req.App, req.Version)
	isRunning := h.transport.IsAppVersionRunning(req.User, req.App, req.Version)
	if !isRunning {
		logger.Warnf(ctx, "[HandleAppServerInvokeRequest] Version %s/%s/%s not running, ensuring... traceId=%s", req.User, req.App, req.Version, req.TraceID)
		ensureStart := time.Now()
		if err := h.transport.EnsureAppVersionRunning(ctx, req.User, req.App, req.Version); err != nil {
			logger.Errorf(ctx, "[HandleAppServerInvokeRequest] Ensure running failed: traceId=%s, err=%v, ensureElapsed=%s",
				req.TraceID, err, time.Since(ensureStart).Truncate(time.Millisecond))
		} else {
			logger.Debugf(ctx, "[HandleAppServerInvokeRequest] Ensure running done: traceId=%s, ensureElapsed=%s",
				req.TraceID, time.Since(ensureStart).Truncate(time.Millisecond))
		}
	}
	if err := h.transport.ForwardToApp(msg); err != nil {
		logger.Errorf(ctx, "[HandleAppServerInvokeRequest] Forward failed: traceId=%s, err=%v, totalElapsed=%s",
			req.TraceID, err, time.Since(start).Truncate(time.Millisecond))
		return
	}
	logger.Debugf(ctx, "[HandleAppServerInvokeRequest] forwarded: traceId=%s, %s/%s/%s, totalElapsed=%s",
		req.TraceID, req.User, req.App, req.Version, time.Since(start).Truncate(time.Millisecond))
}
