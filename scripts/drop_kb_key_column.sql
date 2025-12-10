-- ============================================
-- 删除 kb_key 字段（该字段已被移除，但数据库表中仍存在）
-- 说明：需要从多个表中删除 kb_key 字段
-- 使用方法：在数据库管理工具中执行此脚本
-- ============================================

-- 1. 删除 knowledge_bases 表的 kb_key 字段
ALTER TABLE `knowledge_bases` DROP COLUMN IF EXISTS `kb_key`;

-- 2. 删除 agents 表的 kb_key 字段（如果存在）
ALTER TABLE `agents` DROP COLUMN IF EXISTS `kb_key`;

-- 3. 删除 knowledge_documents 表的 kb_key 字段（如果存在）
ALTER TABLE `knowledge_documents` DROP COLUMN IF EXISTS `kb_key`;

-- 4. 删除 knowledge_chunks 表的 kb_key 字段（如果存在）
ALTER TABLE `knowledge_chunks` DROP COLUMN IF EXISTS `kb_key`;

-- 如果还有索引，也需要删除（如果有的话）
-- ALTER TABLE `knowledge_bases` DROP INDEX IF EXISTS `idx_kb_key`;
-- ALTER TABLE `agents` DROP INDEX IF EXISTS `idx_kb_key`;
-- ALTER TABLE `knowledge_documents` DROP INDEX IF EXISTS `idx_kb_key`;
-- ALTER TABLE `knowledge_chunks` DROP INDEX IF EXISTS `idx_kb_key`;

-- 验证：查看表结构确认字段已删除
-- DESCRIBE `knowledge_bases`;
-- DESCRIBE `agents`;
-- DESCRIBE `knowledge_documents`;
-- DESCRIBE `knowledge_chunks`;

