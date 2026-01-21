package model

import (
	"gorm.io/gorm"
)

// InitTables 初始化数据库表
func InitTables(db *gorm.DB) error {
	// 自动迁移表结构
	return db.AutoMigrate(
		&HubDirectory{},           // Hub 目录
		&HubServiceTree{},         // Hub 服务树
		&HubSnapshot{},            // Hub 快照
		&HubServiceTreeSnapshot{}, // Hub 服务树快照
		&HubFileSnapshot{},        // Hub 文件快照
		// TODO: 后续添加其他表
		// &CodeGenerationLog{},
		// &ServiceFeePayment{},
		// &ServiceRecord{},
	)
}

