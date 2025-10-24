package model

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

type Nats struct {
	models.Base
	Host string `json:"host" gorm:"column:host;type:varchar(255)"`
	Port int    `json:"port" gorm:"column:port;type:int"`
}

func (n *Nats) URL() string {
	return fmt.Sprintf("nats://%s:%d", n.Host, n.Port)
}

func (n *Nats) TableName() string {
	return "nats"
}
