-- ============================================
-- 修复 hr-server 数据库所有表的 AUTO_INCREMENT 问题
-- 
-- 使用方法：
--   mysql -uroot -proot "hr-server" < scripts/migration/fix_hr_server_auto_increment.sql
-- ============================================

-- 临时禁用外键检查
SET FOREIGN_KEY_CHECKS = 0;

-- 修复所有表的主键 AUTO_INCREMENT
-- 注意：如果表已存在数据，需要先确保主键值正确

-- 1. user 表
ALTER TABLE `user` 
  MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT;

-- 2. user_session 表
ALTER TABLE `user_session` 
  MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT;

-- 3. email_verification 表
ALTER TABLE `email_verification` 
  MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT;

-- 4. email_code 表
ALTER TABLE `email_code` 
  MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT;

-- 5. department 表（如果存在）
ALTER TABLE `department` 
  MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT;

-- 重新启用外键检查
SET FOREIGN_KEY_CHECKS = 1;

SELECT '✅ 所有表的主键 AUTO_INCREMENT 已修复' AS message;
