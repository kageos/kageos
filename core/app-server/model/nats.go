package model

import (
	"net"
	"net/url"
	"strconv"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

type Nats struct {
	models.Base
	Host     string `json:"host" gorm:"column:host;type:varchar(255)"`
	Port     int    `json:"port" gorm:"column:port;type:int"`
	User     string `json:"user" gorm:"column:user;type:varchar(255)"`
	Password string `json:"-" gorm:"column:password;type:varchar(255)"`
}

func (n *Nats) URL() string {
	hostPort := net.JoinHostPort(n.Host, strconv.Itoa(n.Port))
	if n.User == "" {
		return "nats://" + hostPort
	}
	return "nats://" + url.UserPassword(n.User, n.Password).String() + "@" + hostPort
}

func (n *Nats) TableName() string {
	return "nats"
}
