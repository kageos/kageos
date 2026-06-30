package model

import (
	"github.com/kageos/kageos/pkg/gormx/models"
)

// AgentChatMessage 工作台聊天消息模型。
// 类型名与表名保留历史命名，以兼容已有数据库。
type AgentChatMessage struct {
	models.Base
	SessionID        string  `gorm:"type:varchar(64);not null;index;comment:会话ID" json:"session_id"`
	Role             string  `gorm:"type:varchar(32);not null;comment:消息角色(system/user/assistant/tool)" json:"role"`
	Content          string  `gorm:"type:longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;comment:消息内容" json:"content"`
	DisplayContent   string  `gorm:"type:longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;comment:前端展示内容，空则展示content" json:"display_content"`
	Files            *string `gorm:"type:longtext;comment:文件引用列表（逗号分隔）" json:"files"`                 // 存储 bucket/object_key 引用，可为NULL
	ToolCalls        *string `gorm:"type:json;comment:assistant的tool_calls(LLM返回)" json:"tool_calls"` // 可为NULL
	ToolCallID       string  `gorm:"type:varchar(64);comment:role=tool时的tool_call_id" json:"tool_call_id"`
	ToolStatus       string  `gorm:"type:varchar(32);comment:role=tool时的执行状态" json:"tool_status"`
	ResultData       *string `gorm:"type:json;comment:role=tool时的结构化结果JSON" json:"result_data"`
	ResultMetadata   *string `gorm:"type:json;comment:role=tool时的结果元数据JSON" json:"result_metadata"`
	LLMConfigID      int64   `gorm:"type:bigint;index;comment:生成该消息使用的LLM配置ID，0表示未记录" json:"llm_config_id"`
	LLMConfigName    string  `gorm:"type:varchar(255);comment:生成该消息使用的LLM配置名称快照" json:"llm_config_name"`
	LLMProvider      string  `gorm:"type:varchar(32);index;comment:生成该消息使用的固定LLM协议标记" json:"llm_provider"`
	LLMModel         string  `gorm:"type:varchar(128);index;comment:生成该消息使用的模型名称" json:"llm_model"`
	LLMUsage         *string `gorm:"type:json;comment:生成该消息的LLM token用量JSON" json:"llm_usage"`
	ModelContextPlan *string `gorm:"type:json;comment:生成该消息时的模型上下文计划JSON" json:"model_context_plan"`
	ContextUsage     string  `gorm:"type:varchar(32);not null;default:'include';index;comment:消息用途标签(include/display_only/artifact/current_turn)，运行时默认保留进完整上下文" json:"context_usage"`
	ArtifactKind     string  `gorm:"type:varchar(64);index;comment:结构化产物类型" json:"artifact_kind"`
	User             string  `gorm:"type:varchar(128);not null;index;comment:创建用户" json:"user"`
}

// TableName 指定表名
func (AgentChatMessage) TableName() string {
	return "agent_chat_messages"
}
