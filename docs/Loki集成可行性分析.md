# Loki 集成可行性分析

## 项目现状分析

### 1. 多租户架构
- **应用存储路径**：`namespace/{user}/{app}/`
- **日志文件路径**：`namespace/{user}/{app}/workplace/logs/{user}_{app}_{version}.log`
- **应用动态创建**：应用在运行时动态创建，路径不固定
- **容器化运行**：应用运行在 Podman 容器中

### 2. 当前日志系统
- **日志库**：使用 `zap` 日志库
- **日志输出**：文件输出（使用 `lumberjack` 进行日志轮转）
- **日志位置**：每个应用实例有独立的日志文件
- **日志格式**：JSON 格式（生产环境）或控制台格式（开发环境）

### 3. 日志文件示例
从项目结构可以看到：
```
namespace/
  beiluo/
    test777/
      workplace/
        logs/
          beiluo_test777_v1.log
          beiluo_test777_v10.log
          ...
    testapi/
      workplace/
        logs/
          beiluo_testapi_v1.log
          beiluo_testapi_v2.log
          ...
```

## Loki 集成可行性评估

### ✅ **完全可行，且非常适合**

### 1. 关于"新增日志需要重启"的担忧

**结论：这个担忧是误解，Loki + Promtail 完全支持动态发现，无需重启。**

#### Promtail 动态发现机制
Promtail 支持多种服务发现方式，可以自动发现新的日志文件：

1. **文件系统发现（推荐）**
   - 使用 `file_sd_configs` 或 `discovery.file`
   - 可以监控整个目录树，自动发现新文件
   - 支持通配符模式匹配

2. **Docker/Podman 发现**
   - 使用 `docker_sd_configs` 或 `podman_sd_configs`
   - 自动发现容器日志
   - 容器启动时自动开始收集

3. **Kubernetes 发现**
   - 如果未来迁移到 K8s，可以使用 `kubernetes_sd_configs`

### 2. 推荐方案：基于文件系统发现

#### 方案优势
- ✅ **无需重启**：Promtail 定期扫描目录，自动发现新日志文件
- ✅ **路径灵活**：支持通配符，匹配所有租户和应用的日志
- ✅ **多租户隔离**：通过标签（labels）区分不同租户和应用
- ✅ **版本管理**：可以区分不同版本的日志

#### Promtail 配置示例

```yaml
server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: ai-agent-os-apps
    static_configs:
      - targets:
          - localhost
        labels:
          job: ai-agent-os
          __path__: /path/to/namespace/*/*/workplace/logs/*.log
    
    # 使用 pipeline_stages 提取标签
    pipeline_stages:
      # 从文件路径提取 user, app, version
      - regex:
          expression: 'namespace/(?P<user>[^/]+)/(?P<app>[^/]+)/workplace/logs/(?P<user2>[^_]+)_(?P<app2>[^_]+)_(?P<version>v\d+)\.log'
      - labels:
          user: user
          app: app
          version: version
      # 解析 JSON 日志（如果是 JSON 格式）
      - json:
          expressions:
            level: level
            msg: msg
            ts: ts
      - labels:
          level: level
      - timestamp:
          source: ts
          format: RFC3339Nano
```

### 3. 更优方案：容器日志直接收集

由于应用运行在 Podman 容器中，可以考虑直接收集容器日志：

#### 方案 A：收集容器 stdout/stderr
```yaml
scrape_configs:
  - job_name: podman-containers
    podman_sd_configs:
      - host: unix:///run/podman/podman.sock
        refresh_interval: 5s
    relabel_configs:
      - source_labels: [__meta_podman_container_name]
        regex: '(.+)'
        target_label: container_name
      - source_labels: [__meta_podman_container_id]
        target_label: container_id
```

#### 方案 B：收集挂载的日志文件（推荐）
结合文件系统发现，收集容器内挂载到宿主机的日志文件：

```yaml
scrape_configs:
  - job_name: ai-agent-os-logs
    static_configs:
      - targets:
          - localhost
        labels:
          job: ai-agent-os
          __path__: /path/to/namespace/*/*/workplace/logs/*.log
    
    pipeline_stages:
      # 从路径提取元数据
      - regex:
          expression: 'namespace/(?P<user>[^/]+)/(?P<app>[^/]+)/workplace/logs/(?P<user2>[^_]+)_(?P<app2>[^_]+)_(?P<version>v\d+)\.log'
      - labels:
          tenant: user
          application: app
          version: version
      # 解析日志内容
      - json:
          expressions:
            level: level
            msg: msg
            caller: caller
      - labels:
          log_level: level
```

### 4. 多租户标签策略

#### 标签设计
```yaml
labels:
  tenant: beiluo          # 租户标识
  application: test777     # 应用标识
  version: v15            # 版本标识
  log_level: info        # 日志级别
  job: ai-agent-os        # 作业标识
```

#### 查询示例
```logql
# 查询特定租户的所有日志
{tenant="beiluo"}

# 查询特定应用的日志
{tenant="beiluo", application="test777"}

# 查询特定版本的日志
{tenant="beiluo", application="test777", version="v15"}

# 查询错误日志
{tenant="beiluo", log_level="error"}

# 时间范围查询
{tenant="beiluo"} | json | line_format "{{.msg}}"
```

## 实施建议

### 阶段一：基础集成（推荐先做）
1. **部署 Loki**
   - 使用 Docker Compose 或直接部署
   - 配置持久化存储

2. **部署 Promtail**
   - 配置文件系统发现
   - 监控 `namespace/*/*/workplace/logs/*.log`
   - 提取租户、应用、版本标签

3. **验证**
   - 创建新应用，验证自动发现
   - 检查日志是否正常收集
   - 测试查询功能

### 阶段二：优化（可选）
1. **日志格式统一**
   - 确保所有日志使用 JSON 格式
   - 统一时间戳格式

2. **性能优化**
   - 配置日志保留策略
   - 设置合适的采样率（如果需要）

3. **监控告警**
   - 集成 Grafana 进行可视化
   - 配置告警规则

## 技术细节

### 1. 路径匹配模式
```yaml
# 匹配所有租户和应用的日志
__path__: /path/to/namespace/*/*/workplace/logs/*.log

# 或者更精确的匹配
__path__: /path/to/namespace/*/*/workplace/logs/*_v*.log
```

### 2. 动态发现配置
```yaml
# Promtail 会自动扫描目录，默认每 5 秒扫描一次
# 可以通过 refresh_interval 调整
file_sd_configs:
  - files:
      - '/etc/promtail/apps/*.json'
    refresh_interval: 5s
```

### 3. 日志解析
由于当前使用 zap 日志库，生产环境输出 JSON 格式，可以直接解析：

```yaml
pipeline_stages:
  - json:
      expressions:
        level: level
        msg: msg
        ts: ts
        caller: caller
  - timestamp:
      source: ts
      format: RFC3339Nano
  - labels:
      log_level: level
```

## 性能影响分析

### 1. 性能开销评估

#### ✅ **对应用本身：几乎无影响**
- **读取方式**：Promtail 通过**文件系统读取**日志文件，与应用写入日志**完全解耦**
- **无侵入性**：应用继续正常写入日志文件，Promtail 只是"旁观者"
- **资源隔离**：Promtail 运行在独立进程/容器中，不影响应用性能
- **结论**：**应用性能影响 < 1%**（几乎可忽略）

#### ⚠️ **Promtail 资源消耗**

**CPU 开销**：
- **文件扫描**：每 5 秒扫描一次目录（可调整），CPU 开销极低
- **日志解析**：JSON 解析和正则匹配，单核可处理 **10-50 MB/s** 日志
- **典型消耗**：**50-200 MB 内存，0.1-1 CPU 核心**

**内存开销**：
- **文件句柄**：每个日志文件约占用 1-2 KB 内存
- **缓冲区**：默认 256 KB 缓冲区/文件
- **估算**：100 个应用 ≈ 100-200 MB 内存

**磁盘 I/O**：
- **读取操作**：顺序读取日志文件，I/O 效率高
- **影响**：与日志写入共享磁盘，但读取优先级较低
- **优化**：使用 SSD 可显著提升性能

#### ⚠️ **Loki 资源消耗**

**CPU 开销**：
- **索引构建**：基于标签的索引，CPU 开销低
- **查询处理**：LogQL 查询，单核可处理 **1000+ QPS**
- **典型消耗**：**1-2 CPU 核心**

**内存开销**：
- **索引缓存**：默认 1 GB（可配置）
- **查询缓存**：可配置缓存大小
- **典型消耗**：**1-4 GB 内存**

**存储开销**：
- **压缩比**：Loki 使用压缩存储，压缩比约 **10:1**
- **示例**：100 GB 原始日志 ≈ 10 GB Loki 存储
- **保留策略**：建议保留 7-30 天

### 2. 性能优化策略

#### 优化 1：调整扫描频率
```yaml
# 降低扫描频率，减少 CPU 开销
scrape_configs:
  - job_name: ai-agent-os-apps
    # 从默认 5 秒调整为 10-30 秒
    refresh_interval: 10s
```

#### 优化 2：限制并发读取
```yaml
# 限制同时读取的文件数量
limits_config:
  max_concurrent_tail: 10  # 默认无限制
```

#### 优化 3：采样策略（高日志量场景）
```yaml
# 对低级别日志进行采样
pipeline_stages:
  - json:
      expressions:
        level: level
  # 只收集 100% 的 error，10% 的 info
  - drop:
      expression: 'level == "info"'
      drop_counter_reason: "sampled_info_logs"
      drop_ratio: 0.9  # 丢弃 90% 的 info 日志
```

#### 优化 4：批量发送
```yaml
# 批量发送，减少网络请求
clients:
  - url: http://loki:3100/loki/api/v1/push
    batchwait: 1s      # 等待 1 秒批量发送
    batchsize: 1048576 # 1 MB 批量大小
    timeout: 10s
```

#### 优化 5：限制标签基数
```yaml
# ❌ 避免使用高基数标签（如 trace_id, request_id）
# ✅ 只使用低基数标签（tenant, app, version, level）
pipeline_stages:
  - labels:
      tenant: user      # ✅ 低基数（几十个租户）
      app: app          # ✅ 低基数（几百个应用）
      version: version  # ✅ 低基数（几个版本）
      # ❌ 不要添加：trace_id, request_id 等高基数标签
```

### 3. 性能基准测试

#### 典型场景评估

**场景 1：小规模（10 个应用）**
- 日志量：~1 GB/天
- Promtail：50 MB 内存，0.1 CPU
- Loki：500 MB 内存，0.5 CPU
- **总开销：< 1 GB 内存，< 1 CPU**

**场景 2：中规模（100 个应用）**
- 日志量：~10 GB/天
- Promtail：200 MB 内存，0.5 CPU
- Loki：2 GB 内存，1 CPU
- **总开销：~2.5 GB 内存，~1.5 CPU**

**场景 3：大规模（1000 个应用）**
- 日志量：~100 GB/天
- Promtail：1 GB 内存，2 CPU
- Loki：4 GB 内存，2 CPU
- **总开销：~5 GB 内存，~4 CPU**
- **建议**：使用采样策略，降低日志量

### 4. 与现有系统对比

#### 当前系统（文件日志）
- **应用开销**：直接写入文件，开销低
- **查询开销**：需要 `grep`、`tail` 等工具，效率低
- **存储开销**：未压缩，占用空间大
- **管理开销**：需要手动管理日志轮转

#### Loki 系统
- **应用开销**：**相同**（仍然写入文件）
- **查询开销**：**显著提升**（LogQL 查询，秒级响应）
- **存储开销**：**减少 90%**（压缩存储）
- **管理开销**：**自动化**（自动发现、自动清理）

**结论**：Loki 在**几乎不增加应用开销**的情况下，**大幅提升查询效率和管理便利性**。

### 5. 性能监控建议

#### 监控 Promtail
```yaml
# Promtail 内置 metrics
# 访问 http://localhost:9080/metrics
# 关键指标：
# - promtail_read_bytes_total：读取的字节数
# - promtail_read_lines_total：读取的行数
# - promtail_positions_errors_total：位置文件错误
```

#### 监控 Loki
```yaml
# Loki 内置 metrics
# 访问 http://localhost:3100/metrics
# 关键指标：
# - loki_ingester_chunks_created_total：创建的块数
# - loki_ingester_chunks_stored_bytes_total：存储的字节数
# - loki_querier_request_duration_seconds：查询延迟
```

## 注意事项

### 1. 性能考虑
- **日志量**：多租户系统可能产生大量日志，需要合理配置保留策略
- **索引**：合理使用标签，避免高基数标签（如 trace_id）
- **存储**：Loki 使用对象存储，需要规划存储容量
- **采样**：高日志量场景建议使用采样策略

### 2. 安全考虑
- **多租户隔离**：确保不同租户的日志不能互相访问
- **访问控制**：在 Grafana 中配置基于租户的访问控制
- **日志脱敏**：敏感信息需要脱敏处理

### 3. 兼容性
- **现有日志**：保留现有文件日志作为备份
- **平滑迁移**：可以同时使用文件日志和 Loki，逐步迁移

## Loki 工作机制详解

### 工作流程

```
应用写入日志文件
    ↓
[日志文件] namespace/{user}/{app}/workplace/logs/{user}_{app}_{version}.log
    ↓
Promtail 读取日志文件（通过文件系统，不影响应用写入）
    ↓
Promtail 解析、提取标签、批量发送
    ↓
Loki 接收、压缩、存储
    ↓
[Loki 存储] 压缩后的日志数据（压缩比约 10:1）
```

### 关键点说明

1. **日志读取**：
   - ✅ Promtail **读取**现有的日志文件（不修改、不删除）
   - ✅ 应用**继续正常写入**日志文件
   - ✅ 两者**完全解耦**，互不影响

2. **日志存储**：
   - ✅ Loki **会存储**日志（压缩格式）
   - ✅ 压缩比约 **10:1**（100 GB 原始日志 ≈ 10 GB Loki 存储）
   - ✅ 原始日志文件**可以保留**（作为备份），也可以**删除**（节省空间）

### 存储策略选择

#### 方案 A：保留原始文件 + Loki（推荐用于过渡期）
```
原始日志文件：100 GB
Loki 存储：10 GB
总存储：110 GB
```
**优点**：
- 双重备份，更安全
- 可以随时回退到文件日志
- 适合过渡期

**缺点**：
- 存储空间增加（但 Loki 是压缩的）

#### 方案 B：只保留 Loki（推荐用于生产环境）
```
原始日志文件：删除（或只保留最近 1-2 天）
Loki 存储：10 GB
总存储：10 GB
```
**优点**：
- 存储空间减少 **90%**
- 统一管理，查询更方便
- 自动清理，无需手动管理

**缺点**：
- 依赖 Loki 可用性
- 需要确保 Loki 备份策略

#### 方案 C：混合策略（推荐）
```
原始日志文件：保留最近 7 天（用于快速排查）
Loki 存储：保留 30 天（用于历史查询）
总存储：约 20 GB（假设每天 1 GB）
```
**优点**：
- 平衡存储和可用性
- 近期日志快速访问
- 历史日志统一查询

### 存储空间对比

假设每天产生 10 GB 日志，保留 7 天：

| 方案 | 原始文件 | Loki 存储 | 总存储 | 节省 |
|------|---------|----------|--------|------|
| 只保留文件 | 70 GB | 0 GB | 70 GB | - |
| 文件 + Loki | 70 GB | 7 GB | 77 GB | -10% |
| 只保留 Loki | 0 GB | 7 GB | 7 GB | **90%** |
| 混合策略 | 7 GB (1天) | 7 GB (7天) | 14 GB | **80%** |

**结论**：即使保留原始文件，Loki 的压缩存储也能显著节省空间。

### 日志清理策略

#### 原始日志文件清理
你的项目已经使用 `lumberjack` 进行日志轮转：
```go
MaxSize:    100,  // 单个文件最大 100 MB
MaxBackups: 3,    // 保留 3 个备份文件
MaxAge:     7,    // 保留 7 天
Compress:   true, // 压缩旧文件
```

#### Loki 清理策略
```yaml
# Loki 配置
compactor:
  retention_enabled: true
  retention_period: 168h  # 保留 7 天

limits_config:
  retention_period: 168h  # 保留 7 天
```

**建议**：
- 原始文件：保留 3-7 天（用于快速排查）
- Loki：保留 7-30 天（用于历史查询和分析）

## 结论

✅ **Loki 完全适合你的项目**

1. **动态发现**：Promtail 支持自动发现新日志文件，无需重启
2. **多租户友好**：通过标签可以完美支持多租户隔离
3. **路径灵活**：通配符匹配可以处理动态路径
4. **容器集成**：可以很好地与 Podman 容器集成
5. **存储优化**：压缩存储可节省 90% 空间

**建议**：
- 采用基于文件系统发现的方案
- 配置 Promtail 监控 `namespace/*/*/workplace/logs/*.log`
- 通过 pipeline 提取租户、应用、版本等标签
- **存储策略**：建议使用混合策略（原始文件保留 3-7 天，Loki 保留 7-30 天）

## 参考资源

- [Loki 官方文档](https://grafana.com/docs/loki/latest/)
- [Promtail 配置文档](https://grafana.com/docs/loki/latest/clients/promtail/configuration/)
- [LogQL 查询语言](https://grafana.com/docs/loki/latest/logql/)

