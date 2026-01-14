package model

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// HubSnapshot Hub 快照模型
// 记录每个版本的完整目录结构，用于版本管理和历史回滚
type HubSnapshot struct {
	models.Base

	// 关联的 Hub 目录
	HubDirectoryID int64 `gorm:"index;not null" json:"hub_directory_id"`
	HubDirectory   *HubDirectory `json:"-" gorm:"foreignKey:HubDirectoryID;references:ID"`

	// 版本信息
	Version    string `gorm:"type:varchar(50);not null;index" json:"version"` // 版本号（如 v1.0.0）
	VersionNum int    `gorm:"not null;index" json:"version_num"`             // 版本号数字部分

	// 快照时间
	SnapshotAt time.Time `gorm:"index" json:"snapshot_at"` // 快照创建时间

	// 快照统计
	DirectoryCount int `gorm:"default:0" json:"directory_count"` // 目录数量
	FileCount       int `gorm:"default:0" json:"file_count"`      // 文件数量
	FunctionCount   int `gorm:"default:0" json:"function_count"`  // 函数数量

	// 快照元数据（JSON格式，存储完整的树结构，用于快速预览）
	SnapshotData string `gorm:"type:json" json:"snapshot_data"` // 完整的 ServiceTree 结构（JSON）

	// 是否为当前版本
	IsCurrent bool `gorm:"default:false;index" json:"is_current"` // 是否为当前版本（用于快速查询）

	// 快照描述（可选）
	Description string `gorm:"type:text" json:"description"` // 快照描述（如：修复了某个bug）
}

func (HubSnapshot) TableName() string {
	return "hub_snapshots"
}

