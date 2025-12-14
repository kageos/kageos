# Control Service 集成实现总结

## ✅ 已完成的工作

### 1. 配置结构体更新

**位置：** `pkg/config/control_service.go`

添加了 `ControlServiceClientConfig` 结构体，用于各服务连接到 Control Service：

```go
type ControlServiceClientConfig struct {
    Enabled       bool   `mapstructure:"enabled"`        // 是否启用
    NatsURL       string `mapstructure:"nats_url"`       // Control Service 的 NATS 地址
    EncryptionKey string `mapstructure:"encryption_key"` // License 加密密钥（32字节）
    KeyPath       string `mapstructure:"key_path"`       // 本地密钥文件路径
}
```

**已更新的服务配置：**
- `AppServerConfig` - 添加了 `ControlService` 字段
- `AgentServerConfig` - 添加了 `ControlService` 字段

---

### 2. 配置文件更新

**已更新的配置文件：**

#### `configs/app-server.yaml`
```yaml
control_service:
  enabled: true
  nats_url: ""  # 如果为空，使用主 NATS 配置
  encryption_key: "ai-agent-os-license-key-32bytes!!"  # 必须与 Control Service 相同
  key_path: ""  # 可选，默认：~/.ai-agent-os/license.key
```

#### `configs/agent-server.yaml`
```yaml
control_service:
  enabled: true
  nats_url: ""  # 如果为空，使用主 NATS 配置
  encryption_key: "ai-agent-os-license-key-32bytes!!"  # 必须与 Control Service 相同
  key_path: ""  # 可选，默认：~/.ai-agent-os/license.key
```

---

### 3. 服务集成

#### app-server

**位置：** `core/app-server/server/server.go`

**实现：**
1. 添加了 `licenseClient` 字段
2. 在 `initNATS` 之后调用 `initLicenseClient`
3. 在 `Stop` 方法中关闭 License Client

**初始化流程：**
```go
// 1. 初始化 NATS
initNATS()

// 2. 初始化 License Client（通过 NATS 获取和刷新 License）
initLicenseClient()

// 3. 初始化其他服务
initServices()
```

**License Client 逻辑：**
- 检查是否启用 Control Service 客户端
- 验证加密密钥（必须是 32 字节）
- 确定使用的 NATS 连接（如果配置了独立的 NATS URL，创建新连接；否则使用现有连接）
- 创建并启动 License Client
- License Client 会自动：
  - 尝试从本地加载密钥
  - 如果本地没有，通过 NATS 请求获取
  - 订阅刷新主题，监听刷新指令

---

#### agent-server

**位置：** `core/agent-server/server/server.go`

**实现：**
1. 添加了 `licenseClient` 字段
2. 在 `initNATS` 之后调用 `initLicenseClient`
3. 在 `Stop` 方法中关闭 License Client

**初始化流程：**
```go
// 1. 初始化 NATS
initNATS()

// 2. 初始化 License Client（通过 NATS 获取和刷新 License）
initLicenseClient()

// 3. 初始化其他服务
initServices()
```

---

## 🔄 工作流程

### 服务启动流程

```
1. 初始化数据库
   ↓
2. 初始化 NATS 连接
   ↓
3. 初始化 License Client
   ├─ 检查配置是否启用
   ├─ 验证加密密钥
   ├─ 确定 NATS 连接（使用主连接或独立连接）
   ├─ 创建 License Client
   └─ 启动 License Client
      ├─ 尝试从本地加载密钥
      ├─ 如果本地没有，通过 NATS 请求获取
      └─ 订阅刷新主题
   ↓
4. 初始化业务服务
   ↓
5. 启动 HTTP 服务器
```

---

### License 激活流程

```
服务启动
  ↓
License Client 启动
  ↓
尝试从本地加载密钥（~/.ai-agent-os/license.key）
  ├─ 成功 → 解密并激活 License
  └─ 失败 → 通过 NATS 请求获取
      ↓
通过 NATS 发送请求（control.license.key.request）
  ↓
Control Service 响应加密的 License
  ↓
解密并激活 License
  ↓
保存到本地文件（~/.ai-agent-os/license.key）
  ↓
订阅刷新主题（control.license.key.refresh）
  ↓
等待刷新指令
```

---

### License 刷新流程

```
Control Service 检测到 License 更新
  ↓
通过 NATS 发布刷新指令（control.license.key.refresh）
  ↓
各服务收到刷新指令
  ↓
读取本地密钥（用于对比）
  ↓
通过 NATS 请求新的密钥
  ↓
对比新旧密钥
  ├─ 相同 → 跳过更新
  └─ 不同 → 解密并更新 License
      ↓
保存新密钥到本地
  ↓
License 已更新
```

---

## 📋 配置说明

### Control Service 客户端配置

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `enabled` | bool | ❌ | 是否启用 Control Service 客户端（默认：true） |
| `nats_url` | string | ❌ | Control Service 的 NATS 地址（如果为空，使用主 NATS 配置） |
| `encryption_key` | string | ✅ | License 加密密钥（32字节，必须与 Control Service 相同） |
| `key_path` | string | ❌ | 本地密钥文件路径（可选，默认：~/.ai-agent-os/license.key） |

### 配置示例

#### 使用主 NATS 连接（推荐）

```yaml
nats:
  url: "nats://127.0.0.1:4223"

control_service:
  enabled: true
  nats_url: ""  # 为空，使用主 NATS 连接
  encryption_key: "ai-agent-os-license-key-32bytes!!"
```

#### 使用独立的 NATS 连接

```yaml
nats:
  url: "nats://127.0.0.1:4223"  # 主 NATS（用于业务）

control_service:
  enabled: true
  nats_url: "nats://127.0.0.1:4224"  # Control Service 的 NATS（独立）
  encryption_key: "ai-agent-os-license-key-32bytes!!"
```

---

## 🔐 安全说明

### 加密密钥

- **必须与 Control Service 相同**：所有服务和 Control Service 必须使用相同的加密密钥
- **32 字节长度**：加密密钥必须是 32 字节（256 位）
- **保密性**：虽然密钥在配置文件中，但建议：
  - 生产环境使用环境变量或密钥管理服务
  - 不要提交密钥到代码仓库

### NATS 连接

- **主连接**：如果 `nats_url` 为空，使用主 NATS 连接（推荐，减少连接数）
- **独立连接**：如果配置了 `nats_url`，创建独立的 NATS 连接（适合隔离场景）

---

## 🧪 测试验证

### 编译测试

```bash
# 编译 app-server
go build ./core/app-server/...

# 编译 agent-server
go build ./core/agent-server/...
```

### 运行测试

1. **启动 Control Service**
   ```bash
   ./control-service
   ```

2. **启动 app-server**
   ```bash
   ./app-server
   ```
   应该看到：
   ```
   [Server] License client initialized successfully
   [License Client] License activated: Edition=enterprise, Customer=...
   ```

3. **启动 agent-server**
   ```bash
   ./agent-server
   ```
   应该看到类似的日志。

---

## 📝 注意事项

### 1. 加密密钥一致性

**重要**：所有服务和 Control Service 必须使用相同的加密密钥，否则无法解密 License。

### 2. NATS 连接

- 如果使用主 NATS 连接，确保主 NATS 连接在 License Client 初始化之前已建立
- 如果使用独立连接，License Client 会自动创建新连接

### 3. 向后兼容

- app-server 仍然支持从文件加载 License（`initLicense` 方法）
- License Client 是新增的功能，不会影响现有的文件加载方式
- 如果 License Client 初始化失败，服务会继续运行（社区版）

### 4. 错误处理

- License Client 初始化失败不会中断服务启动
- 如果无法获取 License，服务会使用社区版
- 所有错误都会记录到日志中

---

## 🎯 下一步

### 可选优化

1. **其他服务集成**：如果需要，可以为 `api-gateway`、`app-runtime`、`app-storage` 等服务添加 License Client 支持

2. **配置验证**：在服务启动时验证加密密钥长度，如果不符合要求，给出明确的错误提示

3. **监控和告警**：添加 License 状态监控，当 License 过期或无效时发送告警

4. **密钥管理**：集成密钥管理服务（如 Vault），从安全的地方获取加密密钥

---

## 📚 相关文档

- [License Client 使用说明](./CLIENT_USAGE.md)
- [Control Service 设计文档](./CONTROL_SERVICE_DESIGN.md)
- [License 激活流程](./ACTIVATION_FLOW.md)
