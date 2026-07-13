package dto

// LLMListReq 获取LLM配置列表请求
type LLMListReq struct {
	Scope    string `json:"scope" form:"scope"` // mine: 我的, market: 市场
	Page     int    `json:"page" form:"page" binding:"required" example:"1"`
	PageSize int    `json:"page_size" form:"page_size" binding:"required" example:"10"`
}

// LLMInfo LLM配置信息
type LLMInfo struct {
	ID           int64  `json:"id" example:"1"`
	Code         string `json:"code,omitempty" example:"default"`
	Name         string `json:"name" example:"OpenAI GPT-4"`
	Provider     string `json:"provider" example:"openai"`
	Protocol     string `json:"protocol" example:"openai_chat_completions"`
	Model        string `json:"model" example:"gpt-4"`
	APIKey       string `json:"api_key,omitempty" example:"sk-xxx"`
	HasAPIKey    bool   `json:"has_api_key" example:"true"`
	APIBase      string `json:"api_base" example:"https://api.openai.com/v1"`
	EndpointPath string `json:"endpoint_path" example:"/v1/chat/completions"`
	APIVersion   string `json:"api_version" example:"2023-06-01"`
	AuthScheme   string `json:"auth_scheme" example:"bearer"`
	Headers      string `json:"headers" example:"{}"`
	Timeout      int    `json:"timeout" example:"300"`
	MaxTokens    int    `json:"max_tokens" example:"8196"`
	ExtraConfig  string `json:"extra_config" example:"{}"`
	Capabilities string `json:"capabilities" example:"{}"`
	IsDefault    bool   `json:"is_default" example:"true"`
	Visibility   int    `json:"visibility" example:"0"`      // 0: 公开, 1: 私有
	Admin        string `json:"admin" example:"user1,user2"` // 管理员列表（逗号分隔）
	IsAdmin      bool   `json:"is_admin" example:"true"`     // 当前用户是否是管理员
	CreatedAt    string `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt    string `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// LLMListResp 获取LLM配置列表响应
type LLMListResp struct {
	Configs []LLMInfo `json:"configs"`
	Total   int64     `json:"total" example:"100"`
}

// LLMGetReq 获取LLM配置详情请求
type LLMGetReq struct {
	ID int64 `json:"id" form:"id" binding:"required" example:"1"`
}

// LLMGetResp 获取LLM配置详情响应
type LLMGetResp struct {
	LLMInfo
}

// LLMGetDefaultResp 获取默认LLM配置响应
type LLMGetDefaultResp struct {
	LLMInfo
}

// LLMCreateReq 创建LLM配置请求
type LLMCreateReq struct {
	Name         string  `json:"name" binding:"required" example:"OpenAI GPT-4"`
	Provider     string  `json:"provider" example:"openai"`
	Protocol     string  `json:"protocol" example:"openai_chat_completions"`
	Model        string  `json:"model" binding:"required" example:"gpt-4"`
	APIKey       string  `json:"api_key" example:"sk-xxx"`
	APIBase      string  `json:"api_base" example:"https://api.openai.com/v1"`
	EndpointPath string  `json:"endpoint_path" example:"/v1/chat/completions"`
	APIVersion   string  `json:"api_version" example:"2023-06-01"`
	AuthScheme   string  `json:"auth_scheme" example:"bearer"`
	Headers      *string `json:"headers" example:"{}"`
	Timeout      int     `json:"timeout" example:"300"`
	MaxTokens    int     `json:"max_tokens" example:"8196"`
	ExtraConfig  *string `json:"extra_config" example:"{}"`
	Capabilities *string `json:"capabilities" example:"{}"`
	IsDefault    bool    `json:"is_default" example:"false"`
	Visibility   int     `json:"visibility" example:"0"`      // 0: 公开, 1: 私有（默认0）
	Admin        string  `json:"admin" example:"user1,user2"` // 管理员列表（逗号分隔，默认创建用户）
}

// LLMCreateResp 创建LLM配置响应
type LLMCreateResp struct {
	ID int64 `json:"id" example:"1"`
}

// LLMUpdateReq 更新LLM配置请求
type LLMUpdateReq struct {
	ID           int64  `json:"-" example:"1"`
	Name         string `json:"name" binding:"required" example:"OpenAI GPT-4"`
	Provider     string `json:"provider" example:"openai"`
	Protocol     string `json:"protocol" example:"openai_chat_completions"`
	Model        string `json:"model" binding:"required" example:"gpt-4"`
	APIKey       string `json:"api_key" example:"sk-xxx"`
	APIBase      string `json:"api_base" example:"https://api.openai.com/v1"`
	EndpointPath string `json:"endpoint_path" example:"/v1/chat/completions"`
	APIVersion   string `json:"api_version" example:"2023-06-01"`
	AuthScheme   string `json:"auth_scheme" example:"bearer"`
	Headers      string `json:"headers" example:"{}"`
	Timeout      int    `json:"timeout" example:"300"`
	MaxTokens    int    `json:"max_tokens" example:"8196"`
	ExtraConfig  string `json:"extra_config" example:"{}"`
	Capabilities string `json:"capabilities" example:"{}"`
	IsDefault    bool   `json:"is_default" example:"false"`
	Visibility   int    `json:"visibility" example:"0"`      // 0: 公开, 1: 私有
	Admin        string `json:"admin" example:"user1,user2"` // 管理员列表（逗号分隔）
}

// LLMUpdateResp 更新LLM配置响应
type LLMUpdateResp struct {
	ID int64 `json:"id" example:"1"`
}

// LLMDeleteReq 删除LLM配置请求
type LLMDeleteReq struct {
	ID int64 `json:"id" form:"id" binding:"required" example:"1"`
}

// LLMSetDefaultReq 设置默认LLM配置请求
type LLMSetDefaultReq struct {
	ID int64 `json:"id" form:"id" binding:"required" example:"1"`
}

// LLMProbeReq 探测 LLM 协议和密钥是否可用。仅在用户主动点击检测时调用。
type LLMProbeReq struct {
	ID           int64  `json:"id" example:"1"`
	Provider     string `json:"provider" example:"openai"`
	Protocol     string `json:"protocol" example:"openai_responses"`
	Model        string `json:"model" example:"gpt-4.1-mini"`
	APIKey       string `json:"api_key" example:"sk-xxx"`
	APIBase      string `json:"api_base" example:"https://api.openai.com/v1"`
	EndpointPath string `json:"endpoint_path" example:"/v1/responses"`
	APIVersion   string `json:"api_version" example:"2023-06-01"`
	AuthScheme   string `json:"auth_scheme" example:"bearer"`
	Headers      string `json:"headers" example:"{}"`
	ExtraConfig  string `json:"extra_config" example:"{}"`
	MaxTokens    int    `json:"max_tokens" example:"64"`
	Timeout      int    `json:"timeout" example:"30"`
}

type LLMProbeAttempt struct {
	Provider string `json:"provider"`
	Protocol string `json:"protocol"`
	APIBase  string `json:"api_base"`
	OK       bool   `json:"ok"`
	Error    string `json:"error,omitempty"`
}

type LLMProbeResp struct {
	OK           bool              `json:"ok"`
	Provider     string            `json:"provider"`
	Protocol     string            `json:"protocol"`
	APIBase      string            `json:"api_base"`
	EndpointPath string            `json:"endpoint_path,omitempty"`
	APIVersion   string            `json:"api_version,omitempty"`
	AuthScheme   string            `json:"auth_scheme,omitempty"`
	Model        string            `json:"model,omitempty"`
	Message      string            `json:"message,omitempty"`
	Error        string            `json:"error,omitempty"`
	Capabilities map[string]bool   `json:"capabilities,omitempty"`
	Attempts     []LLMProbeAttempt `json:"attempts,omitempty"`
}
