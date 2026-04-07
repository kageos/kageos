package v1

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-runtime/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
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
	ctx := handlerContext(msg)
	req, ok := decodeRequest[dto.CreateServiceTreeRuntimeReq](ctx, msg, "HandleServiceTreeCreate")
	if !ok {
		return
	}
	logger.Infof(ctx, "[HandleServiceTreeCreate] Received: user=%s, app=%s, serviceTree=%s",
		req.User, req.App, req.ServiceTree.Code)
	resp, err := h.serviceTreeService.CreateServiceTree(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[HandleServiceTreeCreate] Failed: %v", err)
		respondFailure(ctx, msg, "HandleServiceTreeCreate", err)
		return
	}
	if !respondSuccess(ctx, msg, "HandleServiceTreeCreate", resp) {
		return
	}
	logger.Infof(ctx, "[HandleServiceTreeCreate] Created: %s", resp.ServiceTree)
}

// HandleBatchCreateDirectoryTree 处理批量创建目录树请求
func (h *ServiceTreeHandler) HandleBatchCreateDirectoryTree(msg *nats.Msg) {
	ctx := handlerContext(msg)
	req, ok := decodeRequest[dto.BatchCreateDirectoryTreeRuntimeReq](ctx, msg, "HandleBatchCreateDirectoryTree")
	if !ok {
		return
	}
	tenantUser := req.User
	logger.Infof(ctx, "[HandleBatchCreateDirectoryTree] Received: user=%s, app=%s, itemCount=%d",
		tenantUser, req.App, len(req.Items))
	resp, err := h.serviceTreeService.BatchCreateDirectoryTree(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[HandleBatchCreateDirectoryTree] Failed: %v", err)
		respondFailure(ctx, msg, "HandleBatchCreateDirectoryTree", err)
		return
	}
	if !respondSuccess(ctx, msg, "HandleBatchCreateDirectoryTree", resp) {
		return
	}
	logger.Infof(ctx, "[HandleBatchCreateDirectoryTree] Done: dirs=%d, files=%d", resp.DirectoryCount, resp.FileCount)
}

// HandleBatchWriteFiles 处理批量写文件请求
func (h *ServiceTreeHandler) HandleBatchWriteFiles(msg *nats.Msg) {
	ctx := handlerContext(msg)
	req, ok := decodeRequest[dto.BatchWriteFilesRuntimeReq](ctx, msg, "HandleBatchWriteFiles")
	if !ok {
		return
	}
	logger.Infof(ctx, "[HandleBatchWriteFiles] Received: user=%s, app=%s, fileCount=%d",
		req.User, req.App, len(req.Files))
	resp, err := h.serviceTreeService.BatchWriteFiles(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[HandleBatchWriteFiles] Failed: %v", err)
		respondFailure(ctx, msg, "HandleBatchWriteFiles", err)
		return
	}
	if !respondSuccess(ctx, msg, "HandleBatchWriteFiles", resp) {
		return
	}
	logger.Infof(ctx, "[HandleBatchWriteFiles] Done: fileCount=%d, newVersion=%s", resp.FileCount, resp.NewVersion)
}

// HandleServiceTreeDelete 处理删除服务目录请求（删磁盘目录并从 main.go 移除 import）
func (h *ServiceTreeHandler) HandleServiceTreeDelete(msg *nats.Msg) {
	ctx := handlerContext(msg)
	req, ok := decodeRequest[dto.DeleteServiceTreeRuntimeReq](ctx, msg, "HandleServiceTreeDelete")
	if !ok {
		return
	}
	logger.Infof(ctx, "[HandleServiceTreeDelete] Received: user=%s, app=%s, packagePath=%s",
		req.User, req.App, req.PackagePath)
	resp, err := h.serviceTreeService.DeleteServiceTreeByReq(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[HandleServiceTreeDelete] Failed: %v", err)
		respondFailure(ctx, msg, "HandleServiceTreeDelete", err)
		return
	}
	if !respondSuccess(ctx, msg, "HandleServiceTreeDelete", resp) {
		return
	}
	if resp.Success {
		logger.Infof(ctx, "[HandleServiceTreeDelete] Deleted: %s", req.PackagePath)
	} else {
		logger.Warnf(ctx, "[HandleServiceTreeDelete] Failed: %s", resp.Error)
	}
}
