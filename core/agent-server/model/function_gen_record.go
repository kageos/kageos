package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// FunctionGenRecord 函数生成记录模型
type FunctionGenRecord struct {
	models.Base
	SessionID string `gorm:"type:varchar(64);not null;index;comment:会话ID" json:"session_id"`
	AgentID   int64  `gorm:"type:bigint;not null;index;comment:智能体ID" json:"agent_id"`
	TreeID    int64  `gorm:"type:bigint;not null;index;comment:服务目录ID" json:"tree_id"`
	Status    string `gorm:"type:varchar(32);not null;index;comment:状态(generating/completed/failed)" json:"status"`
	Code      string `gorm:"type:longtext;comment:生成的代码" json:"code"`
	ErrorMsg  string `gorm:"type:text;comment:错误信息" json:"error_msg"`
	User      string `gorm:"type:varchar(128);not null;index;comment:创建用户" json:"user"`
}

// TableName 指定表名
func (FunctionGenRecord) TableName() string {
	return "function_gen_records"
}

// 状态常量
const (
	FunctionGenStatusGenerating = "generating" // 生成中
	FunctionGenStatusCompleted  = "completed"  // 已完成
	FunctionGenStatusFailed    = "failed"      // 失败
)

