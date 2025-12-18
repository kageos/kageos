# Loki 性能测试指南

## 快速性能评估

### 1. 资源消耗预估

根据你的应用规模，参考以下资源消耗：

| 应用数量 | 日志量/天 | Promtail 资源 | Loki 资源 | 总资源 |
|---------|----------|--------------|----------|--------|
| 10 个   | ~1 GB    | 50 MB, 0.1 CPU | 500 MB, 0.5 CPU | < 1 GB, < 1 CPU |
| 100 个  | ~10 GB   | 200 MB, 0.5 CPU | 2 GB, 1 CPU | ~2.5 GB, ~1.5 CPU |
| 1000 个 | ~100 GB  | 1 GB, 2 CPU | 4 GB, 2 CPU | ~5 GB, ~4 CPU |

### 2. 对应用性能的影响

**结论：几乎无影响（< 1%）**

- ✅ Promtail 通过**文件系统读取**日志，与应用写入**完全解耦**
- ✅ 应用继续正常写入日志文件，Promtail 只是"旁观者"
- ✅ Promtail 运行在独立进程/容器中，资源隔离

## 性能测试步骤

### 步骤 1：监控当前系统资源

```bash
# 查看当前 CPU 和内存使用
top
# 或
htop

# 查看磁盘 I/O
iostat -x 1

# 查看文件系统使用
df -h
```

### 步骤 2：部署 Loki + Promtail（测试环境）

```bash
# 使用 Docker Compose 快速部署
docker-compose up -d
```

### 步骤 3：监控 Promtail 资源消耗

```bash
# 查看 Promtail 容器资源使用
docker stats promtail

# 或查看 Promtail metrics
curl http://localhost:9080/metrics | grep promtail
```

**关键指标**：
- `promtail_read_bytes_total`：读取的字节数
- `promtail_read_lines_total`：读取的行数
- `promtail_positions_errors_total`：位置文件错误数

### 步骤 4：监控 Loki 资源消耗

```bash
# 查看 Loki 容器资源使用
docker stats loki

# 或查看 Loki metrics
curl http://localhost:3100/metrics | grep loki
```

**关键指标**：
- `loki_ingester_chunks_created_total`：创建的块数
- `loki_ingester_chunks_stored_bytes_total`：存储的字节数
- `loki_querier_request_duration_seconds`：查询延迟

### 步骤 5：压力测试

#### 测试 1：日志写入性能

```bash
# 模拟大量日志写入
for i in {1..1000}; do
  echo "{\"level\":\"info\",\"msg\":\"test message $i\",\"ts\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}" >> /path/to/namespace/test/app/workplace/logs/test.log
done
```

观察：
- Promtail 是否能及时收集
- CPU 和内存使用是否正常
- 是否有日志丢失

#### 测试 2：查询性能

```bash
# 使用 LogQL 查询
curl -G -s "http://localhost:3100/loki/api/v1/query_range" \
  --data-urlencode 'query={job="ai-agent-os"}' \
  --data-urlencode 'start=1640995200' \
  --data-urlencode 'end=1641081600' \
  --data-urlencode 'limit=1000' | jq
```

观察：
- 查询响应时间
- CPU 使用情况
- 内存使用情况

## 性能优化检查清单

### ✅ Promtail 优化

- [ ] 调整扫描频率（`refresh_interval`）
- [ ] 限制并发读取（`max_concurrent_tail`）
- [ ] 配置批量发送（`batchwait`, `batchsize`）
- [ ] 避免高基数标签
- [ ] 考虑采样策略（高日志量场景）

### ✅ Loki 优化

- [ ] 配置合理的保留策略（`retention_period`）
- [ ] 设置查询限制（`max_query_series`, `query_timeout`）
- [ ] 启用查询缓存（`query_range.results_cache`）
- [ ] 配置压缩策略（`compactor`）
- [ ] 监控存储使用情况

### ✅ 系统优化

- [ ] 使用 SSD 存储（提升 I/O 性能）
- [ ] 确保足够的磁盘空间
- [ ] 监控网络带宽（Promtail → Loki）
- [ ] 配置日志轮转（避免单个文件过大）

## 性能问题排查

### 问题 1：Promtail CPU 使用率高

**可能原因**：
- 扫描频率过高
- 日志文件过多
- 正则表达式过于复杂

**解决方案**：
```yaml
# 降低扫描频率
refresh_interval: 30s

# 限制并发读取
limits_config:
  max_concurrent_tail: 20
```

### 问题 2：Loki 查询慢

**可能原因**：
- 查询时间范围过大
- 标签基数过高
- 存储性能瓶颈

**解决方案**：
```yaml
# 限制查询范围
limits_config:
  max_query_length: 24h
  max_query_series: 500
```

### 问题 3：日志丢失

**可能原因**：
- Promtail 处理速度跟不上日志写入速度
- 网络问题导致发送失败
- Loki 存储空间不足

**解决方案**：
```yaml
# 增加批量大小和缓冲区
clients:
  - batchsize: 2097152  # 2 MB
    batchwait: 2s
```

## 性能基准测试脚本

```bash
#!/bin/bash
# 性能测试脚本

echo "=== Loki 性能测试 ==="

# 1. 测试日志写入性能
echo "1. 测试日志写入性能..."
start_time=$(date +%s)
for i in {1..10000}; do
  echo "{\"level\":\"info\",\"msg\":\"test $i\",\"ts\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}" >> /tmp/test.log
done
end_time=$(date +%s)
echo "写入 10000 条日志耗时: $((end_time - start_time)) 秒"

# 2. 测试查询性能
echo "2. 测试查询性能..."
start_time=$(date +%s)
curl -G -s "http://localhost:3100/loki/api/v1/query_range" \
  --data-urlencode 'query={job="ai-agent-os"}' \
  --data-urlencode 'start='$(date -d '1 hour ago' +%s) \
  --data-urlencode 'end='$(date +%s) \
  --data-urlencode 'limit=1000' > /dev/null
end_time=$(date +%s)
echo "查询耗时: $((end_time - start_time)) 秒"

# 3. 查看资源使用
echo "3. 查看资源使用..."
docker stats --no-stream promtail loki

echo "=== 测试完成 ==="
```

## 监控仪表板配置

在 Grafana 中创建性能监控仪表板，监控以下指标：

1. **Promtail 指标**
   - 读取速率（bytes/s, lines/s）
   - 错误率
   - 文件数量
   - CPU 和内存使用

2. **Loki 指标**
   - 摄入速率（bytes/s）
   - 查询延迟
   - 存储使用
   - 错误率

3. **系统指标**
   - 磁盘 I/O
   - 网络带宽
   - CPU 和内存使用

## 结论

**性能影响总结**：
- ✅ **对应用性能**：几乎无影响（< 1%）
- ⚠️ **额外资源消耗**：根据应用规模，约 1-5 GB 内存，1-4 CPU 核心
- ✅ **性能收益**：查询效率提升 10-100 倍，存储空间减少 90%

**建议**：
1. 从小规模开始测试（10-20 个应用）
2. 逐步扩展到全部应用
3. 根据实际使用情况调整配置
4. 定期监控资源使用情况














