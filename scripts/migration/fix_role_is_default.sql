-- 修复角色 is_default 字段
-- 根据预设角色配置，将"查看者"角色设置为默认角色

-- Directory 资源类型：查看者角色
UPDATE `role` 
SET `is_default` = 1 
WHERE `resource_type` = 'directory' 
  AND `code` = 'viewer' 
  AND `is_system` = 1;

-- Table 资源类型：查看者角色
UPDATE `role` 
SET `is_default` = 1 
WHERE `resource_type` = 'table' 
  AND `code` = 'viewer' 
  AND `is_system` = 1;

-- Form 资源类型：查看者角色
UPDATE `role` 
SET `is_default` = 1 
WHERE `resource_type` = 'form' 
  AND `code` = 'viewer' 
  AND `is_system` = 1;

-- Chart 资源类型：查看者角色
UPDATE `role` 
SET `is_default` = 1 
WHERE `resource_type` = 'chart' 
  AND `code` = 'viewer' 
  AND `is_system` = 1;

-- 确保其他角色不是默认角色（可选，如果需要的话）
UPDATE `role` 
SET `is_default` = 0 
WHERE `code` != 'viewer' 
  AND `is_system` = 1;
