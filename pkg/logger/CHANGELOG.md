# Logger 变更日志

## v1.0.0 (2025-10-14)

### 新增功能

- ✅ 支持多种日志级别：DEBUG, INFO, WARN, ERROR, FATAL
- ✅ 支持上下文日志记录（第一个参数是 ctx）
- ✅ 可配置的输出目标（stdout, stderr, 文件）
- ✅ 可配置的时间格式
- ✅ 线程安全的日志记录
- ✅ 简单的 API 设计
- ✅ 支持从配置文件加载日志配置
- ✅ 完整的测试覆盖
- ✅ 详细的文档和示例

### API 设计

#### 基本使用
```go
logger.Info("Application started")
logger.Debug("Debug message: %s", "some debug info")
logger.Warn("Warning message")
logger.Error("Error message")
```

#### 带上下文的日志（第一个参数是 ctx）
```go
ctx := context.Background()
ctxLogger := logger.WithContext(ctx)
ctxLogger.Info("Message with context")
```

#### 自定义配置
```go
config := &logger.Config{
    Level:      logger.DEBUG,
    Output:     os.Stdout,
    TimeFormat: "2006-01-02 15:04:05.000",
    ShowCaller: true,
}
log := logger.NewLogger(config)
```

### 集成状态

- ✅ 已集成到 containerd 服务
- ✅ 替换了原有的 log 包
- ✅ 支持配置文件和环境变量

### 测试

- ✅ 单元测试通过
- ✅ 示例程序运行正常
- ✅ 集成测试通过








