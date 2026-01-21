-- ============================================
-- 一键修复 hub 数据库的所有问题
-- 包括：删除所有外键约束 + 修复所有主键索引
-- 
-- 使用方法：
--   mysql -uroot -proot hub < scripts/migration/fix_hub_all.sql
-- ============================================

-- 临时禁用外键检查
SET FOREIGN_KEY_CHECKS = 0;

-- ============================================
-- 第一步：删除所有外键约束
-- ============================================

DELIMITER $$

DROP PROCEDURE IF EXISTS drop_all_foreign_keys$$
CREATE PROCEDURE drop_all_foreign_keys()
BEGIN
    DECLARE done INT DEFAULT FALSE;
    DECLARE table_name_var VARCHAR(64);
    DECLARE constraint_name_var VARCHAR(64);
    
    DECLARE cur CURSOR FOR
        SELECT 
            TABLE_NAME,
            CONSTRAINT_NAME
        FROM 
            INFORMATION_SCHEMA.KEY_COLUMN_USAGE
        WHERE 
            TABLE_SCHEMA = DATABASE()
            AND REFERENCED_TABLE_NAME IS NOT NULL
            AND TABLE_NAME IN (
                'hub_service_tree',
                'hub_snapshots',
                'hub_service_tree_snapshots',
                'hub_file_snapshots'
            );
    
    DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = TRUE;
    DECLARE CONTINUE HANDLER FOR SQLEXCEPTION BEGIN END;
    
    OPEN cur;
    
    read_loop: LOOP
        FETCH cur INTO table_name_var, constraint_name_var;
        IF done THEN
            LEAVE read_loop;
        END IF;
        
        SET @sql = CONCAT('ALTER TABLE `', table_name_var, '` DROP FOREIGN KEY `', constraint_name_var, '`');
        PREPARE stmt FROM @sql;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
    END LOOP;
    
    CLOSE cur;
END$$

DELIMITER ;

CALL drop_all_foreign_keys();
DROP PROCEDURE IF EXISTS drop_all_foreign_keys;

-- ============================================
-- 第二步：修复所有表的主键索引
-- ============================================

DELIMITER $$

DROP PROCEDURE IF EXISTS fix_primary_key$$
CREATE PROCEDURE fix_primary_key(IN table_name VARCHAR(64), IN column_name VARCHAR(64))
BEGIN
    DECLARE CONTINUE HANDLER FOR SQLEXCEPTION BEGIN END;
    
    SET @table_exists = (
        SELECT COUNT(*) 
        FROM INFORMATION_SCHEMA.TABLES 
        WHERE TABLE_SCHEMA = DATABASE() 
        AND TABLE_NAME = table_name
    );
    
    IF @table_exists > 0 THEN
        SET @sql = CONCAT('ALTER TABLE `', table_name, '` DROP PRIMARY KEY');
        PREPARE stmt FROM @sql;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
        
        SET @sql = CONCAT('ALTER TABLE `', table_name, '` ADD PRIMARY KEY (`', column_name, '`)');
        PREPARE stmt FROM @sql;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
    END IF;
END$$

DELIMITER ;

-- 修复所有表的主键（按依赖顺序）
-- 第一层：基础表（不被其他表引用）
CALL fix_primary_key('hub_directories', 'id');

-- 第二层：依赖 hub_directories 的表
CALL fix_primary_key('hub_service_tree', 'id');
CALL fix_primary_key('hub_snapshots', 'id');

-- 第三层：依赖 hub_snapshots 的表
CALL fix_primary_key('hub_service_tree_snapshots', 'id');
CALL fix_primary_key('hub_file_snapshots', 'id');

-- ============================================
-- 第三步：修复所有表的 AUTO_INCREMENT
-- ============================================

-- 修复所有表的主键 AUTO_INCREMENT
ALTER TABLE `hub_directories` 
  MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT;

ALTER TABLE `hub_service_tree` 
  MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT;

ALTER TABLE `hub_snapshots` 
  MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT;

ALTER TABLE `hub_service_tree_snapshots` 
  MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT;

ALTER TABLE `hub_file_snapshots` 
  MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT;

DROP PROCEDURE IF EXISTS fix_primary_key;

-- 重新启用外键检查
SET FOREIGN_KEY_CHECKS = 1;

SELECT '✅ 所有问题已修复：外键约束已删除，主键索引已修复，AUTO_INCREMENT 已设置' AS message;
SELECT '✅ 请重启应用，GORM 会自动重建外键约束' AS message;
