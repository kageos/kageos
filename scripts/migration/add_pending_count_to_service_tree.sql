-- ============================================
-- 在 service_tree 表中添加 pending_count 字段
-- ============================================
-- 用途：存储待审批的权限申请数量，用于在树节点上显示申请中数量
-- 更新时机：
--   1. 创建权限申请时：pending_count + 1
--   2. 审批通过时：pending_count - 1
--   3. 审批驳回时：pending_count - 1

ALTER TABLE `service_tree` 
ADD COLUMN `pending_count` INT NOT NULL DEFAULT 0 COMMENT '待审批的权限申请数量' AFTER `admins`;

-- 添加索引（如果需要按数量排序或筛选）
-- CREATE INDEX `idx_pending_count` ON `service_tree` (`pending_count`);
