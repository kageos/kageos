# 前端上传架构 - 多云存储支持

## 🎯 设计目标

**问题**：后续可能需要支持多种云存储（MinIO、七牛云、阿里云 OSS、腾讯云 COS、AWS S3 等），如何保证扩展性？

**解决方案**：策略模式 + 工厂模式 + 后端驱动

---

## 📐 架构设计

### 核心思想

**前端不关心具体存储类型，由后端告诉前端使用哪种上传方式。**

```
前端请求上传凭证
  ↓
后端返回：{ method: "presigned_url", url: "...", ... }
  ↓
前端根据 method 创建对应的上传器
  ↓
执行上传
```

### 三层架构

```
┌─────────────────────────────────────────────────────────┐
│                    业务层                                 │
│              FilesWidget.ts                              │
│       (只关心"上传文件"，不关心"怎么上传")                 │
└────────────────────┬────────────────────────────────────┘
                     ↓ 调用
┌─────────────────────────────────────────────────────────┐
│                   抽象层                                  │
│              uploadFile() - 统一入口                      │
│       1. 请求后端获取上传凭证（包含 method）               │
│       2. 根据 method 创建上传器                           │
│       3. 执行上传                                         │
└────────────────────┬────────────────────────────────────┘
                     ↓ 委托
┌─────────────────────────────────────────────────────────┐
│                   策略层                                  │
│              UploaderFactory                             │
│   ├─ PresignedURLUploader (MinIO/COS/OSS/S3)            │
│   ├─ FormUploader (七牛云/又拍云)                        │
│   └─ SDKUploader (特殊云存储)                            │
└─────────────────────────────────────────────────────────┘
```

---

## 🔧 实现细节

### 1. 后端返回上传方式

```go
// dto/storage.go
type GetUploadTokenResp struct {
    Method UploadMethod `json:"method"`  // ✨ 关键：告诉前端用哪种方式
    
    // 预签名 URL 上传（MinIO、COS、OSS、S3）
    URL     string            `json:"url,omitempty"`
    Headers map[string]string `json:"headers,omitempty"`
    
    // 表单上传（七牛云、又拍云）
    FormData map[string]string `json:"form_data,omitempty"`
    PostURL  string            `json:"post_url,omitempty"`
    
    // SDK 上传（特殊云存储）
    SDKConfig map[string]interface{} `json:"sdk_config,omitempty"`
    
    // 其他字段...
}

// 支持的上传方式
const (
    UploadMethodPresignedURL UploadMethod = "presigned_url"  // 标准 S3 协议
    UploadMethodFormUpload   UploadMethod = "form_upload"    // 表单上传
    UploadMethodSDKUpload    UploadMethod = "sdk_upload"     // SDK 上传
)
```

### 2. 前端统一入口

```typescript
// utils/upload/index.ts
export async function uploadFile(
  router: string,
  file: File,
  onProgress: (progress: UploadProgress) => void
): Promise<string> {
  
  // ✨ Step 1: 请求后端获取上传凭证
  const credentials = await getUploadCredentials(router, file)
  // credentials = {
  //   method: "presigned_url",  // ← 后端告诉前端用哪种方式
  //   url: "http://localhost:9000/...",
  //   headers: { "Content-Type": "..." }
  // }
  
  // ✨ Step 2: 根据 method 创建对应的上传器（策略模式）
  const uploader = UploaderFactory.create(credentials.method)
  
  // ✨ Step 3: 执行上传
  await uploader.upload(credentials, file, onProgress)
  
  // Step 4: 通知后端上传完成
  await notifyUploadComplete(credentials.key, true)
  
  return credentials.key
}
```

### 3. 上传器工厂

```typescript
// utils/upload/index.ts
export class UploaderFactory {
  static create(method: string): Uploader {
    switch (method) {
      case 'presigned_url':
        // 预签名 URL 上传（MinIO、COS、OSS、S3）
        return new PresignedURLUploader()
      
      case 'form_upload':
        // 表单上传（七牛云、又拍云等）
        return new FormUploader()
      
      case 'sdk_upload':
        // SDK 上传（特殊云存储）
        return new SDKUploader()
      
      default:
        throw new Error(`不支持的上传方式: ${method}`)
    }
  }
}
```

### 4. 上传器接口

```typescript
// utils/upload/index.ts
export interface Uploader {
  /**
   * 执行上传
   * @param credentials - 上传凭证（包含 URL、Headers、FormData 等）
   * @param file - 要上传的文件
   * @param onProgress - 进度回调
   */
  upload(
    credentials: UploadCredentials,
    file: File,
    onProgress: (progress: UploadProgress) => void
  ): Promise<void>
  
  /**
   * 取消上传
   */
  cancel(): void
}
```

### 5. 具体上传器实现

#### PresignedURLUploader（MinIO/COS/OSS/S3）

```typescript
// utils/upload/presigned-url.ts
export class PresignedURLUploader implements Uploader {
  private xhr: XMLHttpRequest | null = null

  async upload(credentials, file, onProgress) {
    this.xhr = new XMLHttpRequest()
    
    // 监听上传进度
    this.xhr.upload.onprogress = (e) => {
      onProgress({
        percent: (e.loaded / e.total) * 100,
        loaded: e.loaded,
        total: e.total,
        speed: calculateSpeed(e.loaded),
      })
    }
    
    // 使用预签名 URL 上传（HTTP PUT）
    this.xhr.open('PUT', credentials.url)
    
    // 设置请求头
    Object.entries(credentials.headers).forEach(([key, value]) => {
      this.xhr.setRequestHeader(key, value)
    })
    
    this.xhr.send(file)
  }
  
  cancel() {
    this.xhr?.abort()
  }
}
```

#### FormUploader（七牛云/又拍云）

```typescript
// utils/upload/form-upload.ts
export class FormUploader implements Uploader {
  private xhr: XMLHttpRequest | null = null

  async upload(credentials, file, onProgress) {
    this.xhr = new XMLHttpRequest()
    
    // 监听上传进度
    this.xhr.upload.onprogress = (e) => {
      onProgress({
        percent: (e.loaded / e.total) * 100,
        loaded: e.loaded,
        total: e.total,
        speed: calculateSpeed(e.loaded),
      })
    }
    
    // 构建表单数据
    const formData = new FormData()
    
    // 添加云存储要求的表单字段
    Object.entries(credentials.form_data).forEach(([key, value]) => {
      formData.append(key, value)
    })
    
    // 添加文件
    formData.append('file', file)
    
    // 使用表单上传（HTTP POST）
    this.xhr.open('POST', credentials.post_url)
    this.xhr.send(formData)
  }
  
  cancel() {
    this.xhr?.abort()
  }
}
```

---

## 🚀 扩展性

### 添加新的云存储（例如：阿里云 OSS）

#### 后端（Go）

```go
// 1. 在 storage 包添加新实现
// core/app-storage/storage/aliyunoss.go
type AliyunOSSStorage struct {
    client *oss.Client
}

func (s *AliyunOSSStorage) GenerateUploadCredentials(...) (*UploadCredentials, error) {
    // 阿里云 OSS 支持预签名 URL，返回 presigned_url
    return &UploadCredentials{
        Method: UploadMethodPresignedURL,
        URL:    presignedURL,
        Headers: map[string]string{
            "Content-Type": contentType,
        },
    }, nil
}

// 2. 在 StorageFactory 注册
func NewFactory(...) Storage {
    switch cfg.Storage.Type {
    case "aliyunoss":
        return NewAliyunOSSStorage(...)
    // ... 其他
    }
}
```

#### 前端（TypeScript）

**无需修改！** ✅

因为阿里云 OSS 支持预签名 URL（S3 兼容），前端的 `PresignedURLUploader` 已经支持。

---

### 添加特殊的云存储（例如：华为云 OBS）

#### 后端（Go）

```go
// 如果华为云 OBS 使用特殊的 SDK
type HuaweiOBSStorage struct {
    client *obs.Client
}

func (s *HuaweiOBSStorage) GenerateUploadCredentials(...) (*UploadCredentials, error) {
    // 返回 SDK 配置
    return &UploadCredentials{
        Method: UploadMethodSDKUpload,
        SDKConfig: map[string]interface{}{
            "ak":        s.accessKey,
            "sk":        s.secretKey,
            "endpoint":  s.endpoint,
            "bucket":    bucket,
            "objectKey": key,
        },
    }, nil
}
```

#### 前端（TypeScript）

```typescript
// utils/upload/huawei-obs.ts
export class HuaweiOBSUploader implements Uploader {
  async upload(credentials, file, onProgress) {
    // 使用华为云 OBS SDK
    const obsClient = new ObsClient(credentials.sdk_config)
    
    // 上传并监听进度
    await obsClient.putObject({
      Bucket: credentials.sdk_config.bucket,
      Key: credentials.sdk_config.objectKey,
      Body: file,
      ProgressCallback: (transferred, total) => {
        onProgress({
          percent: (transferred / total) * 100,
          loaded: transferred,
          total,
        })
      }
    })
  }
}

// 注册到工厂
// utils/upload/index.ts
case 'sdk_upload':
  // 根据 sdk_config 判断具体的云存储
  if (credentials.sdk_config.provider === 'huawei') {
    return new HuaweiOBSUploader()
  }
  return new SDKUploader()
```

---

## 🎯 关键优势

### 1. **前端无需关心存储类型**

```typescript
// FilesWidget 的代码永远是这样：
await uploadFile(router, file, onProgress)

// 无论后端用的是 MinIO、COS、OSS、S3、七牛云...
// 前端代码都不需要改！
```

### 2. **后端驱动前端行为**

```
后端配置：
  storage.type = "minio"  →  返回 method: "presigned_url"
  storage.type = "qiniu"  →  返回 method: "form_upload"
  storage.type = "huawei" →  返回 method: "sdk_upload"
```

### 3. **标准接口，易于扩展**

```typescript
// 新增云存储，只需：
// 1. 实现 Uploader 接口
// 2. 注册到 UploaderFactory
// 3. 完成！
```

---

## 📊 数据流

```
FilesWidget
  ↓ uploadFile(router, file, onProgress)
  
utils/upload/index.ts
  ↓ getUploadCredentials(router, file)
  
后端 API: /api/v1/storage/upload_token
  ↓ 返回 { method: "presigned_url", url: "...", ... }
  
UploaderFactory.create("presigned_url")
  ↓ 创建 PresignedURLUploader
  
PresignedURLUploader.upload(credentials, file, onProgress)
  ↓ XMLHttpRequest PUT
  
MinIO/COS/OSS/S3
  ↓ 上传完成
  
notifyUploadComplete(key, true)
  ↓ 通知后端
  
后端更新 file_uploads 表
```

---

## 🔒 安全性

1. **上传凭证有时效**：预签名 URL 会过期
2. **前端无需 AK/SK**：敏感信息在后端
3. **文件大小限制**：后端验证
4. **文件类型限制**：前后端双重验证

---

## 📝 总结

**关键设计原则**：

1. ✅ **依赖倒置**：FilesWidget 依赖抽象的 `uploadFile`，不依赖具体的 MinIO
2. ✅ **策略模式**：不同的上传方式（预签名 URL、表单上传、SDK）作为不同的策略
3. ✅ **工厂模式**：`UploaderFactory` 根据 `method` 创建对应的上传器
4. ✅ **后端驱动**：前端不关心存储类型，由后端告诉前端用哪种方式
5. ✅ **易于扩展**：新增云存储，只需添加新的 `Uploader` 实现

**这个架构可以支持任何云存储，前端无需修改！** 🎉

