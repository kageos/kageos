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

	// 该版本的详情（切换版本时展示此版本当时的 name/description 等，不读目录表）
	Name                 string  `gorm:"type:varchar(255)" json:"name"`
	DetailDescription    string  `gorm:"type:text" json:"detail_description"` // 该版本时的目录描述（与 Description 本版本更新说明 区分）
	Category             string  `gorm:"type:varchar(50)" json:"category"`
	Tags                 string  `gorm:"type:text" json:"tags"`
	ServiceFeePersonal   float64 `gorm:"type:decimal(10,2)" json:"service_fee_personal"`
	ServiceFeeEnterprise float64 `gorm:"type:decimal(10,2)" json:"service_fee_enterprise"`
	PublisherUsername    string  `gorm:"type:varchar(100)" json:"publisher_username"` // 该版本的上传人（发布/推送时的用户）

	// 快照元数据（JSON格式，历史存量兜底字段；新增链路以三字段为准）
	SnapshotData string `gorm:"type:json" json:"snapshot_data"` // 历史存量快照兜底，不再作为当前安装/导出协议单源

	// 快照三字段：各司其职，单源
	SnapshotTree       string `gorm:"type:json" json:"snapshot_tree"`        // 目录结构（展示用：树/列表/面包屑），不含文件 content、不含函数详情
	SnapshotFiles      string `gorm:"type:json" json:"snapshot_files"`        // 文件列表（复制用：按 relative_path 写文件），平铺 []SnapshotFileEntry
	SnapshotFunctionDefs string `gorm:"type:json" json:"snapshot_function_defs"` // 函数定义列表（预览用：入参、描述等），平铺 []HubFunctionInfo

	// 是否为当前版本
	IsCurrent bool `gorm:"default:false;index" json:"is_current"` // 是否为当前版本（用于快速查询）

	// 快照描述（可选）
	Description string `gorm:"type:text" json:"description"` // 快照描述（如：修复了某个bug）
}

func (HubSnapshot) TableName() string {
	return "hub_snapshots"
}
