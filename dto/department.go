package dto

type DepartmentInfo struct {
	ID              int64             `json:"id"`
	Name            string            `json:"name"`
	Code            string            `json:"code"`
	ParentID        *int64            `json:"parent_id"`
	FullCodePath    string            `json:"full_code_path"`
	FullNamePath    string            `json:"full_name_path"`
	Managers        string            `json:"managers"`
	Description     string            `json:"description"`
	Status          string            `json:"status"`
	Sort            int               `json:"sort"`
	IsSystemDefault bool              `json:"is_system_default"`
	Parent          *DepartmentInfo   `json:"parent,omitempty"`
	Children        []*DepartmentInfo `json:"children,omitempty"`
}

// CreateDepartmentReq 创建部门请求
type CreateDepartmentReq struct {
	Name        string `json:"name" binding:"required" example:"技术部"`      // 部门名称
	Code        string `json:"code" binding:"required" example:"tech"`     // 部门编码
	ParentID    int64  `json:"parent_id" example:"0"`                      // 父部门ID（0表示根部门）
	Description string `json:"description" example:"负责技术研发"`               // 部门描述
	Managers    string `json:"managers,omitempty" example:"zhangsan,lisi"` // 部门负责人（可选，多个用户名逗号分隔）
}

// CreateDepartmentResp 创建部门响应
type CreateDepartmentResp struct {
	Department *DepartmentInfo `json:"department"` // 部门信息
}

// UpdateDepartmentReq 更新部门请求
type UpdateDepartmentReq struct {
	Name        *string `json:"name,omitempty" example:"技术部"`               // 部门名称（可选）
	Description *string `json:"description,omitempty" example:"负责技术研发"`     // 部门描述（可选）
	Managers    *string `json:"managers,omitempty" example:"zhangsan,lisi"` // 部门负责人（可选，多个用户名逗号分隔）
	Status      *string `json:"status,omitempty" example:"active"`          // 状态（可选）：active(激活), inactive(停用)
	Sort        *int    `json:"sort,omitempty" example:"0"`                 // 排序（可选）
}

// UpdateDepartmentResp 更新部门响应
type UpdateDepartmentResp struct {
	Department *DepartmentInfo `json:"department"` // 部门信息
}

// GetDepartmentTreeResp 获取部门树响应
type GetDepartmentTreeResp struct {
	Departments []*DepartmentInfo `json:"departments"` // 部门树
}

// GetDepartmentResp 获取部门响应
type GetDepartmentResp struct {
	Department *DepartmentInfo `json:"department"` // 部门信息
}

// AssignUserReq 分配用户组织架构请求
type AssignUserReq struct {
	Username           string  `json:"username" binding:"required" example:"zhangsan"`         // 用户名
	DepartmentFullPath *string `json:"department_full_path,omitempty" example:"/tech/backend"` // 部门完整路径（可选，为空表示移除部门）
	LeaderUsername     *string `json:"leader_username,omitempty" example:"lisi"`               // Leader 用户名（可选，为空表示移除 Leader）
}

// AssignUserResp 分配用户组织架构响应
type AssignUserResp struct {
	User UserInfo `json:"user"` // 用户信息
}

// GetUsersByDepartmentReq 根据部门获取用户请求
type GetUsersByDepartmentReq struct {
	DepartmentFullPath string `json:"department_full_path" form:"department_full_path" binding:"required" example:"/tech/backend"` // 部门完整路径
}

// GetUsersByDepartmentResp 根据部门获取用户响应
type GetUsersByDepartmentResp struct {
	Users []UserInfo `json:"users"` // 用户列表
}

// GetDepartmentsByPathsReq 批量获取部门请求（GET 请求，使用 query 参数）
type GetDepartmentsByPathsReq struct {
	FullCodePaths string `form:"full_code_paths" binding:"required" example:"/tech/backend,/tech/frontend"` // 部门完整路径列表（逗号分隔）
}

// GetDepartmentsByPathsResp 批量获取部门响应
type GetDepartmentsByPathsResp struct {
	Departments []*DepartmentInfo `json:"departments"` // 部门列表
}
