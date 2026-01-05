package dto

// SendEmailCodeReq 发送邮箱验证码请求
type SendEmailCodeReq struct {
	Email string `json:"email" binding:"required,email" example:"user@example.com"` // 邮箱地址
}

// SendEmailCodeResp 发送邮箱验证码响应
type SendEmailCodeResp struct {
	// 使用统一的 response.Response 格式，无需额外字段
}

// RegisterReq 用户注册请求
type RegisterReq struct {
	Username string `json:"username" binding:"required,min=3,max=20" example:"beiluo"`   // 用户名
	Email    string `json:"email" binding:"required,email" example:"beiluo@example.com"` // 邮箱
	Password string `json:"password" binding:"required,min=6" example:"123456"`          // 密码
	Code     string `json:"code" binding:"required,len=6" example:"123456"`              // 验证码
}

// RegisterResp 用户注册响应
type RegisterResp struct {
	UserID int64 `json:"user_id" example:"1"` // 用户ID
}

// LoginReq 用户登录请求
type LoginReq struct {
	Username string `json:"username" binding:"required" example:"beiluo"` // 用户名
	Password string `json:"password" binding:"required" example:"123456"` // 密码
	Remember bool   `json:"remember" example:"false"`                     // 记住我（延长Refresh Token有效期）
}

// LoginResp 用户登录响应
type LoginResp struct {
	Token        string   `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`                 // JWT Token
	RefreshToken string   `json:"refresh_token" example:"refresh_eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."` // 刷新Token
	User         UserInfo `json:"user"`                                                                    // 用户信息
}

// UserInfo 用户信息
type UserInfo struct {
	ID                int64  `json:"id" example:"1"`                            // 用户ID
	Username          string `json:"username" example:"beiluo"`                 // 用户名
	Email             string `json:"email" example:"beiluo@example.com"`        // 邮箱
	RegisterType      string `json:"register_type" example:"email"`             // 注册方式
	Avatar            string `json:"avatar" example:"https://avatar.com/1.jpg"` // 头像
	Nickname          string `json:"nickname" example:"北落"`                   // 昵称
	Signature         string `json:"signature" example:"这个人很懒，什么都没有留下"` // 个人签名/简介
	Gender            string `json:"gender" example:"male"`                     // 性别: male(男), female(女), other(其他), 空字符串表示未设置
	EmailVerified     bool   `json:"email_verified" example:"true"`             // 邮箱是否已验证
	Status            string `json:"status" example:"active"`                   // 用户状态: pending(待邮箱验证), active(已激活)
	CreatedAt         string `json:"created_at" example:"2024-01-01T00:00:00Z"` // 创建时间
	DepartmentFullPath     string `json:"department_full_path,omitempty" example:"/tech/backend"`     // 部门完整路径（可选）
	DepartmentName         string `json:"department_name,omitempty" example:"后端组"`                    // 部门名称（可选，用于显示）
	DepartmentFullNamePath string `json:"department_full_name_path,omitempty" example:"技术部/后端组"`   // 部门完整名称路径（可选，用于展示组织架构全称）
	LeaderUsername         string `json:"leader_username,omitempty" example:"lisi"`                  // Leader 用户名（可选）
	LeaderDisplayName  string `json:"leader_display_name,omitempty" example:"lisi(李四)"`       // Leader 显示名称（可选，用于显示）
}

// RefreshTokenReq 刷新Token请求
type RefreshTokenReq struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"refresh_eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."` // 刷新Token
}

// RefreshTokenResp 刷新Token响应
type RefreshTokenResp struct {
	Token        string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`                 // 新的JWT Token
	RefreshToken string `json:"refresh_token" example:"refresh_eyJhbGciOiJIUzI1NiIsInR0cCI6IkpXVCJ9..."` // 新的Refresh Token
}

// LogoutReq 用户登出请求
type LogoutReq struct {
	Token string `json:"token" binding:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."` // JWT Token
}

// LogoutResp 用户登出响应
type LogoutResp struct {
	// 使用统一的 response.Response 格式，无需额外字段
}
