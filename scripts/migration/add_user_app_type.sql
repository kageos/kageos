-- 添加用户类型和应用类型字段
-- 执行时间：2025-01-18
-- 注意：此脚本会检查字段是否存在，可以安全地多次执行

-- 1. 添加用户类型字段（如果不存在）
SET @dbname = DATABASE();
SET @tablename = 'user';
SET @columnname = 'type';
SET @preparedStatement = (SELECT IF(
  (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
    WHERE
      (TABLE_SCHEMA = @dbname)
      AND (TABLE_NAME = @tablename)
      AND (COLUMN_NAME = @columnname)
  ) > 0,
  'SELECT "Column type already exists in user table" AS message;',
  CONCAT('ALTER TABLE `', @tablename, '` ADD COLUMN `', @columnname, '` TINYINT NOT NULL DEFAULT 0 COMMENT ''用户类型(0:普通用户,1:系统用户,2:智能体用户)'' AFTER `gender`, ADD INDEX `idx_user_type` (`type`);')
));
PREPARE alterIfNotExists FROM @preparedStatement;
EXECUTE alterIfNotExists;
DEALLOCATE PREPARE alterIfNotExists;

-- 2. 添加应用类型字段（如果不存在）
SET @tablename = 'app';
SET @preparedStatement = (SELECT IF(
  (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
    WHERE
      (TABLE_SCHEMA = @dbname)
      AND (TABLE_NAME = @tablename)
      AND (COLUMN_NAME = @columnname)
  ) > 0,
  'SELECT "Column type already exists in app table" AS message;',
  CONCAT('ALTER TABLE `', @tablename, '` ADD COLUMN `', @columnname, '` TINYINT NOT NULL DEFAULT 0 COMMENT ''应用类型(0:用户空间,1:系统空间)'' AFTER `admins`, ADD INDEX `idx_app_type` (`type`);')
));
PREPARE alterIfNotExists FROM @preparedStatement;
EXECUTE alterIfNotExists;
DEALLOCATE PREPARE alterIfNotExists;

-- 3. 初始化 system 用户（如果不存在）
-- 注意：这里只是示例，实际应该在应用启动时通过代码初始化
-- INSERT INTO `user` (`username`, `email`, `status`, `email_verified`, `type`, `created_at`, `updated_at`)
-- SELECT 'system', 'system@ai-agent-os.local', 'active', 1, 1, NOW(), NOW()
-- WHERE NOT EXISTS (SELECT 1 FROM `user` WHERE `username` = 'system');
