-- ============================================
-- 删除 kb_key 字段（MySQL 版本，不支持 IF EXISTS）
-- 说明：需要从多个表中删除 kb_key 字段
-- 使用方法：在数据库管理工具中执行此脚本
-- 注意：如果字段不存在，会报错，可以忽略
-- ============================================

-- 1. 删除 knowledge_bases 表的 kb_key 字段
ALTER TABLE `knowledge_bases` DROP COLUMN `kb_key`;

-- 2. 删除 agents 表的 kb_key 字段
ALTER TABLE `agents` DROP COLUMN `kb_key`;

-- 3. 删除 knowledge_documents 表的 kb_key 字段
ALTER TABLE `knowledge_documents` DROP COLUMN `kb_key`;

-- 4. 删除 knowledge_chunks 表的 kb_key 字段
ALTER TABLE `knowledge_chunks` DROP COLUMN `kb_key`;

-- 如果还有索引，也需要删除（如果有的话）
-- 先查看索引：
-- SHOW INDEX FROM `knowledge_bases` WHERE Key_name LIKE '%kb_key%';
-- SHOW INDEX FROM `agents` WHERE Key_name LIKE '%kb_key%';
-- SHOW INDEX FROM `knowledge_documents` WHERE Key_name LIKE '%kb_key%';
-- SHOW INDEX FROM `knowledge_chunks` WHERE Key_name LIKE '%kb_key%';

-- 然后删除索引（根据实际索引名修改）：
-- ALTER TABLE `knowledge_bases` DROP INDEX `idx_kb_key`;
-- ALTER TABLE `agents` DROP INDEX `idx_kb_key`;
-- ALTER TABLE `knowledge_documents` DROP INDEX `idx_kb_key`;
-- ALTER TABLE `knowledge_chunks` DROP INDEX `idx_kb_key`;

-- 验证：查看表结构确认字段已删除
-- DESCRIBE `knowledge_bases`;
-- DESCRIBE `agents`;
-- DESCRIBE `knowledge_documents`;
-- DESCRIBE `knowledge_chunks`;

