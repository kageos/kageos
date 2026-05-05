# SDK 平台 API 参考

本文档说明 SDK 代码如何调用 AgentOS 平台能力。

## ctx.APICall

平台 API 统一使用：

```go
err := ctx.APICall(method, path, reqBody, respData)
```

规则：

- `method` 使用 `http.MethodGet`、`http.MethodPost` 等。
- `path` 使用平台网关路径，例如 `/hub/api/v1/directories`。
- `reqBody` 是请求体；GET 可传 `nil`。
- `respData` 是响应 `data` 对应结构体指针。
- SDK 会带上 token、trace、request_user、department、client_source、source_type、source_ref。
- 平台 Web API 负责权限、审计和数据边界。

禁止：

- 裸写 HTTP client。
- 硬编码 token。
- 伪造 request_user。
- 直连平台数据库。
- 绕过 app-server 权限检查。

## ctx.SendMessage

业务函数发送消息使用：

```go
err := ctx.SendMessage(&app.SendMessageOpts{
    ToUsers:     "zhangsan,lisi",
    Title:       "通知标题",
    Content:     "正文，默认 markdown",
    ContentType: "markdown",
})
```

规则：

- `ContentType` 默认 `markdown`。
- `ToUsers` 与 `user/users` 组件存储格式一致。
- `ToDepartments` 与 `department/departments` 组件存储格式一致。
- 发送人、来源路径、trace、client_source 从 ctx 生成。
- 不要自建消息表或消息通道。

## /system/openapi

`/system/openapi` 是官方平台接口工作空间。新增平台接口包装函数时：

1. 按平台领域建目录，如 `hub`、`message`、`permission`。
2. 每个目录自己隔离 helper，不建公共 utils 目录。
3. 只通过 `ctx.APICall` 调平台 Web API。
4. 副作用接口要在 skill/SOP 中明确确认和审计要求。
