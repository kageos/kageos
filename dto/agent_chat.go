package dto

import (
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
)

// AgentChatReq 智能体聊天请求
type AgentChatReq struct {
	AgentID   int64  `json:"agent_id" binding:"required" example:"1"` // 智能体ID
	Router    string `json:"router" binding:"required" example:"/luobei/demo/crm"`
	SessionID string `json:"session_id"  example:"1"` //会话id，首次为空

	//这里的Messages可以改一下了，历史记录可以后端来处理，根据会话id来查询历史记录自动加上即可，前端每次只需要传递此次的消息即可
	Messages []Message `json:"messages" binding:"required" example:"[{\"role\":\"user\",\"content\":\"你好\"}]"` // 对话消息列表
}

type FunctionGenAgentChatReq struct {
	AgentID            int64                    `json:"agent_id" binding:"required" example:"1"`        // 智能体ID
	TreeID             int64                    `json:"tree_id" binding:"required" example:"629"`      // 服务目录ID
	SessionID          string                   `json:"session_id" example:""`                        // 会话ID（UUID），首次为空，后端自动生成
	ExistingDirectories []ExistingDirectoryInfo `json:"existing_directories" example:"[{\"code\":\"ticket\",\"name\":\"工单管理\"}]"` // 当前目录下已存在的子目录列表（格式：目录代码:目录名称）
	Message            Message                  `json:"message" binding:"required"`                   // 单条消息（历史记录后端自动加载）
}

// ExistingDirectoryInfo 已存在的子目录信息
type ExistingDirectoryInfo struct {
	Code string `json:"code" example:"ticket"`   // 目录代码
	Name string `json:"name" example:"工单管理"` // 目录名称
}

// Message 对话消息
type Message struct {
	Content string      `json:"content" binding:"required" example:"你好"` // 消息内容
	Files   *types.Files `json:"files,omitempty"`                            // 文件列表（直接使用 types.Files）
}

// AgentChatResp 智能体聊天响应
type AgentChatResp struct {
	Content string `json:"content" example:"你好！有什么可以帮助你的吗？"` // AI回答内容
	Usage   *Usage `json:"usage,omitempty"`                  // Token使用统计（可选）
}

// FunctionGenAgentChatResp 函数生成智能体聊天响应
type FunctionGenAgentChatResp struct {
	SessionID   string `json:"session_id" example:"550e8400-e29b-41d4-a716-446655440000"` // 会话ID（首次创建时返回）
	Content     string `json:"content" example:"正在生成代码..."`                               // AI回答内容
	RecordID    int64  `json:"record_id,omitempty" example:"1"`                           // function_gen 记录ID（如果触发生成）
	Status      string `json:"status" example:"generating"`                               // 状态：generating/completed/failed
	CanContinue bool   `json:"can_continue" example:"false"`                             // 是否可以继续输入（true: 可以继续输入, false: 不能再输入）
	Usage       *Usage `json:"usage,omitempty"`                                           // Token使用统计（可选）
}

// Usage Token使用统计
type Usage struct {
	PromptTokens     int `json:"prompt_tokens" example:"10"`     // 输入token数
	CompletionTokens int `json:"completion_tokens" example:"20"` // 输出token数
	TotalTokens      int `json:"total_tokens" example:"30"`      // 总token数
}

// AddFunctionsReq 添加函数请求（agent-server -> workspace）
// 用于向服务目录添加函数，将生成的代码写入到工作空间对应的目录下
type AddFunctionsReq struct {
	RecordID  int64  `json:"record_id" example:"1"`                                       // function_gen 记录ID
	MessageID int64  `json:"message_id" example:"1"`                                      // 消息ID（关联到 AgentChatMessage.ID）
	AgentID   int64  `json:"agent_id" example:"1"`                                        // 智能体ID
	TreeID    int64  `json:"tree_id" example:"629"`                                       // 服务目录ID
	User      string `json:"user" example:"beiluo"`                                       // 用户标识
	// 处理后的结构化数据（agent-server 处理后的结果）
	FileName   string `json:"file_name" example:"crm_ticket"`   // 从代码中提取的文件名
	SourceCode string `json:"source_code" example:"package..."`  // 处理后的源代码（从 Markdown 中提取）
	Async      bool   `json:"async" example:"false"`            // 是否异步处理（true: 异步，通过回调通知；false: 同步，直接返回结果）
}

// FunctionGenResult 已废弃，请使用 AddFunctionsReq
// Deprecated: 使用 AddFunctionsReq 替代
type FunctionGenResult = AddFunctionsReq

// AddFunctionsResp 添加函数响应（同步模式返回）
type AddFunctionsResp struct {
	Success bool   `json:"success" example:"true"`         // 是否成功
	AppID   int64  `json:"app_id" example:"1"`             // 应用ID
	AppCode string `json:"app_code" example:"myapp"`      // 应用代码
	Error   string `json:"error,omitempty" example:""`     // 错误信息（如果失败）
}

// AddFunctionsAsyncResp 添加函数响应（异步模式返回）
type AddFunctionsAsyncResp struct {
	RecordID int64  `json:"record_id" example:"7"`                              // 生成记录ID
	Message  string `json:"message" example:"函数添加请求已接收，正在异步处理"` // 提示消息
}

// PluginRunReq 插件执行请求
type PluginRunReq struct {
	Content string       `json:"content" binding:"required" example:"请处理这个Excel文件"` // 用户消息内容
	Files   *types.Files `json:"files,omitempty"`                                         // 文件列表（直接使用 types.Files）
}

// PluginRunResp 插件执行响应
type PluginRunResp struct {
	Data  string `json:"data" example:"工单标题,问题描述,优先级,工单状态\n工单1,描述1,低,待处理"` // 处理后的数据（格式化后的文本，供LLM理解）
	Error string `json:"error,omitempty" example:"文件解析失败: 读取 CSV 行失败"`              // 错误信息（如果有），如果设置了此字段，表示插件处理失败，不应调用 LLM
}

// AgentPluginFormReq 智能体插件场景的 Form API 请求（固定格式）
// 用于调用 Form API 时的请求结构
type AgentPluginFormReq struct {
	Content    string       `json:"content,omitempty"`      // 文本输入（可选）
	InputFiles *types.Files `json:"input_files,omitempty"` // 文件输入（可选）
}

// AgentPluginFormResp 智能体插件场景的 Form API 响应（固定格式）
// 用于调用 Form API 时的响应结构
type AgentPluginFormResp struct {
	Result string `json:"result"` // 文本输出
}

// ChatSessionListReq 获取会话列表请求
type ChatSessionListReq struct {
	TreeID   int64 `json:"tree_id" form:"tree_id" binding:"required" example:"629"`     // 服务目录ID
	Page     int   `json:"page" form:"page" binding:"required" example:"1"`            // 页码
	PageSize int   `json:"page_size" form:"page_size" binding:"required" example:"10"` // 每页数量
}

// ChatSessionInfo 会话信息
type ChatSessionInfo struct {
	ID        int64      `json:"id" example:"1"`                           // 会话ID
	TreeID    int64      `json:"tree_id" example:"629"`                     // 服务目录ID
	SessionID string     `json:"session_id" example:"550e8400-e29b-41d4-a716-446655440000"` // 会话ID（UUID）
	AgentID   int64      `json:"agent_id" example:"1"`                     // 关联的智能体ID
	Agent     *AgentInfo `json:"agent,omitempty"`                          // 关联的智能体信息（可选）
	Title     string     `json:"title" example:"会话标题"`                    // 会话标题
	Status    string     `json:"status" example:"active"`                  // 会话状态：active(活跃)/generating(生成中)/done(已完成)
	User      string     `json:"user" example:"beiluo"`                     // 创建用户
	CreatedAt string     `json:"created_at" example:"2006-01-02T15:04:05Z"` // 创建时间
	UpdatedAt string     `json:"updated_at" example:"2006-01-02T15:04:05Z"` // 更新时间
}

// ChatSessionListResp 获取会话列表响应
type ChatSessionListResp struct {
	Sessions []ChatSessionInfo `json:"sessions"` // 会话列表
	Total    int64             `json:"total" example:"100"` // 总数
}

// ChatMessageListReq 获取消息列表请求
type ChatMessageListReq struct {
	SessionID string `json:"session_id" form:"session_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"` // 会话ID
}

// ChatMessageInfo 消息信息
type ChatMessageInfo struct {
	ID        int64  `json:"id" example:"1"`                           // 消息ID
	SessionID string `json:"session_id" example:"550e8400-e29b-41d4-a716-446655440000"` // 会话ID
	AgentID   int64  `json:"agent_id" example:"1"`                     // 处理该消息的智能体ID
	Role      string `json:"role" example:"user"`                      // 消息角色：user/assistant/system
	Content   string `json:"content" example:"你好"`                     // 消息内容
	Files     string `json:"files,omitempty" example:"[{\"url\":\"...\",\"remark\":\"...\"}]"` // 文件列表（JSON字符串，可选）
	User      string `json:"user" example:"beiluo"`                    // 创建用户
	CreatedAt string `json:"created_at" example:"2006-01-02T15:04:05Z"` // 创建时间
}

// ChatMessageListResp 获取消息列表响应
type ChatMessageListResp struct {
	Messages []ChatMessageInfo `json:"messages"` // 消息列表（按创建时间升序）
}

// FunctionGenCallback 函数生成回调（app-server -> agent-server）
type FunctionGenCallback struct {
	RecordID      int64    `json:"record_id"`        // 生成记录ID
	MessageID     int64    `json:"message_id"`       // 消息ID
	Success       bool     `json:"success"`           // 是否成功
	FullCodePaths []string `json:"full_code_paths,omitempty" example:"[\"/user/app/function\"]"` // 生成的函数完整代码路径列表
	AppID         int64    `json:"app_id"`           // 应用ID
	AppCode       string   `json:"app_code"`         // 应用代码（冗余存储，提高查询效率）
	Error         string   `json:"error,omitempty"`   // 错误信息（如果失败）
}

// FunctionGenStatusReq 查询代码生成状态请求
type FunctionGenStatusReq struct {
	RecordID int64 `json:"record_id" form:"record_id" binding:"required" example:"1"` // 生成记录ID
}

// FunctionGenStatusResp 查询代码生成状态响应
type FunctionGenStatusResp struct {
	RecordID      int64    `json:"record_id" example:"1"`                              // 生成记录ID
	Status        string   `json:"status" example:"generating"`                         // 状态：generating/completed/failed
	Code          string   `json:"code,omitempty" example:"package main\n\nfunc main() {}"` // 生成的代码（仅在 completed 时返回）
	ErrorMsg      string   `json:"error_msg,omitempty" example:"生成失败"`                // 错误信息（仅在 failed 时返回）
	FullCodePaths []string `json:"full_code_paths,omitempty" example:"[\"/user/app/function\"]"` // 生成的函数完整代码路径列表（仅在 completed 时返回）
	Duration      int      `json:"duration" example:"30"`                              // 生成耗时（秒）
	CreatedAt     string   `json:"created_at" example:"2006-01-02T15:04:05Z"`           // 创建时间
	UpdatedAt     string   `json:"updated_at" example:"2006-01-02T15:04:05Z"`         // 更新时间
}
