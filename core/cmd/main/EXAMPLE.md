# 统一入口实现示例

## 📝 如何改造服务以支持统一入口

### 步骤 1：修改服务的 Main() 函数，使其接收参数

将 `main()` 改为 `Main(ctx, stopCh)`，接收统一的 context 和停止通道：

```go
// core/agent-server/cmd/main/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/server"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// Main 服务主函数（支持统一入口调用）
// ctx: 统一的上下文
// stopCh: 停止信号通道，服务应该监听此通道并在收到信号时优雅关闭
func Main(ctx context.Context, stopCh <-chan struct{}) error {
	// 获取配置
	cfg := config.GetAgentServerConfig()

	// 初始化日志系统（如果统一入口没有初始化）
	// 注意：统一入口已经初始化了日志系统，这里可以跳过
	// 但如果独立启动，仍然需要初始化
	logConfig := logger.Config{
		Level:      cfg.GetLogLevel(),
		Filename:   "./logs/agent-server.log",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   true,
		IsDev:      cfg.IsDebug(),
	}

	if err := logger.Init(logConfig); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	logger.Infof(ctx, "Logger initialized - Service: agent-server")

	// 创建并启动服务器
	srv, err := server.NewServer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	if err := srv.Start(ctx); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	logger.Infof(ctx, "Agent-server started successfully")

	// 等待停止信号
	select {
	case <-ctx.Done():
		// 上下文被取消
		logger.Infof(ctx, "Context cancelled, shutting down agent-server...")
	case <-stopCh:
		// 收到停止信号
		logger.Infof(ctx, "Received stop signal, shutting down agent-server...")
	}

	// 优雅关闭
	if err := srv.Stop(ctx); err != nil {
		return fmt.Errorf("error during shutdown: %w", err)
	}

	logger.Infof(ctx, "Agent-server stopped")
	return nil
}

// main 独立启动入口（保持向后兼容）
func main() {
	ctx := context.Background()

	// 创建停止通道（独立启动时使用信号）
	stopCh := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 在 goroutine 中运行服务
	errCh := make(chan error, 1)
	go func() {
		errCh <- Main(ctx, stopCh)
	}()

	// 等待信号或错误
	select {
	case sig := <-sigChan:
		fmt.Printf("Received signal: %v\n", sig)
		close(stopCh)
		if err := <-errCh; err != nil {
			fmt.Printf("Error during shutdown: %v\n", err)
			os.Exit(1)
		}
	case err := <-errCh:
		if err != nil {
			fmt.Printf("Service error: %v\n", err)
			os.Exit(1)
		}
	}
}
```

### 步骤 2：在统一入口中注册服务

```go
// core/cmd/main/main.go
package main

import (
	// 导入各个服务的 Main 函数
	agentServerMain "github.com/ai-agent-os/ai-agent-os/core/agent-server/cmd/main"
	appServerMain "github.com/ai-agent-os/ai-agent-os/core/app-server/cmd/main"
	controlServiceMain "github.com/ai-agent-os/ai-agent-os/core/control-service/cmd/app"
	// ... 其他服务
)

func init() {
	// 注册要启动的服务
	services = append(services, ServiceInfo{
		Name: "control-service",
		Main: controlServiceMain.Main,
	})
	services = append(services, ServiceInfo{
		Name: "app-server",
		Main: appServerMain.Main,
	})
	services = append(services, ServiceInfo{
		Name: "agent-server",
		Main: agentServerMain.Main,
	})
	// ... 其他服务
}
```

## 🔧 关键改造点

### 1. Main 函数签名

**统一签名**：所有服务的 Main 函数都接收相同的参数

```go
// 统一签名
func Main(ctx context.Context, stopCh <-chan struct{}) error
```

**参数说明**：
- `ctx`: 统一的上下文，可以传递配置、日志等信息
- `stopCh`: 停止信号通道，服务应该监听此通道并在收到信号时优雅关闭

### 2. 日志系统处理

**方案 A**：统一入口初始化，服务检查是否已初始化（需要添加 IsInitialized 方法）

**方案 B**（推荐）：每个服务仍然初始化自己的日志（使用不同文件）

```go
// 在服务中
logger.Init(logger.Config{
    Filename: "./logs/agent-server.log",  // 不同的日志文件
    // ...
})
```

虽然会覆盖全局 logger 实例，但各自的日志文件是独立的，不影响文件输出。

### 3. 信号处理

**统一入口处理信号**，通过 stopCh 通知服务：

```go
// 在统一入口中
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
<-sigChan

// 通知所有服务关闭
close(stopCh)
```

**服务中监听停止信号**：

```go
// 在服务的 Main 函数中
select {
case <-ctx.Done():
    // 上下文被取消
case <-stopCh:
    // 收到停止信号
}
// 执行优雅关闭
```

### 4. 独立启动兼容性

保持独立启动的兼容性，在 `main()` 中创建 stopCh 并通过信号触发：

```go
func main() {
    ctx := context.Background()
    stopCh := make(chan struct{})
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        <-sigChan
        close(stopCh)
    }()
    
    if err := Main(ctx, stopCh); err != nil {
        os.Exit(1)
    }
}
```

## 📋 完整实现清单

- [ ] 修改 `agent-server/cmd/main/main.go` 实现 `ServiceRunner`
- [ ] 修改 `app-server/cmd/main/main.go` 实现 `ServiceRunner`
- [ ] 修改 `control-service/cmd/main/main.go` 实现 `ServiceRunner`
- [ ] 修改 `app-runtime/cmd/app/main.go` 实现 `ServiceRunner`
- [ ] 修改 `app-storage/cmd/app/main.go` 实现 `ServiceRunner`
- [ ] 修改 `api-gateway/cmd/app/main.go` 实现 `ServiceRunner`
- [ ] 在统一入口中注册所有服务
- [ ] 测试统一启动和独立启动两种方式

## 🎯 使用方式

### 开发环境：统一启动

```bash
go run core/cmd/main/main.go
```

### 生产环境：独立启动

```bash
go run core/agent-server/cmd/app/main.go
go run core/app-server/cmd/app/main.go
# ... 其他服务
```

