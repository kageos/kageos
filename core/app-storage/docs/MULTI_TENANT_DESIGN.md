# 多租户存储架构设计

## 🎯 设计目标

1. **租户隔离**：每个租户的文件完全隔离
2. **精确统计**：可以统计每个函数/应用/租户的存储占用
3. **便于管理**：支持按函数批量删除文件
4. **审计追踪**：知道每个文件属于哪个函数
5. **成本分摊**：可以按租户/应用/函数计费

## 📁 文件存储结构

### Key 格式

```
{tenant}/{app}/{function_path}/{date}/{uuid}.{ext}

示例：
luobei/test88888/tools/cashier_desk/2025/01/03/550e8400-e29b-41d4-a716-446655440000.jpg
│      │         │                  │          │                                      │
│      │         │                  │          │                                      └─ 文件扩展名
│      │         │                  │          └─ UUID（防止文件名冲突）
│      │         │                  └─ 日期分组（年/月/日）
│      │         └─ 函数路径
│      └─ 应用名称
└─ 租户名称
```

### 层级结构

```
ai-agent-os (Bucket)
├── luobei/                              # 租户：luobei
│   ├── test88888/                       # 应用：test88888
│   │   ├── tools/cashier_desk/          # 函数：收银台
│   │   │   ├── 2025/01/03/
│   │   │   │   ├── xxx-xxx-xxx.jpg
│   │   │   │   └── yyy-yyy-yyy.pdf
│   │   │   └── 2025/01/04/
│   │   │       └── zzz-zzz-zzz.png
│   │   └── crm/ticket/                  # 函数：工单系统
│   │       └── 2025/01/03/
│   │           └── aaa-aaa-aaa.xlsx
│   └── another_app/                     # 应用：another_app
│       └── ...
└── another_tenant/                      # 租户：another_tenant
    └── ...
```

## 🔍 查询与统计

### 1. 按租户查询

```go
// 列举租户的所有文件
prefix := "luobei/"
```

### 2. 按应用查询

```go
// 列举应用的所有文件
prefix := "luobei/test88888/"
```

### 3. 按函数查询

```go
// 列举函数的所有文件
prefix := "luobei/test88888/tools/cashier_desk/"
```

### 4. 存储统计

MinIO 的 `ListObjects` API 支持按前缀过滤，我们可以：

- 统计每个函数的文件数量
- 统计每个函数的总大小
- 聚合计算每个应用/租户的存储占用

示例代码：

```go
func GetStorageStats(ctx context.Context, router string) (fileCount int, totalSize int64, err error) {
    bucket := "ai-agent-os"
    prefix := router + "/"
    
    objectCh := client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
        Prefix:    prefix,
        Recursive: true,
    })
    
    for object := range objectCh {
        fileCount++
        totalSize += object.Size
    }
    
    return fileCount, totalSize, nil
}
```

## 🗑️ 批量删除

### 1. 删除函数的所有文件

```bash
POST /api/v1/storage/batch_delete
{
  "router": "luobei/test88888/tools/cashier_desk"
}
```

### 2. 实现逻辑

```go
func DeleteFilesByRouter(ctx context.Context, router string) (int, error) {
    prefix := router + "/"
    
    // 列举所有文件
    objectCh := client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
        Prefix:    prefix,
        Recursive: true,
    })
    
    // 逐个删除
    deletedCount := 0
    for object := range objectCh {
        err := client.RemoveObject(ctx, bucket, object.Key, minio.RemoveObjectOptions{})
        if err == nil {
            deletedCount++
        }
    }
    
    return deletedCount, nil
}
```

## 🔐 权限控制（未来扩展）

基于 router 路径，可以实现细粒度的权限控制：

```go
// 检查用户是否有权限访问某个文件
func CheckPermission(user *User, fileKey string) bool {
    // 从 fileKey 中提取 tenant/app/function
    parts := strings.Split(fileKey, "/")
    tenant := parts[0]
    app := parts[1]
    function := strings.Join(parts[2:len(parts)-4], "/")
    
    // 检查用户是否属于该租户
    if user.Tenant != tenant {
        return false
    }
    
    // 检查用户是否有该应用的权限
    if !user.HasAppPermission(app) {
        return false
    }
    
    return true
}
```

## 💰 成本分摊

可以定期统计每个租户/应用/函数的存储占用，用于计费：

```sql
-- 存储占用记录表（示例）
CREATE TABLE storage_usage (
    id BIGINT PRIMARY KEY,
    tenant VARCHAR(255),
    app VARCHAR(255),
    function_path VARCHAR(500),
    file_count INT,
    total_size BIGINT,
    recorded_at TIMESTAMP
);
```

定时任务：

```go
func RecordStorageUsage() {
    // 遍历所有函数
    for _, function := range getAllFunctions() {
        router := fmt.Sprintf("%s/%s/%s", function.Tenant, function.App, function.Path)
        fileCount, totalSize, _ := storageService.GetStorageStats(ctx, router)
        
        // 记录到数据库
        db.Insert(&StorageUsage{
            Tenant:       function.Tenant,
            App:          function.App,
            FunctionPath: function.Path,
            FileCount:    fileCount,
            TotalSize:    totalSize,
            RecordedAt:   time.Now(),
        })
    }
}
```

## 📊 监控指标

可以监控以下指标：

1. **租户级别**：每个租户的总存储占用
2. **应用级别**：每个应用的总存储占用
3. **函数级别**：每个函数的总存储占用
4. **增长趋势**：存储占用的增长速率
5. **热点函数**：哪些函数上传文件最多

## 🚀 性能优化

### 1. 缓存统计结果

频繁调用 `ListObjects` 会影响性能，可以：

- 将统计结果缓存到 Redis
- 定时更新缓存（例如每小时）
- 提供实时查询和历史查询两种模式

### 2. 异步统计

```go
// 上传文件后，异步更新统计
func OnFileUploaded(router string, fileSize int64) {
    go func() {
        // 更新 Redis 缓存
        redis.IncrBy("storage:count:"+router, 1)
        redis.IncrBy("storage:size:"+router, fileSize)
    }()
}
```

### 3. 分页列举

对于文件数量特别多的函数，使用分页：

```go
func ListFilesWithPagination(ctx context.Context, router string, marker string, limit int) (files []string, nextMarker string, err error) {
    objectCh := client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
        Prefix:    router + "/",
        Recursive: true,
        MaxKeys:   limit,
        Marker:    marker,
    })
    
    // ...
}
```

## 📝 总结

通过 `{router}/{date}/{uuid}.{ext}` 的文件组织方式，我们实现了：

✅ **多租户隔离**：每个租户的文件独立存储  
✅ **精确统计**：可以统计任意粒度的存储占用  
✅ **便于管理**：支持批量删除和查询  
✅ **审计追踪**：每个文件都有明确的归属  
✅ **扩展性强**：便于后续实现权限控制和成本分摊  

这种设计是企业级 SaaS 系统的标准做法，既满足了多租户隔离的安全需求，又便于运营和管理。

