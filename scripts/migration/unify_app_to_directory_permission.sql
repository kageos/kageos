-- ========================================
-- 统一 app 资源类型到 directory 资源类型
-- 目标：删除 app 资源类型，将工作空间权限统一到 directory
-- 作者：AI Agent + 洛北
-- 日期：2026-01-21
-- ========================================

-- ======================================== 
-- Step 0: 迁移前检查
-- ========================================

-- 检查当前 app 资源类型的角色
SELECT 
    '迁移前检查：app 资源类型的角色' as info,
    COUNT(*) as role_count
FROM role
WHERE resource_type = 'app';

-- 检查当前 app 资源类型的权限点
SELECT 
    '迁移前检查：app 资源类型的权限点' as info,
    COUNT(*) as action_count
FROM action
WHERE resource_type = 'app';

-- 检查当前 app 资源类型的角色分配
SELECT 
    '迁移前检查：app 资源类型的角色分配' as info,
    COUNT(*) as assignment_count
FROM role_assignment ra
INNER JOIN role r ON ra.role_id = r.id
WHERE r.resource_type = 'app';

-- ======================================== 
-- Step 1: 迁移角色分配（app -> directory）
-- ========================================

-- 获取 directory 资源类型的 admin 角色ID
SET @directory_admin_role_id = (SELECT id FROM role WHERE code = 'admin' AND resource_type = 'directory' LIMIT 1);

-- 更新所有使用 app 资源类型角色的分配记录
UPDATE role_assignment ra
INNER JOIN role r ON ra.role_id = r.id
SET ra.role_id = @directory_admin_role_id
WHERE r.resource_type = 'app'
  AND @directory_admin_role_id IS NOT NULL;

SELECT 
    '完成 Step 1：迁移角色分配' as info,
    ROW_COUNT() as updated_count;

-- ======================================== 
-- Step 2: 删除 app 资源类型的角色
-- ========================================

-- 删除 app 资源类型的角色权限关联
DELETE rp FROM role_permission rp
INNER JOIN role r ON rp.role_id = r.id
WHERE r.resource_type = 'app';

SELECT 
    '完成 Step 2.1：删除角色权限关联' as info,
    ROW_COUNT() as deleted_count;

-- 删除 app 资源类型的角色
DELETE FROM role
WHERE resource_type = 'app';

SELECT 
    '完成 Step 2.2：删除 app 角色' as info,
    ROW_COUNT() as deleted_count;

-- ======================================== 
-- Step 3: 删除 app 资源类型的权限点
-- ========================================

-- 删除 app 资源类型的权限点
DELETE FROM action
WHERE resource_type = 'app';

SELECT 
    '完成 Step 3：删除 app 权限点' as info,
    ROW_COUNT() as deleted_count;

-- ======================================== 
-- Step 4: 迁移后验证
-- ========================================

-- 验证：检查是否还有 app 资源类型的角色
SELECT 
    '验证：app 资源类型的角色' as check_name,
    COUNT(*) as remaining_count,
    CASE 
        WHEN COUNT(*) = 0 THEN '✅ 已全部清理'
        ELSE '❌ 仍有残留'
    END as status
FROM role
WHERE resource_type = 'app';

-- 验证：检查是否还有 app 资源类型的权限点
SELECT 
    '验证：app 资源类型的权限点' as check_name,
    COUNT(*) as remaining_count,
    CASE 
        WHEN COUNT(*) = 0 THEN '✅ 已全部清理'
        ELSE '❌ 仍有残留'
    END as status
FROM action
WHERE resource_type = 'app';

-- 验证：检查是否还有使用 app 角色的分配
SELECT 
    '验证：使用 app 角色的分配' as check_name,
    COUNT(*) as remaining_count,
    CASE 
        WHEN COUNT(*) = 0 THEN '✅ 已全部清理'
        ELSE '❌ 仍有残留'
    END as status
FROM role_assignment ra
INNER JOIN role r ON ra.role_id = r.id
WHERE r.resource_type = 'app';

-- 验证：检查 directory 资源类型的角色分配统计
SELECT 
    '验证：directory 资源类型的角色分配统计' as check_name,
    COUNT(*) as total_assignments,
    COUNT(DISTINCT ra.subject) as unique_subjects,
    COUNT(DISTINCT ra.resource_path) as unique_resources
FROM role_assignment ra
INNER JOIN role r ON ra.role_id = r.id
WHERE r.resource_type = 'directory';

-- ======================================== 
-- 迁移完成提示
-- ========================================

SELECT 
    '========================================' as message
UNION ALL
SELECT '✅ 迁移完成！'
UNION ALL
SELECT '已将所有 app 资源类型统一到 directory 资源类型'
UNION ALL
SELECT '工作空间权限现在使用 directory:admin、directory:read 等'
UNION ALL
SELECT '========================================';
