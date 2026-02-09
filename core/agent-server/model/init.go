package model

import (
	"gorm.io/gorm"
)

// InitTables 初始化所有表
func InitTables(db *gorm.DB) error {
	// ⭐ 先创建被引用的表（父表），再创建引用它们的表（子表）
	// 这样可以确保外键约束能够正确创建
	if err := db.AutoMigrate(
		// 第一层：基础表（不被其他表引用）
		&LLMConfig{},
		// 工作台会话与消息（仅工作台使用，无智能体）
		&AgentChatSession{},
		&AgentChatMessage{},
		// 工作台模式（独立表，无外键）
		&WorkspaceMode{},
	); err != nil {
		return err
	}

	return nil
}
