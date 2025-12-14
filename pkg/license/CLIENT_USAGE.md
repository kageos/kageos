# License Client 使用说明

## 概述

`pkg/license/client.go` 提供了通用的 License Client，用于各服务实例获取和更新 License 密钥。

## 功能特性

1. **启动时自动获取密钥**：通过 NATS 请求-响应模式获取 License 密钥
2. **本地缓存**：获取到的密钥保存到本地文件，下次启动可直接使用
3. **自动刷新**：监听刷新主题，收到刷新指令时自动获取新密钥并对比更新
4. **密钥对比**：如果新密钥与本地相同，则不更新，避免不必要的操作

## 使用方法

### 1. 创建 License Client

```go
import (
    "github.com/ai-agent-os/ai-agent-os/pkg/license"
    "github.com/nats-io/nats.go"
)

// 连接 NATS
natsConn, err := nats.Connect("nats://127.0.0.1:4223")
if err != nil {
    // 处理错误
}

// 创建 License Client
// encryptionKey: 32字节加密密钥（必须与 Control Service 相同）
// keyPath: 本地密钥文件路径（可选，默认：~/.ai-agent-os/license.key）
encryptionKey := []byte("ai-agent-os-license-key-32bytes!!")
client, err := license.NewClient(natsConn, encryptionKey, "")
if err != nil {
    // 处理错误
}
```

### 2. 启动 Client

```go
ctx := context.Background()
if err := client.Start(ctx); err != nil {
    // 处理错误（注意：如果无法连接 NATS，会使用社区版，不返回错误）
    log.Printf("Failed to start license client: %v", err)
}
```

### 3. 停止 Client

```go
ctx := context.Background()
if err := client.Stop(ctx); err != nil {
    // 处理错误
}
```

## 工作流程

### 启动时

1. 尝试从本地文件加载密钥
2. 如果本地没有，通过 NATS 请求获取密钥
3. 保存密钥到本地文件
4. 订阅刷新主题

### 刷新时

1. 收到刷新指令
2. 请求新的密钥
3. 与本地密钥对比
4. 如果不同，解密并更新 License
5. 保存新密钥到本地

## 配置

### 加密密钥

加密密钥必须与 Control Service 的配置一致（32字节）。

配置文件：`configs/control-service.yaml`

```yaml
license:
  encryption_key: "ai-agent-os-license-key-32bytes!!"
```

### 本地密钥文件路径

默认路径：`~/.ai-agent-os/license.key`

可以通过 `NewClient` 的 `keyPath` 参数自定义。

## 注意事项

1. **加密密钥必须一致**：所有服务实例和 Control Service 必须使用相同的加密密钥
2. **NATS 连接**：如果无法连接 NATS，Client 会使用社区版（不返回错误）
3. **本地密钥文件权限**：密钥文件权限设置为 `0600`（仅所有者可读写）
4. **自动刷新**：Client 会自动监听刷新主题，无需手动处理

## 示例

完整示例：

```go
package main

import (
    "context"
    "github.com/ai-agent-os/ai-agent-os/pkg/license"
    "github.com/nats-io/nats.go"
)

func main() {
    // 连接 NATS
    natsConn, _ := nats.Connect("nats://127.0.0.1:4223")
    defer natsConn.Close()

    // 创建 License Client
    encryptionKey := []byte("ai-agent-os-license-key-32bytes!!")
    client, err := license.NewClient(natsConn, encryptionKey, "")
    if err != nil {
        panic(err)
    }

    // 启动 Client
    ctx := context.Background()
    if err := client.Start(ctx); err != nil {
        // 处理错误
    }

    // 使用 License Manager 检查功能
    mgr := license.GetManager()
    if mgr.IsEnterprise() {
        // 企业版功能
    }

    // 停止 Client
    defer client.Stop(ctx)
}
```
