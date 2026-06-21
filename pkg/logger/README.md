# Logger

当前实现是一个基于 `zap` 和 `lumberjack` 的全局日志包装层，提供统一初始化、开发/生产两种输出形态，以及一组包级日志函数。

## 公开接口

初始化是可选的：

- 如果显式调用 `Init`，会按传入配置初始化一次。
- 如果不初始化，第一次写日志时会自动使用默认配置。
- `Init` 是幂等的，已经初始化后再次调用会直接返回。

```go
type Config struct {
    Level      string // debug, info, warn, error
    Filename   string // 日志文件路径
    MaxSize    int    // 单个文件最大大小（MB）
    MaxBackups int    // 保留旧文件数量
    MaxAge     int    // 保留旧文件天数
    Compress   bool   // 是否压缩旧文件
    IsDev      bool   // 开发环境开关
}

func Init(cfg Config) error
func IsInitialized() bool
func Sync() error
```

包级日志函数：

```go
func Debug(ctx context.Context, msg string, fields ...zap.Field)
func Debugf(ctx context.Context, format string, args ...interface{})
func Info(ctx context.Context, msg string, fields ...zap.Field)
func Infof(ctx context.Context, format string, args ...interface{})
func Warn(ctx context.Context, msg string, fields ...zap.Field)
func Warnf(ctx context.Context, format string, args ...interface{})
func Error(ctx context.Context, msg string, args ...interface{})
func Errorf(ctx context.Context, format string, args ...interface{})
func Fatal(ctx context.Context, msg string, args ...interface{})
func Fatalf(ctx context.Context, format string, args ...interface{})
```

`ctx` 现在主要用于统一调用签名；当前实现不会从上下文提取额外字段。

## 默认行为

未显式初始化时，会自动使用这组默认值：

```go
logger.Config{
    Level:      "info",
    Filename:   "./logs/app.log",
    MaxSize:    100,
    MaxBackups: 3,
    MaxAge:     7,
    Compress:   true,
    IsDev:      true,
}
```

## 输出模式

- `IsDev=true`：同时输出到控制台和文件，使用可读性更高的 console encoder。
- `IsDev=false`：输出到文件，使用 JSON encoder。

两种模式都会带时间、级别和调用位置；文件会通过 `lumberjack` 做滚动切分。

## 示例

```go
package main

import (
    "context"

    "github.com/kageos/kageos-sdk/pkg/logger"
)

func main() {
    _ = logger.Init(logger.Config{
        Level:      "debug",
        Filename:   "./logs/app.log",
        MaxSize:    100,
        MaxBackups: 3,
        MaxAge:     7,
        Compress:   true,
        IsDev:      true,
    })
    defer logger.Sync()

    ctx := context.Background()

    logger.Infof(ctx, "service started on %s", ":9090")
    logger.Warn(ctx, "cache miss")
    logger.Errorf(ctx, "request failed: %v", "timeout")
}
```

## 配置文件示例

```json
{
  "level": "info",
  "filename": "./logs/app.log",
  "max_size": 100,
  "max_backups": 3,
  "max_age": 7,
  "compress": true,
  "is_dev": false
}
```

## 注意事项

- `Fatal` / `Fatalf` 会调用底层 logger 的 fatal 逻辑并退出进程。
- 建议在进程退出前调用一次 `Sync()`，尤其是只写文件的场景。
- 当前包已经不再提供旧版的 `NewLogger`、`LogConfig`、`Output`、`TimeFormat`、`ShowCaller` 等 API。
