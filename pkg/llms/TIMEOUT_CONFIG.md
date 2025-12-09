# LLM客户端超时配置指南

## 🔍 问题描述

如果你遇到前端传递600秒超时时间，但实际还是使用300秒超时的问题，这篇文章将帮你解决。

## 🎯 超时配置的三种方式

### 1. 配置文件方式（推荐）

在配置文件中设置超时时间（单位：秒）：

```json
{
  "providers": {
    "kimi": {
      "api_key": "your-kimi-api-key-here",
      "base_url": "https://api.moonshot.cn/v1/chat/completions",
      "timeout": 600
    }
  },
  "default": "kimi"
}
```

**重要**：配置文件中的`timeout`字段单位是**秒**，不是毫秒。

### 2. 代码创建方式

```go
// 创建带自定义超时的客户端
options := DefaultClientOptions().WithTimeout(600 * time.Second)
client := NewKimiClientWithOptions("your-api-key", options)
```

### 3. 请求级别超时

```go
// 在请求中指定超时时间
requestTimeout := 600 * time.Second
req := &ChatRequest{
    Messages: []Message{{Role: "user", Content: "Hello"}},
    Timeout:  &requestTimeout, // 请求级别超时，覆盖客户端默认超时
}
```

## 🚀 超时优先级

超时时间的优先级从高到低：

1. **请求级别超时** (`req.Timeout`) - 最高优先级
2. **客户端配置超时** (`client.Options.Timeout`) - 中等优先级  
3. **默认超时** (60秒) - 最低优先级

## 🔧 常见问题排查

### 问题1：配置文件超时不生效

**原因**：配置文件中的超时时间没有被正确加载或转换

**解决方案**：
1. 确保配置文件路径正确
2. 检查配置文件格式是否正确
3. 确保调用了`LoadConfig()`函数

```go
// 加载配置文件
err := LoadConfig("config.json")
if err != nil {
    log.Fatal("加载配置失败:", err)
}

// 从配置创建客户端
client, err := CreateClientFromConfig(ProviderKimi)
if err != nil {
    log.Fatal("创建客户端失败:", err)
}
```

### 问题2：请求级别超时不生效

**原因**：请求中的超时字段为nil或值为0

**解决方案**：
```go
// 确保超时字段有值
if req.Timeout == nil || *req.Timeout <= 0 {
    // 设置默认超时
    defaultTimeout := 600 * time.Second
    req.Timeout = &defaultTimeout
}
```

### 问题3：超时单位混淆

**原因**：配置文件使用秒，但代码期望time.Duration

**解决方案**：代码已自动处理单位转换
- 配置文件：`"timeout": 600` (600秒)
- 代码内部：自动转换为`600 * time.Second`

## 📝 完整示例

### 配置文件 (config.json)
```json
{
  "providers": {
    "kimi": {
      "api_key": "your-kimi-api-key",
      "timeout": 600
    }
  },
  "default": "kimi"
}
```

### 代码使用
```go
package main

import (
    "github.com/yunhanshu-net/function-go/pkg/llms"
)

func main() {
    // 1. 加载配置文件
    err := llms.LoadConfig("config.json")
    if err != nil {
        panic(err)
    }

    // 2. 从配置创建客户端
    client, err := llms.CreateClientFromConfig(llms.ProviderKimi)
    if err != nil {
        panic(err)
    }

    // 3. 创建请求（可选：请求级别超时）
    requestTimeout := 600 * time.Second
    req := &llms.ChatRequest{
        Messages: []llms.Message{
            {Role: "user", Content: "请帮我写一个Go程序"},
        },
        Timeout: &requestTimeout, // 请求级别超时
    }

    // 4. 发送请求
    resp, err := client.Chat(context.Background(), req)
    if err != nil {
        panic(err)
    }

    fmt.Println("AI回答:", resp.Content)
}
```

## 🧪 测试超时配置

运行测试确保超时配置正确：

```bash
cd function-go/pkg/llms
go test -v -run TestTimeoutConfiguration
```

## 🔍 调试技巧

在Kimi客户端的Chat方法中，已经添加了调试日志：

```go
// 调试日志：记录超时设置
fmt.Printf("🔍 Kimi超时调试: 客户端默认超时=%v, 请求超时=%v, 最终使用超时=%v\n",
    c.Options.Timeout, req.Timeout, timeout)
```

运行程序时查看这些日志，确认超时时间是否正确设置。

## 📚 相关文件

- `config.go` - 配置文件加载和超时转换
- `factory.go` - 客户端工厂和超时传递
- `kimi.go` - Kimi客户端超时处理
- `interface.go` - 超时配置结构定义

## 🎯 总结

要解决600秒超时不生效的问题：

1. **配置文件方式**：确保`"timeout": 600`正确设置
2. **代码方式**：使用`WithTimeout(600 * time.Second)`
3. **请求方式**：在`ChatRequest`中设置`Timeout: &600*time.Second`
4. **检查日志**：查看调试输出确认超时设置
5. **运行测试**：使用测试文件验证配置

按照以上步骤，你的600秒超时配置就能正常工作了！
