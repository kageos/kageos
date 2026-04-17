# App Storage - 存储服务

基于 MinIO 的对象存储服务，提供文件上传、下载、删除等功能。支持多租户隔离、精确统计和未来的秒传功能。

## 架构说明

遵循标准三层架构：

```
app-storage/
├── api/v1/          # HTTP Handler 层（处理请求响应）
│   └── storage.go   # 存储相关 API
├── service/         # 业务逻辑层（纯 Go 代码）
│   └── storage_service.go
├── model/           # 数据模型层
│   ├── file.go      # 文件元数据模型（预留，用于秒传）
│   └── init.go      # 数据库初始化
├── server/          # 服务器层
│   ├── server.go    # 服务器初始化
│   └── router.go    # 路由注册
├── cmd/app/         # 程序入口
│   └── main.go
└── docs/            # 文档
    ├── MULTI_TENANT_DESIGN.md       # 多租户设计
    ├── DEDUPLICATION_DESIGN.md      # 秒传设计
    └── API_EXAMPLES.md              # API 示例
```

## 特性

### ✅ 已实现

- ✅ **预签名上传**：客户端直接上传到 MinIO，减轻后端压力
- ✅ **预签名下载**：安全的临时访问链接
- ✅ **HTTP 缓存**：浏览器缓存 1 年，秒下载（已启用）
- ✅ **多租户隔离**：按 `{router}/{date}/{uuid}.{ext}` 格式存储
- ✅ **精确统计**：支持按租户/应用/函数统计存储占用
- ✅ **批量管理**：支持列举和批量删除
- ✅ **文件大小限制**：默认 100MB
- ✅ **AGPLv3 隔离**：通过 S3 API 调用，无代码感染

### 🔮 预留功能（未来启用）

- 🔮 **秒传**：相同文件只上传一次（数据库表已创建）
- 🔮 **去重**：物理存储只保留一份，节省成本（架构已预留）
- 🔮 **SeaweedFS 支持**：Apache 2.0 许可的存储后端

## 快速开始

### 1. 部署 MinIO

```bash
bash deploy/dev/scripts/infra.sh podman up -d minio
```

访问控制台：http://localhost:9001
- 用户名：minioadmin
- 密码：minioadmin123

### 2. 启动服务

```bash
bash scripts/start-app-storage.sh
```

服务地址：http://localhost:8083

### 3. 查看 API 文档

访问 Swagger 文档：http://localhost:8083/swagger/index.html

## API 接口

### 1. 获取上传凭证

```bash
POST /api/v1/storage/upload_token
Content-Type: application/json

{
  "router": "luobei/test88888/tools/cashier_desk",
  "file_name": "test.jpg",
  "content_type": "image/jpeg",
  "file_size": 102400,
  "hash": "abc123..."  // 可选，用于秒传（未来）
}
```

响应：

```json
{
  "code": 0,
  "data": {
    "url": "http://localhost:9000/ai-agent-os/luobei/test88888/tools/cashier_desk/2025/01/03/xxx.jpg?X-Amz-...",
    "key": "luobei/test88888/plugins/cashier_desk/2025/01/03/xxx.jpg",
    "method": "PUT",
    "expire": "2025-01-03 15:30:00",
    "headers": {
      "Content-Type": "image/jpeg"
    },
    "bucket": "ai-agent-os"
  }
}
```

**说明**：文件将按照 `{router}/{date}/{uuid}.{ext}` 的格式存储，实现多租户隔离。

### 2. 上传文件（客户端直接调用）

```bash
curl -X PUT "上面返回的 url" \
  -H "Content-Type: image/jpeg" \
  --data-binary "@test.jpg"
```

### 3. 获取下载链接

```bash
GET /api/v1/storage/download/luobei/test88888/plugins/cashier_desk/2025/01/03/xxx.jpg
```

**说明**：下载链接自动包含 HTTP 缓存头（`Cache-Control: max-age=31536000`），浏览器会缓存 1 年，实现秒下载。

### 4. 删除文件

```bash
DELETE /api/v1/storage/files/luobei/test88888/plugins/cashier_desk/2025/01/03/xxx.jpg
```

### 5. 获取文件信息

```bash
GET /api/v1/storage/files/luobei/test88888/plugins/cashier_desk/2025/01/03/xxx.jpg/info
```

### 6. 获取存储统计（按函数）

```bash
GET /api/v1/storage/stats?router=luobei/test88888/plugins/cashier_desk
```

响应：

```json
{
  "code": 0,
  "data": {
    "router": "luobei/test88888/plugins/cashier_desk",
    "file_count": 15,
    "total_size": 2048576,
    "size_human": "2.0 MB"
  }
}
```

### 7. 列举函数下的所有文件

```bash
GET /api/v1/storage/files?router=luobei/test88888/plugins/cashier_desk
```

### 8. 批量删除函数下的所有文件（危险操作）

```bash
POST /api/v1/storage/batch_delete
Content-Type: application/json

{
  "router": "luobei/test88888/tools/cashier_desk"
}
```

## 配置说明

当前官方仅支持 **MinIO**。配置文件：`deploy/dev/config/app-storage.yaml` 或生产模板 `deploy/prod/config/template/app-storage.yaml`

```yaml
server:
  port: 8083
  log_level: "info"
  debug: true

storage:
  type: "minio"
  minio:
    endpoint: "localhost:9000"
    access_key: "minioadmin"
    secret_key: "minioadmin123"
    use_ssl: false
    region: "us-east-1"
    default_bucket: "ai-agent-os"
  upload:
    max_size: 104857600        # 100MB
    token_expire: 3600         # 1小时

# 数据库配置（可选，用于秒传功能）
db:
  type: "mysql"
  host: "127.0.0.1"
  port: 3306
  user: "app"
  password: "app"
  name: "app_storage"
```

## 技术栈

- **MinIO SDK**: `github.com/minio/minio-go/v7` (Apache 2.0)
- **Web 框架**: Gin
- **配置管理**: Viper
- **数据库**: GORM + MySQL（可选，用于秒传）
- **日志**: 统一日志系统

## 多租户架构

### 文件存储路径

```
{tenant}/{app}/{function_path}/{date}/{uuid}.{ext}

示例：
luobei/test88888/tools/cashier_desk/2025/11/03/550e8400-e29b-41d4-a716-446655440000.jpg
│      │         │                  │          │
│      │         │                  │          └─ UUID（防止文件名冲突）
│      │         │                  └─ 日期分组（年/月/日）
│      │         └─ 函数路径
│      └─ 应用名称
└─ 租户名称
```

### 优势

- ✅ **租户隔离**：每个租户的文件完全独立
- ✅ **精确统计**：可以统计任意粒度的存储占用
- ✅ **批量管理**：支持按函数批量删除
- ✅ **审计追踪**：知道每个文件属于哪个函数
- ✅ **成本分摊**：可以按租户/应用/函数计费

## 性能优化

### 1. HTTP 缓存（已启用）

下载链接自动添加缓存控制头：

```http
Cache-Control: public, max-age=31536000, immutable
Expires: Mon, 03 Nov 2026 12:00:00 GMT
```

**效果**：
- 浏览器缓存 1 年
- 再次访问同一文件，直接从本地加载（0ms）
- 减少 90%+ 的重复下载请求

### 2. 秒传（预留，未来启用）

- 数据库表已创建：`file_metadata`, `file_references`
- DTO 字段已预留：`hash` 字段
- 配置开关已预留：`deduplication.enabled`

**预期效果**：
- 相同文件只上传一次
- 节省 30-80% 的存储成本
- 用户体验：秒传（0s）

详见：[秒传架构设计](docs/DEDUPLICATION_DESIGN.md)

## 安全说明

### AGPLv3 隔离

本服务通过 **网络边界** 与 MinIO 交互，使用 Apache 2.0 许可的 `minio-go` SDK：

```
┌─────────────────┐
│  app-storage    │  <- Apache 2.0 / BSL 1.1
│  (Your Code)    │
└────────┬────────┘
         │ HTTP (S3 API)
         │ 网络边界 = AGPLv3 隔离
         ↓
┌─────────────────┐
│  MinIO Server   │  <- AGPLv3 (独立进程)
│  (Podman)       │
└─────────────────┘
```

- ✅ 不直接 import MinIO 内部包
- ✅ 不修改 MinIO 源码
- ✅ 仅通过 S3 协议通信
- ✅ 代码不受 AGPLv3 感染

## 文档

- [多租户设计](docs/MULTI_TENANT_DESIGN.md) - 详细的多租户架构说明
- [秒传设计](docs/DEDUPLICATION_DESIGN.md) - 秒传和去重的实现方案
- [API 示例](docs/API_EXAMPLES.md) - 完整的 API 使用示例和前端集成

## 开发计划

### 已完成 ✅

- [x] 基础存储服务（上传/下载/删除）
- [x] 多租户隔离
- [x] 精确统计
- [x] 批量管理
- [x] HTTP 缓存（秒下载）
- [x] 数据库表预留（秒传）

### 计划中 📋

- [ ] 秒传功能（按需启用）
- [ ] SeaweedFS Provider
- [ ] CDN 加速配置
- [ ] 图片处理（缩略图、水印）
- [ ] 视频转码
- [ ] 文件预览

## FAQ

### Q1: HTTP 缓存如何工作？

A: 当前下载链路会自动返回长期缓存相关响应头，浏览器会缓存静态文件；这部分目前不是通过 `app-storage.yaml` 的独立配置块控制的。

### Q2: 秒传何时启用？

A: 当满足以下条件时，建议启用秒传：
- 文件数量 > 10,000
- 重复上传比例 > 20%
- 大文件场景（视频、压缩包等）
- 存储成本 > $100/月

### Q3: 如何统计每个租户的存储占用？

A: 使用统计 API：
```bash
GET /api/v1/storage/stats?router=tenant_name
```

### Q4: 如何清理某个函数的所有文件？

A: 使用批量删除 API：
```bash
POST /api/v1/storage/batch_delete
{"router": "tenant/app/function"}
```

### Q5: 是否支持其他对象存储？

A: 当前官方只支持 MinIO。其他对象存储如果后面真要支持，会在代码、配置模板和部署文档一起落地后再公开，不再提前对外承诺。
