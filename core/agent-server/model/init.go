package model

import (
	"gorm.io/gorm"
)

// InitTables 初始化所有表
func InitTables(db *gorm.DB) error {
	// 自动迁移所有模型
	if err := db.AutoMigrate(
		&Agent{},
		&KnowledgeBase{},
		&KnowledgeDocument{},
		&KnowledgeChunk{},
		&LLMConfig{},
		&Task{},
		&AgentChatSession{},
		&AgentChatMessage{},
		&FunctionGenRecord{},
	); err != nil {
		return err
	}

	return nil
}
