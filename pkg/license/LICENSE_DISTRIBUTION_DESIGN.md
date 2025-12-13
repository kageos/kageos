# License 密钥分发方案（基于 NATS）

## 🎯 核心设计

**NATS 只用于密钥分发，各实例本地保存并自行验证**

- ✅ **NATS 分发密钥**：License 服务通过 NATS 分发 License 密钥
- ✅ **本地保存**：各实例获取密钥后保存到本地
- ✅ **自行验证**：各实例使用本地密钥自行验证，不依赖 NATS
- ✅ **防止模拟**：即使有人模拟 NATS 消息，因为没有密钥，无法破解
- ✅ **无单点问题**：获取到密钥后，后续验证不依赖 NATS

---

## 🏗️ 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────┐
│                    License 服务                          │
│                                                         │
│  ┌──────────────────────────────────────────────┐      │
│  │  License Manager                             │      │
│  │  - 读取 License 文件                         │      │
│  │  - 验证 License（签名、过期、部署ID等）       │      │
│  │  - 加密 License 密钥                         │      │
│  └──────────────────────────────────────────────┘      │
│                    │                                    │
│                    │ NATS 发布（仅分发密钥）            │
│                    ↓                                    │
│  ┌──────────────────────────────────────────────┐      │
│  │  NATS 主题：license.key                      │      │
│  │  - 发布加密的 License 密钥                   │      │
│  │  - 定期发布（确保新实例能获取）              │      │
│  └──────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────┘
                    │
                    │ NATS 订阅（仅获取密钥）
                    ↓
┌─────────────────────────────────────────────────────────┐
│              所有服务实例（获取密钥后本地验证）            │
│                                                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│  │app-server│  │agent-server│ │app-runtime│            │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘            │
│       │             │             │                   │
│       └─────────────┴─────────────┘                   │
│                    │                                   │
│                    │ 订阅主题：license.key             │
│                    ↓                                   │
│  ┌──────────────────────────────────────────────┐     │
│  │  License Client                               │     │
│  │  - 订阅获取密钥                               │     │
│  │  - 保存密钥到本地                             │     │
│  │  - 使用本地密钥自行验证                       │     │
│  │  - 如果无法获取密钥，默认社区版               │     │
│  └──────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────┘
```

---

## 🔐 密钥分发设计

### 密钥格式

**加密的 License 密钥**：
- License 服务读取 License 文件
- 使用**对称加密**（AES）加密整个 License 内容
- 通过 NATS 发布加密后的密钥

**密钥加密方案**：
- 使用 **AES-256-GCM** 加密
- 密钥派生：从 License 服务配置的密钥派生（或使用预共享密钥）
- 或者：使用 License 服务的私钥签名，各实例用公钥验证

---

### NATS 主题设计

**主题名称**：`license.key`

**消息格式**：

```go
// pkg/license/message.go

// LicenseKeyMessage License 密钥消息
type LicenseKeyMessage struct {
    // 加密的 License 内容（Base64 编码）
    EncryptedLicense string `json:"encrypted_license"`
    
    // 加密算法（如 "aes-256-gcm"）
    Algorithm string `json:"algorithm"`
    
    // 时间戳
    Timestamp int64 `json:"timestamp"`
}
```

---

## 🔄 工作流程

### 1. License 服务启动

```
License 服务启动
  ↓
连接 NATS
  ↓
读取 License 文件
  ↓
验证 License（签名、过期等）
  ↓
加密 License 内容（AES-256-GCM）
  ↓
发布加密的 License 密钥到 NATS（主题：license.key）
  ↓
定期发布（每5分钟，确保新实例能获取）
```

---

### 2. 服务实例启动

```
服务实例启动
  ↓
连接 NATS
  ├─ 成功：继续
  └─ 失败：默认社区版（不获取密钥）
  ↓
检查本地是否有密钥文件
  ├─ 有：使用本地密钥验证
  └─ 无：从 NATS 获取密钥
       ├─ 成功：保存到本地，使用密钥验证
       └─ 失败：默认社区版
  ↓
使用本地密钥自行验证
  ├─ 有效：启用企业功能
  └─ 无效：降级到社区版
```

---

### 3. 新增实例启动

```
新增实例启动
  ↓
连接 NATS
  ↓
订阅 license.key 主题（等待密钥）
  ↓
收到密钥消息
  ↓
保存密钥到本地
  ↓
使用本地密钥自行验证
  ├─ 有效：启用企业功能
  └─ 无效：降级到社区版
```

---

## 💻 实现方案

### 1. License 服务（密钥分发）

```go
// core/license-server/server/server.go

// Server License 服务
type Server struct {
    manager      *license.Manager
    natsConn     *nats.Conn
    licensePath  string
    encryptionKey []byte  // AES 加密密钥
    ticker       *time.Ticker
}

// NewServer 创建 License 服务
func NewServer(natsURL, licensePath string, encryptionKey []byte) (*Server, error) {
    conn, err := nats.Connect(natsURL)
    if err != nil {
        return nil, err
    }
    
    manager := license.NewManager()
    
    server := &Server{
        manager: manager,
        natsConn: conn,
        licensePath: licensePath,
        encryptionKey: encryptionKey,
    }
    
    return server, nil
}

// Start 启动服务
func (s *Server) Start(ctx context.Context) error {
    // 1. 加载 License
    if err := s.loadLicense(); err != nil {
        logger.Warnf(ctx, "[License Service] Failed to load license: %v", err)
        return nil // 即使加载失败，也要继续运行（社区版）
    }
    
    // 2. 发布初始密钥
    s.publishLicenseKey()
    
    // 3. 启动定期任务（每5分钟发布一次，确保新实例能获取）
    s.ticker = time.NewTicker(5 * time.Minute)
    go s.startPeriodicTasks(ctx)
    
    return nil
}

// loadLicense 加载 License
func (s *Server) loadLicense() error {
    return s.manager.LoadLicense(s.licensePath)
}

// publishLicenseKey 发布 License 密钥
func (s *Server) publishLicenseKey() {
    lic := s.manager.GetLicense()
    if lic == nil {
        // 没有 License，不发布密钥（社区版）
        return
    }
    
    // 序列化 License
    licenseData, err := json.Marshal(lic)
    if err != nil {
        logger.Errorf(nil, "[License Service] Failed to marshal license: %v", err)
        return
    }
    
    // 加密 License
    encrypted, err := s.encryptLicense(licenseData)
    if err != nil {
        logger.Errorf(nil, "[License Service] Failed to encrypt license: %v", err)
        return
    }
    
    // 构建消息
    msg := &license.LicenseKeyMessage{
        EncryptedLicense: base64.StdEncoding.EncodeToString(encrypted),
        Algorithm: "aes-256-gcm",
        Timestamp: time.Now().Unix(),
    }
    
    // 发布到 NATS
    data, _ := json.Marshal(msg)
    s.natsConn.Publish("license.key", data)
    
    logger.Infof(nil, "[License Service] Published license key to NATS")
}

// encryptLicense 加密 License
func (s *Server) encryptLicense(data []byte) ([]byte, error) {
    // 使用 AES-256-GCM 加密
    block, err := aes.NewCipher(s.encryptionKey)
    if err != nil {
        return nil, err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }
    
    ciphertext := gcm.Seal(nonce, nonce, data, nil)
    return ciphertext, nil
}

// startPeriodicTasks 启动定期任务
func (s *Server) startPeriodicTasks(ctx context.Context) {
    for {
        select {
        case <-s.ticker.C:
            // 定期发布密钥（确保新实例能获取）
            s.publishLicenseKey()
            
            // 检查 License 是否过期
            lic := s.manager.GetLicense()
            if lic != nil && !lic.IsValid() {
                logger.Warnf(nil, "[License Service] License expired, stopping key distribution")
                // License 过期，停止分发密钥
                return
            }
            
        case <-ctx.Done():
            return
        }
    }
}
```

---

### 2. License Client（各服务实例）

```go
// pkg/license/client.go

// Client License 客户端
type Client struct {
    natsConn     *nats.Conn
    subscription *nats.Subscription
    manager      *Manager
    keyPath      string  // 本地密钥文件路径
    encryptionKey []byte // AES 解密密钥（与 License 服务相同）
    mu           sync.RWMutex
}

// NewClient 创建 License 客户端
func NewClient(natsURL, keyPath string, encryptionKey []byte) (*Client, error) {
    // 连接 NATS
    conn, err := nats.Connect(natsURL)
    if err != nil {
        // 无法连接 NATS，返回 nil（视为社区版）
        logger.Warnf(nil, "[License Client] Failed to connect to NATS: %v, using community edition", err)
        return nil, nil
    }
    
    manager := NewManager()
    
    client := &Client{
        natsConn: conn,
        manager: manager,
        keyPath: keyPath,
        encryptionKey: encryptionKey,
    }
    
    // 1. 先尝试从本地加载密钥
    if err := client.loadLocalKey(); err == nil {
        // 本地有密钥，使用本地密钥验证
        logger.Infof(nil, "[License Client] Loaded license key from local file")
        return client, nil
    }
    
    // 2. 本地没有密钥，从 NATS 获取
    if err := client.fetchKeyFromNATS(); err != nil {
        // 无法获取密钥，返回 nil（视为社区版）
        logger.Warnf(nil, "[License Client] Failed to fetch license key from NATS: %v, using community edition", err)
        conn.Close()
        return nil, nil
    }
    
    // 3. 订阅主题（用于接收密钥更新）
    sub, err := conn.Subscribe("license.key", client.handleKeyMessage)
    if err != nil {
        conn.Close()
        return nil, err
    }
    
    client.subscription = sub
    
    return client, nil
}

// loadLocalKey 从本地加载密钥
func (c *Client) loadLocalKey() error {
    data, err := os.ReadFile(c.keyPath)
    if err != nil {
        return err
    }
    
    // 解密并验证
    return c.setLicenseFromEncrypted(data)
}

// fetchKeyFromNATS 从 NATS 获取密钥
func (c *Client) fetchKeyFromNATS() error {
    // 订阅主题，等待密钥消息
    sub, err := c.natsConn.SubscribeSync("license.key")
    if err != nil {
        return err
    }
    defer sub.Unsubscribe()
    
    // 等待消息（超时10秒）
    msg, err := sub.NextMsg(10 * time.Second)
    if err != nil {
        return err
    }
    
    // 处理密钥消息
    return c.handleKeyMessage(msg)
}

// handleKeyMessage 处理密钥消息
func (c *Client) handleKeyMessage(msg *nats.Msg) {
    var keyMsg LicenseKeyMessage
    if err := json.Unmarshal(msg.Data, &keyMsg); err != nil {
        return
    }
    
    // 解码加密的 License
    encrypted, err := base64.StdEncoding.DecodeString(keyMsg.EncryptedLicense)
    if err != nil {
        return
    }
    
    // 解密并设置 License
    if err := c.setLicenseFromEncrypted(encrypted); err != nil {
        logger.Errorf(nil, "[License Client] Failed to decrypt license: %v", err)
        return
    }
    
    // 保存到本地
    if err := os.WriteFile(c.keyPath, encrypted, 0600); err != nil {
        logger.Warnf(nil, "[License Client] Failed to save license key to local: %v", err)
    } else {
        logger.Infof(nil, "[License Client] Saved license key to local file")
    }
}

// setLicenseFromEncrypted 从加密数据设置 License
func (c *Client) setLicenseFromEncrypted(encrypted []byte) error {
    // 解密
    decrypted, err := c.decryptLicense(encrypted)
    if err != nil {
        return err
    }
    
    // 解析 License
    var lic License
    if err := json.Unmarshal(decrypted, &lic); err != nil {
        return err
    }
    
    // 验证 License（签名、过期等）
    if err := c.manager.validateLicense(&lic); err != nil {
        return err
    }
    
    // 设置 License
    c.manager.setLicense(&lic)
    
    return nil
}

// decryptLicense 解密 License
func (c *Client) decryptLicense(encrypted []byte) ([]byte, error) {
    block, err := aes.NewCipher(c.encryptionKey)
    if err != nil {
        return nil, err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    
    nonceSize := gcm.NonceSize()
    if len(encrypted) < nonceSize {
        return nil, fmt.Errorf("ciphertext too short")
    }
    
    nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, err
    }
    
    return plaintext, nil
}
```

---

## 🔐 安全设计

### 1. 密钥加密

**方案A：对称加密（AES-256-GCM）**（推荐）

- ✅ License 服务和各实例共享同一个加密密钥
- ✅ 密钥通过配置文件或环境变量配置
- ✅ 加密后的密钥通过 NATS 分发

**方案B：非对称加密（RSA）**

- ✅ License 服务使用私钥签名
- ✅ 各实例使用公钥验证
- ✅ 公钥可以嵌入到二进制文件中

---

### 2. 防止模拟 NATS 消息

**防护措施**：
- ✅ **密钥加密**：即使有人模拟 NATS 消息，因为没有加密密钥，无法解密
- ✅ **本地验证**：各实例获取密钥后，使用本地密钥自行验证（RSA 签名验证）
- ✅ **密钥文件权限**：本地密钥文件设置严格权限（0600）

---

### 3. 密钥存储

**本地密钥文件路径**：
- 优先级1：环境变量 `LICENSE_KEY_PATH`
- 优先级2：`~/.ai-agent-os/license.key`
- 优先级3：`./license.key`

**文件权限**：`0600`（仅所有者可读写）

---

## 🎯 优势分析

### ✅ 优势

1. **更安全**
   - 即使有人模拟 NATS 消息，因为没有加密密钥，无法解密
   - 各实例使用本地密钥自行验证，不依赖 NATS 状态

2. **无单点问题**
   - 获取到密钥后，后续验证不依赖 NATS
   - 即使 NATS 挂了，各实例仍可使用本地密钥验证

3. **简化实现**
   - 各实例只需在启动时获取一次密钥
   - 后续验证完全本地化，无需持续连接 NATS

4. **适合集群**
   - 新实例启动时自动从 NATS 获取密钥
   - 无需共享存储，无需手动配置

---

### ⚠️ 注意事项

1. **加密密钥管理**
   - License 服务和各实例需要共享同一个加密密钥
   - 密钥可以通过配置文件或环境变量配置
   - 建议使用密钥管理服务（如 Vault）

2. **密钥更新**
   - License 文件更新时，License 服务会发布新的密钥
   - 各实例收到新密钥后，会更新本地密钥文件

3. **密钥泄露**
   - 如果密钥泄露，需要重新生成密钥并重新分发
   - 建议定期轮换密钥

---

## 📋 实施 Checklist

### License 服务

- [ ] 创建 `core/license-server` 目录结构
- [ ] 实现 License 服务（读取、验证、加密）
- [ ] 实现 NATS 发布逻辑（密钥分发）
- [ ] 实现定期任务（每5分钟发布一次）

### License Client

- [ ] 实现 License Client（订阅 NATS 主题）
- [ ] 实现密钥获取逻辑（从 NATS 或本地）
- [ ] 实现密钥保存逻辑（保存到本地文件）
- [ ] 实现密钥解密和验证逻辑

### 各服务集成

- [ ] app-server 集成 License Client
- [ ] agent-server 集成 License Client
- [ ] app-runtime 集成 License Client
- [ ] 其他服务集成 License Client

---

## 🎯 总结

### 核心设计

1. **NATS 只用于密钥分发**：License 服务通过 NATS 分发加密的 License 密钥
2. **本地保存**：各实例获取密钥后保存到本地
3. **自行验证**：各实例使用本地密钥自行验证，不依赖 NATS
4. **防止模拟**：即使有人模拟 NATS 消息，因为没有加密密钥，无法解密

### 关键优势

- ✅ **更安全**：即使有人模拟 NATS 消息，因为没有加密密钥，无法破解
- ✅ **无单点问题**：获取到密钥后，后续验证不依赖 NATS
- ✅ **简化实现**：各实例只需在启动时获取一次密钥
- ✅ **适合集群**：新实例启动时自动从 NATS 获取密钥

---

## 📞 参考

- [企业部署设计方案](./ENTERPRISE_DEPLOYMENT_DESIGN.md)
- [设计方案总结](./DESIGN_SUMMARY.md)
