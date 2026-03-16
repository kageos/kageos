package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"strconv"
	"strings"
)

type App struct {
	models.Base
	User     string `json:"user" gorm:"column:user;type:varchar(255);not null"`
	Code     string `json:"code" gorm:"column:code;type:varchar(255);not null"` //英文标识
	Name     string `json:"name" gorm:"column:name;type:varchar(255);not null"` //中文名称
	NatsID   int64  `gorm:"column:nats_id;type:bigint" json:"nats_id"`          //不同的nats 会把流量分发到不同的机房
	HostID   int64  `gorm:"column:host_id;type:bigint" json:"host_id"`
	Status   string `gorm:"column:status;type:varchar(50)" json:"status"` // 应用状态: enabled(启用), disabled(禁用)
	Version  string `gorm:"column:version;type:varchar(50)" json:"version"`
	IsPublic bool   `gorm:"column:is_public;type:boolean;default:true" json:"is_public"` // 是否公开，默认公开
	Admins   string `gorm:"column:admins;type:text" json:"admins"`                      // 管理员列表，逗号分隔的用户名
	
	// ⭐ 新增：应用类型（0:用户空间, 1:系统空间）
	Type AppType `json:"type" gorm:"column:type;type:tinyint;default:0;index;comment:应用类型(0:用户空间,1:系统空间)"`
	
	// ⭐ app 级别的待审批权限申请数量
	PendingCount int `json:"pending_count" gorm:"column:pending_count;default:0;comment:app级别待审批的权限申请数量"`

	// ⭐ 仅展示有权限的空间：开启后，非管理员用户进入工作空间时，服务树只展示其有权限的目录（用于 SaaS 多租户场景）
	ShowOnlyPermitted bool `json:"show_only_permitted" gorm:"column:show_only_permitted;type:tinyint(1);default:0;comment:仅展示有权限的空间(0:否,1:是)"`
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

// IsSystemApp 检查是否为系统空间
func (a *App) IsSystemApp() bool {
	return a.Type.IsSystem()
}

// IsUserApp 检查是否为用户空间
func (a *App) IsUserApp() bool {
	return a.Type.IsUser()
}
