package v1

import (
	"strings"

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

// RemovePermission 删除权限
// @Summary 删除权限
// @Description 删除用户的资源权限
// @Tags 权限管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param body body dto.RemovePermissionReq true "删除权限请求"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/permission/remove [post]
func (p *Permission) RemovePermission(c *gin.Context) {
	var req dto.RemovePermissionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	if err := p.permissionService.RemovePermission(ctx, &req); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithMessage(c, "删除权限成功")
}

// GetUserPermissions 获取用户权限
// @Summary 获取用户权限
// @Description 查询用户对指定资源的权限
// @Tags 权限管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param username query string true "用户名"
// @Param resource_path query string false "资源路径（可选）"
// @Param actions query string false "操作类型列表，多个用逗号分隔（可选）"
// @Success 200 {object} dto.GetUserPermissionsResp "查询成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/permission/user [get]
func (p *Permission) GetUserPermissions(c *gin.Context) {
	var req dto.GetUserPermissionsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 解析 actions 参数（如果提供了）
	if actionsStr := c.Query("actions"); actionsStr != "" {
		// 将逗号分隔的字符串转换为数组
		actions := make([]string, 0)
		for _, action := range strings.Split(actionsStr, ",") {
			action = strings.TrimSpace(action)
			if action != "" {
				actions = append(actions, action)
			}
		}
		req.Actions = actions
	}

	ctx := contextx.ToContext(c)
	resp, err := p.permissionService.GetUserPermissions(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// AssignRoleToUser 分配角色给用户
// @Summary 分配角色给用户
// @Description 为用户分配角色
// @Tags 权限管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param body body dto.AssignRoleToUserReq true "分配角色请求"
// @Success 200 {object} response.Response "分配成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/permission/role/assign [post]
func (p *Permission) AssignRoleToUser(c *gin.Context) {
	var req dto.AssignRoleToUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	if err := p.permissionService.AssignRoleToUser(ctx, &req); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithMessage(c, "分配角色成功")
}

// RemoveRoleFromUser 从用户移除角色
// @Summary 从用户移除角色
// @Description 从用户移除角色
// @Tags 权限管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param body body dto.RemoveRoleFromUserReq true "移除角色请求"
// @Success 200 {object} response.Response "移除成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/permission/role/remove [post]
func (p *Permission) RemoveRoleFromUser(c *gin.Context) {
	var req dto.RemoveRoleFromUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	if err := p.permissionService.RemoveRoleFromUser(ctx, &req); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithMessage(c, "移除角色成功")
}

// GetUserRoles 获取用户角色
// @Summary 获取用户角色
// @Description 查询用户的所有角色
// @Tags 权限管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param username query string true "用户名"
// @Success 200 {object} dto.GetUserRolesResp "查询成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/permission/role/user [get]
func (p *Permission) GetUserRoles(c *gin.Context) {
	username := c.Query("username")
	if username == "" {
		response.FailWithMessage(c, "用户名不能为空")
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := p.permissionService.GetUserRoles(ctx, username)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
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
	
	// ⭐ 简化版：直接添加权限（不创建申请记录）
	// 后续可以扩展为：创建申请记录，等待管理员审核
	addReq := dto.AddPermissionReq{
		Username:     username,
		ResourcePath: req.ResourcePath,
		Action:       req.Action,
	}
	
	if err := p.permissionService.AddPermission(ctx, &addReq); err != nil {
		response.FailWithMessage(c, "权限申请失败: "+err.Error())
		return
	}

	resp := dto.ApplyPermissionResp{
		ID:      "", // 暂时返回空字符串，后续可以扩展为申请记录ID
		Status:  "approved", // 简化版：直接批准
		Message: "权限申请已批准，权限已添加",
	}

	response.OkWithData(c, resp)
}

