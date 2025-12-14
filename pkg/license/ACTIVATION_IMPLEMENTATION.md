# License 激活功能实现总结

## ✅ 已完成的工作

### 1. 公钥嵌入代码

**位置：** `pkg/license/manager.go`

公钥已经直接嵌入到代码中的 `embeddedPublicKey` 变量，编译时会自动包含到程序中。

**优点：**
- 简化部署，无需单独的公钥文件
- 防止公钥被替换或篡改
- 公钥公开是安全的（RSA 加密的基本原理）

### 2. 密钥对生成工具

**位置：** `tools/license/gen_keypair.sh`

用于生成 RSA 密钥对（私钥和公钥）。

**使用方法：**
```bash
cd tools/license
bash gen_keypair.sh
```

**输出：**
- `keys/private_key.pem` - 私钥（保密，不要泄露）
- `keys/public_key.pem` - 公钥（已嵌入代码，可忽略）

### 3. License 签名工具

**位置：** `tools/license/sign_license.go`

使用私钥对 License 数据进行签名，生成 License 文件。

**编译：**
```bash
cd tools/license
go build -o sign_license sign_license.go
```

**使用方法：**
```bash
./sign_license \
  -private-key keys/private_key.pem \
  -id "license-001" \
  -edition enterprise \
  -customer "客户公司" \
  -expires-days 365 \
  -max-apps 100 \
  -max-users 50 \
  -output license.json
```

## 🔄 完整激活流程

### 步骤 1: 生成密钥对（仅需一次）

```bash
cd tools/license
bash gen_keypair.sh
```

这会生成：
- `keys/private_key.pem` - 私钥（保密，用于签名 License）
- `keys/public_key.pem` - 公钥（已嵌入代码，无需单独部署）

### 步骤 2: 公钥已嵌入代码

公钥已经嵌入到 `pkg/license/manager.go` 中，编译程序时会自动包含。

### 步骤 3: 为客户生成 License

```bash
cd tools/license
go build -o sign_license sign_license.go

./sign_license \
  -private-key keys/private_key.pem \
  -id "license-enterprise-001" \
  -edition enterprise \
  -customer "ABC 公司" \
  -expires-days 365 \
  -max-apps 100 \
  -max-users 50 \
  -output license.json
```

### 步骤 4: 部署 License 文件

将生成的 `license.json` 文件部署到客户环境，放在以下任一位置：
- `./license.json`（当前目录）
- `~/.ai-agent-os/license.json`（用户目录）
- 或通过环境变量 `LICENSE_PATH` 指定路径

### 步骤 5: 程序自动验证

程序启动时会自动：
1. 加载 License 文件（如果存在）
2. 使用嵌入的公钥验证签名
3. 检查 License 是否过期
4. 如果验证通过，激活企业版功能
5. 如果 License 不存在或无效，使用社区版

## 🔐 安全机制

### 1. RSA 签名验证

- **私钥**：用于签名 License（保密，仅用于生成 License）
- **公钥**：用于验证签名（嵌入代码，可以公开）
- **签名**：License 数据使用 SHA-256 哈希后，用私钥签名
- **验证**：程序使用公钥验证签名，确保 License 未被篡改

### 2. 防篡改保护

- License 文件包含签名，任何修改都会导致签名验证失败
- 即使有人修改了 License 内容，也无法重新生成有效签名（因为没有私钥）

### 3. 过期检查

- License 包含过期时间，程序会检查是否过期
- 过期后自动降级为社区版

## 📋 License 文件格式

```json
{
  "license": {
    "id": "license-001",
    "edition": "enterprise",
    "issued_at": "2025-12-13T00:00:00Z",
    "expires_at": "2026-12-13T00:00:00Z",
    "customer": "客户公司",
    "max_apps": 100,
    "max_users": 50,
    "features": {
      "operate_log": true,
      "workflow": true,
      ...
    }
  },
  "signature": "RSA签名（Base64编码）"
}
```

## 🧪 测试验证

已测试验证：
- ✅ 密钥对生成
- ✅ License 签名生成
- ✅ License 签名验证
- ✅ 公钥嵌入代码加载
- ✅ 过期时间检查
- ✅ 功能开关读取

## 📝 注意事项

1. **私钥安全**
   - 私钥必须严格保密，不要提交到代码仓库
   - 建议将私钥存储在安全的密钥管理系统中
   - 定期备份私钥到安全的地方

2. **公钥公开**
   - 公钥可以安全地编译到程序中
   - 公钥公开不会带来安全风险（这是 RSA 加密的基本原理）

3. **License 文件**
   - License 文件包含签名，无法被篡改
   - 即使被修改，签名验证也会失败

4. **密钥轮换**
   - 如果需要更换密钥对，需要：
     1. 生成新的密钥对
     2. 更新代码中的公钥
     3. 重新编译程序
     4. 使用新私钥为所有客户重新生成 License

## 🎯 下一步

激活功能已经完成，可以：
1. 使用 `sign_license` 工具为客户生成 License
2. 将 License 文件部署到客户环境
3. 程序会自动验证并激活企业版功能

如果需要测试，可以：
1. 生成一个测试 License
2. 将 License 文件放在程序运行目录
3. 启动程序，查看日志确认 License 加载成功
