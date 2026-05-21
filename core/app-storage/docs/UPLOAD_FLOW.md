# 文件上传完整流程

## 🎯 核心问题

**用户拖文件到上传组件时，上传组件怎么知道用什么域名上传？**

**答案：先请求后端获取上传凭证（包含域名），再执行上传！**

---

## 📊 完整流程

### 流程图

```
┌─────────────────────────────────────────────────────────────────┐
│                    用户操作                                        │
│              拖文件到上传组件 / 点击选择文件                        │
└────────────────────┬────────────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────────────┐
│                  前端上传组件                                      │
│              FileUpload.vue                                      │
│              handleFileSelect(file)                              │
└────────────────────┬────────────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────────────┐
│           调用统一上传函数                                         │
│           uploadFile(router, file, onProgress)                   │
└────────────────────┬────────────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────────────┐
│        ✨ Step 1: 获取上传凭证（包含域名）                         │
│        getUploadCredentials(router, file)                       │
│                                                                   │
│        POST /api/v1/storage/upload_token                        │
│        {                                                         │
│          router: "luobei/test88888/tools/cashier_desk",         │
│          file_name: "invoice.pdf",                               │
│          file_size: 102400                                       │
│        }                                                         │
└────────────────────┬────────────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────────────┐
│                    后端处理                                        │
│              StorageService.GenerateUploadToken()                │
│                                                                   │
│              1. 根据 MinIO 配置生成预签名 URL                     │
│              2. 调用 MinIOStorage.GenerateUploadCredentials()     │
│              3. 从预签名 URL 中提取域名信息                        │
│              4. 返回上传凭证                                      │
└────────────────────┬────────────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────────────┐
│              后端返回上传凭证（包含域名）                           │
│        {                                                         │
│          method: "presigned_url",                               │
│          upload_url: "http://localhost:9000/...?X-Amz-Signature=...",  │
│          upload_host: "localhost:9000",        ✨ 上传 host      │
│          upload_domain: "http://localhost:9000", ✨ 上传域名      │
│          headers: { "Content-Type": "application/pdf" },         │
│          key: "luobei/test88888/.../xxx.pdf",                    │
│          bucket: "kageos",                                 │
│          cdn_domain: "https://cdn.example.com"                   │
│        }                                                         │
└────────────────────┬────────────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────────────┐
│        ✨ Step 2: 创建预签名 URL 上传器                            │
│        createUploader(credentials.method)                       │
│                                                                   │
│        method = "presigned_url" → PresignedURLUploader           │
└────────────────────┬────────────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────────────┐
│        ✨ Step 3: 执行上传（此时已知道上传域名）                   │
│        uploader.upload(credentials, file, onProgress)          │
│                                                                   │
│        xhr.open('PUT', credentials.upload_url)  // 使用预签名 URL         │
│        xhr.upload.onprogress = (e) => {                          │
│          console.log(`上传到 ${credentials.upload_domain} ...`)  │
│          onProgress({                                            │
│            percent: ...,                                         │
│            uploadDomain: credentials.upload_domain  ✨          │
│          })                                                      │
│        }                                                         │
│        xhr.send(file)                                            │
└────────────────────┬────────────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────────────┐
│                  上传完成                                          │
│        notifyUploadComplete(key, true)                          │
│        POST /api/v1/storage/upload_complete                     │
└─────────────────────────────────────────────────────────────────┘
```

---

## 💻 代码示例

### 1. 前端组件（FileUpload.vue）

```vue
<template>
  <el-upload
    :auto-upload="false"
    :on-change="handleFileSelect"
    drag
  >
    <div>拖拽文件到此处</div>
  </el-upload>

  <!-- 上传进度（显示上传域名） -->
  <div v-if="uploading">
    <div>上传到: {{ uploadDomain }}</div>
    <el-progress :percentage="uploadPercent" />
  </div>
</template>

<script setup>
async function handleFileSelect(file) {
  // ✨ 调用上传函数，内部会先请求后端获取上传凭证（包含域名）
  const key = await uploadFile(
    props.router,
    file.raw,
    (progress) => {
      // 进度回调（此时已知道上传域名）
      uploadPercent.value = progress.percent
      uploadDomain.value = progress.uploadDomain  // ✨ 从后端返回的凭证中获取
    }
  )
}
</script>
```

### 2. 统一上传函数（utils/upload/index.ts）

```typescript
export async function uploadFile(router, file, onProgress) {
  // ✨ Step 1: 请求后端获取上传凭证（包含域名）
  const credentials = await getUploadCredentials(router, file)
  // credentials = {
  //   method: "presigned_url",
  //   upload_url: "http://localhost:9000/...",
  //   upload_host: "localhost:9000",
  //   upload_domain: "http://localhost:9000",  // ✨ 此时已知道上传域名
  //   ...
  // }
  
  // ✨ Step 2: 校验 method，并创建预签名 URL 上传器
  const uploader = createUploader(credentials.method)
  
  // ✨ Step 3: 执行上传（传递域名信息）
  await uploader.upload(credentials, file, (progress) => {
    onProgress({
      ...progress,
      uploadDomain: credentials.upload_domain,  // ✨ 传递上传域名
    })
  })
}
```

### 3. 获取上传凭证（utils/upload/index.ts）

```typescript
async function getUploadCredentials(router, file) {
  // 请求后端 API
  const res = await fetch('/api/v1/storage/upload_token', {
    method: 'POST',
    body: JSON.stringify({
      router,
      file_name: file.name,
      file_size: file.size,
    }),
  })
  
  const { data } = await res.json()
  
  // ✨ 后端返回的凭证包含域名信息
  // data = {
  //   method: "presigned_url",
  //   upload_host: "localhost:9000",
  //   upload_domain: "http://localhost:9000",
  //   upload_url: "http://localhost:9000/...?X-Amz-Signature=...",
  //   ...
  // }
  
  return data
}
```

### 4. 上传器实现（utils/upload/presigned-url.ts）

```typescript
export class PresignedURLUploader implements Uploader {
  async upload(credentials, file, onProgress) {
    // ✨ 此时已知道上传域名（从 credentials 中获取）
    const uploadDomain = credentials.upload_domain
    console.log(`开始上传到: ${uploadDomain}`)
    
    const xhr = new XMLHttpRequest()
    
    // 监听上传进度
    xhr.upload.onprogress = (e) => {
      onProgress({
        percent: (e.loaded / e.total) * 100,
        uploadDomain,  // ✨ 传递上传域名
      })
    }
    
    // 使用预签名 URL（包含完整域名）
    xhr.open('PUT', credentials.upload_url)
    xhr.send(file)
  }
}
```

---

## 🔑 关键点

### 1. **先请求后端，再上传**

```
用户拖文件
  ↓
调用 uploadFile()
  ↓
先请求后端获取上传凭证（包含域名）  ✨ 关键步骤
  ↓
后端返回：{ upload_domain, url, ... }
  ↓
根据凭证执行上传
```

### 2. **域名从哪里来？**

```
后端流程：
  1. 读取 MinIO 配置
  2. 调用 MinIOStorage.GenerateUploadCredentials()
  3. 生成预签名 URL（例如：http://localhost:9000/...）
  4. 从预签名 URL 中提取域名信息
  5. 返回：{ upload_host, upload_domain, url, ... }
```

### 3. **前端如何使用域名？**

```typescript
// 从后端返回的凭证中获取
const uploadDomain = credentials.upload_domain

// 在进度回调中传递
onProgress({
  percent: 50,
  uploadDomain,  // ✨ 前端组件可以使用
})

// 在组件中显示
<div>上传到: {{ uploadDomain }}</div>
```

---

## 📝 完整示例

### 使用 FileUpload 组件

```vue
<template>
  <div>
    <h2>文件上传</h2>
    
    <FileUpload
      router="luobei/test88888/tools/cashier_desk"
      @success="handleUploadSuccess"
      @error="handleUploadError"
    />
  </div>
</template>

<script setup>
import FileUpload from '@/components/FileUpload.vue'

function handleUploadSuccess(key, fileName) {
  console.log('上传成功:', key, fileName)
}

function handleUploadError(error) {
  console.error('上传失败:', error)
}
</script>
```

### 流程说明

1. **用户拖文件** → `FileUpload.vue` 的 `handleFileSelect()` 被调用
2. **调用上传函数** → `uploadFile(router, file, onProgress)`
3. **请求后端** → `getUploadCredentials()` → 后端返回上传凭证（包含域名）
4. **创建上传器** → `createUploader(method)`
5. **执行上传** → `uploader.upload()` → 此时已知道上传域名
6. **进度回调** → `onProgress({ uploadDomain, ... })` → 前端组件显示域名

---

## ✅ 总结

**回答你的问题：**

> 用户拖文件到上传组件时，上传组件怎么知道用什么域名上传？

**答案：**

1. ✅ **先请求后端**：调用 `/api/v1/storage/upload_token` 获取上传凭证
2. ✅ **后端返回域名**：凭证中包含 `upload_host` 和 `upload_domain`
3. ✅ **再执行上传**：使用返回的凭证（包含域名）执行上传
4. ✅ **显示域名**：在进度回调中传递域名，前端组件显示

**关键流程**：
```
拖文件 → 请求后端获取凭证（包含域名）→ 执行上传（已知道域名）
```

**这是标准的"先获取凭证，再上传"的流程！** 🎉
