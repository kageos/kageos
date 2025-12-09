-- ============================================
-- 检查并删除 kb_key 字段
-- 说明：先检查哪些表有 kb_key 字段，然后删除
-- 使用方法：在数据库管理工具中执行此脚本
-- ============================================

-- 第一步：检查哪些表有 kb_key 字段
SELECT 
    TABLE_NAME,
    COLUMN_NAME,
    COLUMN_TYPE,
    IS_NULLABLE,
    COLUMN_DEFAULT
FROM 
    INFORMATION_SCHEMA.COLUMNS
WHERE 
    TABLE_SCHEMA = DATABASE()
    AND COLUMN_NAME = 'kb_key'
ORDER BY TABLE_NAME;

-- 第二步：根据上面的查询结果，删除对应表的 kb_key 字段
-- 如果 agents 表有 kb_key 字段，执行：
ALTER TABLE `agents` DROP COLUMN `kb_key`;

-- 如果 knowledge_documents 表有 kb_key 字段，执行：
-- ALTER TABLE `knowledge_documents` DROP COLUMN `kb_key`;

-- 如果 knowledge_chunks 表有 kb_key 字段，执行：
-- ALTER TABLE `knowledge_chunks` DROP COLUMN `kb_key`;

-- 第三步：检查索引（如果有的话）
SELECT 
    TABLE_NAME,
    INDEX_NAME,
    COLUMN_NAME
FROM 
    INFORMATION_SCHEMA.STATISTICS
WHERE 
    TABLE_SCHEMA = DATABASE()
    AND COLUMN_NAME = 'kb_key'
ORDER BY TABLE_NAME, INDEX_NAME;

-- 第四步：删除索引（根据上面的查询结果，取消注释对应的行）
-- ALTER TABLE `agents` DROP INDEX `索引名`;
-- ALTER TABLE `knowledge_documents` DROP INDEX `索引名`;
-- ALTER TABLE `knowledge_chunks` DROP INDEX `索引名`;

-- 第五步：验证字段已删除
DESCRIBE `agents`;
-- DESCRIBE `knowledge_documents`;
-- DESCRIBE `knowledge_chunks`;

