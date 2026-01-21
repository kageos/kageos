-- ============================================
-- 修复 agent-server 数据库所有表的主键索引
-- 问题：表可能使用了旧的主键定义（primary_key），导致外键约束无法创建
-- 解决：确保所有表都有正确的主键索引
-- 
-- 使用方法：
--   mysql -uroot -proot "agent-server" < scripts/migration/fix_agent_server_primary_keys.sql
-- ============================================

-- 临时禁用外键检查
SET FOREIGN_KEY_CHECKS = 0;

-- 使用存储过程来安全地删除和重建主键
DELIMITER $$

DROP PROCEDURE IF EXISTS fix_primary_key$$
CREATE PROCEDURE fix_primary_key(IN table_name VARCHAR(64), IN column_name VARCHAR(64))
BEGIN
    DECLARE CONTINUE HANDLER FOR SQLEXCEPTION BEGIN END;
    
    -- 检查表是否存在
    SET @table_exists = (
        SELECT COUNT(*) 
        FROM INFORMATION_SCHEMA.TABLES 
        WHERE TABLE_SCHEMA = DATABASE() 
        AND TABLE_NAME = table_name
    );
    
    IF @table_exists > 0 THEN
        -- 尝试删除主键（如果存在）
        SET @sql = CONCAT('ALTER TABLE `', table_name, '` DROP PRIMARY KEY');
        PREPARE stmt FROM @sql;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
        
        -- 添加主键
        SET @sql = CONCAT('ALTER TABLE `', table_name, '` ADD PRIMARY KEY (`', column_name, '`)');
        PREPARE stmt FROM @sql;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
        
        SELECT CONCAT('✅ 已修复表: ', table_name) AS message;
    ELSE
        SELECT CONCAT('⚠️  表不存在: ', table_name) AS message;
    END IF;
END$$

DELIMITER ;

-- 修复所有表的主键（按依赖顺序）
-- 第一层：基础表（不被其他表引用）
CALL fix_primary_key('plugins', 'id');
CALL fix_primary_key('knowledge_bases', 'id');
CALL fix_primary_key('llm_configs', 'id');

-- 第二层：依赖基础表的表
CALL fix_primary_key('agents', 'id');
CALL fix_primary_key('knowledge_documents', 'id');
CALL fix_primary_key('knowledge_chunks', 'id');

-- 第三层：依赖 Agent 的表
CALL fix_primary_key('agent_chat_sessions', 'id');
CALL fix_primary_key('function_gen_records', 'id');
CALL fix_primary_key('function_group_agents', 'id');

-- 第四层：依赖 AgentChatSession 的表
CALL fix_primary_key('agent_chat_messages', 'id');

-- 删除临时存储过程
DROP PROCEDURE IF EXISTS fix_primary_key;

-- 重新启用外键检查
SET FOREIGN_KEY_CHECKS = 1;

SELECT '✅ 所有主键索引已修复，请重启应用让 GORM 自动创建外键约束' AS message;
