# License 服务方案总结（基于 NATS 广播）

## 🎯 核心设计

**独立的 License 服务 + NATS 广播机制**

- ✅ **独立的 License 服务**：统一管理 License，读取文件、验证、广播状态
- ✅ **NATS 广播**：所有实例通过 NATS 订阅 `license.status` 主题
- ✅ **默认社区版**：如果无法连接到 License 服务或 NATS，视为社区版
- ✅ **实时同步**：License 状态变更实时通知所有实例

---

## 🏗️ 架构图

```
┌─────────────────────────────────────┐
│      License 服务（独立服务）         │
│                                     │
│  - 读取 License 文件                │
│  - 验证 License                     │
│  - 发布状态到 NATS                  │
│  - 定期发布心跳                     │
└──────────────┬──────────────────────┘
               │
               │ NATS 发布
               │ 主题：license.status
               ↓
┌─────────────────────────────────────┐
│         NATS 消息中间件               │
└──────────────┬──────────────────────┘
               │
               │ NATS 订阅
               ↓
┌─────────────────────────────────────┐
│      所有服务实例（订阅状态）         │
│                                     │
│  app-server  ──┐                    │
│  agent-server ─┼── 订阅 license.status│
│  app-runtime  ─┘                    │
│                                     │
│  - 接收 License 状态                │
│  - 更新本地 License 状态             │
│  - 如果无法连接，默认社区版          │
└─────────────────────────────────────┘
```

---

## 📋 关键设计点

### 1. License 服务职责

- ✅ 读取 License 文件（从配置路径）
- ✅ 验证 License（签名、过期、部署ID等）
- ✅ 管理 License 状态
- ✅ 通过 NATS 发布 License 状态
- ✅ 定期发布心跳（每30秒）

---

### 2. NATS 主题设计

**主题名称**：`license.status`

**消息类型**：
- **status** - License 状态消息（启动时、状态变更时）
- **heartbeat** - 心跳消息（每30秒）
- **update** - License 更新消息（文件更新时）

---

### 3. 各服务实例（License Client）

**职责**：
- ✅ 订阅 NATS 主题 `license.status`
- ✅ 接收 License 状态消息
- ✅ 更新本地 License 状态
- ✅ 容错处理（连接失败、心跳超时）

**容错机制**：
- 如果无法连接到 NATS → 默认社区版
- 如果收不到心跳（60秒超时）→ 降级到社区版

---

## 🔄 工作流程

### License 服务启动

```
License 服务启动
  ↓
连接 NATS
  ↓
读取 License 文件
  ↓
验证 License
  ↓
发布初始状态（status）
  ↓
启动定期任务
  ├─ 每30秒发布心跳（heartbeat）
  └─ 定期检查 License 是否过期
```

---

### 服务实例启动

```
服务实例启动
  ↓
连接 NATS
  ↓
创建 License Client
  ├─ 成功：订阅 license.status
  └─ 失败：默认社区版
  ↓
等待接收 License 状态
  ├─ 收到 status → 更新本地状态
  ├─ 收到 heartbeat → 更新心跳时间
  └─ 超时未收到心跳 → 降级到社区版
```

---

## ✅ 优势

1. **集中管理**：License 服务统一管理 License
2. **实时同步**：License 状态变更实时通知所有实例
3. **容错机制**：连接失败或超时自动降级到社区版
4. **适合集群**：所有实例通过 NATS 连接，天然支持集群
5. **简化实现**：各服务实例只需订阅 NATS 主题

---

## ⚠️ 注意事项

1. **License 服务高可用**
   - License 服务是单点，需要保证高可用
   - 建议使用进程管理器（systemd）或容器编排（K8s）

2. **NATS 连接失败**
   - 如果 NATS 连接失败，所有实例降级到社区版
   - 需要确保 NATS 服务的高可用

3. **心跳超时**
   - 如果收不到心跳，自动降级到社区版
   - 心跳超时时间建议60秒

---

## 📋 实施步骤

### 阶段一：License 服务

1. 创建 `core/license-server` 目录
2. 实现 License 服务（读取、验证、广播）
3. 实现 NATS 发布逻辑
4. 实现定期任务（心跳、过期检查）

### 阶段二：License Client

1. 实现 License Client（订阅 NATS 主题）
2. 实现消息处理逻辑
3. 实现心跳检查机制
4. 实现容错处理

### 阶段三：各服务集成

1. app-server 集成 License Client
2. agent-server 集成 License Client
3. app-runtime 集成 License Client
4. 其他服务集成 License Client

---

## 🎯 总结

### 核心设计

1. **独立的 License 服务**：统一管理 License
2. **NATS 广播机制**：所有实例通过 NATS 订阅 License 状态
3. **默认社区版**：如果无法连接 License 服务，视为社区版
4. **实时同步**：License 状态变更实时通知所有实例

### 关键优势

- ✅ **集中管理**：License 服务统一管理 License
- ✅ **实时同步**：License 状态变更实时通知所有实例
- ✅ **容错机制**：连接失败或超时自动降级到社区版
- ✅ **适合集群**：所有实例通过 NATS 连接，天然支持集群
- ✅ **简化实现**：各服务实例只需订阅 NATS 主题

---

## 📞 参考

- [License 服务详细设计方案](./LICENSE_SERVICE_DESIGN.md)
- [企业部署设计方案](./ENTERPRISE_DEPLOYMENT_DESIGN.md)
