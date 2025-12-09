package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// Task 任务模型
type Task struct {
	models.Base
	Status   string `gorm:"type:varchar(32);not null;index" json:"status"` // pending, processing, completed, failed
	Progress int    `gorm:"default:0" json:"progress"`                     // 0-100
	Result   string `gorm:"type:json" json:"result"`                       // JSON 结果
	Error    string `gorm:"type:text" json:"error"`
	Logs     string `gorm:"type:json" json:"logs"`                         // JSON 日志数组
}

// TableName 指定表名
func (Task) TableName() string {
	return "tasks"
}

