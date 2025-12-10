-- ============================================
-- 修复外键约束重复问题（Navicat 专用版）
-- 说明：删除已存在的外键约束，让 GORM AutoMigrate 重新创建
-- 使用方法：在 Navicat 中按顺序执行以下 SQL
-- ============================================

-- ============================================
-- 第一步：查看当前所有外键约束
-- ============================================
SELECT 
    TABLE_NAME AS '表名',
    CONSTRAINT_NAME AS '约束名',
    REFERENCED_TABLE_NAME AS '引用表',
    REFERENCED_COLUMN_NAME AS '引用列'
FROM 
    INFORMATION_SCHEMA.KEY_COLUMN_USAGE
WHERE 
    TABLE_SCHEMA = DATABASE()
    AND REFERENCED_TABLE_NAME IS NOT NULL
    AND TABLE_NAME IN ('service_tree', 'function', 'source_code', 'package')
ORDER BY TABLE_NAME, CONSTRAINT_NAME;

-- ============================================
-- 第二步：根据上面的查询结果，执行对应的删除语句
-- 注意：请根据第一步查询出的实际约束名，取消注释并执行对应的语句
-- ============================================

-- 删除 service_tree 表的外键约束
-- 请将下面的 'fk_service_tree_app' 替换为第一步查询出的实际约束名
-- ALTER TABLE `service_tree` DROP FOREIGN KEY `fk_service_tree_app`;

-- 删除 function 表的外键约束
-- 请将下面的 'fk_function_app' 替换为第一步查询出的实际约束名
-- ALTER TABLE `function` DROP FOREIGN KEY `fk_function_app`;

-- 删除 source_code 表的外键约束
-- 请将下面的 'fk_source_code_app' 替换为第一步查询出的实际约束名
-- ALTER TABLE `source_code` DROP FOREIGN KEY `fk_source_code_app`;

-- 删除 package 表的外键约束（如果存在）
-- 请将下面的 'fk_package_app' 替换为第一步查询出的实际约束名
-- ALTER TABLE `package` DROP FOREIGN KEY `fk_package_app`;

-- ============================================
-- 第三步：验证外键约束已删除（可选）
-- ============================================
-- 执行第一步的查询语句，确认外键约束已删除

-- ============================================
-- 第四步：重启应用
-- 重启应用后，GORM AutoMigrate 会自动重新创建外键约束
-- ============================================

