package dto

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
)

// WorkspaceChatReq 工作台对话请求
type WorkspaceChatReq struct {
	FullCodePath string         `json:"full_code_path" binding:"required"`       // 目录完整路径（必填）
	Message      WorkspaceMsg   `json:"message" binding:"required"`              // 本条消息
	SessionID    string         `json:"session_id"`                              // 会话 ID，空则新建
	AgentID      int64          `json:"agent_id"`                                // 智能体 ID，可选
}

// WorkspaceMsg 工作台单条消息
type WorkspaceMsg struct {
	Content string       `json:"content" binding:"required"`
	Files   *types.Files `json:"files,omitempty"`
}

// WorkspaceChatResp 工作台对话响应
type WorkspaceChatResp struct {
	SessionID string                          `json:"session_id"`
	Content   string                          `json:"content"`
	ToolCalls []WorkspaceChatToolCallSummary  `json:"tool_calls,omitempty"`
	AgentID   int64                           `json:"agent_id,omitempty"` // 当前会话关联的智能体 ID，0 表示未选
}

// WorkspaceChatToolCallSummary 工作台单次 tool 调用摘要（供前端展示）
type WorkspaceChatToolCallSummary struct {
	ID        string `json:"id"`         // tool_call_id（用于关联 tool 消息）
	Name      string `json:"name"`        // 工具名称
	Status    string `json:"status"`     // ok / error
	Arguments string `json:"arguments"`   // 参数（JSON 字符串，可选）
	Result    string `json:"result"`      // 结果内容（从对应的 tool 消息中获取，可选）
	Error     string `json:"error"`       // 错误信息（如果有，可选）
}

// ListWorkspaceSessionsReq 获取工作台会话列表请求
type ListWorkspaceSessionsReq struct {
	FullCodePath string `json:"full_code_path" form:"full_code_path" binding:"required"` // 必填：服务目录完整路径
	Page         int    `json:"page" form:"page"`                                         // 页码，从1开始，默认1
	PageSize     int    `json:"page_size" form:"page_size"`                               // 每页数量，默认20
}

// ListWorkspaceSessionsResp 获取工作台会话列表响应
type ListWorkspaceSessionsResp struct {
	Sessions []*WorkspaceSessionItem `json:"sessions"` // 会话列表
	Total    int64                   `json:"total"`    // 总数
	Page     int                     `json:"page"`     // 当前页码
	PageSize int                     `json:"page_size"` // 每页数量
}

// WorkspaceSessionItem 工作台会话项
type WorkspaceSessionItem struct {
	SessionID string      `json:"session_id"` // 会话ID
	Title     string      `json:"title"`       // 会话标题
	AgentID   *int64      `json:"agent_id"`    // 关联的智能体ID（可为空）
	AgentName string      `json:"agent_name"`  // 智能体名称（如果有）
	Status    string      `json:"status"`      // 会话状态
	CreatedAt models.Time `json:"created_at"`  // 创建时间
	UpdatedAt models.Time `json:"updated_at"` // 更新时间
}

// ListWorkspaceMessagesReq 获取工作台会话消息列表请求
type ListWorkspaceMessagesReq struct {
	SessionID string `json:"session_id" form:"session_id" binding:"required"` // 必填：会话ID
}

// ListWorkspaceMessagesResp 获取工作台会话消息列表响应
type ListWorkspaceMessagesResp struct {
	Messages []WorkspaceMessageInfo `json:"messages"` // 消息列表
}

// WorkspaceMessageInfo 工作台消息信息
type WorkspaceMessageInfo struct {
	ID        int64                              `json:"id"`         // 消息ID
	SessionID string                             `json:"session_id"` // 会话ID
	AgentID   int64                              `json:"agent_id"`   // 智能体ID（0表示未关联）
	Role      string                             `json:"role"`       // 角色：user/assistant/tool
	Content   string                             `json:"content"`    // 消息内容
	ToolCalls []WorkspaceChatToolCallSummary     `json:"tool_calls,omitempty"` // 工具调用列表（仅assistant角色）
	CreatedAt models.Time                        `json:"created_at"` // 创建时间
}

// ToolDef 工具定义（list_tools 返回、LLM tools 入参，即 MCP tool schema）
//
// 与 function 表的关系：function 表的 Request（请求参数）、Response（响应参数）为 []*widget.Field 的 JSON。
// 后续由适配层 FunctionToMCPToolDef 将 Request → InputSchema、Response → OutputSchema，直接得到 MCP ToolDef。
type ToolDef struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	InputSchema   map[string]interface{} `json:"input_schema"`             // 请求参数 → JSON Schema
	OutputSchema  map[string]interface{} `json:"output_schema,omitempty"`  // 响应参数 → JSON Schema（可选，后续从 function.Response 转换）
}

// ListToolsResp 工作台 list_tools 响应
// GET /agent/api/v1/workspace/tools
type ListToolsResp struct {
	Tools []ToolDef `json:"tools"`
}

// CallToolReq 工作台 call_tool 请求（临时测试 /workspace/call_tool 或后续 LLM tool 循环）
// Arguments 为 interface{}：JSON 对象会反序列化为 map[string]interface{}，便于直接传给 CallTool；null/缺省为 nil
type CallToolReq struct {
	FullCodePath string      `json:"full_code_path" binding:"required"`
	ToolName     string      `json:"tool_name" binding:"required"`
	Arguments    interface{} `json:"arguments"`
}

// CallToolResp 工作台 call_tool 响应
type CallToolResp struct {
	Content string `json:"content"`
	IsError bool   `json:"is_error"`
}

// ----- 以下为 app-server 工作空间（user/app）更新接口使用 -----

// UpdateWorkspaceReq 更新工作空间请求（只更新 MySQL 如 Admins；user/app 自路径参数填入）
type UpdateWorkspaceReq struct {
	User   string `json:"-"`       // 从路径 :user 填入，不绑定 body
	App    string `json:"-"`       // 从路径 :app 填入，不绑定 body
	Admins string `json:"admins"`  // 管理员列表，逗号分隔
}

// UpdateWorkspaceResp 更新工作空间响应
type UpdateWorkspaceResp struct {
	User   string `json:"user"`
	App    string `json:"app"`
	Admins string `json:"admins"`
}

// ----- 以下为工作台环境信息接口使用 -----

// GetWorkspaceContextReq 获取工作台环境信息请求
type GetWorkspaceContextReq struct {
	FullCodePath string `json:"full_code_path" form:"full_code_path" binding:"required"`
}

// WorkspaceContextNode 工作台环境节点信息
type WorkspaceContextNode struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`         // 节点名称
	Code        string `json:"code"`         // 节点代码
	Type        string `json:"type"`         // 节点类型：package（目录）或 function（函数）
	Description string `json:"description"`  // 节点描述
	FullCodePath string `json:"full_code_path"` // 完整路径
}

// WorkspaceContextDirectory 工作台环境目录信息
type WorkspaceContextDirectory struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`         // 目录名称
	Code        string `json:"code"`         // 目录代码
	FullCodePath string `json:"full_code_path"` // 完整路径
	Description string `json:"description"`   // 目录描述
	Type        string `json:"type"`          // 节点类型
}

// WorkspaceContextFile 工作台环境文件信息
type WorkspaceContextFile struct {
	FileName     string `json:"file_name"`     // 文件名（不含 .go 后缀）
	RelativePath string `json:"relative_path"` // 文件相对路径
	FileType     string `json:"file_type"`     // 文件类型（go, json, yaml等）
	Content      string `json:"content"`       // 文件代码内容
	ContentLength int   `json:"content_length"` // 内容长度（字符数）
	LineCount     int   `json:"line_count"`     // 文件总行数
}

// GetWorkspaceContextResp 获取工作台环境信息响应
type GetWorkspaceContextResp struct {
	User      string                    `json:"user"`       // 当前用户
	Directory WorkspaceContextDirectory `json:"directory"`   // 当前目录信息
	Children  []WorkspaceContextNode    `json:"children"`   // 子节点列表
	Files     []WorkspaceContextFile    `json:"files"`     // 代码文件列表
}
