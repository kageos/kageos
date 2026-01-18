package dto

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// ============================================
// 角色管理 DTO
// ============================================

// CreateRoleReq 创建角色请求
type CreateRoleReq struct {
	Name        string              `json:"name" binding:"required" example:"开发者"`
	Code        string              `json:"code" binding:"required" example:"developer"`
	Description string              `json:"description" example:"开发人员角色"`
	Permissions map[string][]string `json:"permissions" binding:"required" example:"{\"directory\":[\"directory:read\",\"directory:write\"],\"table\":[\"function:read\",\"function:write\"]}"` // resourceType -> []action
}

// CreateRoleResp 创建角色响应
type CreateRoleResp struct {
	Role *model.Role `json:"role"`
}

// UpdateRoleReq 更新角色请求
type UpdateRoleReq struct {
	Name        *string              `json:"name,omitempty" example:"开发者"`
	Description *string              `json:"description,omitempty" example:"开发人员角色"`
	IsDefault   *bool                `json:"is_default,omitempty" example:"true"` // 是否设置为默认角色
	Permissions *map[string][]string `json:"permissions,omitempty" example:"{\"directory\":[\"directory:read\"],\"table\":[\"function:read\"]}"` // resourceType -> []action
}

// UpdateRoleResp 更新角色响应
type UpdateRoleResp struct {
	Role *model.Role `json:"role"`
}

// GetRoleResp 获取角色响应
type GetRoleResp struct {
	Role *model.Role `json:"role"`
}

// GetRolesResp 获取角色列表响应
type GetRolesResp struct {
	Roles []*model.Role `json:"roles"`
}

// DeleteRoleResp 删除角色响应
type DeleteRoleResp struct {
	Message string `json:"message"`
}

// ============================================
// 角色分配 DTO
// ============================================

// AssignRoleToUserReq 给用户分配角色请求
type AssignRoleToUserReq struct {
	User         string       `json:"user" binding:"required" example:"luobei"`
	App          string       `json:"app" binding:"required" example:"operations"`
	Username     string       `json:"username" binding:"required" example:"zhangsan"`
	RoleCode     string       `json:"role_code" binding:"required" example:"developer"`
	ResourceType string       `json:"resource_type" binding:"required" example:"table"` // ⭐ 资源类型：directory、table、form、chart、app
	ResourcePath string       `json:"resource_path" binding:"required" example:"/luobei/operations/tools/*"`
	StartTime    *models.Time `json:"start_time,omitempty" example:"2025-01-27T00:00:00Z"`
	EndTime      *models.Time `json:"end_time,omitempty" example:"2025-12-31T23:59:59Z"`
}

// AssignRoleToUserResp 给用户分配角色响应
type AssignRoleToUserResp struct {
	Assignment *model.RoleAssignment `json:"assignment"`
}

// AssignRoleToDepartmentReq 给组织架构分配角色请求
type AssignRoleToDepartmentReq struct {
	User           string       `json:"user" binding:"required" example:"luobei"`
	App            string       `json:"app" binding:"required" example:"operations"`
	DepartmentPath string       `json:"department_path" binding:"required" example:"/org/master/bizit"`
	RoleCode       string       `json:"role_code" binding:"required" example:"viewer"`
	ResourceType   string       `json:"resource_type" binding:"required" example:"table"` // ⭐ 资源类型：directory、table、form、chart、app
	ResourcePath   string       `json:"resource_path" binding:"required" example:"/luobei/operations/*"`
	StartTime      *models.Time `json:"start_time,omitempty"`
	EndTime        *models.Time `json:"end_time,omitempty"`
}

// AssignRoleToDepartmentResp 给组织架构分配角色响应
type AssignRoleToDepartmentResp struct {
	Assignment *model.RoleAssignment `json:"assignment"`
}

// RemoveRoleFromUserReq 移除用户角色请求
type RemoveRoleFromUserReq struct {
	User         string `json:"user" binding:"required"`
	App          string `json:"app" binding:"required"`
	Username     string `json:"username" binding:"required"`
	RoleCode     string `json:"role_code" binding:"required"`
	ResourceType string `json:"resource_type" binding:"required"` // ⭐ 资源类型：directory、table、form、chart、app
	ResourcePath string `json:"resource_path" binding:"required"`
}

// RemoveRoleFromUserResp 移除用户角色响应
type RemoveRoleFromUserResp struct {
	Message string `json:"message"`
}

// RemoveRoleFromDepartmentReq 移除组织架构角色请求
type RemoveRoleFromDepartmentReq struct {
	User           string `json:"user" binding:"required"`
	App            string `json:"app" binding:"required"`
	DepartmentPath string `json:"department_path" binding:"required"`
	RoleCode       string `json:"role_code" binding:"required"`
	ResourceType   string `json:"resource_type" binding:"required"` // ⭐ 资源类型：directory、table、form、chart、app
	ResourcePath   string `json:"resource_path" binding:"required"`
}

// RemoveRoleFromDepartmentResp 移除组织架构角色响应
type RemoveRoleFromDepartmentResp struct {
	Message string `json:"message"`
}

// GetUserRolesReq 获取用户角色请求
type GetUserRolesReq struct {
	User     string `json:"user" binding:"required"`
	App      string `json:"app" binding:"required"`
	Username string `json:"username" binding:"required"`
}

// GetUserRolesResp 获取用户角色响应
type GetUserRolesResp struct {
	Assignments []*model.RoleAssignment `json:"assignments"`
}

// GetDepartmentRolesReq 获取组织架构角色请求
type GetDepartmentRolesReq struct {
	User           string `json:"user" binding:"required"`
	App            string `json:"app" binding:"required"`
	DepartmentPath string `json:"department_path" binding:"required"`
}

// GetDepartmentRolesResp 获取组织架构角色响应
type GetDepartmentRolesResp struct {
	Assignments []*model.RoleAssignment `json:"assignments"`
}

// GetRolesForPermissionRequestReq 获取可用于权限申请的角色列表请求
type GetRolesForPermissionRequestReq struct {
	NodeType     string `form:"node_type" json:"node_type" example:"function"`       // 节点类型：package 或 function
	TemplateType string `form:"template_type" json:"template_type" example:"table"` // 模板类型：table、form、chart（仅对 function 有效）
}

// GetRolesForPermissionRequestResp 获取可用于权限申请的角色列表响应
type GetRolesForPermissionRequestResp struct {
	Roles []*model.Role `json:"roles"` // 角色列表（只包含对该资源类型有权限的角色）
}

// ============================================
// 资源权限管理 DTO
// ============================================

// GetResourcePermissionsReq 查询资源权限请求
type GetResourcePermissionsReq struct {
	User         string `json:"user" form:"user" binding:"required" example:"luobei"`         // 租户用户
	App          string `json:"app" form:"app" binding:"required" example:"operations"`       // 应用代码
	ResourcePath string `json:"resource_path" form:"resource_path" binding:"required" example:"/luobei/operations/tools/crm_ticket"` // 资源路径（full-code-path）
}

// ResourcePermissionAssignment 资源权限分配信息
type ResourcePermissionAssignment struct {
	ID           int64     `json:"id" example:"1"`              // 分配ID
	SubjectType  string    `json:"subject_type" example:"user"` // 权限主体类型：user 或 department
	Subject      string    `json:"subject" example:"zhangsan"`   // 权限主体：用户名或组织架构路径
	SubjectName  string    `json:"subject_name" example:"张三"`  // 权限主体名称（用户昵称或部门名称）
	RoleID       int64     `json:"role_id" example:"1"`         // 角色ID
	RoleCode     string    `json:"role_code" example:"developer"` // 角色编码
	RoleName     string    `json:"role_name" example:"开发者"`    // 角色名称
	ResourcePath string       `json:"resource_path" example:"/luobei/operations/tools/crm_ticket"` // 资源路径
	ResourceName string       `json:"resource_name" example:"工单管理"` // 资源名称（节点名称）
	StartTime    models.Time  `json:"start_time" example:"2025-01-27T00:00:00Z"` // 生效开始时间
	EndTime      *models.Time `json:"end_time,omitempty" example:"2025-12-31T23:59:59Z"` // 生效结束时间（nil 表示永久）
	CreatedBy    string       `json:"created_by,omitempty" example:"admin"` // 创建者
	CreatedAt    models.Time  `json:"created_at" example:"2025-01-27T00:00:00Z"` // 创建时间
}

// GetResourcePermissionsResp 查询资源权限响应
type GetResourcePermissionsResp struct {
	Assignments []*ResourcePermissionAssignment `json:"assignments"` // 权限分配列表
	Total       int                              `json:"total" example:"10"` // 总数
}
