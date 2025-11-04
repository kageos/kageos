package model

import (
	"gorm.io/gorm"
)

// InitTables 初始化表结构
func InitTables(db *gorm.DB) error {
	// 自动迁移表结构（预留，暂不启用外键约束）
	return db.AutoMigrate(
		&FileMetadata{},
		&FileReference{},
		&FileUpload{},
		&FileDownload{},
	)
}

