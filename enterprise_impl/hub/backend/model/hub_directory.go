package model

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// HubDirectory Hub 目录模型
// 存储目录的元信息和层级关系
type HubDirectory struct {
	models.Base

	// 基本信息
	Name        string `gorm:"type:varchar(255);not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	Category    string `gorm:"type:varchar(50)" json:"category"`
	Tags        string `gorm:"type:text" json:"tags"` // 逗号分隔的标签

	// 目录路径信息
	PackagePath   string `gorm:"type:varchar(500);not null;index" json:"package_path"`   // 目录路径，如：plugins/cashier
	FullCodePath  string `gorm:"type:varchar(500);not null;index" json:"full_code_path"`  // 完整代码路径，如：/user/app/plugins/cashier
	ParentDirID   int64  `gorm:"index" json:"parent_dir_id"`                              // 父目录ID（0表示根目录）

	// 源信息
	SourceUser          string `gorm:"type:varchar(100);not null" json:"source_user"`
	SourceApp           string `gorm:"type:varchar(100);not null" json:"source_app"`
	SourceDirectoryPath string `gorm:"type:varchar(500);not null" json:"source_directory_path"` // 源目录完整路径

	// 发布信息
	PublisherUsername string     `gorm:"type:varchar(100)" json:"publisher_username"` // 发布者用户名（OS 用户）
	APIKeyID         int64      `gorm:"index" json:"api_key_id"`                    // API Key ID（私有化用户）
	PublishedAt      *time.Time `json:"published_at"`

	// 服务费信息
	ServiceFeePersonal   float64 `gorm:"type:decimal(10,2)" json:"service_fee_personal"`   // 个人用户服务费
	ServiceFeeEnterprise float64 `gorm:"type:decimal(10,2)" json:"service_fee_enterprise"` // 企业用户服务费

	// 统计信息
	DownloadCount int     `gorm:"default:0" json:"download_count"`
	TrialCount    int     `gorm:"default:0" json:"trial_count"`
	Rating        float64 `gorm:"type:decimal(3,2)" json:"rating"`

	// 版本信息
	Version    string `gorm:"type:varchar(50);not null" json:"version"` // 目录版本号（如 v1）
	VersionNum int    `gorm:"not null" json:"version_num"`              // 版本号数字部分

	// 目录结构信息（JSON格式，用于快速预览）
	DirectoryTree string `gorm:"type:json" json:"directory_tree"` // 目录树结构（可选）

	// 统计信息（快照）
	DirectoryCount int `gorm:"default:0" json:"directory_count"` // 子目录数量
	FileCount      int `gorm:"default:0" json:"file_count"`      // 文件数量
	FunctionCount  int `gorm:"default:0" json:"function_count"`   // 函数数量
}

func (HubDirectory) TableName() string {
	return "hub_directories"
}

