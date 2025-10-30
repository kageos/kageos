package model

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// App 应用信息表
type App struct {
	models.Base
	User        string    `gorm:"size:100;not null;index" json:"user"`               // 用户名
	App         string    `gorm:"size:100;not null;index" json:"app"`                // 应用名
	Version     string    `gorm:"size:50;not null" json:"version"`                   // 当前版本
	Status      string    `gorm:"size:20;not null;default:'inactive'" json:"status"` // 应用状态: inactive(未激活), active(已激活)
	ContainerID string    `gorm:"size:100" json:"container_id"`                      // 容器ID
	StartTime   time.Time `json:"start_time"`                                        // 启动时间
	LastSeen    time.Time `json:"last_seen"`                                         // 最后发现时间
}

// TableName 指定表名
func (App) TableName() string {
	return "apps"
}

// AppVersion 应用版本历史表
type AppVersion struct {
	models.Base
	User        string     `gorm:"size:100;not null;index" json:"user"` // 用户名
	App         string     `gorm:"size:100;not null;index" json:"app"`  // 应用名
	Version     string     `gorm:"size:50;not null" json:"version"`     // 版本号
	ContainerID string     `gorm:"size:100" json:"container_id"`        // 容器ID
	ProcessID   int        `json:"process_id"`                          // 进程ID
	StartTime   time.Time  `json:"start_time"`                          // 该版本启动时间
	StopTime    *time.Time `json:"stop_time"`                           // 停止时间
	LastSeen    time.Time  `json:"last_seen"`                           // 最后发现时间
}

// TableName 指定表名
func (AppVersion) TableName() string {
	return "app_versions"
}

// GetKey 获取应用唯一标识
func (a *App) GetKey() string {
	return a.User + "/" + a.App
}

// IsInactive 检查应用是否为未激活状态
func (a *App) IsInactive() bool {
	return a.Status == "inactive"
}

// IsActive 检查应用是否为已激活状态
func (a *App) IsActive() bool {
	return a.Status == "active"
}
