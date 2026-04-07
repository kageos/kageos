# Control Service 集成实现总结

这份文档描述当前仓库中的 License 集成方式。

## 已完成的接入点

### 配置结构

位置：`pkg/config/control_service.go`

各服务通过 `ControlServiceClientConfig` 配置 License Client：

```go
type ControlServiceClientConfig struct {
    Enabled       bool   `mapstructure:"enabled"`
    NatsURL       string `mapstructure:"nats_url"`
    EncryptionKey string `mapstructure:"encryption_key"`
    KeyPath       string `mapstructure:"key_path"`
}
```

### 已接入服务

- `app-server`
- `agent-server`

两边的接入流程一致：

1. 初始化主 NATS 连接
2. 初始化 License Client
3. 启动其他业务服务

## 当前 NATS 主题

- `control.v1.query.license-key.get`
  服务启动或刷新时主动拉取 License
- `license.v1.event.key.updated`
  Control Service 主动推送最新 License
- `license.v1.event.key.refresh`
  Control Service 通知各服务重新拉取或停用 License

## 工作流程

### 服务启动

```text
1. 初始化数据库
2. 初始化 NATS 连接
3. 初始化 License Client
4. License Client 先尝试读取本地 license.key
5. 本地没有可用数据时，发送 control.v1.query.license-key.get
6. Control Service 返回加密 License
7. 客户端解密、激活并写回本地
8. 订阅 license.v1.event.key.updated
9. 订阅 license.v1.event.key.refresh
```

### License 激活

```text
service startup
  -> load ~/.ai-agent-os/license.key
  -> if missing: query control.v1.query.license-key.get
  -> control-service responds with encrypted license
  -> client decrypts and stores local file
```

### License 更新或停用

```text
control-service updates or deactivates license
  -> publish license.v1.event.key.refresh
  -> services receive refresh event
  -> services re-query control.v1.query.license-key.get
  -> empty response means fallback to community edition
```

### 主动推送

```text
control-service activates a new license
  -> publish license.v1.event.key.updated
  -> subscribed services refresh memory and local file directly
```

## 配置示例

### 使用主 NATS 连接

```yaml
nats:
  url: "nats://127.0.0.1:4223"

control_service:
  enabled: true
  nats_url: ""
  encryption_key: "ai-agent-os-license-key-32bytes!!"
  key_path: ""
```

### 使用独立 NATS 连接

```yaml
nats:
  url: "nats://127.0.0.1:4223"

control_service:
  enabled: true
  nats_url: "nats://127.0.0.1:4224"
  encryption_key: "ai-agent-os-license-key-32bytes!!"
  key_path: ""
```

## 说明

- 加密密钥必须与 Control Service 保持一致，长度固定 32 字节
- 主题命名规范统一见 [README.md](/Users/beiluo/Documents/work/code/gitee.com/ai-agent-os/pkg/subjects/README.md)
