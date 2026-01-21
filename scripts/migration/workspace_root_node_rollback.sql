-- ========================================
-- 工作空间根节点迁移 - 回滚脚本
-- 目标：撤销根节点迁移，恢复原始状态
-- 作者：AI Agent + 洛北
-- 日期：2026-01-21
-- ========================================

-- ⚠️ 警告：此脚本会删除迁移时创建的根节点，请谨慎操作！

-- ========================================
-- Step 0: 回滚前检查
-- ========================================

-- 检查当前根节点数量
SELECT 
    '回滚前检查：当前根节点数量' as info,
    COUNT(*) as root_node_count
FROM service_tree
WHERE parent_id = 0 
  AND ref_id IS NOT NULL
  AND ref_id = app_id;

-- 检查会被影响的子节点数量
SELECT 
    '回滚前检查：会被影响的子节点数量' as info,
    COUNT(*) as affected_children
FROM service_tree st_child
WHERE st_child.parent_id IN (
    SELECT st_root.id 
    FROM service_tree st_root
    WHERE st_root.parent_id = 0 
      AND st_root.ref_id IS NOT NULL
      AND st_root.ref_id = st_root.app_id
);

-- ========================================
-- Step 1: 恢复子节点的 parent_id 为 0
-- ========================================

-- 将原本指向根节点的子节点的 parent_id 改回 0
-- ⭐ 使用 JOIN 代替子查询，避免 MySQL "You can't specify target table for update in FROM clause" 错误
UPDATE service_tree st_child
INNER JOIN service_tree st_root
  ON st_child.parent_id = st_root.id
  AND st_root.parent_id = 0 
  AND st_root.ref_id IS NOT NULL
  AND st_root.ref_id = st_root.app_id
SET st_child.parent_id = 0;

-- 显示更新结果
SELECT 
    '完成 Step 1：恢复子节点 parent_id' as info,
    ROW_COUNT() as updated_count;

-- ========================================
-- Step 2: 删除根节点
-- ========================================

-- 删除迁移时创建的根节点（parent_id=0 且 ref_id=app_id）
DELETE FROM service_tree
WHERE parent_id = 0 
  AND ref_id IS NOT NULL
  AND ref_id = app_id;

-- 显示删除结果
SELECT 
    '完成 Step 2：删除根节点' as info,
    ROW_COUNT() as deleted_count;

-- ========================================
-- Step 3: 回滚后验证
-- ========================================

-- 验证：检查是否还有根节点
SELECT 
    '验证：剩余根节点数量' as check_name,
    COUNT(*) as remaining_root_nodes
FROM service_tree
WHERE parent_id = 0 
  AND ref_id IS NOT NULL
  AND ref_id = app_id;

-- 验证：检查根级子节点（parent_id=0）数量
SELECT 
    '验证：根级子节点数量' as check_name,
    COUNT(*) as root_level_children
FROM service_tree
WHERE parent_id = 0;

-- 验证：按 app 统计根级子节点
SELECT 
    '验证：各 app 的根级子节点统计' as check_name,
    a.id as app_id,
    a.user,
    a.code,
    a.name,
    (SELECT COUNT(*) FROM service_tree WHERE app_id = a.id AND parent_id = 0) as root_level_count
FROM app a
ORDER BY root_level_count DESC;

-- ========================================
-- 回滚完成提示
-- ========================================

SELECT 
    '========================================' as divider,
    '回滚完成！' as message,
    '数据库已恢复到迁移前的状态。' as status,
    '如需重新迁移，请执行：workspace_root_node.sql' as next_step,
    '========================================' as divider2;
