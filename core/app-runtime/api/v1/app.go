package v1

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/core/app-runtime/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/msgx"
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
	ctx := context.Background()
	msgInfo, err := msgx.DecodeNatsMsg[dto.CreateAppReq](msg)
	if err != nil {
		logger.Errorf(ctx, "[HandleAppCreate] Failed to decode message: %v", err)
		msgx.RespFailMsg(msg, err)
		return
	}
	tenantUser := msgInfo.Data.User
	logger.Infof(ctx, "[HandleAppCreate] Received app create: tenantUser=%s, app=%s, reply=%s",
		tenantUser, msgInfo.Data.Code, msg.Reply)
	appDir, err := h.appManageService.CreateApp(ctx, tenantUser, msgInfo.Data.Code)
	if err != nil {
		logger.Errorf(ctx, "[HandleAppCreate] Failed to create app: %v", err)
		msgx.RespFailMsg(msg, err)
		return
	}
	resp := dto.CreateAppResp{User: tenantUser, App: msgInfo.Data.Code, AppDir: appDir}
	msgx.RespSuccessMsg(msg, resp)
	logger.Infof(ctx, "[HandleAppCreate] App created successfully: %s", appDir)
}

// HandleAppUpdate 处理应用更新请求
func (h *AppHandler) HandleAppUpdate(msg *nats.Msg) {
	ctx := contextx.NatsTraceContext(msg)
	msgInfo, err := msgx.DecodeNatsMsg[dto.UpdateAppReq](msg)
	if err != nil {
		logger.Errorf(ctx, "[HandleAppUpdate] Failed to decode message: %v", err)
		msgx.RespFailMsg(msg, err)
		return
	}
	tenantUser := msgInfo.Data.User
	result, err := h.appManageService.UpdateApp(ctx, tenantUser, msgInfo.Data.App, msgInfo.Data.CreateFunctions, msgInfo.Data.Requirement, msgInfo.Data.ChangeDescription, msgInfo.Data.SkipBuild)
	if err != nil {
		logger.Errorf(ctx, "[HandleAppUpdate] Failed to update app: %v", err)
		msgx.RespFailMsg(msg, err)
		return
	}
	msgx.RespSuccessMsg(msg, result)
	logger.Infof(ctx, "[HandleAppUpdate] App updated: user=%s, app=%s, newVersion=%s", result.User, result.App, result.NewVersion)
}

// HandleAppDelete 处理应用删除请求
func (h *AppHandler) HandleAppDelete(msg *nats.Msg) {
	ctx := context.Background()
	msgInfo, err := msgx.DecodeNatsMsg[dto.DeleteAppReq](msg)
	if err != nil {
		logger.Errorf(ctx, "[HandleAppDelete] Failed to decode message: %v", err)
		msgx.RespFailMsg(msg, err)
		return
	}
	tenantUser := msgInfo.Data.User
	logger.Infof(ctx, "[HandleAppDelete] Received app delete: tenantUser=%s, app=%s", tenantUser, msgInfo.Data.App)
	err = h.appManageService.DeleteApp(ctx, tenantUser, msgInfo.Data.App)
	if err != nil {
		logger.Errorf(ctx, "[HandleAppDelete] Failed to delete app: %v", err)
		msgx.RespFailMsg(msg, err)
		return
	}
	resp := dto.DeleteAppResp{User: tenantUser, App: msgInfo.Data.App}
	msgx.RespSuccessMsg(msg, resp)
	logger.Infof(ctx, "[HandleAppDelete] App deleted: %s/%s", tenantUser, msgInfo.Data.App)
}
