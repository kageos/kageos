package dto

import "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/model"

type NamespaceCreateReq struct {
	Name string `json:"name" binding:"required" example:"my-namespace"` // 命名空间名称
}
type NamespaceCreateResp struct {
	Success bool   `json:"success" example:"true"`     // 是否成功
	Message string `json:"message" example:"命名空间创建成功"` // 响应消息
}

type CreateAppReq struct {
	User string `json:"user" swaggerignore:"true"`                // 租户用户名（应用所有者，决定应用的所有权）- 内部字段，不显示在文档中
	Code string `json:"code" binding:"required" example:"myapp"`  // 应用名
	Name string `json:"name" binding:"required" example:"腾讯oa系统"` // 应用名
}

type CreateAppResp struct {
	User   string `json:"user" example:"beiluo"`                    // 用户名
	App    string `json:"app" example:"myapp"`                      // 应用名
	AppDir string `json:"app_dir" example:"namespace/beiluo/myapp"` // 应用目录
}

// RequestAppReq 请求应用
type RequestAppReq struct {
	TraceId     string `json:"trace_id" example:"req-123456"` // 追踪ID（由中间件自动填充）
	IsCallback  bool   `json:"is_callback" example:"true"`
	RequestUser string `json:"request_user" swaggerignore:"true"`          // 请求用户（由中间件自动填充）
	Token       string `json:"token" swaggerignore:"true"`                 // 认证 Token（由中间件自动填充，透传到 SDK）
	User        string `json:"user" binding:"required" example:"beiluo"`   // 租户用户名（应用所有者）
	App         string `json:"app" binding:"required" example:"myapp"`     // 应用名
	Version     string `json:"version" binding:"required" example:"v1"`    // 版本号
	Router      string `json:"router" binding:"required" example:"/users"` // 路由路径
	Method      string `json:"method" example:"GET"`                       // 应用内部方法名（可选）
	Body        []byte `json:"body" example:"eyJpZCI6MX0="`                // 请求体（Base64编码）
	UrlQuery    string `json:"url_query" example:"page=1&size=10"`         // URL 查询参数
}

// CallbackAppReq 回调请求
type CallbackAppReq struct {
	Type   string      `json:"type" binding:"required" example:""`
	Router string      `json:"router" binding:"required" example:"/users/app/xxxx"` // 路由路径
	Body   interface{} `json:"body" example:"eyJpZCI6MX0="`                         // 请求体（Base64编码）
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
	User string `json:"user" swaggerignore:"true"`              // 用户名
	App  string `json:"app" binding:"required" example:"myapp"` // 应用名
}

// UpdateAppResp 更新应用响应
type UpdateAppResp struct {
	User       string          `json:"user" example:"beiluo"`    // 用户名
	App        string          `json:"app" example:"myapp"`      // 应用名
	OldVersion string          `json:"old_version" example:"v1"` // 旧版本号
	NewVersion string          `json:"new_version" example:"v2"` // 新版本号
	Diff       *model.DiffData `json:"diff,omitempty"`           // API diff 信息
	Error      string          `json:"error,omitempty"`          // 回调过程中的错误信息
}

// DeleteAppReq 删除应用请求
type DeleteAppReq struct {
	User string `json:"user" binding:"required" example:"beiluo"` // 租户名
	App  string `json:"app" binding:"required" example:"myapp"`   // 应用名
}

// DeleteAppResp 删除应用响应
type DeleteAppResp struct {
	User string `json:"user" example:"beiluo"` // 租户名
	App  string `json:"app" example:"myapp"`   // 应用名
}

// GetAppsReq 获取应用列表请求
type GetAppsReq struct {
	PageInfoReq
	User string `json:"user" swaggerignore:"true"` // 租户名（从JWT Token获取）
}

// GetAppsResp 获取应用列表响应
type GetAppsResp struct {
	PageInfoResp
}

// AppInfo 应用信息
type AppInfo struct {
	ID        int64  `json:"id" example:"1"`                           // 应用ID
	User      string `json:"user" example:"beiluo"`                    // 租户名
	Code      string `json:"code" example:"myapp"`                     // 应用代码
	Name      string `json:"name" example:"我的应用"`                      // 应用名称
	Status    string `json:"status" example:"enabled"`                 // 状态: enabled(启用), disabled(禁用)
	Version   string `json:"version" example:"v1"`                     // 版本
	NatsID    int64  `json:"nats_id" example:"1"`                      // NATS ID
	HostID    int64  `json:"host_id" example:"1"`                      // 主机ID
	CreatedAt string `json:"created_at" example:"2006-01-02 15:04:05"` // 创建时间
	UpdatedAt string `json:"updated_at" example:"2006-01-02 15:04:05"` // 更新时间
}
