package v1

import (
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// OperateLog 操作日志相关API
type OperateLog struct {
	// 使用企业版接口，通过 enterprise.GetOperateLogger() 获取实现
}

// NewOperateLog 创建操作日志API（依赖注入）
// 注意：现在使用企业版接口，不再需要传入核心服务层
func NewOperateLog() *OperateLog {
	return &OperateLog{}
}

// GetTableOperateLogs 查询 Table 操作日志
// @Summary 查询 Table 操作日志
// @Description 查询 Table 操作日志（需要企业版 License）
// @Tags 操作日志
// @Accept json
// @Produce json
// @Param X-Token header string true "JWT Token"
// @Param tenant_user query string false "租户用户（app 的所有者）"
// @Param request_user query string false "请求用户（实际执行操作的用户）"
// @Param app query string false "应用名"
// @Param full_code_path query string false "完整代码路径"
// @Param row_id query int false "记录ID"
// @Param action query string false "操作类型：OnTableAddRow, OnTableUpdateRow, OnTableDeleteRows"
// @Param page query int false "页码（从1开始）" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param order_by query string false "排序字段（默认：created_at DESC）"
// @Success 200 {object} dto.GetTableOperateLogsResp "查询成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未认证"
// @Failure 403 {string} string "需要企业版 License"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/operate_log/table [get]
func (o *OperateLog) GetTableOperateLogs(c *gin.Context) {
	var req dto.GetTableOperateLogsReq
	var resp *dto.GetTableOperateLogsResp
	var err error
	defer func() {
		logger.Infof(c, "GetTableOperateLogs req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 绑定查询参数
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数绑定失败: "+err.Error())
		return
	}

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	// 调用企业版接口
	ctx := contextx.ToContext(c)
	operateLogger := enterprise.GetOperateLogger()
	resp, err = operateLogger.GetTableOperateLogs(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// GetFormOperateLogs 查询 Form 操作日志
// @Summary 查询 Form 操作日志
// @Description 查询 Form 操作日志（需要企业版 License）
// @Tags 操作日志
// @Accept json
// @Produce json
// @Param X-Token header string true "JWT Token"
// @Param tenant_user query string false "租户用户（app 的所有者）"
// @Param request_user query string false "请求用户（实际执行操作的用户）"
// @Param app query string false "应用名"
// @Param full_code_path query string false "完整代码路径"
// @Param action query string false "操作类型：request_app, form_submit"
// @Param page query int false "页码（从1开始）" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param order_by query string false "排序字段（默认：created_at DESC）"
// @Success 200 {object} dto.GetFormOperateLogsResp "查询成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未认证"
// @Failure 403 {string} string "需要企业版 License"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/operate_log/form [get]
func (o *OperateLog) GetFormOperateLogs(c *gin.Context) {
	var req dto.GetFormOperateLogsReq
	var resp *dto.GetFormOperateLogsResp
	var err error
	defer func() {
		logger.Infof(c, "GetFormOperateLogs req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 绑定查询参数
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数绑定失败: "+err.Error())
		return
	}

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	// 调用企业版接口
	ctx := contextx.ToContext(c)
	operateLogger := enterprise.GetOperateLogger()
	resp, err = operateLogger.GetFormOperateLogs(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

