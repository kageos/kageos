package dto

// AddPermissionReq 添加权限请求
type AddPermissionReq struct {
	Username     string `json:"username" binding:"required"`     // 用户名
	ResourcePath string `json:"resource_path" binding:"required"` // 资源路径（full-code-path）
	Action       string `json:"action" binding:"required"`     // 操作类型（如 table:search、function:manage 等）
}

// ApplyPermissionReq 权限申请请求
type ApplyPermissionReq struct {
	ResourcePath string   `json:"resource_path" binding:"required"` // 资源路径（full-code-path）
	Action        string   `json:"action"`                          // 操作类型（可选，如果提供了 actions 则忽略）
	Actions       []string `json:"actions"`                         // 操作类型列表（可选，如果提供则批量申请）
	Reason        string   `json:"reason"`                          // 申请理由（可选）
}

// ApplyPermissionResp 权限申请响应
type ApplyPermissionResp struct {
	ID      string `json:"id"`      // 申请ID（暂时返回空字符串，后续可以扩展为申请记录ID）
	Status  string `json:"status"`  // 申请状态（approved：已批准，pending：待审核）
	Message string `json:"message"` // 响应消息
}

// GetWorkspacePermissionsReq 获取工作空间权限请求
type GetWorkspacePermissionsReq struct {
	AppID int64 `json:"app_id" form:"app_id"` // 应用ID（必填）
}

// PermissionRecord 权限记录
type PermissionRecord struct {
	ID       int64  `json:"id"`        // 权限记录ID
	User     string `json:"user"`      // 用户名
	Resource string `json:"resource"`  // 资源路径
	Action   string `json:"action"`    // 操作类型
	AppID    int64  `json:"app_id"`    // 应用ID
}

// GetWorkspacePermissionsResp 获取工作空间权限响应
// ⭐ 直接返回原始权限记录，让前端处理
type GetWorkspacePermissionsResp struct {
	Records []PermissionRecord `json:"records"` // 原始权限记录
}

