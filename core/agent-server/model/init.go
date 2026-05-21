package model

import (
	"context"
	"fmt"

	"github.com/kageos/kageos/pkg/logger"
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
		&WorkspaceHandoffPacket{},
	); err != nil {
		return err
	}

	// 根因修复：确保库与聊天消息表为 utf8mb4，否则中文/emoji 写入会报 Error 1366；失败则启动失败，必须修好再启
	if err := ensureDatabaseAndChatMessagesUtf8mb4(db); err != nil {
		return fmt.Errorf("数据库字符集必须为 utf8mb4 才能正常存储中文消息: %w（请用有 ALTER 权限的账号执行 ALTER DATABASE 库名 CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; 以及 ALTER TABLE agent_chat_messages CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; 或为当前库用户授予 ALTER 权限后重启）", err)
	}
	logger.Infof(context.TODO(), "[InitTables] 数据库与 agent_chat_messages 已确认为 utf8mb4")
	return nil
}

// ensureDatabaseAndChatMessagesUtf8mb4 设置库默认字符集为 utf8mb4，并将 agent_chat_messages 表转为 utf8mb4（根因解决 Error 1366）
func ensureDatabaseAndChatMessagesUtf8mb4(db *gorm.DB) error {
	var dbName string
	if err := db.Raw("SELECT DATABASE()").Scan(&dbName).Error; err != nil || dbName == "" {
		return fmt.Errorf("无法获取当前数据库名: %w", err)
	}
	// 设置库默认字符集，后续新建表会使用 utf8mb4
	if err := db.Exec(fmt.Sprintf("ALTER DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName)).Error; err != nil {
		return fmt.Errorf("ALTER DATABASE utf8mb4 失败: %w", err)
	}
	// 转换已有表，否则 content 存中文会报 Incorrect string value
	if err := db.Exec("ALTER TABLE agent_chat_messages CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci").Error; err != nil {
		return fmt.Errorf("ALTER TABLE agent_chat_messages utf8mb4 失败: %w", err)
	}
	return nil
}
