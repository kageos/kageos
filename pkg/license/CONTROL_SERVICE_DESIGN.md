# Control Service 设计说明

这份设计文档描述当前使用的 License 协议。

## 目标

Control Service 负责集中管理 License，并通过 NATS 把 License 能力广播给各个服务实例。

当前设计目标只有 3 个：

1. 服务实例启动时可主动拉取 License
2. Control Service 激活新 License 后可主动推送
3. Control Service 检测到变更或停用时，可通知各实例重新拉取

## 当前主题

| 主题 | 方向 | 作用 |
| --- | --- | --- |
| `control.v1.query.license-key.get` | service -> control-service | 主动请求最新 License |
| `license.v1.event.key.updated` | control-service -> services | 主动推送最新 License |
| `license.v1.event.key.refresh` | control-service -> services | 通知重新拉取或停用 |

命名规则统一遵循 [README.md](../subjects/README.md)。

## 组件职责

### Control Service

- 加载并校验 License 文件
- 对 License 做加密封装
- 处理 `control.v1.query.license-key.get`
- 发布 `license.v1.event.key.updated`
- 发布 `license.v1.event.key.refresh`

### 各服务实例

- 启动时先尝试读取本地 License 文件
- 本地无可用数据时，通过 `control.v1.query.license-key.get` 主动请求
- 订阅 `license.v1.event.key.updated`
- 订阅 `license.v1.event.key.refresh`
- 收到 refresh 后重新向 Control Service 请求最新 License

## 数据流

### 启动拉取

```text
service instance
  -> load local license
  -> if missing: publish query to control.v1.query.license-key.get
  -> control-service responds with encrypted license
  -> client decrypts and stores local file
```

### 激活后推送

```text
control-service activates license
  -> encrypts license
  -> publishes license.v1.event.key.updated
  -> subscribed services refresh memory and local file
```

### 变更/停用刷新

```text
control-service detects update or deactivate
  -> publishes license.v1.event.key.refresh
  -> services re-query control.v1.query.license-key.get
  -> empty response means fallback to community edition
```

## 为什么同时保留 query / updated / refresh

- 只有 `query` 不够：新实例无法被动拿到最新 License 之外的主动更新
- 只有 `updated` 不够：服务启动时仍需要一个显式拉取入口
- `refresh` 和 `updated` 不重复：`updated` 携带最新内容，`refresh` 表示“你应该重新取一次当前真值”

## 失败策略

- Control Service 不可用时，客户端保留当前内存 License；首次启动且请求失败则降级为社区版
- 本地 License 文件损坏时，客户端重新走 `control.v1.query.license-key.get`
- 收到 `license.v1.event.key.refresh` 后请求返回空值，表示当前已无有效 License

## 后续扩展

如果未来要增加新的控制命令、通知或健康上报主题：

- 统一按 `pkg/subjects` 规范定义
- 先在文档里明确 target、kind、domain、action，再落代码
