# SDK 指标收集方案分析

## 需求分析

### 目标
- ✅ **业务代码无感知**：业务开发者不需要写任何指标收集代码
- ✅ **自动收集常用指标**：QPS、平均耗时、内存、错误率等
- ✅ **Grafana 可视化**：可以直接在 Grafana 中查看所有指标
- ✅ **多租户支持**：指标需要区分租户、应用、版本

### 需要收集的指标

#### 1. 业务指标
- **QPS（每秒请求数）**：按路由、方法、租户统计
- **请求耗时**：平均耗时、P50、P95、P99
- **错误率**：按错误类型、路由统计
- **并发数**：当前正在处理的请求数

#### 2. 系统指标
- **内存使用**：当前内存、峰值内存、GC 统计
- **CPU 使用**：CPU 使用率
- **Goroutine 数量**：当前运行的 goroutine 数
- **GC 统计**：GC 次数、GC 耗时

#### 3. 应用指标
- **启动时间**：应用运行时长
- **版本信息**：当前应用版本
- **路由数量**：注册的路由总数

## 技术方案分析

### 方案对比

#### 方案 A：Prometheus + 内置 HTTP Server（推荐）

**架构**：
```
业务代码（无感知）
    ↓
SDK 中间件（自动拦截）
    ↓
Prometheus Metrics
    ↓
HTTP /metrics 端点
    ↓
Prometheus Server（抓取）
    ↓
Grafana（可视化）
```

**优点**：
- ✅ 标准方案，生态成熟
- ✅ 业务代码完全无感知
- ✅ 支持多维度标签（tenant, app, version, route, method）
- ✅ 自动暴露 HTTP 端点，Prometheus 自动抓取
- ✅ 支持 Histogram、Counter、Gauge 等多种指标类型

**缺点**：
- ⚠️ 需要额外的 HTTP 端口（但可以复用现有端口）
- ⚠️ 需要部署 Prometheus（但这是标准做法）

**实现复杂度**：⭐⭐（中等）

#### 方案 B：直接写入时序数据库

**架构**：
```
业务代码（无感知）
    ↓
SDK 中间件（自动拦截）
    ↓
直接写入 InfluxDB/TimescaleDB
    ↓
Grafana（可视化）
```

**优点**：
- ✅ 简单直接
- ✅ 不需要 Prometheus

**缺点**：
- ❌ 需要处理连接池、重试等
- ❌ 指标格式不标准
- ❌ 缺少 Prometheus 生态支持

**实现复杂度**：⭐⭐⭐（较高）

#### 方案 C：通过日志输出指标

**架构**：
```
业务代码（无感知）
    ↓
SDK 中间件（自动拦截）
    ↓
输出结构化日志（包含指标）
    ↓
Loki + Promtail（解析日志提取指标）
    ↓
Grafana（可视化）
```

**优点**：
- ✅ 复用现有日志系统

**缺点**：
- ❌ 日志不是为指标设计的，效率低
- ❌ 解析复杂，延迟高
- ❌ 不适合高频指标

**实现复杂度**：⭐⭐⭐⭐（高）

### 推荐方案：Prometheus + 内置 HTTP Server

## 详细设计

### 1. SDK 集成点分析

#### 当前 SDK 架构
```
NATS 消息
    ↓
handleMessageAsync() → handleMessage()
    ↓
handle() → 路由匹配 → handleFunc()
    ↓
业务代码
```

#### 指标收集插入点

**插入点 1：handleMessage()**
- ✅ 记录请求开始时间
- ✅ 增加 QPS 计数
- ✅ 增加并发数
- ✅ 捕获错误

**插入点 2：handle()**
- ✅ 记录路由、方法信息
- ✅ 记录处理耗时
- ✅ 记录错误类型

**插入点 3：应用启动时**
- ✅ 启动 HTTP metrics 服务器
- ✅ 注册系统指标收集器（内存、CPU、Goroutine）

### 2. 指标设计

#### 指标类型选择

| 指标 | 类型 | 说明 | 示例 |
|------|------|------|------|
| QPS | Counter | 累计请求数 | `http_requests_total{tenant="beiluo",app="test",route="/api/user",method="POST"} 1234` |
| 耗时 | Histogram | 请求耗时分布 | `http_request_duration_seconds{tenant="beiluo",route="/api/user",quantile="0.95"} 0.123` |
| 并发数 | Gauge | 当前并发数 | `http_requests_in_flight{tenant="beiluo",app="test"} 5` |
| 错误数 | Counter | 累计错误数 | `http_errors_total{tenant="beiluo",route="/api/user",error_type="validation"} 10` |
| 内存 | Gauge | 当前内存使用 | `go_memstats_alloc_bytes{tenant="beiluo",app="test"} 52428800` |
| Goroutine | Gauge | 当前 Goroutine 数 | `go_goroutines{tenant="beiluo",app="test"} 42` |

#### 标签设计

**必需标签**（用于多租户隔离）：
- `tenant`: 租户标识（如 "beiluo"）
- `app`: 应用标识（如 "test777"）
- `version`: 版本标识（如 "v15"）

**可选标签**（用于细粒度分析）：
- `route`: 路由路径（如 "/api/user"）
- `method`: HTTP 方法（如 "POST"）
- `error_type`: 错误类型（如 "validation", "panic"）
- `status`: 状态码（如 "200", "500"）

### 3. 实现方案

#### 3.1 创建 Metrics 包

**文件结构**：
```
sdk/agent-app/metrics/
  ├── metrics.go          # 指标定义和初始化
  ├── middleware.go       # 中间件（自动拦截）
  ├── system.go          # 系统指标收集
  └── server.go          # HTTP metrics 服务器
```

#### 3.2 指标定义

```go
// metrics/metrics.go
package metrics

var (
    // HTTP 请求总数（Counter）
    httpRequestsTotal *prometheus.CounterVec
    
    // HTTP 请求耗时（Histogram）
    httpRequestDuration *prometheus.HistogramVec
    
    // HTTP 当前并发数（Gauge）
    httpRequestsInFlight *prometheus.GaugeVec
    
    // HTTP 错误总数（Counter）
    httpErrorsTotal *prometheus.CounterVec
    
    // 系统指标（自动收集）
    goMemStats *prometheus.GaugeVec
    goGoroutines prometheus.Gauge
)
```

#### 3.3 中间件集成

**方案 A：在 handleMessage() 中集成（推荐）**

```go
// app/handle.go
func (a *App) handleMessage(msg *nats.Msg) {
    // 1. 解析请求
    var req dto.RequestAppReq
    // ... 解析逻辑 ...
    
    // 2. 开始指标收集（自动）
    startTime := time.Now()
    labels := metrics.GetLabels(req) // 自动获取标签
    
    // 3. 增加并发数
    metrics.IncConcurrent(labels)
    defer metrics.DecConcurrent(labels)
    
    // 4. 增加请求计数
    metrics.IncRequestTotal(labels)
    
    // 5. 处理请求
    resp, err := a.handle(&req)
    
    // 6. 记录耗时和错误
    duration := time.Since(startTime)
    metrics.ObserveDuration(labels, duration)
    if err != nil {
        metrics.IncErrorTotal(labels, err)
    }
    
    // 7. 发送响应
    // ...
}
```

**优点**：
- ✅ 完全无感知，业务代码不需要修改
- ✅ 自动收集所有请求的指标
- ✅ 可以获取完整的请求信息（route, method）

#### 3.4 HTTP Metrics 服务器

**方案 A：独立端口（推荐）**

```go
// metrics/server.go
func StartMetricsServer(port int) {
    http.Handle("/metrics", promhttp.Handler())
    go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
```

**方案 B：复用现有端口**

如果应用已经有 HTTP 服务器，可以复用：
```go
// 在现有 HTTP 服务器上添加路由
router.GET("/metrics", gin.WrapH(promhttp.Handler()))
```

**推荐**：使用独立端口，避免影响业务接口

#### 3.5 系统指标自动收集

```go
// metrics/system.go
func CollectSystemMetrics() {
    // 注册 Go 运行时指标
    prometheus.MustRegister(prometheus.NewGoCollector())
    prometheus.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
    
    // 自定义系统指标
    go func() {
        ticker := time.NewTicker(5 * time.Second)
        for range ticker.C {
            var m runtime.MemStats
            runtime.ReadMemStats(&m)
            
            labels := metrics.GetSystemLabels()
            metrics.SetMemoryUsage(labels, m.Alloc)
            metrics.SetGoroutineCount(runtime.NumGoroutine())
        }
    }()
}
```

### 4. 业务代码无感知实现

#### 关键点

1. **自动初始化**：在 `NewApp()` 时自动初始化 metrics
2. **自动拦截**：在 `handleMessage()` 中自动收集指标
3. **自动启动**：在 `Run()` 时自动启动 metrics 服务器
4. **零配置**：使用环境变量或默认值，无需业务代码配置

#### 实现示例

```go
// app/app.go
func NewApp() (*App, error) {
    // ... 现有初始化代码 ...
    
    // 自动初始化 metrics（业务代码无感知）
    if err := metrics.Init(env.User, env.App, env.Version); err != nil {
        logger.Warnf(context.Background(), "Failed to init metrics: %v", err)
        // 不返回错误，metrics 失败不影响业务
    }
    
    // ... 其他初始化 ...
}

func (a *App) Run() {
    // ... 现有代码 ...
    
    // 自动启动 metrics 服务器（业务代码无感知）
    metricsPort := os.Getenv("METRICS_PORT")
    if metricsPort == "" {
        metricsPort = "9090" // 默认端口
    }
    metrics.StartMetricsServer(metricsPort)
    
    // ... 其他启动逻辑 ...
}
```

### 5. Prometheus 配置

#### 服务发现

由于应用是动态创建的，需要动态服务发现：

**方案 A：基于文件的服务发现**

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'ai-agent-os-apps'
    file_sd_configs:
      - files:
          - '/etc/prometheus/apps/*.json'
        refresh_interval: 5s
```

**方案 B：基于 NATS 的服务发现**

应用启动时，通过 NATS 通知 Prometheus 有新应用：
```go
// 应用启动时
discovery.NotifyMetricsEndpoint(user, app, version, metricsPort)
```

**方案 C：静态配置 + 通配符**

如果应用端口固定，可以使用静态配置：
```yaml
scrape_configs:
  - job_name: 'ai-agent-os-apps'
    static_configs:
      - targets: ['localhost:9090'] # 每个应用都暴露在 9090
    relabel_configs:
      - source_labels: [__address__]
        regex: '(.+):(\d+)'
        target_label: __param_target
        replacement: '${1}:${2}'
```

### 6. Grafana 仪表板

#### 预定义面板

1. **QPS 面板**
   - 总 QPS
   - 按租户的 QPS
   - 按应用的 QPS
   - 按路由的 QPS

2. **耗时面板**
   - 平均耗时
   - P50、P95、P99 耗时
   - 按路由的耗时分布

3. **错误率面板**
   - 总错误率
   - 按错误类型的错误数
   - 按路由的错误率

4. **系统资源面板**
   - 内存使用趋势
   - CPU 使用率
   - Goroutine 数量
   - GC 统计

5. **多租户概览**
   - 所有租户的指标对比
   - 租户选择器

#### 查询示例

```promql
# 总 QPS
sum(rate(http_requests_total[5m]))

# 按租户的 QPS
sum(rate(http_requests_total[5m])) by (tenant)

# 平均耗时
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# 错误率
sum(rate(http_errors_total[5m])) / sum(rate(http_requests_total[5m])) * 100

# 当前并发数
sum(http_requests_in_flight)
```

## 实施步骤

### 阶段一：基础指标收集（1-2 天）

1. **创建 metrics 包**
   - 定义指标
   - 实现初始化函数
   - 实现中间件函数

2. **集成到 SDK**
   - 在 `handleMessage()` 中集成
   - 在 `NewApp()` 中自动初始化
   - 在 `Run()` 中自动启动服务器

3. **测试验证**
   - 验证指标收集
   - 验证 HTTP 端点
   - 验证标签正确性

### 阶段二：系统指标收集（1 天）

1. **实现系统指标收集**
   - 内存指标
   - CPU 指标
   - Goroutine 指标
   - GC 指标

2. **测试验证**
   - 验证系统指标准确性

### 阶段三：Prometheus 集成（1 天）

1. **部署 Prometheus**
   - 配置服务发现
   - 配置抓取规则

2. **测试验证**
   - 验证 Prometheus 能抓取指标
   - 验证指标正确性

### 阶段四：Grafana 仪表板（1 天）

1. **创建仪表板**
   - 创建预定义面板
   - 配置多租户支持

2. **测试验证**
   - 验证可视化效果
   - 验证查询性能

## 性能影响分析

### 指标收集开销

| 操作 | 开销 | 说明 |
|------|------|------|
| Counter.Inc() | < 1μs | 原子操作，极快 |
| Histogram.Observe() | < 5μs | 需要计算分桶，稍慢 |
| Gauge.Set() | < 1μs | 原子操作，极快 |
| HTTP /metrics | 1-5ms | 序列化指标，可接受 |

### 总体影响

- **CPU 开销**：< 1%（指标收集是轻量级操作）
- **内存开销**：< 10 MB（指标数据本身很小）
- **网络开销**：可忽略（Prometheus 定期抓取，不是实时推送）

**结论**：性能影响极小，可以忽略不计。

## 注意事项

### 1. 标签基数控制

**避免高基数标签**：
- ❌ 不要使用 `trace_id`、`request_id` 作为标签
- ❌ 不要使用用户 ID 作为标签
- ✅ 只使用低基数标签：tenant, app, version, route, method

### 2. 指标保留策略

- **指标数据**：Prometheus 默认保留 15 天（可配置）
- **长期存储**：可以使用 Thanos 或 Cortex 进行长期存储

### 3. 多租户隔离

- **标签隔离**：通过 `tenant` 标签区分租户
- **Grafana 权限**：配置基于租户的访问控制
- **Prometheus 查询**：使用 `tenant="xxx"` 过滤

### 4. 错误处理

- **Metrics 失败不影响业务**：如果 metrics 初始化失败，只记录警告，不阻止应用启动
- **优雅降级**：如果 Prometheus 不可用，指标仍然收集，只是无法查询

## 总结

### 方案优势

1. ✅ **完全无感知**：业务代码不需要任何修改
2. ✅ **自动收集**：所有指标自动收集，无需手动埋点
3. ✅ **标准方案**：使用 Prometheus 标准，生态成熟
4. ✅ **多租户支持**：通过标签完美支持多租户
5. ✅ **性能影响小**：开销 < 1%，可忽略不计
6. ✅ **易于扩展**：可以轻松添加新指标

### 实施建议

1. **先实现基础指标**：QPS、耗时、错误率
2. **逐步添加系统指标**：内存、CPU、Goroutine
3. **最后完善可视化**：创建 Grafana 仪表板

### 预期效果

- ✅ 业务开发者：**零感知**，正常写业务代码
- ✅ 运维人员：**一键查看**所有应用的指标
- ✅ 多租户：**自动隔离**，每个租户看到自己的指标
- ✅ 问题排查：**快速定位**性能瓶颈和错误







