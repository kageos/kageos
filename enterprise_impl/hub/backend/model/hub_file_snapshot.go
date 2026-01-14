package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// HubFileSnapshot Hub 文件快照模型
// 记录每个版本的文件内容，用于版本管理和历史回滚
type HubFileSnapshot struct {
	models.Base

	// 关联的快照
	HubSnapshotID int64 `gorm:"index;not null" json:"hub_snapshot_id"`
	HubSnapshot   *HubSnapshot `json:"-" gorm:"foreignKey:HubSnapshotID;references:ID"`

	// 关联的服务树节点（package 类型，文件所属的目录）
	HubServiceTreeID int64 `gorm:"index" json:"hub_service_tree_id"`

	// 文件信息
	FileName     string `gorm:"type:varchar(255);not null" json:"file_name"`     // 文件名（不含扩展名）
	RelativePath string `gorm:"type:varchar(500);not null" json:"relative_path"` // 文件相对路径（如：user.go 或 subdir/user.go）
	FileType     string `gorm:"type:varchar(50);not null" json:"file_type"`     // 文件类型（go, json, yaml等）
	Content      string `gorm:"type:longtext;not null" json:"content"`          // 文件内容

	// 版本信息
	FileVersion    string `gorm:"type:varchar(50)" json:"file_version"`     // 文件版本号
	FileVersionNum int    `gorm:"default:0" json:"file_version_num"`      // 文件版本号数字部分

	// 元数据
	FileSize    int    `gorm:"default:0" json:"file_size"`        // 文件大小（字节）
	ContentHash string `gorm:"type:varchar(64);index" json:"content_hash"` // 内容hash（用于去重）
}

func (HubFileSnapshot) TableName() string {
	return "hub_file_snapshots"
}

