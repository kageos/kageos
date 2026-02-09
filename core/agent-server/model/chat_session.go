package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// AgentChatSession 智能体聊天会话模型
type AgentChatSession struct {
	models.Base
	TreeID       int64  `gorm:"type:bigint;not null;index;comment:服务目录ID" json:"tree_id"`
	FullCodePath string `gorm:"type:varchar(512);index;comment:服务目录完整路径（workspace 用，有语意）" json:"full_code_path"`
	Source       string `gorm:"type:varchar(32);index;comment:来源(workspace=工作台,空=function_gen)" json:"source"`
	SessionID    string `gorm:"type:varchar(64);not null;uniqueIndex;comment:会话ID（UUID）" json:"session_id"`
	AgentID      *int64 `gorm:"type:bigint;index;comment:智能体ID（已废弃，工作台恒为空）" json:"agent_id"`
	Title        string `gorm:"type:varchar(255);comment:会话标题" json:"title"`
	Status       string `gorm:"type:varchar(32);not null;default:'active';index;comment:会话状态(active/generating/done)" json:"status"`
	User         string `gorm:"type:varchar(128);not null;index;comment:创建用户" json:"user"`
}

// 会话状态常量
const (
	ChatSessionStatusActive     = "active"     // 活跃状态，可以继续输入
	ChatSessionStatusGenerating = "generating" // 生成中，锁定会话，不允许输入
	ChatSessionStatusDone       = "done"       // 已完成，会话结束，不能再输入
)

// TableName 指定表名
func (AgentChatSession) TableName() string {
	return "agent_chat_sessions"
}

