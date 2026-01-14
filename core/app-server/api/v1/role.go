package v1

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/gin-gonic/gin"
)

// Role 角色管理 API
type Role struct {
	permissionService enterprise.PermissionService
}

// NewRoleHandlerFromPermissionService 创建角色管理 API 处理器（从 PermissionService 创建）
func NewRoleHandlerFromPermissionService(permissionService enterprise.PermissionService) *Role {
	return &Role{
		permissionService: permissionService,
	}
}

// CreateRole 创建角色
// @Summary 创建角色
// @Description 创建自定义角色，配置权限点
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param body body dto.CreateRoleReq true "创建角色请求"
// @Success 200 {object} dto.CreateRoleResp "创建成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/role [post]
func (r *Role) CreateRole(c *gin.Context) {
	var req dto.CreateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := r.permissionService.CreateRole(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// UpdateRole 更新角色
// @Summary 更新角色
// @Description 更新角色信息或权限配置
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param id path int true "角色ID"
// @Param body body dto.UpdateRoleReq true "更新角色请求"
// @Success 200 {object} dto.UpdateRoleResp "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/role/{id} [put]
func (r *Role) UpdateRole(c *gin.Context) {
	idStr := c.Param("id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		response.FailWithMessage(c, "角色ID无效")
		return
	}

	var req dto.UpdateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := r.permissionService.UpdateRole(ctx, id, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// DeleteRole 删除角色
// @Summary 删除角色
// @Description 删除自定义角色（系统角色不能删除）
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param id path int true "角色ID"
// @Success 200 {object} dto.DeleteRoleResp "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/role/{id} [delete]
func (r *Role) DeleteRole(c *gin.Context) {
	idStr := c.Param("id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		response.FailWithMessage(c, "角色ID无效")
		return
	}

	ctx := contextx.ToContext(c)
	if err := r.permissionService.DeleteRole(ctx, id); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithMessage(c, "删除角色成功")
}

// GetRole 获取角色
// @Summary 获取角色
// @Description 获取角色详细信息（包含权限列表）
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param id path int true "角色ID"
// @Success 200 {object} dto.GetRoleResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/role/{id} [get]
func (r *Role) GetRole(c *gin.Context) {
	idStr := c.Param("id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		response.FailWithMessage(c, "角色ID无效")
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := r.permissionService.GetRole(ctx, id)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// GetRoles 获取所有角色
// @Summary 获取所有角色
// @Description 获取所有角色列表（从内存缓存读取），支持按资源类型过滤
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param resource_type query string false "资源类型过滤（directory、table、form、chart、app）"
// @Success 200 {object} dto.GetRolesResp "获取成功"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/role [get]
func (r *Role) GetRoles(c *gin.Context) {
	resourceType := c.Query("resource_type")
	
	ctx := contextx.ToContext(c)
	resp, err := r.permissionService.GetRoles(ctx, resourceType)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// AssignRoleToUser 给用户分配角色
// @Summary 给用户分配角色
// @Description 给指定用户分配角色，指定资源路径
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param body body dto.AssignRoleToUserReq true "分配角色请求"
// @Success 200 {object} dto.AssignRoleToUserResp "分配成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/role/assign/user [post]
func (r *Role) AssignRoleToUser(c *gin.Context) {
	var req dto.AssignRoleToUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := r.permissionService.AssignRoleToUser(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// AssignRoleToDepartment 给组织架构分配角色
// @Summary 给组织架构分配角色
// @Description 给指定组织架构分配角色，组织架构下所有成员自动获得权限
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param body body dto.AssignRoleToDepartmentReq true "分配角色请求"
// @Success 200 {object} dto.AssignRoleToDepartmentResp "分配成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/role/assign/department [post]
func (r *Role) AssignRoleToDepartment(c *gin.Context) {
	var req dto.AssignRoleToDepartmentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := r.permissionService.AssignRoleToDepartment(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// RemoveRoleFromUser 移除用户角色
// @Summary 移除用户角色
// @Description 移除用户的角色分配
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param body body dto.RemoveRoleFromUserReq true "移除角色请求"
// @Success 200 {object} dto.RemoveRoleFromUserResp "移除成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/role/remove/user [post]
func (r *Role) RemoveRoleFromUser(c *gin.Context) {
	var req dto.RemoveRoleFromUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	if err := r.permissionService.RemoveRoleFromUser(ctx, &req); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithMessage(c, "移除角色成功")
}

// RemoveRoleFromDepartment 移除组织架构角色
// @Summary 移除组织架构角色
// @Description 移除组织架构的角色分配
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param body body dto.RemoveRoleFromDepartmentReq true "移除角色请求"
// @Success 200 {object} dto.RemoveRoleFromDepartmentResp "移除成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/role/remove/department [post]
func (r *Role) RemoveRoleFromDepartment(c *gin.Context) {
	var req dto.RemoveRoleFromDepartmentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	if err := r.permissionService.RemoveRoleFromDepartment(ctx, &req); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithMessage(c, "移除角色成功")
}

// GetUserRoles 获取用户角色
// @Summary 获取用户角色
// @Description 获取指定用户的所有角色分配
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param body body dto.GetUserRolesReq true "获取用户角色请求"
// @Success 200 {object} dto.GetUserRolesResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/role/user [post]
func (r *Role) GetUserRoles(c *gin.Context) {
	var req dto.GetUserRolesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := r.permissionService.GetUserRoles(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// GetDepartmentRoles 获取组织架构角色
// @Summary 获取组织架构角色
// @Description 获取指定组织架构的所有角色分配
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param body body dto.GetDepartmentRolesReq true "获取组织架构角色请求"
// @Success 200 {object} dto.GetDepartmentRolesResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/role/department [post]
func (r *Role) GetDepartmentRoles(c *gin.Context) {
	var req dto.GetDepartmentRolesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := r.permissionService.GetDepartmentRoles(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// GetRolesForPermissionRequest 获取可用于权限申请的角色列表（根据节点类型过滤）
// @Summary 获取可用于权限申请的角色列表
// @Description 根据节点类型和模板类型，返回包含该资源类型权限的角色列表
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param node_type query string true "节点类型（package 或 function）"
// @Param template_type query string false "模板类型（table、form、chart，仅对 function 有效）"
// @Success 200 {object} dto.GetRolesForPermissionRequestResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/role/for_request [get]
func (r *Role) GetRolesForPermissionRequest(c *gin.Context) {
	var req dto.GetRolesForPermissionRequestReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 参数验证
	if req.NodeType == "" {
		response.FailWithMessage(c, "node_type 参数不能为空")
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := r.permissionService.GetRolesForPermissionRequest(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}
