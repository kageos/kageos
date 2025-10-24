# Logger 日志库

一个简单易用的 Go 日志库，支持上下文和配置。

## 特性

- 支持多种日志级别：DEBUG, INFO, WARN, ERROR, FATAL
- 支持上下文日志记录
- 可配置的输出目标（stdout, stderr, 文件）
- 可配置的时间格式
- 线程安全
- 简单的 API 设计

## 快速开始

### 初始化（可选）

日志库**不需要初始化**，如果不初始化会使用默认配置：

```go
// 默认配置：
// - 级别: INFO
// - 输出: stdout
// - 时间格式: "2006-01-02 15:04:05"
// - 显示调用者: false
```

如果需要自定义配置，可以在程序启动时初始化：

```go
// 方式1: 手动初始化
logger.Init(&logger.Config{
    Level:      logger.DEBUG,
    Output:     os.Stdout,
    TimeFormat: "2006-01-02 15:04:05.000",
    ShowCaller: true,
})

// 方式2: 从配置文件初始化
logConfig := &logger.LogConfig{
    Level:      "debug",
    Output:     "stdout",
    TimeFormat: "2006-01-02 15:04:05.000",
    ShowCaller: true,
}
logger.InitFromConfig(logConfig)
```

### 基本使用

```go
package main

import (
    "context"
    "github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

func main() {
    ctx := context.Background()
    
    // 使用默认日志器（第一个参数是 ctx）
    logger.Info(ctx, "Application started")
    logger.Debug(ctx, "Debug message: %s", "some debug info")
    logger.Warn(ctx, "Warning message")
    logger.Error(ctx, "Error message")
}
```

### 带上下文的日志

```go
func handleRequest(ctx context.Context, userID string) {
    // 直接使用全局函数，第一个参数是 ctx
    logger.Info(ctx, "Processing request for user: %s", userID)
    
    // 处理逻辑...
    
    logger.Info(ctx, "Request processed successfully")
}

func main() {
    ctx := context.Background()
    ctx = context.WithValue(ctx, "request_id", "req-123")
    
    handleRequest(ctx, "user-789")
}
```

### 自定义日志器

```go
import (
    "os"
    "github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

func main() {
    // 创建自定义配置
    config := &logger.Config{
        Level:      logger.DEBUG,
        Output:     os.Stdout,
        TimeFormat: "2006-01-02 15:04:05.000",
        ShowCaller: true,
    }
    
    // 创建日志器
    log := logger.NewLogger(config)
    
    // 使用日志器
    log.Info("Custom logger message")
    
    // 设置默认日志器
    logger.SetDefaultLogger(log)
}
```

### 从配置文件加载

```go
import (
    "github.com/ai-agent-os/ai-agent-os/pkg/logger"
    "github.com/ai-agent-os/ai-agent-os/pkg/config"
)

func main() {
    // 加载日志配置
    manager := config.NewSimpleManager()
    var logConfig logger.LogConfig
    if err := manager.LoadYAML("logger.yaml", &logConfig); err != nil {
        // 使用默认配置
        logConfig = *logger.GetDefaultLogConfig()
    }
    
    // 创建日志器
    log := logger.NewLogger(logConfig.ToLoggerConfig())
    logger.SetDefaultLogger(log)
    
    logger.Info("Logger configured from file")
}
```

## 配置

### 日志级别

- `DEBUG`: 调试信息
- `INFO`: 一般信息
- `WARN`: 警告信息
- `ERROR`: 错误信息
- `FATAL`: 致命错误（会退出程序）

### 输出目标

- `stdout`: 标准输出
- `stderr`: 标准错误
- `文件路径`: 输出到指定文件

### 时间格式

使用 Go 的时间格式字符串，例如：
- `2006-01-02 15:04:05` (默认)
- `2006-01-02 15:04:05.000`
- `2006/01/02 15:04:05`

## 配置文件示例

### logger.yaml

```yaml
level: "info"
output: "stdout"
time_format: "2006-01-02 15:04:05"
show_caller: false
```

### logger.json

```json
{
  "level": "debug",
  "output": "/var/log/app.log",
  "time_format": "2006-01-02 15:04:05.000",
  "show_caller": true
}
```

## API 参考

### 全局函数

```go
// 使用默认日志器（第一个参数是 ctx）
logger.Debug(ctx context.Context, msg string, args ...interface{})
logger.Info(ctx context.Context, msg string, args ...interface{})
logger.Warn(ctx context.Context, msg string, args ...interface{})
logger.Error(ctx context.Context, msg string, args ...interface{})
logger.Fatal(ctx context.Context, msg string, args ...interface{})

// 带上下文的日志器
logger.WithContext(ctx context.Context) *ContextLogger
```

### Logger 结构体

```go
type Logger struct {
    // 内部字段
}

// 创建日志器
func NewLogger(config *Config) *Logger

// 设置日志级别
func (l *Logger) SetLevel(level Level)

// 设置输出
func (l *Logger) SetOutput(w io.Writer)

// 记录日志
func (l *Logger) Debug(msg string, args ...interface{})
func (l *Logger) Info(msg string, args ...interface{})
func (l *Logger) Warn(msg string, args ...interface{})
func (l *Logger) Error(msg string, args ...interface{})
func (l *Logger) Fatal(msg string, args ...interface{})

// 带上下文的日志器
func (l *Logger) WithContext(ctx context.Context) *ContextLogger
```

### ContextLogger 结构体

```go
type ContextLogger struct {
    // 内部字段
}

// 记录日志（自动包含上下文信息）
func (cl *ContextLogger) Debug(msg string, args ...interface{})
func (cl *ContextLogger) Info(msg string, args ...interface{})
func (cl *ContextLogger) Warn(msg string, args ...interface{})
func (cl *ContextLogger) Error(msg string, args ...interface{})
func (cl *ContextLogger) Fatal(msg string, args ...interface{})
```

## 测试

```bash
go test ./pkg/logger
```

## 示例

查看 `example.go` 文件了解完整的使用示例。
