package v1

import (
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// QuickLink 快链相关API
type QuickLink struct {
	quickLinkService *service.QuickLinkService
}

// NewQuickLink 创建快链API（依赖注入）
func NewQuickLink(quickLinkService *service.QuickLinkService) *QuickLink {
	return &QuickLink{
		quickLinkService: quickLinkService,
	}
}

// CreateQuickLink 创建快链
// @Summary 创建快链
// @Description 创建快链，保存表单状态、表格状态、图表筛选条件等
// @Tags 快链管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param request body dto.CreateQuickLinkReq true "创建快链请求"
// @Success 200 {object} dto.CreateQuickLinkResp "创建成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未认证"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/quicklink/create [post]
func (q *QuickLink) CreateQuickLink(c *gin.Context) {
	var req dto.CreateQuickLinkReq
	var resp *dto.CreateQuickLinkResp
	var err error
	defer func() {
		logger.Infof(c, "CreateQuickLink req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 绑定请求参数
	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 从context获取username
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.FailWithMessage(c, "未提供用户信息")
		return
	}

	// 验证模板类型
	if req.TemplateType != "form" && req.TemplateType != "table" && req.TemplateType != "chart" {
		response.FailWithMessage(c, "模板类型必须是 form、table 或 chart")
		return
	}

	// 创建快链
	quickLink, err := q.quickLinkService.CreateQuickLink(
		username,
		req.Name,
		req.FunctionRouter,
		req.FunctionMethod,
		req.TemplateType,
		req.RequestParams,
		req.ResponseParams,
		convertFieldMetadata(req.FieldMetadata),
		req.Metadata,
	)
	if err != nil {
		response.FailWithMessage(c, "创建快链失败: "+err.Error())
		return
	}

	// 生成快链URL（前端会自己构建完整URL）
	quickLinkURL := fmt.Sprintf("?_quicklink_id=%d", quickLink.ID)

	resp = &dto.CreateQuickLinkResp{
		ID:   quickLink.ID,
		Name: quickLink.Name,
		URL:  quickLinkURL,
	}
	response.OkWithData(c, resp)
}

// GetQuickLink 获取快链（公开访问，不验证用户）
// @Summary 获取快链
// @Description 根据快链ID获取快链数据（公开访问，用于分享链接）
// @Tags 快链管理
// @Accept json
// @Produce json
// @Param id query int true "快链ID"
// @Success 200 {object} dto.GetQuickLinkResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 404 {string} string "快链不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/quicklink/get [get]
func (q *QuickLink) GetQuickLink(c *gin.Context) {
	var resp *dto.GetQuickLinkResp
	var err error
	defer func() {
		logger.Infof(c, "GetQuickLink resp:%+v err:%v", resp, err)
	}()

	// 获取快链ID
	id := c.Query("id")
	if id == "" {
		response.FailWithMessage(c, "快链ID不能为空")
		return
	}

	var quickLinkID int64
	if _, err = fmt.Sscanf(id, "%d", &quickLinkID); err != nil {
		response.FailWithMessage(c, "快链ID格式错误")
		return
	}

	// 获取快链（不验证用户，允许公开访问）
	quickLink, err := q.quickLinkService.GetQuickLink(quickLinkID)
	if err != nil {
		response.FailWithMessage(c, "快链不存在: "+err.Error())
		return
	}

	// 转换为DTO
	resp = convertQuickLinkToDTO(quickLink)
	response.OkWithData(c, resp)
}

// ListQuickLinks 获取快链列表
// @Summary 获取快链列表
// @Description 获取当前用户的快链列表
// @Tags 快链管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param function_router query string false "函数路由（可选，用于过滤）"
// @Param page query int false "页码，默认1"
// @Param page_size query int false "每页数量，默认10"
// @Success 200 {object} dto.ListQuickLinksResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未认证"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/quicklink/list [get]
func (q *QuickLink) ListQuickLinks(c *gin.Context) {
	var req dto.ListQuickLinksReq
	var resp *dto.ListQuickLinksResp
	var err error
	defer func() {
		logger.Infof(c, "ListQuickLinks req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 绑定查询参数
	if err = c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 从context获取username
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.FailWithMessage(c, "未提供用户信息")
		return
	}

	// 设置默认值
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 10
	}

	// 获取快链列表
	quickLinks, total, err := q.quickLinkService.ListQuickLinks(username, req.FunctionRouter, req.Page, req.PageSize)
	if err != nil {
		response.FailWithMessage(c, "获取快链列表失败: "+err.Error())
		return
	}

	// 转换为DTO
	items := make([]dto.QuickLinkItem, 0, len(quickLinks))
	for _, quickLink := range quickLinks {
		items = append(items, dto.QuickLinkItem{
			ID:             quickLink.ID,
			Name:           quickLink.Name,
			FunctionRouter: quickLink.FunctionRouter,
			FunctionMethod: quickLink.FunctionMethod,
			TemplateType:   quickLink.TemplateType,
			CreatedAt:      time.Time(quickLink.CreatedAt).Format(time.RFC3339),
			UpdatedAt:      time.Time(quickLink.UpdatedAt).Format(time.RFC3339),
		})
	}

	resp = &dto.ListQuickLinksResp{
		List:  items,
		Total: total,
	}
	response.OkWithData(c, resp)
}

// UpdateQuickLink 更新快链
// @Summary 更新快链
// @Description 更新快链的名称、参数等
// @Tags 快链管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param id path int true "快链ID"
// @Param request body dto.UpdateQuickLinkReq true "更新快链请求"
// @Success 200 {object} dto.UpdateQuickLinkResp "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未认证"
// @Failure 404 {string} string "快链不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/quicklink/:id [put]
func (q *QuickLink) UpdateQuickLink(c *gin.Context) {
	var req dto.UpdateQuickLinkReq
	var resp *dto.UpdateQuickLinkResp
	var err error
	defer func() {
		logger.Infof(c, "UpdateQuickLink req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 获取快链ID
	id := c.Param("id")
	var quickLinkID int64
	if _, err = fmt.Sscanf(id, "%d", &quickLinkID); err != nil {
		response.FailWithMessage(c, "快链ID格式错误")
		return
	}

	// 绑定请求参数
	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 从context获取username
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.FailWithMessage(c, "未提供用户信息")
		return
	}

	// 更新快链
	quickLink, err := q.quickLinkService.UpdateQuickLink(
		quickLinkID,
		username,
		req.Name,
		req.RequestParams,
		req.ResponseParams,
		convertFieldMetadata(req.FieldMetadata),
		req.Metadata,
	)
	if err != nil {
		response.FailWithMessage(c, "更新快链失败: "+err.Error())
		return
	}

	resp = &dto.UpdateQuickLinkResp{
		ID:   quickLink.ID,
		Name: quickLink.Name,
	}
	response.OkWithData(c, resp)
}

// DeleteQuickLink 删除快链
// @Summary 删除快链
// @Description 删除指定的快链
// @Tags 快链管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param id path int true "快链ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未认证"
// @Failure 404 {string} string "快链不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/quicklink/:id [delete]
func (q *QuickLink) DeleteQuickLink(c *gin.Context) {
	var err error
	defer func() {
		logger.Infof(c, "DeleteQuickLink err:%v", err)
	}()

	// 获取快链ID
	id := c.Param("id")
	var quickLinkID int64
	if _, err = fmt.Sscanf(id, "%d", &quickLinkID); err != nil {
		response.FailWithMessage(c, "快链ID格式错误")
		return
	}

	// 从context获取username
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.FailWithMessage(c, "未提供用户信息")
		return
	}

	// 删除快链
	if err = q.quickLinkService.DeleteQuickLink(quickLinkID, username); err != nil {
		response.FailWithMessage(c, "删除快链失败: "+err.Error())
		return
	}

	response.OkWithMessage(c, "删除成功")
}

// convertQuickLinkToDTO 将model.QuickLink转换为dto.GetQuickLinkResp
func convertQuickLinkToDTO(quickLink *model.QuickLink) *dto.GetQuickLinkResp {
	// 解析请求参数
	requestParams, _ := quickLink.GetRequestParams()
	if requestParams == nil {
		requestParams = make(map[string]interface{})
	}

	// 解析响应参数
	responseParams, _ := quickLink.GetResponseParams()
	if responseParams == nil {
		responseParams = make(map[string]interface{})
	}

	// 解析字段元数据
	fieldMetadata, _ := quickLink.GetFieldMetadata()
	if fieldMetadata == nil {
		fieldMetadata = make(map[string]interface{})
	}

	// 转换字段元数据格式（从 map[string]interface{} 转换为 map[string]map[string]interface{}）
	convertedFieldMetadata := make(map[string]map[string]interface{})
	for k, v := range fieldMetadata {
		if m, ok := v.(map[string]interface{}); ok {
			convertedFieldMetadata[k] = m
		}
	}

	// 解析其他元数据
	metadata, _ := quickLink.GetMetadata()
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return &dto.GetQuickLinkResp{
		ID:             quickLink.ID,
		Name:           quickLink.Name,
		FunctionRouter: quickLink.FunctionRouter,
		FunctionMethod: quickLink.FunctionMethod,
		TemplateType:   quickLink.TemplateType,
		RequestParams:  requestParams,
		ResponseParams: responseParams,
		FieldMetadata:  convertedFieldMetadata,
		Metadata:       metadata,
		CreatedAt:      time.Time(quickLink.CreatedAt).Format(time.RFC3339),
		UpdatedAt:      time.Time(quickLink.UpdatedAt).Format(time.RFC3339),
		CreatedBy:      quickLink.CreatedBy,
	}
}

// convertFieldMetadata 转换字段元数据格式（从 map[string]map[string]interface{} 转换为 map[string]interface{}）
func convertFieldMetadata(fieldMetadata map[string]map[string]interface{}) map[string]interface{} {
	if fieldMetadata == nil {
		return nil
	}
	result := make(map[string]interface{})
	for k, v := range fieldMetadata {
		result[k] = v
	}
	return result
}
