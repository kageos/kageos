package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// AgentChatSession 智能体聊天会话模型
type AgentChatSession struct {
	models.Base
	TreeID    int64  `gorm:"type:bigint;not null;index;comment:服务目录ID" json:"tree_id"`
	SessionID string `gorm:"type:varchar(64);not null;uniqueIndex;comment:会话ID（UUID）" json:"session_id"`
	AgentID   int64  `gorm:"type:bigint;not null;index;comment:智能体ID" json:"agent_id"` // 关联的智能体ID
	Title     string `gorm:"type:varchar(255);comment:会话标题" json:"title"`              // 自动生成或用户自定义
	User      string `gorm:"type:varchar(128);not null;index;comment:创建用户" json:"user"`
	
	// 关联的智能体（预加载）
	Agent *Agent `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
}

// TableName 指定表名
func (AgentChatSession) TableName() string {
	return "agent_chat_sessions"
}

