-- 权限系统重构：创建新表结构
-- 执行时间：2025-01-06
-- 说明：完全自研权限系统，脱离 Casbin

-- ============================================
-- 1. 工作空间权限表（只存储已生效的权限）
-- ============================================
CREATE TABLE IF NOT EXISTS `workspace_permission` (
  `id` BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
  `user` VARCHAR(100) NOT NULL COMMENT '租户用户名（从 resource_path 解析，如 /user/app/... 中的 user）',
  `app` VARCHAR(100) NOT NULL COMMENT '应用代码（从 resource_path 解析，如 /user/app/... 中的 app）',
  `subject_type` VARCHAR(20) NOT NULL COMMENT '权限主体类型：user（用户）、department（组织架构）',
  `subject` VARCHAR(150) NOT NULL COMMENT '权限主体：用户名或组织架构路径（如 /org/master/bizit）',
  `resource_path` VARCHAR(150) NOT NULL COMMENT '资源路径（如 /luobei/operations/crm/ticket）',
  `resource_type` VARCHAR(20) NOT NULL COMMENT '资源类型：app（工作空间）、directory（目录）、function（函数）',
  `action` VARCHAR(50) NOT NULL COMMENT '操作类型：app:read、directory:read、function:write 等',
  `is_wildcard` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否通配符路径：0-精确路径，1-通配符路径（如 /path/*）',
  `start_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '权限开始时间（默认创建时间）',
  `end_time` DATETIME DEFAULT NULL COMMENT '权限结束时间（NULL 表示永久权限）',
  `source_type` VARCHAR(20) NOT NULL COMMENT '权限来源：grant（授权）、approval（审批通过）',
  `source_id` BIGINT DEFAULT NULL COMMENT '来源记录ID（授权记录ID或审批记录ID）',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `created_by` VARCHAR(100) DEFAULT NULL COMMENT '创建者用户名（授权时为授权人，审批通过时为审批人）',
  
  -- 索引设计
  INDEX `idx_user_app` (`user`, `app`),
  INDEX `idx_user_app_resource` (`user`, `app`, `resource_path`),
  INDEX `idx_user_app_subject` (`user`, `app`, `subject_type`, `subject`),
  INDEX `idx_subject_resource` (`subject_type`, `subject`, `resource_path`),
  INDEX `idx_resource_action` (`resource_path`, `action`),
  INDEX `idx_time` (`start_time`, `end_time`),
  
  -- 唯一约束：防止重复授权
  UNIQUE KEY `uk_user_app_subject_resource_action` (`user`, `app`, `subject_type`, `subject`, `resource_path`, `action`, `is_wildcard`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='工作空间权限表（只存储已生效的权限）';

-- ============================================
-- 2. 权限申请审批表
-- ============================================
CREATE TABLE IF NOT EXISTS `permission_request` (
  `id` BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
  `app_id` BIGINT NOT NULL COMMENT '工作空间ID',
  `applicant_username` VARCHAR(100) NOT NULL COMMENT '申请人用户名',
  `subject_type` VARCHAR(20) NOT NULL COMMENT '权限主体类型：user（用户）、department（组织架构）',
  `subject` VARCHAR(150) NOT NULL COMMENT '权限主体（申请人自己或组织架构路径）',
  `resource_path` VARCHAR(150) NOT NULL COMMENT '资源路径',
  `action` VARCHAR(50) NOT NULL COMMENT '操作类型',
  `start_time` DATETIME NOT NULL COMMENT '权限开始时间（申请时指定）',
  `end_time` DATETIME DEFAULT NULL COMMENT '权限结束时间（申请时指定，NULL 表示永久）',
  `reason` TEXT COMMENT '申请原因',
  `status` VARCHAR(20) NOT NULL DEFAULT 'pending' COMMENT '申请状态：pending（待审批）、approved（已批准）、rejected（已拒绝）、cancelled（已取消）',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '申请时间',
  `approved_at` DATETIME DEFAULT NULL COMMENT '审批时间',
  `approved_by` VARCHAR(100) DEFAULT NULL COMMENT '审批人用户名',
  `rejected_at` DATETIME DEFAULT NULL COMMENT '拒绝时间',
  `rejected_by` VARCHAR(100) DEFAULT NULL COMMENT '拒绝人用户名',
  `reject_reason` TEXT COMMENT '拒绝原因',
  `cancelled_at` DATETIME DEFAULT NULL COMMENT '取消时间',
  `cancelled_by` VARCHAR(100) DEFAULT NULL COMMENT '取消人用户名（申请人自己取消）',
  `permission_id` BIGINT DEFAULT NULL COMMENT '关联的权限记录ID（审批通过后创建）',
  
  -- 索引设计
  INDEX `idx_applicant` (`applicant_username`, `created_at`),
  INDEX `idx_resource_status` (`resource_path`, `status`),
  INDEX `idx_approver` (`approved_by`, `approved_at`),
  INDEX `idx_status` (`status`, `created_at`),
  INDEX `idx_app_status` (`app_id`, `status`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='权限申请审批表';

-- ============================================
-- 3. 授权记录表
-- ============================================
CREATE TABLE IF NOT EXISTS `permission_grant_log` (
  `id` BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
  `app_id` BIGINT NOT NULL COMMENT '工作空间ID',
  `grantor_username` VARCHAR(100) NOT NULL COMMENT '授权人用户名（谁授权的）',
  `grantee_type` VARCHAR(20) NOT NULL COMMENT '被授权人类型：user（用户）、department（组织架构）',
  `grantee` VARCHAR(150) NOT NULL COMMENT '被授权人（用户名或组织架构路径）',
  `resource_path` VARCHAR(150) NOT NULL COMMENT '资源路径',
  `action` VARCHAR(50) NOT NULL COMMENT '操作类型',
  `start_time` DATETIME NOT NULL COMMENT '权限开始时间',
  `end_time` DATETIME DEFAULT NULL COMMENT '权限结束时间（NULL 表示永久）',
  `granted_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '授权时间',
  `revoked_at` DATETIME DEFAULT NULL COMMENT '撤销时间（NULL 表示未撤销）',
  `revoked_by` VARCHAR(100) DEFAULT NULL COMMENT '撤销人用户名',
  `revoke_reason` TEXT COMMENT '撤销原因',
  `permission_id` BIGINT DEFAULT NULL COMMENT '关联的权限记录ID（授权时创建的权限记录）',
  
  -- 索引设计
  INDEX `idx_grantor` (`grantor_username`, `granted_at`),
  INDEX `idx_grantee` (`grantee_type`, `grantee`, `granted_at`),
  INDEX `idx_resource` (`resource_path`, `granted_at`),
  INDEX `idx_app_grantor` (`app_id`, `grantor_username`, `granted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='授权记录表（用于审计追溯）';

-- ============================================
-- 4. 审批策略配置表（目录级别配置）
-- ============================================
CREATE TABLE IF NOT EXISTS `approval_policy` (
  `id` BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
  `app_id` BIGINT NOT NULL COMMENT '工作空间ID',
  `resource_path` VARCHAR(150) NOT NULL COMMENT '目录路径（只配置在目录级别）',
  `policy_expression` VARCHAR(150) NOT NULL COMMENT '审批策略表达式（使用内置变量，如 $current_node_admins）',
  `description` VARCHAR(150) DEFAULT NULL COMMENT '策略描述',
  `is_enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用：0-禁用，1-启用',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `created_by` VARCHAR(100) DEFAULT NULL COMMENT '创建者用户名',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `updated_by` VARCHAR(100) DEFAULT NULL COMMENT '更新者用户名',
  
  -- 索引设计
  INDEX `idx_resource_policy` (`resource_path`, `is_enabled`),
  INDEX `idx_app_resource` (`app_id`, `resource_path`, `is_enabled`),
  
  -- 唯一约束：每个目录只能有一个启用的策略
  UNIQUE KEY `uk_app_resource_enabled` (`app_id`, `resource_path`, `is_enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='审批策略配置表（目录级别配置）';

-- ============================================
-- 5. 在 service_tree 表中添加 admins 字段
-- ============================================
ALTER TABLE `service_tree` 
ADD COLUMN IF NOT EXISTS `admins` VARCHAR(150) DEFAULT NULL COMMENT '节点管理员列表，逗号分隔的用户名（如 user1,user2,user3）' AFTER `tags`;

-- 为 admins 字段添加索引（用于查询用户管理的节点）
ALTER TABLE `service_tree`
ADD INDEX IF NOT EXISTS `idx_admins` (`admins`(255));

-- ============================================
-- 6. 初始化节点管理员（将创建者添加为管理员）
-- ============================================
-- 注意：这个需要根据实际情况调整，如果 service_tree 表有 created_by 字段，可以使用
-- UPDATE service_tree SET admins = created_by WHERE admins IS NULL AND created_by IS NOT NULL;

