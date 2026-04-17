# CDN 域名返回设计

> 历史设计草案：本文包含多存储后端设想。当前官方实现与 `deploy/*` 主线只支持 **MinIO**，不要把文中的 `tencentcos / aliyunoss` 示例当成已落地能力。

## 🎯 为什么需要返回域名？

### 使用场景

1. **CDN 加速访问**
   - 文件上传后，前端需要知道如何快速访问文件
   - CDN 域名通常比原始存储域名快很多

2. **自定义域名**
   - 用户可能配置了自己的域名绑定到存储桶
   - 例如：`https://files.example.com` → MinIO Bucket

3. **前端展示**
   - 上传成功后，前端需要显示文件预览/下载链接
   - 使用 CDN 域名构建 URL，速度更快

4. **统一访问入口**
   - 即使存储后端切换，前端访问域名不变
   - 通过 CDN 统一管理

---

## 📊 架构设计

### 1. Storage Interface

```go
type Storage interface {
    GetCDNDomain() string  // 返回 CDN 域名
    // ...
}
```

### 2. Config Interface

```go
type Config interface {
    GetCDNDomain() string  // 从配置读取 CDN 域名
    // ...
}
```

### 3. API Response

```json
{
  "key": "luobei/test88888/plugins/cashier_desk/2025/11/03/xxx.pdf",
  "bucket": "ai-agent-os",
  "method": "presigned_url",
  "url": "http://localhost:9000/ai-agent-os/xxx.pdf?X-Amz-Signature=...",
  "cdn_domain": "https://cdn.example.com",  // ✨ CDN 域名
  "expire": "2025-11-04 00:00:00"
}
```

---

## 🎨 前端使用

### 场景 1：上传后显示文件链接

```typescript
// 上传成功后，使用 CDN 域名构建访问 URL
const uploadResponse = await getUploadToken(...)

// 方案 A：使用预签名 URL（临时访问）
const downloadURL = uploadResponse.url

// 方案 B：使用 CDN 域名（永久访问，需要配置访问策略）
const cdnURL = uploadResponse.cdn_domain 
  ? `${uploadResponse.cdn_domain}/${uploadResponse.key}`
  : downloadURL

// 显示文件链接
showFileLink(cdnURL)
```

### 场景 2：列表展示文件

```typescript
// 文件列表接口返回 key
const files = [
  { key: "path/to/file1.pdf" },
  { key: "path/to/file2.jpg" }
]

// 使用 CDN 域名构建访问 URL
const cdnDomain = getCDNDomain()  // 从配置或首次上传响应获取

files.forEach(file => {
  const fileURL = `${cdnDomain}/${file.key}`
  renderFileItem(file, fileURL)
})
```

### 场景 3：图片预览

```vue
<template>
  <div>
    <img 
      v-if="fileUrl" 
      :src="fileUrl" 
      alt="文件预览"
    />
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps<{
  fileKey: string
  cdnDomain?: string
}>()

// 优先使用 CDN 域名
const fileUrl = computed(() => {
  if (props.cdnDomain) {
    return `${props.cdnDomain}/${props.fileKey}`
  }
  // 降级：请求预签名 URL
  return getPresignedURL(props.fileKey)
})
</script>
```

---

## 📝 配置示例

### MinIO（本地开发）

```yaml
storage:
  type: "minio"
  minio:
    endpoint: "localhost:9000"
    default_bucket: "ai-agent-os"
    cdn_domain: ""  # 本地开发，不使用 CDN
```

### MinIO（生产环境 + CDN）

```yaml
storage:
  type: "minio"
  minio:
    endpoint: "minio.example.com:9000"
    default_bucket: "ai-agent-os"
    cdn_domain: "https://cdn.example.com"  # 配置 CDN 域名
```

### 腾讯云 COS（自动 CDN）

```yaml
storage:
  type: "tencentcos"
  tencentcos:
    endpoint: "cos.ap-guangzhou.myqcloud.com"
    default_bucket: "my-bucket-xxx"
    cdn_domain: "https://my-bucket-xxx-xxx.cos.ap-guangzhou.myqcloud.com"  # COS 默认域名
    # 或使用自定义 CDN 域名
    # cdn_domain: "https://files.example.com"  # 自定义域名
```

### 阿里云 OSS（OSS + CDN）

```yaml
storage:
  type: "aliyunoss"
  aliyunoss:
    endpoint: "oss-cn-hangzhou.aliyuncs.com"
    default_bucket: "my-bucket"
    cdn_domain: "https://my-bucket.oss-cn-hangzhou.aliyuncs.com"  # OSS 域名
    # 或使用阿里云 CDN
    # cdn_domain: "https://cdn.example.com"  # CDN 加速域名
```

---

## ✅ 优势总结

### 1. **性能提升**

```
无 CDN：用户 → 存储服务（慢）
有 CDN：用户 → CDN 边缘节点（快）
```

### 2. **访问控制**

- 可以通过 CDN 配置访问策略（防盗链、IP 白名单等）
- 统一管理访问入口

### 3. **成本优化**

- CDN 流量费用通常比存储服务流量费用低
- 减少存储服务带宽压力

### 4. **前端灵活性**

```typescript
// 前端可以根据 CDN 域名构建 URL
const fileURL = cdnDomain 
  ? `${cdnDomain}/${fileKey}`  // 使用 CDN（永久访问）
  : presignedURL                // 使用预签名 URL（临时访问）
```

---

## 🔄 完整流程

### 1. 上传文件

```typescript
// 请求上传凭证
const response = await fetch('/api/v1/storage/upload_token', {
  method: 'POST',
  body: JSON.stringify({
    router: 'luobei/test88888/plugins/cashier_desk',
    file_name: 'invoice.pdf',
    file_size: 102400
  })
})

const { key, cdn_domain, url } = await response.json()
// key: "luobei/test88888/plugins/cashier_desk/2025/11/03/xxx.pdf"
// cdn_domain: "https://cdn.example.com"
// url: "http://localhost:9000/...?X-Amz-Signature=..."（预签名上传 URL）
```

### 2. 执行上传

```typescript
// 使用预签名 URL 上传
await uploadFile(url, file)
```

### 3. 上传成功后显示文件

```typescript
// 使用 CDN 域名构建访问 URL
const fileURL = cdn_domain 
  ? `${cdn_domain}/${key}`  // 永久访问（需要配置访问策略）
  : await getPresignedDownloadURL(key)  // 临时访问

// 显示文件链接
showFileLink(fileURL)
```

---

## ⚠️ 注意事项

### 1. **CDN 域名访问策略**

如果使用 CDN 域名，需要确保：

- ✅ **公开访问**：配置 Bucket 为公开读取
- ✅ **访问控制**：配置 CDN 访问策略（防盗链、IP 白名单等）
- ✅ **HTTPS**：使用 HTTPS 域名，确保安全

### 2. **预签名 URL vs CDN 域名**

| 特性 | 预签名 URL | CDN 域名 |
|------|-----------|---------|
| **访问方式** | 临时授权 | 永久访问 |
| **安全性** | ✅ 高（有过期时间）| ⚠️ 需配置访问策略 |
| **速度** | 取决于存储服务 | ✅ 快（CDN 加速）|
| **适用场景** | 临时分享 | 公开文件、图片预览 |

### 3. **降级策略**

```typescript
// 如果 CDN 域名未配置，降级使用预签名 URL
const fileURL = cdnDomain 
  ? `${cdnDomain}/${key}`
  : await getPresignedDownloadURL(key)
```

---

## 🎯 总结

**返回 CDN 域名是必要的！** ✅

1. ✅ **性能提升**：CDN 加速访问
2. ✅ **前端灵活性**：可以根据 CDN 域名构建 URL
3. ✅ **统一管理**：通过 CDN 统一访问入口
4. ✅ **成本优化**：减少存储服务带宽压力

**实现要点**：
- Storage 接口返回 CDN 域名
- 配置文件中可配置 CDN 域名
- API 响应中包含 CDN 域名
- 前端根据 CDN 域名构建访问 URL（可选）

🎉
