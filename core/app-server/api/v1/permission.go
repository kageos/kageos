package v1

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/gin-gonic/gin"
)

// Permission 权限管理处理器
type Permission struct {
	permissionService *service.PermissionService
}

// NewPermission 创建权限管理处理器
func NewPermission(permissionService *service.PermissionService) *Permission {
	return &Permission{
		permissionService: permissionService,
	}
}

// AddPermission 添加权限
// @Summary 添加权限
// @Description 为用户添加资源权限
// @Tags 权限管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param body body dto.AddPermissionReq true "添加权限请求"
// @Success 200 {object} response.Response "添加成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/permission/add [post]
func (p *Permission) AddPermission(c *gin.Context) {
	var req dto.AddPermissionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	if err := p.permissionService.AddPermission(ctx, &req); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithMessage(c, "添加权限成功")
}

// ApplyPermission 权限申请
// @Summary 权限申请
// @Description 用户申请资源权限（简化版：直接添加权限，不创建申请记录）
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

	// ⭐ 从 context 中获取当前用户名（JWT 中间件已设置）
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.FailWithMessage(c, "无法获取当前用户信息")
		return
	}

	// ⭐ 确定要申请的权限点列表
	actions := make([]string, 0)
	if len(req.Actions) > 0 {
		// 如果提供了 actions 数组，使用数组
		actions = req.Actions
	} else if req.Action != "" {
		// 如果只提供了单个 action，使用单个
		actions = []string{req.Action}
	} else {
		response.FailWithMessage(c, "请至少指定一个操作类型")
		return
	}

	// ⭐ 批量添加权限（不创建申请记录）
	// 后续可以扩展为：创建申请记录，等待管理员审核
	successCount := 0
	failedActions := make([]string, 0)

	for _, action := range actions {
		addReq := dto.AddPermissionReq{
			Subject:      username,
			ResourcePath: req.ResourcePath,
			Action:       action,
			EndTime:      req.EndTime, // 传递有效期参数
		}

		if err := p.permissionService.AddPermission(ctx, &addReq); err != nil {
			failedActions = append(failedActions, action)
			continue
		}
		successCount++
	}

	if successCount == 0 {
		response.FailWithMessage(c, "权限申请失败，所有权限点都添加失败")
		return
	}

	var message string
	if successCount == len(actions) {
		message = fmt.Sprintf("权限申请已批准，已成功添加 %d 个权限", successCount)
	} else {
		message = fmt.Sprintf("权限申请部分成功，已成功添加 %d/%d 个权限，失败：%v",
			successCount, len(actions), failedActions)
	}

	resp := dto.ApplyPermissionResp{
		ID:      "",         // 暂时返回空字符串，后续可以扩展为申请记录ID
		Status:  "approved", // 简化版：直接批准
		Message: message,
	}

	response.OkWithData(c, resp)
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
// @Param app_id query int true "应用ID"
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

	// ⭐ 参数验证：必须提供 user 和 app
	if req.User == "" || req.App == "" {
		response.FailWithMessage(c, "必须提供 user 和 app 参数")
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

// ApprovePermissionRequest 审批通过权限申请
// @Summary 审批通过权限申请
// @Description 管理员审批通过权限申请，创建权限记录
// @Tags 权限管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param body body dto.ApprovePermissionRequestReq true "审批请求"
// @Success 200 {object} response.Response "审批成功"
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
// @Success 200 {object} response.Response "拒绝成功"
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

// GrantPermission 授权权限（管理员主动授权）
// @Summary 授权权限
// @Description 管理员主动授权权限，不需要审批流程
// @Tags 权限管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param body body dto.GrantPermissionReq true "授权请求"
// @Success 200 {object} response.Response "授权成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/permission/grant [post]
func (p *Permission) GrantPermission(c *gin.Context) {
	var req dto.GrantPermissionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	if err := p.permissionService.GrantPermission(ctx, &req); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithMessage(c, "授权成功")
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
