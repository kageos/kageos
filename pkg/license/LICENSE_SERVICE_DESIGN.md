# License 服务设计方案（基于 NATS 广播）

## 🎯 核心设计

**独立的 License 服务 + NATS 广播机制**

- ✅ **独立的 License 服务**：统一管理 License
- ✅ **NATS 广播**：所有实例通过 NATS 订阅 License 状态
- ✅ **默认社区版**：如果无法连接到 License 服务，视为社区版
- ✅ **实时同步**：License 状态变更实时通知所有实例

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
│  │  - 管理 License 状态                         │      │
│  └──────────────────────────────────────────────┘      │
│                    │                                    │
│                    │ NATS 发布                          │
│                    ↓                                    │
│  ┌──────────────────────────────────────────────┐      │
│  │  NATS 主题：license.status                   │      │
│  │  - 发布 License 状态变更                     │      │
│  │  - 定期发布心跳（确认服务存活）              │      │
│  └──────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────┘
                    │
                    │ NATS 订阅
                    ↓
┌─────────────────────────────────────────────────────────┐
│              所有服务实例（订阅 License 状态）            │
│                                                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│  │app-server│  │agent-server│ │app-runtime│            │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘            │
│       │             │             │                   │
│       └─────────────┴─────────────┘                   │
│                    │                                   │
│                    │ 订阅主题：license.status          │
│                    ↓                                   │
│  ┌──────────────────────────────────────────────┐     │
│  │  License Client                               │     │
│  │  - 订阅 License 状态                         │     │
│  │  - 更新本地 License 状态                     │     │
│  │  - 如果无法连接，默认社区版                   │     │
│  └──────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────┘
```

---

## 📋 License 服务设计

### 服务职责

1. ✅ **读取 License 文件**
   - 从配置路径读取 License 文件
   - 支持环境变量指定路径

2. ✅ **验证 License**
   - RSA 签名验证
   - 过期时间检查
   - 部署标识验证（如果启用）

3. ✅ **管理 License 状态**
   - 维护当前 License 状态（有效/无效/社区版）
   - 定期检查 License 是否过期
   - 支持在线验证（如果启用）

4. ✅ **广播 License 状态**
   - 通过 NATS 发布 License 状态
   - 定期发布心跳（确认服务存活）
   - License 状态变更时立即广播

---

### License 状态消息格式

```go
// pkg/license/message.go

// LicenseStatusMessage License 状态消息
type LicenseStatusMessage struct {
    // 消息类型
    Type string `json:"type"` // "status" | "heartbeat" | "update"
    
    // License 信息
    License *License `json:"license,omitempty"` // License 对象（如果有效）
    
    // 状态信息
    IsValid    bool   `json:"is_valid"`     // License 是否有效
    IsCommunity bool  `json:"is_community"` // 是否为社区版
    Edition    string `json:"edition"`      // "community" | "enterprise"
    
    // 错误信息（如果无效）
    Error      string `json:"error,omitempty"` // 错误信息
    
    // 时间戳
    Timestamp  int64  `json:"timestamp"` // Unix 时间戳
}
```

---

### NATS 主题设计

**主题名称**：`license.status`

**消息类型**：

1. **status** - License 状态消息
   - 服务启动时发布
   - License 状态变更时发布
   - 包含完整的 License 信息

2. **heartbeat** - 心跳消息
   - 定期发布（每30秒）
   - 确认 License 服务存活
   - 如果订阅者收不到心跳，视为社区版

3. **update** - License 更新消息
   - License 文件更新时发布
   - License 激活/撤销时发布

---

## 🔌 License Client 设计（各服务实例）

### 客户端职责

1. ✅ **订阅 License 状态**
   - 订阅 NATS 主题 `license.status`
   - 接收 License 状态消息

2. ✅ **更新本地 License 状态**
   - 根据收到的消息更新本地 License 状态
   - 更新 `license.GetManager()` 的状态

3. ✅ **容错处理**
   - 如果无法连接到 NATS，默认社区版
   - 如果收不到心跳（超时），降级到社区版
   - 自动重连机制

---

### 客户端实现

```go
// pkg/license/client.go

// Client License 客户端
type Client struct {
    conn        *nats.Conn
    subscription *nats.Subscription
    manager     *Manager
    lastHeartbeat time.Time
    heartbeatTimeout time.Duration
    mu          sync.RWMutex
}

// NewClient 创建 License 客户端
func NewClient(natsURL string, manager *Manager) (*Client, error) {
    // 连接 NATS
    conn, err := nats.Connect(natsURL)
    if err != nil {
        // 无法连接 NATS，返回 nil（视为社区版）
        return nil, nil
    }
    
    client := &Client{
        conn: conn,
        manager: manager,
        heartbeatTimeout: 60 * time.Second, // 60秒超时
    }
    
    // 订阅主题
    sub, err := conn.Subscribe("license.status", client.handleMessage)
    if err != nil {
        conn.Close()
        return nil, err
    }
    
    client.subscription = sub
    
    // 启动心跳检查
    go client.startHeartbeatCheck()
    
    return client, nil
}

// handleMessage 处理 License 状态消息
func (c *Client) handleMessage(msg *nats.Msg) {
    var statusMsg LicenseStatusMessage
    if err := json.Unmarshal(msg.Data, &statusMsg); err != nil {
        return
    }
    
    switch statusMsg.Type {
    case "status", "update":
        // 更新 License 状态
        c.updateLicenseStatus(&statusMsg)
    case "heartbeat":
        // 更新心跳时间
        c.mu.Lock()
        c.lastHeartbeat = time.Now()
        c.mu.Unlock()
    }
}

// updateLicenseStatus 更新 License 状态
func (c *Client) updateLicenseStatus(msg *LicenseStatusMessage) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    if msg.IsValid && msg.License != nil {
        // License 有效，更新到 Manager
        c.manager.setLicense(msg.License)
    } else {
        // License 无效，清除 License（降级到社区版）
        c.manager.setLicense(nil)
    }
}

// startHeartbeatCheck 启动心跳检查
func (c *Client) startHeartbeatCheck() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        c.mu.RLock()
        lastHeartbeat := c.lastHeartbeat
        timeout := c.heartbeatTimeout
        c.mu.RUnlock()
        
        // 如果超过超时时间未收到心跳，降级到社区版
        if time.Since(lastHeartbeat) > timeout {
            c.manager.setLicense(nil)
        }
    }
}
```

---

## 🚀 License 服务实现

### 服务结构

```go
// core/license-server/server/server.go

// Server License 服务
type Server struct {
    manager     *license.Manager
    natsConn    *nats.Conn
    licensePath string
    ticker      *time.Ticker
    mu          sync.RWMutex
}

// NewServer 创建 License 服务
func NewServer(natsURL, licensePath string) (*Server, error) {
    // 连接 NATS
    conn, err := nats.Connect(natsURL)
    if err != nil {
        return nil, err
    }
    
    // 创建 License Manager
    manager := license.NewManager()
    
    server := &Server{
        manager: manager,
        natsConn: conn,
        licensePath: licensePath,
    }
    
    return server, nil
}

// Start 启动服务
func (s *Server) Start(ctx context.Context) error {
    // 1. 加载 License
    if err := s.loadLicense(); err != nil {
        logger.Warnf(ctx, "[License Service] Failed to load license: %v", err)
        // 即使加载失败，也要发布社区版状态
    }
    
    // 2. 发布初始状态
    s.publishStatus()
    
    // 3. 启动定期任务
    s.ticker = time.NewTicker(30 * time.Second)
    go s.startPeriodicTasks(ctx)
    
    return nil
}

// loadLicense 加载 License
func (s *Server) loadLicense() error {
    return s.manager.LoadLicense(s.licensePath)
}

// publishStatus 发布 License 状态
func (s *Server) publishStatus() {
    msg := s.buildStatusMessage()
    data, _ := json.Marshal(msg)
    s.natsConn.Publish("license.status", data)
}

// buildStatusMessage 构建状态消息
func (s *Server) buildStatusMessage() *license.LicenseStatusMessage {
    s.mu.RLock()
    lic := s.manager.GetLicense()
    s.mu.RUnlock()
    
    msg := &license.LicenseStatusMessage{
        Type: "status",
        Timestamp: time.Now().Unix(),
    }
    
    if lic != nil && lic.IsValid() {
        msg.License = lic
        msg.IsValid = true
        msg.IsCommunity = false
        msg.Edition = lic.Edition
    } else {
        msg.IsValid = false
        msg.IsCommunity = true
        msg.Edition = "community"
    }
    
    return msg
}

// startPeriodicTasks 启动定期任务
func (s *Server) startPeriodicTasks(ctx context.Context) {
    for {
        select {
        case <-s.ticker.C:
            // 1. 发布心跳
            s.publishHeartbeat()
            
            // 2. 检查 License 是否过期
            s.checkLicenseExpiry()
            
        case <-ctx.Done():
            return
        }
    }
}

// publishHeartbeat 发布心跳
func (s *Server) publishHeartbeat() {
    msg := &license.LicenseStatusMessage{
        Type: "heartbeat",
        Timestamp: time.Now().Unix(),
    }
    data, _ := json.Marshal(msg)
    s.natsConn.Publish("license.status", data)
}

// checkLicenseExpiry 检查 License 是否过期
func (s *Server) checkLicenseExpiry(ctx context.Context) {
    lic := s.manager.GetLicense()
    if lic != nil && !lic.IsValid() {
        // License 已过期，发布更新消息
        s.publishStatus()
    }
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
发布初始状态消息（status）
  ↓
启动定期任务
  ├─ 每30秒发布心跳（heartbeat）
  └─ 定期检查 License 是否过期
```

---

### 2. 服务实例启动

```
服务实例启动
  ↓
连接 NATS
  ↓
创建 License Client
  ├─ 成功：订阅 license.status 主题
  └─ 失败：默认社区版（不订阅）
  ↓
等待接收 License 状态消息
  ├─ 收到 status 消息：更新本地 License 状态
  ├─ 收到 heartbeat 消息：更新心跳时间
  └─ 超时未收到心跳：降级到社区版
```

---

### 3. License 状态变更

```
License 文件更新
  ↓
License 服务检测到变更
  ↓
重新加载 License
  ↓
验证 License
  ↓
发布更新消息（update）
  ↓
所有订阅者收到消息
  ↓
更新本地 License 状态
```

---

## 🎯 优势分析

### ✅ 优势

1. **集中管理**
   - License 服务统一管理 License
   - 所有实例通过 NATS 订阅，无需各自读取文件

2. **实时同步**
   - License 状态变更实时通知所有实例
   - 无需重启服务

3. **容错机制**
   - 如果无法连接 License 服务，默认社区版
   - 如果收不到心跳，自动降级到社区版

4. **适合集群**
   - 所有实例通过 NATS 连接，天然支持集群
   - 无需共享存储（License 文件只需在 License 服务上）

5. **简化实现**
   - 各服务实例只需订阅 NATS 主题
   - 无需各自实现 License 验证逻辑

---

### ⚠️ 注意事项

1. **License 服务高可用**
   - License 服务是单点，需要保证高可用
   - 建议使用进程管理器（systemd、supervisor）或容器编排（K8s）

2. **NATS 连接失败**
   - 如果 NATS 连接失败，所有实例降级到社区版
   - 需要确保 NATS 服务的高可用

3. **心跳超时**
   - 如果收不到心跳，自动降级到社区版
   - 心跳超时时间需要合理设置（建议60秒）

---

## 📋 实施 Checklist

### License 服务

- [ ] 创建 `core/license-server` 目录结构
- [ ] 实现 License 服务（读取、验证、广播）
- [ ] 实现 NATS 发布逻辑
- [ ] 实现定期任务（心跳、过期检查）
- [ ] 创建启动脚本或 systemd 服务

### License Client

- [ ] 实现 License Client（订阅 NATS 主题）
- [ ] 实现消息处理逻辑
- [ ] 实现心跳检查机制
- [ ] 实现容错处理（连接失败、超时）

### 各服务集成

- [ ] app-server 集成 License Client
- [ ] agent-server 集成 License Client
- [ ] app-runtime 集成 License Client
- [ ] 其他服务集成 License Client

---

## 🎯 总结

### 核心设计

1. **独立的 License 服务**：统一管理 License
2. **NATS 广播机制**：所有实例通过 NATS 订阅 License 状态
3. **默认社区版**：如果无法连接 License 服务，视为社区版
4. **实时同步**：License 状态变更实时通知所有实例

### 关键优势

- ✅ **集中管理**：License 服务统一管理 License
- ✅ **实时同步**：License 状态变更实时通知所有实例
- ✅ **容错机制**：连接失败或超时自动降级到社区版
- ✅ **适合集群**：所有实例通过 NATS 连接，天然支持集群
- ✅ **简化实现**：各服务实例只需订阅 NATS 主题

---

## 📞 参考

- [企业部署设计方案](./ENTERPRISE_DEPLOYMENT_DESIGN.md)
- [设计方案总结](./DESIGN_SUMMARY.md)
