-- ========================================
-- 服务树即知识库 - 数据库迁移脚本
-- 目标：删除知识库系统，统一使用服务树管理文档
-- 作者：AI Agent + 洛北
-- 日期：2026-01-21
-- ========================================

-- ========================================
-- Step 0: 迁移前检查
-- ========================================

-- 检查当前知识库数量
SELECT 
    '迁移前检查：knowledge_base 表总数' as info,
    COUNT(*) as kb_count
FROM knowledge_base;

-- 检查当前知识库文档数量
SELECT 
    '迁移前检查：knowledge_base_document 表总数' as info,
    COUNT(*) as doc_count
FROM knowledge_base_document;

-- 检查当前智能体绑定知识库的情况
SELECT 
    '迁移前检查：智能体绑定知识库情况' as info,
    COUNT(*) as total_agents,
    COUNT(CASE WHEN knowledge_base_id IS NOT NULL THEN 1 END) as agents_with_kb
FROM agents;

-- ========================================
-- Step 1: 修改 agents 表
-- ========================================

-- 1.1 新增 docs_paths 字段（JSON 数组格式）
ALTER TABLE agents
  ADD COLUMN docs_paths TEXT COMMENT '文档路径数组（JSON 格式，如 ["​/system/official/sdk", "/user/myapp/docs"]）';

-- 1.2 迁移现有数据（将 knowledge_base_id 转换为 docs_paths）
-- 假设 knowledge_base_id = 1 对应标准库
UPDATE agents 
SET docs_paths = '["​/system/official/sdk"]'
WHERE knowledge_base_id = 1;

-- 其他知识库的智能体，暂时也指向标准库（后续可以手动调整）
UPDATE agents 
SET docs_paths = '["​/system/official/sdk"]'
WHERE knowledge_base_id IS NOT NULL AND knowledge_base_id != 1;

-- 未绑定知识库的智能体，使用默认标准库
UPDATE agents 
SET docs_paths = '["​/system/official/sdk"]'
WHERE knowledge_base_id IS NULL OR knowledge_base_id = 0;

-- 1.3 删除 knowledge_base_id 字段
ALTER TABLE agents
  DROP COLUMN knowledge_base_id;

-- ========================================
-- Step 2: 修改 service_tree 表
-- ========================================

-- 2.1 新增 is_standard_lib 字段
ALTER TABLE service_tree
  ADD COLUMN is_standard_lib TINYINT(1) DEFAULT 0 COMMENT '是否标准库节点（自动对所有用户开放权限）';

-- 2.2 创建索引
CREATE INDEX idx_is_standard_lib ON service_tree(is_standard_lib);
CREATE INDEX idx_full_code_path ON service_tree(full_code_path(255));

-- ========================================
-- Step 3: 创建标准库目录结构
-- ========================================

-- 3.1 创建系统根目录
INSERT INTO service_tree (name, code, parent_id, type, is_standard_lib, full_code_path, app_id, created_by, updated_by, created_at, updated_at)
SELECT 
    '系统',
    'system',
    0,
    'package',
    1,
    '/system',
    1,
    'system',
    'system',
    NOW(),
    NOW()
WHERE NOT EXISTS (SELECT 1 FROM service_tree WHERE full_code_path = '/system');

-- 3.2 创建官方标准库目录
INSERT INTO service_tree (name, code, parent_id, type, is_standard_lib, full_code_path, app_id, created_by, updated_by, created_at, updated_at)
SELECT 
    '官方标准库',
    'official',
    (SELECT id FROM service_tree WHERE full_code_path = '/system' LIMIT 1),
    'package',
    1,
    '/system/official',
    1,
    'system',
    'system',
    NOW(),
    NOW()
WHERE NOT EXISTS (SELECT 1 FROM service_tree WHERE full_code_path = '/system/official');

-- 3.3 创建 SDK 文档目录
INSERT INTO service_tree (name, code, parent_id, type, is_standard_lib, full_code_path, app_id, created_by, updated_by, created_at, updated_at)
SELECT 
    'SDK 文档',
    'sdk',
    (SELECT id FROM service_tree WHERE full_code_path = '/system/official' LIMIT 1),
    'package',
    1,
    '/system/official/sdk',
    1,
    'system',
    'system',
    NOW(),
    NOW()
WHERE NOT EXISTS (SELECT 1 FROM service_tree WHERE full_code_path = '/system/official/sdk');

-- 3.4 创建插件库目录
INSERT INTO service_tree (name, code, parent_id, type, is_standard_lib, full_code_path, app_id, created_by, updated_by, created_at, updated_at)
SELECT 
    '插件库',
    'plugins',
    (SELECT id FROM service_tree WHERE full_code_path = '/system/official' LIMIT 1),
    'package',
    1,
    '/system/official/plugins',
    1,
    'system',
    'system',
    NOW(),
    NOW()
WHERE NOT EXISTS (SELECT 1 FROM service_tree WHERE full_code_path = '/system/official/plugins');

-- 3.5 创建模板库目录
INSERT INTO service_tree (name, code, parent_id, type, is_standard_lib, full_code_path, app_id, created_by, updated_by, created_at, updated_at)
SELECT 
    '模板库',
    'templates',
    (SELECT id FROM service_tree WHERE full_code_path = '/system/official' LIMIT 1),
    'package',
    1,
    '/system/official/templates',
    1,
    'system',
    'system',
    NOW(),
    NOW()
WHERE NOT EXISTS (SELECT 1 FROM service_tree WHERE full_code_path = '/system/official/templates');

-- ========================================
-- Step 4: 标记现有的标准库节点
-- ========================================

-- 标记所有 /system/official/* 路径下的节点为标准库节点
UPDATE service_tree 
SET is_standard_lib = 1 
WHERE full_code_path LIKE '/system/official/%';

-- ========================================
-- Step 5: 删除知识库相关表
-- ========================================

-- 5.1 删除知识库文档表
DROP TABLE IF EXISTS knowledge_base_document;

-- 5.2 删除知识库表
DROP TABLE IF EXISTS knowledge_base;

-- ========================================
-- Step 6: 迁移后验证
-- ========================================

-- 验证 1：检查所有智能体是否都有 docs_paths
SELECT 
    '验证：智能体文档路径' as check_name,
    COUNT(*) as total_agents,
    COUNT(CASE WHEN docs_paths IS NOT NULL AND docs_paths != '' THEN 1 END) as agents_with_docs_paths,
    COUNT(CASE WHEN docs_paths IS NULL OR docs_paths = '' THEN 1 END) as agents_without_docs_paths
FROM agents;

-- 验证 2：检查标准库节点数量
SELECT 
    '验证：标准库节点数量' as check_name,
    COUNT(*) as total_standard_lib_nodes
FROM service_tree
WHERE is_standard_lib = 1;

-- 验证 3：检查标准库目录结构
SELECT 
    '验证：标准库目录结构' as check_name,
    id,
    name,
    code,
    type,
    full_code_path,
    is_standard_lib
FROM service_tree
WHERE full_code_path LIKE '/system/%'
ORDER BY full_code_path;

-- 验证 4：检查知识库表是否已删除
SELECT 
    '验证：知识库表是否已删除' as check_name,
    CASE 
        WHEN COUNT(*) = 0 THEN '✅ 知识库表已删除'
        ELSE '❌ 知识库表仍存在'
    END as status
FROM information_schema.tables
WHERE table_schema = DATABASE()
  AND table_name IN ('knowledge_base', 'knowledge_base_document');

-- 验证 5：统计迁移结果
SELECT 
    '迁移结果统计' as info,
    (SELECT COUNT(*) FROM agents) as total_agents,
    (SELECT COUNT(*) FROM agents WHERE docs_paths IS NOT NULL AND docs_paths != '') as agents_with_docs_paths,
    (SELECT COUNT(*) FROM service_tree WHERE is_standard_lib = 1) as standard_lib_nodes,
    CASE 
        WHEN (SELECT COUNT(*) FROM agents WHERE docs_paths IS NULL OR docs_paths = '') = 0 
        THEN '✅ 迁移成功'
        ELSE '⚠️ 部分智能体未设置 docs_paths'
    END as migration_status;

-- ========================================
-- 迁移完成提示
-- ========================================

SELECT 
    '========================================' as separator
UNION ALL
SELECT '✅ 迁移完成！' as message
UNION ALL
SELECT '主要改动：' as message
UNION ALL
SELECT '1. agents 表：删除 knowledge_base_id，新增 docs_paths（JSON 数组）' as message
UNION ALL
SELECT '2. service_tree 表：新增 is_standard_lib（标准库节点标识）' as message
UNION ALL
SELECT '3. 创建标准库目录：/system/official/sdk、/system/official/plugins、/system/official/templates' as message
UNION ALL
SELECT '4. 删除表：knowledge_base、knowledge_base_document' as message
UNION ALL
SELECT '========================================' as separator;
