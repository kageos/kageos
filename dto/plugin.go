package dto

// PluginListReq 获取插件列表请求
type PluginListReq struct {
	Enabled  *bool  `json:"enabled" form:"enabled"` // true, false
	Scope    string `json:"scope" form:"scope"`    // mine: 我的, market: 市场
	Page     int    `json:"page" form:"page" binding:"required" example:"1"`
	PageSize int    `json:"page_size" form:"page_size" binding:"required" example:"10"`
}

// PluginInfo 插件信息
type PluginInfo struct {
	ID          int64   `json:"id" example:"1"`
	Name        string  `json:"name" example:"Excel解析插件"`
	Code        string  `json:"code" example:"excel_parser"`
	Description string  `json:"description" example:"解析Excel文件为Markdown表格"`
	Enabled     bool    `json:"enabled" example:"true"`
	Subject     string  `json:"subject" example:"plugins.beiluo.1"` // NATS主题，自动生成
	NatsHost    string  `json:"nats_host" example:"nats://127.0.0.1:4223"` // NATS 服务器地址
	Config      *string `json:"config" example:"{\"timeout\": 30, \"max_file_size\": 10485760}"` // 插件配置（JSON）
	User        string  `json:"user" example:"beiluo"` // 创建用户（保留用于向后兼容）
	Visibility  int     `json:"visibility" example:"0"` // 0: 公开, 1: 私有
	Admin       string  `json:"admin" example:"user1,user2"` // 管理员列表（逗号分隔）
	IsAdmin     bool    `json:"is_admin" example:"true"` // 当前用户是否是管理员
	CreatedAt   string  `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt   string  `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// PluginListResp 获取插件列表响应
type PluginListResp struct {
	Plugins  []PluginInfo `json:"plugins"`
	Total    int64        `json:"total" example:"100"`
	Page     int          `json:"page" example:"1"`
	PageSize int          `json:"page_size" example:"10"`
}

// CreatePluginReq 创建插件请求
type CreatePluginReq struct {
	Name        string  `json:"name" binding:"required" example:"Excel解析插件"`
	Code        string  `json:"code" binding:"required" example:"excel_parser"`
	Description string  `json:"description" example:"解析Excel文件为Markdown表格"`
	Enabled     bool    `json:"enabled" example:"true"`
	Config      *string `json:"config" example:"{\"timeout\": 30, \"max_file_size\": 10485760}"` // 插件配置（JSON）
	Visibility  int     `json:"visibility" example:"0"` // 0: 公开, 1: 私有（默认0）
	Admin       string  `json:"admin" example:"user1,user2"` // 管理员列表（逗号分隔，默认创建用户）
}

// UpdatePluginReq 更新插件请求
type UpdatePluginReq struct {
	Name        string  `json:"name" binding:"required" example:"Excel解析插件"`
	Code        string  `json:"code" binding:"required" example:"excel_parser"`
	Description string  `json:"description" example:"解析Excel文件为Markdown表格"`
	Enabled     bool    `json:"enabled" example:"true"`
	Config      *string `json:"config" example:"{\"timeout\": 30, \"max_file_size\": 10485760}"` // 插件配置（JSON）
	Visibility  int     `json:"visibility" example:"0"` // 0: 公开, 1: 私有
	Admin       string  `json:"admin" example:"user1,user2"` // 管理员列表（逗号分隔）
}

