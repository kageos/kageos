-- ============================================
-- 在 app 表中添加 pending_count 字段
-- ============================================
-- 用途：存储 app 级别的待审批权限申请数量
-- 更新时机：
--   1. 创建 app 级别权限申请时：pending_count + 1
--   2. 审批通过时：pending_count - 1
--   3. 审批驳回时：pending_count - 1

ALTER TABLE `app` 
ADD COLUMN `pending_count` INT NOT NULL DEFAULT 0 COMMENT 'app级别待审批的权限申请数量' AFTER `admins`;

-- 添加索引（如果需要按数量排序或筛选）
-- CREATE INDEX `idx_pending_count` ON `app` (`pending_count`);
