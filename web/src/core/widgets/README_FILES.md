# FilesWidget 文件上传组件

## 组件特性

### ✅ 组件自治
- 完全独立的文件上传逻辑
- 不依赖外部状态管理
- 自行处理上传进度、错误、成功状态

### ✅ 依赖倒置
- 依赖抽象的 `uploadFile` 工具，而非具体的 MinIO 实现
- 依赖 `FormRendererContext` 接口，而非具体的 FormRenderer 实现
- 通过配置驱动，支持不同的存储后端（MinIO、COS、OSS、S3）

### ✅ 遵循 SOLID 原则
- **单一职责**：只负责文件上传和管理
- **开闭原则**：通过配置扩展，无需修改代码
- **里氏替换**：可替换任何 `BaseWidget`
- **接口隔离**：只使用必需的接口方法
- **依赖倒置**：依赖抽象，不依赖具体实现

---

## 数据结构

### FilesData（对应后端 Go 结构）

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

## 配置参数

```typescript
interface FilesConfig {
  accept?: string      // 文件类型限制（例如：.pdf,.doc,image/*）
  max_size?: string    // 单个文件最大大小（例如：10MB, 1GB）
  max_count?: number   // 最大文件数量，默认 5
}
```

### Accept 支持的格式

1. **扩展名**：`.pdf`, `.doc`, `.docx`, `.jpg`, `.png`
2. **MIME 类型**：`application/pdf`, `image/jpeg`
3. **MIME 通配符**：`image/*`, `video/*`, `audio/*`
4. **混合使用**：`.pdf,image/*,video/*,application/zip`

---

## 使用示例

### 后端字段定义

```json
{
  "code": "attachment",
  "name": "附件",
  "widget": {
    "type": "files",
    "config": {
      "accept": ".pdf,.doc,.docx,image/*",
      "max_size": "10MB",
      "max_count": 5
    }
  }
}
```

### 前端自动渲染

FilesWidget 会自动：
1. 渲染拖拽上传区域
2. 验证文件类型和大小
3. 调用 `uploadFile()` 上传到 MinIO
4. 显示上传进度
5. 管理已上传的文件列表
6. 支持文件描述、下载、删除

---

## 上传流程

```
用户拖拽/选择文件
  ↓
FilesWidget.handleFileSelect()
  ↓
验证文件（类型、大小、数量）
  ↓
调用 uploadFile(router, file, onProgress)
  ↓
uploadFile 内部流程：
  1. 请求后端获取上传凭证（包含域名）
  2. 根据 method 创建对应的上传器
  3. 执行上传（显示进度）
  4. 通知后端上传完成
  ↓
更新 FilesWidget 状态
  ↓
更新 formManager 的字段值
```

---

## 依赖关系

```
FilesWidget
  ├─ BaseWidget（继承）
  ├─ uploadFile（工具函数，抽象的上传逻辑）
  │   ├─ PresignedURLUploader（MinIO、COS、OSS、S3）
  │   ├─ FormUploader（七牛云、又拍云）
  │   └─ SDKUploader（特殊云存储）
  └─ FormRendererContext（接口，获取 router）
```

**关键点**：
- `FilesWidget` 不直接依赖 MinIO
- `FilesWidget` 通过 `uploadFile` 工具函数实现上传
- `uploadFile` 根据后端返回的 `method` 动态选择上传器
- 后端切换存储类型（MinIO → COS → OSS），前端无需修改

---

## 扩展性

### 添加新的存储后端

**后端**：
1. 在 `app-storage` 添加新的 `Storage` 实现（例如：`AliyunOSSStorage`）
2. 更新 `StorageFactory` 注册新类型
3. 修改 `configs/app-storage.yaml` 配置

**前端**：
1. 如果是标准 S3 协议，无需修改
2. 如果需要特殊上传方式，在 `utils/upload` 添加新的 `Uploader`
3. 更新 `UploaderFactory` 注册新类型

**FilesWidget 完全无需修改！** ✅

---

## 表格渲染

FilesWidget 支持在表格中渲染：

```vue
<!-- TableRenderer.vue -->
<el-table-column label="附件">
  <template #default="{ row }">
    <!-- FilesWidget.renderTableCell() -->
    <el-tag>3 个文件</el-tag>
    <div>invoice.pdf</div>
    <div>contract.docx</div>
    <span>+1</span>
  </template>
</el-table-column>
```

---

## 技术亮点

1. ✅ **组件自治**：完全独立的上传逻辑，不依赖外部状态
2. ✅ **依赖倒置**：依赖抽象接口，不依赖具体实现
3. ✅ **策略模式**：`uploadFile` + `UploaderFactory` 实现不同上传策略
4. ✅ **工厂模式**：`WidgetFactory` 动态创建组件
5. ✅ **类型安全**：完整的 TypeScript 类型定义
6. ✅ **错误处理**：完善的错误提示和日志记录
7. ✅ **用户体验**：拖拽上传、进度显示、文件管理
8. ✅ **扩展性强**：新增存储后端，前端无需修改

---

## 注意事项

### 1. router 参数
FilesWidget 需要 `router` 参数来构建上传路径。它通过 `this.formRenderer.getFunctionRouter()` 获取。

### 2. 临时 Widget
如果是临时 Widget（`formRenderer === null`），则无法上传文件。这是合理的，因为文件上传需要函数上下文。

### 3. 文件下载
下载文件时，需要调用 `/api/v1/storage/file/url` 获取预签名下载链接。

### 4. 文件删除
删除文件只是从前端状态中移除，不会删除云端文件。如果需要删除云端文件，应在后端实现。

---

## 未来优化

1. **拖拽排序**：支持拖拽调整文件顺序
2. **批量上传**：支持一次选择多个文件
3. **断点续传**：大文件支持断点续传
4. **预览功能**：图片、PDF 等文件的预览
5. **压缩功能**：上传前自动压缩图片
6. **秒传功能**：利用文件哈希实现秒传

