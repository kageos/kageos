# 上传架构设计：前后端解耦 + 多存储支持

> 历史设计草案：本文包含七牛云等多存储上传设想。当前官方实现与 `deploy/*` 主线只支持 **MinIO**，这里的多后端内容不代表现状能力。

## 🎯 你的问题（完全正确！）

> 生成的上传token是否需要再返回上传的类型？我感觉需要，因为前端需要针对这个token类型来实现对应的上传实现？跟后端一样是个接口？例如七牛云上传接口？腾讯云上传接口，minio上传接口？然后针对不同的上传实现不同的回调？

**答案：你说得完全对！** ✅

---

## 📖 S3 协议是什么？

### 定义

**S3（Simple Storage Service）** 是 AWS 推出的对象存储服务标准，后来成为了**事实上的行业标准**。

### 核心特性

```
1. RESTful API：基于 HTTP 协议
   - PUT /bucket/key    → 上传文件
   - GET /bucket/key    → 下载文件
   - DELETE /bucket/key → 删除文件

2. 预签名 URL（Presigned URL）
   - 临时授权 URL
   - 有过期时间
   - 无需暴露密钥

3. Bucket + Object Key
   - Bucket：存储桶（命名空间）
   - Object Key：对象键（文件路径）

4. S3 兼容性
   - 很多云存储都兼容 S3 协议
   - 可以使用统一的 SDK（如 aws-sdk）
```

### S3 兼容性对比

| 存储 | S3 兼容 | 上传方式 | 前端实现 |
|------|---------|---------|---------|
| **MinIO** | ✅ 100% | 预签名 URL | XMLHttpRequest PUT |
| **AWS S3** | ✅ 原生 | 预签名 URL | XMLHttpRequest PUT |
| **阿里云 OSS** | ✅ 兼容 | 预签名 URL | XMLHttpRequest PUT |
| **腾讯云 COS** | ✅ 兼容 | 预签名 URL | XMLHttpRequest PUT |
| **七牛云** | ⚠️ 部分 | 表单上传 | XMLHttpRequest POST + FormData ⚠️ |
| **又拍云** | ⚠️ 部分 | 自定义 | 又拍云 SDK ⚠️ |

**结论**：
- ✅ 大部分云存储支持 S3 协议（前端实现一样）
- ⚠️ 少部分云存储有自己的协议（前端需要特殊处理）

---

## 🏗️ 为什么需要返回上传类型？

### 问题分析

虽然大部分云存储支持 S3 协议，但确实有例外：

#### 标准 S3 上传（MinIO、COS、OSS、S3）

```typescript
// 前端实现完全一样
xhr.open('PUT', presignedURL)
xhr.setRequestHeader('Content-Type', file.type)
xhr.send(file)
```

#### 七牛云表单上传

```typescript
// 需要使用 POST + FormData
const formData = new FormData()
formData.append('token', qiniuToken)  // 七牛云的 token
formData.append('key', key)
formData.append('file', file)

xhr.open('POST', 'https://up-z2.qiniup.com')
xhr.send(formData)
```

**完全不一样！** ⚠️

---

## 🎨 解决方案：前后端都用接口抽象

### 架构图

```
┌────────────────────────────────────────────────────────────────┐
│                          前端                                   │
│  ┌──────────────────────────────────────────────────────┐      │
│  │       Uploader Interface (策略模式)                    │      │
│  │  ┌────────────────────────────────────────────┐      │      │
│  │  │  upload(credentials, file, onProgress)     │      │      │
│  │  │  cancel()                                  │      │      │
│  │  └────────────────────────────────────────────┘      │      │
│  └───────────────────┬──────────────────────────────────┘      │
│                      ↓                                          │
│          ┌───────────┴────────────┐                            │
│          │   UploaderFactory       │  (工厂模式)                │
│          │   (根据 method 创建)    │                            │
│          └───────────┬─────────────┘                            │
│                      ↓                                          │
│    ┌─────────────────┼──────────────────┬──────────────┐       │
│    ↓                 ↓                  ↓              ↓       │
│ ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│ │Presigned │  │  Form    │  │   SDK    │  │  Custom  │       │
│ │  URL     │  │ Upload   │  │ Upload   │  │  Upload  │       │
│ │(MinIO/   │  │(七牛云)  │  │(又拍云)  │  │  (扩展)  │       │
│ │ COS/OSS) │  │          │  │          │  │          │       │
│ └──────────┘  └──────────┘  └──────────┘  └──────────┘       │
└────────────────────────────────────────────────────────────────┘
                               ↑
                               │ HTTP Request
                               │ { method, url, headers, form_data, ... }
                               ↓
┌────────────────────────────────────────────────────────────────┐
│                          后端                                   │
│  ┌──────────────────────────────────────────────────────┐      │
│  │       Storage Interface                               │      │
│  │  ┌────────────────────────────────────────────┐      │      │
│  │  │  GetUploadMethod()                         │      │      │
│  │  │  GenerateUploadCredentials()              │      │      │
│  │  └────────────────────────────────────────────┘      │      │
│  └───────────────────┬──────────────────────────────────┘      │
│                      ↓                                          │
│          ┌───────────┴────────────┐                            │
│          │   StorageFactory        │  (工厂模式)                │
│          │   (根据 type 创建)      │                            │
│          └───────────┬─────────────┘                            │
│                      ↓                                          │
│    ┌─────────────────┼──────────────────┬──────────────┐       │
│    ↓                 ↓                  ↓              ↓       │
│ ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│ │  MinIO   │  │Tencent   │  │ Aliyun   │  │  Qiniu   │       │
│ │ Storage  │  │   COS    │  │   OSS    │  │  Cloud   │       │
│ │(presigned│  │(presigned│  │(presigned│  │  (form)  │       │
│ │  _url)   │  │  _url)   │  │  _url)   │  │          │       │
│ └──────────┘  └──────────┘  └──────────┘  └──────────┘       │
└────────────────────────────────────────────────────────────────┘
```

---

## 📊 后端实现

### 1. Storage Interface（存储接口）

```go
// storage/interface.go
type UploadMethod string

const (
    UploadMethodPresignedURL UploadMethod = "presigned_url" // 标准 S3
    UploadMethodFormUpload   UploadMethod = "form_upload"   // 七牛云
    UploadMethodSDKUpload    UploadMethod = "sdk_upload"    // 特殊
)

type UploadCredentials struct {
    Method UploadMethod  // ✅ 上传方式
    
    // 预签名 URL 上传
    URL     string
    Headers map[string]string
    
    // 表单上传
    FormData map[string]string
    PostURL  string
    
    // SDK 上传
    SDKConfig map[string]interface{}
}

type Storage interface {
    GetUploadMethod() UploadMethod
    GenerateUploadCredentials(...) (*UploadCredentials, error)
}
```

### 2. MinIO Implementation

```go
// storage/minio.go
func (s *MinIOStorage) GetUploadMethod() UploadMethod {
    return UploadMethodPresignedURL  // MinIO 使用预签名 URL
}

func (s *MinIOStorage) GenerateUploadCredentials(...) (*UploadCredentials, error) {
    presignedURL, _ := s.client.PresignedPutObject(...)
    
    return &UploadCredentials{
        Method: UploadMethodPresignedURL,
        URL:    presignedURL.String(),
        Headers: map[string]string{
            "Content-Type": contentType,
        },
    }, nil
}
```

### 3. Qiniu Implementation（示例）

```go
// storage/qiniu.go（未来实现）
func (s *QiniuStorage) GetUploadMethod() UploadMethod {
    return UploadMethodFormUpload  // 七牛云使用表单上传
}

func (s *QiniuStorage) GenerateUploadCredentials(...) (*UploadCredentials, error) {
    token := s.generateQiniuToken(...)
    
    return &UploadCredentials{
        Method: UploadMethodFormUpload,
        PostURL: "https://up-z2.qiniup.com",
        FormData: map[string]string{
            "token": token,
            "key": key,
        },
    }, nil
}
```

### 4. API Response

```go
// dto/storage.go
type GetUploadTokenResp struct {
    Key    string       `json:"key"`
    Bucket string       `json:"bucket"`
    Expire string       `json:"expire"`
    Method UploadMethod `json:"method"`  // ✅ 告诉前端用哪种方式上传
    
    // 预签名 URL 字段
    URL     string            `json:"url,omitempty"`
    Headers map[string]string `json:"headers,omitempty"`
    
    // 表单上传字段
    FormData map[string]string `json:"form_data,omitempty"`
    PostURL  string            `json:"post_url,omitempty"`
    
    // SDK 上传字段
    SDKConfig map[string]interface{} `json:"sdk_config,omitempty"`
}
```

---

## 🎨 前端实现

### 1. Uploader Interface（上传器接口）

```typescript
// web/src/utils/upload/index.ts
export interface Uploader {
  upload(
    credentials: UploadCredentials,
    file: File,
    onProgress: (progress: UploadProgress) => void
  ): Promise<void>
  
  cancel(): void
}
```

### 2. UploaderFactory（上传器工厂）

```typescript
export class UploaderFactory {
  static create(method: string): Uploader {
    switch (method) {
      case 'presigned_url':
        return new PresignedURLUploader()  // MinIO、COS、OSS、S3
      
      case 'form_upload':
        return new FormUploader()  // 七牛云
      
      case 'sdk_upload':
        return new SDKUploader()  // 特殊云存储
      
      default:
        throw new Error(`不支持的上传方式: ${method}`)
    }
  }
}
```

### 3. PresignedURLUploader（预签名 URL 上传器）

```typescript
// web/src/utils/upload/presigned-url.ts
export class PresignedURLUploader implements Uploader {
  private xhr: XMLHttpRequest | null = null

  async upload(credentials, file, onProgress) {
    return new Promise((resolve, reject) => {
      this.xhr = new XMLHttpRequest()
      
      // 监听进度
      this.xhr.upload.onprogress = (e) => {
        const percent = (e.loaded / e.total) * 100
        onProgress({ percent, loaded: e.loaded, total: e.total })
      }
      
      this.xhr.onload = () => resolve()
      this.xhr.onerror = () => reject(new Error('上传失败'))
      
      // HTTP PUT 上传（MinIO、COS、OSS、S3）
      this.xhr.open('PUT', credentials.url)
      this.xhr.setRequestHeader('Content-Type', file.type)
      this.xhr.send(file)
    })
  }
  
  cancel() {
    this.xhr?.abort()
  }
}
```

### 4. FormUploader（表单上传器）

```typescript
// web/src/utils/upload/form-upload.ts
export class FormUploader implements Uploader {
  private xhr: XMLHttpRequest | null = null

  async upload(credentials, file, onProgress) {
    return new Promise((resolve, reject) => {
      this.xhr = new XMLHttpRequest()
      
      this.xhr.upload.onprogress = (e) => {
        onProgress({ percent: (e.loaded / e.total) * 100, ... })
      }
      
      // 构建表单数据
      const formData = new FormData()
      Object.entries(credentials.form_data).forEach(([key, value]) => {
        formData.append(key, value)
      })
      formData.append('file', file)
      
      // HTTP POST 上传（七牛云）
      this.xhr.open('POST', credentials.post_url)
      this.xhr.send(formData)
    })
  }
}
```

### 5. 统一入口

```typescript
// web/src/utils/upload/index.ts
export async function uploadFile(
  router: string,
  file: File,
  onProgress: (progress: UploadProgress) => void
): Promise<string> {
  
  // 1. 获取上传凭证（包含 method）
  const credentials = await getUploadCredentials(router, file)
  
  // 2. 根据 method 创建对应的上传器
  const uploader = UploaderFactory.create(credentials.method)
  
  // 3. 执行上传
  await uploader.upload(credentials, file, onProgress)
  
  return credentials.key
}
```

---

## 🔄 完整流程

### 1. 用户选择文件

```typescript
const file = e.target.files[0]
```

### 2. 前端请求上传凭证

```typescript
POST /api/v1/storage/upload_token
{
  "router": "luobei/test88888/plugins/cashier_desk",
  "file_name": "invoice.pdf",
  "file_size": 102400
}
```

### 3. 后端返回上传凭证（包含上传方式）

#### MinIO/COS/OSS/S3 响应：

```json
{
  "method": "presigned_url",  // ✅ 告诉前端使用预签名 URL
  "key": "luobei/test88888/plugins/cashier_desk/2025/11/03/xxx.pdf",
  "bucket": "ai-agent-os",
  "url": "http://localhost:9000/ai-agent-os/xxx.pdf?X-Amz-Signature=...",
  "headers": {
    "Content-Type": "application/pdf"
  }
}
```

#### 七牛云响应（未来）：

```json
{
  "method": "form_upload",  // ✅ 告诉前端使用表单上传
  "key": "luobei/test88888/plugins/cashier_desk/2025/11/03/xxx.pdf",
  "post_url": "https://up-z2.qiniup.com",
  "form_data": {
    "token": "qiniu_token_xxx",
    "key": "xxx.pdf"
  }
}
```

### 4. 前端根据 method 选择上传器

```typescript
const uploader = UploaderFactory.create(credentials.method)
// method = "presigned_url" → PresignedURLUploader
// method = "form_upload"   → FormUploader
// method = "sdk_upload"    → SDKUploader
```

### 5. 执行上传（不同的实现）

#### 预签名 URL 上传：

```typescript
xhr.open('PUT', credentials.url)
xhr.setRequestHeader('Content-Type', file.type)
xhr.send(file)
```

#### 表单上传：

```typescript
const formData = new FormData()
formData.append('token', credentials.form_data.token)
formData.append('key', credentials.form_data.key)
formData.append('file', file)

xhr.open('POST', credentials.post_url)
xhr.send(formData)
```

### 6. 监听进度（统一接口）

```typescript
xhr.upload.onprogress = (e) => {
  onProgress({
    percent: (e.loaded / e.total) * 100,
    loaded: e.loaded,
    total: e.total
  })
}
```

### 7. 上传完成通知后端

```typescript
POST /api/v1/storage/upload_complete
{
  "key": "xxx.pdf",
  "success": true
}
```

---

## ✅ 优势总结

### 1. **前后端都用接口抽象**

```
后端：Storage Interface → MinIO/COS/OSS/Qiniu 实现
前端：Uploader Interface → PresignedURL/Form/SDK 实现
```

### 2. **返回上传方式**

```json
{ "method": "presigned_url" }  // 告诉前端用哪种上传器
```

### 3. **工厂模式创建**

```typescript
// 前端
const uploader = UploaderFactory.create(method)

// 后端
const storage = StorageFactory.create(type)
```

### 4. **统一进度监控**

```typescript
// 所有上传器都支持进度回调
onProgress({ percent, loaded, total })
```

### 5. **易于扩展**

```typescript
// 新增存储：添加新的 Uploader 实现
class NewStorageUploader implements Uploader {
  upload(...) { /* 特定实现 */ }
}
```

---

## 🎯 结论

你的思考**完全正确**！

1. ✅ **需要返回上传类型**（`method` 字段）
2. ✅ **前端需要根据类型选择不同的上传实现**
3. ✅ **前后端都用接口抽象**（SOLID 原则）
4. ✅ **工厂模式创建实例**
5. ✅ **统一的进度监控**

这是**企业级**的架构设计！🎉
