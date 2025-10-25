package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

type App struct {
	models.Base
	User    string `json:"user" gorm:"column:user;type:varchar(255);not null"`
	Name    string `json:"name" gorm:"column:name;type:varchar(255);not null"`
	NatsID  int64  `gorm:"column:nats_id;type:bigint" json:"nats_id"` //不同的nats 会把流量分发到不同的机房
	HostID  int64  `gorm:"column:host_id;type:bigint" json:"host_id"`
	Status  string `gorm:"column:status;type:varchar(50)" json:"status"` //启用/废弃
	Version string `gorm:"column:version;type:varchar(50)" json:"version"`
}

func (App) TableName() string {
	return "app"
}
