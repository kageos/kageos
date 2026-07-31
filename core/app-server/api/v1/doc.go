package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/app-server/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/logger"
)

type Doc struct {
	docService        *service.DocService
	permissionService *service.PermissionService
}

// NewDoc 创建文档处理器（依赖注入）
func NewDoc(docService *service.DocService, permissionService *service.PermissionService) *Doc {
	return &Doc{
		docService:        docService,
		permissionService: permissionService,
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
// @Router /workspace/api/v1/docs/info/{full-code-path} [get]
func (s *Doc) GetDoc(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "路径不能为空")
		return
	}
	if err := requireAccess(c, s.permissionService, fullCodePath, access.ActionRead); err != nil {
		response.FailWithMessage(c, err.Error())
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
// @Router /workspace/api/v1/docs/info/{full-code-path} [put]
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
	if err := requireAccess(c, s.permissionService, fullCodePath, access.ActionUpdate); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

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
// @Router /workspace/api/v1/docs/info/{full-code-path} [delete]
func (s *Doc) DeleteDoc(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "路径不能为空")
		return
	}
	if err := requireAccess(c, s.permissionService, fullCodePath, access.ActionDelete); err != nil {
		response.FailWithMessage(c, err.Error())
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

// SearchDocs 搜索文档（模糊搜索）
// @Summary 搜索文档
// @Description 根据关键词搜索文档，支持跨应用搜索。关键词为空时返回最近创建的文档。可通过 include_content 控制是否返回文档内容。
// @Tags 文档
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param keyword query string false "搜索关键词（可选，为空时返回最近创建的文档）"
// @Param page query int false "页码（默认 1）"
// @Param page_size query int false "每页数量（默认 10，最大 100）"
// @Param include_content query bool false "是否包含文档内容（默认 true）"
// @Success 200 {object} dto.SearchDocsResp "搜索成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/docs/search [get]
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

// BatchGetDocs 批量获取文档（精确查询）
// @Summary 批量获取文档
// @Description 根据路径列表批量获取文档（精确匹配）。可通过 include_content 控制是否返回文档内容。
// @Tags 文档
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param paths query []string true "文档路径列表（必填，支持 paths[]=value1&paths[]=value2）"
// @Param include_content query bool false "是否包含文档内容（默认 true）"
// @Success 200 {object} dto.BatchGetDocsResp "查询成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/docs/batch [get]
func (s *Doc) BatchGetDocs(c *gin.Context) {
	var req dto.BatchGetDocsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	if len(req.Paths) == 0 {
		response.FailWithMessage(c, "paths 参数不能为空")
		return
	}
	for _, path := range req.Paths {
		if err := requireAccess(c, s.permissionService, path, access.ActionRead); err != nil {
			response.FailWithMessage(c, err.Error())
			return
		}
	}

	ctx := contextx.ToContext(c)
	resp, err := s.docService.BatchGetDocs(ctx, &req)
	if err != nil {
		logger.Errorf(c, "[Doc API] 批量获取文档失败: paths=%v, error=%v", req.Paths, err)
		response.FailWithMessage(c, "批量获取文档失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}
