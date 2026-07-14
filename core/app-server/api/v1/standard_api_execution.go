package v1

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/logger"
)

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
// @Router /workspace/api/v1/form-submissions/{full-code-path} [post]
func (s *StandardAPI) FormSubmit(c *gin.Context) {
	fullCodePath := normalizeFullCodePathParam(c)
	if fullCodePath == "" {
		response.BadRequest(c, "full-code-path 参数不能为空")
		return
	}
	if err := requireWorkspaceDataAccess(c, s.teamAccessService, fullCodePath, access.ActionWrite); err != nil {
		response.Error(c, err)
		return
	}

	// 构建请求对象
	req, err := s.buildRequestAppReq(c, fullCodePath)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, err := s.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()

	formLogReq := &dto.RecordFormOperateLogReq{
		TenantUser:     req.User,
		RequestUser:    req.RequestUser,
		App:            req.App,
		Router:         req.Router,
		Action:         "form_submit",
		FunctionMethod: req.Method,
		RequestBody:    req.Body,
		ResponseBody:   buildFormOperateLogResponseBody(resp, err, mill),
		IPAddress:      c.ClientIP(),
		UserAgent:      c.GetHeader("User-Agent"),
		TraceID:        req.TraceId,
		DurationMillis: mill,
		Status:         "success",
		Summary:        "表单提交成功",
	}
	if resp != nil {
		formLogReq.Version = resp.Version
	}
	if err != nil || (resp != nil && resp.Error != "") {
		formLogReq.Status = "failed"
		formLogReq.Summary = "表单提交失败"
	}
	if logErr := s.appService.RecordFormOperateLog(ctx, formLogReq); logErr != nil {
		logger.Warnf(ctx, "[FormSubmit] 记录 Form 操作日志失败: %v", logErr)
	}

	// 构建响应元数据
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill

	if err != nil {
		response.Error(c, err)
		return
	}

	if resp.Error != "" {
		response.ApplicationError(c, resp.ErrCode, resp.Error, metadata)
		return
	}

	s.appService.IncrementFunctionRunCount(ctx, "/"+strings.TrimPrefix(fullCodePath, "/"))
	response.OkWithData(c, resp.Result, metadata)
}

// RuntimePython 工作台 run_python 私有执行入口。
// 它只把请求转发到目标工作区应用内置的 /_runtime/python，不作为用户可见 Form 暴露。
func (s *StandardAPI) RuntimePython(c *gin.Context) {
	fullCodePath := normalizeFullCodePathParam(c)
	if fullCodePath == "" {
		response.BadRequest(c, "full-code-path 参数不能为空")
		return
	}
	if err := requireAgentToolRuntimeSource(c); err != nil {
		response.Error(c, err)
		return
	}
	_, _, workspaceRoot, err := parseWorkspaceRootPath(fullCodePath)
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := requireAccess(c, s.teamAccessService, workspaceRoot, access.ActionWrite); err != nil {
		response.Error(c, err)
		return
	}

	req, err := s.buildRuntimePythonRequestAppReq(c, workspaceRoot)
	if err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, err := s.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()

	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill

	if err != nil {
		response.Error(c, err)
		return
	}
	if resp.Error != "" {
		response.ApplicationError(c, resp.ErrCode, resp.Error, metadata)
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
// @Router /workspace/api/v1/charts/{full-code-path} [get]
func (s *StandardAPI) ChartQuery(c *gin.Context) {
	fullCodePath := normalizeFullCodePathParam(c)
	if fullCodePath == "" {
		response.BadRequest(c, "full-code-path 参数不能为空")
		return
	}
	if err := requireWorkspaceDataAccess(c, s.teamAccessService, fullCodePath, access.ActionRead); err != nil {
		response.Error(c, err)
		return
	}

	// 构建请求对象
	req, err := s.buildRequestAppReq(c, fullCodePath)
	if err != nil {
		response.Error(c, err)
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
		response.Error(c, err)
		return
	}

	if resp.Error != "" {
		response.ApplicationError(c, resp.ErrCode, resp.Error, metadata)
		return
	}

	s.appService.IncrementFunctionRunCount(ctx, "/"+strings.TrimPrefix(fullCodePath, "/"))
	response.OkWithData(c, resp.Result, metadata)
}
