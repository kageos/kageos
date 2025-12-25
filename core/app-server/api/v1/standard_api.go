package v1

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// StandardAPI 标准接口处理器
// 提供标准化的 RESTful 接口，使用 full-code-path 作为路径参数
type StandardAPI struct {
	appService *service.AppService
}

// NewStandardAPI 创建标准接口处理器
func NewStandardAPI(appService *service.AppService) *StandardAPI {
	return &StandardAPI{
		appService: appService,
	}
}

// parseFullCodePath 从路径参数解析 full-code-path
// 格式：/{user}/{app}/{...}
func parseFullCodePath(fullCodePath string) (user, app string, router string, err error) {
	// 移除开头的斜杠
	fullCodePath = strings.TrimPrefix(fullCodePath, "/")
	parts := strings.Split(fullCodePath, "/")

	if len(parts) < 3 {
		return "", "", "", fmt.Errorf("full-code-path 格式错误，至少需要包含 user/app/function")
	}

	user = parts[0]
	app = parts[1]
	router = strings.Join(parts[2:], "/")

	return user, app, router, nil
}

// buildRequestAppReq 构建 RequestAppReq 请求对象
func (s *StandardAPI) buildRequestAppReq(c *gin.Context, fullCodePath string) (*dto.RequestAppReq, error) {
	user, app, router, err := parseFullCodePath(fullCodePath)
	if err != nil {
		return nil, err
	}

	req := &dto.RequestAppReq{
		User:        user,
		App:         app,
		Router:      router,
		Method:      c.Request.Method,
		TraceId:     contextx.GetTraceId(c),
		RequestUser: contextx.GetRequestUser(c),
		Token:       contextx.GetToken(c),
	}

	// 绑定请求体（POST、PUT、PATCH、DELETE 等方法通常有请求体）
	if c.Request.ContentLength > 0 && (c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" || c.Request.Method == "DELETE") {
		all, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return nil, err
		}
		defer c.Request.Body.Close()
		req.Body = all
	}

	// 绑定查询参数
	req.UrlQuery = c.Request.URL.RawQuery

	return req, nil
}

// buildCallbackAppReq 构建 CallbackApp 请求对象
func (s *StandardAPI) buildCallbackAppReq(c *gin.Context, fullCodePath string, callbackType string) (*dto.RequestAppReq, error) {
	user, app, router, err := parseFullCodePath(fullCodePath)
	if err != nil {
		return nil, err
	}

	req := &dto.RequestAppReq{
		User:        user,
		App:         app,
		Router:      "/_callback",
		Method:      c.Request.Method,
		TraceId:     contextx.GetTraceId(c),
		RequestUser: contextx.GetRequestUser(c),
		Token:       contextx.GetToken(c),
	}

	// 读取请求体
	all, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	defer c.Request.Body.Close()

	// 构建回调请求体
	mp := make(map[string]interface{})
	mp["method"] = c.Request.Method
	mp["router"] = router
	mp["body"] = all
	mp["type"] = callbackType

	// 绑定查询参数
	req.UrlQuery = c.Request.URL.RawQuery

	// 将回调信息序列化为 JSON
	marshal, err := json.Marshal(mp)
	if err != nil {
		return nil, err
	}
	req.Body = marshal

	return req, nil
}

// ============================================
// Table 函数接口
// ============================================

// TableSearch Table 查询接口
// @Summary Table 查询
// @Description 查询表格数据（列表），支持分页、排序、搜索
// @Tags 标准接口
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full-code-path path string true "函数完整路径，如：/luobei/operations/tools/pdftools/to_images"
// @Param page query int false "页码（可选，默认 1）"
// @Param page_size query int false "每页数量（可选，默认 20）"
// @Param sorts query string false "排序（可选，格式：id:desc,name:asc）"
// @Success 200 {object} dto.RequestAppResp "查询成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "权限不足"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/table/search/{full-code-path} [get]
func (s *StandardAPI) TableSearch(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "full-code-path 参数不能为空")
		return
	}

	// 构建请求对象
	req, err := s.buildRequestAppReq(c, fullCodePath)
	if err != nil {
		response.FailWithMessage(c, "解析路径参数失败: "+err.Error())
		return
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, err := s.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()

	// 构建响应元数据
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill

	if err != nil {
		response.FailWithMessage(c, err.Error(), metadata)
		return
	}

	if resp.Error != "" {
		response.Result(resp.ErrCode, nil, resp.Error, c, metadata)
		return
	}

	response.OkWithData(c, resp.Result, metadata)
}

// TableCreate Table 新增接口
// @Summary Table 新增
// @Description 新增表格记录
// @Tags 标准接口
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full-code-path path string true "函数完整路径，如：/luobei/operations/tools/pdftools/to_images"
// @Param body body object true "新增记录的字段数据"
// @Success 200 {object} dto.RequestAppResp "新增成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "权限不足"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/table/create/{full-code-path} [post]
func (s *StandardAPI) TableCreate(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "full-code-path 参数不能为空")
		return
	}

	// 构建回调请求对象（调用 OnTableAddRow）
	req, err := s.buildCallbackAppReq(c, fullCodePath, "OnTableAddRow")
	if err != nil {
		response.FailWithMessage(c, "构建请求失败: "+err.Error())
		return
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, err := s.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()

	// 构建响应元数据
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill

	if err != nil {
		response.FailWithMessage(c, err.Error(), metadata)
		return
	}

	if resp.Error != "" {
		response.Result(resp.ErrCode, nil, resp.Error, c, metadata)
		return
	}

	response.OkWithData(c, resp.Result, metadata)
}

// TableUpdate Table 更新接口
// @Summary Table 更新
// @Description 更新表格记录
// @Tags 标准接口
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full-code-path path string true "函数完整路径，如：/luobei/operations/tools/pdftools/to_images"
// @Param body body object true "更新记录的字段数据（必须包含 id 字段）"
// @Success 200 {object} dto.RequestAppResp "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "权限不足"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/table/update/{full-code-path} [put]
func (s *StandardAPI) TableUpdate(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "full-code-path 参数不能为空")
		return
	}

	// 读取请求体，用于记录操作日志
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.FailWithMessage(c, "读取请求体失败: "+err.Error())
		return
	}
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes))) // 重新设置请求体，供后续使用

	// 构建回调请求对象（调用 OnTableUpdateRow）
	req, err := s.buildCallbackAppReq(c, fullCodePath, "OnTableUpdateRow")
	if err != nil {
		response.FailWithMessage(c, "构建请求失败: "+err.Error())
		return
	}

	// 解析请求体，用于记录操作日志
	var bodyData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &bodyData); err == nil {
		user, app, router, _ := parseFullCodePath(fullCodePath)
		logReq := &dto.RecordTableOperateLogReq{
			TenantUser:  user,
			RequestUser: req.RequestUser,
			App:         app,
			Router:      router,
			Action:      "OnTableUpdateRow",
			IPAddress:   c.ClientIP(),
			UserAgent:   c.GetHeader("User-Agent"),
			TraceID:     req.TraceId,
		}

		// 获取 row_id
		if rowIDStr := c.Query("_row_id"); rowIDStr != "" {
			if id, err := strconv.ParseInt(rowIDStr, 10, 64); err == nil {
				logReq.RowID = id
			}
		} else if id, ok := bodyData["id"].(float64); ok {
			logReq.RowID = int64(id)
		}

		// 获取 updates 和 old_values
		if updatesData, ok := bodyData["updates"].(map[string]interface{}); ok {
			logReq.Updates, _ = json.Marshal(updatesData)
		}
		if oldValuesData, ok := bodyData["old_values"].(map[string]interface{}); ok {
			logReq.OldValues, _ = json.Marshal(oldValuesData)
		}

		// 异步记录操作日志
		ctx := contextx.ToContext(c)
		go func() {
			if err := s.appService.RecordTableOperateLog(ctx, logReq); err != nil {
				logger.Warnf(ctx, "[TableUpdate] 记录 Table 更新操作日志失败: %v", err)
			}
		}()
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, err := s.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()

	// 构建响应元数据
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill

	if err != nil {
		response.FailWithMessage(c, err.Error(), metadata)
		return
	}

	if resp.Error != "" {
		response.Result(resp.ErrCode, nil, resp.Error, c, metadata)
		return
	}

	response.OkWithData(c, resp.Result, metadata)
}

// TableDelete Table 删除接口
// @Summary Table 删除
// @Description 删除表格记录（支持批量删除）
// @Tags 标准接口
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full-code-path path string true "函数完整路径，如：/luobei/operations/tools/pdftools/to_images"
// @Param body body object true "删除记录的ID列表，格式：{\"ids\": [1, 2, 3]}"
// @Success 200 {object} dto.RequestAppResp "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "权限不足"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/table/delete/{full-code-path} [delete]
func (s *StandardAPI) TableDelete(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "full-code-path 参数不能为空")
		return
	}

	// 读取请求体，用于记录操作日志
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.FailWithMessage(c, "读取请求体失败: "+err.Error())
		return
	}
	// 重新设置请求体，供后续使用
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	// 构建回调请求对象（调用 OnTableDeleteRows）
	req, err := s.buildCallbackAppReq(c, fullCodePath, "OnTableDeleteRows")
	if err != nil {
		response.FailWithMessage(c, "构建请求失败: "+err.Error())
		return
	}

	// 解析请求体，用于记录操作日志
	var bodyData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &bodyData); err == nil {
		user, app, router, _ := parseFullCodePath(fullCodePath)
		logReq := &dto.RecordTableOperateLogReq{
			TenantUser:  user,
			RequestUser: req.RequestUser,
			App:         app,
			Router:      router,
			Action:      "OnTableDeleteRows",
			IPAddress:   c.ClientIP(),
			UserAgent:   c.GetHeader("User-Agent"),
			TraceID:     req.TraceId,
		}

		// 获取 ids 列表
		if ids, ok := bodyData["ids"].([]interface{}); ok {
			rowIDs := make([]int64, 0, len(ids))
			for _, id := range ids {
				if idFloat, ok := id.(float64); ok {
					rowIDs = append(rowIDs, int64(idFloat))
				}
			}
			logReq.RowIDs = rowIDs
		}

		// 异步记录操作日志
		ctx := contextx.ToContext(c)
		go func() {
			if err := s.appService.RecordTableOperateLog(ctx, logReq); err != nil {
				logger.Warnf(ctx, "[TableDelete] 记录 Table 删除操作日志失败: %v", err)
			}
		}()
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, err := s.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()

	// 构建响应元数据
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill

	if err != nil {
		response.FailWithMessage(c, err.Error(), metadata)
		return
	}

	if resp.Error != "" {
		response.Result(resp.ErrCode, nil, resp.Error, c, metadata)
		return
	}

	response.OkWithData(c, resp.Result, metadata)
}

// ============================================
// Form 函数接口
// ============================================

// FormSubmit Form 提交接口
// @Summary Form 提交
// @Description 提交表单数据
// @Tags 标准接口
// @Accept json
// @Accept application/x-www-form-urlencoded
// @Accept multipart/form-data
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full-code-path path string true "函数完整路径，如：/luobei/operations/tools/pdftools/to_images"
// @Param body body object true "表单字段数据"
// @Success 200 {object} dto.RequestAppResp "提交成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "权限不足"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/form/submit/{full-code-path} [post]
func (s *StandardAPI) FormSubmit(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "full-code-path 参数不能为空")
		return
	}

	// 构建请求对象
	req, err := s.buildRequestAppReq(c, fullCodePath)
	if err != nil {
		response.FailWithMessage(c, "解析路径参数失败: "+err.Error())
		return
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, err := s.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()

	// 构建响应元数据
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill

	if err != nil {
		response.FailWithMessage(c, err.Error(), metadata)
		return
	}

	if resp.Error != "" {
		response.Result(resp.ErrCode, nil, resp.Error, c, metadata)
		return
	}

	response.OkWithData(c, resp.Result, metadata)
}

// ============================================
// Chart 函数接口
// ============================================

// ChartQuery Chart 查询接口
// @Summary Chart 查询
// @Description 查询图表数据
// @Tags 标准接口
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full-code-path path string true "函数完整路径，如：/luobei/operations/tools/pdftools/to_images"
// @Param query query object false "图表查询条件"
// @Success 200 {object} dto.RequestAppResp "查询成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "权限不足"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/chart/query/{full-code-path} [get]
func (s *StandardAPI) ChartQuery(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "full-code-path 参数不能为空")
		return
	}

	// 构建请求对象
	req, err := s.buildRequestAppReq(c, fullCodePath)
	if err != nil {
		response.FailWithMessage(c, "解析路径参数失败: "+err.Error())
		return
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, err := s.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()

	// 构建响应元数据
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill

	if err != nil {
		response.FailWithMessage(c, err.Error(), metadata)
		return
	}

	if resp.Error != "" {
		response.Result(resp.ErrCode, nil, resp.Error, c, metadata)
		return
	}

	response.OkWithData(c, resp.Result, metadata)
}

// ============================================
// Callback 接口
// ============================================

// CallbackOnSelectFuzzy 模糊搜索回调接口
// @Summary 模糊搜索回调
// @Description Select 组件的模糊搜索回调
// @Tags 标准接口
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full-code-path path string true "函数完整路径，如：/luobei/operations/tools/pdftools/to_images"
// @Param body body object true "搜索条件，格式：{\"code\": \"field_code\", \"type\": \"by_values\", \"value\": [1, 2, 3], \"request\": {...}}"
// @Success 200 {object} dto.RequestAppResp "查询成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "权限不足"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/callback/on_select_fuzzy/{full-code-path} [post]
func (s *StandardAPI) CallbackOnSelectFuzzy(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "full-code-path 参数不能为空")
		return
	}

	// 构建回调请求对象（调用 OnSelectFuzzy）
	req, err := s.buildCallbackAppReq(c, fullCodePath, "OnSelectFuzzy")
	if err != nil {
		response.FailWithMessage(c, "构建请求失败: "+err.Error())
		return
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, err := s.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()

	// 构建响应元数据
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill

	if err != nil {
		response.FailWithMessage(c, err.Error(), metadata)
		return
	}

	if resp.Error != "" {
		response.Result(resp.ErrCode, nil, resp.Error, c, metadata)
		return
	}

	response.OkWithData(c, resp.Result, metadata)
}

