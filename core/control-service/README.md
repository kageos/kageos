# Control Service（控制服务）

Control Service 是一个轻量级的系统控制服务，承担 License 管理和系统级控制职责。

## 🎯 服务定位

- ✅ **核心职责**：License 管理（密钥分发）
- ✅ **扩展职责**：系统控制指令（下机、重启、维护模式等）
- ✅ **轻量级**：服务崩溃不影响其他服务进程
- ✅ **非关键路径**：其他服务可以独立运行，不依赖此服务

---

## 📋 服务职责

### 1. License 管理（核心职责）

- ✅ 读取 License 文件
- ✅ 验证 License（签名、过期、部署ID等）
- ✅ 加密 License 密钥（AES-256-GCM）
- ✅ 通过 NATS 分发 License 密钥（主题：`control.license.key`）
- ✅ 定期发布密钥（每5分钟，确保新实例能获取）

---

### 2. 系统控制指令（扩展职责）

- ✅ **下机指令**：优雅关闭所有服务
- ✅ **重启指令**：重启所有服务
- ✅ **维护模式**：进入/退出维护模式
- ✅ **配置更新**：通知配置更新
- ✅ **功能开关**：启用/禁用某些功能

**实现方式**：
- ✅ 通过 NATS 发布控制指令（主题：`control.command`）
- ✅ 各服务订阅并执行相应操作

---

### 3. 系统通知/公告（扩展职责）

- ✅ **系统维护通知**：通知系统维护时间
- ✅ **重要消息**：发布重要系统消息
- ✅ **版本更新通知**：通知新版本发布

**实现方式**：
- ✅ 通过 NATS 发布通知（主题：`control.notification`）

---

## 🏗️ 架构设计

```
┌─────────────────────────────────────┐
│      Control Service（控制服务）      │
│                                     │
│  - License 管理                     │
│  - 控制指令处理                     │
│  - HTTP API 服务器                  │
└──────────────┬──────────────────────┘
               │
               │ NATS 发布
               │ 主题：
               │ - control.license.key
               │ - control.command
               │ - control.notification
               ↓
┌─────────────────────────────────────┐
│         NATS 消息中间件               │
└──────────────┬──────────────────────┘
               │
               │ NATS 订阅
               ↓
┌─────────────────────────────────────┐
│      所有服务实例（订阅并执行）        │
│                                     │
│  - 获取 License 密钥                │
│  - 接收控制指令并执行               │
│  - 接收系统通知                     │
└─────────────────────────────────────┘
```

---

## 🚀 启动服务

### 环境变量

```bash
# License 文件路径（可选，默认：./license.json）
export LICENSE_PATH=./license.json

# NATS 服务器地址（可选，默认：nats://127.0.0.1:4222）
export NATS_URL=nats://127.0.0.1:4222

# License 加密密钥（必须，32字节）
export LICENSE_ENCRYPTION_KEY=your-32-byte-encryption-key-here!!
```

### 运行服务

```bash
cd core/control-service
go run cmd/app/main.go
```

---

## 📡 HTTP API

### License 相关

- `GET /api/v1/license/status` - 获取 License 状态

### 控制指令

- `POST /api/v1/control/shutdown` - 下机指令
- `POST /api/v1/control/restart` - 重启指令
- `POST /api/v1/control/maintenance` - 维护模式

### 通知

- `POST /api/v1/control/notification` - 发布通知

---

## 📋 API 示例

### 下机指令

```bash
curl -X POST http://localhost:9095/api/v1/control/shutdown \
  -H "Content-Type: application/json" \
  -d '{
    "graceful": true,
    "timeout": 30
  }'
```

### 维护模式

```bash
curl -X POST http://localhost:9095/api/v1/control/maintenance \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "message": "系统维护中，预计30分钟后恢复"
  }'
```

### 发布通知

```bash
curl -X POST http://localhost:9095/api/v1/control/notification \
  -H "Content-Type: application/json" \
  -d '{
    "type": "maintenance",
    "title": "系统维护通知",
    "message": "系统将于今晚22:00进行维护",
    "level": "warning"
  }'
```

---

## 🔐 NATS 主题

### License 密钥分发

**主题**：`control.license.key`

**消息格式**：
```json
{
  "encrypted_license": "base64-encoded-encrypted-license",
  "algorithm": "aes-256-gcm",
  "timestamp": 1234567890
}
```

---

### 控制指令

**主题**：`control.command`

**消息格式**：
```json
{
  "type": "shutdown",
  "params": {
    "graceful": true,
    "timeout": 30
  },
  "target_services": [],
  "timestamp": 1234567890
}
```

---

### 系统通知

**主题**：`control.notification`

**消息格式**：
```json
{
  "type": "maintenance",
  "title": "系统维护通知",
  "message": "系统将于今晚22:00进行维护",
  "level": "warning",
  "timestamp": 1234567890
}
```

---

## 🎯 服务特点

### 1. 轻量级

- ✅ **简单职责**：只承担轻量级职责
- ✅ **无状态**：服务本身无状态，易于重启
- ✅ **独立运行**：不依赖其他服务

---

### 2. 非关键路径

- ✅ **服务崩溃不影响其他服务**：其他服务可以独立运行
- ✅ **容错机制**：各服务可以检测 Control Service 是否可用
- ✅ **降级策略**：如果 Control Service 不可用，各服务降级到默认行为

---

### 3. 易于扩展

- ✅ **模块化设计**：各职责模块化，易于扩展
- ✅ **插件化**：可以轻松添加新的控制指令
- ✅ **配置化**：通过配置文件控制功能开关

---

## 📞 参考

- [Control Service 详细设计方案](../../pkg/license/CONTROL_SERVICE_DESIGN.md)
- [License 密钥分发方案](../../pkg/license/LICENSE_DISTRIBUTION_DESIGN.md)
