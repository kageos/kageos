package dto

type NamespaceCreateReq struct {
	Name string `json:"name" binding:"required" example:"my-namespace"` // 命名空间名称
}
type NamespaceCreateResp struct {
	Success bool   `json:"success" example:"true"`     // 是否成功
	Message string `json:"message" example:"命名空间创建成功"` // 响应消息
}

type CreateAppReq struct {
	User string `json:"user" swaggerignore:"true"`              // 租户用户名（应用所有者，决定应用的所有权）- 内部字段，不显示在文档中
	App  string `json:"app" binding:"required" example:"myapp"` // 应用名
}

type CreateAppResp struct {
	User   string `json:"user" example:"beiluo"`                    // 用户名
	App    string `json:"app" example:"myapp"`                      // 应用名
	AppDir string `json:"app_dir" example:"namespace/beiluo/myapp"` // 应用目录
	Status string `json:"status" example:"created"`                 // 状态
}

// RequestAppReq 请求应用
type RequestAppReq struct {
	TraceId     string `json:"trace_id" example:"req-123456"`              // 追踪ID（由中间件自动填充）
	RequestUser string `json:"request_user" example:"beiluo"`              // 请求用户（由中间件自动填充）
	User        string `json:"user" binding:"required" example:"beiluo"`   // 应用所属用户
	App         string `json:"app" binding:"required" example:"myapp"`     // 应用名
	Version     string `json:"version" binding:"required" example:"v1"`    // 版本号
	Router      string `json:"router" binding:"required" example:"/users"` // 路由路径
	Method      string `json:"method" example:"GET"`                       // 应用内部方法名（可选）
	Body        []byte `json:"body" example:"eyJpZCI6MX0="`                // 请求体（Base64编码）
	UrlQuery    string `json:"url_query" example:"page=1&size=10"`         // URL 查询参数
}

// RequestAppResp 应用响应
type RequestAppResp struct {
	TraceId string      `json:"trace_id" example:"req-123456"` // 追踪ID
	Version string      `json:"version" example:"v1"`
	Result  interface{} `json:"result,omitempty"`                 // 结果
	Error   string      `json:"error,omitempty" example:"应用内部错误"` // 错误信息
}

// UpdateAppReq 更新应用请求
type UpdateAppReq struct {
	User string `json:"user" swaggerignore:"true" example:"beiluo"` // 用户名
	App  string `json:"app" binding:"required" example:"myapp"`     // 应用名
}

// UpdateAppResp 更新应用响应
type UpdateAppResp struct {
	User       string `json:"user" example:"beiluo"`    // 用户名
	App        string `json:"app" example:"myapp"`      // 应用名
	OldVersion string `json:"old_version" example:"v1"` // 旧版本号
	NewVersion string `json:"new_version" example:"v2"` // 新版本号
	Status     string `json:"status" example:"updated"` // 状态
}

// DeleteAppReq 删除应用请求
type DeleteAppReq struct {
	User string `json:"user" binding:"required" example:"beiluo"` // 用户名
	App  string `json:"app" binding:"required" example:"myapp"`   // 应用名
}

// DeleteAppResp 删除应用响应
type DeleteAppResp struct {
	User   string `json:"user" example:"beiluo"`    // 用户名
	App    string `json:"app" example:"myapp"`      // 应用名
	Status string `json:"status" example:"deleted"` // 状态
}
