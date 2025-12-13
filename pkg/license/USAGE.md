# License 系统使用指南

## 🚀 快速开始

### 社区版（默认）

**不需要任何配置**，直接启动即可：

```bash
./app-server
```

系统会自动检测到没有 License 文件，使用社区版功能。

---

### 企业版

#### 1. 获取 License 文件

从 License 签发方获取 `license.json` 文件。

#### 2. 放置 License 文件

将 `license.json` 放到以下任一位置（按优先级）：

1. **环境变量指定路径**：
   ```bash
   export LICENSE_PATH=/path/to/license.json
   ./app-server
   ```

2. **当前目录**：
   ```bash
   cp license.json ./
   ./app-server
   ```

3. **用户目录**：
   ```bash
   mkdir -p ~/.ai-agent-os
   cp license.json ~/.ai-agent-os/
   ./app-server
   ```

#### 3. 放置公钥文件

将 `license_public_key.pem` 放到以下任一位置（按优先级）：

1. **环境变量指定路径**：
   ```bash
   export LICENSE_PUBLIC_KEY_PATH=/path/to/license_public_key.pem
   ```

2. **当前目录**：
   ```bash
   cp license_public_key.pem ./
   ```

3. **用户目录**：
   ```bash
   cp license_public_key.pem ~/.ai-agent-os/
   ```

#### 4. 启动服务器

```bash
./app-server
```

服务器启动时会自动加载和验证 License。

---

## 📋 License 文件格式

### 示例

```json
{
  "license": {
    "id": "license-xxx",
    "edition": "enterprise",
    "issued_at": "2025-01-01T00:00:00Z",
    "expires_at": "2026-01-01T00:00:00Z",
    "customer": "Your Company Name",
    "max_apps": 100,
    "max_users": 50,
    "features": {
      "operate_log": true,
      "workflow": true,
      "approval": true,
      "comment": true,
      "rbac": true,
      "scheduled_task": false,
      "recycle_bin": true,
      "change_log": true,
      "notification": true,
      "config_management": false,
      "quick_link": true
    }
  },
  "signature": "RSA签名（Base64编码）"
}
```

### 字段说明

- **id**: License 唯一标识
- **edition**: 版本类型（`community`, `professional`, `enterprise`, `flagship`）
- **issued_at**: 签发时间（ISO 8601 格式）
- **expires_at**: 过期时间（ISO 8601 格式，空字符串表示永久）
- **customer**: 客户名称
- **max_apps**: 最大应用数量（0 表示无限制）
- **max_users**: 最大用户数量（0 表示无限制）
- **features**: 功能开关对象
- **signature**: RSA 签名（Base64 编码）

---

## 🔍 验证 License 状态

### 查看日志

服务器启动时会在日志中显示 License 状态：

```
[Server] Initializing license...
[Server] License loaded: Edition=enterprise, Customer=Your Company, ExpiresAt=2026-01-01T00:00:00Z
[Enterprise] Enterprise edition detected: Edition=enterprise, Customer=Your Company
[Enterprise] Initializing operate log feature...
[Enterprise] Operate log feature initialized
```

### 社区版日志

```
[Server] Initializing license...
[Server] Community edition (no license file)
[Enterprise] Community edition detected, using default implementations
```

---

## ⚠️ 常见问题

### 1. License 文件不存在

**现象**：系统使用社区版

**解决**：这是正常的，社区版不需要 License 文件。

---

### 2. License 验证失败

**现象**：日志显示 `License signature verification failed`

**可能原因**：
- License 文件被篡改
- 公钥文件不匹配
- 签名格式错误

**解决**：
1. 检查 License 文件是否完整
2. 检查公钥文件是否正确
3. 联系 License 签发方重新生成 License

---

### 3. License 已过期

**现象**：日志显示 `License has expired`

**解决**：联系 License 签发方续期或更新 License。

---

### 4. 硬件绑定不匹配

**现象**：日志显示 `License hardware binding mismatch`

**原因**：License 绑定了特定硬件，当前机器不匹配

**解决**：
1. 在绑定的机器上使用
2. 联系 License 签发方重新生成 License（取消硬件绑定）

---

### 5. 功能不可用

**现象**：某个企业功能无法使用

**检查**：
1. 查看 License 中的 `features` 字段，确认该功能是否开启
2. 查看日志，确认功能是否已初始化

**解决**：
- 如果功能未开启，联系 License 签发方升级 License
- 如果功能已开启但无法使用，检查服务器日志

---

## 🔧 开发环境

### 测试 License

在开发环境中，可以使用测试 License：

```go
// 测试代码
licenseMgr := license.GetManager()
licenseMgr.SetLicensePath("./test-license.json")
licenseMgr.LoadLicense("")
```

### 禁用 License 验证（仅开发）

```go
// 不加载 License，自动使用社区版
// 什么都不做即可
```

---

## 📞 技术支持

如有问题，请联系技术支持团队。
