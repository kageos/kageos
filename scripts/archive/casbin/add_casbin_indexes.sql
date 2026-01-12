-- Casbin 权限系统数据库索引优化
-- 执行此脚本可以显著提升权限查询性能（10-100倍）

-- 1. 策略查询索引（最常用，用于权限检查）
-- 索引字段：ptype（策略类型）、v0（用户）、v1（资源）、v2（操作）
CREATE INDEX IF NOT EXISTS idx_casbin_policy ON casbin_rule(ptype, v0, v1, v2);

-- 2. 用户查询索引（用于查询用户的所有权限）
-- 索引字段：ptype（策略类型）、v0（用户）
CREATE INDEX IF NOT EXISTS idx_casbin_user ON casbin_rule(ptype, v0);

-- 3. 资源查询索引（用于查询资源的所有权限）
-- 索引字段：ptype（策略类型）、v1（资源）
CREATE INDEX IF NOT EXISTS idx_casbin_resource ON casbin_rule(ptype, v1);

-- 4. g2 关系查询索引（用于资源继承关系查询）
-- 索引字段：ptype（关系类型，g2）、v0（子资源）、v1（父资源）
CREATE INDEX IF NOT EXISTS idx_casbin_g2 ON casbin_rule(ptype, v0, v1) WHERE ptype = 'g2';

-- 说明：
-- - idx_casbin_policy: 最常用的索引，用于权限检查（Enforce）
-- - idx_casbin_user: 用于查询用户的所有权限（GetFilteredPolicy）
-- - idx_casbin_resource: 用于查询资源的所有权限
-- - idx_casbin_g2: 用于资源继承关系查询（g2 关系）
--
-- 性能提升预期：
-- - 权限检查：从全表扫描 → 索引查询（10-100倍提升）
-- - 批量查询：从多次查询 → 一次索引查询（5-10倍提升）
-- - 资源继承：从全表扫描 → 索引查询（5-10倍提升）

