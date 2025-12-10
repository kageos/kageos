# LLMs - 大模型调用库

这是一个抽象的大模型调用库，支持多种AI提供商，提供统一的接口进行AI对话。

## 特性

- **统一接口**：所有AI提供商使用相同的接口
- **多提供商支持**：支持DeepSeek、千问、豆包、Kimi等
- **流式支持**：支持实时流式响应，提升用户体验
- **配置管理**：支持配置文件管理API密钥
- **错误处理**：完善的错误处理机制
- **使用统计**：支持token使用统计
- **易于扩展**：简单的接口设计，易于添加新提供商

## 支持的提供商

| 提供商 | 状态 | 说明 |
|--------|------|------|
| DeepSeek | ✅ 已实现 | 代码生成能力强，推荐使用 |
| 千问 | ✅ 已实现 | 中文理解好，价格便宜 |
| 豆包 | ✅ 已实现 | 字节跳动出品，价格便宜 |
| Kimi | ✅ 已实现 | 月之暗面出品，长文本处理强 |
| Claude | ✅ 已实现 | Anthropic出品，推理能力强 |
| Gemini | ✅ 已实现 | Google出品，多模态支持 |
| GLM | ✅ 已实现 | 智谱AI出品，GLM-4.5系列，思考模式 |

## 快速开始

### 1. 基本使用

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/yunhanshu-net/function-go/pkg/llms"
)

func main() {
    // 创建DeepSeek客户端
    client, err := llms.NewLLMClient(llms.ProviderDeepSeek, "your-api-key")
    if err != nil {
        log.Fatal(err)
    }

    // 构造对话请求
    req := &llms.ChatRequest{
        Messages: []llms.Message{
            {Role: "system", Content: "你是function-go框架专家"},
            {Role: "user", Content: "请帮我创建一个图书管理系统"},
        },
        MaxTokens:  4000,
        Temperature: 0.7,
    }

    // 调用AI
    resp, err := client.Chat(context.Background(), req)
    if err != nil {
        log.Fatal(err)
    }

    if resp.Error != "" {
        fmt.Printf("错误: %s\n", resp.Error)
        return
    }

    fmt.Printf("AI回答: %s\n", resp.Content)
}
```

### 2. 流式聊天

```go
// 创建客户端
client, err := llms.NewLLMClient(llms.ProviderGLM, "your-api-key")
if err != nil {
    log.Fatal(err)
}

// 构造对话请求
req := &llms.ChatRequest{
    Messages: []llms.Message{
        {Role: "user", Content: "请详细介绍一下人工智能"},
    },
    MaxTokens:  2000,
    Temperature: 0.7,
}

// 开始流式聊天
stream, err := client.ChatStream(context.Background(), req)
if err != nil {
    log.Fatal(err)
}

// 处理流式响应
for chunk := range stream {
    if chunk.Error != "" {
        fmt.Printf("错误: %s\n", chunk.Error)
        break
    }
    
    if chunk.Content != "" {
        // 实时打印内容
        fmt.Print(chunk.Content)
    }
    
    if chunk.Done {
        fmt.Println("\n\n聊天完成")
        if chunk.Usage != nil {
            fmt.Printf("Token使用: %d\n", chunk.Usage.TotalTokens)
        }
        break
    }
}
```

### 3. 使用配置文件

```go
// 加载配置文件
err := llms.LoadConfig("config.json")
if err != nil {
    log.Fatal(err)
}

// 创建默认客户端
client, err := llms.CreateDefaultClient()
if err != nil {
    log.Fatal(err)
}

// 使用客户端
resp, err := client.Chat(context.Background(), req)
```

### 4. 多提供商使用

```go
// 支持的提供商列表
providers := llms.GetSupportedProviders()
for _, provider := range providers {
    fmt.Printf("提供商: %s (%s)\n", 
        provider, llms.GetProviderDisplayName(provider))
}

// 创建指定提供商客户端
client, err := llms.NewLLMClient(llms.ProviderQwen, "your-qwen-api-key")
```

## 配置说明

### 配置文件格式

```json
{
  "providers": {
    "deepseek": {
      "api_key": "your-deepseek-api-key-here",
      "base_url": "https://api.deepseek.com/v1/chat/completions",
      "timeout": 60
    },
    "qwen": {
      "api_key": "your-qwen-api-key-here",
      "base_url": "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation",
      "timeout": 60
    }
  },
  "default": "deepseek"
}
```

### 环境变量支持

你也可以通过环境变量设置API密钥：

```bash
export DEEPSEEK_API_KEY="your-api-key"
export QWEN_API_KEY="your-api-key"
export DOUBAO_API_KEY="your-api-key"
export KIMI_API_KEY="your-api-key"
export CLAUDE_API_KEY="your-api-key"
export GEMINI_API_KEY="your-api-key"
export GLM_API_KEY="your-api-key"
```

## API参考

### 核心接口

#### LLMClient

```go
type LLMClient interface {
    // Chat 核心方法：根据对话列表返回AI回答
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    
    // ChatStream 流式聊天方法：返回流式响应通道
    ChatStream(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error)
    
    // GetModelName 获取模型名称
    GetModelName() string
    
    // GetProvider 获取提供商名称
    GetProvider() string
}
```

#### ChatRequest

```go
type ChatRequest struct {
    Messages   []Message `json:"messages"`   // 对话历史
    Model     string    `json:"model"`       // 模型名称（可选）
    MaxTokens int       `json:"max_tokens"`  // 最大token数（可选）
    Temperature float64 `json:"temperature"` // 温度参数（可选）
}
```

#### ChatResponse

```go
type ChatResponse struct {
    Content string `json:"content"` // AI回答内容
    Error   string `json:"error"`   // 错误信息（如果有）
    Usage   *Usage `json:"usage"`   // 使用统计（可选）
}
```

#### StreamChunk

```go
type StreamChunk struct {
    Content string `json:"content"`           // 流式内容片段
    Done    bool   `json:"done"`              // 是否完成
    Error   string `json:"error,omitempty"`   // 错误信息（如果有）
    Usage   *Usage `json:"usage,omitempty"`   // 使用统计（完成时提供）
}
```

### 工厂函数

```go
// 创建LLM客户端
func NewLLMClient(provider Provider, apiKey string) (LLMClient, error)

// 获取支持的提供商列表
func GetSupportedProviders() []Provider

// 获取提供商显示名称
func GetProviderDisplayName(provider Provider) string
```

### 配置管理

```go
// 加载配置文件
func LoadConfig(configPath string) error

// 创建默认客户端
func CreateDefaultClient() (LLMClient, error)

// 从配置创建客户端
func CreateClientFromConfig(provider Provider) (LLMClient, error)
```

## 扩展新提供商

要添加新的AI提供商，只需要：

1. 实现`LLMClient`接口
2. 在`factory.go`中添加新的case
3. 在`Provider`常量中添加新值

示例：

```go
// 新提供商实现
type NewProviderClient struct {
    APIKey string
}

func (n *NewProviderClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
    // 实现具体的API调用逻辑
}

func (n *NewProviderClient) GetModelName() string {
    return "new-provider-model"
}

func (n *NewProviderClient) GetProvider() string {
    return "NewProvider"
}

// 在factory.go中添加
case ProviderNewProvider:
    return NewNewProviderClient(apiKey), nil
```

## GLM 特殊功能

### 思考模式

GLM-4.5 系列支持深度思考模式，可以通过 `thinking.type` 参数控制：

```go
// 创建GLM客户端
client, err := llms.NewLLMClient(llms.ProviderGLM, "your-glm-api-key")
if err != nil {
    log.Fatal(err)
}

// 设置模型
glmClient := client.(*llms.GLMClient)
glmClient.SetModel("glm-4.5") // 或 glm-4.5-air, glm-4.5-x 等

// 使用思考模式
resp, err := glmClient.ChatWithThinking(ctx, req, true) // 启用思考模式
```

### 支持的模型

GLM-4.5 系列提供多个模型选择：

- `glm-4.5`: 最强大的推理模型，3550亿参数
- `glm-4.5-air`: 高性价比轻量级强性能
- `glm-4.5-x`: 高性能强推理极速响应
- `glm-4.5-airx`: 轻量级强性能极速响应
- `glm-4.5-flash`: 免费高效多功能

```go
// 获取支持的模型列表
models := glmClient.GetSupportedModels()
for _, model := range models {
    fmt.Printf("支持模型: %s\n", model)
}

// 检查思考模式支持
if glmClient.IsThinkingEnabled() {
    fmt.Println("当前模型支持思考模式")
}
```

## 流式支持

### 支持的提供商

| 提供商 | 流式支持 | 说明 |
|--------|----------|------|
| GLM | ✅ 完全支持 | 支持思考模式流式输出 |
| DeepSeek | ✅ 完全支持 | 高性能流式响应 |
| 千问 | ✅ 完全支持 | 阿里云流式API |
| Claude | ⚠️ 暂不支持 | 返回降级提示 |
| Kimi | ⚠️ 暂不支持 | 返回降级提示 |
| 豆包 | ⚠️ 暂不支持 | 返回降级提示 |
| Gemini | ⚠️ 暂不支持 | 返回降级提示 |
| Qwen3Coder | ⚠️ 暂不支持 | 返回降级提示 |

### 流式使用场景

1. **实时对话**：用户可以看到AI逐步生成回答
2. **长文本生成**：避免长时间等待，提升用户体验
3. **Web应用**：支持Server-Sent Events (SSE)
4. **调试分析**：实时查看AI的思考过程

### 性能优势

- **首字响应时间**：通常比非流式快50-80%
- **用户体验**：实时反馈，避免长时间等待
- **资源利用**：可以提前开始处理部分响应

## 最佳实践

### 1. 错误处理

```go
resp, err := client.Chat(ctx, req)
if err != nil {
    // 处理网络错误、超时等
    log.Printf("调用失败: %v", err)
    return
}

if resp.Error != "" {
    // 处理API返回的错误
    log.Printf("API错误: %s", resp.Error)
    return
}
```

### 2. 超时控制

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

resp, err := client.Chat(ctx, req)
```

### 3. 重试机制

```go
var resp *ChatResponse
var err error

for i := 0; i < 3; i++ {
    resp, err = client.Chat(ctx, req)
    if err == nil && resp.Error == "" {
        break
    }
    time.Sleep(time.Duration(i+1) * time.Second)
}
```

## 许可证

MIT License





