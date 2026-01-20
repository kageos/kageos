package v1

import (
	"strconv"

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

// GetDoc 获取文档内容
// @Summary 获取文档内容
// @Description 根据 ServiceTree 节点ID获取文档内容
// @Tags 文档
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param tree_id path int true "ServiceTree 节点ID"
// @Success 200 {object} map[string]interface{} "文档内容"
// @Failure 400 {string} string "请求参数错误"
// @Failure 404 {string} string "文档不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/service_tree/{tree_id}/doc [get]
func (s *Doc) GetDoc(c *gin.Context) {
	treeIDStr := c.Param("tree_id")
	treeID, err := strconv.ParseInt(treeIDStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "无效的节点ID: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	doc, err := s.docService.GetDoc(ctx, treeID)
	if err != nil {
		logger.Errorf(c, "[Doc API] 获取文档失败: treeID=%d, error=%v", treeID, err)
		response.FailWithMessage(c, "获取文档失败: "+err.Error())
		return
	}

	response.OkWithData(c, doc)
}

// CreateDoc 创建文档
// @Summary 创建文档
// @Description 为指定的 ServiceTree 节点创建文档
// @Tags 文档
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param tree_id path int true "ServiceTree 节点ID"
// @Param request body dto.CreateDocReq true "创建文档请求"
// @Success 200 {object} map[string]interface{} "文档信息"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/service_tree/{tree_id}/doc [post]
func (s *Doc) CreateDoc(c *gin.Context) {
	treeIDStr := c.Param("tree_id")
	treeID, err := strconv.ParseInt(treeIDStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "无效的节点ID: "+err.Error())
		return
	}

	var req dto.CreateDocReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	// 从 URL 路径参数填充 TreeID 到 DTO
	req.TreeID = treeID

	ctx := contextx.ToContext(c)
	doc, err := s.docService.CreateDoc(ctx, &req)
	if err != nil {
		logger.Errorf(c, "[Doc API] 创建文档失败: treeID=%d, error=%v", treeID, err)
		response.FailWithMessage(c, "创建文档失败: "+err.Error())
		return
	}

	response.OkWithData(c, doc)
}

// UpdateDoc 更新文档
// @Summary 更新文档
// @Description 更新指定 ServiceTree 节点的文档内容
// @Tags 文档
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param tree_id path int true "ServiceTree 节点ID"
// @Param request body dto.UpdateDocReq true "更新文档请求"
// @Success 200 {object} map[string]interface{} "文档信息"
// @Failure 400 {string} string "请求参数错误"
// @Failure 404 {string} string "文档不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/service_tree/{tree_id}/doc [put]
func (s *Doc) UpdateDoc(c *gin.Context) {
	treeIDStr := c.Param("tree_id")
	treeID, err := strconv.ParseInt(treeIDStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "无效的节点ID: "+err.Error())
		return
	}

	var req dto.UpdateDocReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	// 从 URL 路径参数填充 TreeID 到 DTO
	req.TreeID = treeID

	ctx := contextx.ToContext(c)
	doc, err := s.docService.UpdateDoc(ctx, &req)
	if err != nil {
		logger.Errorf(c, "[Doc API] 更新文档失败: treeID=%d, error=%v", treeID, err)
		response.FailWithMessage(c, "更新文档失败: "+err.Error())
		return
	}

	response.OkWithData(c, doc)
}

// DeleteDoc 删除文档
// @Summary 删除文档
// @Description 删除指定 ServiceTree 节点的文档
// @Tags 文档
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param tree_id path int true "ServiceTree 节点ID"
// @Success 200 {object} map[string]interface{} "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 404 {string} string "文档不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/service_tree/{tree_id}/doc [delete]
func (s *Doc) DeleteDoc(c *gin.Context) {
	treeIDStr := c.Param("tree_id")
	treeID, err := strconv.ParseInt(treeIDStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "无效的节点ID: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	if err := s.docService.DeleteDoc(ctx, treeID); err != nil {
		logger.Errorf(c, "[Doc API] 删除文档失败: treeID=%d, error=%v", treeID, err)
		response.FailWithMessage(c, "删除文档失败: "+err.Error())
		return
	}

	response.OkWithMessage(c, "文档删除成功")
}

// ==================== 基于路径的文档接口（新接口，用于 /doc/*full-code-path） ====================

// GetDocByPath 根据完整路径获取文档
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
// @Router /api/v1/doc/{full-code-path} [get]
func (s *Doc) GetDocByPath(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "路径不能为空")
		return
	}

	ctx := contextx.ToContext(c)
	doc, err := s.docService.GetDocByPath(ctx, fullCodePath)
	if err != nil {
		logger.Errorf(c, "[Doc API] 获取文档失败: fullCodePath=%s, error=%v", fullCodePath, err)
		response.FailWithMessage(c, "获取文档失败: "+err.Error())
		return
	}

	response.OkWithData(c, doc)
}

// UpdateDocByPath 根据完整路径更新文档
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
// @Router /api/v1/doc/{full-code-path} [put]
func (s *Doc) UpdateDocByPath(c *gin.Context) {
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
	doc, err := s.docService.UpdateDocByPath(ctx, &req)
	if err != nil {
		logger.Errorf(c, "[Doc API] 更新文档失败: fullCodePath=%s, error=%v", fullCodePath, err)
		response.FailWithMessage(c, "更新文档失败: "+err.Error())
		return
	}

	response.OkWithData(c, doc)
}

// DeleteDocByPath 根据完整路径删除文档
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
// @Router /api/v1/doc/{full-code-path} [delete]
func (s *Doc) DeleteDocByPath(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "路径不能为空")
		return
	}

	ctx := contextx.ToContext(c)
	if err := s.docService.DeleteDocByPath(ctx, fullCodePath); err != nil {
		logger.Errorf(c, "[Doc API] 删除文档失败: fullCodePath=%s, error=%v", fullCodePath, err)
		response.FailWithMessage(c, "删除文档失败: "+err.Error())
		return
	}

	response.OkWithMessage(c, "文档删除成功")
}
