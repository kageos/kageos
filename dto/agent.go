package dto

// AgentListReq 获取智能体列表请求
type AgentListReq struct {
	AgentType        string `json:"agent_type" form:"agent_type"`                    // knowledge_only, plugin
	Enabled          *bool  `json:"enabled" form:"enabled"`                          // true, false
	KnowledgeBaseID  *int64 `json:"knowledge_base_id" form:"knowledge_base_id"`      // 按知识库ID过滤（可选）
	LLMConfigID      *int64 `json:"llm_config_id" form:"llm_config_id"`              // 按LLM配置ID过滤（可选，0表示默认LLM）
	PluginFunctionPath string `json:"plugin_function_path" form:"plugin_function_path"`                     // 按插件函数路径过滤（可选）
	Scope            string `json:"scope" form:"scope"`                              // mine: 我的, market: 市场
	Page             int    `json:"page" form:"page" binding:"required" example:"1"`
	PageSize         int    `json:"page_size" form:"page_size" binding:"required" example:"10"`
}

// AgentInfo 智能体信息
type AgentInfo struct {
	ID              int64              `json:"id" example:"1"`
	Name            string             `json:"name" example:"Excel生成智能体"`
	AgentType       string             `json:"agent_type" example:"plugin"` // knowledge_only, plugin
	ChatType        string             `json:"chat_type" example:"function_gen"`
	Enabled         bool               `json:"enabled" example:"true"`
	Author          string             `json:"author" example:"beiluo"`
	Description     string             `json:"description" example:"基于Excel文件生成管理系统"`
	Timeout         int                `json:"timeout" example:"30"`
	PluginFunctionPath string             `json:"plugin_function_path" example:"/system/official/agent/plugin/excel_or_csv/table_parse"` // 插件函数路径（full-code-path，仅 plugin 类型需要）
	KnowledgeBaseID     int64              `json:"knowledge_base_id" example:"1"`
	KnowledgeBase       *KnowledgeBaseInfo `json:"knowledge_base,omitempty"`  // 预加载的知识库信息
	LLMConfigID         int64              `json:"llm_config_id" example:"1"` // LLM配置ID，如果为0则使用默认LLM
	LLMConfig           *LLMConfigInfo     `json:"llm_config,omitempty"`      // 预加载的LLM配置信息
	SystemPromptTemplate string            `json:"system_prompt_template" example:"你是一个专业的代码生成助手。以下是相关的知识库内容，请参考这些内容来生成代码：\n{knowledge}"` // System Prompt模板，支持{knowledge}变量
	Metadata            string             `json:"metadata" example:"{}"`
	Logo                string             `json:"logo,omitempty" example:"https://example.com/logo.png"` // 智能体 Logo URL（可选）
	Greeting            string             `json:"greeting,omitempty" example:"欢迎使用本智能体！"` // 开场白内容（可选）
	GreetingType        string             `json:"greeting_type,omitempty" example:"md"` // 开场白格式类型：text, md, html
	GenerationCount     int64              `json:"generation_count" example:"100"` // 生成次数统计
	Visibility          int                `json:"visibility" example:"0"` // 0: 公开, 1: 私有
	Admin               string             `json:"admin" example:"user1,user2"` // 管理员列表（逗号分隔）
	IsAdmin             bool               `json:"is_admin" example:"true"` // 当前用户是否是管理员（前端计算或后端返回）
	CreatedAt       string             `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt       string             `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// KnowledgeBaseInfo 知识库信息（用于预加载）
type KnowledgeBaseInfo struct {
	ID            int64  `json:"id" example:"1"`
	Name          string `json:"name" example:"Excel知识库"`
	Description   string `json:"description" example:"Excel相关文档"`
	Status        string `json:"status" example:"active"`
	DocumentCount int    `json:"document_count" example:"10"`
}

// LLMConfigInfo LLM配置信息（用于预加载）
type LLMConfigInfo struct {
	ID        int64  `json:"id" example:"1"`
	Name      string `json:"name" example:"OpenAI GPT-4"`
	Provider  string `json:"provider" example:"openai"`
	Model     string `json:"model" example:"gpt-4"`
	IsDefault bool   `json:"is_default" example:"true"`
}

// AgentListResp 获取智能体列表响应
type AgentListResp struct {
	Agents []AgentInfo `json:"agents"`
	Total  int64       `json:"total" example:"100"`
}

// AgentGetReq 获取智能体详情请求
type AgentGetReq struct {
	ID int64 `json:"id" form:"id" binding:"required" example:"1"`
}

// AgentGetResp 获取智能体详情响应
type AgentGetResp struct {
	AgentInfo
}

// AgentCreateReq 创建智能体请求
type AgentCreateReq struct {
	Name            string `json:"name" binding:"required" example:"Excel生成智能体"`
	AgentType       string `json:"agent_type" binding:"required" example:"plugin"` // knowledge_only, plugin
	ChatType        string `json:"chat_type" binding:"required" example:"chat-task"`
	Author          string `json:"author" example:"beiluo"`
	Description     string `json:"description" example:"基于Excel文件生成管理系统"`
	Timeout         int    `json:"timeout" example:"30"`
	PluginFunctionPath string `json:"plugin_function_path" example:"/system/official/agent/plugin/excel_or_csv/table_parse"` // 插件函数路径（full-code-path，仅 plugin 类型需要）
	KnowledgeBaseID     int64  `json:"knowledge_base_id" binding:"required" example:"1"`
	LLMConfigID         int64  `json:"llm_config_id" example:"1"` // LLM配置ID，如果为0则使用默认LLM
	SystemPromptTemplate string `json:"system_prompt_template" example:"你是一个专业的代码生成助手。以下是相关的知识库内容，请参考这些内容来生成代码：\n{knowledge}"` // System Prompt模板，支持{knowledge}变量
	Metadata            string `json:"metadata" example:"{}"`
	Logo                string `json:"logo" example:"https://example.com/logo.png"` // 智能体 Logo URL（可选）
	Greeting            string `json:"greeting" example:"欢迎使用本智能体！"` // 开场白内容（可选）
	GreetingType        string `json:"greeting_type" example:"md"` // 开场白格式类型：text, md, html（默认text）
	Visibility          int    `json:"visibility" example:"0"` // 0: 公开, 1: 私有（默认0）
	Admin               string `json:"admin" example:"user1,user2"` // 管理员列表（逗号分隔，默认创建用户）
}

// AgentCreateResp 创建智能体响应
type AgentCreateResp struct {
	ID int64 `json:"id" example:"1"`
}

// AgentUpdateReq 更新智能体请求
type AgentUpdateReq struct {
	ID              int64  `json:"id" binding:"required" example:"1"`
	Name            string `json:"name" binding:"required" example:"Excel生成智能体"`
	AgentType       string `json:"agent_type" binding:"required" example:"plugin"`
	ChatType        string `json:"chat_type" binding:"required" example:"function_gen"`
	Author          string `json:"author" example:"beiluo"`
	Description     string `json:"description" example:"基于Excel文件生成管理系统"`
	Timeout         int    `json:"timeout" example:"30"`
	PluginFunctionPath string `json:"plugin_function_path" example:"/system/official/agent/plugin/excel_or_csv/table_parse"` // 插件函数路径（full-code-path，仅 plugin 类型需要）
	KnowledgeBaseID     int64  `json:"knowledge_base_id" binding:"required" example:"1"`
	LLMConfigID         int64  `json:"llm_config_id" example:"1"` // LLM配置ID，如果为0则使用默认LLM
	SystemPromptTemplate string `json:"system_prompt_template" example:"你是一个专业的代码生成助手。以下是相关的知识库内容，请参考这些内容来生成代码：\n{knowledge}"` // System Prompt模板，支持{knowledge}变量
	Metadata            string `json:"metadata" example:"{}"`
	Logo                string `json:"logo" example:"https://example.com/logo.png"` // 智能体 Logo URL（可选）
	Greeting            string `json:"greeting" example:"欢迎使用本智能体！"` // 开场白内容（可选）
	GreetingType        string `json:"greeting_type" example:"md"` // 开场白格式类型：text, md, html（默认text）
	Visibility          int    `json:"visibility" example:"0"` // 0: 公开, 1: 私有
	Admin               string `json:"admin" example:"user1,user2"` // 管理员列表（逗号分隔）
}

// AgentUpdateResp 更新智能体响应
type AgentUpdateResp struct {
	ID int64 `json:"id" example:"1"`
}

// AgentDeleteReq 删除智能体请求
type AgentDeleteReq struct {
	ID int64 `json:"id" binding:"required" example:"1"`
}

// AgentEnableReq 启用智能体请求
type AgentEnableReq struct {
	ID int64 `json:"id" binding:"required" example:"1"`
}

// AgentDisableReq 禁用智能体请求
type AgentDisableReq struct {
	ID int64 `json:"id" binding:"required" example:"1"`
}
