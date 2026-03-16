-- ============================================
-- 在 app 表中添加 show_only_permitted 字段
-- ============================================
-- 用途：仅展示有权限的空间。开启后，非管理员用户进入该工作空间时，服务树只展示其有权限的目录（SaaS 多租户场景）
-- 若使用 GORM AutoMigrate，可自动加列，本脚本供手动执行或非 GORM 环境使用。

ALTER TABLE `app`
ADD COLUMN `show_only_permitted` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '仅展示有权限的空间(0:否,1:是)' AFTER `pending_count`;
