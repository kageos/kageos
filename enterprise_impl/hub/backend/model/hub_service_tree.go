package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

const (
	HubServiceTreeTypePackage  = "package"
	HubServiceTreeTypeFunction = "function"
)

// HubServiceTree Hub 服务树模型（类似 app-server 的 ServiceTree）
// 用于存储 Hub 目录的完整树结构，包括目录（package）和函数（function）节点
type HubServiceTree struct {
	models.Base

	// 关联的 Hub 目录
	HubDirectoryID int64 `gorm:"index;not null" json:"hub_directory_id"`
	HubDirectory   *HubDirectory `json:"-" gorm:"foreignKey:HubDirectoryID;references:ID"`

	// 节点信息（类似 ServiceTree）
	Name          string `gorm:"type:varchar(255)" json:"name"`
	Code          string `gorm:"type:varchar(255)" json:"code"`
	ParentID      int64  `gorm:"index;default:0" json:"parent_id"` // 父节点ID（0表示根节点）
	Type          string `gorm:"type:varchar(50);not null" json:"type"` // "package" 或 "function"
	Description   string `gorm:"type:text" json:"description"`
	FullCodePath  string `gorm:"type:varchar(500);index" json:"full_code_path"` // 完整代码路径

	// 引用信息
	RefID int64 `gorm:"default:0" json:"ref_id"` // 指向真实的 package 或 function ID（在源应用中）

	// 版本信息
	Version    string `gorm:"type:varchar(50)" json:"version"` // 节点版本号
	VersionNum int    `gorm:"default:0" json:"version_num"`    // 版本号数字部分

	// 快照版本（用于版本管理）
	SnapshotVersion    string `gorm:"type:varchar(50);index" json:"snapshot_version"`     // 快照版本号（如 v1.0.0）
	SnapshotVersionNum int    `gorm:"index" json:"snapshot_version_num"`                  // 快照版本号数字部分

	// 元数据
	TemplateType string `gorm:"type:varchar(50)" json:"template_type"` // 函数类型（仅 function 类型有效）
	Tags         string `gorm:"type:text" json:"tags"`                 // 标签（逗号分隔）

	// 层级关系（用于快速查询）
	Level int `gorm:"default:0" json:"level"` // 节点层级（根节点为0）
}

func (HubServiceTree) TableName() string {
	return "hub_service_tree"
}

// IsPackage 判断是否为 package 类型
func (hst *HubServiceTree) IsPackage() bool {
	return hst.Type == HubServiceTreeTypePackage
}

// IsFunction 判断是否为 function 类型
func (hst *HubServiceTree) IsFunction() bool {
	return hst.Type == HubServiceTreeTypeFunction
}

// IsRoot 判断是否为根节点
func (hst *HubServiceTree) IsRoot() bool {
	return hst.ParentID == 0
}

