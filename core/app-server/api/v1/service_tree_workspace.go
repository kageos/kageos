package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
)

// GetWorkspaceContext 获取工作台环境信息
// GET /workspace/api/v1/workspace/context?full_code_path=...&file_source=snapshot|runtime
func (s *ServiceTree) GetWorkspaceContext(c *gin.Context) {
	var req dto.GetWorkspaceContextReq
	req.FullCodePath = c.Query("full_code_path")
	req.FileSource = c.Query("file_source")
	if req.FullCodePath == "" {
		response.BadRequest(c, "full_code_path 必填")
		return
	}

	ctx := contextx.ToContext(c)
	if err := requireAccess(c, s.teamAccessService, req.FullCodePath, access.ActionRead); err != nil {
		response.Error(c, err)
		return
	}
	resp, err := s.serviceTreeService.GetWorkspaceContext(ctx, &req)
	if err != nil {
		response.Internal(c, "获取工作台环境信息失败: "+err.Error())
		return
	}
	// 函数/文档会被解析到父目录执行。除资源本身外，还必须拥有父目录读取权限，
	// 避免只授权单个资源时意外读取同目录源码和兄弟节点。
	if directoryPath := access.NormalizeResourcePath(resp.Directory.FullCodePath); directoryPath != access.NormalizeResourcePath(req.FullCodePath) {
		if err := requireAccess(c, s.teamAccessService, directoryPath, access.ActionRead); err != nil {
			response.Error(c, err)
			return
		}
	}

	response.OkWithData(c, resp)
}

// WriteFileContent 工作台写入单个文本文件（实时写盘，不编译）
// POST /workspace/api/v1/workspace/files/write
func (s *ServiceTree) WriteFileContent(c *gin.Context) {
	var req dto.WriteFileContentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if req.FullCodePath == "" || req.FileName == "" {
		response.BadRequest(c, "full_code_path、file_name 必填")
		return
	}
	if err := requireAccess(c, s.teamAccessService, req.FullCodePath, access.ActionAdmin); err != nil {
		response.Error(c, err)
		return
	}
	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.WriteFileContent(ctx, &req)
	if err != nil {
		response.Internal(c, "写入文件失败: "+err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// ReplaceFileContent 工作台文件 search-replace（实时写盘）
// POST /workspace/api/v1/workspace/files/replace
func (s *ServiceTree) ReplaceFileContent(c *gin.Context) {
	var req dto.ReplaceFileContentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if req.FullCodePath == "" || req.FileName == "" || len(req.Replacements) == 0 {
		response.BadRequest(c, "full_code_path、file_name、replacements 必填")
		return
	}
	if err := requireAccess(c, s.teamAccessService, req.FullCodePath, access.ActionAdmin); err != nil {
		response.Error(c, err)
		return
	}
	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.ReplaceFileContent(ctx, &req)
	if err != nil {
		response.Internal(c, "替换文件失败: "+err.Error())
		return
	}
	if !resp.Success {
		response.Internal(c, resp.Message)
		return
	}
	response.OkWithData(c, resp)
}

// DeleteFile 工作台删除文件（删磁盘+删节点）
// POST /workspace/api/v1/workspace/files/delete
func (s *ServiceTree) DeleteFile(c *gin.Context) {
	var req dto.DeleteFileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if req.FullCodePath == "" || req.FileName == "" {
		response.BadRequest(c, "full_code_path、file_name 必填")
		return
	}
	if err := requireAccess(c, s.teamAccessService, req.FullCodePath, access.ActionDelete); err != nil {
		response.Error(c, err)
		return
	}
	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.DeleteFile(ctx, &req)
	if err != nil {
		response.Internal(c, "删除文件失败: "+err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// ReadAppLog 读取应用日志（支持 version、关键词检索）
// POST /workspace/api/v1/workspace/logs/read
func (s *ServiceTree) ReadAppLog(c *gin.Context) {
	var req dto.ReadAppLogReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if req.FullCodePath == "" {
		response.BadRequest(c, "full_code_path 必填")
		return
	}
	if err := requireAccess(c, s.teamAccessService, req.FullCodePath, access.ActionAdmin); err != nil {
		response.Error(c, err)
		return
	}
	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.ReadAppLog(ctx, &req)
	if err != nil {
		response.Internal(c, "读取日志失败: "+err.Error())
		return
	}
	response.OkWithData(c, resp)
}
