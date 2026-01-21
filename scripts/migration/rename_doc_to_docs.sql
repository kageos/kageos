-- ========================================
-- 表名重命名迁移脚本
-- 目标：将 doc 表重命名为 docs，统一命名规范
-- 作者：AI Agent + 洛北
-- 日期：2026-01-21
-- ========================================

-- ========================================
-- Step 0: 迁移前检查
-- ========================================

-- 检查 doc 表是否存在
SELECT 
    '迁移前检查：doc 表是否存在' as info,
    COUNT(*) as table_exists
FROM information_schema.tables 
WHERE table_schema = DATABASE() 
  AND table_name = 'doc';

-- 检查 docs 表是否已存在（避免冲突）
SELECT 
    '迁移前检查：docs 表是否已存在' as info,
    COUNT(*) as table_exists
FROM information_schema.tables 
WHERE table_schema = DATABASE() 
  AND table_name = 'docs';

-- 检查 doc 表的数据量
SELECT 
    '迁移前检查：doc 表数据量' as info,
    COUNT(*) as row_count
FROM doc;

-- ========================================
-- Step 1: 重命名表
-- ========================================

-- 将 doc 表重命名为 docs
RENAME TABLE `doc` TO `docs`;

-- 显示重命名结果
SELECT '完成 Step 1：表重命名成功 (doc → docs)' as info;

-- ========================================
-- Step 2: 迁移后验证
-- ========================================

-- 验证：检查 docs 表是否存在
SELECT 
    '验证：docs 表是否存在' as check_name,
    COUNT(*) as table_exists,
    CASE 
        WHEN COUNT(*) = 1 THEN '✅ docs 表存在'
        ELSE '❌ docs 表不存在'
    END as status
FROM information_schema.tables 
WHERE table_schema = DATABASE() 
  AND table_name = 'docs';

-- 验证：检查 doc 表是否已删除
SELECT 
    '验证：doc 表是否已删除' as check_name,
    COUNT(*) as table_exists,
    CASE 
        WHEN COUNT(*) = 0 THEN '✅ doc 表已删除'
        ELSE '❌ doc 表仍然存在'
    END as status
FROM information_schema.tables 
WHERE table_schema = DATABASE() 
  AND table_name = 'doc';

-- 验证：检查 docs 表的数据量
SELECT 
    '验证：docs 表数据量' as check_name,
    COUNT(*) as row_count
FROM docs;

-- 验证：检查 docs 表的结构
SHOW CREATE TABLE docs;

-- ========================================
-- 迁移结果统计
-- ========================================

SELECT 
    '迁移结果统计' as info,
    (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'docs') as docs_table_exists,
    (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'doc') as doc_table_exists,
    (SELECT COUNT(*) FROM docs) as docs_row_count,
    CASE 
        WHEN (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'docs') = 1
         AND (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'doc') = 0
        THEN '✅ 迁移成功'
        ELSE '❌ 迁移失败'
    END as migration_status;

-- ========================================
-- 迁移完成提示
-- ========================================

SELECT 
    '========================================' as separator,
    '迁移完成！' as message,
    '表名已从 doc 重命名为 docs' as change,
    '请重启后端服务以使用新表名' as next_step;

-- ========================================
-- 回滚说明
-- ========================================
-- 如果需要回滚，请执行：
-- RENAME TABLE `docs` TO `doc`;
