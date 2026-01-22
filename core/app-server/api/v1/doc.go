package v1

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

type Doc struct {
	docService *service.DocService
}

// NewDoc 创建文档处理器（依赖注入）
func NewDoc(docService *service.DocService) *Doc {
	return &Doc{
		docService: docService,
	}
}

// GetDoc 根据完整路径获取文档
// @Summary 根据完整路径获取文档
// @Description 根据完整路径获取文档内容
// @Tags 文档
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full-code-path path string true "完整路径（如：/user/app/docs/guide）"
// @Success 200 {object} map[string]interface{} "文档内容"
// @Failure 400 {string} string "请求参数错误"
// @Failure 404 {string} string "文档不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/docs/{full-code-path} [get]
func (s *Doc) GetDoc(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "路径不能为空")
		return
	}

	ctx := contextx.ToContext(c)
	doc, err := s.docService.GetDoc(ctx, fullCodePath)
	if err != nil {
		logger.Errorf(c, "[Doc API] 获取文档失败: fullCodePath=%s, error=%v", fullCodePath, err)
		response.FailWithMessage(c, "获取文档失败: "+err.Error())
		return
	}

	response.OkWithData(c, doc)
}

// UpdateDoc 根据完整路径更新文档
// @Summary 根据完整路径更新文档
// @Description 根据完整路径更新文档内容
// @Tags 文档
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full-code-path path string true "完整路径（如：/user/app/docs/guide）"
// @Param request body dto.UpdateDocReq true "更新文档请求"
// @Success 200 {object} map[string]interface{} "文档信息"
// @Failure 400 {string} string "请求参数错误"
// @Failure 404 {string} string "文档不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/docs/{full-code-path} [put]
func (s *Doc) UpdateDoc(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "路径不能为空")
		return
	}

	var req dto.UpdateDocReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	// 从 URL 路径参数填充 FullCodePath 到 DTO
	req.FullCodePath = fullCodePath

	ctx := contextx.ToContext(c)
	doc, err := s.docService.UpdateDoc(ctx, &req)
	if err != nil {
		logger.Errorf(c, "[Doc API] 更新文档失败: fullCodePath=%s, error=%v", fullCodePath, err)
		response.FailWithMessage(c, "更新文档失败: "+err.Error())
		return
	}

	response.OkWithData(c, doc)
}

// DeleteDoc 根据完整路径删除文档
// @Summary 根据完整路径删除文档
// @Description 根据完整路径删除文档（同时删除节点）
// @Tags 文档
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full-code-path path string true "完整路径（如：/user/app/docs/guide）"
// @Success 200 {object} map[string]interface{} "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 404 {string} string "文档不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/docs/{full-code-path} [delete]
func (s *Doc) DeleteDoc(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "路径不能为空")
		return
	}

	ctx := contextx.ToContext(c)
	if err := s.docService.DeleteDoc(ctx, fullCodePath); err != nil {
		logger.Errorf(c, "[Doc API] 删除文档失败: fullCodePath=%s, error=%v", fullCodePath, err)
		response.FailWithMessage(c, "删除文档失败: "+err.Error())
		return
	}

	response.OkWithMessage(c, "文档删除成功")
}

// GetDocsBatch 根据路径列表批量获取文档
// @Summary 根据路径列表批量获取文档
// @Description 根据路径列表批量获取文档内容（POST 请求，避免与 wildcard 路由冲突）
// @Tags 文档
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param body body dto.GetDocsByPathsReq true "文档路径列表"
// @Success 200 {object} dto.GetDocsByPathsResp "文档列表"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/docs/query [post]
func (s *Doc) GetDocsBatch(c *gin.Context) {
	var req dto.GetDocsByPathsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	if len(req.Paths) == 0 {
		response.FailWithMessage(c, "路径列表不能为空")
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := s.docService.GetDocsByPaths(ctx, req.Paths)
	if err != nil {
		logger.Errorf(c, "[Doc API] 批量获取文档失败: paths=%v, error=%v", req.Paths, err)
		response.FailWithMessage(c, "批量获取文档失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// SearchDocs 搜索文档
// @Summary 搜索文档
// @Description 根据关键词搜索文档，支持跨应用搜索
// @Tags 文档
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param keyword query string false "搜索关键词（可选，用于搜索名称和路径）"
// @Param page query int true "页码" default(1)
// @Param page_size query int true "每页数量" default(10)
// @Success 200 {object} dto.SearchDocsResp "搜索成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/docs/search [get]
func (s *Doc) SearchDocs(c *gin.Context) {
	var req dto.SearchDocsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := s.docService.SearchDocs(ctx, &req)
	if err != nil {
		logger.Errorf(c, "[Doc API] 搜索文档失败: keyword=%s, error=%v", req.Keyword, err)
		response.FailWithMessage(c, "搜索文档失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}
