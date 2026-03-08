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

// ServiceTreeHandler 处理服务目录相关的 NATS 请求
type ServiceTreeHandler struct {
	serviceTreeService *service.ServiceTreeService
}

// NewServiceTreeHandler 创建 ServiceTree 处理器（依赖注入）
func NewServiceTreeHandler(serviceTreeService *service.ServiceTreeService) *ServiceTreeHandler {
	return &ServiceTreeHandler{serviceTreeService: serviceTreeService}
}

// HandleServiceTreeCreate 处理服务目录创建请求
func (h *ServiceTreeHandler) HandleServiceTreeCreate(msg *nats.Msg) {
	ctx := context.Background()
	msgInfo, err := msgx.DecodeNatsMsg[dto.CreateServiceTreeRuntimeReq](msg)
	if err != nil {
		logger.Errorf(ctx, "[HandleServiceTreeCreate] Failed to decode message: %v", err)
		msgx.RespFailMsg(msg, err)
		return
	}
	logger.Infof(ctx, "[HandleServiceTreeCreate] Received: user=%s, app=%s, serviceTree=%s",
		msgInfo.Data.User, msgInfo.Data.App, msgInfo.Data.ServiceTree.Code)
	resp, err := h.serviceTreeService.CreateServiceTree(ctx, &msgInfo.Data)
	if err != nil {
		logger.Errorf(ctx, "[HandleServiceTreeCreate] Failed: %v", err)
		msgx.RespFailMsg(msg, err)
		return
	}
	msgx.RespSuccessMsg(msg, resp)
	logger.Infof(ctx, "[HandleServiceTreeCreate] Created: %s", resp.ServiceTree)
}

// HandleBatchCreateDirectoryTree 处理批量创建目录树请求
func (h *ServiceTreeHandler) HandleBatchCreateDirectoryTree(msg *nats.Msg) {
	ctx := context.Background()
	msgInfo, err := msgx.DecodeNatsMsg[dto.BatchCreateDirectoryTreeRuntimeReq](msg)
	if err != nil {
		logger.Errorf(ctx, "[HandleBatchCreateDirectoryTree] Failed to decode: %v", err)
		msgx.RespFailMsg(msg, err)
		return
	}
	tenantUser := msgInfo.Data.User
	logger.Infof(ctx, "[HandleBatchCreateDirectoryTree] Received: user=%s, app=%s, itemCount=%d",
		tenantUser, msgInfo.Data.App, len(msgInfo.Data.Items))
	resp, err := h.serviceTreeService.BatchCreateDirectoryTree(ctx, &msgInfo.Data)
	if err != nil {
		logger.Errorf(ctx, "[HandleBatchCreateDirectoryTree] Failed: %v", err)
		msgx.RespFailMsg(msg, err)
		return
	}
	msgx.RespSuccessMsg(msg, resp)
	logger.Infof(ctx, "[HandleBatchCreateDirectoryTree] Done: dirs=%d, files=%d", resp.DirectoryCount, resp.FileCount)
}

// HandleBatchWriteFiles 处理批量写文件请求
func (h *ServiceTreeHandler) HandleBatchWriteFiles(msg *nats.Msg) {
	ctx := contextx.NatsTraceContext(msg)
	msgInfo, err := msgx.DecodeNatsMsg[dto.BatchWriteFilesRuntimeReq](msg)
	if err != nil {
		logger.Errorf(ctx, "[HandleBatchWriteFiles] Failed to decode: %v", err)
		msgx.RespFailMsg(msg, err)
		return
	}
	logger.Infof(ctx, "[HandleBatchWriteFiles] Received: user=%s, app=%s, fileCount=%d",
		msgInfo.Data.User, msgInfo.Data.App, len(msgInfo.Data.Files))
	resp, err := h.serviceTreeService.BatchWriteFiles(ctx, &msgInfo.Data)
	if err != nil {
		logger.Errorf(ctx, "[HandleBatchWriteFiles] Failed: %v", err)
		msgx.RespFailMsg(msg, err)
		return
	}
	msgx.RespSuccessMsg(msg, resp)
	logger.Infof(ctx, "[HandleBatchWriteFiles] Done: fileCount=%d, newVersion=%s", resp.FileCount, resp.NewVersion)
}

// HandleServiceTreeDelete 处理删除服务目录请求（删磁盘目录并从 main.go 移除 import）
func (h *ServiceTreeHandler) HandleServiceTreeDelete(msg *nats.Msg) {
	ctx := context.Background()
	msgInfo, err := msgx.DecodeNatsMsg[dto.DeleteServiceTreeRuntimeReq](msg)
	if err != nil {
		logger.Errorf(ctx, "[HandleServiceTreeDelete] Failed to decode: %v", err)
		msgx.RespFailMsg(msg, err)
		return
	}
	logger.Infof(ctx, "[HandleServiceTreeDelete] Received: user=%s, app=%s, packagePath=%s",
		msgInfo.Data.User, msgInfo.Data.App, msgInfo.Data.PackagePath)
	resp, err := h.serviceTreeService.DeleteServiceTreeByReq(ctx, &msgInfo.Data)
	if err != nil {
		logger.Errorf(ctx, "[HandleServiceTreeDelete] Failed: %v", err)
		msgx.RespFailMsg(msg, err)
		return
	}
	msgx.RespSuccessMsg(msg, resp)
	if resp.Success {
		logger.Infof(ctx, "[HandleServiceTreeDelete] Deleted: %s", msgInfo.Data.PackagePath)
	} else {
		logger.Warnf(ctx, "[HandleServiceTreeDelete] Failed: %s", resp.Error)
	}
}
