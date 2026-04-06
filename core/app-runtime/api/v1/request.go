package v1

import (
	"context"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-runtime/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/nats-io/nats.go"
)

// RequestRouter 由 server 实现，用于转发请求到应用、判断/确保版本运行（避免 api 层依赖 server）
type RequestRouter interface {
	ForwardToApp(msg *nats.Msg) error
	IsAppVersionRunning(user, app, version string) bool
	EnsureAppVersionRunning(ctx context.Context, user, app, version string) error
}

// RequestHandler 处理来自 function_server 的请求（转发给应用）
type RequestHandler struct {
	appManageService *service.AppManageService
	router           RequestRouter
}

// NewRequestHandler 创建 Request 处理器（依赖注入）
func NewRequestHandler(appManageService *service.AppManageService, router RequestRouter) *RequestHandler {
	return &RequestHandler{appManageService: appManageService, router: router}
}

// HandleFunctionServerRequest 处理来自 app-server/function_server 的请求，转发给应用
func (h *RequestHandler) HandleFunctionServerRequest(msg *nats.Msg) {
	start := time.Now()
	ctx := context.Background()
	user := msg.Header.Get("user")
	app := msg.Header.Get("app")
	version := msg.Header.Get("version")
	traceId := msg.Header.Get(contextx.TraceIdHeader)
	router := msg.Header.Get("router")
	method := msg.Header.Get("method")

	logger.Infof(ctx, "[HandleFunctionServerRequest] received: traceId=%s, %s/%s/%s, method=%s, router=%s, dataLen=%d",
		traceId, user, app, version, method, router, len(msg.Data))

	if user == "" || app == "" || version == "" {
		logger.Errorf(ctx, "[HandleFunctionServerRequest] Missing headers: user=%s, app=%s, version=%s, traceId=%s", user, app, version, traceId)
		return
	}
	h.appManageService.QPSTracker.RecordRequest(user, app, version)
	isRunning := h.router.IsAppVersionRunning(user, app, version)
	if !isRunning {
		logger.Warnf(ctx, "[HandleFunctionServerRequest] Version %s/%s/%s not running, ensuring... traceId=%s", user, app, version, traceId)
		ensureStart := time.Now()
		if err := h.router.EnsureAppVersionRunning(ctx, user, app, version); err != nil {
			logger.Errorf(ctx, "[HandleFunctionServerRequest] Ensure running failed: traceId=%s, err=%v, ensureElapsed=%s",
				traceId, err, time.Since(ensureStart).Truncate(time.Millisecond))
		} else {
			logger.Infof(ctx, "[HandleFunctionServerRequest] Ensure running done: traceId=%s, ensureElapsed=%s",
				traceId, time.Since(ensureStart).Truncate(time.Millisecond))
		}
	}
	if err := h.router.ForwardToApp(msg); err != nil {
		logger.Errorf(ctx, "[HandleFunctionServerRequest] Forward failed: traceId=%s, err=%v, totalElapsed=%s",
			traceId, err, time.Since(start).Truncate(time.Millisecond))
		return
	}
	logger.Infof(ctx, "[HandleFunctionServerRequest] forwarded: traceId=%s, %s/%s/%s, totalElapsed=%s",
		traceId, user, app, version, time.Since(start).Truncate(time.Millisecond))
}
