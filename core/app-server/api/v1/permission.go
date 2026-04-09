package v1

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Permission 权限管理处理器（仅依赖 permissionService，不直接依赖 repository）
type Permission struct {
	permissionService *service.PermissionService
}

// NewPermission 创建权限管理处理器
func NewPermission(permissionService *service.PermissionService) *Permission {
	return &Permission{
		permissionService: permissionService,
	}
}

// ApplyPermission 权限申请
// @Summary 权限申请
// @Description 用户申请资源权限，创建申请记录，等待管理员审批
// @Tags 权限管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param body body dto.ApplyPermissionReq true "权限申请请求"
// @Success 200 {object} dto.ApplyPermissionResp "申请成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/permission/apply [post]
func (p *Permission) ApplyPermission(c *gin.Context) {
	var req dto.ApplyPermissionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)

	// ⭐ 权限申请逻辑在企业版目录（enterprise_impl ApplyPermissionByResourcePath），api 只调 service
	resp, err := p.permissionService.ApplyPermission(ctx, &req)
	if err != nil {
		logger.Errorf(ctx, "[ApplyPermission] 创建角色申请失败: role_id=%d, resource_path=%s, error=%v",
			req.RoleID, req.ResourcePath, err)
		response.FailWithMessage(c, "创建角色申请失败: "+err.Error())
		return
	}

	response.OkWithData(c, dto.ApplyPermissionResp{
		ID:      fmt.Sprintf("%d", resp.RequestID),
		Status:  "pending",
		Message: "角色申请已提交，等待审批",
	})
}

// GetWorkspacePermissions 获取工作空间的所有权限
// @Summary 获取工作空间权限
// @Description 获取整个工作空间（应用）的所有节点权限，用于权限申请页面显示已有权限。
// @Description 支持两种方式：
// @Description 1. 获取当前用户权限：不传 username 和 department_full_path，系统从 context 中获取（JWT 中间件已设置）
// @Description 2. 获取指定用户权限：传递 username 和 department_full_path 参数，可以查询其他用户的权限（需要管理员权限）
// @Tags 权限管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param resource_path query string true "工作空间资源路径，格式 /user/app"
// @Param username query string false "用户名（可选，不传则获取当前用户权限）"
// @Param department_full_path query string false "组织架构路径（可选，不传则从 context 获取）"
// @Success 200 {object} dto.GetWorkspacePermissionsResp "查询成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/permission/workspace [get]
func (p *Permission) GetWorkspacePermissions(c *gin.Context) {
	var req dto.GetWorkspacePermissionsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	if req.ResourcePath == "" && (req.User == "" || req.App == "") {
		response.FailWithMessage(c, "必须提供 resource_path 或 user/app 参数")
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := p.permissionService.GetWorkspacePermissions(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// CreatePermissionRequest 创建权限申请
// @Summary 创建权限申请
// @Description 用户申请资源权限，创建申请记录，等待管理员审批
// @Tags 权限管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param body body dto.CreatePermissionRequestReq true "权限申请请求"
// @Success 200 {object} dto.CreatePermissionRequestResp "申请成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/permission/request/create [post]
func (p *Permission) CreatePermissionRequest(c *gin.Context) {
	var req dto.CreatePermissionRequestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := p.permissionService.CreatePermissionRequest(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// GetResourcePermissions 查询资源的所有权限分配
// @Summary 查询资源权限
// @Description 查询指定资源路径的所有权限分配（用于权限管理 Tab）
// @Tags 权限管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param user query string true "租户用户"
// @Param app query string true "应用代码"
// @Param resource_path query string true "资源路径（full-code-path）"
// @Success 200 {object} dto.GetResourcePermissionsResp "查询成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/permission/resource [get]
func (p *Permission) GetResourcePermissions(c *gin.Context) {
	var req dto.GetResourcePermissionsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := p.permissionService.GetResourcePermissions(ctx, &req)
	if err != nil {
		logger.Errorf(ctx, "[GetResourcePermissions] 查询资源权限失败: resource_path=%s, error=%v",
			req.ResourcePath, err)
		response.FailWithMessage(c, "查询资源权限失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// ApprovePermissionRequest 审批通过权限申请
// @Summary 审批通过权限申请
// @Description 管理员审批通过权限申请，创建权限记录
// @Tags 权限管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param body body dto.ApprovePermissionRequestReq true "审批请求"
// @Success 200 {object} map[string]interface{} "审批成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/permission/request/approve [post]
func (p *Permission) ApprovePermissionRequest(c *gin.Context) {
	var req dto.ApprovePermissionRequestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	if err := p.permissionService.ApprovePermissionRequest(ctx, &req); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithMessage(c, "审批通过成功")
}

// RejectPermissionRequest 审批拒绝权限申请
// @Summary 审批拒绝权限申请
// @Description 管理员审批拒绝权限申请
// @Tags 权限管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param body body dto.RejectPermissionRequestReq true "拒绝请求"
// @Success 200 {object} map[string]interface{} "拒绝成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/permission/request/reject [post]
func (p *Permission) RejectPermissionRequest(c *gin.Context) {
	var req dto.RejectPermissionRequestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	if err := p.permissionService.RejectPermissionRequest(ctx, &req); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithMessage(c, "审批拒绝成功")
}

// GetPermissionRequests 获取权限申请列表
// @Summary 获取权限申请列表
// @Description 获取权限申请列表，支持筛选和分页
// @Tags 权限管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param app_id query int false "工作空间ID"
// @Param status query string false "申请状态（pending、approved、rejected）"
// @Param applicant query string false "申请人用户名"
// @Param resource_path query string false "资源路径"
// @Param page query int false "页码（默认1）"
// @Param page_size query int false "每页数量（默认20）"
// @Success 200 {object} dto.GetPermissionRequestsResp "查询成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/permission/requests [get]
func (p *Permission) GetPermissionRequests(c *gin.Context) {
	var req dto.GetPermissionRequestsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
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
	resp, err := p.permissionService.GetPermissionRequests(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}
