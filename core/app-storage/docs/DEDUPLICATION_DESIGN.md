# 秒传与去重架构设计

## 🎯 设计目标

1. **秒传**：相同文件只上传一次，后续秒传
2. **去重**：物理存储只保留一份，节省存储成本
3. **不堵路**：当前架构已预留，未来可无缝启用

## 📊 当前状态（已完成）

### ✅ 已预留的基础设施

#### 1. 数据库表（已创建）

```sql
-- 文件元数据表（记录文件 hash 和物理存储位置）
CREATE TABLE file_metadata (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    hash VARCHAR(64) NOT NULL UNIQUE,          -- 文件 SHA256 hash
    size BIGINT NOT NULL,                      -- 文件大小
    content_type VARCHAR(100),                 -- MIME 类型
    storage_key VARCHAR(500) NOT NULL,         -- MinIO 中的实际存储位置
    ref_count INT DEFAULT 1,                   -- 引用计数
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_hash (hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 文件引用表（记录哪些函数使用了哪些文件）
CREATE TABLE file_references (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    file_id BIGINT NOT NULL,                   -- 关联 file_metadata.id
    router VARCHAR(500) NOT NULL,              -- 函数路径
    logical_key VARCHAR(500) NOT NULL UNIQUE,  -- 逻辑 Key（用户看到的）
    uploaded_by VARCHAR(100),                  -- 上传者
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_router (router),
    INDEX idx_file_id (file_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**说明**：
- 表已创建，但**当前未启用**，不影响现有功能
- 启动日志：`[Server] Database initialized successfully (tables created for future deduplication)`

#### 2. DTO 字段预留

```go
type GetUploadTokenReq struct {
    FileName    string `json:"file_name" binding:"required"`
    ContentType string `json:"content_type"`
    FileSize    int64  `json:"file_size"`
    Router      string `json:"router" binding:"required"`
    Hash        string `json:"hash,omitempty"`  // ✅ 预留字段
}
```

#### 3. 配置开关

```yaml
# configs/app-storage.yaml
minio:
  # 秒传功能（预留，未来启用）
  deduplication:
    enabled: false             # ✅ 当前关闭
    hash_algorithm: "sha256"   # 使用的 hash 算法
  
  # 缓存控制（已启用）
  cache:
    enabled: true              # ✅ 已启用 HTTP 缓存
    max_age: 31536000          # 浏览器缓存时间（秒，1年）
```

#### 4. HTTP 缓存头（已实现）

```go
// GetFileURL 生成下载链接时，自动添加缓存控制头
reqParams["response-cache-control"] = "public, max-age=31536000, immutable"
reqParams["response-expires"] = "Mon, 03 Nov 2026 12:00:00 GMT"
```

**效果**：
- 浏览器会缓存文件 1 年
- 再次下载同一文件时，直接从本地缓存读取（秒下载）
- 减少服务器压力和流量成本

---

## 🚀 未来实现（按需开启）

### 阶段 1：秒传检查 API

#### 新增 API

```go
// CheckFile 检查文件是否已存在（秒传检查）
POST /api/v1/storage/check_file
{
  "hash": "sha256-hash-of-file",
  "size": 1024000,
  "router": "luobei/test88888/tools/cashier_desk",
  "file_name": "invoice.pdf"
}

// 响应 1：文件已存在，可以秒传
{
  "code": 0,
  "data": {
    "exists": true,
    "key": "luobei/test88888/tools/cashier_desk/2025/11/03/xxx.pdf",
    "message": "文件秒传成功"
  }
}

// 响应 2：文件不存在，需要上传
{
  "code": 0,
  "data": {
    "exists": false,
    "upload_token_required": true
  }
}
```

#### 实现逻辑

```go
func (s *StorageService) CheckFile(ctx context.Context, hash string, size int64, router string, fileName string) (exists bool, key string, err error) {
    // 1. 查询数据库，看是否有相同 hash 的文件
    fileMeta, err := s.db.Where("hash = ? AND size = ?", hash, size).First(&FileMetadata{}).Error
    if err != nil {
        return false, "", nil  // 文件不存在
    }
    
    // 2. 生成逻辑 Key
    logicalKey := s.generateFileKey(router, fileName)
    
    // 3. 创建文件引用（不复制物理文件）
    s.db.Create(&FileReference{
        FileID:     fileMeta.ID,
        Router:     router,
        LogicalKey: logicalKey,
    })
    
    // 4. 增加引用计数
    s.db.Model(&FileMetadata{}).Where("id = ?", fileMeta.ID).UpdateColumn("ref_count", gorm.Expr("ref_count + 1"))
    
    return true, logicalKey, nil
}
```

### 阶段 2：前端 Hash 计算

```typescript
// 前端上传流程（支持秒传）
async function uploadFileWithDedup(router: string, file: File) {
  // 1. 计算文件 SHA256 hash
  const hash = await calculateSHA256(file);
  
  // 2. 检查文件是否已存在
  const checkRes = await fetch('/api/v1/storage/check_file', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Token': getJwtToken(),
    },
    body: JSON.stringify({
      hash,
      size: file.size,
      router,
      file_name: file.name,
    }),
  });
  
  const checkData = await checkRes.json();
  
  // 3. 秒传成功
  if (checkData.data.exists) {
    console.log('秒传成功！', checkData.data.key);
    return checkData.data.key;
  }
  
  // 4. 正常上传（带上 hash）
  const tokenRes = await fetch('/api/v1/storage/upload_token', {
    method: 'POST',
    body: JSON.stringify({
      router,
      file_name: file.name,
      content_type: file.type,
      file_size: file.size,
      hash,  // 带上 hash
    }),
  });
  
  // 5. 上传到 MinIO
  const tokenData = await tokenRes.json();
  await fetch(tokenData.data.url, {
    method: 'PUT',
    body: file,
  });
  
  // 6. 通知后端上传完成（记录 hash）
  await fetch('/api/v1/storage/upload_complete', {
    method: 'POST',
    body: JSON.stringify({
      key: tokenData.data.key,
      hash,
      size: file.size,
    }),
  });
  
  return tokenData.data.key;
}

// SHA256 计算
async function calculateSHA256(file: File): Promise<string> {
  const buffer = await file.arrayBuffer();
  const hashBuffer = await crypto.subtle.digest('SHA-256', buffer);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
  return hashHex;
}
```

### 阶段 3：文件删除优化

```go
// DeleteFile 删除时，减少引用计数
func (s *StorageService) DeleteFile(ctx context.Context, key string) error {
    // 1. 查找文件引用
    var ref FileReference
    err := s.db.Where("logical_key = ?", key).First(&ref).Error
    if err != nil {
        return fmt.Errorf("文件引用不存在")
    }
    
    // 2. 删除文件引用
    s.db.Delete(&ref)
    
    // 3. 减少引用计数
    var fileMeta FileMetadata
    s.db.First(&fileMeta, ref.FileID)
    fileMeta.RefCount--
    
    // 4. 如果引用计数为 0，删除物理文件
    if fileMeta.RefCount <= 0 {
        // 从 MinIO 删除
        s.client.RemoveObject(ctx, bucket, fileMeta.StorageKey, minio.RemoveObjectOptions{})
        // 从数据库删除
        s.db.Delete(&fileMeta)
    } else {
        // 更新引用计数
        s.db.Save(&fileMeta)
    }
    
    return nil
}
```

---

## 📊 架构对比

### 当前架构（无去重）

```
用户 A 上传 invoice.pdf (100MB)
  ↓
MinIO: luobei/app1/function1/2025/11/03/uuid-a.pdf (100MB)

用户 B 上传相同的 invoice.pdf (100MB)
  ↓
MinIO: luobei/app2/function2/2025/11/03/uuid-b.pdf (100MB)

总存储：200MB
```

### 未来架构（启用去重）

```
用户 A 上传 invoice.pdf (100MB)
  ↓
计算 hash: abc123...
  ↓
MinIO: shared/abc123.pdf (100MB)
DB: file_metadata { hash: abc123, storage_key: shared/abc123.pdf, ref_count: 1 }
DB: file_references { logical_key: luobei/app1/function1/.../uuid-a.pdf, file_id: 1 }

用户 B 上传相同的 invoice.pdf (100MB)
  ↓
计算 hash: abc123...
  ↓
检查数据库：hash 已存在！秒传！
DB: file_metadata { hash: abc123, ref_count: 2 }  // 引用计数 +1
DB: file_references { logical_key: luobei/app2/function2/.../uuid-b.pdf, file_id: 1 }

总存储：100MB（节省 50%）
```

---

## 💰 成本节省估算

### 假设场景

- 10 个租户，每个租户 10 个应用
- 每个应用 100 个函数
- 每个函数平均上传 50 个文件
- 平均文件大小：5MB
- 重复率：30%

### 成本对比

| 项目 | 无去重 | 启用去重 | 节省 |
|------|--------|----------|------|
| **总文件数** | 500,000 | 500,000 | - |
| **物理存储** | 2,500 GB | 1,750 GB | **750 GB (30%)** |
| **存储成本/月** | $50 | $35 | **$15** |
| **流量成本/月** | $100 | $70 | **$30** |
| **总节省/月** | - | - | **$45** |
| **年度节省** | - | - | **$540** |

### 大文件场景（更明显）

如果是视频文件（平均 500MB）：
- 无去重：250,000 GB = 244 TB
- 启用去重：175,000 GB = 171 TB
- 节省：**73 TB**
- 年度节省：**$8,760**

---

## 🎯 启用时机

### 建议启用条件

1. **文件数量 > 10,000**
2. **重复上传比例 > 20%**
3. **大文件场景**（视频、压缩包等）
4. **存储成本 > $100/月**

### 启用步骤

1. 配置开关：`deduplication.enabled: true`
2. 前端集成 hash 计算
3. 部署新版本
4. 监控去重效果

---

## ✅ 总结

### 已完成（不堵路）

✅ **数据库表已创建**：`file_metadata`, `file_references`  
✅ **DTO 字段已预留**：`hash` 字段  
✅ **配置开关已预留**：`deduplication.enabled`  
✅ **HTTP 缓存已启用**：`Cache-Control: max-age=31536000`  

### 未来实现（按需）

⏳ `CheckFile` API（秒传检查）  
⏳ 前端 hash 计算  
⏳ `UploadComplete` 通知  
⏳ 引用计数管理  

### 架构保证

当前的 `router/date/uuid.ext` 文件组织方式，完全兼容未来的去重架构：

```
逻辑 Key（用户看到的）：
  luobei/app1/function1/2025/11/03/uuid-a.pdf

物理 Key（实际存储）：
  shared/abc123...def456.pdf

映射关系存储在 file_references 表中。
```

用户无感知，后端自动优化，完美！🎉

