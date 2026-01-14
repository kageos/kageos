package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// HubServiceTreeSnapshot Hub 服务树快照模型
// 记录每个版本的服务树节点状态，用于版本回滚和对比
type HubServiceTreeSnapshot struct {
	models.Base

	// 关联的快照
	HubSnapshotID int64 `gorm:"index;not null" json:"hub_snapshot_id"`
	HubSnapshot   *HubSnapshot `json:"-" gorm:"foreignKey:HubSnapshotID;references:ID"`

	// 关联的服务树节点（可选，如果节点已删除则为0）
	HubServiceTreeID int64 `gorm:"index" json:"hub_service_tree_id"`

	// 节点信息（快照时的完整状态）
	Name          string `gorm:"type:varchar(255)" json:"name"`
	Code          string `gorm:"type:varchar(255)" json:"code"`
	ParentID      int64  `gorm:"index;default:0" json:"parent_id"`
	Type          string `gorm:"type:varchar(50);not null" json:"type"` // "package" 或 "function"
	Description   string `gorm:"type:text" json:"description"`
	FullCodePath  string `gorm:"type:varchar(500);index" json:"full_code_path"`
	RefID         int64  `gorm:"default:0" json:"ref_id"`
	Version       string `gorm:"type:varchar(50)" json:"version"`
	VersionNum    int    `gorm:"default:0" json:"version_num"`
	TemplateType  string `gorm:"type:varchar(50)" json:"template_type"`
	Tags          string `gorm:"type:text" json:"tags"`
	Level         int    `gorm:"default:0" json:"level"`

	// 操作类型（用于版本对比）
	Operation string `gorm:"type:varchar(20)" json:"operation"` // "add", "update", "delete"
}

func (HubServiceTreeSnapshot) TableName() string {
	return "hub_service_tree_snapshots"
}

