-- ============================================
-- 修复 hub 数据库所有表的 AUTO_INCREMENT 问题
-- 
-- 使用方法：
--   mysql -uroot -proot hub < scripts/migration/fix_hub_auto_increment.sql
-- ============================================

-- 临时禁用外键检查
SET FOREIGN_KEY_CHECKS = 0;

-- 修复所有表的主键 AUTO_INCREMENT
-- 注意：如果表已存在数据，需要先确保主键值正确

-- 1. hub_directories
ALTER TABLE `hub_directories` 
  MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT;

-- 2. hub_service_tree
ALTER TABLE `hub_service_tree` 
  MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT;

-- 3. hub_snapshots
ALTER TABLE `hub_snapshots` 
  MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT;

-- 4. hub_service_tree_snapshots
ALTER TABLE `hub_service_tree_snapshots` 
  MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT;

-- 5. hub_file_snapshots
ALTER TABLE `hub_file_snapshots` 
  MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT;

-- 重新启用外键检查
SET FOREIGN_KEY_CHECKS = 1;

SELECT '✅ 所有表的主键 AUTO_INCREMENT 已修复' AS message;
