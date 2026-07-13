package v1

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
)

// GetServiceTreeDetail 获取服务目录详情
// @Summary 获取服务目录详情
// @Description 根据ID或full-code-path获取服务目录详情
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param id query int false "服务目录ID（优先使用）"
// @Param full_code_path query string false "完整代码路径（如果未提供ID则使用）"
// @Success 200 {object} dto.GetServiceTreeDetailResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "服务目录不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/directories [get]
func (s *ServiceTree) GetServiceTreeDetail(c *gin.Context) {
	var req dto.GetServiceTreeDetailReq

	// 从 query 参数获取 ID
	idStr := c.Query("id")
	if idStr != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			response.BadRequest(c, "无效的ID参数")
			return
		}
		req.ID = id
	}

	// 从 query 参数获取 full_code_path
	req.FullCodePath = c.Query("full_code_path")

	if req.ID == 0 && req.FullCodePath == "" {
		response.BadRequest(c, "必须提供 ID 或 full_code_path 参数")
		return
	}
	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.GetServiceTreeDetail(ctx, &req)
	if err != nil {
		response.Internal(c, "获取服务目录详情失败: "+err.Error())
		return
	}
	if err := requireWorkspaceDataAccess(c, s.teamAccessService, resp.FullCodePath, access.ActionRead); err != nil {
		response.Error(c, err)
		return
	}

	response.OkWithData(c, resp)
}

// BatchGetServiceTreeDetails 批量获取服务目录详情
// @Summary 批量获取服务目录详情
// @Description 根据 full_code_path 列表批量获取目录、函数和文档详情
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param request body dto.BatchGetServiceTreeDetailsReq true "批量资源路径"
// @Success 200 {object} dto.BatchGetServiceTreeDetailsResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/directory-queries [post]
func (s *ServiceTree) BatchGetServiceTreeDetails(c *gin.Context) {
	var req dto.BatchGetServiceTreeDetailsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.BatchGetServiceTreeDetails(ctx, &req)
	if err != nil {
		response.Internal(c, "批量获取服务目录详情失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// GetDirectoryOverview 获取目录概览
// @Summary 获取目录概览
// @Description 汇总当前目录及可读子目录/函数的资源统计、函数任务和 Agent 任务配置
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full_code_path query string true "目录完整路径"
// @Success 200 {object} dto.GetDirectoryOverviewResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /workspace/api/v1/directory-overviews [get]
func (s *ServiceTree) GetDirectoryOverview(c *gin.Context) {
	req := dto.GetDirectoryOverviewReq{
		FullCodePath: c.Query("full_code_path"),
	}
	if req.FullCodePath == "" {
		response.BadRequest(c, "必须提供 full_code_path 参数")
		return
	}
	if err := requireWorkspaceDataAccess(c, s.teamAccessService, req.FullCodePath, access.ActionRead); err != nil {
		response.Error(c, err)
		return
	}

	resp, err := s.serviceTreeService.GetDirectoryOverview(contextx.ToContext(c), &req)
	if err != nil {
		response.Internal(c, "获取目录概览失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// SearchFunctions 搜索函数
// @Summary 搜索函数
// @Description 根据关键词、类型等条件搜索函数，支持分页
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param user query string false "用户名（可选，用于过滤应用）"
// @Param app query string false "应用名（可选，用于过滤应用）"
// @Param keyword query string false "搜索关键词（可选，用于搜索名称和路径）"
// @Param full_code_path query string false "完整路径（可选，精确或目录前缀搜索）"
// @Param template_type query string false "模板类型过滤（可选，如：form、table、chart）"
// @Param page query int true "页码" default(1)
// @Param page_size query int true "每页数量" default(10)
// @Success 200 {object} dto.SearchFunctionsResp "搜索成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/function-search-results [get]
func (s *ServiceTree) SearchFunctions(c *gin.Context) {
	var req dto.SearchFunctionsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 验证分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100 // 限制最大每页数量
	}

	ctx := contextx.ToContext(c)
	req.CurrentUser = contextx.GetRequestUser(ctx)
	resp, err := s.serviceTreeService.SearchFunctions(ctx, &req)
	if err != nil {
		response.Internal(c, "搜索函数失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// SearchResources 全站资源搜索
// @Summary 全站资源搜索
// @Description 根据关键词搜索目录、函数和文档，支持分页
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param user query string false "用户名（可选，用于过滤应用）"
// @Param app query string false "应用名（可选，用于过滤应用）"
// @Param keyword query string false "搜索关键词"
// @Param full_code_path query string false "完整路径（可选，精确或目录前缀搜索）"
// @Param resource_type query string false "资源类型（all/package/function/docs）"
// @Param page query int true "页码" default(1)
// @Param page_size query int true "每页数量" default(20)
// @Success 200 {object} dto.SearchResourcesResp "搜索成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/resource-search-results [get]
func (s *ServiceTree) SearchResources(c *gin.Context) {
	var req dto.SearchResourcesReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	ctx := contextx.ToContext(c)
	req.CurrentUser = contextx.GetRequestUser(ctx)
	resp, err := s.serviceTreeService.SearchResources(ctx, &req)
	if err != nil {
		response.Internal(c, "搜索资源失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}
