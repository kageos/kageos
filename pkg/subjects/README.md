# NATS 主题规范

`pkg/subjects` 是跨服务 NATS 主题的唯一真值来源。跨服务协议不要在业务代码里直接手写字符串。

## 命名格式

统一使用点分隔结构：

`<target>.<version>.<kind>.<domain>.<action>[.<scope...>]`

约束如下：

- `target`：主题主要由谁消费或归谁负责，例如 `runtime`、`app`、`gateway`
- `version`：当前固定为 `v1`
- `kind`：固定使用 `cmd`、`query`、`event`、`reply`
- 静态 token：统一使用小写 kebab-case
- 动态 scope：统一放在尾部，当前主要是 `{user}.{app}.{version}`

## 使用规则

- 固定主题使用 `const`
- `QueueSubscribe` 的 queue group 也统一放在 `pkg/subjects`
- 动态主题使用 `BuildXxxSubject()`
- 跨服务收发统一引用 `pkg/subjects`
- 单个文件内部的订阅装配直接引用常量或 builder
- 未来新增主题统一沿用这套结构，保持单一命名风格

## 当前已实现主题

### runtime / app 调用链

- `runtime.v1.cmd.app.create`
- `runtime.v1.cmd.app.update`
- `runtime.v1.cmd.app.delete`
- `runtime.v1.cmd.app.invoke.{user}.{app}.{version}`
- `app.v1.cmd.invoke.{user}.{app}.{version}`
- `app.v1.cmd.control.{user}.{app}.{version}`
- `app-server.v1.reply.app.invoke.{user}.{app}.{version}`
- `app.v1.cmd.discovery.request`
- `runtime.v1.event.lifecycle.{user}.{app}.{version}`

### gateway

- `gateway.v1.cmd.token.invalidate`
- `gateway.v1.cmd.token.remove-blacklist`

## 文档同步要求

如果主题名发生调整，需要同时更新：

- `pkg/subjects/*.go`
- 相关调用方/消费方代码
- 设计文档和 README

否则代码和排障文档会再次漂移。
