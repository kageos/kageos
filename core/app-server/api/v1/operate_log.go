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

// OperateLog 操作日志相关API
type OperateLog struct {
	operateLogService *service.OperateLogService
	teamAccessService *service.TeamAccessService
}

func NewOperateLog(operateLogService *service.OperateLogService, teamAccessService *service.TeamAccessService) *OperateLog {
	return &OperateLog{operateLogService: operateLogService, teamAccessService: teamAccessService}
}

// GetOperateLogs 查询通用操作日志
// @Summary 查询通用操作日志
// @Description 查询统一存储在 operate_logs 中的操作日志
// @Tags 操作日志
// @Accept json
// @Produce json
// @Param X-Token header string true "JWT Token"
// @Param id query int false "操作日志 ID"
// @Param tenant_user query string false "租户用户（app 的所有者）"
// @Param company_code query string false "企业代码（默认当前登录企业）"
// @Param actor_user query string false "执行用户"
// @Param target_user query string false "被操作用户"
// @Param app query string false "应用名"
// @Param resource_type query string false "资源类型：table/form/team_access/directory/function"
// @Param resource_path query string false "资源路径"
// @Param resource_path_prefix query string false "资源路径前缀"
// @Param action query string false "操作类型"
// @Param status query string false "状态：success/failed"
// @Param source query string false "操作来源：browser/agent/openapi/public_share"
// @Param source_type query string false "来源类型：openapi_token/public_share/agent_tool/scheduled_task"
// @Param source_ref query string false "来源引用"
// @Param executor_type query string false "实际执行者类型：user/agent/scheduled_function/openapi/public_share"
// @Param workspace_session_id query string false "工作台会话 ID"
// @Param trace_id query string false "Trace ID"
// @Param row_id query int false "Table 记录 ID"
// @Param keyword query string false "关键词：匹配操作人、被操作人、资源路径、Trace 或摘要"
// @Param page query int false "页码（从1开始）" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param order_by query string false "排序字段（默认：created_at DESC）"
// @Success 200 {object} dto.GetOperateLogsResp "查询成功"
// @Router /workspace/api/v1/operate_log/general [get]
func (o *OperateLog) GetOperateLogs(c *gin.Context) {
	var req dto.GetOperateLogsReq
	var resp *dto.GetOperateLogsResp
	var err error
	defer func() {
		total := int64(0)
		if resp != nil {
			total = resp.Total
		}
		logger.Infof(c, "GetOperateLogs resource_path=%s resource_path_prefix=%s page=%d page_size=%d total=%d err=%v",
			req.ResourcePath, req.ResourcePathPrefix, req.Page, req.PageSize, total, err)
	}()

	if err := c.ShouldBindQuery(&req); err != nil {
		response.Internal(c, "参数绑定失败: "+err.Error())
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
	auditResourcePath := req.ResourcePath
	if auditResourcePath == "" {
		auditResourcePath = req.ResourcePathPrefix
	}
	if auditResourcePath == "" {
		response.BadRequest(c, "resource_path 或 resource_path_prefix 不能为空")
		return
	}
	auditResourcePath = access.NormalizeResourcePath(auditResourcePath)
	if err := requireAccess(c, o.teamAccessService, auditResourcePath, access.ActionRead); err != nil {
		response.Error(c, err)
		return
	}
	if req.ResourcePath != "" {
		req.ResourcePath = auditResourcePath
	}
	if req.ResourcePathPrefix != "" {
		req.ResourcePathPrefix = auditResourcePath
	}

	ctx := contextx.ToContext(c)
	if companyCode := contextx.GetRequestCompanyCode(ctx); companyCode != "" {
		if req.CompanyCode != "" && req.CompanyCode != companyCode {
			response.Internal(c, "不能查询其他企业的操作日志")
			return
		}
		req.CompanyCode = companyCode
	}
	resp, err = o.operateLogService.GetOperateLogs(ctx, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OkWithData(c, resp)
}
