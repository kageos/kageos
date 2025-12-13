# License 设计方案总结（企业部署场景）

## 🎯 核心设计原则

### 1. 部署级验证（不是服务级）

**设计**：
- ✅ **一个 License 对应一个部署**（不是单个服务）
- ✅ **所有服务共享同一个 License**
- ✅ **License 文件放在共享存储**（集群部署）

**优势**：
- ✅ 简单清晰：不需要为每个服务生成 License
- ✅ 适合集群：所有服务共享同一个 License
- ✅ 易于管理：只需要管理一个 License 文件

---

### 2. 共享 License 文件

**单机部署**：
- License 文件：`./license.json` 或 `~/.ai-agent-os/license.json`
- 所有服务读取同一个文件

**集群部署**：
- License 文件：`/shared/license.json`（NFS 挂载）或对象存储
- 所有服务从共享存储读取同一个文件

---

### 3. 硬件绑定策略

**单机部署**：
- ✅ 启用硬件绑定（绑定单机硬件）

**集群部署**：
- ❌ 不启用硬件绑定（服务可能在不同机器）
- ✅ 使用部署标识验证（`deployment_id`）

---

## 📋 License 数据结构（企业部署）

```json
{
  "license": {
    "id": "license-xxx",
    "edition": "enterprise",
    "issued_at": "2025-01-01T00:00:00Z",
    "expires_at": "2026-01-01T00:00:00Z",
    "customer": "Company Name",
    
    // ⭐ 部署信息（新增）
    "deployment_id": "deployment-xxx",      // 部署ID（用于集群部署）
    "deployment_type": "single|cluster",     // 部署类型
    
    // 资源限制
    "max_apps": 100,
    "max_users": 50,
    
    // 功能开关
    "features": {
      "operate_log": true,
      "workflow": true,
      // ...
    },
    
    // ⭐ 硬件绑定（可选，仅单机部署）
    "hardware_id": "optional-hardware-binding",
    
    // ⭐ 在线验证（推荐）
    "verify_url": "https://license.example.com/verify",
    "verify_interval": 86400  // 验证间隔（秒），默认24小时
  },
  "signature": "RSA签名..."
}
```

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

### 可选防护（按需启用）⭐⭐⭐

6. ⚠️ **硬件绑定**（仅单机部署）
   - 单机部署时启用
   - 集群部署时不启用

---

## 🎯 验证流程

### 服务启动时

```
服务启动
  ↓
读取共享 License 文件（从环境变量或默认路径）
  ↓
验证签名（RSA 签名验证）
  ↓
验证过期时间
  ↓
验证部署标识（如果 License 有 deployment_id）
  ↓
验证硬件绑定（仅单机部署，如果 License 有 hardware_id）
  ↓
设置 License 到内存
  ↓
启动在线验证（如果 License 有 verify_url）
  ↓
提供服务
```

---

### 定期验证（如果启用在线验证）

```
每24小时（或配置的间隔）
  ↓
连接 License 验证服务器
  ↓
发送 License ID 和部署信息
  ↓
服务器验证 License 状态
  ├─ License 是否有效
  ├─ License 是否被撤销
  └─ 部署信息是否匹配
  ↓
返回验证结果
  ├─ 有效：继续使用
  └─ 无效：禁用企业功能（降级到社区版）
```

---

## 📂 License 文件路径（企业部署）

### 单机部署

**License 文件**：
- 环境变量：`LICENSE_PATH`
- 当前目录：`./license.json`
- 用户目录：`~/.ai-agent-os/license.json`

**公钥文件**：
- 环境变量：`LICENSE_PUBLIC_KEY_PATH`
- 当前目录：`./license_public_key.pem`
- 用户目录：`~/.ai-agent-os/license_public_key.pem`

---

### 集群部署

**License 文件**：
- 环境变量：`LICENSE_PATH=/shared/license.json`（推荐）
- NFS 挂载：`/shared/license.json`
- 对象存储：S3、MinIO 等（需要实现下载逻辑）

**公钥文件**：
- 环境变量：`LICENSE_PUBLIC_KEY_PATH=/shared/license_public_key.pem`（推荐）
- NFS 挂载：`/shared/license_public_key.pem`
- 对象存储：S3、MinIO 等（需要实现下载逻辑）
- **或者嵌入到二进制文件中**（最安全，推荐）

---

## 🔐 防破解策略（企业部署场景）

### 1. RSA 签名验证（必须）⭐⭐⭐⭐⭐

**防护能力**：⭐⭐⭐⭐（高）

**实现**：
- ✅ License 文件包含 RSA 签名
- ✅ 使用公钥验证签名
- ✅ 签名验证失败会拒绝 License

**集群部署考虑**：
- ✅ 公钥文件放在共享存储
- ✅ 或者将公钥嵌入到二进制文件中（更安全）

---

### 2. 部署标识验证（推荐）⭐⭐⭐⭐

**防护能力**：⭐⭐⭐⭐（高）

**实现**：
- ✅ License 包含 `deployment_id`
- ✅ 服务启动时验证 `deployment_id` 是否匹配
- ✅ 防止 License 在不同部署间共享

**优势**：
- ✅ 适合集群部署
- ✅ 简单有效
- ✅ 易于管理

---

### 3. 在线验证（推荐）⭐⭐⭐⭐⭐

**防护能力**：⭐⭐⭐⭐⭐（极高）

**实现**：
- ✅ 定期在线验证（每24小时）
- ✅ 可以实时撤销 License
- ✅ 可以监控 License 使用情况

**集群部署考虑**：
- ✅ 所有服务都连接到同一个 License 服务器
- ✅ License 服务器记录部署信息
- ✅ 可以实时撤销 License

---

### 4. 硬件绑定（仅单机部署）⭐⭐⭐

**防护能力**：⭐⭐⭐（中）

**实现**：
- ✅ 单机部署时启用
- ❌ 集群部署时不启用

**集群部署考虑**：
- ❌ 不能绑定单机硬件（服务可能在不同机器）
- ✅ 使用部署标识验证替代

---

## 🎯 推荐实施方案

### 阶段一：基础防护（立即实施）⭐⭐⭐⭐⭐

1. ✅ **RSA 签名验证**（已实现）
2. ✅ **过期检查**（已实现）
3. ✅ **部署标识验证**（需要实现）
4. ✅ **共享 License 文件**（支持环境变量指定路径）

---

### 阶段二：增强防护（3-6个月）⭐⭐⭐⭐

5. ✅ **在线验证**（定期验证）
6. ✅ **时间验证**（防止时间回退）

---

### 阶段三：可选防护（按需启用）⭐⭐⭐

7. ⚠️ **硬件绑定**（仅单机部署）
8. ⚠️ **代码混淆**（可选）

---

## 📋 实施 Checklist

### 立即实施

- [ ] 修改 License 数据结构（添加 `deployment_id`、`deployment_type`）
- [ ] 实现部署标识验证（`verifyDeploymentID()`）
- [ ] 支持共享 License 文件（环境变量指定路径）
- [ ] 修改硬件绑定逻辑（集群部署时不启用）

### 3-6个月实施

- [ ] 搭建 License 验证服务器
- [ ] 实现在线验证机制
- [ ] 实现定期验证
- [ ] 实现 License 撤销功能

---

## 🎯 总结

### 核心设计

1. **部署级验证**：一个 License 对应一个部署
2. **共享 License 文件**：所有服务读取同一个 License 文件
3. **在线验证**：定期在线验证 License 有效性
4. **硬件绑定可选**：单机部署启用，集群部署不启用

### 关键优势

- ✅ **简单清晰**：不需要为每个服务生成 License
- ✅ **适合集群**：所有服务共享同一个 License
- ✅ **易于管理**：只需要管理一个 License 文件
- ✅ **安全可靠**：在线验证可以实时撤销 License

---

## 📞 参考

- [企业部署设计方案](./ENTERPRISE_DEPLOYMENT_DESIGN.md)
- [安全防护方案](./SECURITY.md)
- [License 系统设计](./README.md)
