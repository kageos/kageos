# FilesWidget 使用示例

## 场景 1：发票管理系统

### 后端字段定义

```json
{
  "code": "invoice_files",
  "name": "发票附件",
  "widget": {
    "type": "files",
    "config": {
      "accept": ".pdf,image/*",
      "max_size": "10MB",
      "max_count": 3
    }
  }
}
```

### 效果

- 只允许上传 PDF 和图片
- 单个文件不超过 10MB
- 最多 3 个文件

---

## 场景 2：合同管理系统

### 后端字段定义

```json
{
  "code": "contract_files",
  "name": "合同文件",
  "widget": {
    "type": "files",
    "config": {
      "accept": ".pdf,.doc,.docx",
      "max_size": "50MB",
      "max_count": 10
    }
  }
}
```

### 效果

- 只允许上传 PDF 和 Word 文档
- 单个文件不超过 50MB
- 最多 10 个文件

---

## 场景 3：多媒体资料

### 后端字段定义

```json
{
  "code": "media_files",
  "name": "多媒体资料",
  "widget": {
    "type": "files",
    "config": {
      "accept": "image/*,video/*,audio/*",
      "max_size": "100MB",
      "max_count": 20
    }
  }
}
```

### 效果

- 允许上传图片、视频、音频
- 单个文件不超过 100MB
- 最多 20 个文件

---

## 场景 4：不限制类型

### 后端字段定义

```json
{
  "code": "all_files",
  "name": "所有文件",
  "widget": {
    "type": "files",
    "config": {
      "accept": "*",
      "max_size": "",
      "max_count": 5
    }
  }
}
```

### 效果

- 不限制文件类型
- 不限制文件大小
- 最多 5 个文件

---

## 提交数据示例

### 前端提交的数据

```json
{
  "invoice_files": {
    "files": [
      {
        "name": "invoice_2024_001.pdf",
        "description": "2024年1月发票",
        "hash": "",
        "size": 1024000,
        "upload_ts": 1704067200000,
        "local_path": "",
        "is_uploaded": true,
        "url": "luobei/test88888/tools/cashier_desk/2024/01/invoice_2024_001.pdf",
        "downloaded": false
      },
      {
        "name": "invoice_photo.jpg",
        "description": "发票照片",
        "hash": "",
        "size": 512000,
        "upload_ts": 1704067300000,
        "local_path": "",
        "is_uploaded": true,
        "url": "luobei/test88888/tools/cashier_desk/2024/01/invoice_photo.jpg",
        "downloaded": false
      }
    ],
    "remark": "本月所有发票",
    "metadata": {}
  }
}
```

### 后端接收的数据（Go 结构）

```go
type InvoiceRequest struct {
    InvoiceFiles Files `json:"invoice_files"`
}

type Files struct {
    Files    []*File                `json:"files"`
    Remark   string                 `json:"remark"`
    Metadata map[string]interface{} `json:"metadata"`
}

type File struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Hash        string `json:"hash"`
    Size        int64  `json:"size"`
    UploadTs    int64  `json:"upload_ts"`
    LocalPath   string `json:"local_path"`
    IsUploaded  bool   `json:"is_uploaded"`
    Url         string `json:"url"`
    Downloaded  bool   `json:"downloaded,omitempty"`
}
```

---

## 用户操作流程

### 1. 拖拽上传

```
用户拖拽 invoice.pdf 到上传区域
  ↓
FilesWidget 验证文件类型（✅ .pdf）
  ↓
FilesWidget 验证文件大小（✅ 5MB < 10MB）
  ↓
FilesWidget 验证文件数量（✅ 1 < 3）
  ↓
FilesWidget 调用 uploadFile()
  ↓
显示上传进度：0% → 50% → 100%
  ↓
上传成功，添加到文件列表
```

### 2. 添加文件描述

```
用户点击文件卡片中的描述输入框
  ↓
输入："2024年1月发票"
  ↓
FilesWidget 更新文件描述
  ↓
更新 formManager 的字段值
```

### 3. 下载文件

```
用户点击"下载"按钮
  ↓
FilesWidget 调用 /api/v1/storage/file/url
  ↓
后端返回预签名下载链接
  ↓
在新标签页打开下载链接
```

### 4. 删除文件

```
用户点击"删除"按钮
  ↓
显示确认弹窗："确定删除此文件？"
  ↓
用户点击"确定"
  ↓
FilesWidget 从文件列表中移除
  ↓
更新 formManager 的字段值
```

### 5. 添加备注

```
用户在备注输入框输入："本月所有发票"
  ↓
FilesWidget 更新备注
  ↓
更新 formManager 的字段值
```

### 6. 提交表单

```
用户点击"提交"按钮
  ↓
FormRenderer 收集所有字段值
  ↓
包括 invoice_files 的完整数据
  ↓
发送到后端 API
```

---

## 表格中的显示

### TableRenderer 中的渲染

```vue
<el-table-column label="附件" prop="invoice_files">
  <template #default="{ row }">
    <!-- FilesWidget.renderTableCell() -->
    <div class="files-table-cell">
      <el-tag size="small">2 个文件</el-tag>
      <div class="file-item">
        <el-icon><document /></el-icon>
        <span>invoice_2024_001.pdf</span>
      </div>
      <div class="file-item">
        <el-icon><document /></el-icon>
        <span>invoice_photo.jpg</span>
      </div>
    </div>
  </template>
</el-table-column>
```

---

## 技术细节

### 上传流程

```typescript
// FilesWidget.handleFileSelect()
async handleFileSelect(rawFile: File) {
  // 1. 验证文件
  if (!this.validateFile(rawFile)) return

  // 2. 添加到上传列表（显示进度）
  const uid = `${Date.now()}_${Math.random()}`
  this.uploadingFiles.value.push({ uid, name, size, percent: 0, status: 'uploading' })

  // 3. 调用统一上传工具
  const key = await uploadFile(
    this.router,  // 例如：luobei/test88888/tools/cashier_desk
    rawFile,
    (progress) => {
      // 更新进度
      file.percent = progress.percent
    }
  )

  // 4. 上传成功，添加到文件列表
  const newFile: FileItem = {
    name: rawFile.name,
    description: '',
    hash: '',
    size: rawFile.size,
    upload_ts: Date.now(),
    local_path: '',
    is_uploaded: true,
    url: key,  // 文件 Key
    downloaded: false,
  }

  this.updateFiles([...currentFiles, newFile])
}
```

### uploadFile 内部流程

```typescript
// utils/upload/index.ts
export async function uploadFile(router, file, onProgress) {
  // 1. 请求后端获取上传凭证
  const credentials = await getUploadCredentials(router, file)
  // credentials = {
  //   method: "presigned_url",
  //   url: "http://localhost:9000/ai-agent-os/luobei/test88888/.../invoice.pdf?...",
  //   upload_host: "localhost:9000",
  //   upload_domain: "http://localhost:9000",
  //   headers: { "Content-Type": "application/pdf" }
  // }

  // 2. 根据 method 创建上传器
  const uploader = UploaderFactory.create(credentials.method)
  // uploader = new PresignedURLUploader()

  // 3. 执行上传
  await uploader.upload(credentials, file, onProgress)

  // 4. 通知后端上传完成
  await notifyUploadComplete(credentials.key, true)

  return credentials.key
}
```

---

## 存储路径示例

### MinIO 存储结构

```
ai-agent-os (bucket)
  └── luobei (tenant)
      └── test88888 (app)
          └── tools (package)
              └── cashier_desk (function)
                  └── 2024
                      └── 01
                          ├── invoice_2024_001.pdf
                          └── invoice_photo.jpg
```

### 文件 Key 格式

```
luobei/test88888/tools/cashier_desk/2024/01/invoice_2024_001.pdf
```

这样设计的好处：
1. 多租户隔离
2. 按函数分类
3. 便于统计和清理
4. 支持按时间归档

---

## 错误处理

### 文件类型不匹配

```
用户上传 .exe 文件，但 accept 限制为 .pdf
  ↓
FilesWidget.validateFile() 返回 false
  ↓
显示错误提示："不支持的文件类型，仅支持：.pdf"
  ↓
不执行上传
```

### 文件大小超限

```
用户上传 20MB 文件，但 max_size 限制为 10MB
  ↓
FilesWidget.validateFile() 返回 false
  ↓
显示错误提示："文件大小不能超过 10MB"
  ↓
不执行上传
```

### 文件数量超限

```
已上传 3 个文件，max_count 限制为 3
  ↓
FilesWidget.validateFile() 返回 false
  ↓
显示错误提示："最多只能上传 3 个文件"
  ↓
不执行上传
```

### 上传失败

```
网络错误或后端错误
  ↓
uploadFile() 抛出异常
  ↓
FilesWidget 捕获异常
  ↓
更新上传状态为 'error'
  ↓
显示错误提示："上传失败: 网络错误"
```

---

## 总结

FilesWidget 提供了完整的文件上传解决方案：

1. ✅ **易用性**：拖拽上传，自动验证
2. ✅ **可配置**：支持灵活的类型、大小、数量限制
3. ✅ **进度显示**：实时显示上传进度
4. ✅ **文件管理**：描述、下载、删除
5. ✅ **错误处理**：友好的错误提示
6. ✅ **扩展性**：支持不同的存储后端
7. ✅ **类型安全**：完整的 TypeScript 类型定义
8. ✅ **组件自治**：不依赖外部状态，完全独立

