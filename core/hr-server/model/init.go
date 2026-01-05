package model

import (
	"gorm.io/gorm"
)

// InitModels 初始化所有模型（自动迁移）
func InitModels(db *gorm.DB) error {
	// 迁移用户相关表
	err := db.AutoMigrate(
		&User{},
		&UserSession{},
		&EmailCode{},
		&EmailVerification{},
		&Department{}, // ⭐ 新增：部门表
	)
	if err != nil {
		return err
	}

	return nil
}

