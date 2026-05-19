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
prefix := "luobei/test88888/plugins/cashier_desk/"
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

## 🔐 权限控制边界

当前实现使用 router 前缀组织对象 Key，并通过登录用户、桶和对象引用生成访问 URL。细粒度权限控制不属于 app-storage 的当前 MVP 能力，后续如需加入，应先在产品边界和权限模型中明确。

## 💰 统计边界

当前统计能力面向文件数量和总大小查询，不包含计费、配额或成本分摊实现。

## 📊 监控指标

可以监控以下指标：

1. **租户级别**：每个租户的总存储占用
2. **应用级别**：每个应用的总存储占用
3. **函数级别**：每个函数的总存储占用
4. **增长趋势**：存储占用的增长速率
5. **热点函数**：哪些函数上传文件最多

## 🚀 性能优化边界

当前实现直接基于 MinIO 对象列举和数据库元数据记录完成核心操作。缓存统计、异步统计和历史趋势不属于当前 MVP。

### 分页列举

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
✅ **边界清楚**：不混入未落地的权限、计费和成本分摊能力

这种设计满足当前个人和小团队 MVP 的文件隔离、查询和清理需求。
