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
// @Router /api/v1/docs/info/{full-code-path} [get]
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
// @Router /api/v1/docs/info/{full-code-path} [put]
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
// @Router /api/v1/docs/info/{full-code-path} [delete]
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

// QueryDocs 统一查询文档接口（支持路径批量查询和关键词搜索）
// @Summary 查询文档（统一接口）
// @Description 支持两种查询模式：1) 路径批量查询：提供 paths 参数；2) 关键词搜索：提供 keyword 参数。可通过 include_content 控制是否返回文档内容。
// @Tags 文档
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param body body dto.QueryDocsReq true "查询请求"
// @Success 200 {object} dto.QueryDocsResp "查询成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/docs/query [post]
func (s *Doc) QueryDocs(c *gin.Context) {
	var req dto.QueryDocsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	// 验证：至少提供一种查询方式
	if len(req.Paths) == 0 && req.Keyword == "" {
		response.FailWithMessage(c, "请提供 paths（路径列表）或 keyword（搜索关键词）")
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := s.docService.QueryDocs(ctx, &req)
	if err != nil {
		logger.Errorf(c, "[Doc API] 查询文档失败: paths=%v, keyword=%s, error=%v", req.Paths, req.Keyword, err)
		response.FailWithMessage(c, "查询文档失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}
