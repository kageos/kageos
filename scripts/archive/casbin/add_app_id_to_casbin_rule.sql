-- 为 casbin_rule 表添加 app_id 字段，方便根据工作空间查询权限
-- 执行此脚本可以显著提升按工作空间查询权限的性能

-- 1. 添加 app_id 字段（如果不存在）
-- MySQL 不支持 ALTER TABLE ADD COLUMN IF NOT EXISTS，需要使用动态 SQL 检查
SET @dbname = DATABASE();
SET @tablename = 'casbin_rule';
SET @columnname = 'app_id';
SET @preparedStatement = (SELECT IF(
    (
        SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
        WHERE
            (TABLE_SCHEMA = @dbname)
            AND (TABLE_NAME = @tablename)
            AND (COLUMN_NAME = @columnname)
    ) > 0,
    'SELECT 1',  -- 列已存在，不执行任何操作
    CONCAT('ALTER TABLE `', @tablename, '` ADD COLUMN `', @columnname, '` BIGINT DEFAULT NULL COMMENT ''应用ID（用于快速查询工作空间权限）''')
));
PREPARE alterIfNotExists FROM @preparedStatement;
EXECUTE alterIfNotExists;
DEALLOCATE PREPARE alterIfNotExists;

-- 2. 添加索引（提升查询性能）
-- 为了兼容性，也使用动态 SQL 检查索引是否存在
SET @indexname1 = 'idx_casbin_app_id';
SET @preparedStatement1 = (SELECT IF(
    (
        SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
        WHERE
            (TABLE_SCHEMA = @dbname)
            AND (TABLE_NAME = @tablename)
            AND (INDEX_NAME = @indexname1)
    ) > 0,
    'SELECT 1',  -- 索引已存在，不执行任何操作
    CONCAT('CREATE INDEX `', @indexname1, '` ON `', @tablename, '`(`app_id`)')
));
PREPARE createIndex1 FROM @preparedStatement1;
EXECUTE createIndex1;
DEALLOCATE PREPARE createIndex1;

-- 3. 添加复合索引（用于按工作空间和用户查询权限）
SET @indexname2 = 'idx_casbin_app_user';
SET @preparedStatement2 = (SELECT IF(
    (
        SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
        WHERE
            (TABLE_SCHEMA = @dbname)
            AND (TABLE_NAME = @tablename)
            AND (INDEX_NAME = @indexname2)
    ) > 0,
    'SELECT 1',  -- 索引已存在，不执行任何操作
    CONCAT('CREATE INDEX `', @indexname2, '` ON `', @tablename, '`(`app_id`, `ptype`, `v0`)')
));
PREPARE createIndex2 FROM @preparedStatement2;
EXECUTE createIndex2;
DEALLOCATE PREPARE createIndex2;

-- 说明：
-- - app_id: 应用ID，从 full_code_path 中解析出 user 和 app，然后查询 app.id 填充
-- - idx_casbin_app_id: 用于快速查询某个工作空间的所有权限
-- - idx_casbin_app_user: 用于快速查询某个工作空间下某个用户的所有权限
--
-- 性能提升预期：
-- - 按工作空间查询权限：从全表扫描 → 索引查询（10-100倍提升）
-- - 按工作空间和用户查询权限：从多次查询 → 一次索引查询（5-10倍提升）
