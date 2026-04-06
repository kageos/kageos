# LLMs

统一的大模型调用封装，提供：

- 统一的 `Chat` / `ChatStream` 接口
- 多 provider 工厂创建
- 配置文件与环境变量两种初始化方式
- 工具调用字段透传
- GLM 思考模式等 provider-specific 扩展

当前代码仓库的正确 import path 是：

```go
import "github.com/ai-agent-os/ai-agent-os/pkg/llms"
```

## 支持的 Provider

工厂层支持这些 provider 常量：

- `ProviderDeepSeek`
- `ProviderQwen`
- `ProviderQwen3Coder`
- `ProviderDouBao`
- `ProviderKimi`
- `ProviderClaude`
- `ProviderGemini`
- `ProviderGLM`
- `ProviderMiniMax`
- `ProviderXiaomi`

对应环境变量：

| Provider | 环境变量 |
| --- | --- |
| DeepSeek | `DEEPSEEK_API_KEY` |
| Qwen / Qwen3Coder | `QIANWEN_API_KEY` |
| DouBao | `DOUBAO_API_KEY` |
| Kimi | `KIMI_API_KEY` |
| Claude | `CLAUDE_API_KEY` |
| Gemini | `GEMINI_API_KEY` |
| GLM | `GLM_API_KEY` |
| MiniMax | `MINIMAX_API_KEY` |
| Xiaomi | `XIAOMI_API_KEY` |

## 快速开始

最常见的方式是直接从环境变量创建：

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/ai-agent-os/ai-agent-os/pkg/llms"
)

func main() {
    client, err := llms.NewLLMClientFromEnv(llms.ProviderDeepSeek)
    if err != nil {
        log.Fatal(err)
    }

    req := &llms.ChatRequest{
        Messages: []llms.Message{
            {Role: "system", Content: "你是一个简洁的技术助手"},
            {Role: "user", Content: "用三句话解释 goroutine"},
        },
        MaxTokens:   800,
        Temperature: 0.2,
    }

    resp, err := client.Chat(context.Background(), req)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Content)
}
```

也可以显式传 API Key：

```go
client, err := llms.NewLLMClient(llms.ProviderGLM, "<api-key>")
```

## 流式调用

```go
stream, err := client.ChatStream(context.Background(), req)
if err != nil {
    log.Fatal(err)
}

for chunk := range stream {
    if chunk.Error != "" {
        log.Fatal(chunk.Error)
    }
    if chunk.Content != "" {
        fmt.Print(chunk.Content)
    }
    if chunk.Done {
        break
    }
}
```

目前 `GLM`、`DeepSeek`、`Qwen`、`Kimi` 以及部分 OpenAI 兼容 provider 实现了真实流式逻辑；`Claude`、`Gemini`、`DouBao`、`Qwen3Coder` 当前会返回“不支持流式”的完成块。

## 配置文件

配置结构对应 `Config` / `ProviderConfig`：

```json
{
  "providers": {
    "deepseek": {
      "api_key": "<api-key>",
      "base_url": "https://api.deepseek.com/v1/chat/completions",
      "timeout": 300
    },
    "glm": {
      "api_key": "<api-key>",
      "timeout": 600
    }
  },
  "default": "deepseek"
}
```

用法：

```go
if err := llms.LoadConfig("llms.json"); err != nil {
    log.Fatal(err)
}

client, err := llms.CreateDefaultClient()
if err != nil {
    log.Fatal(err)
}
```

`timeout` 字段单位是秒，加载时会转换成 `time.Duration`。

## 关键类型

最核心的接口：

```go
type LLMClient interface {
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    ChatStream(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error)
    GetModelName() string
    GetProvider() string
}
```

`ChatRequest` 常用字段：

- `Messages`: 对话消息列表
- `Model`: 可选，覆盖客户端默认模型
- `MaxTokens`: 可选，覆盖默认 token 上限
- `Temperature`: 可选
- `Timeout`: 请求级超时，优先级高于客户端默认超时
- `UseThinking`: GLM 使用的思考模式开关
- `Tools` / `ToolChoice`: OpenAI-compatible provider 的工具调用参数

`ChatResponse` / `StreamChunk` 都可能带 `ToolCalls`，用于透传工具调用结果。

## 超时与默认值

- `DefaultClientOptions()` 当前默认超时是 `1200s`
- `ChatRequest.Timeout` 会覆盖客户端级超时
- 如果 provider 支持自定义 `BaseURL`，可以通过 `ClientOptions.WithBaseURL(...)` 或配置文件传入

示例：

```go
timeout := 45 * time.Second
req := &llms.ChatRequest{
    Messages: []llms.Message{
        {Role: "user", Content: "总结这段日志"},
    },
    Timeout: &timeout,
}
```

## GLM 扩展

`GLMClient` 暴露了额外能力：

```go
client, err := llms.NewLLMClientFromEnv(llms.ProviderGLM)
if err != nil {
    log.Fatal(err)
}

glmClient := client.(*llms.GLMClient)
glmClient.SetModel("glm-4.6")

resp, err := glmClient.ChatWithThinking(context.Background(), req, true)
if err != nil {
    log.Fatal(err)
}
```

当前 `GLMClient` 默认模型是 `glm-4.6`。`GetSupportedModels()` 返回当前代码里登记的模型列表；`IsThinkingEnabled()` 会根据当前模型判断是否支持思考模式。

## Concrete Client 扩展

`LLMClient` 只保证最小接口。部分具体实现还提供额外方法，例如：

- `GetSupportedModels()`
- `GetPricingInfo()`
- `SetModel(...)`
- `ChatWithThinking(...)`（仅 GLM）

如果需要这些能力，请做具体类型断言。

## 测试

默认测试集只保留离线可跑的单元测试：

```bash
go test ./pkg/llms
```

真实联网测试已经隔离到 `integration` build tag：

```bash
go test -tags=integration ./pkg/llms
```

运行前需要先配置对应 provider 的环境变量。




