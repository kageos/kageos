# 权限表改名分析：action -> role_action

## 📋 当前表结构

### 权限系统相关表

```
1. action               - 权限点表（存储所有权限点定义）
2. role                 - 角色表（存储角色定义）
3. role_permission      - 角色权限关联表（role_id + action_id）
4. role_assignment      - 角色分配表（用户/组织架构 + 角色 + 资源路径）
5. permission_request   - 权限申请表（权限申请审批流程）
```

### 表关系图

```
┌─────────────────────────────────────────────────────────────────┐
│                         权限系统表关系                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  action (权限点表)                                               │
│    ├─ id                                                        │
│    ├─ code (directory:read, table:write)                       │
│    ├─ name (目录读取、表格写入)                                  │
│    └─ resource_type (directory, table, form, chart, app, docs) │
│                                                                 │
│  ↓ (外键：action_id)                                            │
│                                                                 │
│  role_permission (角色权限关联表)                                │
│    ├─ role_id → role.id                                         │
│    └─ action_id → action.id                                     │
│                                                                 │
│  ↓ (外键：role_id)                                              │
│                                                                 │
│  role (角色表)                                                   │
│    ├─ id                                                        │
│    ├─ code (viewer, developer, admin)                          │
│    └─ resource_type (directory, table, form, chart, app, docs) │
│                                                                 │
│  ↓ (外键：role_id)                                              │
│                                                                 │
│  role_assignment (角色分配表)                                    │
│    ├─ role_id → role.id                                         │
│    ├─ subject (用户名/组织架构路径)                              │
│    └─ resource_path (资源路径)                                   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## 🎯 改名建议

### 方案对比

| 表名 | 当前名称 | 建议改名 | 理由 |
|------|---------|---------|------|
| 权限点表 | **action** | `role_action` | ❌ **不建议** |
| 角色权限表 | **role_permission** | `role_action_permission` | ❌ **不建议** |

### 为什么不建议改名？

#### 理由 1：action 表的独立性和通用性

**action 表不仅仅服务于 role 系统**：

1. **权限点是独立的概念**
   - `action` 表存储的是**系统中所有可能的权限点**
   - 它是一个**基础数据表**，定义了系统支持的所有操作
   - 不仅 `role` 系统使用，未来其他权限模型也可能使用

2. **未来扩展性**
   - 如果未来引入其他权限模型（如 ABAC、ACL），`action` 表仍然适用
   - 改名为 `role_action` 会限制其使用范围
   - `action` 是一个更通用的命名

3. **语义清晰性**
   ```
   action               ✅ 清晰：这是一个权限点/操作
   role_action          ❌ 混淆：听起来像是"角色操作"，而不是"权限点"
   ```

#### 理由 2：role_permission 表已经很清晰

**当前命名已经表达了正确的语义**：

1. **`role_permission` 的语义**
   - `role` + `permission`：角色的权限
   - 清晰表达了"角色拥有哪些权限"的概念
   - `permission` 在这里指的是"角色的权限配置"

2. **`role_action_permission` 的问题**
   - 名字过长，不易读
   - `action` 和 `permission` 语义重复（action 本质上就是一种 permission）
   - 容易让人混淆：这是"角色-操作-权限"还是"角色的操作权限"？

#### 理由 3：数据库命名的通用规范

**参考业界最佳实践**：

1. **多对多关联表命名**
   - 格式：`table1_table2`
   - 例如：`user_role`（用户和角色的关联）
   - 例如：`role_permission`（角色和权限的关联）

2. **基础数据表命名**
   - 单数名词：`action`、`role`、`user`
   - 不需要加前缀（除非有命名冲突）

3. **前缀的使用场景**
   - 同一业务的**业务表**：`order_item`、`order_payment`
   - 避免命名冲突：`sys_user`、`app_config`
   - **基础数据表不需要前缀**

#### 理由 4：改名成本高，收益低

**改名带来的成本**：

1. **代码修改量大**
   - 所有引用 `action` 表的代码都需要修改
   - 所有引用 `role_permission` 表的代码都需要修改
   - Repository、Service、DTO、API 层都需要修改

2. **数据库迁移风险**
   - 需要重命名表
   - 需要重命名外键约束
   - 需要重命名索引
   - 可能影响正在运行的查询

3. **收益不明显**
   - 只是改了个名字，功能没有任何变化
   - 没有解决实际问题
   - 没有提升性能或可维护性

## ✅ 推荐方案：保持现有命名

### 当前命名的优点

| 表名 | 优点 |
|------|------|
| `action` | ✅ 简洁清晰<br>✅ 语义明确（权限点/操作）<br>✅ 通用性强（不局限于 role）<br>✅ 符合业界规范 |
| `role_permission` | ✅ 语义清晰（角色的权限）<br>✅ 符合多对多关联表命名规范<br>✅ 长度适中 |
| `role_assignment` | ✅ 语义清晰（角色分配）<br>✅ 表达了"分配"的动作 |

### 当前命名体系的逻辑

```
基础表（无前缀）：
  - action     (权限点)
  - role       (角色)
  - user       (用户)
  
关联表（table1_table2）：
  - role_permission   (角色 <-> 权限点)
  - user_role         (如果有的话：用户 <-> 角色)
  
业务表（有前缀）：
  - role_assignment   (角色分配业务)
  - permission_request (权限申请业务)
```

这个命名体系是**清晰且符合业界规范**的！

## 🚫 不推荐方案：改名为 role_action

### 如果改名，需要做的事情

#### 1. 修改 Model 定义

**文件**：`core/app-server/model/action.go`
```go
// 修改前
func (*Action) TableName() string {
    return "action"
}

// 修改后
func (*Action) TableName() string {
    return "role_action"  // ⚠️ 不推荐
}
```

**文件**：`core/app-server/model/role.go`
```go
// 修改前
func (*RolePermission) TableName() string {
    return "role_permission"
}

// 修改后
func (*RolePermission) TableName() string {
    return "role_action_permission"  // ⚠️ 不推荐
}
```

#### 2. 数据库迁移脚本

**GORM AutoMigrate 无法自动重命名表**，需要手动写迁移脚本：

```go
// 数据库迁移：重命名表
func MigrateRenameActionTable(db *gorm.DB) error {
    // 1. 重命名 action 表
    if err := db.Exec("ALTER TABLE action RENAME TO role_action").Error; err != nil {
        return fmt.Errorf("重命名 action 表失败: %w", err)
    }
    
    // 2. 重命名 role_permission 表
    if err := db.Exec("ALTER TABLE role_permission RENAME TO role_action_permission").Error; err != nil {
        return fmt.Errorf("重命名 role_permission 表失败: %w", err)
    }
    
    // 3. 更新外键约束名称（如果有的话）
    // ALTER TABLE role_action_permission DROP FOREIGN KEY fk_action_id;
    // ALTER TABLE role_action_permission ADD CONSTRAINT fk_role_action_id 
    //   FOREIGN KEY (action_id) REFERENCES role_action(id);
    
    // 4. 更新索引名称（如果需要的话）
    // ...
    
    return nil
}
```

#### 3. 影响范围评估

**需要修改的文件（估计）**：

1. **Model 层**（2 个文件）
   - `core/app-server/model/action.go`
   - `core/app-server/model/role.go`

2. **Repository 层**（2 个文件）
   - `enterprise_impl/permission/repository/action_repository.go`
   - `enterprise_impl/permission/repository/role_permission_repository.go`

3. **Service 层**（3 个文件）
   - `enterprise_impl/permission/service/action_service.go`
   - `enterprise_impl/permission/service/role_service.go`
   - `enterprise_impl/permission/service/role_cache.go`

4. **迁移脚本**（1 个文件）
   - 需要新建迁移脚本

5. **测试文件**（如果有的话）
   - 所有测试文件中引用这些表的地方

**总计**：至少 8-10 个文件需要修改

#### 4. 风险评估

| 风险类型 | 风险等级 | 说明 |
|---------|---------|------|
| **数据丢失** | 🟡 中 | 重命名表时如果出错，可能导致数据不可访问 |
| **外键约束失败** | 🟡 中 | 外键约束可能因为表名变化而失效 |
| **代码遗漏** | 🔴 高 | 可能有遗漏的引用未修改 |
| **回滚困难** | 🟡 中 | 改名后如果有问题，回滚需要再次重命名 |
| **线上影响** | 🔴 高 | 如果在生产环境执行，可能影响正在运行的查询 |

## 📊 综合评估

### 改名 vs 不改名

| 维度 | 改名为 role_action | 保持 action |
|------|-------------------|-------------|
| **语义清晰度** | 🔴 降低（混淆"角色操作"） | 🟢 清晰（权限点） |
| **通用性** | 🔴 降低（局限于 role） | 🟢 高（通用权限点） |
| **扩展性** | 🔴 降低（难以扩展到其他权限模型） | 🟢 高（支持多种权限模型） |
| **符合规范** | 🔴 不符合业界规范 | 🟢 符合业界规范 |
| **代码修改量** | 🔴 大（8-10 个文件） | 🟢 无需修改 |
| **迁移风险** | 🔴 中高风险 | 🟢 无风险 |
| **收益** | 🔴 无明显收益 | 🟢 稳定可靠 |

### 推荐指数

| 方案 | 推荐指数 | 理由 |
|------|---------|------|
| **保持现有命名** | ⭐⭐⭐⭐⭐ | 清晰、通用、符合规范、无风险 |
| **改名为 role_action** | ⭐ | 收益低、风险高、不符合规范 |

## 🎯 最终建议

### 强烈建议：保持现有命名

**理由**：

1. ✅ **当前命名已经很好**
   - `action` 清晰表达了"权限点"的概念
   - `role_permission` 清晰表达了"角色权限关联"的概念
   - 符合业界命名规范

2. ✅ **改名没有明显收益**
   - 功能没有任何变化
   - 性能没有任何提升
   - 可维护性没有提升

3. ✅ **改名有明显风险**
   - 代码修改量大（8-10 个文件）
   - 数据库迁移风险
   - 可能遗漏某些引用

4. ✅ **前缀一致性不是必须的**
   - 基础数据表不需要前缀
   - 只有业务表或有命名冲突时才需要前缀
   - `action` 是基础数据表，不需要 `role_` 前缀

### 如果一定要改（不推荐）

如果你坚持要改名，建议采用以下流程：

1. **备份数据库**（必须！）
   ```bash
   mysqldump -u root -p database_name > backup.sql
   ```

2. **在测试环境充分测试**
   - 创建迁移脚本
   - 修改所有相关代码
   - 运行全部测试用例
   - 手动测试所有权限相关功能

3. **分阶段部署**
   - 先部署到测试环境
   - 测试 1-2 天，确认无问题
   - 再部署到生产环境

4. **准备回滚方案**
   - 准备反向迁移脚本（重命名回去）
   - 准备代码回滚方案

## 📚 参考案例

### 业界常见命名

**Spring Security（Java）**
```
authority         - 权限点（类似 action）
role              - 角色
user_authorities  - 用户权限关联
role_authorities  - 角色权限关联
```

**Laravel Permission（PHP）**
```
permissions       - 权限点（类似 action）
roles             - 角色
role_has_permissions - 角色权限关联
```

**Casbin（Go）**
```
policy            - 权限策略（类似 action）
role              - 角色
grouping_policy   - 角色关联
```

**结论**：业界都使用简洁的命名，基础数据表不加前缀！

## 💡 总结

1. **强烈建议保持现有命名**
   - `action` ✅
   - `role_permission` ✅

2. **不建议改名的原因**
   - 语义会变得混淆
   - 通用性降低
   - 改名成本高，收益低
   - 不符合业界规范

3. **命名规范建议**
   - 基础数据表：单数名词，不加前缀（`action`, `role`, `user`）
   - 关联表：`table1_table2`（`role_permission`, `user_role`）
   - 业务表：有意义的前缀（`role_assignment`, `permission_request`）

4. **如果一定要改**
   - 需要修改 8-10 个文件
   - 需要写数据库迁移脚本
   - 需要充分测试
   - 需要准备回滚方案
   - **但仍然不推荐** ❌

---

**最终建议**：保持现有命名，专注于功能开发和优化，而不是纠结于命名的前缀一致性。当前命名已经很好了！✅
