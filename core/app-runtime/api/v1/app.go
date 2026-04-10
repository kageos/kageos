package v1

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-runtime/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/nats-io/nats.go"
)

// AppHandler 处理 app 相关的 NATS 请求（创建、更新、删除）
type AppHandler struct {
	appManageService *service.AppManageService
}

// NewAppHandler 创建 App 处理器（依赖注入）
func NewAppHandler(appManageService *service.AppManageService) *AppHandler {
	return &AppHandler{appManageService: appManageService}
}

// HandleAppCreate 处理应用创建请求
func (h *AppHandler) HandleAppCreate(msg *nats.Msg) {
	ctx := handlerContext(msg)
	req, ok := decodeRequest[dto.CreateAppReq](ctx, msg, "HandleAppCreate")
	if !ok {
		return
	}
	tenantUser := req.User
	logger.Infof(ctx, "[HandleAppCreate] Received app create: tenantUser=%s, app=%s, reply=%s",
		tenantUser, req.Code, msg.Reply)
	appDir, err := h.appManageService.CreateApp(ctx, tenantUser, req.Code)
	if err != nil {
		logger.Errorf(ctx, "[HandleAppCreate] Failed to create app: %v", err)
		respondFailure(ctx, msg, "HandleAppCreate", err)
		return
	}
	resp := dto.CreateAppResp{User: tenantUser, App: req.Code, AppDir: appDir}
	if !respondSuccess(ctx, msg, "HandleAppCreate", resp) {
		return
	}
	logger.Infof(ctx, "[HandleAppCreate] App created successfully: %s", appDir)
}

// HandleAppUpdate 处理应用更新请求
func (h *AppHandler) HandleAppUpdate(msg *nats.Msg) {
	ctx := handlerContext(msg)
	req, ok := decodeRequest[dto.UpdateAppReq](ctx, msg, "HandleAppUpdate")
	if !ok {
		return
	}
	tenantUser := req.User
	result, err := h.appManageService.UpdateApp(ctx, tenantUser, req.App, req.RequestedSourceFiles(), req.Requirement, req.ChangeDescription, req.ShouldWriteOnly())
	if err != nil {
		logger.Errorf(ctx, "[HandleAppUpdate] Failed to update app: %v", err)
		respondFailure(ctx, msg, "HandleAppUpdate", err)
		return
	}
	if !respondSuccess(ctx, msg, "HandleAppUpdate", result) {
		return
	}
	logger.Infof(ctx, "[HandleAppUpdate] App updated: user=%s, app=%s, newVersion=%s", result.User, result.App, result.NewVersion)
}

// HandleAppDelete 处理应用删除请求
func (h *AppHandler) HandleAppDelete(msg *nats.Msg) {
	ctx := handlerContext(msg)
	req, ok := decodeRequest[dto.DeleteAppReq](ctx, msg, "HandleAppDelete")
	if !ok {
		return
	}
	tenantUser := req.User
	logger.Infof(ctx, "[HandleAppDelete] Received app delete: tenantUser=%s, app=%s", tenantUser, req.App)
	err := h.appManageService.DeleteApp(ctx, tenantUser, req.App)
	if err != nil {
		logger.Errorf(ctx, "[HandleAppDelete] Failed to delete app: %v", err)
		respondFailure(ctx, msg, "HandleAppDelete", err)
		return
	}
	resp := dto.DeleteAppResp{User: tenantUser, App: req.App}
	if !respondSuccess(ctx, msg, "HandleAppDelete", resp) {
		return
	}
	logger.Infof(ctx, "[HandleAppDelete] App deleted: %s/%s", tenantUser, req.App)
}
