# pprof 性能分析工具使用指南

## 📋 概述

本项目已集成 Go 的 `pprof` 性能分析工具，可以用于分析 CPU、内存、goroutine 等性能瓶颈。

## 🚀 快速开始

### 1. 启动服务

使用统一入口启动所有服务：

```bash
go run core/cmd/main/main.go
```

或者单独启动某个服务（如 app-server）：

```bash
go run core/app-server/cmd/app/main.go
```

### 2. 访问 pprof 端点

服务启动后，可以通过以下 URL 访问 pprof：

- **app-server**: `http://localhost:9090/debug/pprof/`
- **api-gateway**: `http://localhost:5173/debug/pprof/`

## 📊 可用的 Profile 类型

### 1. 主页面（Index）

访问 `http://localhost:9090/debug/pprof/` 查看所有可用的 profile 类型。

### 2. CPU Profile（CPU 性能分析）

**用途**：分析 CPU 使用情况，找出 CPU 密集型操作。

**使用方法**：

```bash
# 采集 30 秒的 CPU profile
go tool pprof http://localhost:9090/debug/pprof/profile?seconds=30

# 或者直接下载 profile 文件
curl http://localhost:9090/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof
```

**交互式命令**：
- `top` - 显示占用 CPU 最多的函数
- `top10` - 显示前 10 个
- `list <函数名>` - 查看函数的具体代码
- `web` - 生成调用图（需要安装 graphviz）
- `svg` - 生成 SVG 格式的调用图

### 3. Heap Profile（内存分析）

**用途**：分析内存使用情况，找出内存泄漏或占用过多的代码。

**使用方法**：

```bash
# 交互式分析
go tool pprof http://localhost:9090/debug/pprof/heap

# 或下载文件
curl http://localhost:9090/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

**交互式命令**：
- `top` - 显示占用内存最多的函数
- `top10 -cum` - 按累计内存占用排序
- `list <函数名>` - 查看函数的具体代码
- `web` - 生成内存分配调用图

### 4. Goroutine Profile（Goroutine 分析）

**用途**：分析所有 goroutine 的状态，找出 goroutine 泄漏或阻塞。

**使用方法**：

```bash
go tool pprof http://localhost:9090/debug/pprof/goroutine
```

**交互式命令**：
- `top` - 显示 goroutine 数量最多的函数
- `list <函数名>` - 查看函数的具体代码
- `web` - 生成 goroutine 调用图

### 5. Block Profile（阻塞分析）

**用途**：分析阻塞操作（如 channel 阻塞、mutex 等待）。

**注意**：需要先启用 block profiling：

```go
import _ "net/http/pprof"
import "runtime"

func init() {
    runtime.SetBlockProfileRate(1) // 启用 block profiling
}
```

**使用方法**：

```bash
go tool pprof http://localhost:9090/debug/pprof/block
```

### 6. Mutex Profile（互斥锁分析）

**用途**：分析互斥锁竞争情况。

**注意**：需要先启用 mutex profiling：

```go
import "runtime"

func init() {
    runtime.SetMutexProfileFraction(1) // 启用 mutex profiling
}
```

**使用方法**：

```bash
go tool pprof http://localhost:9090/debug/pprof/mutex
```

### 7. Allocs Profile（内存分配分析）

**用途**：分析内存分配情况。

**使用方法**：

```bash
go tool pprof http://localhost:9090/debug/pprof/allocs
```

### 8. Trace（执行追踪）

**用途**：追踪程序执行过程，分析延迟、调度等问题。

**使用方法**：

```bash
# 采集 5 秒的 trace
curl http://localhost:9090/debug/pprof/trace?seconds=5 > trace.out

# 使用 go tool trace 分析
go tool trace trace.out
```

## 🔍 实际使用场景

### 场景 1：分析压力测试期间的性能瓶颈

1. **启动服务**：
   ```bash
   go run core/cmd/main/main.go
   ```

2. **在另一个终端运行压力测试**：
   ```bash
   cd test/压力测试
   ./详细压力测试.sh
   ```

3. **采集 CPU profile**（在压力测试期间）：
   ```bash
   go tool pprof http://localhost:5173/debug/pprof/profile?seconds=30
   ```

4. **分析结果**：
   ```
   (pprof) top10
   (pprof) list <函数名>
   (pprof) web
   ```

### 场景 2：分析内存泄漏

1. **启动服务并运行一段时间**

2. **采集多个时间点的 heap profile**：
   ```bash
   # 第一次采集
   curl http://localhost:9090/debug/pprof/heap > heap1.prof
   
   # 等待一段时间（如 5 分钟）
   sleep 300
   
   # 第二次采集
   curl http://localhost:9090/debug/pprof/heap > heap2.prof
   ```

3. **对比分析**：
   ```bash
   go tool pprof -base heap1.prof heap2.prof
   ```

### 场景 3：分析 goroutine 泄漏

1. **采集 goroutine profile**：
   ```bash
   go tool pprof http://localhost:9090/debug/pprof/goroutine
   ```

2. **查看 goroutine 堆栈**：
   ```
   (pprof) top20
   (pprof) list <函数名>
   ```

## 📝 常用命令速查

### pprof 交互式命令

| 命令 | 说明 |
|------|------|
| `top` | 显示占用资源最多的函数 |
| `top10` | 显示前 10 个 |
| `top -cum` | 按累计值排序 |
| `list <函数名>` | 查看函数的具体代码和资源占用 |
| `web` | 生成调用图（需要 graphviz） |
| `svg` | 生成 SVG 格式的调用图 |
| `png` | 生成 PNG 格式的调用图 |
| `help` | 显示帮助信息 |
| `exit` 或 `quit` | 退出 |

### 命令行参数

```bash
# 指定采样时间（秒）
go tool pprof http://localhost:9090/debug/pprof/profile?seconds=60

# 指定输出格式
go tool pprof -http=:8080 http://localhost:9090/debug/pprof/heap  # Web UI
go tool pprof -svg http://localhost:9090/debug/pprof/heap > heap.svg  # SVG
go tool pprof -png http://localhost:9090/debug/pprof/heap > heap.png  # PNG
```

## 🌐 Web UI（推荐）

使用 Web UI 可以更直观地查看性能数据：

```bash
# 启动 Web UI（默认端口 8080）
go tool pprof -http=:8080 http://localhost:9090/debug/pprof/heap

# 然后在浏览器中访问 http://localhost:8080
```

Web UI 功能：
- **Top**：显示占用资源最多的函数
- **Graph**：可视化调用图
- **Flame Graph**：火焰图（最直观）
- **Peek**：查看函数调用关系
- **Source**：查看源代码

## 🔧 安装依赖

### Graphviz（用于生成调用图）

**macOS**：
```bash
brew install graphviz
```

**Ubuntu/Debian**：
```bash
sudo apt-get install graphviz
```

**Windows**：
下载安装包：https://graphviz.org/download/

## ⚠️ 注意事项

1. **生产环境**：建议通过配置控制是否启用 pprof，避免暴露性能数据
2. **性能影响**：pprof 采集会对性能有轻微影响，但通常可以忽略
3. **采样时间**：CPU profile 建议采集 30-60 秒，太短可能不准确
4. **内存占用**：heap profile 文件可能较大，注意磁盘空间

## 📚 参考资源

- [Go pprof 官方文档](https://pkg.go.dev/net/http/pprof)
- [Go 性能优化实战](https://github.com/golang/go/wiki/Performance)
- [Dave Cheney 的 pprof 教程](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)

