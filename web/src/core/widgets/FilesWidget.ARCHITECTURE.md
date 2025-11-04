# FilesWidget 架构设计

## 🎯 设计目标

1. **组件自治**：FilesWidget 完全独立，不依赖外部状态
2. **依赖倒置**：依赖抽象的上传工具，不依赖具体的 MinIO
3. **多云支持**：支持 MinIO、七牛云、阿里云 OSS、腾讯云 COS、AWS S3 等
4. **后端驱动**：前端不关心存储类型，由后端告诉前端用哪种方式

---

## 📐 三层架构

```
┌─────────────────────────────────────────────────────────┐
│                    业务层                                 │
│              FilesWidget.ts                              │
│       (只关心"上传文件"，不关心"怎么上传")                 │
│       - handleFileSelect()                               │
│       - validateFile()                                   │
│       - updateFiles()                                    │
└────────────────────┬────────────────────────────────────┘
                     ↓ 调用
┌─────────────────────────────────────────────────────────┐
│                   抽象层                                  │
│              uploadFile() - 统一入口                      │
│       1. getUploadCredentials(router, file)              │
│          ↓ POST /api/v1/storage/upload_token            │
│          ↓ 返回 { method, url, headers, ... }            │
│       2. UploaderFactory.create(method)                  │
│       3. uploader.upload(credentials, file, onProgress)  │
│       4. notifyUploadComplete(key, success)              │
└────────────────────┬────────────────────────────────────┘
                     ↓ 委托
┌─────────────────────────────────────────────────────────┐
│                   策略层                                  │
│              UploaderFactory                             │
│   ├─ PresignedURLUploader (MinIO/COS/OSS/S3)            │
│   │   - 使用 XMLHttpRequest PUT                          │
│   │   - 监听 upload.onprogress                           │
│   ├─ FormUploader (七牛云/又拍云)                        │
│   │   - 使用 FormData POST                               │
│   │   - 监听 upload.onprogress                           │
│   └─ SDKUploader (特殊云存储)                            │
│       - 使用云存储 SDK                                    │
│       - SDK 提供进度回调                                  │
└─────────────────────────────────────────────────────────┘
```

---

## 🔄 数据流

### 完整上传流程

```
用户拖文件到 FilesWidget
  ↓
FilesWidget.handleFileSelect(file)
  ├─ validateFile(file)  // 验证类型、大小、数量
  │   ├─ 检查文件类型（accept）
  │   ├─ 检查文件大小（max_size）
  │   └─ 检查文件数量（max_count）
  ├─ uploadingFiles.push({ uid, name, size, percent: 0, status: 'uploading' })
  └─ uploadFile(this.router, file, onProgress)
      ↓
      ├─ Step 1: getUploadCredentials(router, file)
      │   ↓ POST /api/v1/storage/upload_token
      │   ↓ Body: { router, file_name, file_size, content_type }
      │   ↓ 后端处理：
      │   │   ├─ 根据配置的存储类型（minio/cos/oss）
      │   │   ├─ 调用 Storage.GenerateUploadCredentials()
      │   │   ├─ 生成预签名 URL 或表单凭证
      │   │   └─ 返回 { method, url, headers, upload_host, upload_domain, ... }
      │   ↓ 返回 credentials
      │
      ├─ Step 2: UploaderFactory.create(credentials.method)
      │   ↓ 根据 method 创建对应的上传器
      │   ├─ "presigned_url" → PresignedURLUploader
      │   ├─ "form_upload" → FormUploader
      │   └─ "sdk_upload" → SDKUploader
      │
      ├─ Step 3: uploader.upload(credentials, file, onProgress)
      │   ↓ 执行上传
      │   ├─ xhr.open('PUT', credentials.url)  // PresignedURL
      │   ├─ xhr.upload.onprogress = (e) => onProgress({ percent, ... })
      │   └─ xhr.send(file)
      │   ↓ 上传到 MinIO/COS/OSS/S3
      │
      └─ Step 4: notifyUploadComplete(key, true)
          ↓ POST /api/v1/storage/upload_complete
          ↓ 后端更新 file_uploads 表状态
  ↓
FilesWidget 更新状态
  ├─ uploadingFiles[uid].status = 'success'
  ├─ uploadingFiles[uid].percent = 100
  └─ updateFiles([...currentFiles, newFile])
      ↓ formManager.setValue(fieldPath, { raw: { files, remark, metadata }, ... })
  ↓
ElMessage.success('上传成功')
```

---

## 🔑 关键设计

### 1. 后端驱动前端行为

**后端配置**：
```yaml
# configs/app-storage.yaml
storage:
  type: "minio"  # 或 "tencentcos" / "aliyunoss" / "awss3"
```

**后端返回**：
```json
{
  "method": "presigned_url",
  "url": "http://localhost:9000/ai-agent-os/luobei/.../file.pdf?X-Amz-Signature=...",
  "headers": { "Content-Type": "application/pdf" },
  "upload_host": "localhost:9000",
  "upload_domain": "http://localhost:9000",
  "key": "luobei/test88888/tools/cashier_desk/2024/11/file.pdf",
  "bucket": "ai-agent-os",
  "expire": "2024-11-04T11:30:00Z"
}
```

**前端处理**：
```typescript
// ✅ 前端只需要根据 method 创建上传器，无需关心具体存储类型
const uploader = UploaderFactory.create(credentials.method)
await uploader.upload(credentials, file, onProgress)
```

**切换存储类型**：
```yaml
# 后端配置改为腾讯云 COS
storage:
  type: "tencentcos"
```

**前端无需修改！** ✅

---

### 2. 策略模式实现多云支持

```typescript
// 上传器接口（策略接口）
export interface Uploader {
  upload(
    credentials: UploadCredentials,
    file: File,
    onProgress: (progress: UploadProgress) => void
  ): Promise<void>
  
  cancel(): void
}

// 具体策略 1：预签名 URL（MinIO、COS、OSS、S3）
export class PresignedURLUploader implements Uploader {
  async upload(credentials, file, onProgress) {
    const xhr = new XMLHttpRequest()
    xhr.open('PUT', credentials.url)
    xhr.upload.onprogress = (e) => onProgress({ percent: ... })
    xhr.send(file)
  }
}

// 具体策略 2：表单上传（七牛云、又拍云）
export class FormUploader implements Uploader {
  async upload(credentials, file, onProgress) {
    const formData = new FormData()
    Object.entries(credentials.form_data).forEach(([k, v]) => formData.append(k, v))
    formData.append('file', file)
    
    const xhr = new XMLHttpRequest()
    xhr.open('POST', credentials.post_url)
    xhr.upload.onprogress = (e) => onProgress({ percent: ... })
    xhr.send(formData)
  }
}

// 具体策略 3：SDK 上传（特殊云存储）
export class SDKUploader implements Uploader {
  async upload(credentials, file, onProgress) {
    const sdk = createSDK(credentials.sdk_config)
    await sdk.upload(file, { onProgress })
  }
}
```

---

### 3. Router 的传递链

```
TableRenderer
  ↓ functionData.router = "luobei/test88888/tools/cashier_desk"
  ↓ :router="props.functionData.router"
FormDialog
  ↓ props.router = "luobei/test88888/tools/cashier_desk"
  ↓ formFunctionDetail.router = props.router
FormRenderer
  ↓ functionDetail.router = "luobei/test88888/tools/cashier_desk"
  ↓ formRendererContext.getFunctionRouter()
FilesWidget
  ↓ this.router = this.formRenderer.getFunctionRouter()
  ↓ uploadFile(this.router, file, onProgress)
后端上传服务
  ↓ 使用 router 构建文件存储路径
MinIO
  ↓ Key: luobei/test88888/tools/cashier_desk/2024/11/file.pdf
```

---

## 🛡️ 安全边界

### 1. 临时 Widget vs 标准 Widget

```typescript
constructor(props: WidgetRenderProps) {
  super(props)
  
  // ✅ 获取 router（如果是临时 Widget 则为空）
  this.router = this.getRouter()
  
  // ✅ 只有标准 Widget 才初始化空值
  if (!this.isTemporary && (!this.value.value || this.value.value.raw === null)) {
    this.initializeEmptyValue()
  }
}

render() {
  // ✅ 临时 Widget（表格渲染）只显示简单的文件列表
  if (this.isTemporary) {
    return this.renderTableCell()
  }
  
  // ✅ 标准 Widget 显示完整上传界面
  return h('div', { class: 'files-widget' }, [
    // 上传区域
    // 已上传文件列表
    // 备注
  ])
}

async handleFileSelect(rawFile: File) {
  // ✅ 临时 Widget 不支持上传
  if (this.isTemporary) {
    ElMessage.error('临时组件不支持文件上传')
    return
  }
  
  // ✅ 检查 router 是否存在
  if (!this.router) {
    ElMessage.error('缺少函数路径，无法上传文件')
    return
  }
  
  // ... 执行上传
}
```

### 2. 文件验证

```typescript
private validateFile(file: File): boolean {
  const maxSize = this.parseMaxSize(this.filesConfig.max_size)
  const maxCount = this.filesConfig.max_count || 5
  const currentFiles = this.getCurrentFiles()

  // ✅ 检查数量限制
  if (currentFiles.length >= maxCount) {
    ElMessage.error(`最多只能上传 ${maxCount} 个文件`)
    return false
  }

  // ✅ 检查大小限制
  if (file.size > maxSize) {
    ElMessage.error(`文件大小不能超过 ${this.filesConfig.max_size}`)
    return false
  }

  // ✅ 检查文件类型
  if (this.filesConfig.accept && this.filesConfig.accept !== '*') {
    const accept = this.filesConfig.accept.split(',').map(a => a.trim())
    const fileName = file.name.toLowerCase()
    const fileType = file.type.toLowerCase()

    const isAccepted = accept.some(pattern => {
      if (pattern.startsWith('.')) return fileName.endsWith(pattern)
      if (pattern.includes('/*')) return fileType.startsWith(pattern.split('/')[0])
      return fileType === pattern
    })

    if (!isAccepted) {
      ElMessage.error(`不支持的文件类型，仅支持：${this.filesConfig.accept}`)
      return false
    }
  }

  return true
}
```

---

## 📦 数据结构

### FilesData（对应后端 Go）

```typescript
interface FilesData {
  files: FileItem[]        // 文件列表
  remark: string          // 备注
  metadata: Record<string, any>  // 元数据
}

interface FileItem {
  name: string           // 文件名
  description: string    // 文件描述
  hash: string          // 文件哈希
  size: number          // 文件大小（字节）
  upload_ts: number     // 上传时间戳
  local_path: string    // 本地路径
  is_uploaded: boolean  // 是否已上传到云端
  url: string           // 文件 Key/URL
  downloaded?: boolean  // 是否已下载到本地
}
```

---

## 🚀 扩展性

### 添加新的云存储（例如：华为云 OBS）

#### 后端（Go）

```go
// 1. 实现 Storage 接口
// core/app-storage/storage/huawei_obs.go
type HuaweiOBSStorage struct {
    client *obs.Client
}

func (s *HuaweiOBSStorage) GenerateUploadCredentials(...) (*UploadCredentials, error) {
    // 如果支持预签名 URL
    return &UploadCredentials{
        Method: UploadMethodPresignedURL,
        URL:    presignedURL,
        Headers: map[string]string{
            "Content-Type": contentType,
        },
    }, nil
}

// 2. 注册到工厂
// core/app-storage/storage/factory.go
func NewFactory(cfg storage.Config) (storage.Storage, error) {
    switch cfg.Storage.Type {
    case "huaweiobs":
        return NewHuaweiOBSStorage(cfg)
    // ... 其他
    }
}
```

#### 前端（TypeScript）

**如果支持预签名 URL（S3 兼容）**：
```
无需修改！✅
```

**如果需要特殊 SDK**：
```typescript
// 1. 实现 Uploader 接口
// utils/upload/huawei-obs.ts
export class HuaweiOBSUploader implements Uploader {
  async upload(credentials, file, onProgress) {
    const obsClient = new ObsClient(credentials.sdk_config)
    await obsClient.putObject({
      Bucket: credentials.sdk_config.bucket,
      Key: credentials.sdk_config.objectKey,
      Body: file,
      ProgressCallback: (transferred, total) => {
        onProgress({ percent: (transferred / total) * 100, ... })
      }
    })
  }
}

// 2. 注册到工厂
// utils/upload/index.ts
case 'sdk_upload':
  if (credentials.sdk_config.provider === 'huawei') {
    return new HuaweiOBSUploader()
  }
  return new SDKUploader()
```

---

## ✅ 总结

1. **组件自治**：FilesWidget 完全独立，不依赖外部状态
2. **依赖倒置**：依赖 `uploadFile` 抽象工具，不依赖具体的 MinIO
3. **策略模式**：不同的上传方式作为不同的策略
4. **工厂模式**：根据后端返回的 `method` 创建对应的上传器
5. **后端驱动**：前端不关心存储类型，由后端告诉前端用哪种方式
6. **易于扩展**：新增云存储，前端无需修改（如果是标准 S3 协议）

**这个架构可以支持任何云存储，前端无需修改！** 🎉

