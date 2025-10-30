package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

type Host struct {
	models.Base
	Domain   string `json:"domain" gorm:"column:domain;type:varchar(255)"`
	NatsID   int64  `json:"nats_id" gorm:"column:nats_id;type:bigint"`
	Status   string `json:"status" gorm:"column:status;type:varchar(50);default:'enabled'"` // 主机状态: enabled(启用), disabled(禁用)
	AppCount int    `json:"app_count" gorm:"column:app_count;type:int"`                     //这个nats下面有多少app，

	Nats *Nats `json:"nats" gorm:"foreignKey:ID;references:ID"`
}

func (Host) TableName() string {
	return "host"
}

// IsEnabled 检查主机是否为启用状态
func (h *Host) IsEnabled() bool {
	return h.Status == "enabled"
}

// IsDisabled 检查主机是否为禁用状态
func (h *Host) IsDisabled() bool {
	return h.Status == "disabled"
}
