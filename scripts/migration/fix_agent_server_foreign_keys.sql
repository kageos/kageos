-- 修复 agent-server 数据库外键约束问题
-- 问题：plugins 表可能使用了旧的主键定义，导致外键约束无法创建
-- 解决：确保所有表都有正确的主键索引
--
-- 使用方法：
-- mysql -uroot -proot agent_server_db < scripts/migration/fix_agent_server_foreign_keys.sql
-- 或者
-- mysql -uroot -proot -e "USE agent_server_db; SOURCE scripts/migration/fix_agent_server_foreign_keys.sql;"

-- 注意：根据配置文件，数据库名可能是 "agent-server" 而不是 "agent_server_db"
-- 请根据实际情况修改 USE 语句

-- 1. 临时禁用外键检查
SET FOREIGN_KEY_CHECKS = 0;

-- 2. 删除可能存在的旧外键约束（使用存储过程处理错误）
-- 注意：MySQL 不支持 IF EXISTS，需要手动检查或使用存储过程

-- 3. 确保所有表都有正确的主键索引
-- 如果表存在但没有主键，添加主键
-- 如果主键定义不正确，先删除再添加

-- plugins 表：确保主键存在
-- 如果表不存在，这些语句会失败，但不会影响后续的 GORM 迁移
-- 如果主键已存在，DROP PRIMARY KEY 会失败，但可以忽略

-- 使用存储过程来安全地删除和重建主键
DELIMITER $$

DROP PROCEDURE IF EXISTS fix_primary_key$$
CREATE PROCEDURE fix_primary_key(IN table_name VARCHAR(64), IN column_name VARCHAR(64))
BEGIN
    DECLARE CONTINUE HANDLER FOR SQLEXCEPTION BEGIN END;
    
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
END$$

DELIMITER ;

-- 修复各个表的主键
CALL fix_primary_key('plugins', 'id');
CALL fix_primary_key('knowledge_bases', 'id');
CALL fix_primary_key('llm_configs', 'id');
CALL fix_primary_key('agents', 'id');
CALL fix_primary_key('knowledge_documents', 'id');
CALL fix_primary_key('knowledge_chunks', 'id');
CALL fix_primary_key('agent_chat_sessions', 'id');
CALL fix_primary_key('function_gen_records', 'id');
CALL fix_primary_key('function_group_agents', 'id');
CALL fix_primary_key('agent_chat_messages', 'id');

-- 删除临时存储过程
DROP PROCEDURE IF EXISTS fix_primary_key;

-- 4. 重新启用外键检查
SET FOREIGN_KEY_CHECKS = 1;

-- 完成后，GORM 的 AutoMigrate 会自动重新创建外键约束
