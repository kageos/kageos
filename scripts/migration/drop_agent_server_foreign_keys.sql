-- ============================================
-- 删除 agent-server 数据库的所有外键约束
-- 说明：删除所有外键约束，让 GORM AutoMigrate 自动重建
-- 使用方法：
--   mysql -uroot -proot agent_server_db < scripts/migration/drop_agent_server_foreign_keys.sql
-- 或者根据配置文件中的数据库名：
--   mysql -uroot -proot "agent-server" < scripts/migration/drop_agent_server_foreign_keys.sql
-- ============================================

-- 第一步：查询所有需要删除的外键约束（用于确认）
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
    AND TABLE_NAME IN (
        'agents',
        'knowledge_documents',
        'knowledge_chunks',
        'agent_chat_sessions',
        'function_gen_records',
        'function_group_agents',
        'agent_chat_messages'
    )
ORDER BY TABLE_NAME, CONSTRAINT_NAME;

-- 第二步：生成删除外键约束的 SQL 语句
SELECT 
    CONCAT('ALTER TABLE `', TABLE_NAME, '` DROP FOREIGN KEY `', CONSTRAINT_NAME, '`;') AS '执行语句'
FROM 
    INFORMATION_SCHEMA.KEY_COLUMN_USAGE
WHERE 
    TABLE_SCHEMA = DATABASE()
    AND REFERENCED_TABLE_NAME IS NOT NULL
    AND TABLE_NAME IN (
        'agents',
        'knowledge_documents',
        'knowledge_chunks',
        'agent_chat_sessions',
        'function_gen_records',
        'function_group_agents',
        'agent_chat_messages'
    )
ORDER BY TABLE_NAME, CONSTRAINT_NAME;

-- ============================================
-- 第三步：手动执行上面生成的 SQL 语句
-- 或者使用下面的存储过程自动删除
-- ============================================

DELIMITER $$

DROP PROCEDURE IF EXISTS drop_all_foreign_keys$$
CREATE PROCEDURE drop_all_foreign_keys()
BEGIN
    DECLARE done INT DEFAULT FALSE;
    DECLARE table_name_var VARCHAR(64);
    DECLARE constraint_name_var VARCHAR(64);
    DECLARE sql_stmt TEXT;
    
    -- 游标：查询所有外键约束
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
                'agents',
                'knowledge_documents',
                'knowledge_chunks',
                'agent_chat_sessions',
                'function_gen_records',
                'function_group_agents',
                'agent_chat_messages'
            );
    
    DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = TRUE;
    DECLARE CONTINUE HANDLER FOR SQLEXCEPTION 
    BEGIN
        -- 忽略错误，继续执行
    END;
    
    -- 临时禁用外键检查
    SET FOREIGN_KEY_CHECKS = 0;
    
    OPEN cur;
    
    read_loop: LOOP
        FETCH cur INTO table_name_var, constraint_name_var;
        IF done THEN
            LEAVE read_loop;
        END IF;
        
        -- 构建删除外键的 SQL
        SET sql_stmt = CONCAT('ALTER TABLE `', table_name_var, '` DROP FOREIGN KEY `', constraint_name_var, '`');
        
        -- 执行删除
        SET @sql = sql_stmt;
        PREPARE stmt FROM @sql;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
        
        SELECT CONCAT('已删除外键: ', table_name_var, '.', constraint_name_var) AS message;
    END LOOP;
    
    CLOSE cur;
    
    -- 重新启用外键检查
    SET FOREIGN_KEY_CHECKS = 1;
    
    SELECT '所有外键约束已删除，请重启应用让 GORM 自动重建' AS message;
END$$

DELIMITER ;

-- 执行存储过程（自动删除所有外键）
CALL drop_all_foreign_keys();

-- 删除临时存储过程
DROP PROCEDURE IF EXISTS drop_all_foreign_keys;
