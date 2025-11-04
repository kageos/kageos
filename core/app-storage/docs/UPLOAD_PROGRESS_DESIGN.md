# 上传进度监控设计方案

## 🎯 核心问题

**不同存储引擎（MinIO、腾讯云 COS、阿里云 OSS、AWS S3）的上传机制可能不同，如何统一实现上传进度监控？**

---

## 📊 上传模式对比

### 模式 A：前端直接上传（推荐）⭐

```
前端 → (HTTP PUT) → 存储服务 (MinIO/COS/OSS/S3)
  ↓
XMLHttpRequest.upload.onprogress
```

**特点**：
- ✅ **通用性强**：所有存储都支持 HTTP PUT
- ✅ **进度准确**：浏览器原生支持
- ✅ **节省带宽**：不经过后端
- ✅ **简单高效**：无需额外开发

**适用场景**：
- ✅ 所有存储引擎（MinIO、COS、OSS、S3）
- ✅ 小文件上传（<100MB）
- ✅ 对上传内容无需后端审核

---

### 模式 B：后端代理上传

```
前端 → (HTTP POST) → app-storage → 存储服务
  ↓                      ↓
progress              转发并计算进度
```

**特点**：
- ✅ **内容可控**：后端可以审核、扫描
- ✅ **统一处理**：后端统一逻辑
- ❌ **占用带宽**：文件经过后端
- ❌ **复杂度高**：需要实现流式传输

**适用场景**：
- ✅ 需要病毒扫描、内容审核
- ✅ 需要加密、水印处理
- ❌ 大文件上传（会占用服务器资源）

---

### 模式 C：分片上传（大文件）

```
前端 → 分片1 → 存储服务
     → 分片2 → 存储服务
     → 分片N → 存储服务
       ↓
    合并分片
```

**特点**：
- ✅ **支持大文件**：>1GB
- ✅ **支持断点续传**
- ✅ **并发上传**：提高速度
- ❌ **实现复杂**：需要管理分片

**适用场景**：
- ✅ 超大文件（>100MB）
- ✅ 网络不稳定场景

---

## 🚀 推荐方案：前端直接上传（通用）

### 核心思路

**所有存储引擎都支持 HTTP PUT 上传，浏览器原生支持进度监控，无需区分存储类型！**

---

## 🎨 实现方案

### 1. 后端：Storage 接口（已完成）

```go
// storage/interface.go
type Storage interface {
    // 生成上传预签名 URL（所有存储都支持）
    GenerateUploadURL(ctx context.Context, bucket, key, contentType string, expire time.Duration) (url string, err error)
}
```

**MinIO、COS、OSS、S3 都返回标准的 HTTP PUT URL**

---

### 2. 前端：统一上传方法（通用）

```typescript
// utils/upload.ts

/**
 * 通用文件上传（支持所有存储引擎）
 */
export async function uploadFileWithProgress(
  router: string,
  file: File,
  onProgress: (percent: number, loaded: number, total: number) => void
): Promise<string> {
  
  // Step 1: 获取上传凭证（与存储类型无关）
  const { url, key } = await getUploadToken(router, file)
  
  // Step 2: 使用 XMLHttpRequest 上传（通用方案）
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    
    // 监听上传进度（浏览器原生支持，与存储无关）
    xhr.upload.addEventListener('progress', (e) => {
      if (e.lengthComputable) {
        const percent = Math.round((e.loaded / e.total) * 100)
        onProgress(percent, e.loaded, e.total)
      }
    })
    
    // 上传完成
    xhr.addEventListener('load', async () => {
      if (xhr.status === 200) {
        // 通知后端上传成功
        await notifyUploadComplete(key, true)
        resolve(key)
      } else {
        await notifyUploadComplete(key, false, xhr.statusText)
        reject(new Error(`上传失败: ${xhr.statusText}`))
      }
    })
    
    // 上传失败
    xhr.addEventListener('error', async () => {
      await notifyUploadComplete(key, false, '网络错误')
      reject(new Error('网络错误'))
    })
    
    // 上传中断
    xhr.addEventListener('abort', async () => {
      await notifyUploadComplete(key, false, '用户取消')
      reject(new Error('上传已取消'))
    })
    
    // 发起上传（HTTP PUT，所有存储都支持）
    xhr.open('PUT', url)
    xhr.setRequestHeader('Content-Type', file.type)
    xhr.send(file)
  })
}

/**
 * 获取上传凭证（后端接口，与存储类型无关）
 */
async function getUploadToken(router: string, file: File) {
  const res = await fetch('/api/v1/storage/upload_token', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Token': getJwtToken(),
    },
    body: JSON.stringify({
      router,
      file_name: file.name,
      content_type: file.type,
      file_size: file.size,
    }),
  })
  
  const { data } = await res.json()
  return data
}

/**
 * 通知后端上传完成（用于审计）
 */
async function notifyUploadComplete(key: string, success: boolean, error?: string) {
  await fetch('/api/v1/storage/upload_complete', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Token': getJwtToken(),
    },
    body: JSON.stringify({ key, success, error }),
  })
}
```

---

### 3. Vue 组件示例

```vue
<template>
  <div class="upload-container">
    <el-upload
      :auto-upload="false"
      :on-change="handleFileSelect"
      :show-file-list="false"
      drag
    >
      <el-icon class="el-icon--upload"><upload-filled /></el-icon>
      <div class="el-upload__text">
        拖拽文件到此处，或<em>点击上传</em>
      </div>
    </el-upload>
    
    <!-- 上传进度 -->
    <div v-if="uploading" class="upload-progress">
      <div class="file-info">
        <span>{{ fileName }}</span>
        <span class="file-size">{{ formatSize(uploadedSize) }} / {{ formatSize(totalSize) }}</span>
      </div>
      
      <el-progress 
        :percentage="uploadPercent" 
        :status="uploadStatus"
        :stroke-width="12"
      />
      
      <div class="upload-speed">
        <span v-if="uploadPercent < 100">
          速度: {{ uploadSpeed }}
        </span>
        <span v-else-if="uploadStatus === 'success'" class="success">
          <el-icon><check /></el-icon>
          上传成功
        </span>
        <span v-else-if="uploadStatus === 'exception'" class="error">
          <el-icon><close /></el-icon>
          上传失败
        </span>
      </div>
      
      <!-- 取消按钮 -->
      <el-button 
        v-if="uploadPercent < 100" 
        @click="cancelUpload"
        type="danger"
        size="small"
      >
        取消上传
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { uploadFileWithProgress } from '@/utils/upload'

const props = defineProps<{
  router: string  // 函数路径
}>()

const emit = defineEmits<{
  success: [key: string]
  error: [error: Error]
}>()

const uploading = ref(false)
const uploadPercent = ref(0)
const uploadStatus = ref<'success' | 'exception' | undefined>()
const fileName = ref('')
const uploadedSize = ref(0)
const totalSize = ref(0)
const startTime = ref(0)
const currentXHR = ref<XMLHttpRequest | null>(null)

// 计算上传速度
const uploadSpeed = computed(() => {
  if (!startTime.value || uploadedSize.value === 0) return '0 KB/s'
  
  const elapsed = (Date.now() - startTime.value) / 1000  // 秒
  const speed = uploadedSize.value / elapsed  // 字节/秒
  
  if (speed < 1024) return `${speed.toFixed(0)} B/s`
  if (speed < 1024 * 1024) return `${(speed / 1024).toFixed(2)} KB/s`
  return `${(speed / (1024 * 1024)).toFixed(2)} MB/s`
})

async function handleFileSelect(file: any) {
  const rawFile = file.raw
  if (!rawFile) return
  
  fileName.value = rawFile.name
  totalSize.value = rawFile.size
  uploading.value = true
  uploadPercent.value = 0
  uploadedSize.value = 0
  uploadStatus.value = undefined
  startTime.value = Date.now()
  
  try {
    const key = await uploadFileWithProgress(
      props.router,
      rawFile,
      (percent, loaded, total) => {
        uploadPercent.value = percent
        uploadedSize.value = loaded
        totalSize.value = total
      }
    )
    
    uploadStatus.value = 'success'
    ElMessage.success('上传成功')
    emit('success', key)
    
    // 2 秒后隐藏
    setTimeout(() => {
      uploading.value = false
    }, 2000)
    
  } catch (err: any) {
    uploadStatus.value = 'exception'
    ElMessage.error(`上传失败: ${err.message}`)
    emit('error', err)
  }
}

function cancelUpload() {
  if (currentXHR.value) {
    currentXHR.value.abort()
    uploading.value = false
    ElMessage.warning('已取消上传')
  }
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(2) + ' MB'
}
</script>

<style scoped>
.upload-container {
  padding: 20px;
}

.upload-progress {
  margin-top: 20px;
  padding: 20px;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  background: #f5f7fa;
}

.file-info {
  display: flex;
  justify-content: space-between;
  margin-bottom: 12px;
  font-size: 14px;
  color: #606266;
}

.file-size {
  color: #909399;
}

.upload-speed {
  margin-top: 8px;
  font-size: 13px;
  color: #606266;
}

.success {
  color: #67c23a;
  display: flex;
  align-items: center;
  gap: 4px;
}

.error {
  color: #f56c6c;
  display: flex;
  align-items: center;
  gap: 4px;
}
</style>
```

---

## 🎯 为什么这个方案通用？

### 1. **HTTP 标准协议**

| 存储 | 上传方式 | 前端实现 |
|------|---------|---------|
| MinIO | HTTP PUT + 预签名 URL | XMLHttpRequest ✅ |
| 腾讯云 COS | HTTP PUT + 预签名 URL | XMLHttpRequest ✅ |
| 阿里云 OSS | HTTP PUT + 预签名 URL | XMLHttpRequest ✅ |
| AWS S3 | HTTP PUT + 预签名 URL | XMLHttpRequest ✅ |

**所有云存储都遵循 S3 协议，前端实现完全一致！**

### 2. **浏览器原生支持**

```typescript
xhr.upload.addEventListener('progress', (e) => {
  // e.loaded: 已上传字节数
  // e.total: 文件总大小
  // 所有浏览器原生支持，与存储无关
})
```

### 3. **后端抽象接口**

```go
// 所有存储实现都返回标准的预签名 URL
type Storage interface {
    GenerateUploadURL(...) (url string, err error)
}

// MinIO 实现
func (s *MinIOStorage) GenerateUploadURL(...) (string, error) {
    return s.client.PresignedPutObject(...)  // 返回 HTTP PUT URL
}

// 腾讯云 COS 实现
func (s *TencentCOSStorage) GenerateUploadURL(...) (string, error) {
    return s.client.GetPresignedURL(http.MethodPut, ...)  // 返回 HTTP PUT URL
}

// 阿里云 OSS 实现
func (s *AliyunOSSStorage) GenerateUploadURL(...) (string, error) {
    return s.bucket.SignURL(key, http.MethodPut, ...)  // 返回 HTTP PUT URL
}
```

**前端无需关心后端用的是哪个存储！**

---

## 🚀 高级功能扩展

### 1. 大文件分片上传

如果未来需要支持大文件（>1GB），可以扩展 Storage 接口：

```go
// storage/interface.go
type Storage interface {
    // ... 现有方法
    
    // 分片上传（可选，大文件支持）
    InitiateMultipartUpload(ctx context.Context, bucket, key string) (uploadID string, err error)
    UploadPart(ctx context.Context, bucket, key, uploadID string, partNumber int, data io.Reader) (etag string, err error)
    CompleteMultipartUpload(ctx context.Context, bucket, key, uploadID string, parts []Part) error
}
```

### 2. 后端代理上传

如果需要后端审核文件内容：

```go
// storage/interface.go
type Storage interface {
    // ... 现有方法
    
    // 流式上传（可选，代理上传支持）
    UploadStream(ctx context.Context, bucket, key string, reader io.Reader, size int64, onProgress func(loaded int64)) error
}
```

前端改为上传到 `app-storage`：

```typescript
const formData = new FormData()
formData.append('file', file)

const xhr = new XMLHttpRequest()
xhr.upload.onprogress = (e) => {
  onProgress(Math.round((e.loaded / e.total) * 100))
}
xhr.open('POST', '/api/v1/storage/upload')
xhr.send(formData)
```

---

## 📊 方案对比总结

| 特性 | 前端直接上传 | 后端代理上传 | 分片上传 |
|------|-------------|-------------|---------|
| **通用性** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |
| **实现复杂度** | ⭐⭐⭐⭐⭐（简单）| ⭐⭐⭐（中等）| ⭐⭐（复杂）|
| **服务器带宽** | ⭐⭐⭐⭐⭐（不占用）| ⭐（占用大）| ⭐⭐⭐⭐ |
| **进度准确性** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **文件大小支持** | ⭐⭐⭐（<100MB）| ⭐⭐（<100MB）| ⭐⭐⭐⭐⭐（GB级）|
| **内容可控** | ⭐（无法审核）| ⭐⭐⭐⭐⭐ | ⭐（无法审核）|

---

## ✅ 最终推荐

### 当前阶段（MVP）

✅ **使用前端直接上传 + XMLHttpRequest**

**理由**：
1. ✅ 简单高效，快速上线
2. ✅ 通用性强，支持所有存储
3. ✅ 节省服务器资源
4. ✅ 进度监控准确

### 未来扩展（按需）

1. **需要内容审核** → 添加后端代理上传
2. **支持大文件** → 添加分片上传
3. **断点续传** → 基于分片上传实现

---

## 🎯 核心要点

1. ✅ **所有存储引擎都支持标准 HTTP PUT 上传**
2. ✅ **浏览器原生支持上传进度监控**
3. ✅ **前端实现与存储类型无关**
4. ✅ **后端通过接口抽象，返回标准预签名 URL**
5. ✅ **无需为不同存储编写不同的前端代码**

**一套代码，支持所有存储！** 🎉

