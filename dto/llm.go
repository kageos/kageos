package dto

// LLMListReq 获取LLM配置列表请求
type LLMListReq struct {
	Page     int `json:"page" form:"page" binding:"required" example:"1"`
	PageSize int `json:"page_size" form:"page_size" binding:"required" example:"10"`
}

// LLMInfo LLM配置信息
type LLMInfo struct {
	ID          int64  `json:"id" example:"1"`
	Name        string `json:"name" example:"OpenAI GPT-4"`
	Provider    string `json:"provider" example:"openai"` // openai, claude, local, etc.
	Model       string `json:"model" example:"gpt-4"`
	APIBase     string `json:"api_base" example:"https://api.openai.com/v1"`
	Timeout     int    `json:"timeout" example:"120"`
	MaxTokens   int    `json:"max_tokens" example:"4000"`
	ExtraConfig string `json:"extra_config" example:"{}"`
	IsDefault   bool   `json:"is_default" example:"true"`
	CreatedAt   string `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt   string `json:"updated_at" example:"2024-01-01T00:00:00Z"`
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
	Name        string  `json:"name" binding:"required" example:"OpenAI GPT-4"`
	Provider    string  `json:"provider" binding:"required" example:"openai"`
	Model       string  `json:"model" binding:"required" example:"gpt-4"`
	APIKey      string  `json:"api_key" example:"sk-xxx"`
	APIBase     string  `json:"api_base" example:"https://api.openai.com/v1"`
	Timeout     int     `json:"timeout" example:"120"`
	MaxTokens   int     `json:"max_tokens" example:"4000"`
	ExtraConfig *string `json:"extra_config" example:"{}"`
	IsDefault   bool    `json:"is_default" example:"false"`
}

// LLMCreateResp 创建LLM配置响应
type LLMCreateResp struct {
	ID int64 `json:"id" example:"1"`
}

// LLMUpdateReq 更新LLM配置请求
type LLMUpdateReq struct {
	ID          int64  `json:"id" binding:"required" example:"1"`
	Name        string `json:"name" binding:"required" example:"OpenAI GPT-4"`
	Provider    string `json:"provider" binding:"required" example:"openai"`
	Model       string `json:"model" binding:"required" example:"gpt-4"`
	APIKey      string `json:"api_key" example:"sk-xxx"`
	APIBase     string `json:"api_base" example:"https://api.openai.com/v1"`
	Timeout     int    `json:"timeout" example:"120"`
	MaxTokens   int    `json:"max_tokens" example:"4000"`
	ExtraConfig string `json:"extra_config" example:"{}"`
	IsDefault   bool   `json:"is_default" example:"false"`
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
