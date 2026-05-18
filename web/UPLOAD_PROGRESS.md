# 上传进度监听机制详解

> 状态：草案
> 更新时间：2026-05-17
> 负责人窗口：事项 8 / codex/document-archive-cleanup

## ❓ 常见误解

### 误解：需要后端提供进度监听接口

**错误理解**：
```
前端上传文件 → 后端 8083 接收文件 → 后端转发到 MinIO
                 ↑
                 需要后端提供进度接口？❌
```

**正确理解**：
```
前端直接上传到 MinIO（使用预签名 URL）
  ↓
前端监听 XMLHttpRequest.upload.onprogress
  ↓
无需后端参与进度监听 ✅
```

---

## ✅ 正确的上传流程

### MinIO/COS/OSS/S3（预签名 URL）

```
┌─────────────────────────────────────────────────────────────────┐
│ Step 1: 获取上传凭证                                              │
│                                                                   │
│ 前端 → POST /api/v1/storage/upload_token                        │
│        { router, file_name, file_size, content_type }           │
│   ↓                                                              │
│ 后端 app-storage (8083)                                         │
│   ├─ 生成预签名 URL                                              │
│   └─ 返回 { method: "presigned_url", url: "http://...", ... }   │
│   ↓                                                              │
│ 前端收到凭证                                                      │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ Step 2: 直接上传到 MinIO（不经过后端 8083）                      │
│                                                                   │
│ 前端 → XMLHttpRequest PUT http://localhost:9000/...?signature=..│
│   ↓                                                              │
│ MinIO (9000)                                                    │
│   ├─ 接收文件流                                                  │
│   └─ 存储文件                                                     │
│   ↓                                                              │
│ 前端监听进度：xhr.upload.onprogress ✅                           │
│   ├─ e.loaded / e.total = 进度百分比                             │
│   └─ 实时更新 UI                                                 │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ Step 3: 通知后端上传完成                                          │
│                                                                   │
│ 前端 → POST /api/v1/storage/upload_complete                     │
│        { key, success: true }                                    │
│   ↓                                                              │
│ 后端 app-storage (8083)                                         │
│   └─ 更新 file_uploads 表状态为 "completed"                      │
└─────────────────────────────────────────────────────────────────┘
```

---

## 📊 进度监听原理

### 1. MinIO/COS/OSS/S3（预签名 URL）

**关键点**：前端直接上传到存储服务，浏览器原生支持进度监听

```typescript
// utils/upload/presigned-url.ts
export class PresignedURLUploader implements Uploader {
  async upload(credentials, file, onProgress) {
    const xhr = new XMLHttpRequest()

    // ✅ 浏览器原生支持进度监听（无需后端参与）
    xhr.upload.addEventListener('progress', (e) => {
      if (e.lengthComputable) {
        const percent = Math.round((e.loaded / e.total) * 100)
        const speed = this.calculateSpeed(e.loaded)

        onProgress({
          percent,      // 上传百分比
          loaded: e.loaded,  // 已上传字节
          total: e.total,    // 总字节数
          speed,        // 上传速度
        })
      }
    })

    // ✅ 直接上传到 MinIO（使用预签名 URL）
    xhr.open('PUT', credentials.url)  // http://localhost:9000/...
    xhr.setRequestHeader('Content-Type', file.type)
    xhr.send(file)
  }
}
```

**为什么不需要后端提供进度接口？**

1. **浏览器原生能力**：`XMLHttpRequest.upload.onprogress` 是浏览器提供的标准 API
2. **直接上传**：前端直接连接到 MinIO（9000 端口），不经过后端
3. **TCP 层监听**：浏览器在 TCP 层监听发送进度，无需应用层参与

---

### 2. 七牛云/又拍云（表单上传）

**关键点**：也是直接上传到七牛云，浏览器同样可以监听进度

```typescript
// utils/upload/form-upload.ts
export class FormUploader implements Uploader {
  async upload(credentials, file, onProgress) {
    const xhr = new XMLHttpRequest()

    // ✅ 浏览器原生支持进度监听（无需后端参与）
    xhr.upload.addEventListener('progress', (e) => {
      if (e.lengthComputable) {
        onProgress({
          percent: Math.round((e.loaded / e.total) * 100),
          loaded: e.loaded,
          total: e.total,
        })
      }
    })

    // 构建表单数据
    const formData = new FormData()
    Object.entries(credentials.form_data).forEach(([key, value]) => {
      formData.append(key, value)  // token、key 等
    })
    formData.append('file', file)

    // ✅ 直接上传到七牛云（不经过后端）
    xhr.open('POST', credentials.post_url)  // https://upload.qiniup.com/...
    xhr.send(formData)
  }
}
```

---

### 3. SDK 上传（特殊云存储）

**关键点**：云存储 SDK 提供进度回调

```typescript
// utils/upload/sdk-upload.ts
export class SDKUploader implements Uploader {
  async upload(credentials, file, onProgress) {
    const sdk = createSDK(credentials.sdk_config)

    // ✅ SDK 提供进度回调（无需后端参与）
    await sdk.upload(file, {
      onProgress: (percent, loaded, total) => {
        onProgress({ percent, loaded, total })
      }
    })
  }
}
```

---

## 🔧 技术细节

### XMLHttpRequest.upload.onprogress

**浏览器原生 API**，不需要服务器支持：

```javascript
xhr.upload.addEventListener('progress', (event) => {
  if (event.lengthComputable) {
    const percent = (event.loaded / event.total) * 100
    console.log(`上传进度: ${percent}%`)
  }
})
```

**event 对象属性**：
- `event.loaded`：已上传的字节数
- `event.total`：总字节数
- `event.lengthComputable`：是否可计算长度（通常为 true）

**关键点**：
1. 这是浏览器在 **TCP 层** 监听的，不需要服务器返回进度
2. 只要是 HTTP(S) 上传，都可以监听进度
3. 适用于任何存储服务（MinIO、七牛云、阿里云、腾讯云...）

---

## 🚫 不需要后端进度接口的原因

### 错误方案（不需要这样做）

```
前端 → 后端 8083 → MinIO
        ↑
        后端提供进度接口？❌
```

**问题**：
1. 后端成为瓶颈（所有文件都要经过后端）
2. 占用后端带宽和内存
3. 增加延迟
4. 复杂度高

### 正确方案（预签名 URL）

```
前端 → 直接上传到 MinIO
  ↓
浏览器监听 xhr.upload.onprogress ✅
```

**优势**：
1. 后端不参与文件传输（只提供凭证）
2. 直连存储服务，速度快
3. 浏览器原生支持进度监听
4. 简单高效

---

## 📦 实际示例

### 完整的上传进度显示

```vue
<template>
  <div class="upload-progress">
    <el-progress :percentage="uploadPercent" />
    <div>速度: {{ uploadSpeed }}</div>
    <div>已上传: {{ formatSize(uploadedSize) }} / {{ formatSize(totalSize) }}</div>
    <div>上传到: {{ uploadDomain }}</div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { uploadFile } from '@/utils/upload'

const uploadPercent = ref(0)
const uploadedSize = ref(0)
const totalSize = ref(0)
const uploadSpeed = ref('')
const uploadDomain = ref('')

async function handleUpload(file) {
  const startTime = Date.now()

  try {
    // ✅ 调用统一上传函数
    const key = await uploadFile(
      'luobei/test88888/plugins/cashier_desk',
      file,
      (progress) => {
        // ✅ 实时更新进度（浏览器原生提供）
        uploadPercent.value = progress.percent
        uploadedSize.value = progress.loaded
        totalSize.value = progress.total
        uploadDomain.value = progress.uploadDomain

        // 计算速度
        const elapsed = (Date.now() - startTime) / 1000
        const speed = progress.loaded / elapsed
        uploadSpeed.value = formatSpeed(speed)
      }
    )

    console.log('上传成功，文件 Key:', key)
  } catch (error) {
    console.error('上传失败:', error)
  }
}
</script>
```

---

## 🎯 总结

### MinIO 上传进度监听

| 问题 | 答案 |
|-----|------|
| 需要后端提供进度接口吗？ | ❌ **不需要** |
| 前端如何监听进度？ | ✅ 使用 `XMLHttpRequest.upload.onprogress` |
| 文件经过后端吗？ | ❌ **不经过**，直接上传到 MinIO |
| 后端只负责什么？ | ✅ 生成预签名 URL 凭证 |

### 七牛云上传进度监听

| 问题 | 答案 |
|-----|------|
| 需要后端提供进度接口吗？ | ❌ **不需要** |
| 前端如何监听进度？ | ✅ 使用 `XMLHttpRequest.upload.onprogress` |
| 文件经过后端吗？ | ❌ **不经过**，直接上传到七牛云 |
| 后端只负责什么？ | ✅ 生成七牛云上传 token |

### 关键优势

1. ✅ **性能高**：前端直连存储服务，不经过后端
2. ✅ **带宽省**：后端不参与文件传输
3. ✅ **进度准**：浏览器原生监听，无延迟
4. ✅ **代码简**：不需要实现后端进度接口

---

## 🔗 相关文档

- [XMLHttpRequest.upload - MDN](https://developer.mozilla.org/zh-CN/docs/Web/API/XMLHttpRequest/upload)
- [ProgressEvent - MDN](https://developer.mozilla.org/zh-CN/docs/Web/API/ProgressEvent)
- [MinIO Presigned URLs](https://min.io/docs/minio/linux/developers/go/API.html#presignedputobject)
- [AWS S3 Presigned URLs](https://docs.aws.amazon.com/AmazonS3/latest/userguide/PresignedUrlUploadObject.html)

---

**结论：前端直接上传到存储服务，使用浏览器原生 API 监听进度，后端只负责生成上传凭证，不需要提供进度监听接口！** 🎉
