-- 补偿脚本：填充缺失的 app_id
-- 用于修复因时序问题导致的 app_id 为 NULL 的记录
-- 建议定期执行（如每天凌晨执行一次）

-- 更新 app_id 为 NULL 的策略记录
UPDATE casbin_rule cr
INNER JOIN (
    -- 从 v1（资源路径）解析出 user 和 app，查询 app.id
    SELECT 
        cr.id,
        a.id AS app_id
    FROM casbin_rule cr
    INNER JOIN app a ON (
        -- 解析路径：/user/app/... 或 /user/app/*
        a.user = SUBSTRING_INDEX(SUBSTRING_INDEX(cr.v1, '/', 2), '/', -1)
        AND a.code = SUBSTRING_INDEX(SUBSTRING_INDEX(cr.v1, '/', 3), '/', -1)
    )
    WHERE cr.ptype = 'p' 
      AND cr.app_id IS NULL
      AND cr.v1 LIKE '/%/%/%'  -- 确保路径格式正确（至少包含 /user/app/...）
) AS matched ON cr.id = matched.id
SET cr.app_id = matched.app_id;

-- 查询修复结果
SELECT 
    COUNT(*) AS total_fixed,
    COUNT(DISTINCT app_id) AS unique_apps
FROM casbin_rule
WHERE ptype = 'p' AND app_id IS NOT NULL;

-- 查询仍为 NULL 的记录（需要人工检查）
SELECT 
    id,
    v0 AS username,
    v1 AS resource_path,
    v2 AS action,
    app_id
FROM casbin_rule
WHERE ptype = 'p' AND app_id IS NULL
LIMIT 100;

