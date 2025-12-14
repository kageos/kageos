# 升级企业版流程测试文档

## 📋 完整流程概览

```
1. 启动 Control Service（社区版）
   ↓
2. 通过 API 上传 License 文件激活
   ↓
3. Control Service 保存 License 并发布到 NATS
   ↓
4. 各服务启动时通过 NATS 请求获取 License
   ↓
5. 各服务激活企业版功能
```

---

## 🚀 测试步骤

### 1. 启动 Control Service

```bash
cd core/control-service
go run cmd/app/main.go
```

**预期输出**：
```
[Control Service] NATS connected successfully
[Control Service] Control-service started successfully
[License] License file not found, using community edition
```

**注意**：服务启动后会一直运行（等待信号），这是正常的，不是卡住。

---

### 2. 检查当前状态（社区版）

```bash
curl http://localhost:9096/api/v1/license/status
```

**预期响应**：
```json
{
  "is_valid": false,
  "is_community": true,
  "edition": "community"
}
```

---

### 3. 生成测试 License

```bash
cd tools/license

# 生成密钥对（如果还没有）
bash gen_keypair.sh

# 生成测试 License
go build -o sign_license sign_license.go
./sign_license \
  -private-key keys/private_key.pem \
  -id "test-license-001" \
  -edition enterprise \
  -customer "测试公司" \
  -expires-days 365 \
  -max-apps 100 \
  -max-users 50 \
  -output test_license.json
```

---

### 4. 激活企业版（上传 License）

```bash
curl -X POST http://localhost:9096/api/v1/license/activate \
  -H "Content-Type: application/json" \
  -d @test_license.json
```

**预期响应**：
```json
{
  "message": "license activated successfully",
  "status": {
    "is_valid": true,
    "is_community": false,
    "edition": "enterprise",
    "customer": "测试公司",
    "expires_at": "2026-12-13T23:59:59Z",
    "features": {
      "operate_log": true
    }
  }
}
```

---

### 5. 验证激活状态

```bash
curl http://localhost:9096/api/v1/license/status
```

**预期响应**：
```json
{
  "is_valid": true,
  "is_community": false,
  "edition": "enterprise",
  "customer": "测试公司",
  "expires_at": "2026-12-13T23:59:59Z"
}
```

---

### 6. 检查 License 文件

```bash
cat ./license.json
```

**预期**：License 文件已保存到 Control Service 本地。

---

### 7. 验证 NATS 发布

检查 Control Service 日志，应该看到：
```
[License Service] Published license key to NATS
[License Service] Published refresh instruction to NATS
```

---

### 8. 测试各服务获取 License

启动其他服务（app-server、agent-server 等），它们应该：
1. 通过 NATS 请求获取 License
2. 保存到本地（`~/.ai-agent-os/license.key`）
3. 激活企业版功能

**检查日志**：
```
[License Client] Requesting license key from Control Service...
[License Client] Saved license key to local file
[License Client] License activated: Edition=enterprise
```

---

## ✅ 验证清单

- [ ] Control Service 启动成功
- [ ] 初始状态为社区版
- [ ] 可以成功上传 License 文件
- [ ] License 文件保存到本地
- [ ] License 签名验证通过
- [ ] 状态更新为企业版
- [ ] License 发布到 NATS
- [ ] 刷新指令发布到 NATS
- [ ] 各服务可以获取 License
- [ ] 各服务激活企业版功能

---

## 🔧 故障排查

### 问题1：Control Service 启动卡住

**原因**：服务启动后正常等待信号，不是卡住。

**解决**：这是正常行为，服务会一直运行直到收到 SIGINT 或 SIGTERM 信号。

---

### 问题2：NATS 连接失败

**错误**：
```
failed to connect to NATS: dial tcp 127.0.0.1:4223: connect: connection refused
```

**解决**：
1. 确保 NATS 服务器正在运行
2. 检查配置文件中的 NATS URL 是否正确

---

### 问题3：License 激活失败

**可能原因**：
1. License 文件格式错误
2. License 签名验证失败
3. License 已过期

**检查**：
1. 查看 Control Service 日志
2. 验证 License 文件格式
3. 检查签名是否正确

---

### 问题4：各服务无法获取 License

**可能原因**：
1. NATS 连接失败
2. Control Service 未运行
3. 加密密钥不匹配

**检查**：
1. 确保 Control Service 正在运行
2. 检查各服务的 NATS 配置
3. 确保所有服务使用相同的 `encryption_key`

---

## 📝 注意事项

1. **加密密钥**：所有服务（Control Service 和各服务实例）必须使用相同的 `encryption_key`
2. **NATS 连接**：确保 NATS 服务器正在运行
3. **License 文件**：License 文件必须包含有效的 RSA 签名
4. **过期时间**：License 过期后会自动降级为社区版

---

## 🎯 完整测试脚本

```bash
#!/bin/bash

# 1. 启动 Control Service（后台运行）
cd core/control-service
go run cmd/app/main.go &
CONTROL_PID=$!
sleep 2

# 2. 检查状态
echo "=== 检查初始状态 ==="
curl http://localhost:9096/api/v1/license/status

# 3. 生成测试 License
cd ../../tools/license
go build -o sign_license sign_license.go
./sign_license \
  -private-key keys/private_key.pem \
  -id "test-license-001" \
  -edition enterprise \
  -customer "测试公司" \
  -expires-days 365 \
  -max-apps 100 \
  -max-users 50 \
  -output test_license.json

# 4. 激活 License
echo "=== 激活 License ==="
curl -X POST http://localhost:9096/api/v1/license/activate \
  -H "Content-Type: application/json" \
  -d @test_license.json

# 5. 验证状态
echo "=== 验证激活状态 ==="
curl http://localhost:9096/api/v1/license/status

# 6. 清理
kill $CONTROL_PID
```

