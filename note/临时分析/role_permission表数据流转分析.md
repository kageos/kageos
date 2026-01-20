# role_permission 表数据流转分析

## 表结构说明

### 三张核心表

```
1. role (角色表)
   - 存储角色定义
   - 例如：目录开发者、函数开发者、表格管理员等

2. role_permission (角色-权限点关联表) ⭐ 本文重点
   - 存储角色和权限点（action）的关系
   - role_id -> action_id
   - 一个角色可以有多个权限点

3. role_assignment (角色分配表)
   - 存储用户/组织架构与角色、资源路径的关系
   - 记录：谁（subject）在哪个资源（resource_path）上拥有什么角色（role_id）
```

## 一、role_permission 表的写入场景

### 场景 1: 创建角色时写入

**触发时机**：调用 `CreateRole` API 创建新角色

**代码位置**：`enterprise_impl/permission/service/role_service.go:46-120`

**写入逻辑**：
```go
// 为角色的每个权限点（action）创建一条 role_permission 记录
for permResourceType, actions := range req.Permissions {
    for _, actionCode := range actions {
        // 查询 Action 表获取 ActionID
        action, err := s.actionRepo.GetActionByCode(ctx, actionCode)
        
        // 创建 role_permission 记录
        rolePerm := &model.RolePermission{
            RoleID:   role.ID,      // 角色 ID
            ActionID: action.ID,    // 权限点 ID（外键）
        }
        s.rolePermissionRepo.CreateRolePermission(ctx, rolePerm)
    }
}
```

**示例**：
```
创建一个"目录开发者"角色，配置了 5 个权限点：
- directory:read
- directory:write
- directory:update
- directory:delete
- directory:admin

结果：在 role_permission 表中写入 5 条记录
┌─────────┬──────────┐
│ role_id │ action_id│
├─────────┼──────────┤
│   123   │    1     │ (directory:read)
│   123   │    2     │ (directory:write)
│   123   │    3     │ (directory:update)
│   123   │    4     │ (directory:delete)
│   123   │    5     │ (directory:admin)
└─────────┴──────────┘
```

### 场景 2: 更新角色时重写

**触发时机**：调用 `UpdateRole` API 更新角色的权限配置

**代码位置**：`enterprise_impl/permission/service/role_service.go:123-207`

**写入逻辑**：
```go
// 1. 先删除该角色的所有旧权限
if err := s.rolePermissionRepo.DeleteByRoleID(ctx, roleID); err != nil {
    return nil, fmt.Errorf("删除旧权限失败: %w", err)
}

// 2. 然后添加新权限（逻辑同创建角色）
for permResourceType, actions := range *req.Permissions {
    for _, actionCode := range actions {
        // ... 创建新的 role_permission 记录
    }
}
```

**示例**：
```
更新"目录开发者"角色，将权限从 5 个减少到 3 个：
- directory:read
- directory:write
- directory:update

操作过程：
1. 删除旧的 5 条 role_permission 记录
2. 创建新的 3 条 role_permission 记录

结果：表中该角色只保留 3 条记录
```

### 场景 3: 初始化系统预设角色时写入

**触发时机**：系统启动时调用 `EnsureSystemRolesExist`

**代码位置**：`enterprise_impl/permission/service/role_service.go:606-773`

**写入逻辑**：
```go
// 如果角色已存在，先删除旧权限
if err := s.rolePermissionRepo.DeleteByRoleID(ctx, role.ID); err != nil {
    return fmt.Errorf("删除预设角色旧权限失败: %w", err)
}

// 然后添加新权限（逻辑同创建角色）
```

## 二、role_permission 表的删除场景

### 场景 1: 删除角色时删除

**触发时机**：调用 `DeleteRole` API 删除角色

**代码位置**：`enterprise_impl/permission/service/role_service.go:209-239`

**删除逻辑**：
```go
// 1. 先删除该角色的所有 role_permission 记录
if err := s.rolePermissionRepo.DeleteByRoleID(ctx, roleID); err != nil {
    return fmt.Errorf("删除角色权限失败: %w", err)
}

// 2. 然后删除角色本身
if err := s.roleRepo.DeleteRole(ctx, roleID); err != nil {
    return fmt.Errorf("删除角色失败: %w", err)
}
```

**SQL 实现**：
```sql
-- DeleteByRoleID 的 SQL 实现
DELETE FROM role_permission WHERE role_id = ?
```

**示例**：
```
删除"目录开发者"角色（role_id = 123）：
1. 删除 role_permission 表中所有 role_id = 123 的记录（5 条）
2. 删除 role 表中的角色记录（1 条）
```

### 场景 2: 更新角色时删除旧权限

**触发时机**：调用 `UpdateRole` API（见上文"场景 2: 更新角色时重写"）

**删除逻辑**：先删除旧权限，再创建新权限

## 三、权限申请审批通过时的数据流

### 核心问题：申请一条记录审批通过后，会写入几条 role_permission 记录？

**答案：0 条！** ⭐

**原因**：审批通过时，**不会往 role_permission 表写入记录**，而是往 **role_assignment 表**写入记录。

### 完整流程分析

**1. 用户申请权限**
```
用户 A 申请"目录开发者"角色，资源路径：/system/official/agent/table
```

**2. 创建权限申请记录**
```
写入 permission_request 表：
┌────┬──────────┬─────────┬───────────┬──────────────────────────────┬─────────┐
│ id │ username │ role_id │ subject   │ resource_path                │ status  │
├────┼──────────┼─────────┼───────────┼──────────────────────────────┼─────────┤
│ 1  │ userA    │   123   │ userA     │ /system/official/agent/table │ pending │
└────┴──────────┴─────────┴───────────┴──────────────────────────────┴─────────┘
```

**3. 管理员审批通过**
```go
// 代码位置：enterprise_impl/permission/service/approval_service.go:91-214

// 审批通过的核心操作：
s.roleService.AssignRoleToUser(ctx, &dto.AssignRoleToUserReq{
    User:         user,
    App:          app,
    Username:     request.Subject,
    RoleCode:     role.Code,
    ResourceType: role.ResourceType,
    ResourcePath: request.ResourcePath,
    StartTime:    &startTime,
    EndTime:      endTime,
})
```

**4. 写入 role_assignment 表（不是 role_permission 表！）**
```
写入 role_assignment 表（1 条记录）：
┌────┬──────┬──────────┬──────────────┬─────────┬─────────┬──────────────────────────────┬────────────┬──────────┐
│ id │ user │ app      │ subject_type │ subject │ role_id │ resource_path                │ start_time │ end_time │
├────┼──────┼──────────┼──────────────┼─────────┼─────────┼──────────────────────────────┼────────────┼──────────┤
│ 1  │ sys  │ official │ user         │ userA   │   123   │ /system/official/agent/table │ 2025-01-20 │ NULL     │
└────┴──────┴──────────┴──────────────┴─────────┴─────────┴──────────────────────────────┴────────────┴──────────┘

同时更新 permission_request 表：
- status: pending -> approved
- approver_username: adminB
- role_assignment_id: 1
```

**5. 权限计算时的关联查询**
```
当用户 A 访问 /system/official/agent/table 时：

1. 查询 role_assignment 表：
   找到 userA 在该资源路径上的角色（role_id = 123）

2. 查询 role_permission 表：
   找到 role_id = 123 的所有权限点（5 条记录）

3. 最终权限：
   userA 在 /system/official/agent/table 上拥有：
   - directory:read
   - directory:write
   - directory:update
   - directory:delete
   - directory:admin
```

## 四、为什么 role_permission 表会有很多删除记录？

根据上述分析，role_permission 表的记录删除主要发生在以下场景：

### 1. 角色配置调整（最常见）
```
管理员调整角色权限配置，例如：
- 将"目录开发者"的权限从 5 个改为 3 个
- 将"函数开发者"的权限从 3 个改为 5 个

每次调整都会：
1. 删除旧的所有 role_permission 记录
2. 创建新的 role_permission 记录
```

**操作频率**：如果经常调整角色配置，会产生大量的删除和写入操作。

### 2. 删除角色
```
删除不再使用的角色，例如：
- 删除"临时开发者"角色
- 删除"测试角色"

每次删除角色都会：
1. 删除该角色的所有 role_permission 记录
2. 删除该角色本身
```

### 3. 系统预设角色更新
```
系统启动时，如果预设角色的权限配置有变化：
1. 删除旧的 role_permission 记录
2. 创建新的 role_permission 记录
```

## 五、数据统计和查询

### 查询某个角色有多少权限点
```sql
SELECT COUNT(*) FROM role_permission WHERE role_id = 123;
-- 结果：5（假设该角色配置了 5 个权限点）
```

### 查询所有角色的权限点总数
```sql
SELECT role_id, COUNT(*) as permission_count 
FROM role_permission 
GROUP BY role_id;

-- 结果示例：
-- role_id | permission_count
-- --------|------------------
--   123   |        5
--   124   |        3
--   125   |        7
```

### 查询某个用户在某个资源路径上有哪些权限
```sql
-- 1. 查找用户的角色分配
SELECT ra.role_id 
FROM role_assignment ra
WHERE ra.user = 'system' 
  AND ra.app = 'official'
  AND ra.subject = 'userA'
  AND ra.resource_path = '/system/official/agent/table';

-- 2. 查找角色的权限点
SELECT rp.action_id, a.code, a.name
FROM role_permission rp
JOIN action a ON rp.action_id = a.id
WHERE rp.role_id IN (123); -- 从上一步查询得到的 role_id

-- 结果示例：
-- action_id | code              | name
-- ----------|-------------------|-------------
--     1     | directory:read    | 目录读取
--     2     | directory:write   | 目录写入
--     3     | directory:update  | 目录更新
--     4     | directory:delete  | 目录删除
--     5     | directory:admin   | 目录管理
```

## 六、关键总结

### 1. role_permission 表的特点
- ✅ 存储**角色**和**权限点（action）**的关系
- ✅ 一个角色可以有多个权限点（一对多关系）
- ✅ 通过 `action_id` 外键关联到 `action` 表

### 2. 审批通过后的数据流
- ❌ **不会往 role_permission 表写入记录**
- ✅ **会往 role_assignment 表写入 1 条记录**
- ✅ 记录的是：用户/组织架构 + 角色 + 资源路径的关系

### 3. role_permission 记录数量计算
```
假设系统中有 10 个角色：
- 每个角色平均配置了 5 个权限点

role_permission 表中的记录数 = 10 × 5 = 50 条

审批通过 100 个权限申请后：
- role_permission 表记录数：50 条（不变！）
- role_assignment 表记录数：100 条（每个申请 1 条）
```

### 4. 删除记录的原因
- ⭐ 更新角色权限配置（先删除旧权限，再创建新权限）
- ⭐ 删除角色（删除该角色的所有权限点）
- ⭐ 系统预设角色更新（先删除旧权限，再创建新权限）

### 5. 与权限申请的关系
```
权限申请流程：
1. 用户申请权限（选择角色 + 资源路径）
2. 管理员审批
3. 审批通过：写入 role_assignment 表（1 条记录）
4. 权限计算：role_assignment + role_permission + action

关键点：
- role_permission 表在角色创建时就已经写入
- 权限申请审批只影响 role_assignment 表
- role_permission 表记录数取决于角色数量和配置，与审批数量无关
```

## 七、最佳实践建议

### 1. 角色设计
- ✅ 预先设计好角色和权限点，避免频繁修改
- ✅ 使用系统预设角色（如：开发者、管理员、访客）
- ✅ 只在必要时创建自定义角色

### 2. 权限配置
- ✅ 一次性配置好角色的权限点，避免频繁更新
- ✅ 测试环境充分测试后再应用到生产环境
- ❌ 避免频繁删除和重建角色

### 3. 数据库维护
- ✅ 定期清理无用的角色（会自动删除 role_permission 记录）
- ✅ 监控 role_permission 表的增长趋势
- ✅ 如果发现大量删除操作，检查是否有频繁的角色配置调整

## 八、问题排查

### 问题 1: 为什么 role_permission 表有很多记录？
**原因**：角色数量多，或者某些角色配置了很多权限点

**排查**：
```sql
-- 查看哪个角色配置了最多的权限点
SELECT role_id, COUNT(*) as count 
FROM role_permission 
GROUP BY role_id 
ORDER BY count DESC 
LIMIT 10;
```

### 问题 2: 为什么 role_permission 表有很多删除操作？
**原因**：频繁更新角色配置，或者频繁删除角色

**排查**：
- 检查应用日志，搜索 `UpdateRole` 和 `DeleteRole` 的调用频率
- 检查是否有自动化脚本在频繁修改角色配置

### 问题 3: 审批通过后，为什么 role_permission 表没有新记录？
**原因**：这是正常的！审批通过只影响 role_assignment 表

**验证**：
```sql
-- 查看最近的角色分配记录
SELECT * FROM role_assignment 
ORDER BY created_at DESC 
LIMIT 10;
```
