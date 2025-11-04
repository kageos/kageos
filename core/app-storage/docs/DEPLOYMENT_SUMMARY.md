# App Storage 部署总结

## ✅ 已完成

### 1. 基础功能

| 功能 | 状态 | 说明 |
|------|------|------|
| **MinIO 部署** | ✅ | Podman 容器运行，端口 9000/9001 |
| **存储服务** | ✅ | app-storage 运行在 8083 端口 |
| **上传凭证** | ✅ | 预签名 URL，客户端直接上传 |
| **下载链接** | ✅ | 预签名 URL + HTTP 缓存头 |
| **文件删除** | ✅ | 单个删除 + 批量删除 |
| **存储统计** | ✅ | 按租户/应用/函数统计 |

### 2. 多租户架构

| 功能 | 状态 | 说明 |
|------|------|------|
| **路径隔离** | ✅ | `{router}/{date}/{uuid}.{ext}` |
| **精确统计** | ✅ | 任意粒度统计（租户/应用/函数） |
| **批量管理** | ✅ | 按函数批量删除文件 |
| **审计追踪** | ✅ | 每个文件都有明确归属 |

### 3. 性能优化

| 功能 | 状态 | 说明 |
|------|------|------|
| **HTTP 缓存** | ✅ 已启用 | `Cache-Control: max-age=31536000` |
| **秒下载** | ✅ 已启用 | 浏览器缓存 1 年，重复访问秒加载 |
| **数据库表** | ✅ 已创建 | `file_metadata`, `file_references` |
| **秒传功能** | 🔮 已预留 | 配置开关 `deduplication.enabled: false` |

### 4. 数据库

| 表名 | 状态 | 用途 |
|------|------|------|
| `file_metadata` | ✅ 已创建 | 记录文件 hash 和物理存储位置 |
| `file_references` | ✅ 已创建 | 记录文件引用关系（哪个函数用了哪些文件） |

**说明**：表已创建，但当前秒传功能未启用（`deduplication.enabled: false`）

## 🎯 服务状态

### MinIO

```bash
# 状态
podman ps | grep minio

# 控制台
http://localhost:9001
用户名: minioadmin
密码: minioadmin123
```

### App Storage

```bash
# 状态
curl http://localhost:8083/health

# Swagger 文档
http://localhost:8083/swagger/index.html

# 日志
tail -f logs/app-storage.log
```

## 📁 文件存储结构

```
MinIO Bucket: ai-agent-os
├── luobei/                              # 租户
│   ├── test88888/                       # 应用
│   │   ├── tools/
│   │   │   ├── cashier_desk/            # 函数
│   │   │   │   ├── 2025/11/03/
│   │   │   │   │   ├── xxx-xxx-xxx.jpg
│   │   │   │   │   └── yyy-yyy-yyy.pdf
│   │   │   │   └── 2025/11/04/
│   │   │   │       └── zzz-zzz-zzz.png
│   │   │   └── another_function/
│   │   └── crm/
│   │       └── ticket/
│   └── another_app/
└── another_tenant/
```

## 🔧 配置文件

### app-storage.yaml

```yaml
server:
  port: 8083

minio:
  endpoint: "localhost:9000"
  default_bucket: "ai-agent-os"
  
  # 秒传功能（预留）
  deduplication:
    enabled: false             # ✅ 未启用（按需开启）
  
  # 缓存控制
  cache:
    enabled: true              # ✅ 已启用
    max_age: 31536000          # 浏览器缓存 1 年

# 数据库（共用 app-server 的数据库）
db:
  name: "app_db"               # ✅ 表已自动创建
```

## 🚀 API 端点

| API | 方法 | 说明 |
|-----|------|------|
| `/api/v1/storage/upload_token` | POST | 获取上传凭证 |
| `/api/v1/storage/download/:key` | GET | 获取下载链接 |
| `/api/v1/storage/files/:key` | DELETE | 删除文件 |
| `/api/v1/storage/files/:key/info` | GET | 获取文件信息 |
| `/api/v1/storage/stats?router=xxx` | GET | 存储统计 |
| `/api/v1/storage/files?router=xxx` | GET | 列举文件 |
| `/api/v1/storage/batch_delete` | POST | 批量删除 |

## 📊 性能指标

### HTTP 缓存效果

```
首次下载：
  ├─ 请求服务器 → MinIO
  ├─ 响应时间：~100ms
  └─ 流量消耗：文件大小

再次下载（秒下载）：
  ├─ 直接从浏览器缓存读取
  ├─ 响应时间：0ms
  └─ 流量消耗：0
```

### 预期秒传效果（未来启用）

假设重复率 30%：

```
无去重：
  10,000 文件 × 5MB = 50GB

启用去重：
  10,000 文件 × 5MB × (1 - 30%) = 35GB
  节省：15GB (30%)
```

## 🔮 未来扩展路径

### 阶段 1：当前状态（✅ 已完成）

- [x] 基础存储服务
- [x] 多租户隔离
- [x] HTTP 缓存（秒下载）
- [x] 数据库表预留

### 阶段 2：启用秒传（按需）

触发条件：
- 文件数量 > 10,000
- 重复上传比例 > 20%
- 存储成本 > $100/月

实现步骤：
1. 修改配置：`deduplication.enabled: true`
2. 前端集成 hash 计算
3. 部署新版本
4. 监控去重效果

### 阶段 3：高级功能（未来）

- [ ] SeaweedFS 支持（Apache 2.0）
- [ ] CDN 加速
- [ ] 图片处理（缩略图、水印）
- [ ] 视频转码
- [ ] 文件预览

## ⚠️ 注意事项

### 1. 数据库连接

当前使用 `app_db` 数据库（共用 app-server），表已自动创建：
- `file_metadata`（文件元数据）
- `file_references`（文件引用）

### 2. 秒传功能

**当前状态**：未启用（`deduplication.enabled: false`）

**原因**：
- 基础设施已预留（数据库表、DTO 字段、配置开关）
- 等待实际使用场景数据，按需启用
- 避免过早优化

**启用时机**：
- 重复上传比例 > 20%
- 或存储成本压力较大

### 3. HTTP 缓存

**当前状态**：已启用（`cache.enabled: true`）

**效果**：
- 下载链接自动添加 `Cache-Control: max-age=31536000`
- 浏览器缓存文件 1 年
- 再次访问同一文件，秒加载（0ms）

### 4. AGPLv3 隔离

通过 S3 API 调用 MinIO，无代码感染：

```
app-storage (BSL 1.1)
    ↓ HTTP (S3 API)
MinIO (AGPLv3, 独立进程)
```

## 📝 命令速查

```bash
# 启动 MinIO
bash scripts/podman/minio.sh

# 启动 app-storage
bash scripts/start-app-storage.sh

# 查看日志
tail -f logs/app-storage.log

# 测试服务
curl http://localhost:8083/health

# 查看数据库表
# （需要数据库客户端或容器访问）
```

## 🎉 总结

### 已完成 ✅

1. ✅ **MinIO 部署**：Podman 容器运行
2. ✅ **存储服务**：app-storage 正常运行
3. ✅ **多租户隔离**：按 router 分类存储
4. ✅ **HTTP 缓存**：秒下载已启用
5. ✅ **数据库表**：秒传基础设施已预留
6. ✅ **完整文档**：架构设计、API 示例、多租户设计

### 路径畅通 🛣️

- 当前架构完全支持未来的秒传功能
- 数据库表已创建，无需重构
- 配置开关已预留，随时可启用
- 前端 DTO 已包含 hash 字段

**结论**：基础设施完整，未来扩展无障碍！🚀

