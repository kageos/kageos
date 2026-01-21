# 工作空间根节点迁移脚本说明

## 📋 概述

本迁移脚本用于将工作空间根节点统一到 `service_tree` 表中，作为 `package` 类型节点处理，从而简化代码逻辑并统一权限管理。

## 🎯 迁移目标

1. 为每个 `app` 创建对应的 `service_tree` 根节点
2. 根节点特征：`parent_id = 0`, `ref_id = app_id`, `type = 'package'`
3. 将原有根级子节点的 `parent_id` 指向新创建的根节点
4. 保持数据完整性和一致性

## 📁 文件说明

- **workspace_root_node.sql**：主迁移脚本（包含检查、迁移、验证）
- **workspace_root_node_rollback.sql**：回滚脚本（恢复到迁移前状态）
- **README_workspace_root_node.md**：本说明文档

## 🚀 使用方法

### 方法 1：使用 MySQL 客户端（推荐）

```bash
# 1. 连接到数据库
mysql -h localhost -u root -p ai_agent_os

# 2. 执行迁移脚本
source /path/to/workspace_root_node.sql;

# 3. 查看执行结果，确认验证通过
# （脚本会自动输出验证结果）

# 4. 如果需要回滚
source /path/to/workspace_root_node_rollback.sql;
```

### 方法 2：使用命令行直接执行

```bash
# 执行迁移
mysql -h localhost -u root -p ai_agent_os < workspace_root_node.sql

# 如果需要回滚
mysql -h localhost -u root -p ai_agent_os < workspace_root_node_rollback.sql
```

### 方法 3：在 Navicat 等工具中执行

1. 打开 Navicat，连接到数据库
2. 新建查询，粘贴 `workspace_root_node.sql` 的内容
3. 执行查询
4. 查看结果，确认迁移成功

## ✅ 验证步骤

迁移脚本会自动执行以下验证：

### 1. 检查所有 app 是否都有根节点

**预期结果**：所有 app 的 `root_count` 都应该是 1

```sql
-- 示例输出
app_id | user   | code       | root_count | status
-------|--------|------------|------------|--------
129    | luobei | operations | 1          | ✅ 正常
130    | system | official   | 1          | ✅ 正常
```

### 2. 检查是否还有孤立的根级子节点

**预期结果**：所有 `parent_id=0` 的节点都应该是根节点（`ref_id = app_id`）

```sql
-- 示例输出
id    | name     | app_id | parent_id | ref_id | status
------|----------|--------|-----------|--------|------------
10001 | 运营中心 | 129    | 0         | 129    | ✅ 这是根节点
10002 | 官方空间 | 130    | 0         | 130    | ✅ 这是根节点
```

### 3. 检查根节点的子节点数量

**预期结果**：根节点应该有对应的子节点

```sql
-- 示例输出
app_id | user   | code       | root_node_id | children_count
-------|--------|------------|--------------|---------------
129    | luobei | operations | 10001        | 5
130    | system | official   | 10002        | 10
```

### 4. 统计迁移结果

**预期结果**：`migration_status` 应该是 `✅ 迁移成功`

```sql
-- 示例输出
total_apps | apps_with_root | total_root_nodes | migration_status
-----------|----------------|------------------|------------------
10         | 10             | 10               | ✅ 迁移成功
```

## 🔄 回滚步骤

如果迁移后发现问题，可以使用回滚脚本恢复：

```bash
# 执行回滚脚本
mysql -h localhost -u root -p ai_agent_os < workspace_root_node_rollback.sql
```

**回滚操作**：
1. 将子节点的 `parent_id` 恢复为 0
2. 删除迁移时创建的根节点
3. 验证回滚结果

## ⚠️ 注意事项

### 迁移前

1. **备份数据库**：在生产环境执行前，务必先备份数据库
   ```bash
   mysqldump -u root -p ai_agent_os > backup_$(date +%Y%m%d_%H%M%S).sql
   ```

2. **在测试环境验证**：先在测试环境执行，确认无误后再在生产环境执行

3. **检查数据完整性**：确保 `app` 表和 `service_tree` 表数据完整

### 迁移中

1. **关注执行结果**：注意查看每一步的执行结果和影响行数

2. **检查验证输出**：确保所有验证步骤都通过

3. **记录执行日志**：保存执行日志，方便问题排查

### 迁移后

1. **应用程序兼容**：确保应用程序代码已更新，支持新的根节点结构

2. **功能测试**：
   - 测试工作空间访问
   - 测试权限申请和审批
   - 测试服务树显示和操作

3. **性能监控**：观察服务树查询和权限计算的性能

## 📊 数据结构变化

### 迁移前

```
app 表
├─ id: 129, user: "luobei", code: "operations"

service_tree 表
├─ id: 1217, parent_id: 0, type: "package", app_id: 129 (根级子节点)
├─ id: 1218, parent_id: 1217, type: "function", app_id: 129
```

### 迁移后

```
app 表
├─ id: 129, user: "luobei", code: "operations"

service_tree 表
├─ id: 10001, parent_id: 0, type: "package", app_id: 129, ref_id: 129 ⭐ (根节点)
│   ├─ id: 1217, parent_id: 10001, type: "package", app_id: 129
│   └─ id: 1218, parent_id: 1217, type: "function", app_id: 129
```

**关键变化**：
- 新增了根节点（`ref_id = app_id` 标识）
- 原有根级子节点的 `parent_id` 从 0 改为指向根节点

## 🐛 常见问题

### Q1：迁移后发现某些 app 没有根节点？

**排查**：
```sql
SELECT a.* 
FROM app a
WHERE NOT EXISTS (
    SELECT 1 FROM service_tree st 
    WHERE st.app_id = a.id AND st.parent_id = 0 AND st.ref_id = a.id
);
```

**解决**：手动为这些 app 创建根节点
```sql
INSERT INTO service_tree (name, code, parent_id, type, app_id, ref_id, full_code_path, version, version_num, created_at, updated_at)
VALUES ('app名称', 'app_code', 0, 'package', app_id, app_id, '/user/app_code', 'v1', 1, NOW(), NOW());
```

### Q2：迁移后出现根节点重复？

**排查**：
```sql
SELECT app_id, COUNT(*) as root_count
FROM service_tree
WHERE parent_id = 0 AND ref_id = app_id
GROUP BY app_id
HAVING root_count > 1;
```

**解决**：删除重复的根节点，只保留一个
```sql
-- 保留 id 最小的根节点，删除其他
DELETE FROM service_tree 
WHERE id NOT IN (
    SELECT * FROM (
        SELECT MIN(id) 
        FROM service_tree 
        WHERE parent_id = 0 AND ref_id = app_id 
        GROUP BY app_id
    ) as t
) AND parent_id = 0 AND ref_id = app_id;
```

### Q3：迁移后还有孤立的根级子节点（parent_id=0 但不是根节点）？

**排查**：
```sql
SELECT * FROM service_tree
WHERE parent_id = 0
  AND (ref_id IS NULL OR ref_id != app_id);
```

**解决**：将这些节点的 `parent_id` 指向对应的根节点
```sql
UPDATE service_tree st
SET parent_id = (
    SELECT id FROM service_tree 
    WHERE app_id = st.app_id AND parent_id = 0 AND ref_id = app_id
    LIMIT 1
)
WHERE parent_id = 0
  AND (ref_id IS NULL OR ref_id != app_id);
```

## 📝 相关文档

- **架构分析**：`note/临时分析/01_工作空间根节点架构优化方案.md`
- **深度评估**：`note/临时分析/02_架构优化方案深度评估.md`
- **实施计划**：`note/todos/03_工作空间根节点重构实施计划.doing.md`

## 📞 联系方式

如果遇到问题，请查看：
1. 验证输出结果
2. 常见问题部分
3. 相关文档

## 📅 版本历史

- **v1.0.0** (2026-01-21)：初始版本，支持基本迁移和回滚

---

**最后更新**：2026-01-21  
**维护者**：AI Agent + 洛北
