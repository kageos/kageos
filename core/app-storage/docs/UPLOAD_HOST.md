# 上传域名和 Host 返回设计

## 🎯 为什么需要返回上传域名和 Host？

### 使用场景

1. **进度监听定位**
   - XMLHttpRequest 的 `upload.onprogress` 需要知道上传到哪个域名
   - 用于日志记录和调试

2. **CORS 配置**
   - 前端需要知道上传目标域名，以便配置 CORS
   - 跨域场景下的请求头设置

3. **错误定位**
   - 上传失败时，知道是哪个域名的问题
   - 便于排查网络问题

4. **日志和监控**
   - 记录上传到哪个存储服务
   - 监控不同存储服务的上传性能

---

## 📊 架构设计

### 1. UploadCredentials（上传凭证）

```go
type UploadCredentials struct {
    Method UploadMethod
    
    // 预签名 URL
    URL     string
    Headers map[string]string
    
    // ✨ 上传域名信息
    UploadHost   string // 上传目标 host（例如：localhost:9000）
    UploadDomain string // 上传完整域名（例如：http://localhost:9000）
}
```

### 2. API Response（API 响应）

```json
{
  "key": "luobei/test88888/plugins/cashier_desk/2025/11/03/xxx.pdf",
  "method": "presigned_url",
  "url": "http://localhost:9000/ai-agent-os/xxx.pdf?X-Amz-Signature=...",
  "upload_host": "localhost:9000",          // ✨ 上传目标 host
  "upload_domain": "http://localhost:9000", // ✨ 上传完整域名
  "headers": {
    "Content-Type": "application/pdf"
  }
}
```

---

## 🎨 前端使用

### 场景 1：进度监听

```typescript
// web/src/utils/upload/presigned-url.ts
export class PresignedURLUploader implements Uploader {
  async upload(credentials, file, onProgress) {
    const uploadDomain = credentials.upload_domain || 'unknown'
    console.log(`[Upload] 开始上传到: ${uploadDomain}`)
    
    const xhr = new XMLHttpRequest()
    
    // 监听上传进度（知道上传到哪个域名）
    xhr.upload.addEventListener('progress', (e) => {
      const percent = (e.loaded / e.total) * 100
      
      onProgress({
        percent,
        loaded: e.loaded,
        total: e.total,
        uploadDomain, // ✨ 包含上传域名信息
      })
      
      // 记录进度日志（包含域名）
      console.log(`[Upload] ${uploadDomain} - 进度: ${percent}%`)
    })
    
    xhr.open('PUT', credentials.url)
    xhr.send(file)
  }
}
```

### 场景 2：CORS 配置

```typescript
// 如果需要跨域上传，可以根据 upload_host 配置 CORS
const uploadHost = credentials.upload_host // localhost:9000

// 检查是否需要 CORS
if (isCrossOrigin(uploadHost)) {
  // 设置 CORS 相关请求头
  xhr.withCredentials = true
}
```

### 场景 3：错误处理和日志

```typescript
xhr.addEventListener('error', () => {
  const uploadDomain = credentials.upload_domain
  console.error(`[Upload] ${uploadDomain} - 网络错误`)
  
  // 发送错误日志到监控系统
  sendErrorLog({
    uploadDomain,
    error: '网络错误',
    timestamp: Date.now(),
  })
})
```

### 场景 4：多存储服务监控

```typescript
// 监控不同存储服务的上传性能
const uploadMetrics = {
  [credentials.upload_domain]: {
    uploadCount: 0,
    totalSize: 0,
    averageSpeed: 0,
  }
}

xhr.upload.addEventListener('progress', (e) => {
  const speed = calculateSpeed(e.loaded)
  uploadMetrics[credentials.upload_domain].averageSpeed = speed
})
```

---

## 📝 实现细节

### 1. 域名提取（MinIO 示例）

```go
// storage/minio.go
func (s *MinIOStorage) extractDomainInfo(uploadURL string) (host string, domain string) {
    parsedURL, err := url.Parse(uploadURL)
    if err != nil {
        return "", ""
    }
    
    // 提取 host（hostname:port）
    host = parsedURL.Host  // localhost:9000
    
    // 提取完整域名（scheme://host）
    scheme := parsedURL.Scheme
    if scheme == "" {
        scheme = "http"
    }
    domain = fmt.Sprintf("%s://%s", scheme, host)  // http://localhost:9000
    
    return host, domain
}
```

### 2. 不同存储的域名格式

#### MinIO

```
URL: http://localhost:9000/ai-agent-os/file.pdf?X-Amz-Signature=...
upload_host: localhost:9000
upload_domain: http://localhost:9000
```

#### 腾讯云 COS

```
URL: https://my-bucket-xxx.cos.ap-guangzhou.myqcloud.com/file.pdf?sign=...
upload_host: my-bucket-xxx.cos.ap-guangzhou.myqcloud.com
upload_domain: https://my-bucket-xxx.cos.ap-guangzhou.myqcloud.com
```

#### 阿里云 OSS

```
URL: https://my-bucket.oss-cn-hangzhou.aliyuncs.com/file.pdf?OSSAccessKeyId=...
upload_host: my-bucket.oss-cn-hangzhou.aliyuncs.com
upload_domain: https://my-bucket.oss-cn-hangzhou.aliyuncs.com
```

---

## ✅ 优势总结

### 1. **进度监听更准确**

```typescript
// 知道上传到哪个域名，便于调试
xhr.upload.onprogress = (e) => {
  console.log(`上传到 ${uploadDomain} - 进度: ${percent}%`)
}
```

### 2. **错误定位更精确**

```typescript
// 上传失败时，知道是哪个域名的问题
xhr.onerror = () => {
  console.error(`上传到 ${uploadDomain} 失败`)
  // 可以针对不同域名做不同的错误处理
}
```

### 3. **CORS 配置更灵活**

```typescript
// 根据上传域名配置 CORS
if (needsCORS(credentials.upload_host)) {
  xhr.withCredentials = true
}
```

### 4. **监控和日志更完善**

```typescript
// 记录不同存储服务的上传性能
monitorUpload({
  uploadDomain: credentials.upload_domain,
  uploadSpeed: speed,
  uploadTime: duration,
})
```

---

## 🔄 完整流程

### 1. 请求上传凭证

```typescript
POST /api/v1/storage/upload_token
{
  "router": "luobei/test88888/plugins/cashier_desk",
  "file_name": "invoice.pdf"
}
```

### 2. 后端返回上传信息

```json
{
  "key": "luobei/test88888/plugins/cashier_desk/2025/11/03/xxx.pdf",
  "method": "presigned_url",
  "url": "http://localhost:9000/ai-agent-os/xxx.pdf?X-Amz-Signature=...",
  "upload_host": "localhost:9000",
  "upload_domain": "http://localhost:9000",
  "headers": {
    "Content-Type": "application/pdf"
  }
}
```

### 3. 前端使用上传信息

```typescript
const { url, upload_host, upload_domain } = credentials

// 记录上传域名
console.log(`开始上传到: ${upload_domain}`)

// 发起上传
xhr.open('PUT', url)

// 监听进度（知道上传到哪个域名）
xhr.upload.onprogress = (e) => {
  console.log(`${upload_domain} - 进度: ${percent}%`)
}

xhr.send(file)
```

---

## 🎯 总结

**返回上传域名和 Host 是必要的！** ✅

1. ✅ **进度监听**：知道上传到哪个域名，便于调试和日志
2. ✅ **CORS 配置**：根据上传域名配置跨域请求
3. ✅ **错误定位**：上传失败时，知道是哪个域名的问题
4. ✅ **监控统计**：记录不同存储服务的上传性能

**实现要点**：
- `upload_host`：上传目标 host（hostname:port）
- `upload_domain`：上传完整域名（scheme://host）
- 从预签名 URL 中自动提取
- 前端用于进度监听、日志、监控

🎉

