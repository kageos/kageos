package dto

// UpdateWorkspaceReq 更新工作空间请求（只更新 MySQL 记录，不涉及容器更新）
// ⭐ user 和 app 从路径参数获取，不在请求体中
type UpdateWorkspaceReq struct {
	User   string `json:"-" swaggerignore:"true"`                       // 用户名（从路径参数获取，不在请求体中）
	App    string `json:"-" swaggerignore:"true"`                      // 应用名（从路径参数获取，不在请求体中）
	Admins string `json:"admins,omitempty" example:"user1,user2,user3"` // 管理员列表，逗号分隔的用户名
}

// UpdateWorkspaceResp 更新工作空间响应
type UpdateWorkspaceResp struct {
	User   string `json:"user" example:"beiluo"`   // 用户名
	App    string `json:"app" example:"myapp"`     // 应用名
	Admins string `json:"admins" example:"user1,user2,user3"` // 管理员列表
}
