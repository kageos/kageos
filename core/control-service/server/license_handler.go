package server

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/core/control-service/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/msgx"
	"github.com/nats-io/nats.go"
)

// LicenseCommandHandler 处理 control-service 的 NATS 命令/查询。
type LicenseCommandHandler struct {
	licenseService *service.LicenseService
}

// NewLicenseCommandHandler 创建 LicenseCommandHandler。
func NewLicenseCommandHandler(licenseService *service.LicenseService) *LicenseCommandHandler {
	return &LicenseCommandHandler{licenseService: licenseService}
}

// HandleLicenseKeyGetQuery 处理 license key 查询。
func (h *LicenseCommandHandler) HandleLicenseKeyGetQuery(msg *nats.Msg) {
	ctx := context.Background()
	logger.Infof(ctx, "[Control Service] Received license key request")

	keyMsg, err := h.licenseService.BuildKeyMessage(ctx)
	if err != nil {
		logger.Errorf(ctx, "[Control Service] Failed to build key message: %v", err)
		_ = msgx.RespFailMsg(msg, err)
		return
	}

	if err := msgx.RespSuccessMsg(msg, keyMsg); err != nil {
		logger.Errorf(ctx, "[Control Service] Failed to send key message response: %v", err)
		return
	}

	if keyMsg.EncryptedLicense == "" {
		logger.Infof(ctx, "[Control Service] Sent license key response (community edition, no license)")
	} else {
		logger.Infof(ctx, "[Control Service] Sent license key response (license size: %d bytes)", len(keyMsg.EncryptedLicense))
	}
}
