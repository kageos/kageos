package model

import (
	"gorm.io/gorm"
)

// InitTables 初始化表结构
func InitTables(db *gorm.DB) error {
	// 自动迁移表结构
	return db.AutoMigrate(
		&FileUpload{},
		&FileDownload{},
	)
}

