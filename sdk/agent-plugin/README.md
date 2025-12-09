# Agent Plugin SDK

Agent Plugin SDK 是一个用于开发 Agent 插件的 SDK，简化了插件的开发流程。

## 功能特性

- ✅ 自动连接 NATS
- ✅ 自动订阅插件主题
- ✅ 统一的请求/响应处理
- ✅ 上下文信息传递（trace_id, user 等）
- ✅ 错误处理和日志记录
- ✅ 优雅关闭

## 快速开始

### 1. 创建插件处理函数

```go
package main

import (
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-plugin"
)

func handlePlugin(ctx *plugin.Context, req *dto.PluginRunReq) (*dto.PluginRunResp, error) {
	// 处理用户消息和文件
	// req.Message: 用户消息
	// req.Files: 文件列表（包含 URL 和备注）
	
	// 处理逻辑...
	processedData := "处理后的数据..."
	
	// 返回响应
	return &dto.PluginRunResp{
		Data: processedData,
	}, nil
}
```

### 2. 初始化并运行插件

```go
package main

import (
	"log"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-plugin"
)

func main() {
	// 创建插件实例
	// natsURL: NATS 服务器地址（如 "nats://127.0.0.1:4222"）
	// subject: 订阅的主题（如 "agent.function_gen.beiluo.1.run"）
	p, err := plugin.NewPlugin("nats://127.0.0.1:4222", "agent.function_gen.beiluo.1.run", handlePlugin)
	if err != nil {
		log.Fatalf("创建插件失败: %v", err)
	}
	defer p.Close()

	// 运行插件（阻塞，直到收到退出信号）
	if err := p.Run(); err != nil {
		log.Fatalf("运行插件失败: %v", err)
	}
}
```

## 完整示例

```go
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-plugin"
)

func excelProcessor(ctx *plugin.Context, req *dto.PluginRunReq) (*dto.PluginRunResp, error) {
	// 打印上下文信息
	fmt.Printf("TraceID: %s\n", ctx.GetTraceID())
	fmt.Printf("RequestUser: %s\n", ctx.GetRequestUser())

	// 处理用户消息
	fmt.Printf("收到消息: %s\n", req.Message)

	// 处理文件列表
	var fileURLs []string
	for _, file := range req.Files {
		fmt.Printf("文件: %s (备注: %s)\n", file.Url, file.Remark)
		fileURLs = append(fileURLs, file.Url)
	}

	// 模拟处理逻辑（实际应该下载文件、解析 Excel 等）
	processedData := fmt.Sprintf("已处理 %d 个文件:\n%s", len(fileURLs), strings.Join(fileURLs, "\n"))

	// 返回处理后的数据（格式化后的文本，供 LLM 理解）
	return &dto.PluginRunResp{
		Data: processedData,
	}, nil
}

func main() {
	// 创建插件实例
	p, err := plugin.NewPlugin("nats://127.0.0.1:4222", "agent.function_gen.beiluo.1.run", excelProcessor)
	if err != nil {
		log.Fatalf("创建插件失败: %v", err)
	}
	defer p.Close()

	// 运行插件
	if err := p.Run(); err != nil {
		log.Fatalf("运行插件失败: %v", err)
	}
}
```

## 插件主题

插件需要手动指定订阅的主题：
- 格式: `agent.{chat_type}.{user}.{id}.run`
- 示例: `agent.function_gen.beiluo.1.run`
- 在创建插件时直接传入主题字符串即可

## 请求/响应格式

### 请求 (PluginRunReq)

```go
type PluginRunReq struct {
	Message string       // 用户消息
	Files   []PluginFile // 文件列表
}

type PluginFile struct {
	Url    string // 文件URL
	Remark string // 文件备注
}
```

### 响应 (PluginRunResp)

```go
type PluginRunResp struct {
	Data string // 处理后的数据（格式化后的文本，供LLM理解）
}
```

## 错误处理

如果处理函数返回错误，SDK 会自动：
1. 记录错误日志
2. 通过 NATS 返回错误响应

```go
func handlePlugin(ctx *plugin.Context, req *dto.PluginRunReq) (*dto.PluginRunResp, error) {
	// 返回错误
	return nil, fmt.Errorf("处理失败: %v", err)
}
```

## 优雅关闭

插件支持优雅关闭：
- 收到 `SIGINT` 或 `SIGTERM` 信号时自动关闭
- 取消 NATS 订阅
- 关闭 NATS 连接

## 注意事项

1. **处理时间**: 插件处理应该在合理时间内完成（建议 < 600 秒），否则可能超时
2. **错误处理**: 确保处理函数能正确处理所有错误情况
3. **数据格式**: 返回的 `Data` 应该是格式化后的文本，方便 LLM 理解
4. **并发**: SDK 内部使用 goroutine 处理消息，但处理函数本身是同步的

