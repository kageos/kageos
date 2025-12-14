# License 激活流程详解

## 🔐 公钥和私钥是什么？

### 简单理解

想象一下：
- **私钥** = 你的**签名笔**（只有你有，用来签名）
- **公钥** = 你的**签名验证器**（所有人都能看到，用来验证签名）

### 工作原理

1. **签名过程**（用私钥）：
   - 你有一份 License 文件（包含客户信息、过期时间等）
   - 用私钥对 License 进行签名
   - 生成一个签名（就像你的手写签名）

2. **验证过程**（用公钥）：
   - 程序收到 License 文件
   - 用公钥验证签名
   - 如果签名正确 → License 有效
   - 如果签名错误 → License 无效（可能是伪造的）

### 关键点

- ✅ **公钥可以公开**：所有人都能看到，不影响安全
- ✅ **私钥必须保密**：只有你能签名，泄露了别人就能伪造 License
- ✅ **无法伪造**：没有私钥，无法生成有效的签名

---

## 🎯 License 激活流程（最佳实践）

### 整体流程

```
┌─────────────────────────────────────────────────────────┐
│  1. 生成密钥对（你，作为软件提供商）                      │
│                                                         │
│  私钥（保密） ←→ 公钥（公开）                            │
└─────────────────────────────────────────────────────────┘
                    │
                    │ 公钥编译到程序中
                    ↓
┌─────────────────────────────────────────────────────────┐
│  2. 编译程序（公钥已经嵌入）                              │
│                                                         │
│  程序 = 代码 + 公钥                                      │
└─────────────────────────────────────────────────────────┘
                    │
                    │ 分发给客户
                    ↓
┌─────────────────────────────────────────────────────────┐
│  3. 客户部署程序                                         │
│                                                         │
│  客户拿到：程序（包含公钥）                            │
└─────────────────────────────────────────────────────────┘
                    │
                    │ 客户需要激活
                    ↓
┌─────────────────────────────────────────────────────────┐
│  4. 你生成 License（用私钥签名）                          │
│                                                         │
│  License 文件 = License 数据 + 签名                      │
└─────────────────────────────────────────────────────────┘
                    │
                    │ 发送给客户
                    ↓
┌─────────────────────────────────────────────────────────┐
│  5. 客户激活                                             │
│                                                         │
│  程序读取 License → 用公钥验证签名 → 激活成功            │
└─────────────────────────────────────────────────────────┘
```

---

## 📋 详细步骤

### 步骤 1：生成密钥对（一次性，你来做）

```bash
# 生成私钥（2048位，足够安全）
openssl genrsa -out private_key.pem 2048

# 从私钥生成公钥
openssl rsa -in private_key.pem -pubout -out public_key.pem
```

**结果**：
- `private_key.pem` - **私钥**（保密，只有你有）
- `public_key.pem` - **公钥**（可以公开）

---

### 步骤 2：将公钥编译到程序中（编译时）

**方式一：使用 embed（推荐）**

```go
// pkg/license/embedded_public_key.go
package license

import _ "embed"

//go:embed public_key.pem
var embeddedPublicKeyData string

func init() {
    embeddedPublicKey = embeddedPublicKeyData
}
```

**方式二：直接设置（简单）**

```go
// pkg/license/embedded_public_key.go
package license

func init() {
    embeddedPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
（这里是公钥内容）
-----END PUBLIC KEY-----`
}
```

**编译**：
```bash
go build -o app-server ./core/app-server/cmd/app
```

**结果**：程序已经包含公钥，可以验证签名

---

### 步骤 3：客户部署程序

客户拿到编译好的程序，程序里已经包含公钥。

---

### 步骤 4：生成 License（你来做，用私钥签名）

**你需要创建一个 License 签名工具**，流程如下：

```go
// tools/license-signer/main.go
package main

import (
    "crypto/rsa"
    "crypto/sha256"
    "crypto/x509"
    "encoding/base64"
    "encoding/json"
    "encoding/pem"
    "os"
)

func main() {
    // 1. 读取私钥
    privateKeyData, _ := os.ReadFile("private_key.pem")
    block, _ := pem.Decode(privateKeyData)
    privateKey, _ := x509.ParsePKCS1PrivateKey(block.Bytes)
    
    // 2. 创建 License 数据
    license := License{
        ID: "license-xxx",
        Edition: "enterprise",
        Customer: "客户公司名称",
        ExpiresAt: time.Now().Add(365 * 24 * time.Hour), // 1年后过期
        MaxApps: 100,
        MaxUsers: 50,
        Features: Features{
            OperateLog: true,
            Workflow: true,
        },
    }
    
    // 3. 序列化 License
    licenseJSON, _ := json.Marshal(license)
    
    // 4. 计算哈希
    hash := sha256.Sum256(licenseJSON)
    
    // 5. 用私钥签名
    signature, _ := rsa.SignPKCS1v15(nil, privateKey, crypto.SHA256, hash[:])
    
    // 6. 构建 License 文件
    licenseFile := LicenseFile{
        License: license,
        Signature: base64.StdEncoding.EncodeToString(signature),
    }
    
    // 7. 保存到文件
    data, _ := json.Marshal(licenseFile)
    os.WriteFile("license.json", data, 0644)
}
```

**结果**：生成 `license.json` 文件（包含 License 数据和签名）

---

### 步骤 5：客户激活

**客户操作**：
1. 将 `license.json` 文件放到指定位置（如 `./license.json`）
2. 启动程序
3. 程序自动：
   - 读取 License 文件
   - 用公钥验证签名
   - 如果签名正确 → 激活企业版
   - 如果签名错误 → 使用社区版

---

## 🎯 最佳实践建议

### 1. 密钥管理

**✅ 推荐做法**：
- **私钥**：保存在安全的地方（如密钥管理服务、加密存储）
- **公钥**：编译到程序中（公开，不影响安全）
- **密钥轮换**：如果需要更换密钥，需要重新编译程序

**❌ 不推荐**：
- 私钥不要提交到代码仓库
- 私钥不要放在配置文件中
- 不要使用弱密钥（至少 2048 位）

---

### 2. License 生成

**✅ 推荐做法**：
- 创建独立的 License 签名工具（不放在主程序中）
- License 包含必要信息：客户名称、过期时间、功能开关等
- 签名后生成 JSON 文件，方便分发

**❌ 不推荐**：
- 不要在主程序中包含私钥
- 不要使用简单的字符串签名（容易被破解）

---

### 3. 激活流程

**✅ 推荐做法**：
- **自动激活**：程序启动时自动检测 License 文件
- **无需手动操作**：客户只需要放置 License 文件即可
- **容错处理**：如果 License 无效，自动降级到社区版

**❌ 不推荐**：
- 不要要求客户手动输入激活码
- 不要要求在线激活（除非必要）

---

### 4. 分布式部署

**✅ 推荐做法**（我们当前的方案）：
- **Control Service**：统一管理 License，通过 NATS 分发密钥
- **各服务实例**：启动时通过 NATS 请求获取密钥
- **自动刷新**：License 更新时自动推送刷新指令

**优势**：
- 适合集群部署
- 无需共享存储
- 自动同步更新

---

## 🔄 完整激活流程（我们的方案）

### 1. 你（软件提供商）的操作

```bash
# 1. 生成密钥对（一次性）
openssl genrsa -out private_key.pem 2048
openssl rsa -in private_key.pem -pubout -out public_key.pem

# 2. 将公钥编译到程序中
# （使用 embed 或直接设置）

# 3. 编译程序
go build -o app-server ./core/app-server/cmd/app

# 4. 为客户生成 License（使用签名工具）
./license-signer --customer "客户公司" --expires "2026-12-31" --output license.json
```

---

### 2. 客户的操作

```bash
# 1. 部署程序（程序已包含公钥）

# 2. 将 License 文件放到 Control Service
cp license.json /path/to/control-service/

# 3. 启动 Control Service
./control-service

# 4. 启动其他服务（自动获取密钥并激活）
./app-server
./agent-server
```

---

### 3. 程序自动处理

```
Control Service 启动
  ↓
读取 License 文件
  ↓
验证签名（用公钥）
  ↓
如果有效 → 加密并分发密钥
  ↓
各服务启动
  ↓
通过 NATS 请求获取密钥
  ↓
解密并激活企业版
```

---

## 💡 关键理解

### 为什么公钥可以公开？

**类比**：
- 公钥 = 银行的**验钞机**（所有人都能看到，用来验证钞票真伪）
- 私钥 = 印钞厂的**印钞机**（只有银行有，用来印钞票）

**原理**：
- 公钥只能**验证**签名，不能**生成**签名
- 没有私钥，无法伪造有效的签名
- 所以公钥可以公开，不影响安全

---

### 为什么私钥必须保密？

**如果私钥泄露**：
- 任何人都可以用私钥签名 License
- 可以生成任意 License（永久有效、无限用户等）
- 相当于可以免费使用企业版

**保护措施**：
- 私钥保存在安全的地方
- 不要提交到代码仓库
- 使用密钥管理服务（如 Vault）

---

## 🎯 我们的实现方案

### 当前设计

1. **公钥编译到程序**：所有服务都包含公钥
2. **Control Service 管理 License**：统一读取和分发
3. **NATS 分发密钥**：各服务通过 NATS 获取密钥
4. **自动激活**：无需手动操作

### 优势

- ✅ **简单**：客户只需要 License 文件
- ✅ **安全**：私钥不泄露，无法伪造
- ✅ **分布式**：适合集群部署
- ✅ **自动**：无需手动激活

---

## 📝 总结

### 核心概念

1. **私钥**：你用来签名 License（保密）
2. **公钥**：程序用来验证签名（公开，编译到程序中）
3. **License**：包含客户信息和签名（你生成，客户使用）

### 激活流程

1. 你生成密钥对（一次性）
2. 公钥编译到程序（编译时）
3. 你为客户生成 License（用私钥签名）
4. 客户部署程序并放置 License 文件
5. 程序自动验证并激活

### 最佳实践

- ✅ 公钥编译到程序
- ✅ 私钥严格保密
- ✅ 自动激活流程
- ✅ 分布式支持

---

## 🔧 下一步

1. **创建 License 签名工具**：用于生成签名的 License 文件
2. **完善公钥嵌入**：提供 embed 示例
3. **测试激活流程**：确保整个流程正常工作

需要我帮你实现 License 签名工具吗？
