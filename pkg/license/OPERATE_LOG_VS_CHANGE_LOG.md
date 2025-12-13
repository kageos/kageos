# OperateLog vs ChangeLog 区别详解

## 📊 核心区别

| 维度 | OperateLog（操作日志） | ChangeLog（变更日志） |
|------|----------------------|---------------------|
| **记录对象** | 用户的操作行为 | 系统/应用/数据的变更历史 |
| **记录内容** | 谁、什么时候、做了什么操作 | 什么、从什么变成什么 |
| **记录粒度** | 操作级别（新增、更新、删除） | 变更级别（字段级、记录级） |
| **使用场景** | 审计、合规、问题追溯 | 版本管理、历史查看、回滚 |
| **目标用户** | 管理员、审计人员 | 开发者、用户 |
| **数据量** | 大（每条操作都记录） | 中（只记录变更） |
| **存储方式** | 数据库表 | 版本文件/数据库 |

---

## 🔍 详细对比

### 1. OperateLog（操作日志）

#### 定义
**操作日志**记录用户在平台上的**所有操作行为**，包括：
- 谁（User）
- 什么时候（Timestamp）
- 做了什么操作（Action：create、update、delete）
- 操作了什么资源（Resource：table_row、function、app）
- 操作的详细信息（Details：IP地址、User Agent、变更内容）

#### 记录内容示例

```json
{
  "id": 12345,
  "user": "admin",
  "action": "update",
  "resource": "table_row",
  "resource_id": "ticket_001",
  "tree_id": 1,
  "changes": {
    "table": "crm_ticket",
    "row_id": "ticket_001",
    "fields": {
      "status": {
        "old_value": "pending",
        "new_value": "resolved"
      },
      "assignee": {
        "old_value": null,
        "new_value": "user123"
      }
    }
  },
  "ip_address": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "created_at": "2025-01-15T10:30:00Z"
}
```

#### 使用场景

1. **审计追踪**
   - 谁修改了敏感数据？
   - 什么时候修改的？
   - 修改了什么内容？

2. **合规检查**
   - 满足 GDPR、SOX 等合规要求
   - 提供完整的操作审计链

3. **问题追溯**
   - 数据异常时，追溯是谁的操作导致的
   - 安全事件调查

4. **权限控制**
   - 记录所有敏感操作
   - 异常操作告警

#### 特点

- ✅ **全面记录**：记录所有操作，不遗漏
- ✅ **详细信息**：包含 IP、User Agent、变更前后值
- ✅ **不可篡改**：操作日志一旦记录，不能修改
- ✅ **长期保存**：需要长期保存，用于审计

---

### 2. ChangeLog（变更日志）

#### 定义
**变更日志**记录**系统/应用/数据的变更历史**，包括：
- 什么（What：应用、API、数据）
- 从什么变成什么（From → To）
- 什么时候变更的（When）
- 变更类型（Type：新增、修改、删除）

#### 记录内容示例

```json
{
  "version": "v2",
  "timestamp": "2025-01-15T10:30:00Z",
  "app": "crm_ticket",
  "changes": {
    "apis": {
      "added": [
        {
          "code": "create_ticket",
          "path": "/api/v1/ticket/create",
          "method": "POST"
        }
      ],
      "modified": [
        {
          "code": "update_ticket",
          "path": "/api/v1/ticket/update",
          "changes": {
            "request_params": {
              "added": ["priority"],
              "removed": []
            }
          }
        }
      ],
      "deleted": []
    },
    "tables": {
      "added": ["ticket_comment"],
      "modified": ["ticket"],
      "deleted": []
    }
  }
}
```

#### 使用场景

1. **版本管理**
   - 应用版本间的变更对比
   - API 变更历史
   - 数据库 Schema 变更

2. **历史查看**
   - 查看应用的历史版本
   - 了解功能演进过程

3. **回滚支持**
   - 了解变更内容，决定是否回滚
   - 回滚到指定版本

4. **文档生成**
   - 自动生成变更说明
   - 发布 Release Notes

#### 特点

- ✅ **版本化**：按版本记录变更
- ✅ **结构化**：变更内容结构化存储
- ✅ **可对比**：支持版本间对比
- ✅ **可回滚**：支持回滚到历史版本

---

## 🎯 实际应用场景对比

### 场景一：工单系统

#### OperateLog 记录
```
用户 admin 在 2025-01-15 10:30:00 更新了工单 ticket_001
- 状态：pending → resolved
- 负责人：null → user123
- IP：192.168.1.100
```

#### ChangeLog 记录
```
应用 crm_ticket v2.0（2025-01-15）
- 新增 API：create_ticket
- 修改 API：update_ticket（新增 priority 参数）
- 新增表：ticket_comment
```

---

### 场景二：数据异常调查

#### 使用 OperateLog
```
问题：工单 ticket_001 的状态被错误修改
调查：
1. 查询 OperateLog，找到所有修改 ticket_001 的操作
2. 发现 user123 在 10:30:00 修改了状态
3. 查看操作详情：IP、User Agent、变更内容
4. 确认是误操作还是恶意操作
```

#### 使用 ChangeLog
```
问题：应用升级后功能异常
调查：
1. 查看 ChangeLog，了解 v2.0 的变更
2. 发现 update_ticket API 新增了 priority 参数
3. 检查前端是否适配了新参数
4. 确认是兼容性问题
```

---

## 💡 为什么需要两个功能？

### 1. 不同的目标

- **OperateLog**：回答"谁做了什么"
- **ChangeLog**：回答"什么变了"

### 2. 不同的用户

- **OperateLog**：管理员、审计人员
- **ChangeLog**：开发者、用户

### 3. 不同的需求

- **OperateLog**：合规、审计、安全
- **ChangeLog**：版本管理、历史查看、回滚

---

## 📋 功能分发建议

### OperateLog（操作日志）

**建议：保留企业版** ⭐⭐⭐

**理由**：
- ✅ **合规需求**：企业需要操作审计，个人不需要
- ✅ **高商业价值**：企业级功能，直接促进升级
- ✅ **资源消耗**：操作日志会产生大量数据
- ✅ **差异化明显**：企业版的核心价值之一

**参考案例**：
- GitLab：操作日志在企业版
- GitHub：操作日志在企业版

---

### ChangeLog（变更日志）

**建议：移到社区版** ⭐⭐

**理由**：
- ✅ **基础功能**：查看变更历史是基础需求
- ✅ **提升透明度**：变更日志让用户了解系统变化
- ✅ **降低使用门槛**：个人开发者也需要查看变更历史
- ✅ **促进协作**：团队协作时，变更日志很重要

**参考案例**：
- GitHub：变更日志在免费版可用
- GitLab：变更日志在免费版可用

**企业版增强**：
- 社区版：基础变更日志（版本、变更列表）
- 企业版：详细变更日志（字段级变更、变更对比、变更统计）

---

## 🔄 数据流对比

### OperateLog 数据流

```
用户操作
  ↓
业务逻辑（CreateApp、UpdateApp、DeleteApp）
  ↓
Callback（OnTableAddRow、OnTableUpdateRow）
  ↓
OperateLogger.CreateOperateLogger()
  ↓
数据库表：operate_logs
  ↓
审计查询、合规检查
```

### ChangeLog 数据流

```
应用更新
  ↓
onAppUpdate 回调
  ↓
获取当前 API 定义
  ↓
对比上一版本 API
  ↓
生成变更记录
  ↓
保存到文件：api-logs/v2.json
  ↓
前端展示、版本对比
```

---

## 📊 存储对比

### OperateLog 存储

```sql
CREATE TABLE operate_logs (
  id BIGINT PRIMARY KEY,
  user VARCHAR(255),
  action VARCHAR(50),
  resource VARCHAR(100),
  resource_id VARCHAR(255),
  tree_id BIGINT,
  changes JSON,
  ip_address VARCHAR(50),
  user_agent VARCHAR(500),
  created_at TIMESTAMP
);

-- 索引
CREATE INDEX idx_user ON operate_logs(user);
CREATE INDEX idx_action ON operate_logs(action);
CREATE INDEX idx_resource ON operate_logs(resource);
CREATE INDEX idx_created_at ON operate_logs(created_at);
```

**特点**：
- 数据库表存储
- 需要索引优化查询
- 数据量大，需要定期归档

---

### ChangeLog 存储

```json
// api-logs/v1.json
{
  "version": "v1",
  "timestamp": "2025-01-10T00:00:00Z",
  "apis": [...]
}

// api-logs/v2.json
{
  "version": "v2",
  "timestamp": "2025-01-15T00:00:00Z",
  "apis": [...],
  "changes": {
    "added": [...],
    "modified": [...],
    "deleted": [...]
  }
}
```

**特点**：
- 文件存储（JSON）
- 按版本存储
- 数据量小，不需要索引

---

## 🎯 总结

### 核心区别

| 功能 | 记录什么 | 为什么需要 | 谁用 |
|------|---------|-----------|------|
| **OperateLog** | 用户操作 | 审计、合规、安全 | 管理员 |
| **ChangeLog** | 系统变更 | 版本管理、历史查看 | 开发者 |

### 分发策略

- **OperateLog**：保留企业版（合规需求，高价值）
- **ChangeLog**：移到社区版（基础功能，提升体验）

### 企业版增强

- **OperateLog**：完整版（详细审计、合规报告）
- **ChangeLog**：增强版（字段级变更、变更对比、变更统计）

---

## 📞 参考

- [OperateLog 接口定义](../enterprise/operate_log.go)
- [API Diff 功能](../sdk/agent-app/API_DIFF_README.md)
- [功能分发策略](./FEATURE_STRATEGY.md)
