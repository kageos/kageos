# 千问3 Coder 集成指南

## 🚀 概述

千问3 Coder 是阿里云推出的专业代码生成模型，具有强大的代码能力，可通过 API 将其集成到业务中。

## ✨ 核心特性

### 🎯 代码生成能力
- **多语言支持**: Go、Python、JavaScript、Java、C++等
- **智能补全**: 基于上下文的代码补全
- **函数调用**: 支持工具调用和文件操作
- **代码优化**: 自动优化和重构建议

### 📊 模型规格
- **上下文长度**: 1,000,000 Token
- **最大输出**: 65,536 Token
- **支持模型**: 
  - `qwen3-coder-plus` (推荐)
  - `qwen3-coder-plus-2025-07-22`
  - `qwen3-coder-flash`
  - `qwen3-coder-flash-2025-07-28`

### 💰 价格优势
- **限时优惠**: 最高5折优惠
- **缓存优惠**: 命中缓存的输入Token享受2.5折
- **免费额度**: 各100万Token（百炼开通后180天内）

## 🔧 快速开始

### 1. 获取API Key
1. 访问 [阿里云百炼](https://bailian.console.aliyun.com/)
2. 开通百炼服务
3. 获取API Key

### 2. 基本使用
```go
package main

import (
    "context"
    "fmt"
    "github.com/yunhanshu-net/function-go/pkg/llms"
)

func main() {
    // 创建千问3 Coder客户端
    client := llms.NewQwen3CoderClient("your-api-key")
    
    // 构造代码生成请求
    req := &llms.ChatRequest{
        Messages: []llms.Message{
            {
                Role: "system", 
                Content: "你是一个专业的Go语言开发助手，请生成可运行的代码",
            },
            {
                Role: "user", 
                Content: "请用Go语言编写一个快速排序函数",
            },
        },
        MaxTokens:   2000,
        Temperature: 0.1, // 代码生成需要低温度
    }
    
    // 调用API
    resp, err := client.Chat(context.Background(), req)
    if err != nil {
        fmt.Printf("请求失败: %v\n", err)
        return
    }
    
    if resp.Error != "" {
        fmt.Printf("API错误: %s\n", resp.Error)
        return
    }
    
    fmt.Printf("生成的代码:\n%s\n", resp.Content)
    
    if resp.Usage != nil {
        fmt.Printf("Token使用: 输入%d, 输出%d, 总计%d\n",
            resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
    }
}
```

### 3. 使用工厂函数
```go
// 通过工厂函数创建客户端
client, err := llms.NewLLMClient(llms.ProviderQwen3Coder, "your-api-key")
if err != nil {
    log.Fatal(err)
}

// 使用客户端
resp, err := client.Chat(ctx, req)
```

## 🎯 使用场景

### 1. 代码生成
```go
req := &llms.ChatRequest{
    Messages: []llms.Message{
        {
            Role: "user", 
            Content: "请用Go语言创建一个完整的Web服务器，包含路由、中间件和错误处理",
        },
    },
    MaxTokens:   4000,
    Temperature: 0.1,
}
```

### 2. 代码补全
```go
req := &llms.ChatRequest{
    Messages: []llms.Message{
        {
            Role: "user", 
            Content: "请补全以下Go函数:\nfunc calculateArea(width, height float64) ",
        },
    },
    MaxTokens:   1000,
    Temperature: 0.1,
}
```

### 3. 代码优化
```go
req := &llms.ChatRequest{
    Messages: []llms.Message{
        {
            Role: "user", 
            Content: "请优化以下Go代码的性能:\n[你的代码]",
        },
    },
    MaxTokens:   2000,
    Temperature: 0.1,
}
```

### 4. 函数调用（工具使用）
```go
req := &llms.ChatRequest{
    Messages: []llms.Message{
        {
            Role: "user", 
            Content: "请创建一个Python文件，包含一个计算斐波那契数列的函数",
        },
    },
    MaxTokens:   1500,
    Temperature: 0.1,
}
```

## ⚙️ 配置参数

### 模型选择
```go
req := &llms.ChatRequest{
    Model: "qwen3-coder-plus", // 最新稳定版
    // 或者使用快照版
    // Model: "qwen3-coder-plus-2025-07-22",
}
```

### 温度控制
```go
req := &llms.ChatRequest{
    Temperature: 0.1, // 代码生成推荐使用低温度
    // 0.0-0.3: 高确定性，适合代码生成
    // 0.3-0.7: 平衡，适合代码优化
    // 0.7-1.0: 高创造性，适合代码重构
}
```

### Token控制
```go
req := &llms.ChatRequest{
    MaxTokens: 4000, // 根据代码复杂度调整
    // 简单函数: 500-1000
    // 中等复杂度: 1000-2000
    // 复杂系统: 2000-4000
}
```

## 🔍 最佳实践

### 1. 提示词优化
```go
// ✅ 好的提示词
messages := []llms.Message{
    {
        Role: "system", 
        Content: "你是一个专业的Go语言开发助手。请生成可运行、符合Go最佳实践的代码。",
    },
    {
        Role: "user", 
        Content: "请用Go语言编写一个HTTP服务器，要求：\n1. 支持GET和POST请求\n2. 包含错误处理\n3. 添加日志记录\n4. 支持配置文件",
    },
}

// ❌ 不好的提示词
messages := []llms.Message{
    {
        Role: "user", 
        Content: "写个服务器", // 太模糊
    },
}
```

### 2. 错误处理
```go
resp, err := client.Chat(ctx, req)
if err != nil {
    // 网络错误或超时
    log.Printf("网络错误: %v", err)
    return
}

if resp.Error != "" {
    // API错误
    log.Printf("API错误: %s", resp.Error)
    return
}

// 检查响应内容
if resp.Content == "" {
    log.Printf("响应内容为空")
    return
}
```

### 3. 性能优化
```go
// 设置合理的超时时间
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

// 使用适当的Token数量
req := &llms.ChatRequest{
    MaxTokens: 2000, // 根据实际需要设置
}
```

## 🧪 测试

### 运行测试
```bash
# 进入测试目录
cd function-go/pkg/llms

# 运行千问3 Coder测试
go test -v -run TestQwen3Coder

# 运行特定测试
go test -v -run TestQwen3CoderCodeGeneration

# 运行性能测试
go test -v -bench=BenchmarkQwen3CoderChat
```

### 配置测试API Key
在 `qwen3_coder_test.go` 文件中更新：
```go
const testQwen3CoderAPIKey = "your-real-api-key-here"
```

## 📊 监控和成本控制

### Token使用监控
```go
if resp.Usage != nil {
    log.Printf("本次请求Token使用:")
    log.Printf("  输入: %d tokens", resp.Usage.PromptTokens)
    log.Printf("  输出: %d tokens", resp.Usage.CompletionTokens)
    log.Printf("  总计: %d tokens", resp.Usage.TotalTokens)
    
    // 计算成本（根据实际价格）
    inputCost := float64(resp.Usage.PromptTokens) / 1000 * 0.004
    outputCost := float64(resp.Usage.CompletionTokens) / 1000 * 0.016
    totalCost := inputCost + outputCost
    
    log.Printf("预估成本: ¥%.4f", totalCost)
}
```

### 成本优化建议
1. **使用缓存**: 相同输入享受2.5折优惠
2. **批量处理**: 一次请求处理多个相关任务
3. **Token控制**: 合理设置MaxTokens，避免浪费
4. **模型选择**: 根据任务复杂度选择合适的模型

## 🚨 注意事项

### 1. API限制
- 注意API调用频率限制
- 监控Token使用量
- 设置合理的超时时间

### 2. 代码质量
- 生成的代码需要人工审查
- 建议添加单元测试
- 注意安全性问题

### 3. 成本控制
- 监控每日Token使用量
- 设置成本告警
- 合理使用免费额度

## 🔗 相关链接

- [千问3 Coder 官方文档](https://help.aliyun.com/zh/bailian/)
- [阿里云百炼控制台](https://bailian.console.aliyun.com/)
- [function-go 框架](https://github.com/yunhanshu-net/function-go)

## 💡 总结

千问3 Coder 是一个强大的代码生成工具，通过合理的配置和使用，可以显著提高开发效率。记住：

1. **选择合适的模型和参数**
2. **优化提示词质量**
3. **监控Token使用和成本**
4. **人工审查生成的代码**
5. **充分利用免费额度**

祝你使用愉快！🚀



