# 统一入口启动多个微服务 - 可行性分析

## 📋 方案概述

将各个服务的 `main()` 函数改为 `Main()` 函数，然后在统一入口中调用，实现一次启动多个服务。

## ✅ 优势

1. **开发环境便捷**：一次启动所有服务，提升开发效率
2. **代码复用**：避免重复的启动逻辑
3. **统一管理**：集中管理服务生命周期

## ⚠️ 需要注意的问题

### 1. **日志系统冲突**（关键问题）

**问题描述**：
- `logger.Init()` 使用全局变量，多次调用会互相覆盖
- 最后一个服务的日志配置会生效，其他服务的日志可能丢失

**当前实现**：
```go
var (
    logger      *zap.Logger  // 全局单例
    sugar       *zap.SugaredLogger
    initialized bool
)
```

**解决方案**：
- **方案 A**：统一入口只初始化一次日志系统，所有服务共享（推荐）
- **方案 B**：修改日志系统支持多实例（复杂，不推荐）
- **方案 C**：每个服务使用独立的日志文件，但共享日志系统（推荐）

### 2. **信号处理冲突**

**问题描述**：
- 每个服务的 `Main()` 函数都在等待 `SIGINT/SIGTERM` 信号
- 多个服务同时等待同一个信号，会导致关闭逻辑混乱

**解决方案**：
- **统一入口处理信号**：只在统一入口等待信号，然后通知所有服务关闭
- **服务不阻塞**：服务的 `Main()` 函数不等待信号，而是返回一个可关闭的 Service 对象

### 3. **错误处理**

**问题描述**：
- 一个服务启动失败，是否影响其他服务？

**解决方案**：
- **独立启动**：每个服务在独立的 goroutine 中启动，互不影响
- **错误收集**：收集所有服务的启动错误，统一处理
- **优雅降级**：部分服务启动失败，其他服务继续运行

### 4. **端口冲突**

**问题描述**：
- 需要确保各服务端口不冲突

**解决方案**：
- **配置检查**：启动前检查端口是否被占用
- **配置分离**：每个服务使用独立的配置文件

## 🎯 推荐实现方案

### 方案 A：统一入口 + 协程启动（推荐用于开发环境）

**架构**：
```
统一入口 (core/cmd/main/main.go)
  ├── 初始化日志系统（只一次）
  ├── 创建统一的 context 和 stopCh
  ├── 启动 Control Service (goroutine, 传入 ctx 和 stopCh)
  ├── 启动 App Server (goroutine, 传入 ctx 和 stopCh)
  ├── 启动 Agent Server (goroutine, 传入 ctx 和 stopCh)
  ├── 启动 App Runtime (goroutine, 传入 ctx 和 stopCh)
  ├── 启动 App Storage (goroutine, 传入 ctx 和 stopCh)
  ├── 启动 API Gateway (goroutine, 传入 ctx 和 stopCh)
  └── 统一等待信号，关闭 stopCh，优雅关闭所有服务
```

**Main 函数签名**：
```go
// 所有服务的 Main 函数统一签名
func Main(ctx context.Context, stopCh <-chan struct{}) error
```

**优点**：
- ✅ 简单直接，易于实现
- ✅ 适合开发环境
- ✅ 统一管理服务生命周期
- ✅ Main 函数接收参数，统一入口可以控制启动参数
- ✅ 保持独立启动的兼容性（main 函数创建 stopCh）

**缺点**：
- ⚠️ 需要修改各个服务的 `Main()` 函数，使其接收参数并监听 stopCh

### 方案 B：保持独立启动（推荐用于生产环境）

**架构**：
```
各个服务独立启动
  ├── Control Service (独立进程)
  ├── App Server (独立进程)
  ├── Agent Server (独立进程)
  └── ...
```

**优点**：
- ✅ 服务隔离，互不影响
- ✅ 适合生产环境
- ✅ 可以独立扩展和部署

**缺点**：
- ⚠️ 开发环境需要手动启动多个服务

## 📝 实现步骤

### 步骤 1：修改各个服务的 Main() 函数，使其接收参数

将 `main()` 改为 `Main(ctx, stopCh)`，接收统一的 context 和停止通道：

```go
// core/agent-server/cmd/main/main.go
func Main(ctx context.Context, stopCh <-chan struct{}) error {
    // ... 初始化逻辑 ...
    
    srv, err := server.NewServer(cfg)
    if err != nil {
        return err
    }
    
    if err := srv.Start(ctx); err != nil {
        return err
    }
    
    // 等待停止信号
    select {
    case <-ctx.Done():
        // 上下文被取消
    case <-stopCh:
        // 收到停止信号
    }
    
    // 优雅关闭
    return srv.Stop(ctx)
}

// main 独立启动入口（保持向后兼容）
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

### 步骤 2：创建统一入口

```go
// core/cmd/main/main.go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // 1. 初始化统一日志系统
    logger.Init(...)
    
    // 2. 创建停止通道
    stopCh := make(chan struct{})
    
    // 3. 在 goroutine 中启动所有服务
    var wg sync.WaitGroup
    for _, svc := range services {
        wg.Add(1)
        go func(info ServiceInfo) {
            defer wg.Done()
            if err := info.Main(ctx, stopCh); err != nil {
                // 处理错误
            }
        }(svc)
    }
    
    // 4. 统一等待信号
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
    
    // 5. 通知所有服务停止
    close(stopCh)
    cancel()
    wg.Wait()
}
```

## 🔍 关键决策点

### 1. 日志系统如何处理？

**推荐**：统一入口初始化一次，所有服务共享日志系统，但使用不同的日志文件。

```go
// 统一入口初始化
logger.Init(logger.Config{
    Filename: "./logs/all-services.log",  // 统一日志文件
    // ...
})

// 各个服务仍然可以有自己的日志文件（通过配置）
// 但共享日志系统实例
```

### 2. 信号处理如何处理？

**推荐**：统一入口处理信号，各个服务不阻塞，而是返回 Service 对象。

### 3. 错误处理策略？

**推荐**：
- 启动阶段：收集所有错误，如果关键服务失败则退出
- 运行阶段：服务独立运行，一个服务崩溃不影响其他服务

## 🎯 最终建议

### 开发环境
- ✅ **使用统一入口**：方便开发，一次启动所有服务
- ✅ **统一日志系统**：简化日志管理
- ✅ **统一信号处理**：优雅关闭所有服务

### 生产环境
- ✅ **保持独立启动**：服务隔离，独立扩展
- ✅ **使用进程管理器**：如 systemd、supervisor、docker-compose 等

## 📌 总结

**可行性**：✅ **完全可行**

**适用场景**：
- ✅ 开发环境：强烈推荐
- ✅ 测试环境：推荐
- ⚠️ 生产环境：不推荐（应保持独立启动）

**实现难度**：中等
- 需要修改各个服务的启动逻辑
- 需要统一处理日志和信号
- 需要处理错误和优雅关闭

