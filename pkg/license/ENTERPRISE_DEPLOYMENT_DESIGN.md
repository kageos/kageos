# License 企业部署设计方案

## 🎯 核心问题

**这是企业部署，不是单机软件！**

- ✅ 可能是**单机部署**（所有服务在一台机器）
- ✅ 可能是**集群部署**（多个服务分布在多台机器）
- ✅ License 验证不只是验证一个服务，可能会验证**多个服务**

---

## 📊 系统架构分析

### 服务列表

| 服务 | 职责 | 是否需要 License 验证 |
|------|------|---------------------|
| **app-server** | 应用管理、用户管理 | ✅ 需要（核心服务） |
| **agent-server** | 代码生成、智能体 | ✅ 需要（核心服务） |
| **app-runtime** | 容器化运行时 | ✅ 需要（核心服务） |
| **api-gateway** | API 网关 | ⚠️ 可选（转发请求） |
| **app-storage** | 存储服务 | ⚠️ 可选（存储功能） |

**关键点**：
- 多个服务需要验证同一个 License
- 服务可能部署在不同的机器上
- 服务可能动态扩缩容

---

## 🏗️ 部署场景分析

### 场景一：单机部署

```
┌─────────────────────────────────────┐
│         单机部署                     │
│                                     │
│  ┌──────────┐  ┌──────────┐      │
│  │app-server│  │agent-server│     │
│  └──────────┘  └──────────┘      │
│                                     │
│  ┌──────────┐  ┌──────────┐      │
│  │app-runtime│  │api-gateway│    │
│  └──────────┘  └──────────┘      │
│                                     │
│  License 文件：./license.json      │
│  公钥文件：./license_public_key.pem │
└─────────────────────────────────────┘
```

**特点**：
- ✅ 所有服务在同一台机器
- ✅ License 文件可以放在共享位置
- ✅ 硬件绑定可以基于单机硬件信息

---

### 场景二：集群部署（多节点）

```
┌─────────────────────────────────────────────────────┐
│                  集群部署                             │
│                                                     │
│  ┌─────────────┐      ┌─────────────┐             │
│  │  节点 1      │      │  节点 2      │             │
│  │             │      │             │             │
│  │ app-server  │      │ app-server  │             │
│  │ agent-server│      │ agent-server│             │
│  └─────────────┘      └─────────────┘             │
│                                                     │
│  ┌─────────────┐      ┌─────────────┐             │
│  │  节点 3      │      │  节点 4      │             │
│  │             │      │             │             │
│  │ app-runtime │      │ app-runtime │             │
│  └─────────────┘      └─────────────┘             │
│                                                     │
│  共享存储：/shared/license.json                    │
│  公钥文件：/shared/license_public_key.pem         │
└─────────────────────────────────────────────────────┘
```

**特点**：
- ✅ 多个服务分布在多台机器
- ✅ License 文件需要放在共享存储（NFS、对象存储等）
- ⚠️ 硬件绑定需要重新设计（不能绑定单机硬件）

---

## 🎯 License 验证粒度设计

### 方案一：部署级验证（推荐）⭐⭐⭐⭐⭐

**设计**：
- ✅ **一个 License 对应一个部署**（不是单个服务）
- ✅ **所有服务共享同一个 License**
- ✅ **License 文件放在共享存储**

**License 文件位置**：
- 单机部署：`./license.json` 或 `~/.ai-agent-os/license.json`
- 集群部署：`/shared/license.json` 或 `NFS/license.json` 或对象存储

**验证流程**：
```
服务启动
  ↓
读取共享 License 文件
  ↓
验证签名
  ↓
验证过期时间
  ↓
验证硬件绑定（如果启用）
  ↓
设置 License 到内存
  ↓
提供服务
```

**优势**：
- ✅ 简单清晰：一个部署一个 License
- ✅ 易于管理：只需要管理一个 License 文件
- ✅ 适合集群：所有服务共享同一个 License

**硬件绑定策略**：
- ⚠️ **集群部署时，硬件绑定需要重新设计**
- 方案A：不启用硬件绑定（推荐，集群环境）
- 方案B：绑定集群标识（如集群ID、部署ID）

---

### 方案二：服务级验证（不推荐）

**设计**：
- ❌ 每个服务有独立的 License
- ❌ 需要为每个服务生成 License

**问题**：
- ❌ 管理复杂：需要管理多个 License
- ❌ 成本高：需要为每个服务购买 License
- ❌ 不适合集群：服务可能动态扩缩容

---

## 🔐 安全防护策略（企业部署场景）

### 1. RSA 签名验证（必须）⭐⭐⭐⭐⭐

**防护能力**：⭐⭐⭐⭐（高）

**实现**：
- ✅ License 文件包含 RSA 签名
- ✅ 使用公钥验证签名
- ✅ 签名验证失败会拒绝 License

**集群部署考虑**：
- ✅ 公钥文件也需要放在共享存储
- ✅ 或者将公钥嵌入到二进制文件中（更安全）

---

### 2. 硬件绑定（集群场景需要重新设计）⭐⭐⭐

**单机部署**：
- ✅ 可以绑定单机硬件（MAC、CPU、主板等）
- ✅ 防止 License 在机器间共享

**集群部署**：
- ⚠️ **不能绑定单机硬件**（服务可能在不同机器）
- ✅ **方案A：不启用硬件绑定**（推荐）
- ✅ **方案B：绑定集群标识**（如集群ID、部署ID）

**实现方案**：

```go
// 集群部署时，硬件绑定策略
func getHardwareID() string {
    // 方案A：不启用硬件绑定（集群部署）
    // 返回空字符串，跳过硬件绑定验证
    
    // 方案B：绑定集群标识
    clusterID := os.Getenv("CLUSTER_ID")
    if clusterID != "" {
        return clusterID
    }
    
    // 方案C：绑定部署标识
    deploymentID := os.Getenv("DEPLOYMENT_ID")
    if deploymentID != "" {
        return deploymentID
    }
    
    // 单机部署：使用硬件信息
    // ...
}
```

---

### 3. 在线验证（推荐）⭐⭐⭐⭐

**防护能力**：⭐⭐⭐⭐⭐（极高）

**实现方案**：
- ✅ **定期在线验证**（每天或每周）
- ✅ **服务器端验证**（License 服务器）
- ✅ **License 状态同步**（撤销、续期等）

**集群部署考虑**：
- ✅ 所有服务都连接到同一个 License 服务器
- ✅ License 服务器记录部署信息（License ID、部署ID、服务列表）
- ✅ 可以实时撤销 License

**架构设计**：

```
┌─────────────────────────────────────────────────────┐
│              企业部署（多个服务）                      │
│                                                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│  │app-server│  │agent-server│ │app-runtime│     │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘      │
│       │             │             │             │
│       └─────────────┴─────────────┘             │
│                    │                            │
│                    │ 定期验证（每24小时）        │
│                    ↓                            │
│  ┌─────────────────────────────────────┐        │
│  │      License 验证服务器              │        │
│  │                                     │        │
│  │  - 验证 License 有效性              │        │
│  │  - 检查 License 是否被撤销          │        │
│  │  - 记录部署信息                     │        │
│  └─────────────────────────────────────┘        │
└─────────────────────────────────────────────────────┘
```

---

### 4. 时间验证（推荐）⭐⭐⭐⭐

**防护能力**：⭐⭐⭐⭐（高）

**实现方案**：
- ✅ **在线时间验证**（从服务器获取时间）
- ✅ **时间戳验证**（防止时间回退）
- ✅ **时间窗口检查**（防止时间跳跃）

**集群部署考虑**：
- ✅ 所有服务从同一个时间服务器获取时间
- ✅ 或者使用 NTP 同步时间

---

## 📋 License 文件设计（企业部署）

### License 数据结构

```json
{
  "license": {
    "id": "license-xxx",
    "edition": "enterprise",
    "issued_at": "2025-01-01T00:00:00Z",
    "expires_at": "2026-01-01T00:00:00Z",
    "customer": "Company Name",
    
    // ⭐ 部署信息（新增）
    "deployment_id": "deployment-xxx",  // 部署ID（用于集群部署）
    "deployment_type": "single|cluster", // 部署类型
    
    // 资源限制
    "max_apps": 100,
    "max_users": 50,
    
    // 功能开关
    "features": {
      "operate_log": true,
      "workflow": true,
      // ...
    },
    
    // ⭐ 硬件绑定（可选，集群部署时不使用）
    "hardware_id": "optional-hardware-binding",
    
    // ⭐ 在线验证（推荐）
    "verify_url": "https://license.example.com/verify",
    "verify_interval": 86400  // 验证间隔（秒），默认24小时
  },
  "signature": "RSA签名..."
}
```

---

## 🎯 推荐方案

### 方案：部署级验证 + 在线验证

**设计**：
1. ✅ **部署级验证**：一个 License 对应一个部署
2. ✅ **共享 License 文件**：所有服务读取同一个 License 文件
3. ✅ **在线验证**：定期在线验证 License 有效性
4. ✅ **硬件绑定可选**：单机部署启用，集群部署不启用

**优势**：
- ✅ 简单清晰：一个部署一个 License
- ✅ 适合集群：所有服务共享同一个 License
- ✅ 易于管理：只需要管理一个 License 文件
- ✅ 安全可靠：在线验证可以实时撤销 License

---

## 📂 License 文件路径设计

### 单机部署

**License 文件路径**：
- 优先级1：环境变量 `LICENSE_PATH`
- 优先级2：`./license.json`
- 优先级3：`~/.ai-agent-os/license.json`

**公钥文件路径**：
- 优先级1：环境变量 `LICENSE_PUBLIC_KEY_PATH`
- 优先级2：`./license_public_key.pem`
- 优先级3：`~/.ai-agent-os/license_public_key.pem`

---

### 集群部署

**License 文件路径**：
- 优先级1：环境变量 `LICENSE_PATH`（指向共享存储）
- 优先级2：`/shared/license.json`（NFS 挂载）
- 优先级3：对象存储（S3、MinIO 等）

**公钥文件路径**：
- 优先级1：环境变量 `LICENSE_PUBLIC_KEY_PATH`（指向共享存储）
- 优先级2：`/shared/license_public_key.pem`（NFS 挂载）
- 优先级3：对象存储（S3、MinIO 等）
- 优先级4：嵌入到二进制文件中（最安全）

---

## 🔐 安全防护策略（简化版）

### 基础防护（必须实现）⭐⭐⭐⭐⭐

1. ✅ **RSA 签名验证**
   - License 文件包含 RSA 签名
   - 使用公钥验证签名
   - 签名验证失败会拒绝 License

2. ✅ **过期检查**
   - 检查 `expires_at` 字段
   - 如果已过期，License 无效

3. ✅ **部署标识验证**（集群部署）
   - 验证 `deployment_id` 是否匹配
   - 防止 License 在不同部署间共享

---

### 增强防护（强烈推荐）⭐⭐⭐⭐

4. ✅ **在线验证**（定期验证）
   - 每24小时在线验证一次
   - 可以实时撤销 License
   - 可以监控 License 使用情况

5. ✅ **时间验证**（防止时间回退）
   - 从服务器获取时间
   - 检查时间是否回退

---

### 可选防护（高级）⭐⭐⭐

6. ⚠️ **硬件绑定**（仅单机部署）
   - 单机部署时启用
   - 集群部署时不启用

7. ⚠️ **代码混淆**（可选）
   - 使用 `garble` 工具
   - 混淆关键代码

---

## 💻 实现方案

### 1. License 数据结构（增强）

```go
// pkg/license/license.go

type License struct {
    // ... 原有字段 ...
    
    // ⭐ 部署信息（新增）
    DeploymentID   string `json:"deployment_id,omitempty"`   // 部署ID
    DeploymentType string `json:"deployment_type,omitempty"` // 部署类型：single, cluster
    
    // ⭐ 在线验证（新增）
    VerifyURL      string `json:"verify_url,omitempty"`      // 在线验证URL
    VerifyInterval int    `json:"verify_interval,omitempty"` // 验证间隔（秒）
}
```

---

### 2. License 管理器（简化版）

```go
// pkg/license/manager.go

// LoadLicense 加载 License（企业部署场景）
func (m *Manager) LoadLicense(path string) error {
    // 1. 读取 License 文件（从共享存储或本地）
    // 2. 验证签名
    // 3. 验证过期时间
    // 4. 验证部署标识（如果启用）
    // 5. 验证硬件绑定（仅单机部署）
    // 6. 启动在线验证（如果启用）
}
```

---

### 3. 部署标识验证

```go
// pkg/license/manager.go

// verifyDeploymentID 验证部署标识
func (m *Manager) verifyDeploymentID(license *License) error {
    // 如果 License 有 deployment_id，验证是否匹配
    if license.DeploymentID != "" {
        currentDeploymentID := getDeploymentID()
        if currentDeploymentID != license.DeploymentID {
            return fmt.Errorf("license deployment ID mismatch")
        }
    }
    return nil
}

// getDeploymentID 获取部署标识
func getDeploymentID() string {
    // 优先级1：环境变量
    if id := os.Getenv("DEPLOYMENT_ID"); id != "" {
        return id
    }
    
    // 优先级2：配置文件
    // ...
    
    // 优先级3：Kubernetes 命名空间（如果是 K8s 部署）
    // ...
    
    return ""
}
```

---

## 🎯 实施建议

### 阶段一：基础实现（立即实施）

1. ✅ **RSA 签名验证**（已实现）
2. ✅ **过期检查**（已实现）
3. ✅ **部署标识验证**（需要实现）
4. ✅ **共享 License 文件**（支持环境变量指定路径）

---

### 阶段二：增强防护（3-6个月）

5. ✅ **在线验证**（定期验证）
6. ✅ **时间验证**（防止时间回退）

---

### 阶段三：可选防护（可选）

7. ⚠️ **硬件绑定**（仅单机部署）
8. ⚠️ **代码混淆**（可选）

---

## 📋 实施 Checklist

### 立即实施

- [ ] 修改 License 数据结构（添加部署信息）
- [ ] 实现部署标识验证
- [ ] 支持共享 License 文件（环境变量指定路径）
- [ ] 修改硬件绑定逻辑（集群部署时不启用）

### 3-6个月实施

- [ ] 搭建 License 验证服务器
- [ ] 实现在线验证机制
- [ ] 实现定期验证
- [ ] 实现 License 撤销功能

---

## 🎯 总结

### 核心设计原则

1. **部署级验证**：一个 License 对应一个部署（不是单个服务）
2. **共享 License 文件**：所有服务读取同一个 License 文件
3. **在线验证**：定期在线验证 License 有效性
4. **硬件绑定可选**：单机部署启用，集群部署不启用

### 关键点

- ✅ **简单清晰**：不需要为每个服务生成 License
- ✅ **适合集群**：所有服务共享同一个 License
- ✅ **易于管理**：只需要管理一个 License 文件
- ✅ **安全可靠**：在线验证可以实时撤销 License

---

## 📞 参考

- [License 系统设计](./README.md)
- [安全防护方案](./SECURITY.md)
