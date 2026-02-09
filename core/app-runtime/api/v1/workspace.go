package v1

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/core/app-runtime/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/msgx"
	"github.com/nats-io/nats.go"
)

// WorkspaceHandler 处理工作区文件/目录相关的 NATS 请求（读目录、replace、删文件）
type WorkspaceHandler struct {
	appManageService *service.AppManageService
}

// NewWorkspaceHandler 创建 Workspace 处理器（依赖注入）
func NewWorkspaceHandler(appManageService *service.AppManageService) *WorkspaceHandler {
	return &WorkspaceHandler{appManageService: appManageService}
}

// HandleReadDirectoryFiles 处理读取目录文件请求
func (h *WorkspaceHandler) HandleReadDirectoryFiles(msg *nats.Msg) {
	ctx := context.Background()
	msgInfo, err := msgx.DecodeNatsMsg[dto.ReadDirectoryFilesRuntimeReq](msg)
	if err != nil {
		logger.Errorf(ctx, "[HandleReadDirectoryFiles] Failed to decode: %v", err)
		msgx.RespFailMsg(msg, err)
		return
	}
	req := &msgInfo.Data
	logger.Infof(ctx, "[HandleReadDirectoryFiles] user=%s, app=%s, path=%s", req.User, req.App, req.DirectoryPath)
	files, err := h.appManageService.ReadDirectoryFiles(ctx, req.User, req.App, req.DirectoryPath)
	if err != nil {
		logger.Errorf(ctx, "[HandleReadDirectoryFiles] Failed: %v", err)
		msgx.RespFailMsg(msg, err)
		return
	}
	resp := dto.ReadDirectoryFilesRuntimeResp{Success: true, Message: "读取成功", Files: files}
	msgx.RespSuccessMsg(msg, resp)
	logger.Infof(ctx, "[HandleReadDirectoryFiles] fileCount=%d", len(files))
}

// HandleReplaceInFileBatch 处理文件批量 search-replace 请求
func (h *WorkspaceHandler) HandleReplaceInFileBatch(msg *nats.Msg) {
	ctx := context.Background()
	msgInfo, err := msgx.DecodeNatsMsg[dto.ReplaceInFileBatchReq](msg)
	if err != nil {
		logger.Errorf(ctx, "[HandleReplaceInFileBatch] Failed to decode: %v", err)
		msgx.RespFailMsg(msg, err)
		return
	}
	req := &msgInfo.Data
	allOrNothing := req.AllOrNothing
	totalCount, fullContent, details, err := h.appManageService.ReplaceInFileBatch(ctx, req.User, req.App, req.DirectoryPath, req.FileName, req.Replacements, allOrNothing, req.ReturnFullContent)
	if err != nil {
		resp := dto.ReplaceInFileBatchResp{Success: false, Message: err.Error(), Details: details}
		msgx.RespSuccessMsg(msg, resp)
		return
	}
	resp := dto.ReplaceInFileBatchResp{Success: true, Message: "替换成功", ReplaceCount: totalCount}
	if req.ReturnFullContent && fullContent != "" {
		resp.FullContent = fullContent
	}
	msgx.RespSuccessMsg(msg, resp)
	logger.Infof(ctx, "[HandleReplaceInFileBatch] path=%s, file=%s, count=%d", req.DirectoryPath, req.FileName, totalCount)
}

// HandleDeleteFile 处理删除磁盘文件请求
func (h *WorkspaceHandler) HandleDeleteFile(msg *nats.Msg) {
	ctx := context.Background()
	msgInfo, err := msgx.DecodeNatsMsg[dto.DeleteFileRuntimeReq](msg)
	if err != nil {
		logger.Errorf(ctx, "[HandleDeleteFile] Failed to decode: %v", err)
		msgx.RespFailMsg(msg, err)
		return
	}
	req := &msgInfo.Data
	err = h.appManageService.DeleteFile(ctx, req.User, req.App, req.DirectoryPath, req.FileName)
	if err != nil {
		logger.Errorf(ctx, "[HandleDeleteFile] Failed: %v", err)
		msgx.RespFailMsg(msg, err)
		return
	}
	resp := dto.DeleteFileRuntimeResp{Success: true, Message: "已删除"}
	msgx.RespSuccessMsg(msg, resp)
	logger.Infof(ctx, "[HandleDeleteFile] path=%s, file=%s", req.DirectoryPath, req.FileName)
}
