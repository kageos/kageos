package v1

import (
	"github.com/kageos/kageos/core/app-runtime/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/nats-io/nats.go"
)

// AppDatabaseHandler resolves SDK private database capabilities into
// package-scoped low-privilege DSNs.
type AppDatabaseHandler struct {
	appDatabaseService *service.AppDatabaseService
}

func NewAppDatabaseHandler(appDatabaseService *service.AppDatabaseService) *AppDatabaseHandler {
	return &AppDatabaseHandler{appDatabaseService: appDatabaseService}
}

func (h *AppDatabaseHandler) HandleResolve(msg *nats.Msg) {
	ctx := handlerContext(msg)
	req, ok := decodeRequest[dto.AppDBResolveReq](ctx, msg, "HandleAppDBResolve")
	if !ok {
		return
	}
	if h == nil || h.appDatabaseService == nil || !h.appDatabaseService.IsEnabled() {
		respondFailure(ctx, msg, "HandleAppDBResolve", service.ErrAppDatabaseDisabled)
		return
	}

	resp, err := h.appDatabaseService.Resolve(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[HandleAppDBResolve] Failed: user=%s app=%s package=%s access=%s err=%v", req.User, req.App, req.PackagePath, req.Access, err)
		respondFailure(ctx, msg, "HandleAppDBResolve", err)
		return
	}
	if !respondSuccess(ctx, msg, "HandleAppDBResolve", resp) {
		return
	}
	logger.Infof(ctx, "[HandleAppDBResolve] Resolved: user=%s app=%s package=%s access=%s db=%s", req.User, req.App, req.PackagePath, resp.Access, resp.DatabaseName)
}
