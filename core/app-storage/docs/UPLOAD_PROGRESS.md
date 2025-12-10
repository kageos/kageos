# 文件上传进度实现方案

## 🎯 问题

前端直接上传到 MinIO，如何显示上传进度？

```
客户端 → (直接上传) → MinIO
```

因为不经过 app-storage，所以需要前端自己实现进度监控。

---

## 📊 解决方案

### 方案 A：XMLHttpRequest + Progress 事件（推荐）

#### 优点
- ✅ 原生支持，兼容性好
- ✅ 进度回调准确
- ✅ 支持取消上传

#### 实现代码

```typescript
// 完整上传流程（带进度）
async function uploadFileWithProgress(
  router: string,
  file: File,
  onProgress: (percent: number) => void
): Promise<string> {
  
  // 1. 获取上传凭证
  const tokenRes = await fetch('/api/v1/storage/upload_token', {
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
  });
  
  const { url, key } = (await tokenRes.json()).data;
  
  // 2. 使用 XMLHttpRequest 上传（支持进度）
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    
    // 监听上传进度
    xhr.upload.addEventListener('progress', (e) => {
      if (e.lengthComputable) {
        const percent = Math.round((e.loaded / e.total) * 100);
        onProgress(percent);
      }
    });
    
    // 监听上传完成
    xhr.addEventListener('load', async () => {
      if (xhr.status === 200) {
        // 3. 通知后端上传成功
        await fetch('/api/v1/storage/upload_complete', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Token': getJwtToken(),
          },
          body: JSON.stringify({ key, success: true }),
        });
        
        resolve(key);
      } else {
        // 上传失败
        await fetch('/api/v1/storage/upload_complete', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Token': getJwtToken(),
          },
          body: JSON.stringify({
            key,
            success: false,
            error: `Upload failed: ${xhr.statusText}`,
          }),
        });
        
        reject(new Error(`Upload failed: ${xhr.statusText}`));
      }
    });
    
    // 监听错误
    xhr.addEventListener('error', async () => {
      await fetch('/api/v1/storage/upload_complete', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Token': getJwtToken(),
        },
        body: JSON.stringify({
          key,
          success: false,
          error: 'Network error',
        }),
      });
      
      reject(new Error('Network error'));
    });
    
    // 发起上传请求
    xhr.open('PUT', url);
    xhr.setRequestHeader('Content-Type', file.type);
    xhr.send(file);
  });
}
```

#### 使用示例

```vue
<template>
  <div>
    <input type="file" @change="handleFileChange" />
    <el-progress v-if="uploading" :percentage="uploadPercent" />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const uploading = ref(false)
const uploadPercent = ref(0)

async function handleFileChange(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  
  uploading.value = true
  uploadPercent.value = 0
  
  try {
    const router = 'luobei/test88888/plugins/cashier_desk'
    const key = await uploadFileWithProgress(router, file, (percent) => {
      uploadPercent.value = percent
    })
    
    console.log('上传成功:', key)
    ElMessage.success('上传成功')
  } catch (err) {
    console.error('上传失败:', err)
    ElMessage.error('上传失败')
  } finally {
    uploading.value = false
  }
}
</script>
```

---

### 方案 B：Axios + onUploadProgress

#### 优点
- ✅ 语法简洁
- ✅ 支持拦截器
- ✅ 支持取消上传

#### 实现代码

```typescript
import axios, { AxiosProgressEvent } from 'axios'

async function uploadFileWithAxios(
  router: string,
  file: File,
  onProgress: (percent: number) => void
): Promise<string> {
  
  // 1. 获取上传凭证
  const tokenRes = await axios.post('/api/v1/storage/upload_token', {
    router,
    file_name: file.name,
    content_type: file.type,
    file_size: file.size,
  }, {
    headers: { 'X-Token': getJwtToken() },
  })
  
  const { url, key } = tokenRes.data.data
  
  try {
    // 2. 上传到 MinIO（带进度）
    await axios.put(url, file, {
      headers: { 'Content-Type': file.type },
      onUploadProgress: (progressEvent: AxiosProgressEvent) => {
        if (progressEvent.total) {
          const percent = Math.round((progressEvent.loaded / progressEvent.total) * 100)
          onProgress(percent)
        }
      },
    })
    
    // 3. 通知后端上传成功
    await axios.post('/api/v1/storage/upload_complete', {
      key,
      success: true,
    }, {
      headers: { 'X-Token': getJwtToken() },
    })
    
    return key
    
  } catch (err) {
    // 上传失败，通知后端
    await axios.post('/api/v1/storage/upload_complete', {
      key,
      success: false,
      error: err.message,
    }, {
      headers: { 'X-Token': getJwtToken() },
    })
    
    throw err
  }
}
```

---

### 方案 C：fetch + ReadableStream（复杂）

#### 缺点
- ❌ 不支持 upload progress（只支持 download progress）
- ❌ 需要手动封装
- ❌ 兼容性略差

**不推荐使用**，因为 `fetch` 不支持监听上传进度。

---

## 🎨 Vue 组件封装

### 完整的文件上传组件

```vue
<!-- FileUploader.vue -->
<template>
  <div class="file-uploader">
    <el-upload
      :auto-upload="false"
      :on-change="handleFileChange"
      :show-file-list="false"
      drag
    >
      <el-icon class="el-icon--upload"><upload-filled /></el-icon>
      <div class="el-upload__text">
        将文件拖到此处，或<em>点击上传</em>
      </div>
    </el-upload>
    
    <!-- 上传进度 -->
    <div v-if="uploading" class="upload-progress">
      <div class="file-info">
        <el-icon><document /></el-icon>
        <span>{{ fileName }}</span>
        <span class="file-size">{{ formatSize(fileSize) }}</span>
      </div>
      
      <el-progress :percentage="uploadPercent" />
      
      <div class="upload-status">
        <span v-if="uploadPercent < 100">
          上传中... {{ uploadPercent }}%
        </span>
        <span v-else class="success">
          <el-icon><check /></el-icon>
          上传成功
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { UploadFilled, Document, Check } from '@element-plus/icons-vue'

const props = defineProps<{
  router: string  // 函数路径
}>()

const emit = defineEmits<{
  success: [key: string]
  error: [error: Error]
}>()

const uploading = ref(false)
const uploadPercent = ref(0)
const fileName = ref('')
const fileSize = ref(0)

async function handleFileChange(file: any) {
  const rawFile = file.raw
  if (!rawFile) return
  
  fileName.value = rawFile.name
  fileSize.value = rawFile.size
  uploading.value = true
  uploadPercent.value = 0
  
  try {
    const key = await uploadFileWithProgress(
      props.router,
      rawFile,
      (percent) => {
        uploadPercent.value = percent
      }
    )
    
    ElMessage.success('上传成功')
    emit('success', key)
    
    // 2 秒后隐藏进度条
    setTimeout(() => {
      uploading.value = false
    }, 2000)
    
  } catch (err) {
    ElMessage.error(`上传失败: ${err.message}`)
    emit('error', err)
    uploading.value = false
  }
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(2) + ' MB'
}

// 导入上面的 uploadFileWithProgress 函数
// ...
</script>

<style scoped>
.file-uploader {
  padding: 20px;
}

.upload-progress {
  margin-top: 20px;
  padding: 15px;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  background: #f5f7fa;
}

.file-info {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  font-size: 14px;
}

.file-size {
  color: #909399;
  margin-left: auto;
}

.upload-status {
  margin-top: 8px;
  font-size: 13px;
  color: #606266;
}

.upload-status .success {
  color: #67c23a;
  display: flex;
  align-items: center;
  gap: 4px;
}
</style>
```

### 使用示例

```vue
<template>
  <div>
    <h2>收银台 - 文件上传</h2>
    
    <FileUploader
      router="luobei/test88888/tools/cashier_desk"
      @success="handleUploadSuccess"
      @error="handleUploadError"
    />
    
    <!-- 已上传的文件列表 -->
    <div v-if="uploadedFiles.length > 0">
      <h3>已上传文件</h3>
      <ul>
        <li v-for="file in uploadedFiles" :key="file.key">
          {{ file.name }} - {{ file.key }}
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import FileUploader from './FileUploader.vue'

const uploadedFiles = ref<Array<{ key: string; name: string }>>([])

function handleUploadSuccess(key: string) {
  console.log('文件上传成功:', key)
  uploadedFiles.value.push({
    key,
    name: key.split('/').pop() || key,
  })
}

function handleUploadError(error: Error) {
  console.error('文件上传失败:', error)
}
</script>
```

---

## 🚀 高级功能

### 1. 大文件分片上传

对于超大文件（>100MB），建议使用分片上传：

```typescript
async function uploadLargeFile(
  router: string,
  file: File,
  onProgress: (percent: number) => void
): Promise<string> {
  const chunkSize = 5 * 1024 * 1024 // 5MB per chunk
  const totalChunks = Math.ceil(file.size / chunkSize)
  
  // MinIO 支持 Multipart Upload，但需要使用 minio-js SDK
  // 这里简化处理，实际项目中可以封装
  
  // ... 分片上传逻辑
}
```

### 2. 断点续传

```typescript
// 保存上传进度到 localStorage
function saveUploadProgress(key: string, loaded: number) {
  localStorage.setItem(`upload_${key}`, loaded.toString())
}

// 恢复上传
function resumeUpload(key: string): number {
  const loaded = localStorage.getItem(`upload_${key}`)
  return loaded ? parseInt(loaded) : 0
}
```

### 3. 多文件并发上传

```typescript
async function uploadMultipleFiles(
  router: string,
  files: File[],
  onProgress: (overall: number, details: number[]) => void
): Promise<string[]> {
  const progressMap = new Map<number, number>()
  
  const promises = files.map((file, index) => {
    return uploadFileWithProgress(router, file, (percent) => {
      progressMap.set(index, percent)
      
      // 计算总进度
      const overall = Array.from(progressMap.values())
        .reduce((sum, p) => sum + p, 0) / files.length
      
      onProgress(overall, Array.from(progressMap.values()))
    })
  })
  
  return Promise.all(promises)
}
```

---

## 📊 总结

| 方案 | 难度 | 兼容性 | 推荐度 |
|------|------|--------|--------|
| **XMLHttpRequest** | 简单 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ 推荐 |
| **Axios** | 简单 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ 推荐 |
| **fetch** | 复杂 | ⭐⭐⭐⭐ | ❌ 不推荐 |

### 核心要点

1. ✅ **使用 XMLHttpRequest 或 Axios**：原生支持上传进度
2. ✅ **监听 progress 事件**：实时更新进度条
3. ✅ **上传完成后调用 upload_complete**：通知后端，记录审计
4. ✅ **处理错误情况**：网络错误、上传失败都要通知后端
5. ✅ **用户体验**：显示文件名、大小、进度百分比

### 关键流程

```
1. 前端请求 upload_token
   ↓
2. 后端记录上传意图（status = pending）
   ↓
3. 前端使用 XMLHttpRequest 上传到 MinIO（监听 progress）
   ↓
4. 前端调用 upload_complete（success = true/false）
   ↓
5. 后端更新状态（status = completed/failed）
```

完美！这样既能监控上传进度，又能完整记录审计日志！🎉

