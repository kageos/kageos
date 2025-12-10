package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"strconv"
	"strings"
)

type App struct {
	models.Base
	User    string `json:"user" gorm:"column:user;type:varchar(255);not null"`
	Code    string `json:"code" gorm:"column:code;type:varchar(255);not null"` //英文标识
	Name    string `json:"name" gorm:"column:name;type:varchar(255);not null"` //中文名称
	NatsID  int64  `gorm:"column:nats_id;type:bigint" json:"nats_id"`          //不同的nats 会把流量分发到不同的机房
	HostID  int64  `gorm:"column:host_id;type:bigint" json:"host_id"`
	Status  string `gorm:"column:status;type:varchar(50)" json:"status"` // 应用状态: enabled(启用), disabled(禁用)
	Version string `gorm:"column:version;type:varchar(50)" json:"version"`
}

func (App) TableName() string {
	return "app"
}

// GetFullName 获取应用全名（用户名/应用名）
func (a *App) GetFullName() string {
	return a.User + "/" + a.Code
}

// GetPrefix 获取应用前缀路径
func (a *App) GetPrefix() string {
	return "/" + a.User + "/" + a.Code
}

// IsEnabled 判断应用是否处于启用状态
func (a *App) IsEnabled() bool {
	return a.Status == "enabled"
}

// IsDisabled 判断应用是否被禁用
func (a *App) IsDisabled() bool {
	return a.Status == "disabled"
}

func (a *App) GetVersionNumber() int {

	version := a.Version
	// 去掉 "v" 前缀
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")
	// 提取数字部分
	num, err := strconv.Atoi(version)
	if err != nil {
		return 0
	}
	return num
}
