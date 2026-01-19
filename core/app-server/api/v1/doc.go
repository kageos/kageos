package v1

import (
	"strconv"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
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

	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
		Format  string `json:"format"` // 可选，默认为 markdown
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	doc, err := s.docService.CreateDoc(ctx, treeID, req.Title, req.Content, req.Format)
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

	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		Format  string `json:"format"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	doc, err := s.docService.UpdateDoc(ctx, treeID, req.Title, req.Content, req.Format)
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
