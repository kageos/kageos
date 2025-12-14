# 如何判断服务是否已激活

本文档说明在项目中如何判断服务是否已经激活（企业版）。

---

## 📋 核心概念

### 激活状态的定义

- **未激活（社区版）**：`GetLicense()` 返回 `nil`，或 License 无效/过期
- **已激活（企业版）**：`GetLicense()` 返回非 `nil` 且 `IsValid()` 返回 `true`

### 版本类型

- `community` - 社区版（未激活）
- `professional` - 专业版（已激活）
- `enterprise` - 企业版（已激活）
- `flagship` - 旗舰版（已激活）

---

## 🔍 判断方法

### 方法 1：使用封装的功能检查方法（推荐）⭐

```go
import "github.com/ai-agent-os/ai-agent-os/pkg/license"

// 获取 License 管理器
manager := license.GetManager()

// 检查是否有操作日志功能（推荐：语义化方法）
if manager.HasOperateLogFeature() {
    // 支持操作日志功能，初始化该功能
    fmt.Println("支持操作日志功能")
} else {
    // 不支持该功能，使用默认实现
    fmt.Println("不支持操作日志功能")
}
```

**说明**：
- **推荐使用封装方法**：如 `HasOperateLogFeature()`，更语义化、更易读
- 不同版本（企业版、旗舰版、至尊版等）支持的功能不同
- 每个功能都有对应的封装方法，避免硬编码字符串
- 这样设计便于后续扩展新版本和新功能

### 方法 1.1：使用 HasFeature() 检查功能（通用方法）

```go
import (
    "github.com/ai-agent-os/ai-agent-os/pkg/license"
    "github.com/ai-agent-os/ai-agent-os/enterprise"
)

// 获取 License 管理器
manager := license.GetManager()

// 检查是否有某个功能（使用功能常量）
if manager.HasFeature(enterprise.FeatureOperateLog) {
    // 支持操作日志功能
    fmt.Println("支持操作日志功能")
} else {
    // 不支持该功能
    fmt.Println("不支持操作日志功能")
}
```

**说明**：
- `HasFeature()` 是通用方法，可以检查任意功能
- 功能常量定义在 `enterprise` 包下，避免硬编码字符串
- 如果某个功能没有封装方法，可以使用此方法

### 方法 2：使用 IsActivated() 方法（简单检查）

```go
import "github.com/ai-agent-os/ai-agent-os/pkg/license"

// 获取 License 管理器
manager := license.GetManager()

// 判断是否已激活（激活 + 企业版 + 未过期）
if manager.IsActivated() {
    // 已激活（企业版或旗舰版，且未过期）
    lic := manager.GetLicense()
    fmt.Printf("已激活：%s 版本，客户：%s\n", lic.Edition, lic.Customer)
} else {
    // 未激活（社区版、已过期或非企业版）
    fmt.Println("未激活，使用社区版")
}
```

**说明**：`IsActivated()` 方法会同时检查：
- License 是否存在（不为 nil）
- License 是否有效（未过期）
- 是否是企业版（enterprise 或 flagship）

**注意**：`IsActivated()` 只能判断是否激活，不能判断具体功能。**推荐使用 `HasFeature()` 来精确控制功能**。

### 方法 3：检查 License 是否存在且有效

```go
import "github.com/ai-agent-os/ai-agent-os/pkg/license"

// 获取 License 管理器
manager := license.GetManager()

// 获取当前 License
lic := manager.GetLicense()

// 判断是否已激活
if lic != nil && lic.IsValid() {
    // 已激活（企业版）
    fmt.Printf("已激活：%s 版本，客户：%s\n", lic.Edition, lic.Customer)
} else {
    // 未激活（社区版）
    fmt.Println("未激活，使用社区版")
}
```

### 方法 4：使用 IsEnterprise() 方法

```go
import "github.com/ai-agent-os/ai-agent-os/pkg/license"

manager := license.GetManager()

// 检查是否是企业版（enterprise 或 flagship）
// 注意：此方法不检查 License 是否有效，建议使用 IsActivated()
if manager.IsEnterprise() {
    // 企业版或旗舰版
    lic := manager.GetLicense()
    fmt.Printf("企业版：%s\n", lic.Customer)
} else {
    // 未激活或专业版
    fmt.Println("未激活或专业版")
}
```

### 方法 5：检查版本类型

```go
import "github.com/ai-agent-os/ai-agent-os/pkg/license"

manager := license.GetManager()
edition := manager.GetEdition()

switch edition {
case license.EditionEnterprise, license.EditionFlagship:
    // 已激活（企业版或旗舰版）
    fmt.Println("企业版已激活")
case license.EditionProfessional:
    // 已激活（专业版）
    fmt.Println("专业版已激活")
case license.EditionCommunity:
    // 未激活（社区版）
    fmt.Println("未激活，使用社区版")
}
```

### 方法 5：检查特定功能（不推荐用于激活判断）

```go
import "github.com/ai-agent-os/ai-agent-os/pkg/license"

manager := license.GetManager()

// 检查是否有某个功能
// 注意：此方法用于检查功能可用性，不应用于判断是否激活
// 判断激活应使用 IsActivated() 方法
if manager.HasFeature("operate_log") {
    // 支持操作日志功能
    fmt.Println("支持操作日志功能")
} else {
    // 不支持该功能
    fmt.Println("不支持操作日志功能")
}
```

---

## 📊 完整示例

### 示例 1：在服务初始化时检查

```go
package server

import (
    "github.com/ai-agent-os/ai-agent-os/pkg/license"
    "github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

func (s *Server) initEnterprise() error {
    ctx := s.ctx
    
    // 获取 License 管理器
    licenseMgr := license.GetManager()
    lic := licenseMgr.GetLicense()
    
    // 检查是否有有效的 License
    if lic == nil || !lic.IsValid() {
        logger.Infof(ctx, "[Enterprise] Community edition detected")
        // 社区版：使用空实现
        return nil
    }
    
    // 有有效的 License，根据功能开关初始化企业功能
    logger.Infof(ctx, "[Enterprise] License detected: Edition=%s, Customer=%s",
        lic.Edition, lic.Customer)
    
    // 初始化操作日志功能（如果 License 支持）
    if licenseMgr.HasOperateLogFeature() {
        logger.Infof(ctx, "[Enterprise] Initializing operate log feature...")
        // ... 初始化操作日志
    }
    
    // 后续可以添加更多功能的初始化，例如：
    // if licenseMgr.HasWorkflowFeature() {
    //     // 初始化工作流功能
    // }
    
    return nil
}
```

### 示例 2：在业务逻辑中检查

```go
package service

import (
    "github.com/ai-agent-os/ai-agent-os/pkg/license"
)

func (s *AppService) CreateApp(ctx context.Context, req *CreateAppRequest) error {
    manager := license.GetManager()
    
    // 检查应用数量限制
    currentCount := s.getAppCount(ctx)
    if err := manager.CheckAppLimit(currentCount); err != nil {
        return err // 返回错误，提示用户升级
    }
    
    // 检查是否支持某个功能
    if !manager.HasFeature("operate_log") {
        // 社区版，不记录操作日志
        return s.createAppWithoutLog(ctx, req)
    }
    
    // 企业版，记录操作日志
    return s.createAppWithLog(ctx, req)
}
```

### 示例 3：在 API 中返回激活状态

```go
package api

import (
    "github.com/ai-agent-os/ai-agent-os/pkg/license"
    "github.com/gin-gonic/gin"
)

func (a *API) GetLicenseStatus(c *gin.Context) {
    manager := license.GetManager()
    lic := manager.GetLicense()
    
    status := map[string]interface{}{
        "is_activated": false,
        "edition":     "community",
    }
    
    if lic != nil && lic.IsValid() {
        status["is_activated"] = true
        status["edition"] = lic.Edition
        status["customer"] = lic.Customer
        status["expires_at"] = lic.ExpiresAt
    }
    
    c.JSON(200, status)
}
```

---

## 🎯 常用检查模式

### 模式 1：功能检查（推荐）⭐

```go
manager := license.GetManager()

// 推荐：使用封装的功能检查方法
hasOperateLog := manager.HasOperateLogFeature()

// 或者使用通用方法
hasOperateLog := manager.HasFeature(enterprise.FeatureOperateLog)
```

### 模式 1.1：简单激活检查

```go
manager := license.GetManager()

// 使用 IsActivated() 方法
isActivated := manager.IsActivated()

// 或者手动检查
lic := manager.GetLicense()
isActivated := lic != nil && lic.IsValid() && manager.IsEnterprise()
```

### 模式 2：企业版检查

```go
manager := license.GetManager()
isEnterprise := manager.IsEnterprise()
```

### 模式 3：功能可用性检查

```go
manager := license.GetManager()
hasOperateLog := manager.HasFeature("operate_log")
```

### 模式 4：资源限制检查

```go
manager := license.GetManager()

// 检查应用数量限制
if err := manager.CheckAppLimit(currentAppCount); err != nil {
    return err
}

// 检查用户数量限制
if err := manager.CheckUserLimit(currentUserCount); err != nil {
    return err
}
```

---

## 📝 API 方法说明

### Manager 方法

| 方法 | 返回类型 | 说明 |
|------|----------|------|
| `HasOperateLogFeature()` | `bool` | **推荐**：是否有操作日志功能（封装方法） |
| `HasFeature(featureName string)` | `bool` | 检查是否有某个功能（通用方法） |
| `IsActivated()` | `bool` | 是否已激活（激活 + 企业版 + 未过期） |
| `GetLicense()` | `*License` | 获取当前 License（nil 表示社区版） |
| `IsEnterprise()` | `bool` | 是否是企业版（enterprise 或 flagship，不检查有效性） |
| `GetEdition()` | `Edition` | 获取版本类型 |
| `GetMaxApps()` | `int` | 获取最大应用数量（-1 表示无限制） |
| `GetMaxUsers()` | `int` | 获取最大用户数量（-1 表示无限制） |
| `CheckAppLimit(currentCount int)` | `error` | 检查应用数量是否超过限制 |
| `CheckUserLimit(currentCount int)` | `error` | 检查用户数量是否超过限制 |

### License 方法

| 方法 | 返回类型 | 说明 |
|------|----------|------|
| `IsValid()` | `bool` | License 是否有效（未过期） |
| `HasFeature(featureName string)` | `bool` | 检查是否有某个功能 |
| `GetEdition()` | `Edition` | 获取版本类型 |
| `GetMaxApps()` | `int` | 获取最大应用数量 |
| `GetMaxUsers()` | `int` | 获取最大用户数量 |

---

## ⚠️ 注意事项

### 1. License 可能为 nil

```go
lic := manager.GetLicense()
if lic == nil {
    // 社区版，未激活
    return
}
// 使用 lic 时确保不为 nil
```

### 2. License 可能过期

```go
lic := manager.GetLicense()
if lic != nil && !lic.IsValid() {
    // License 已过期，视为未激活
    return
}
```

### 3. 线程安全

`Manager` 的方法是线程安全的，可以在多个 goroutine 中并发调用。

### 4. 性能考虑

`GetLicense()` 和 `IsEnterprise()` 等方法都是只读操作，性能开销很小，可以频繁调用。

---

## 🔗 相关文档

- [License 使用指南](./USAGE.md)
- [License 管理器 API](./README.md)
- [激活流程说明](./ACTIVATION_FLOW.md)

---

## 💡 最佳实践

1. **在服务启动时检查一次**：在服务初始化时检查激活状态，避免在每次请求时都检查。

2. **使用 IsEnterprise() 进行快速检查**：如果只需要判断是否是企业版，使用 `IsEnterprise()` 比 `GetLicense() != nil` 更语义化。

3. **功能检查优先于版本检查**：如果需要检查某个功能是否可用，直接使用 `HasFeature()` 而不是先检查版本。

4. **资源限制检查**：在创建资源前检查限制，使用 `CheckAppLimit()` 和 `CheckUserLimit()` 方法。

5. **错误处理**：当 License 无效或过期时，应该优雅降级到社区版，而不是直接报错。

