package dto

import (
	"encoding/json"

	"github.com/ai-agent-os/ai-agent-os/pkg/functionschema"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// WorkspaceChatReq 工作台对话请求（只认 LLM，单模式）
type WorkspaceChatReq struct {
	FullCodePath string       `json:"full_code_path" binding:"required"` // 目录完整路径（必填）
	Message      WorkspaceMsg `json:"message" binding:"required"`        // 本条消息
	SessionID    string       `json:"session_id"`                        // 会话 ID，空则新建
	ModeCode     string       `json:"mode_code"`                         // 工作台模式代码，空则默认 dev
	LLMConfigID  int64        `json:"llm_config_id"`                     // LLM 配置 ID，0 表示使用默认 LLM
	Resume       bool         `json:"resume,omitempty"`                  // true 时不保存 message，只基于已有会话消息继续执行
}

// WorkspaceMsg 工作台单条消息
type WorkspaceMsg struct {
	Content        string `json:"content" binding:"required"`
	DisplayContent string `json:"display_content,omitempty"` // 前端展示内容，模型仍使用 content
	Files          string `json:"files,omitempty"`           // 文件引用字符串，格式 bucket/object_key，多文件逗号分隔
	ContextUsage   string `json:"context_usage,omitempty"`   // include/display_only/artifact
	ArtifactKind   string `json:"artifact_kind,omitempty"`   // 结构化产物类型
}

// WorkspaceChatResp 工作台对话响应
type WorkspaceChatResp struct {
	SessionID string                         `json:"session_id"`
	Content   string                         `json:"content"`
	ToolCalls []WorkspaceChatToolCallSummary `json:"tool_calls,omitempty"`
}

// WorkspaceChatToolCallSummary 工作台单次 tool 调用摘要（供前端展示）
type WorkspaceChatToolCallSummary struct {
	ID         string              `json:"id"`                    // tool_call_id（用于关联 tool 消息）
	Name       string              `json:"name"`                  // 工具名称
	Status     string              `json:"status"`                // ok / error
	Arguments  string              `json:"arguments"`             // 参数（JSON 字符串，可选）
	Result     string              `json:"result"`                // 结果内容（从对应的 tool 消息中获取，可选）
	ResultData interface{}         `json:"result_data,omitempty"` // 结构化结果（优先供前端展示/提取文件）
	Metadata   *ToolResultMetadata `json:"metadata,omitempty"`    // 工具结果元数据（如哪些输出字段是文件）
	Error      string              `json:"error"`                 // 错误信息（如果有，可选）
}

type ToolResultMetadata struct {
	DisplayFileFields []string `json:"display_file_fields,omitempty"` // data 中按文件引用展示的字段名，如 output_files
}

// ListWorkspaceSessionsReq 获取工作台会话列表请求
type ListWorkspaceSessionsReq struct {
	FullCodePath string `json:"full_code_path" form:"full_code_path" binding:"required"` // 必填：服务目录完整路径
	Page         int    `json:"page" form:"page"`                                        // 页码，从1开始，默认1
	PageSize     int    `json:"page_size" form:"page_size"`                              // 每页数量，默认20
}

// ListWorkspaceSessionsResp 获取工作台会话列表响应
type ListWorkspaceSessionsResp struct {
	Sessions []*WorkspaceSessionItem `json:"sessions"`  // 会话列表
	Total    int64                   `json:"total"`     // 总数
	Page     int                     `json:"page"`      // 当前页码
	PageSize int                     `json:"page_size"` // 每页数量
}

// WorkspaceSessionItem 工作台会话项
type WorkspaceSessionItem struct {
	SessionID         string      `json:"session_id"`                    // 会话ID
	Title             string      `json:"title"`                         // 会话标题
	User              string      `json:"user"`                          // 创建该会话的用户
	AgentID           *int64      `json:"agent_id"`                      // 关联的智能体ID（可为空）
	AgentName         string      `json:"agent_name"`                    // 智能体名称（如果有）
	ModeCode          string      `json:"mode_code"`                     // 工作台模式代码
	Status            string      `json:"status"`                        // 会话状态（active/generating/output/pending_confirmation/pending_test/done/cancelled）
	RoleID            string      `json:"role_id,omitempty"`             // 当前工作台角色 ID
	RoleDisplayName   string      `json:"role_display_name,omitempty"`   // 当前工作台角色展示名称
	FullCodePath      string      `json:"full_code_path,omitempty"`      // 所属目录完整路径
	DirectoryName     string      `json:"directory_name,omitempty"`      // 所属目录展示名称
	ParentSessionID   string      `json:"parent_session_id,omitempty"`   // 阶段交接来源会话ID
	HandoffKind       string      `json:"handoff_kind,omitempty"`        // 阶段交接产物类型
	HandoffTargetRole string      `json:"handoff_target_role,omitempty"` // 阶段交接目标身份
	ContextPolicy     string      `json:"context_policy,omitempty"`      // 模型上下文策略
	ArchivedForModel  bool        `json:"archived_for_model,omitempty"`  // 是否已归档且不再进入模型上下文
	ArchiveReason     string      `json:"archive_reason,omitempty"`      // 归档原因
	CreatedAt         models.Time `json:"created_at"`                    // 创建时间
	UpdatedAt         models.Time `json:"updated_at"`                    // 更新时间
}

// WorkspaceHandoffReq 创建阶段交接会话请求。
type WorkspaceHandoffReq struct {
	SourceSessionID string          `json:"source_session_id" binding:"required"`
	FullCodePath    string          `json:"full_code_path" binding:"required"`
	TargetRole      string          `json:"target_role" binding:"required"`
	ArtifactKind    string          `json:"artifact_kind" binding:"required"`
	Artifact        json.RawMessage `json:"artifact" binding:"required"`
	Remark          string          `json:"remark,omitempty"`
	ContextPolicy   string          `json:"context_policy,omitempty"`
	Title           string          `json:"title,omitempty"`
	DisplayContent  string          `json:"display_content,omitempty"`
}

// WorkspaceHandoffResp 阶段交接会话创建结果。
type WorkspaceHandoffResp struct {
	SessionID       string `json:"session_id"`
	SourceSessionID string `json:"source_session_id"`
	TargetRole      string `json:"target_role"`
	ArtifactKind    string `json:"artifact_kind"`
	ContextPolicy   string `json:"context_policy"`
	HandoffPacketID int64  `json:"handoff_packet_id,omitempty"`
	MessageID       int64  `json:"message_id,omitempty"`
	Content         string `json:"content"`
	DisplayContent  string `json:"display_content"`
}

// ResolveWorkspacePendingInteractionReq 清除工作台会话的待交互状态。
type ResolveWorkspacePendingInteractionReq struct {
	SessionID string `json:"session_id" binding:"required"`
}

// CancelWorkspaceChatReq 取消工作台会话执行请求
type CancelWorkspaceChatReq struct {
	SessionID string `json:"session_id" binding:"required"`
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
	ID             int64                          `json:"id"`                        // 消息ID
	SessionID      string                         `json:"session_id"`                // 会话ID
	AgentID        int64                          `json:"agent_id"`                  // 智能体ID（0表示未关联）
	Role           string                         `json:"role"`                      // 角色：user/assistant/tool
	User           string                         `json:"user"`                      // 创建该消息的用户
	Content        string                         `json:"content"`                   // 消息内容（user 仅存用户文字，不含 <files> 块）
	DisplayContent string                         `json:"display_content,omitempty"` // 前端展示内容，空则展示 content
	Files          *string                        `json:"files,omitempty"`           // 用户消息附带的文件引用字符串，仅 user 角色可能有
	ToolCalls      []WorkspaceChatToolCallSummary `json:"tool_calls,omitempty"`      // 工具调用列表（仅assistant角色）
	LLMConfigID    int64                          `json:"llm_config_id,omitempty"`   // assistant 消息生成时使用的 LLM 配置 ID
	LLMConfigName  string                         `json:"llm_config_name,omitempty"` // assistant 消息生成时使用的 LLM 配置名称快照
	LLMProvider    string                         `json:"llm_provider,omitempty"`    // assistant 消息生成时使用的 LLM provider
	LLMModel       string                         `json:"llm_model,omitempty"`       // assistant 消息生成时使用的模型名称
	ContextUsage   string                         `json:"context_usage,omitempty"`   // 模型上下文用途
	ArtifactKind   string                         `json:"artifact_kind,omitempty"`   // 结构化产物类型
	CreatedAt      models.Time                    `json:"created_at"`                // 创建时间
}

// ToolDef 工具定义（list_tools 返回、LLM tools 入参，即 MCP tool schema）
//
// 与 function 表的关系：function 表的 schema 内保存 form/table/chart 的字段配置。
// 后续由适配层 FunctionToMCPToolDef 将 schema 中的输入/输出字段转换为 MCP ToolDef。
type ToolDef struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	InputSchema  map[string]interface{} `json:"input_schema"`            // 请求参数 → JSON Schema
	OutputSchema map[string]interface{} `json:"output_schema,omitempty"` // 响应参数 → JSON Schema（可选，后续从 function schema 转换）
}

// ----- 以下为 app-server 工作空间资源更新接口使用（canonical 标识为 resource_path=/user/app） -----

// UpdateWorkspaceReq 更新工作空间请求（只更新 MySQL 如 Admins；canonical 标识为 resource_path）
type UpdateWorkspaceReq struct {
	User         string `json:"-"`                       // 兼容字段，优先从 resource_path 解析
	App          string `json:"-"`                       // 兼容字段，优先从 resource_path 解析
	ResourcePath string `json:"resource_path,omitempty"` // 工作空间资源路径，规范为 /user/app
	Admins       string `json:"admins"`                  // 管理员列表，逗号分隔
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
	// FileSource 文件列表来源：snapshot（默认，快照表）/ runtime（实时从 app-runtime 磁盘读，更准）
	FileSource string `json:"file_source" form:"file_source"`
}

// WorkspaceContextNode 工作台环境节点信息
type WorkspaceContextNode struct {
	ID           int64                          `json:"id"`
	Name         string                         `json:"name"`                // 节点名称
	Code         string                         `json:"code"`                // 节点代码
	Type         string                         `json:"type"`                // 节点类型：package（目录）或 function（函数）
	Description  string                         `json:"description"`         // 节点描述
	FullCodePath string                         `json:"full_code_path"`      // 完整路径
	TemplateType string                         `json:"template_type"`       // 函数类型（仅 function 有效）：table、form、chart
	Callbacks    []string                       `json:"callbacks,omitempty"` // 函数回调能力摘要（仅 function 有效）
	Schema       *functionschema.FunctionSchema `json:"schema,omitempty"`    // 函数 schema 摘要（仅 function 有效）
}

// WorkspaceContextDirectory 工作台环境目录信息
type WorkspaceContextDirectory struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`           // 目录名称
	Code         string `json:"code"`           // 目录代码
	FullCodePath string `json:"full_code_path"` // 完整路径
	Description  string `json:"description"`    // 目录描述
	Type         string `json:"type"`           // 节点类型
}

// WorkspaceContextFile 工作台环境文件信息
type WorkspaceContextFile struct {
	FileName      string `json:"file_name"`      // 文件名（不含 .go 后缀）
	RelativePath  string `json:"relative_path"`  // 文件相对路径
	FileType      string `json:"file_type"`      // 文件类型（go, json, yaml等）
	Content       string `json:"content"`        // 文件代码内容
	ContentLength int    `json:"content_length"` // 内容长度（字符数）
	LineCount     int    `json:"line_count"`     // 文件总行数
}

// GetWorkspaceContextResp 获取工作台环境信息响应
type GetWorkspaceContextResp struct {
	User                   string                    `json:"user"`                      // 当前用户
	DepartmentFullPath     string                    `json:"department_full_path"`      // 当前用户部门完整路径（存储/逻辑用，英文 code 路径）
	DepartmentFullNamePath string                    `json:"department_full_name_path"` // 当前用户部门中文名称路径（仅展示用，如 技术部/后端组）
	Directory              WorkspaceContextDirectory `json:"directory"`                 // 当前目录信息
	Children               []WorkspaceContextNode    `json:"children"`                  // 子节点列表
	Files                  []WorkspaceContextFile    `json:"files"`                     // 代码文件列表
}

// ReplaceItem 单次替换项（预期次数不传或 0 表示默认 1）
type ReplaceItem struct {
	SearchString  string `json:"search_string" binding:"required"` // 要被替换的原文
	ReplaceString string `json:"replace_string"`                   // 替换后的内容
	ExpectedCount int    `json:"expected_count"`                   // 预期匹配次数，不传或 0 表示 1；若实际次数不符且 all_or_nothing 则不落盘
}

// ReplaceFileContentReq 工作台文件 search-replace 请求（统一批量：多组替换同一文件，全部生效才落盘）
type ReplaceFileContentReq struct {
	FullCodePath      string        `json:"full_code_path" form:"full_code_path" binding:"required"` // 目录完整路径
	FileName          string        `json:"file_name" form:"file_name" binding:"required"`           // 文件名（如 handler 或 handler.go）
	Replacements      []ReplaceItem `json:"replacements" form:"replacements" binding:"required"`     // 替换列表，按顺序执行；每项可设 expected_count，不传或 0 视为 1
	AllOrNothing      bool          `json:"all_or_nothing" form:"all_or_nothing"`                    // 为 true 时仅当所有项 actual_count==expected_count 才落盘，默认 true
	ReturnFullContent bool          `json:"return_full_content" form:"return_full_content"`          // 是否在响应中返回替换后的完整文件内容
}

// ReplaceItemResult 单次替换结果（用于校验失败时返回哪一项不符）
type ReplaceItemResult struct {
	Index         int `json:"index"`          // 替换项下标（从 0 开始）
	ExpectedCount int `json:"expected_count"` // 预期匹配次数
	ActualCount   int `json:"actual_count"`   // 实际匹配次数
}

// ReplaceFileContentResp 工作台文件 search-replace 响应
type ReplaceFileContentResp struct {
	Success      bool                `json:"success"`
	Message      string              `json:"message"`
	ReplaceCount int                 `json:"replace_count"`          // 总替换次数
	FullContent  string              `json:"full_content,omitempty"` // 替换后的完整文件内容（成功且 return_full_content 时返回）
	Details      []ReplaceItemResult `json:"details,omitempty"`      // 未落盘时返回哪几项 expected_count 不符
}

// DeleteFileReq 工作台删除文件请求（删节点+删磁盘）
type DeleteFileReq struct {
	FullCodePath string `json:"full_code_path" form:"full_code_path" binding:"required"` // 目录完整路径
	FileName     string `json:"file_name" form:"file_name" binding:"required"`           // 文件名（如 handler 或 handler.go）
}

// DeleteFileResp 工作台删除文件响应
type DeleteFileResp struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ReadAppLogReq 读取应用日志请求（agent-server -> app-server）
type ReadAppLogReq struct {
	FullCodePath string `json:"full_code_path" binding:"required"` // 目录完整路径（用于解析 user/app）
	Version      string `json:"version"`                           // 版本号（如 v48），为空默认当前版本
	Lines        int    `json:"lines"`                             // 返回行数（默认 200，最大 1000）
	Keyword      string `json:"keyword"`                           // 关键词（可选）
	ContextLines int    `json:"context_lines"`                     // 命中上下文行数（可选，默认 2，最大 5）
	MaxMatches   int    `json:"max_matches"`                       // 最大命中数（可选，默认 50，最大 200）
	IgnoreCase   bool   `json:"ignore_case"`                       // 关键词是否忽略大小写（可选）
}

// ReadAppLogResp 读取应用日志响应（app-server -> agent-server）
type ReadAppLogResp struct {
	Success         bool   `json:"success"`
	Message         string `json:"message"`
	ResolvedVersion string `json:"resolved_version"` // 实际读取的版本
	LogFile         string `json:"log_file"`         // 日志文件名
	TotalLines      int    `json:"total_lines"`      // 日志总行数
	ReturnedLines   int    `json:"returned_lines"`   // 返回行数
	MatchCount      int    `json:"match_count"`      // 命中数（keyword 模式）
	Truncated       bool   `json:"truncated"`        // 是否因限制被截断
	Content         string `json:"content"`          // 日志内容
}
