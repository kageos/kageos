# License 系统设计文档

## 📋 概述

License 系统用于区分社区版和企业版，控制企业功能的可用性。

### 设计原则

1. **社区版优先**：没有 License 文件时，自动使用社区版（JetBrains 模式）
2. **企业版验证**：有 License 文件时，验证签名和有效性
3. **功能开关**：License 中包含功能开关，精确控制每个功能的可用性
4. **安全可靠**：使用 RSA 签名防止篡改，支持硬件绑定

---

## 🏗️ 架构设计

### 目录结构

```
pkg/license/
├── license.go      # License 数据结构
├── manager.go      # License 管理器（加载、验证、功能检查）
└── README.md       # 本文档
```

### 核心组件

1. **License 结构**：定义 License 数据格式
2. **Manager**：全局单例，管理 License 的加载和验证
3. **功能检查**：提供 `HasFeature()` 方法检查功能可用性

---

## 📝 License 文件格式

### JSON 结构

```json
{
  "license": {
    "id": "license-xxx",
    "edition": "enterprise",
    "issued_at": "2025-01-01T00:00:00Z",
    "expires_at": "2026-01-01T00:00:00Z",
    "customer": "Company Name",
    "max_apps": 100,
    "max_users": 50,
    "features": {
      "operate_log": true,
      "workflow": true,
      "approval": false,
      "comment": true,
      "rbac": true,
      "scheduled_task": false,
      "recycle_bin": true,
      "change_log": true,
      "notification": true,
      "config_management": false,
      "quick_link": true
    },
    "hardware_id": "optional-hardware-binding"
  },
  "signature": "RSA签名（Base64编码）"
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | License ID（唯一标识） |
| `edition` | string | 版本：`community`, `professional`, `enterprise`, `flagship` |
| `issued_at` | time | 签发时间 |
| `expires_at` | time | 过期时间（零值表示永久） |
| `customer` | string | 客户名称 |
| `max_apps` | int | 最大应用数量（0 表示无限制） |
| `max_users` | int | 最大用户数量（0 表示无限制） |
| `features` | object | 功能开关 |
| `hardware_id` | string | 硬件ID（可选，用于硬件绑定） |
| `signature` | string | RSA 签名（Base64 编码） |

---

## 🔐 安全机制

### 1. RSA 签名验证

- **公钥**：存储在 `license_public_key.pem` 文件中
- **私钥**：由 License 签发方保管（不开源）
- **签名算法**：RSA-PKCS1v15 + SHA256

### 2. 硬件绑定（可选）

- 如果 License 中包含 `hardware_id`，会验证当前机器的硬件ID
- 硬件ID 基于 MAC 地址、CPU ID 等硬件信息生成

### 3. 过期检查

- 检查 `expires_at` 字段
- 如果已过期，License 无效

---

## 🚀 使用方式

### 1. 服务器启动时加载

```go
// core/app-server/server/server.go
func (s *Server) initLicense(ctx context.Context) error {
    licenseMgr := license.GetManager()
    return licenseMgr.LoadLicense("") // 使用默认路径
}
```

### 2. 检查功能可用性

```go
// 在业务代码中
licenseMgr := license.GetManager()

if licenseMgr.HasFeature("operate_log") {
    // 使用操作日志功能
    logger.CreateOperateLogger(...)
}
```

### 3. 企业功能注册

```go
// enterprise_impl/operatelog/init.go
func init() {
    // 检查 License
    licenseMgr := license.GetManager()
    if licenseMgr.HasFeature("operate_log") {
        // 注册企业实现
        enterprise.RegisterOperateLogger(service.NewOperateLogService())
    }
}
```

---

## 📂 License 文件路径

### 查找优先级

1. **环境变量**：`LICENSE_PATH`
2. **当前目录**：`./license.json`
3. **用户目录**：`~/.ai-agent-os/license.json`

### 公钥文件路径

1. **环境变量**：`LICENSE_PUBLIC_KEY_PATH`
2. **当前目录**：`./license_public_key.pem`
3. **用户目录**：`~/.ai-agent-os/license_public_key.pem`

---

## 🔧 开发工具

### 生成 License 文件（需要私钥）

```go
// tools/license-generator/main.go
// 这个工具不在开源仓库中，由 License 签发方使用
```

### 测试 License

```go
// 测试时可以设置自定义路径
licenseMgr := license.GetManager()
licenseMgr.SetLicensePath("./test-license.json")
licenseMgr.LoadLicense("")
```

---

## 📊 版本对比

| 功能 | 社区版 | 专业版 | 企业版 | 旗舰版 |
|------|--------|--------|--------|--------|
| 基础功能 | ✅ | ✅ | ✅ | ✅ |
| 操作日志 | ❌ | ✅ | ✅ | ✅ |
| 工作流 | ❌ | ❌ | ✅ | ✅ |
| 审批流程 | ❌ | ❌ | ✅ | ✅ |
| 权限管理 | ❌ | ❌ | ✅ | ✅ |
| 定时任务 | ❌ | ❌ | ❌ | ✅ |
| 回收站 | ❌ | ❌ | ✅ | ✅ |
| 变更日志 | ❌ | ❌ | ✅ | ✅ |
| 通知中心 | ❌ | ❌ | ✅ | ✅ |
| 配置管理 | ❌ | ❌ | ❌ | ✅ |
| 快链 | ❌ | ❌ | ✅ | ✅ |

---

## ⚠️ 注意事项

1. **社区版不需要 License**：没有 License 文件时，自动使用社区版
2. **License 验证失败**：如果验证失败，会降级到社区版（不中断启动）
3. **功能开关**：每个功能都需要在 License 中明确开启
4. **硬件绑定**：如果启用硬件绑定，License 只能在指定机器上使用

---

## 🔄 更新流程

1. **客户申请**：客户向 License 签发方申请 License
2. **生成 License**：使用私钥生成签名 License 文件
3. **交付 License**：将 License 文件交付给客户
4. **客户部署**：客户将 License 文件放到指定路径
5. **自动生效**：服务器启动时自动加载和验证

---

## 📞 技术支持

如有问题，请联系技术支持团队。
