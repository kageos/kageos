-- ========================================
-- 工作空间根节点迁移脚本
-- 目标：为所有 app 创建对应的 service_tree 根节点
-- 作者：AI Agent + 洛北
-- 日期：2026-01-21
-- ========================================

-- ========================================
-- Step 0: 迁移前检查
-- ========================================

-- 检查当前 app 数量
SELECT 
    '迁移前检查：app 表总数' as info,
    COUNT(*) as app_count
FROM app;

-- 检查已有根节点的 app 数量（ref_id = app_id）
SELECT 
    '迁移前检查：已有根节点的 app 数量' as info,
    COUNT(DISTINCT st.app_id) as app_with_root_count
FROM service_tree st
WHERE st.parent_id = 0 
  AND st.ref_id IS NOT NULL
  AND st.ref_id = st.app_id;

-- 检查需要迁移的 app 数量
SELECT 
    '迁移前检查：需要迁移的 app 数量' as info,
    COUNT(*) as apps_need_migration
FROM app a
WHERE NOT EXISTS (
    SELECT 1 FROM service_tree st 
    WHERE st.app_id = a.id 
      AND st.parent_id = 0 
      AND st.ref_id = a.id
);

-- 检查当前根级子节点数量（parent_id=0 但不是根节点本身）
SELECT 
    '迁移前检查：根级子节点数量' as info,
    COUNT(*) as root_level_children
FROM service_tree st
WHERE st.parent_id = 0
  AND (st.ref_id IS NULL OR st.ref_id != st.app_id);

-- ========================================
-- Step 1: 为所有现有 app 创建根节点
-- ========================================

-- 插入根节点（只为没有根节点的 app 创建）
INSERT INTO service_tree (
    name, 
    code, 
    parent_id, 
    type, 
    description, 
    tags, 
    admins, 
    pending_count, 
    app_id, 
    ref_id, 
    full_code_path, 
    version, 
    version_num, 
    created_by, 
    updated_by, 
    created_at, 
    updated_at
)
SELECT 
    a.name,                           -- name：使用 app 的名称
    a.code,                           -- code：使用 app 的 code
    0,                                -- parent_id：根节点
    'package',                        -- type：统一为 package
    '',                               -- description：空
    '',                               -- tags：空
    a.admins,                         -- admins：继承 app 的管理员列表
    COALESCE(a.pending_count, 0),    -- pending_count：继承 app 的待办数量，如果为 NULL 则为 0
    a.id,                             -- app_id：关联的 app ID
    a.id,                             -- ref_id：指向 app 表（标识这是根节点）
    CONCAT('/', a.user, '/', a.code), -- full_code_path：/user/app
    'v1',                             -- version：初始版本
    1,                                -- version_num：版本号 1
    COALESCE(a.created_by, 'system'), -- created_by：继承 app 的创建者，如果为空则为 system
    COALESCE(a.updated_by, 'system'), -- updated_by：继承 app 的更新者，如果为空则为 system
    COALESCE(a.created_at, NOW()),    -- created_at：继承 app 的创建时间，如果为空则为当前时间
    COALESCE(a.updated_at, NOW())     -- updated_at：继承 app 的更新时间，如果为空则为当前时间
FROM app a
WHERE NOT EXISTS (
    SELECT 1 FROM service_tree st 
    WHERE st.app_id = a.id 
      AND st.parent_id = 0 
      AND st.ref_id = a.id
);

-- 显示插入结果
SELECT 
    '完成 Step 1：创建根节点' as info,
    ROW_COUNT() as inserted_count;

-- ========================================
-- Step 2: 更新原有根级子节点的 parent_id
-- ========================================

-- 更新原有根级子节点，将其 parent_id 指向新创建的根节点
-- ⭐ 使用 JOIN 代替子查询，避免 MySQL "You can't specify target table for update in FROM clause" 错误
UPDATE service_tree st
INNER JOIN service_tree st_root 
  ON st_root.app_id = st.app_id 
  AND st_root.parent_id = 0
  AND st_root.ref_id = st.app_id
SET st.parent_id = st_root.id
WHERE st.parent_id = 0
  AND (st.ref_id IS NULL OR st.ref_id != st.app_id)  -- 不是根节点本身
  AND st.app_id IS NOT NULL;

-- 显示更新结果
SELECT 
    '完成 Step 2：更新子节点 parent_id' as info,
    ROW_COUNT() as updated_count;

-- ========================================
-- Step 3: 迁移后验证
-- ========================================

-- 验证：检查所有 app 是否都有根节点
SELECT 
    '验证：所有 app 的根节点情况' as check_name,
    a.id as app_id,
    a.user,
    a.code,
    a.name,
    (SELECT COUNT(*) FROM service_tree WHERE app_id = a.id AND parent_id = 0 AND ref_id = a.id) as root_count,
    (SELECT id FROM service_tree WHERE app_id = a.id AND parent_id = 0 AND ref_id = a.id LIMIT 1) as root_node_id,
    CASE 
        WHEN (SELECT COUNT(*) FROM service_tree WHERE app_id = a.id AND parent_id = 0 AND ref_id = a.id) = 1 THEN '✅ 正常'
        WHEN (SELECT COUNT(*) FROM service_tree WHERE app_id = a.id AND parent_id = 0 AND ref_id = a.id) = 0 THEN '❌ 缺少根节点'
        ELSE '⚠️ 根节点重复'
    END as status
FROM app a
ORDER BY status DESC, a.id;

-- 验证：检查是否还有孤立的根级子节点（parent_id=0 但不是根节点）
SELECT 
    '验证：检查孤立的根级子节点' as check_name,
    st.id,
    st.name,
    st.code,
    st.app_id,
    st.parent_id,
    st.ref_id,
    st.full_code_path,
    CASE 
        WHEN st.ref_id = st.app_id THEN '✅ 这是根节点'
        ELSE '❌ 孤立节点（parent_id=0 但不是根节点）'
    END as status
FROM service_tree st
WHERE st.parent_id = 0
ORDER BY status DESC, st.id;

-- 验证：检查根节点的子节点数量
SELECT 
    '验证：根节点的子节点统计' as check_name,
    a.id as app_id,
    a.user,
    a.code,
    a.name,
    st_root.id as root_node_id,
    (SELECT COUNT(*) FROM service_tree WHERE parent_id = st_root.id) as children_count
FROM app a
LEFT JOIN service_tree st_root ON st_root.app_id = a.id AND st_root.parent_id = 0 AND st_root.ref_id = a.id
ORDER BY children_count DESC;

-- 验证：统计迁移结果
SELECT 
    '迁移结果统计' as info,
    (SELECT COUNT(*) FROM app) as total_apps,
    (SELECT COUNT(DISTINCT app_id) FROM service_tree WHERE parent_id = 0 AND ref_id IS NOT NULL AND ref_id = app_id) as apps_with_root,
    (SELECT COUNT(*) FROM service_tree WHERE parent_id = 0 AND ref_id IS NOT NULL AND ref_id = app_id) as total_root_nodes,
    CASE 
        WHEN (SELECT COUNT(*) FROM app) = (SELECT COUNT(DISTINCT app_id) FROM service_tree WHERE parent_id = 0 AND ref_id IS NOT NULL AND ref_id = app_id) 
        THEN '✅ 迁移成功'
        ELSE '❌ 迁移不完整'
    END as migration_status;

-- ========================================
-- 迁移完成提示
-- ========================================

SELECT 
    '========================================' as separator,
    '迁移完成！' as message,
    '请检查上面的验证结果，确保所有 app 都有根节点。' as next_step,
    '如果发现问题，请执行回滚脚本：workspace_root_node_rollback.sql' as rollback_info,
    '========================================' as separator2;
