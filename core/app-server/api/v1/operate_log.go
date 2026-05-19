package v1

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// OperateLog 操作日志相关API
type OperateLog struct {
	operateLogService *service.OperateLogService
}

func NewOperateLog(operateLogService *service.OperateLogService) *OperateLog {
	return &OperateLog{operateLogService: operateLogService}
}

// GetTableOperateLogs 查询 Table 操作日志
// @Summary 查询 Table 操作日志
// @Description 查询 Table 操作日志
// @Tags 操作日志
// @Accept json
// @Produce json
// @Param X-Token header string true "JWT Token"
// @Param tenant_user query string false "租户用户（app 的所有者）"
// @Param request_user query string false "请求用户（实际执行操作的用户）"
// @Param app query string false "应用名"
// @Param full_code_path query string false "完整代码路径"
// @Param full_code_path_prefix query string false "完整代码路径前缀，用于查询目录下日志"
// @Param row_id query int false "记录ID"
// @Param action query string false "操作类型：OnTableAddRow, OnTableUpdateRow, OnTableDeleteRows"
// @Param keyword query string false "关键词：匹配操作人、资源路径、Trace、版本或记录ID"
// @Param page query int false "页码（从1开始）" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param order_by query string false "排序字段（默认：created_at DESC）"
// @Success 200 {object} dto.GetTableOperateLogsResp "查询成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未认证"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/operate_log/table [get]
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

	ctx := contextx.ToContext(c)
	resp, err = o.operateLogService.GetTableOperateLogs(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}
