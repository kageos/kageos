package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// HubDirectoryStar 目录星星记录（类似 GitHub star）
// 用于统计 star_count 及「当前用户是否已 star」
type HubDirectoryStar struct {
	models.Base
	HubDirectoryID int64  `gorm:"uniqueIndex:idx_hub_dir_user;not null" json:"hub_directory_id"`
	Username       string `gorm:"type:varchar(100);uniqueIndex:idx_hub_dir_user;not null" json:"username"`
}

func (HubDirectoryStar) TableName() string {
	return "hub_directory_stars"
}
