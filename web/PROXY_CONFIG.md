# 代理配置说明

## 📊 服务端口分布

| 服务 | 端口 | 说明 |
|-----|------|------|
| `api-gateway` | 9090 | API 网关（统一入口） |
| `app-server` | 9091 | 主业务 API |
| `app-storage` | 9092 | 文件存储服务 |
| MinIO | 9000 | 对象存储 |
| Vite Dev Server | 5173 | 前端开发服务器 |

---

## ⚙️ Vite 代理配置

### 配置文件：`web/vite.config.ts`

```typescript
export default defineConfig({
  server: {
    proxy: {
      // 统一通过网关代理所有 API 请求
      '/api': {
        target: 'http://localhost:9090',  // 网关地址
        changeOrigin: true,
      },
      // Swagger 文档也通过网关
      '/swagger': {
        target: 'http://localhost:9090',  // 网关地址
        changeOrigin: true,
      },
    },
  },
})
```

### 代理规则

**统一网关代理**
```
前端请求：http://localhost:5173/api/v1/storage/upload_token
  ↓ 代理到
API Gateway：http://localhost:9090/api/v1/storage/upload_token
  ↓ 网关转发
app-storage：http://localhost:9092/api/v1/storage/upload_token

前端请求：http://localhost:5173/api/v1/app/list
  ↓ 代理到
API Gateway：http://localhost:9090/api/v1/app/list
  ↓ 网关转发
app-server：http://localhost:9091/api/v1/app/list
```

---

## 🔄 请求流程

### 1. 获取上传凭证

```
前端 (5173)
  ↓ POST /api/v1/storage/upload_token
Vite 代理
  ↓ 转发到 localhost:9090 (网关)
API Gateway (9090)
  ↓ 转发到 localhost:9092
app-storage (9092)
  ↓ 生成预签名 URL
  ↓ 返回 { method: "presigned_url", url: "http://localhost:9000/...", ... }
前端 (5173)
  ↓ 收到凭证
```

### 2. 直接上传到 MinIO

```
前端 (5173)
  ↓ XMLHttpRequest PUT http://localhost:9000/...?signature=...
  ↓ 不经过代理！直接连接 MinIO
MinIO (9000)
  ↓ 接收文件
  ↓ 存储文件
前端 (5173)
  ↓ xhr.upload.onprogress 监听进度 ✅
```

**关键点**：
- 上传文件时，前端直接连接 MinIO（9000 端口）
- 不经过 Vite 代理（5173）
- 不经过网关（9090）和 app-storage（9092）
- 浏览器原生监听上传进度

### 3. 通知上传完成

```
前端 (5173)
  ↓ POST /api/v1/storage/upload_complete
Vite 代理
  ↓ 转发到 localhost:9090 (网关)
API Gateway (9090)
  ↓ 转发到 localhost:9092
app-storage (9092)
  ↓ 更新 file_uploads 表状态
```

---

## 🎯 为什么不需要后端提供进度接口？

### 错误理解

```
前端 (5173) → Vite 代理 (5173) → app-storage (8083) → MinIO (9000)
                                      ↑
                                  需要进度接口？❌
```

**问题**：
- 文件要经过 app-storage，占用后端带宽
- 后端需要实现进度转发
- 增加延迟

### 正确理解

```
┌─────────────────────────────────────────────────────────────────┐
│ Step 1: 获取上传凭证（经过代理）                                  │
│                                                                   │
│ 前端 (5173) → Vite 代理 → app-storage (8083)                    │
│   ↓                                                              │
│ 返回预签名 URL                                                    │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ Step 2: 直接上传到 MinIO（不经过代理）                           │
│                                                                   │
│ 前端 (5173) → 直接连接 → MinIO (9000)                           │
│   ↓                                                              │
│ xhr.upload.onprogress 监听进度 ✅                                │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ Step 3: 通知上传完成（经过代理）                                  │
│                                                                   │
│ 前端 (5173) → Vite 代理 → app-storage (8083)                    │
│   ↓                                                              │
│ 更新数据库状态                                                    │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🛠️ 测试步骤

### 1. 启动服务

```bash
# 启动 MinIO（如果使用 Podman）
podman start minio

# 启动 app-storage
cd /Users/beiluo/Documents/work/code/ai-agent-os
./bin/app-storage

# 启动 app-server
./bin/app-server

# 启动前端开发服务器
cd web
npm run dev
```

### 2. 测试代理

```bash
# 测试存储服务代理
curl http://localhost:5173/api/v1/storage/health

# 应该返回：
# {
#   "service": "app-storage",
#   "status": "ok",
#   "timestamp": "2024-11-04 12:00:00"
# }
```

### 3. 测试上传凭证

```bash
# 测试获取上传凭证
curl -X POST http://localhost:5173/api/v1/storage/upload_token \
  -H "Content-Type: application/json" \
  -H "X-Token: your-token" \
  -d '{
    "router": "luobei/test88888/plugins/cashier_desk",
    "file_name": "test.pdf",
    "file_size": 1024,
    "content_type": "application/pdf"
  }'

# 应该返回：
# {
#   "code": 200,
#   "data": {
#     "method": "presigned_url",
#     "url": "http://localhost:9000/ai-agent-os/luobei/.../test.pdf?...",
#     "key": "luobei/test88888/plugins/cashier_desk/2024/11/test.pdf",
#     "bucket": "ai-agent-os",
#     "expire": "2024-11-04T13:00:00Z"
#   }
# }
```

### 4. 测试文件上传

在浏览器中：
1. 打开 `http://localhost:5173`
2. 进入表单页面
3. 拖拽文件到上传组件
4. 观察上传进度
5. 查看 Network 面板：
   - `POST /api/v1/storage/upload_token` → 9090（网关）→ 9092（app-storage）
   - `PUT http://localhost:9000/...` → MinIO（直接连接）
   - `POST /api/v1/storage/upload_complete` → 9090（网关）→ 9092（app-storage）

---

## 🔍 常见问题

### Q1: 为什么上传文件不经过代理？

**A**: 因为使用预签名 URL，前端直接连接 MinIO（9000 端口）。

**优势**：
- 性能高（不经过后端）
- 带宽省（后端不参与文件传输）
- 浏览器原生支持进度监听

### Q2: 如何监听上传进度？

**A**: 使用 `XMLHttpRequest.upload.onprogress`，这是浏览器原生 API。

```typescript
xhr.upload.addEventListener('progress', (e) => {
  const percent = (e.loaded / e.total) * 100
  console.log(`上传进度: ${percent}%`)
})
```

### Q3: 不同云存储的进度监听有区别吗？

**A**: 没有区别，都使用 `XMLHttpRequest.upload.onprogress`。

| 云存储 | 上传方式 | 进度监听 |
|-------|---------|---------|
| MinIO | 预签名 URL PUT | `xhr.upload.onprogress` ✅ |
| 腾讯云 COS | 预签名 URL PUT | `xhr.upload.onprogress` ✅ |
| 阿里云 OSS | 预签名 URL PUT | `xhr.upload.onprogress` ✅ |
| AWS S3 | 预签名 URL PUT | `xhr.upload.onprogress` ✅ |
| 七牛云 | 表单 POST | `xhr.upload.onprogress` ✅ |

### Q4: 为什么使用统一网关？

**A**: 统一网关提供更好的架构优势：

**统一网关配置**：
```typescript
{
  '/api': { target: 'http://localhost:9090' },    // 所有 API 通过网关
  '/swagger': { target: 'http://localhost:9090' }, // Swagger 文档也通过网关
}
```

**优势**：
- ✅ 统一入口，简化配置
- ✅ 网关负责路由转发和负载均衡
- ✅ 前端无需关心后端服务端口
- ✅ 支持服务发现和动态路由
- ✅ 便于监控和日志收集

---

## 📝 总结

1. ✅ **统一网关**：所有 API 请求通过 `api-gateway` (9090) 统一入口
2. ✅ **代理配置**：Vite 代理 `/api` 和 `/swagger` 到网关
3. ✅ **直接上传**：前端直接连接 MinIO（9000），不经过网关
4. ✅ **进度监听**：使用浏览器原生 API，不需要后端提供接口
5. ✅ **统一接口**：所有云存储都使用相同的进度监听方式

**架构优势**：
- 🎯 **统一入口**：前端只需知道网关地址，无需关心后端服务端口
- 🚀 **负载均衡**：网关支持多实例负载均衡
- 🔒 **安全隔离**：后端服务不直接暴露给前端
- 📊 **监控统一**：所有请求经过网关，便于监控和日志

**关键点：后端只负责生成上传凭证，文件传输由前端直接完成，进度监听由浏览器原生支持！** 🎉

