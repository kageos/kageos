package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// WorkspaceEvent 工作台埋点事件（如无法实现的需求、需求不明确等），便于追溯与分析
type WorkspaceEvent struct {
	models.Base
	SessionID    string `gorm:"type:varchar(64);index;comment:会话ID，便于追溯" json:"session_id"`
	FullCodePath string `gorm:"type:varchar(512);index;comment:工作目录完整路径" json:"full_code_path"`
	User         string `gorm:"type:varchar(128);not null;index;comment:触发用户" json:"user"`
	EventType    string `gorm:"type:varchar(64);not null;index;comment:事件类型(unsupported_demand/unclear_requirement/task_failed等)" json:"event_type"`
	Description  string `gorm:"type:varchar(1024);not null;comment:一句话描述" json:"description"`
	Context      string `gorm:"type:varchar(2048);comment:上下文摘要" json:"context"`
	Extra        string `gorm:"type:text;comment:额外JSON" json:"extra"`
}

// TableName 指定表名
func (WorkspaceEvent) TableName() string {
	return "workspace_events"
}
