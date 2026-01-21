package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// FunctionGroupAgent 函数组和智能体的关联表
// 记录每个 FullGroupCode 对应的智能体和生成记录
type FunctionGroupAgent struct {
	models.Base
	FullGroupCode string `gorm:"type:varchar(500);not null;index;comment:完整函数组代码" json:"full_group_code"`
	AgentID       int64  `gorm:"type:bigint;not null;index;comment:智能体ID" json:"agent_id"`
	RecordID      int64  `gorm:"type:bigint;not null;index;comment:生成记录ID" json:"record_id"`
	AppID         int64  `gorm:"type:bigint;not null;index;comment:应用ID" json:"app_id"`
	AppCode       string `gorm:"type:varchar(128);not null;index;comment:应用代码" json:"app_code"` // 冗余存储，提高查询效率
	User          string `gorm:"type:varchar(128);not null;index;comment:创建用户" json:"user"`

	// 关联
	Agent  *Agent            `gorm:"foreignKey:AgentID;references:ID" json:"agent,omitempty"`
	Record *FunctionGenRecord `gorm:"foreignKey:RecordID;references:ID" json:"record,omitempty"`
}

// TableName 指定表名
func (FunctionGroupAgent) TableName() string {
	return "function_group_agents"
}

