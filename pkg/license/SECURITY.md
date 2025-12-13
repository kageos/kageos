# License 安全防护方案（企业部署场景）

## 🎯 核心目标

**防止 License 被破解，保护企业版功能不被非法使用**

**重要说明**：
- ✅ 这是**企业部署**，不是单机软件
- ✅ 可能是**单机部署**，也可能是**集群部署**
- ✅ License 验证是**部署级**的，不是服务级的
- ✅ 多个服务共享同一个 License

---

## 🏗️ 部署场景

### 单机部署

```
所有服务在同一台机器
License 文件：./license.json
```

### 集群部署

```
多个服务分布在多台机器
License 文件：/shared/license.json（共享存储）
```

---

## 🔍 常见破解方式分析

### 1. 修改 License 文件

**破解方式**：
- 修改过期时间（`expires_at`）
- 修改功能开关（`features`）
- 修改资源限制（`max_apps`、`max_users`）

**防护措施**：
- ✅ **RSA 签名验证**（已实现）
- ✅ **签名验证失败会拒绝 License**

---

### 2. 绕过签名验证

**破解方式**：
- 修改验证代码，跳过签名检查
- 替换公钥文件
- 修改公钥加载逻辑

**防护措施**：
- ⚠️ **代码混淆**（需要实现）
- ⚠️ **反调试**（需要实现）
- ⚠️ **公钥嵌入**（需要实现）

---

### 3. 修改代码逻辑

**破解方式**：
- 修改 `HasFeature()` 方法，总是返回 `true`
- 修改 `IsEnterprise()` 方法，总是返回 `true`
- 跳过 License 检查

**防护措施**：
- ⚠️ **代码混淆**（需要实现）
- ⚠️ **反调试**（需要实现）
- ⚠️ **关键逻辑加密**（需要实现）

---

### 4. 时间回退

**破解方式**：
- 修改系统时间，绕过过期检查
- 使用时间回退工具

**防护措施**：
- ⚠️ **在线时间验证**（需要实现）
- ⚠️ **时间戳验证**（需要实现）

---

### 5. 硬件ID伪造

**破解方式**：
- 修改硬件ID生成逻辑
- 伪造硬件ID

**防护措施**：
- ⚠️ **硬件指纹算法**（需要实现）
- ⚠️ **多重硬件信息**（需要实现）

---

## 🛡️ 多层防护方案

### 第一层：RSA 签名验证（已实现）✅

**防护能力**：⭐⭐⭐⭐（高）

**实现**：
- License 文件包含 RSA 签名
- 使用公钥验证签名
- 签名验证失败会拒绝 License

**优点**：
- ✅ 防止修改 License 文件
- ✅ 防止伪造 License

**缺点**：
- ⚠️ 如果公钥被替换，可能被绕过
- ⚠️ 如果验证代码被修改，可能被绕过

---

### 第二层：部署标识验证（推荐）⭐⭐⭐⭐

**防护能力**：⭐⭐⭐⭐（高）

**设计**：
- ✅ **部署级验证**：一个 License 对应一个部署
- ✅ **部署标识**：使用 `deployment_id` 标识部署
- ✅ **防止 License 共享**：不同部署不能共享 License

**实现方案**：

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

**集群部署考虑**：
- ✅ 所有服务使用相同的 `DEPLOYMENT_ID`
- ✅ License 中的 `deployment_id` 必须匹配
- ✅ 防止 License 在不同部署间共享

---

### 第三层：硬件绑定（仅单机部署）⭐⭐⭐

**防护能力**：⭐⭐⭐（中）

**设计**：
- ✅ **单机部署**：启用硬件绑定
- ❌ **集群部署**：不启用硬件绑定（服务可能在不同机器）

**实现方案**：

```go
// pkg/license/manager.go

// verifyHardwareBinding 验证硬件绑定
func (m *Manager) verifyHardwareBinding(license *License) error {
    // 如果 License 有 hardware_id，验证是否匹配
    if license.HardwareID != "" {
        // 检查部署类型
        if license.DeploymentType == "cluster" {
            // 集群部署：不启用硬件绑定
            return nil
        }
        
        // 单机部署：验证硬件ID
        currentHardwareID := getHardwareID()
        if currentHardwareID != license.HardwareID {
            return fmt.Errorf("license hardware binding mismatch")
        }
    }
    return nil
}

// getHardwareID 获取硬件ID（仅单机部署）
func getHardwareID() string {
    // 基于多个硬件信息生成唯一ID
    // MAC 地址、CPU ID、主板序列号等
    // ...
}
```

---

### 第三层：在线验证（推荐）⭐⭐⭐⭐⭐

**防护能力**：⭐⭐⭐⭐⭐（极高）

**实现方案**：
- ✅ **定期在线验证**（每天或每周）
- ✅ **服务器端验证**（License 服务器）
- ✅ **License 状态同步**（撤销、续期等）

**实现方案**：

```go
// pkg/license/online_verify.go
package license

import (
    "context"
    "crypto/tls"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

// OnlineVerifier 在线验证器
type OnlineVerifier struct {
    verifyURL string
    client    *http.Client
    interval  time.Duration // 验证间隔
}

// NewOnlineVerifier 创建在线验证器
func NewOnlineVerifier(verifyURL string) *OnlineVerifier {
    return &OnlineVerifier{
        verifyURL: verifyURL,
        client: &http.Client{
            Timeout: 10 * time.Second,
            Transport: &http.Transport{
                TLSClientConfig: &tls.Config{
                    InsecureSkipVerify: false, // 验证服务器证书
                },
            },
        },
        interval: 24 * time.Hour, // 每24小时验证一次
    }
}

// VerifyLicense 在线验证 License
func (v *OnlineVerifier) VerifyLicense(ctx context.Context, licenseID string) (*VerifyResponse, error) {
    // 构建验证请求
    req, err := http.NewRequestWithContext(ctx, "POST", v.verifyURL, nil)
    if err != nil {
        return nil, err
    }
    
    // 添加 License ID
    req.Header.Set("X-License-ID", licenseID)
    
    // 添加硬件ID（用于验证）
    req.Header.Set("X-Hardware-ID", getHardwareID())
    
    // 发送请求
    resp, err := v.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to verify license online: %w", err)
    }
    defer resp.Body.Close()
    
    // 解析响应
    var verifyResp VerifyResponse
    if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
        return nil, fmt.Errorf("failed to parse verify response: %w", err)
    }
    
    return &verifyResp, nil
}

// VerifyResponse 验证响应
type VerifyResponse struct {
    Valid      bool      `json:"valid"`       // License 是否有效
    Revoked    bool      `json:"revoked"`     // License 是否被撤销
    ExpiresAt  time.Time `json:"expires_at"`  // 过期时间
    Message    string    `json:"message"`     // 消息
}

// StartPeriodicVerify 启动定期验证
func (v *OnlineVerifier) StartPeriodicVerify(ctx context.Context, licenseID string) {
    ticker := time.NewTicker(v.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // 执行验证
            resp, err := v.VerifyLicense(ctx, licenseID)
            if err != nil {
                logger.Warnf(ctx, "[License] Online verification failed: %v", err)
                continue
            }
            
            // 检查 License 状态
            if resp.Revoked {
                logger.Errorf(ctx, "[License] License has been revoked: %s", resp.Message)
                // 可以在这里禁用企业功能
            }
            
            if !resp.Valid {
                logger.Errorf(ctx, "[License] License is invalid: %s", resp.Message)
                // 可以在这里禁用企业功能
            }
        }
    }
}
```

---

### 第四层：代码混淆和反调试（可选）⭐⭐⭐

**防护能力**：⭐⭐⭐（中）

**实现方案**：
- ⚠️ **代码混淆**（使用工具如 `garble`）
- ⚠️ **反调试**（检测调试器）
- ⚠️ **关键逻辑加密**（加密 License 检查逻辑）

**实现方案**：

```go
// pkg/license/anti_debug.go
package license

import (
    "os"
    "runtime"
    "syscall"
)

// IsDebugging 检测是否在调试
func IsDebugging() bool {
    // 检测方法1：检查父进程
    ppid := os.Getppid()
    if ppid > 0 {
        // 检查父进程名称
        // ...
    }
    
    // 检测方法2：检查调试器（Linux）
    if runtime.GOOS == "linux" {
        // 检查 /proc/self/status 中的 TracerPid
        // ...
    }
    
    // 检测方法3：检查环境变量
    if os.Getenv("GODEBUG") != "" {
        return true
    }
    
    return false
}

// CheckAntiDebug 检查反调试
func CheckAntiDebug() error {
    if IsDebugging() {
        return fmt.Errorf("debugging detected, license verification disabled")
    }
    return nil
}
```

---

### 第五层：时间验证（推荐）⭐⭐⭐⭐

**防护能力**：⭐⭐⭐⭐（高）

**实现方案**：
- ✅ **在线时间验证**（从服务器获取时间）
- ✅ **时间戳验证**（防止时间回退）
- ✅ **时间窗口检查**（防止时间跳跃）

**实现方案**：

```go
// pkg/license/time_verify.go
package license

import (
    "context"
    "time"
)

// TimeVerifier 时间验证器
type TimeVerifier struct {
    lastServerTime time.Time
    lastLocalTime  time.Time
    verifyURL      string
}

// VerifyTime 验证时间（防止时间回退）
func (v *TimeVerifier) VerifyTime(ctx context.Context) error {
    // 1. 从服务器获取时间
    serverTime, err := v.getServerTime(ctx)
    if err != nil {
        // 如果无法获取服务器时间，使用本地时间（但记录警告）
        logger.Warnf(ctx, "[License] Failed to get server time, using local time")
        return nil
    }
    
    // 2. 检查时间是否回退
    if !v.lastServerTime.IsZero() && serverTime.Before(v.lastServerTime) {
        return fmt.Errorf("time rollback detected, license verification disabled")
    }
    
    // 3. 检查时间差异（防止时间跳跃）
    localTime := time.Now()
    diff := serverTime.Sub(localTime)
    if diff > 24*time.Hour || diff < -24*time.Hour {
        return fmt.Errorf("time difference too large, license verification disabled")
    }
    
    // 4. 更新记录
    v.lastServerTime = serverTime
    v.lastLocalTime = localTime
    
    return nil
}

// getServerTime 从服务器获取时间
func (v *TimeVerifier) getServerTime(ctx context.Context) (time.Time, error) {
    // 从 License 验证服务器获取时间
    // ...
}
```

---

### 第六层：服务器端验证（推荐）⭐⭐⭐⭐⭐

**防护能力**：⭐⭐⭐⭐⭐（极高）

**实现方案**：
- ✅ **License 服务器**（集中管理所有 License）
- ✅ **定期验证**（每天或每周）
- ✅ **实时撤销**（可以立即撤销 License）
- ✅ **使用统计**（监控 License 使用情况）

**架构设计**：

```
客户端
  ↓
定期请求 License 服务器
  ↓
License 服务器验证
  ├─ License 是否有效
  ├─ License 是否被撤销
  ├─ 硬件ID是否匹配
  └─ 使用统计
  ↓
返回验证结果
  ├─ 有效：继续使用
  └─ 无效：禁用企业功能
```

---

## 🎯 推荐方案（企业部署场景）

### 基础防护（必须实现）⭐⭐⭐⭐⭐

1. ✅ **RSA 签名验证**（已实现）
2. ✅ **过期检查**（已实现）
3. ✅ **部署标识验证**（需要实现）

---

### 增强防护（强烈推荐）⭐⭐⭐⭐

4. ✅ **在线验证**（定期验证）
5. ✅ **时间验证**（防止时间回退）

---

### 可选防护（按需启用）⭐⭐⭐

6. ⚠️ **硬件绑定**（仅单机部署）
7. ⚠️ **代码混淆**（可选）
8. ⚠️ **反调试**（可选）

---

## 💻 实现方案（企业部署场景）

### 1. 部署标识验证（必须）

```go
// pkg/license/hardware.go
package license

import (
    "crypto/sha256"
    "encoding/hex"
    "os"
    "os/exec"
    "runtime"
    "strings"
)

// getHardwareID 获取硬件ID（用于硬件绑定）
func getHardwareID() string {
    var parts []string
    
    // 1. MAC 地址
    if mac := getMACAddress(); mac != "" {
        parts = append(parts, mac)
    }
    
    // 2. 机器ID（Linux）
    if machineID := getMachineID(); machineID != "" {
        parts = append(parts, machineID)
    }
    
    // 3. CPU ID（Linux）
    if cpuID := getCPUID(); cpuID != "" {
        parts = append(parts, cpuID)
    }
    
    // 4. 主板序列号（Linux）
    if boardSerial := getBoardSerial(); boardSerial != "" {
        parts = append(parts, boardSerial)
    }
    
    // 5. 主机名（fallback）
    if len(parts) == 0 {
        if hostname, err := os.Hostname(); err == nil {
            parts = append(parts, hostname)
        }
    }
    
    // 组合并哈希
    combined := strings.Join(parts, "|")
    hash := sha256.Sum256([]byte(combined))
    return hex.EncodeToString(hash[:16]) // 使用前16字节
}

// getMACAddress 获取 MAC 地址
func getMACAddress() string {
    if runtime.GOOS == "linux" {
        // Linux: 获取第一个非回环网络接口的 MAC 地址
        cmd := exec.Command("sh", "-c", "ip link show | grep -A 1 'state UP' | grep -oP 'link/ether \\K[^ ]+' | head -1")
        output, err := cmd.Output()
        if err == nil {
            return strings.TrimSpace(string(output))
        }
    }
    // TODO: 其他平台
    return ""
}

// getMachineID 获取机器ID（Linux）
func getMachineID() string {
    if runtime.GOOS == "linux" {
        data, err := os.ReadFile("/etc/machine-id")
        if err == nil {
            return strings.TrimSpace(string(data))
        }
    }
    return ""
}

// getCPUID 获取 CPU ID（Linux）
func getCPUID() string {
    if runtime.GOOS == "linux" {
        // 读取 /proc/cpuinfo
        data, err := os.ReadFile("/proc/cpuinfo")
        if err == nil {
            lines := strings.Split(string(data), "\n")
            for _, line := range lines {
                if strings.HasPrefix(line, "Serial") {
                    parts := strings.Split(line, ":")
                    if len(parts) == 2 {
                        return strings.TrimSpace(parts[1])
                    }
                }
            }
        }
    }
    return ""
}

// getBoardSerial 获取主板序列号（Linux）
func getBoardSerial() string {
    if runtime.GOOS == "linux" {
        data, err := os.ReadFile("/sys/class/dmi/id/board_serial")
        if err == nil {
            return strings.TrimSpace(string(data))
        }
    }
    return ""
}
```

---

### 2. 实现在线验证

```go
// pkg/license/online_verify.go
package license

import (
    "context"
    "crypto/tls"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

// OnlineVerifier 在线验证器
type OnlineVerifier struct {
    verifyURL string
    client    *http.Client
    interval  time.Duration
}

// NewOnlineVerifier 创建在线验证器
func NewOnlineVerifier(verifyURL string) *OnlineVerifier {
    return &OnlineVerifier{
        verifyURL: verifyURL,
        client: &http.Client{
            Timeout: 10 * time.Second,
            Transport: &http.Transport{
                TLSClientConfig: &tls.Config{
                    InsecureSkipVerify: false,
                },
            },
        },
        interval: 24 * time.Hour, // 每24小时验证一次
    }
}

// VerifyLicense 在线验证 License
func (v *OnlineVerifier) VerifyLicense(ctx context.Context, licenseID, hardwareID string) (*VerifyResponse, error) {
    // 构建验证请求
    url := fmt.Sprintf("%s/verify?license_id=%s&hardware_id=%s", v.verifyURL, licenseID, hardwareID)
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }
    
    // 发送请求
    resp, err := v.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to verify license online: %w", err)
    }
    defer resp.Body.Close()
    
    // 解析响应
    var verifyResp VerifyResponse
    if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
        return nil, fmt.Errorf("failed to parse verify response: %w", err)
    }
    
    return &verifyResp, nil
}

// VerifyResponse 验证响应
type VerifyResponse struct {
    Valid     bool      `json:"valid"`      // License 是否有效
    Revoked   bool      `json:"revoked"`    // License 是否被撤销
    ExpiresAt time.Time `json:"expires_at"` // 过期时间
    Message   string    `json:"message"`    // 消息
}

// StartPeriodicVerify 启动定期验证
func (v *OnlineVerifier) StartPeriodicVerify(ctx context.Context, licenseID, hardwareID string, callback func(bool, string)) {
    go func() {
        ticker := time.NewTicker(v.interval)
        defer ticker.Stop()
        
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                // 执行验证
                resp, err := v.VerifyLicense(ctx, licenseID, hardwareID)
                if err != nil {
                    logger.Warnf(ctx, "[License] Online verification failed: %v", err)
                    // 验证失败时，可以选择禁用企业功能或继续使用
                    continue
                }
                
                // 检查 License 状态
                if resp.Revoked || !resp.Valid {
                    logger.Errorf(ctx, "[License] License is invalid or revoked: %s", resp.Message)
                    // 回调通知
                    if callback != nil {
                        callback(false, resp.Message)
                    }
                } else {
                    logger.Infof(ctx, "[License] Online verification successful")
                    if callback != nil {
                        callback(true, "")
                    }
                }
            }
        }
    }()
}
```

---

### 3. 集成到 License 管理器

```go
// pkg/license/manager.go（增强版）

// Manager License 管理器（增强版）
type Manager struct {
    license        *License
    licensePath    string
    publicKey      *rsa.PublicKey
    onlineVerifier *OnlineVerifier // ⭐ 在线验证器
    mu             sync.RWMutex
    lastVerifyTime time.Time // ⭐ 上次验证时间
}

// LoadLicense 加载 License（增强版）
func (m *Manager) LoadLicense(path string) error {
    // ... 原有逻辑 ...
    
    // ⭐ 如果 License 有在线验证 URL，启动定期验证
    if m.license.VerifyURL != "" {
        m.onlineVerifier = NewOnlineVerifier(m.license.VerifyURL)
        m.onlineVerifier.StartPeriodicVerify(
            context.Background(),
            m.license.ID,
            getHardwareID(),
            m.onLicenseStatusChanged, // 回调函数
        )
    }
    
    return nil
}

// onLicenseStatusChanged License 状态变更回调
func (m *Manager) onLicenseStatusChanged(valid bool, message string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if !valid {
        // License 无效或被撤销，清除 License
        logger.Errorf(nil, "[License] License invalidated: %s", message)
        m.license = nil // 降级到社区版
    }
}
```

---

## 🎯 推荐实施策略

### 阶段一：基础防护（立即实施）⭐⭐⭐⭐⭐

1. ✅ **完善硬件绑定**
   - 实现 `getHardwareID()` 函数
   - 基于多个硬件信息生成唯一ID

2. ✅ **增强签名验证**
   - 确保公钥文件安全
   - 考虑将公钥嵌入到二进制文件中

---

### 阶段二：增强防护（3-6个月）⭐⭐⭐⭐

3. ✅ **实现在线验证**
   - 搭建 License 验证服务器
   - 实现定期验证机制

4. ✅ **实现时间验证**
   - 防止时间回退
   - 时间窗口检查

---

### 阶段三：高级防护（可选）⭐⭐⭐

5. ⚠️ **代码混淆**
   - 使用 `garble` 工具
   - 混淆关键代码

6. ⚠️ **反调试**
   - 检测调试器
   - 检测代码修改

---

## 📊 防护效果评估

### 基础防护（RSA + 硬件绑定）

**防护能力**：⭐⭐⭐⭐（高）
- ✅ 防止修改 License 文件
- ✅ 防止伪造 License
- ✅ 防止 License 共享
- ⚠️ 可能被代码修改绕过

---

### 增强防护（+ 在线验证）

**防护能力**：⭐⭐⭐⭐⭐（极高）
- ✅ 防止修改 License 文件
- ✅ 防止伪造 License
- ✅ 防止 License 共享
- ✅ 可以实时撤销 License
- ✅ 可以监控 License 使用情况
- ⚠️ 需要网络连接

---

### 高级防护（+ 代码混淆 + 反调试）

**防护能力**：⭐⭐⭐⭐⭐（极高）
- ✅ 所有增强防护的功能
- ✅ 防止代码修改
- ✅ 防止调试分析
- ⚠️ 增加开发复杂度

---

## 🎯 最终建议

### 推荐方案：基础防护 + 在线验证

**理由**：
1. ✅ **防护能力强**：可以防止大部分破解方式
2. ✅ **实施成本适中**：不需要复杂的代码混淆
3. ✅ **可维护性好**：代码清晰，易于维护
4. ✅ **可扩展性强**：可以后续添加高级防护

**实施优先级**：
1. ⭐⭐⭐⭐⭐ **完善硬件绑定**（立即实施）
2. ⭐⭐⭐⭐⭐ **实现在线验证**（3-6个月）
3. ⭐⭐⭐ **代码混淆**（可选）

---

## 📋 实施 Checklist

### 立即实施

- [ ] 完善 `getHardwareID()` 函数
- [ ] 实现基于多个硬件信息的硬件指纹算法
- [ ] 测试硬件绑定功能

### 3-6个月实施

- [ ] 搭建 License 验证服务器
- [ ] 实现在线验证机制
- [ ] 实现定期验证
- [ ] 实现 License 撤销功能

### 可选实施

- [ ] 代码混淆（使用 `garble`）
- [ ] 反调试检测
- [ ] 关键逻辑加密

---

## 📞 参考

- [License 系统设计](./README.md)
- [硬件绑定实现](./hardware.go)（待实现）
- [在线验证实现](./online_verify.go)（待实现）
