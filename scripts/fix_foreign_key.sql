-- ============================================
-- 修复外键约束重复问题
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

-- 第二步：删除 service_tree 表的外键约束
-- 注意：如果约束名不是 fk_service_tree_app，请根据上面的查询结果修改
SET @constraint_name = (
    SELECT CONSTRAINT_NAME 
    FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE 
    WHERE TABLE_SCHEMA = DATABASE() 
    AND TABLE_NAME = 'service_tree' 
    AND REFERENCED_TABLE_NAME = 'app'
    LIMIT 1
);

SET @sql = IF(@constraint_name IS NOT NULL, 
    CONCAT('ALTER TABLE `service_tree` DROP FOREIGN KEY `', @constraint_name, '`'), 
    'SELECT "service_tree 表没有指向 app 表的外键约束" AS message');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 第三步：删除 function 表的外键约束
SET @constraint_name = (
    SELECT CONSTRAINT_NAME 
    FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE 
    WHERE TABLE_SCHEMA = DATABASE() 
    AND TABLE_NAME = 'function' 
    AND REFERENCED_TABLE_NAME = 'app'
    LIMIT 1
);

SET @sql = IF(@constraint_name IS NOT NULL, 
    CONCAT('ALTER TABLE `function` DROP FOREIGN KEY `', @constraint_name, '`'), 
    'SELECT "function 表没有指向 app 表的外键约束" AS message');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 第四步：删除 source_code 表的外键约束
SET @constraint_name = (
    SELECT CONSTRAINT_NAME 
    FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE 
    WHERE TABLE_SCHEMA = DATABASE() 
    AND TABLE_NAME = 'source_code' 
    AND REFERENCED_TABLE_NAME = 'app'
    LIMIT 1
);

SET @sql = IF(@constraint_name IS NOT NULL, 
    CONCAT('ALTER TABLE `source_code` DROP FOREIGN KEY `', @constraint_name, '`'), 
    'SELECT "source_code 表没有指向 app 表的外键约束" AS message');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 第五步：删除 package 表的外键约束（如果存在）
SET @constraint_name = (
    SELECT CONSTRAINT_NAME 
    FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE 
    WHERE TABLE_SCHEMA = DATABASE() 
    AND TABLE_NAME = 'package' 
    AND REFERENCED_TABLE_NAME = 'app'
    LIMIT 1
);

SET @sql = IF(@constraint_name IS NOT NULL, 
    CONCAT('ALTER TABLE `package` DROP FOREIGN KEY `', @constraint_name, '`'), 
    'SELECT "package 表没有指向 app 表的外键约束" AS message');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 第六步：再次查看外键约束（确认已删除）
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

