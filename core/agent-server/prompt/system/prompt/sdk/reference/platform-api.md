# SDK 平台 API 参考

本文档说明 SDK 代码如何调用 AgentOS 平台能力。

## ctx.APICall

平台 API 统一使用：

```go
err := ctx.APICall(method, path, reqBody, respData)
```

规则：

- `method` 使用 `http.MethodGet`、`http.MethodPost` 等。
- `path` 使用平台网关路径，例如 `/workspace/api/v1/operate_log/table`。
- `reqBody` 是请求体；GET 可传 `nil`。
- `respData` 是响应 `data` 对应结构体指针。
- SDK 会带上 token、trace、request_user、department、client_source、source_type、source_ref。
- 平台 Web API 负责身份上下文、Table 更新日志和数据边界。

禁止：

- 裸写 HTTP client。
- 硬编码 token。
- 伪造 request_user。
- 直连平台数据库。
- 伪造平台运行上下文。
- 调用已删除的 `/scheduled_tasks`、`/scheduled_agent_tasks`、全局消息、备份控制面或企业 License 接口。

## /system/openapi

`/system/openapi` 是官方平台接口工作空间。新增平台接口包装函数时：

1. 按平台领域建目录，如 `operate_log`。
2. 每个目录自己隔离 helper，不建公共 utils 目录。
3. 只通过 `ctx.APICall` 调平台 Web API。
4. 副作用接口要在当前身份文档包中明确确认要求和运行上下文来源。
