-- ============================================
-- 删除 agent-server 数据库的所有外键约束（简化版）
-- 说明：直接删除所有外键约束，让 GORM AutoMigrate 自动重建
-- 使用方法：
--   mysql -uroot -proot agent_server_db < scripts/migration/drop_agent_server_foreign_keys_simple.sql
-- 或者根据配置文件中的数据库名：
--   mysql -uroot -proot "agent-server" < scripts/migration/drop_agent_server_foreign_keys_simple.sql
-- ============================================

-- 临时禁用外键检查
SET FOREIGN_KEY_CHECKS = 0;

-- 查询并生成删除外键的 SQL 语句
-- 执行此查询后，复制结果中的 SQL 语句并执行
SELECT 
    CONCAT('ALTER TABLE `', TABLE_NAME, '` DROP FOREIGN KEY `', CONSTRAINT_NAME, '`;') AS '执行语句'
FROM 
    INFORMATION_SCHEMA.KEY_COLUMN_USAGE
WHERE 
    TABLE_SCHEMA = DATABASE()
    AND REFERENCED_TABLE_NAME IS NOT NULL
    AND TABLE_NAME IN (
        'agents',
        'knowledge_documents',
        'knowledge_chunks',
        'agent_chat_sessions',
        'function_gen_records',
        'function_group_agents',
        'agent_chat_messages'
    )
ORDER BY TABLE_NAME, CONSTRAINT_NAME;

-- 重新启用外键检查
SET FOREIGN_KEY_CHECKS = 1;

-- ============================================
-- 说明：
-- 1. 执行上面的查询，会生成所有需要删除的外键约束的 SQL 语句
-- 2. 复制查询结果中的 SQL 语句，在同一个会话中执行
-- 3. 或者使用上面的 drop_agent_server_foreign_keys.sql 脚本（自动删除）
-- 4. 删除完成后，重启应用，GORM 会自动重建外键约束
-- ============================================
