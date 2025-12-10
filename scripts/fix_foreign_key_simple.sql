-- ============================================
-- 修复外键约束重复问题（简化版）
-- 说明：删除已存在的外键约束，让 GORM AutoMigrate 重新创建
-- 使用方法：在 Navicat 中执行此脚本
-- ============================================

-- 第一步：查看当前所有外键约束（用于确认）
SELECT 
    TABLE_NAME,
    CONSTRAINT_NAME,
    REFERENCED_TABLE_NAME,
    REFERENCED_COLUMN_NAME
FROM 
    INFORMATION_SCHEMA.KEY_COLUMN_USAGE
WHERE 
    TABLE_SCHEMA = DATABASE()
    AND REFERENCED_TABLE_NAME IS NOT NULL
    AND TABLE_NAME IN ('service_tree', 'function', 'source_code', 'package')
ORDER BY TABLE_NAME, CONSTRAINT_NAME;

-- 第二步：根据上面的查询结果，删除对应的外键约束
-- 如果上面的查询结果中有约束，请取消注释下面的对应行并执行

-- 删除 service_tree 表的外键约束（请根据实际约束名修改）
-- ALTER TABLE `service_tree` DROP FOREIGN KEY `fk_service_tree_app`;

-- 删除 function 表的外键约束（请根据实际约束名修改）
-- ALTER TABLE `function` DROP FOREIGN KEY `fk_function_app`;

-- 删除 source_code 表的外键约束（请根据实际约束名修改）
-- ALTER TABLE `source_code` DROP FOREIGN KEY `fk_source_code_app`;

-- 删除 package 表的外键约束（请根据实际约束名修改）
-- ALTER TABLE `package` DROP FOREIGN KEY `fk_package_app`;

-- 第三步：如果上面的查询结果中有多个约束名，请逐一删除
-- 例如，如果 service_tree 表有多个外键约束，需要分别删除：
-- ALTER TABLE `service_tree` DROP FOREIGN KEY `约束名1`;
-- ALTER TABLE `service_tree` DROP FOREIGN KEY `约束名2`;

-- 第四步：删除后，再次执行第一步的查询，确认外键约束已删除
-- 然后重启应用，让 GORM AutoMigrate 重新创建外键约束

