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
		&Plugin{},        // 被 Agent 引用
		&KnowledgeBase{}, // 被 Agent、KnowledgeDocument、KnowledgeChunk 引用
		&LLMConfig{},     // 被 Agent 引用
		
		// 第二层：依赖基础表的表
		&Agent{},         // 引用 Plugin、KnowledgeBase、LLMConfig
		&KnowledgeDocument{}, // 引用 KnowledgeBase
		&KnowledgeChunk{},    // 引用 KnowledgeBase
		
		// 第三层：依赖 Agent 的表
		&AgentChatSession{}, // 引用 Agent
		&FunctionGenRecord{}, // 可能引用 Agent
		&FunctionGroupAgent{}, // 引用 Agent、FunctionGenRecord
		
		// 第四层：依赖 AgentChatSession 的表
		&AgentChatMessage{}, // 引用 AgentChatSession
	); err != nil {
		return err
	}

	return nil
}
