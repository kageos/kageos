-- ============================================
-- 快速修复外键约束重复问题
-- 使用方法：在 Navicat 中直接执行此脚本
-- ============================================

-- 第一步：查询所有需要删除的外键约束
SELECT 
    CONCAT('ALTER TABLE `', TABLE_NAME, '` DROP FOREIGN KEY `', CONSTRAINT_NAME, '`;') AS '执行语句'
FROM 
    INFORMATION_SCHEMA.KEY_COLUMN_USAGE
WHERE 
    TABLE_SCHEMA = DATABASE()
    AND REFERENCED_TABLE_NAME = 'app'
    AND TABLE_NAME IN ('service_tree', 'function', 'source_code', 'package');

-- 第二步：复制上面查询结果中的 SQL 语句，在 Navicat 中执行
-- 例如，如果查询结果中有：
-- ALTER TABLE `service_tree` DROP FOREIGN KEY `fk_service_tree_app`;
-- 请复制并执行这条语句

-- 第三步：执行完所有删除语句后，重启应用即可

