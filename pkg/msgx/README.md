# msgx 约定

`pkg/msgx` 负责仓库内最基础的 NATS request-reply 原语。

它只解决 3 件事：

- 构建带 trace / request-user 透传的 JSON 请求消息
- 统一解析 `code/msg` 响应头
- 统一做 JSON 解码和成功/失败回包

`msgx` 不负责业务主题命名，也不负责某条业务链上的专属 header 语义。

## 角色边界

- `pkg/subjects`：定义主题真值
- `pkg/msgx`：定义最基础的消息收发原语
- 领域 transport：定义某条链路自己的 subject / header / request 语义

例如：

- `pkg/appinvoke` 定义 app invoke 链的 subject 和 header
- 其他跨服务链路可以在自己的领域 transport 包里封装专属语义

## 当前推荐 API

- `BuildJSONRequest`
- `RequestJSON`
- `RespondJSONSuccess`
- `RespondJSONFailure`
- `DecodeJSON`

## 使用规则

1. 纯 request-reply 场景优先直接复用 `msgx`
2. 如果某条链路有自己的 header 或 subject 约定，先建领域 transport，再在内部调用 `msgx`
3. handler 层负责 `DecodeJSON / RespondJSONSuccess / RespondJSONFailure`
4. publisher / query client 层负责 `BuildJSONRequest / RequestJSON`

## 命名约定

对外 API 使用 `JSON` 命名，避免同时维护多套等价包装。
