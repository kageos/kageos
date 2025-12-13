# 系统控制服务设计方案（Control Service）

## 🎯 服务定位

**轻量级的系统控制服务，承担 License 管理和系统级控制职责**

- ✅ **核心职责**：License 管理（密钥分发）
- ✅ **扩展职责**：系统控制指令（下机、重启、维护模式等）
- ✅ **轻量级**：服务崩溃不影响其他服务进程
- ✅ **非关键路径**：其他服务可以独立运行，不依赖此服务

---

## 🏗️ 服务命名

### 推荐名称

1. **Control Service**（控制服务）⭐⭐⭐⭐⭐
   - ✅ 简洁明了
   - ✅ 符合轻量级职责
   - ✅ 易于理解

2. **System Coordinator**（系统协调器）⭐⭐⭐⭐
   - ✅ 更准确，强调协调作用
   - ✅ 体现系统级职责

3. **Management Service**（管理服务）⭐⭐⭐
   - ✅ 通用，但可能过于宽泛

**最终推荐**：**Control Service**（控制服务）

---

## 📋 服务职责

### 1. License 管理（核心职责）⭐⭐⭐⭐⭐

**功能**：
- ✅ 读取 License 文件
- ✅ 验证 License（签名、过期、部署ID等）
- ✅ 加密 License 密钥
- ✅ 通过 NATS 分发 License 密钥（主题：`control.license.key`）

**特点**：
- ✅ 定期发布密钥（每5分钟）
- ✅ 确保新实例能获取密钥

---

### 2. 系统控制指令（扩展职责）⭐⭐⭐⭐

**功能**：
- ✅ **下机指令**：优雅关闭所有服务
- ✅ **重启指令**：重启所有服务
- ✅ **维护模式**：进入/退出维护模式
- ✅ **配置更新**：通知配置更新
- ✅ **功能开关**：启用/禁用某些功能

**实现方式**：
- ✅ 通过 NATS 发布控制指令（主题：`control.command`）
- ✅ 各服务订阅并执行相应操作

**控制指令格式**：

```go
// pkg/control/message.go

// ControlCommand 控制指令
type ControlCommand struct {
    // 指令类型
    Type string `json:"type"` // "shutdown" | "restart" | "maintenance" | "config_update" | "feature_toggle"
    
    // 指令参数
    Params map[string]interface{} `json:"params,omitempty"`
    
    // 目标服务（空表示所有服务）
    TargetServices []string `json:"target_services,omitempty"` // ["app-server", "agent-server", ...]
    
    // 时间戳
    Timestamp int64 `json:"timestamp"`
}

// 指令类型常量
const (
    CommandTypeShutdown      = "shutdown"       // 下机
    CommandTypeRestart        = "restart"        // 重启
    CommandTypeMaintenance    = "maintenance"    // 维护模式
    CommandTypeConfigUpdate   = "config_update"  // 配置更新
    CommandTypeFeatureToggle  = "feature_toggle" // 功能开关
)
```

**示例指令**：

```json
// 下机指令
{
  "type": "shutdown",
  "params": {
    "graceful": true,
    "timeout": 30
  },
  "target_services": [],
  "timestamp": 1234567890
}

// 维护模式
{
  "type": "maintenance",
  "params": {
    "enabled": true,
    "message": "系统维护中，预计30分钟后恢复"
  },
  "target_services": [],
  "timestamp": 1234567890
}
```

---

### 3. 系统通知/公告（扩展职责）⭐⭐⭐

**功能**：
- ✅ **系统维护通知**：通知系统维护时间
- ✅ **重要消息**：发布重要系统消息
- ✅ **版本更新通知**：通知新版本发布

**实现方式**：
- ✅ 通过 NATS 发布通知（主题：`control.notification`）

**通知格式**：

```go
// pkg/control/message.go

// Notification 系统通知
type Notification struct {
    // 通知类型
    Type string `json:"type"` // "maintenance" | "important" | "version_update"
    
    // 通知内容
    Title   string `json:"title"`
    Message string `json:"message"`
    Level   string `json:"level"` // "info" | "warning" | "error"
    
    // 时间戳
    Timestamp int64 `json:"timestamp"`
}
```

---

### 4. 配置分发（可选职责）⭐⭐

**功能**：
- ✅ **集中配置分发**：某些配置的集中分发
- ✅ **配置热更新**：通知配置更新

**实现方式**：
- ✅ 通过 NATS 发布配置（主题：`control.config`）

**注意**：
- ⚠️ 此功能需要谨慎设计，避免与现有配置系统冲突
- ⚠️ 建议仅用于轻量级配置，不用于核心配置

---

### 5. 健康检查协调（可选职责）⭐⭐

**功能**：
- ✅ **健康检查协调**：协调各服务的健康检查
- ✅ **服务状态监控**：监控各服务状态

**实现方式**：
- ✅ 各服务定期上报健康状态（主题：`control.health.<service-name>`）
- ✅ Control Service 汇总并监控

**注意**：
- ⚠️ 此功能需要谨慎设计，避免与现有监控系统冲突
- ⚠️ 建议仅用于轻量级监控，不用于核心监控

---

## 🏗️ 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────┐
│              Control Service（控制服务）                   │
│                                                         │
│  ┌──────────────────────────────────────────────┐      │
│  │  License Manager                             │      │
│  │  - 读取 License 文件                         │      │
│  │  - 验证 License                             │      │
│  │  - 加密并分发密钥                           │      │
│  └──────────────────────────────────────────────┘      │
│                                                         │
│  ┌──────────────────────────────────────────────┐      │
│  │  Control Command Handler                     │      │
│  │  - 接收控制指令（HTTP API）                  │      │
│  │  - 发布控制指令到 NATS                      │      │
│  └──────────────────────────────────────────────┘      │
│                                                         │
│  ┌──────────────────────────────────────────────┐      │
│  │  Notification Manager                        │      │
│  │  - 发布系统通知/公告                         │      │
│  └──────────────────────────────────────────────┘      │
│                                                         │
│  ┌──────────────────────────────────────────────┐      │
│  │  HTTP API Server                             │      │
│  │  - 接收控制指令（下机、重启等）              │      │
│  │  - 发布通知                                  │      │
│  └──────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────┘
                    │
                    │ NATS 发布
                    ↓
┌─────────────────────────────────────────────────────────┐
│         NATS 消息中间件                                   │
│                                                         │
│  主题：                                                  │
│  - control.license.key    (License 密钥)                │
│  - control.command        (控制指令)                     │
│  - control.notification   (系统通知)                     │
└─────────────────────────────────────────────────────────┘
                    │
                    │ NATS 订阅
                    ↓
┌─────────────────────────────────────────────────────────┐
│              所有服务实例（订阅并执行）                    │
│                                                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│  │app-server│  │agent-server│ │app-runtime│            │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘            │
│       │             │             │                   │
│       └─────────────┴─────────────┘                   │
│                    │                                   │
│                    │ 订阅主题                          │
│                    │ - control.license.key             │
│                    │ - control.command                 │
│                    │ - control.notification           │
│                    ↓                                   │
│  ┌──────────────────────────────────────────────┐     │
│  │  Control Client                               │     │
│  │  - 获取 License 密钥                          │     │
│  │  - 接收控制指令并执行                         │     │
│  │  - 接收系统通知                               │     │
│  └──────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────┘
```

---

## 🔄 工作流程

### 1. 下机指令流程

```
管理员在 Control Service 点击下机
  ↓
Control Service 接收 HTTP 请求
  ↓
构建下机指令
  ↓
发布到 NATS（主题：control.command）
  ↓
所有服务订阅并收到指令
  ↓
各服务执行优雅关闭
  ├─ 停止接收新请求
  ├─ 等待现有请求完成
  └─ 关闭服务
```

---

### 2. 维护模式流程

```
管理员在 Control Service 启用维护模式
  ↓
Control Service 接收 HTTP 请求
  ↓
构建维护模式指令
  ↓
发布到 NATS（主题：control.command）
  ↓
所有服务订阅并收到指令
  ↓
各服务进入维护模式
  ├─ 返回维护提示
  └─ 拒绝新请求（可选）
```

---

## 💻 实现方案

### 1. Control Service 结构

```go
// core/control-service/server/server.go

// Server Control Service 服务器
type Server struct {
    // 配置
    cfg *config.ControlServiceConfig
    
    // 核心组件
    natsConn   *nats.Conn
    httpServer *gin.Engine
    
    // 管理器
    licenseManager *license.Manager
    commandHandler *CommandHandler
    notificationManager *NotificationManager
    
    // 上下文
    ctx context.Context
}

// NewServer 创建 Control Service 服务器
func NewServer(cfg *config.ControlServiceConfig) (*Server, error) {
    s := &Server{
        cfg: cfg,
        ctx: context.Background(),
    }
    
    // 初始化各个组件
    if err := s.initNATS(); err != nil {
        return nil, err
    }
    
    if err := s.initLicenseManager(); err != nil {
        return nil, err
    }
    
    if err := s.initCommandHandler(); err != nil {
        return nil, err
    }
    
    if err := s.initNotificationManager(); err != nil {
        return nil, err
    }
    
    if err := s.initRouter(); err != nil {
        return nil, err
    }
    
    return s, nil
}
```

---

### 2. HTTP API 设计

```go
// core/control-service/api/v1/control.go

// ControlAPI 控制 API
type ControlAPI struct {
    server *server.Server
}

// Shutdown 下机指令
// POST /api/v1/control/shutdown
func (api *ControlAPI) Shutdown(c *gin.Context) {
    var req struct {
        Graceful bool `json:"graceful"` // 是否优雅关闭
        Timeout  int  `json:"timeout"`  // 超时时间（秒）
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // 构建下机指令
    command := &control.ControlCommand{
        Type: control.CommandTypeShutdown,
        Params: map[string]interface{}{
            "graceful": req.Graceful,
            "timeout":  req.Timeout,
        },
        Timestamp: time.Now().Unix(),
    }
    
    // 发布到 NATS
    if err := api.server.PublishCommand(command); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"message": "Shutdown command sent"})
}

// Maintenance 维护模式
// POST /api/v1/control/maintenance
func (api *ControlAPI) Maintenance(c *gin.Context) {
    var req struct {
        Enabled bool   `json:"enabled"` // 是否启用维护模式
        Message string `json:"message"` // 维护提示消息
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // 构建维护模式指令
    command := &control.ControlCommand{
        Type: control.CommandTypeMaintenance,
        Params: map[string]interface{}{
            "enabled": req.Enabled,
            "message": req.Message,
        },
        Timestamp: time.Now().Unix(),
    }
    
    // 发布到 NATS
    if err := api.server.PublishCommand(command); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"message": "Maintenance mode command sent"})
}
```

---

### 3. Control Client（各服务实例）

```go
// pkg/control/client.go

// Client Control Service 客户端
type Client struct {
    natsConn     *nats.Conn
    subscriptions []*nats.Subscription
    commandHandler func(*ControlCommand) error
    mu           sync.RWMutex
}

// NewClient 创建 Control Service 客户端
func NewClient(natsURL string) (*Client, error) {
    conn, err := nats.Connect(natsURL)
    if err != nil {
        // 无法连接 NATS，返回 nil（不影响服务运行）
        return nil, nil
    }
    
    client := &Client{
        natsConn: conn,
    }
    
    // 订阅控制指令
    sub, err := conn.Subscribe("control.command", client.handleCommand)
    if err != nil {
        conn.Close()
        return nil, err
    }
    client.subscriptions = append(client.subscriptions, sub)
    
    // 订阅系统通知
    sub, err = conn.Subscribe("control.notification", client.handleNotification)
    if err != nil {
        conn.Close()
        return nil, err
    }
    client.subscriptions = append(client.subscriptions, sub)
    
    return client, nil
}

// handleCommand 处理控制指令
func (c *Client) handleCommand(msg *nats.Msg) {
    var command ControlCommand
    if err := json.Unmarshal(msg.Data, &command); err != nil {
        return
    }
    
    switch command.Type {
    case CommandTypeShutdown:
        c.handleShutdown(&command)
    case CommandTypeRestart:
        c.handleRestart(&command)
    case CommandTypeMaintenance:
        c.handleMaintenance(&command)
    // ...
    }
}

// handleShutdown 处理下机指令
func (c *Client) handleShutdown(command *ControlCommand) {
    graceful := true
    if v, ok := command.Params["graceful"].(bool); ok {
        graceful = v
    }
    
    timeout := 30
    if v, ok := command.Params["timeout"].(float64); ok {
        timeout = int(v)
    }
    
    // 执行优雅关闭
    if graceful {
        // 优雅关闭逻辑
        // ...
    } else {
        // 立即关闭
        os.Exit(0)
    }
}
```

---

## 🎯 服务特点

### 1. 轻量级

- ✅ **简单职责**：只承担轻量级职责
- ✅ **无状态**：服务本身无状态，易于重启
- ✅ **独立运行**：不依赖其他服务

---

### 2. 非关键路径

- ✅ **服务崩溃不影响其他服务**：其他服务可以独立运行
- ✅ **容错机制**：各服务可以检测 Control Service 是否可用
- ✅ **降级策略**：如果 Control Service 不可用，各服务降级到默认行为

---

### 3. 易于扩展

- ✅ **模块化设计**：各职责模块化，易于扩展
- ✅ **插件化**：可以轻松添加新的控制指令
- ✅ **配置化**：通过配置文件控制功能开关

---

## 📋 实施 Checklist

### 阶段一：基础功能

- [ ] 创建 `core/control-service` 目录结构
- [ ] 实现 License 管理（密钥分发）
- [ ] 实现 HTTP API 服务器
- [ ] 实现 NATS 发布逻辑

### 阶段二：控制指令

- [ ] 实现下机指令
- [ ] 实现重启指令
- [ ] 实现维护模式
- [ ] 实现 Control Client（各服务实例）

### 阶段三：扩展功能

- [ ] 实现系统通知/公告
- [ ] 实现配置分发（可选）
- [ ] 实现健康检查协调（可选）

---

## 🎯 总结

### 核心设计

1. **服务名称**：Control Service（控制服务）
2. **核心职责**：License 管理（密钥分发）
3. **扩展职责**：系统控制指令（下机、重启、维护模式等）
4. **轻量级**：服务崩溃不影响其他服务进程

### 关键优势

- ✅ **集中管理**：License 和系统控制集中管理
- ✅ **轻量级**：服务简单，易于维护
- ✅ **非关键路径**：服务崩溃不影响其他服务
- ✅ **易于扩展**：可以轻松添加新的控制指令

---

## 📞 参考

- [License 密钥分发方案](./LICENSE_DISTRIBUTION_DESIGN.md)
- [企业部署设计方案](./ENTERPRISE_DEPLOYMENT_DESIGN.md)
