package dto

// AgentChatReq 智能体聊天请求
type AgentChatReq struct {
	AgentID   int64  `json:"agent_id" binding:"required" example:"1"` // 智能体ID
	Router    string `json:"router" binding:"required" example:"/luobei/demo/crm"`
	SessionID string `json:"session_id"  example:"1"` //会话id，首次为空

	//这里的Messages可以改一下了，历史记录可以后端来处理，根据会话id来查询历史记录自动加上即可，前端每次只需要传递此次的消息即可
	Messages []Message `json:"messages" binding:"required" example:"[{\"role\":\"user\",\"content\":\"你好\"}]"` // 对话消息列表
}

type FunctionGenAgentChatReq struct {
	AgentID      int64    `json:"agent_id" binding:"required" example:"1"`        // 智能体ID
	TreeID       int64    `json:"tree_id" binding:"required" example:"629"`      // 服务目录ID
	Package      string   `json:"package" example:"crm"`                        // Package 名称（从前端传递）
	SessionID    string   `json:"session_id" example:""`                        // 会话ID（UUID），首次为空，后端自动生成
	ExistingFiles []string `json:"existing_files" example:"[\"crm_ticket\",\"crm_user\"]"` // 当前 package 下已存在的文件名（不含 .go 后缀）
	Message      Message  `json:"message" binding:"required"`                   // 单条消息（历史记录后端自动加载）
}

// Message 对话消息
type Message struct {
	Content string `json:"content" binding:"required" example:"你好"` // 消息内容
	Files   []struct {
		Url    string `json:"url"` //文件url
		Remark string `json:"remark"`
	} `json:"files"`
}

// AgentChatResp 智能体聊天响应
type AgentChatResp struct {
	Content string `json:"content" example:"你好！有什么可以帮助你的吗？"` // AI回答内容
	Usage   *Usage `json:"usage,omitempty"`                  // Token使用统计（可选）
}

// FunctionGenAgentChatResp 函数生成智能体聊天响应
type FunctionGenAgentChatResp struct {
	SessionID string `json:"session_id" example:"550e8400-e29b-41d4-a716-446655440000"` // 会话ID（首次创建时返回）
	Content   string `json:"content" example:"正在生成代码..."`                               // AI回答内容
	RecordID  int64  `json:"record_id,omitempty" example:"1"`                           // function_gen 记录ID（如果触发生成）
	Status    string `json:"status" example:"generating"`                               // 状态：generating/completed/failed
	Usage     *Usage `json:"usage,omitempty"`                                           // Token使用统计（可选）
}

// Usage Token使用统计
type Usage struct {
	PromptTokens     int `json:"prompt_tokens" example:"10"`     // 输入token数
	CompletionTokens int `json:"completion_tokens" example:"20"` // 输出token数
	TotalTokens      int `json:"total_tokens" example:"30"`      // 总token数
}

// FunctionGenResult 函数生成结果（用于 NATS 消息队列，供 app-server 消费）
type FunctionGenResult struct {
	RecordID int64  `json:"record_id" example:"1"`                                       // function_gen 记录ID
	AgentID  int64  `json:"agent_id" example:"1"`                                        // 智能体ID
	TreeID   int64  `json:"tree_id" example:"629"`                                       // 服务目录ID
	User     string `json:"user" example:"beiluo"`                                       // 用户标识
	Code     string `json:"code" example:"package main\n\nfunc main() {\n\t// 生成的代码\n}"` // 生成的代码内容
}

// PluginFile 插件文件信息
type PluginFile struct {
	Url    string `json:"url" example:"https://example.com/file.xlsx"` // 文件URL
	Remark string `json:"remark" example:"Excel文件"`                    // 文件备注
}

// PluginRunReq 插件执行请求
type PluginRunReq struct {
	Message string       `json:"message" binding:"required" example:"请处理这个Excel文件"`                                    // 用户消息
	Files   []PluginFile `json:"files" example:"[{\"url\":\"https://example.com/file.xlsx\",\"remark\":\"Excel文件\"}]"` // 文件列表
}

// PluginRunResp 插件执行响应
type PluginRunResp struct {
	Data  string `json:"data" example:"工单标题,问题描述,优先级,工单状态\n工单1,描述1,低,待处理"` // 处理后的数据（格式化后的文本，供LLM理解）
	Error string `json:"error,omitempty" example:"文件解析失败: 读取 CSV 行失败"`              // 错误信息（如果有），如果设置了此字段，表示插件处理失败，不应调用 LLM
}
