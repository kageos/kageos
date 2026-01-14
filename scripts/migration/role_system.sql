-- 角色系统数据库设计
-- 执行时间：2025-01-27
-- 说明：全局角色系统，支持用户和组织架构角色分配

-- ============================================
-- 1. 全局角色表
-- ============================================
CREATE TABLE IF NOT EXISTS `role` (
  `id` BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
  `name` VARCHAR(100) NOT NULL COMMENT '角色名称（如：开发者、管理员、查看者、游客）',
  `code` VARCHAR(50) NOT NULL UNIQUE COMMENT '角色编码（如：developer、admin、viewer、visitor）',
  `description` VARCHAR(500) COMMENT '角色描述',
  `is_system` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否系统预设角色（不可删除）',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `created_by` VARCHAR(100) COMMENT '创建者用户名',
  
  INDEX `idx_code` (`code`)
) COMMENT='全局角色表';

-- ============================================
-- 2. 角色权限配置表（按资源类型配置权限）
-- ============================================
CREATE TABLE IF NOT EXISTS `role_permission` (
  `id` BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
  `role_id` BIGINT NOT NULL COMMENT '角色ID',
  `resource_type` VARCHAR(20) NOT NULL COMMENT '资源类型：directory（目录）、table（表格函数）、form（表单函数）、chart（图表函数）、app（工作空间）',
  `action` VARCHAR(50) NOT NULL COMMENT '权限点（如 function:read、directory:manage）',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  
  INDEX `idx_role_id` (`role_id`),
  INDEX `idx_resource_type` (`resource_type`),
  INDEX `idx_action` (`action`),
  INDEX `idx_role_resource` (`role_id`, `resource_type`),
  UNIQUE KEY `uk_role_resource_action` (`role_id`, `resource_type`, `action`)
) COMMENT='角色权限配置表（按资源类型配置权限）';

-- ============================================
-- 3. 角色分配表（统一表，支持用户和组织架构）
-- ============================================
CREATE TABLE IF NOT EXISTS `role_assignment` (
  `id` BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
  `user` VARCHAR(100) NOT NULL COMMENT '租户用户名（从 resource_path 解析）',
  `app` VARCHAR(100) NOT NULL COMMENT '应用代码（从 resource_path 解析）',
  `subject_type` VARCHAR(20) NOT NULL COMMENT '权限主体类型：user（用户）、department（组织架构）',
  `subject` VARCHAR(150) NOT NULL COMMENT '权限主体：用户名或组织架构路径（如 /org/master/bizit）',
  `role_id` BIGINT NOT NULL COMMENT '角色ID',
  `resource_path` VARCHAR(150) NOT NULL COMMENT '资源路径（角色生效范围，如 /user/app/tools/*）',
  `start_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '生效开始时间',
  `end_time` DATETIME DEFAULT NULL COMMENT '生效结束时间（NULL 表示永久）',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `created_by` VARCHAR(100) COMMENT '创建者用户名',
  
  INDEX `idx_user_app` (`user`, `app`),
  INDEX `idx_subject_type_subject` (`subject_type`, `subject`),
  INDEX `idx_role_id` (`role_id`),
  INDEX `idx_resource` (`resource_path`),
  INDEX `idx_time` (`start_time`, `end_time`),
  INDEX `idx_user_app_subject` (`user`, `app`, `subject_type`, `subject`)
) COMMENT='角色分配表（统一表，支持用户和组织架构）';
