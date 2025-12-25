package dto

// AddPermissionReq 添加权限请求
type AddPermissionReq struct {
	Username     string `json:"username" binding:"required"`     // 用户名
	ResourcePath string `json:"resource_path" binding:"required"` // 资源路径（full-code-path）
	Action       string `json:"action" binding:"required"`     // 操作类型（如 table:search、function:execute 等）
}

// RemovePermissionReq 删除权限请求
type RemovePermissionReq struct {
	Username     string `json:"username" binding:"required"`     // 用户名
	ResourcePath string `json:"resource_path" binding:"required"` // 资源路径（full-code-path）
	Action       string `json:"action" binding:"required"`     // 操作类型
}

// GetUserPermissionsReq 获取用户权限请求
type GetUserPermissionsReq struct {
	Username     string   `json:"username" form:"username" binding:"required"`     // 用户名
	ResourcePath string   `json:"resource_path" form:"resource_path"`             // 资源路径（可选，如果提供则只查询该资源的权限）
	Actions      []string `json:"actions" form:"actions"`                         // 操作类型列表（可选，如果提供则只查询这些操作的权限）
}

// GetUserPermissionsResp 获取用户权限响应
type GetUserPermissionsResp struct {
	Username     string              `json:"username"`      // 用户名
	ResourcePath string              `json:"resource_path"` // 资源路径（如果请求中提供了）
	Permissions  map[string]bool     `json:"permissions"`   // 权限结果（action -> hasPermission）
	AllResources map[string]map[string]bool `json:"all_resources,omitempty"` // 所有资源的权限（resourcePath -> action -> hasPermission），仅在查询所有资源时返回
}

// AddRoleReq 添加角色请求
type AddRoleReq struct {
	RoleName string `json:"role_name" binding:"required"` // 角色名称
}

// AssignRoleToUserReq 分配角色给用户请求
type AssignRoleToUserReq struct {
	Username string `json:"username" binding:"required"` // 用户名
	RoleName string `json:"role_name" binding:"required"` // 角色名称
}

// RemoveRoleFromUserReq 从用户移除角色请求
type RemoveRoleFromUserReq struct {
	Username string `json:"username" binding:"required"` // 用户名
	RoleName string `json:"role_name" binding:"required"` // 角色名称
}

// GetUserRolesResp 获取用户角色响应
type GetUserRolesResp struct {
	Username string   `json:"username"` // 用户名
	Roles    []string `json:"roles"`   // 角色列表
}

