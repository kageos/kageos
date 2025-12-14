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

type Function struct {
	functionService *service.FunctionService
}

// NewFunction 创建 Function 处理器（依赖注入）
func NewFunction(functionService *service.FunctionService) *Function {
	return &Function{
		functionService: functionService,
	}
}

// GetFunction 获取函数详情
// @Summary 获取函数详情
// @Description 根据函数ID获取函数的详细信息
// @Tags 函数管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param function_id query int true "函数ID"
// @Success 200 {object} dto.GetFunctionResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "函数不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/function/get [get]
func (f *Function) GetFunction(c *gin.Context) {
	var resp *dto.GetFunctionResp
	var err error

	// 从query参数获取函数ID
	functionIDStr := c.Query("function_id")
	if functionIDStr == "" {
		response.FailWithMessage(c, "缺少function_id参数")
		return
	}

	functionID, err := strconv.ParseInt(functionIDStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "无效的函数ID")
		return
	}

	ctx := contextx.ToContext(c)
	resp, err = f.functionService.GetFunction(ctx, functionID)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// GetFunctionsByApp 获取应用下所有函数
// @Summary 获取应用下所有函数
// @Description 根据应用ID获取该应用下的所有函数列表
// @Tags 函数管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param app_id query int true "应用ID"
// @Success 200 {object} dto.GetFunctionsByAppResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/function/list [get]
func (f *Function) GetFunctionsByApp(c *gin.Context) {
	var resp *dto.GetFunctionsByAppResp
	var err error

	// 从query参数获取应用ID
	appIDStr := c.Query("app_id")
	if appIDStr == "" {
		response.FailWithMessage(c, "缺少app_id参数")
		return
	}

	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "无效的应用ID")
		return
	}

	defer func() {
		logger.Infof(c, "GetFunctionsByApp app_id:%d resp:%+v err:%v", appID, resp, err)
	}()

	ctx := contextx.ToContext(c)
	resp, err = f.functionService.GetFunctionsByApp(ctx, appID)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// ForkFunctionGroup Fork 函数组（支持批量）
// @Summary Fork 函数组
// @Description 批量 Fork 函数组到目标 package
// @Tags 函数管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param request body dto.ForkFunctionGroupReq true "Fork 请求，source_to_target_map 的 key=函数组的full_group_code，value=服务目录的full_code_path"
// @Success 200 {object} dto.ForkFunctionGroupResp "Fork 成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/function/fork [post]
func (f *Function) ForkFunctionGroup(c *gin.Context) {
	var req dto.ForkFunctionGroupReq
	var resp *dto.ForkFunctionGroupResp
	var err error

	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "ForkFunctionGroup req:%+v resp:%+v err:%v", req, resp, err)
	}()

	ctx := contextx.ToContext(c)
	resp, err = f.functionService.ForkFunctionGroup(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// GetFunctionGroupInfo 获取函数组信息（用于 Hub 发布）
// @Summary 获取函数组信息
// @Description 根据完整函数组代码获取函数组信息，包括源代码和描述，用于 Hub 发布
// @Tags 函数管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full_group_code query string true "完整函数组代码，格式：/{user}/{app}/{package_path}/{group_code}"
// @Success 200 {object} dto.GetFunctionGroupInfoResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "函数组不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/function/group-info [get]
func (f *Function) GetFunctionGroupInfo(c *gin.Context) {
	var resp *dto.GetFunctionGroupInfoResp
	var err error

	// 从 query 参数获取完整函数组代码
	fullGroupCode := c.Query("full_group_code")
	if fullGroupCode == "" {
		response.FailWithMessage(c, "缺少 full_group_code 参数")
		return
	}

	defer func() {
		logger.Infof(c, "GetFunctionGroupInfo full_group_code:%s resp:%+v err:%v", fullGroupCode, resp, err)
	}()

	ctx := contextx.ToContext(c)
	resp, err = f.functionService.GetFunctionGroupInfo(ctx, fullGroupCode)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}
