package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// AgentChatMessage 智能体聊天消息模型
type AgentChatMessage struct {
	models.Base
	SessionID    string  `gorm:"type:varchar(64);not null;index;comment:会话ID" json:"session_id"`
	AgentID      *int64  `gorm:"type:bigint;index;comment:智能体ID，工作台消息可为空" json:"agent_id"` // 处理该消息的智能体ID
	Role         string  `gorm:"type:varchar(32);not null;comment:消息角色(system/user/assistant/tool)" json:"role"`
	Content      string  `gorm:"type:longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;comment:消息内容" json:"content"`
	Files        *string `gorm:"type:json;comment:文件列表（JSON格式）" json:"files"`     // 存储文件URL数组的JSON，可为NULL
	ToolCalls    *string `gorm:"type:json;comment:assistant的tool_calls(LLM返回)" json:"tool_calls"` // 可为NULL
	ToolCallID   string  `gorm:"type:varchar(64);comment:role=tool时的tool_call_id" json:"tool_call_id"`
	User         string  `gorm:"type:varchar(128);not null;index;comment:创建用户" json:"user"`
}

// TableName 指定表名
func (AgentChatMessage) TableName() string {
	return "agent_chat_messages"
}

