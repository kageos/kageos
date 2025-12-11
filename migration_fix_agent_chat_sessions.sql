-- ============================================
-- 修复 agent_chat_sessions 表外键约束问题
-- ============================================
-- 
-- 问题：agent_chat_sessions 表中存在无效的 agent_id（为 0 或指向不存在的智能体）
-- 导致无法添加外键约束
--
-- 解决步骤：
-- 1. 查找并删除旧的外键约束（如果存在）
-- 2. 清理无效的 agent_id 数据
-- 3. 重新运行应用，让 GORM AutoMigrate 自动创建外键约束
-- ============================================

-- 步骤 1: 查找外键约束名称
-- SELECT CONSTRAINT_NAME 
-- FROM information_schema.KEY_COLUMN_USAGE 
-- WHERE TABLE_SCHEMA = DATABASE() 
-- AND TABLE_NAME = 'agent_chat_sessions' 
-- AND COLUMN_NAME = 'agent_id' 
-- AND REFERENCED_TABLE_NAME = 'agents';

-- 步骤 2: 删除旧的外键约束（如果存在，将 CONSTRAINT_NAME 替换为上面查询的结果）
-- ALTER TABLE agent_chat_sessions DROP FOREIGN KEY fk_agent_chat_sessions_agent;

-- 步骤 3: 更新无效的 agent_id 数据（不删除数据）
-- 将 agent_id 为 0 的记录更新为有效的 agent_id（例如：3）
-- 请根据实际情况修改 agent_id 的值
UPDATE agent_chat_sessions SET agent_id = 3 WHERE agent_id = 0;

-- 将 agent_id 指向不存在智能体的记录更新为有效的 agent_id（例如：3）
-- 请根据实际情况修改 agent_id 的值
UPDATE agent_chat_sessions s
LEFT JOIN agents a ON s.agent_id = a.id
SET s.agent_id = 3
WHERE s.agent_id != 0 AND a.id IS NULL;

-- 步骤 4: 检查更新结果
-- SELECT COUNT(*) as invalid_count 
-- FROM agent_chat_sessions s
-- LEFT JOIN agents a ON s.agent_id = a.id
-- WHERE s.agent_id = 0 OR (s.agent_id != 0 AND a.id IS NULL);
-- 应该返回 0

-- 或者查看更新后的数据
-- SELECT id, session_id, agent_id, title FROM agent_chat_sessions WHERE agent_id = 3;

-- 步骤 5: 重新运行应用，GORM AutoMigrate 会自动创建外键约束

