# App Storage 架构设计

## 🏗️ 分层架构

遵循标准的 **三层架构**（Controller-Service-Repository），确保职责清晰、易于测试和维护。

```
┌──────────────────────────────────────────────────────────┐
│                    HTTP Request                          │
└────────────────────┬─────────────────────────────────────┘
                     ↓
┌──────────────────────────────────────────────────────────┐
│  API Layer (Controller / Handler)                        │
│  - 处理 HTTP 请求/响应                                     │
│  - 参数验证和绑定                                          │
│  - 调用 Service 层                                        │
│  - 统一错误处理和响应格式                                   │
│                                                           │
│  📁 api/v1/storage.go                                    │
│  type Storage struct {                                   │
│      storageService *service.StorageService  // ✅ 只依赖 Service │
│  }                                                       │
└────────────────────┬─────────────────────────────────────┘
                     ↓
┌──────────────────────────────────────────────────────────┐
│  Service Layer (Business Logic)                          │
│  - 业务逻辑处理                                            │
│  - 调用 Repository 层                                     │
│  - 调用外部服务（MinIO）                                   │
│  - 事务管理                                               │
│                                                           │
│  📁 service/storage_service.go                           │
│  type StorageService struct {                            │
│      client   *minio.Client          // MinIO 客户端      │
│      cfg      *config.AppStorageConfig                   │
│      fileRepo *repository.FileRepository  // ✅ 依赖 Repository │
│  }                                                       │
└────────────────────┬─────────────────────────────────────┘
                     ↓
┌──────────────────────────────────────────────────────────┐
│  Repository Layer (Data Access)                          │
│  - 数据库操作封装                                          │
│  - CRUD 操作                                             │
│  - 查询构建                                               │
│  - 数据持久化                                             │
│                                                           │
│  📁 repository/file_repository.go                        │
│  type FileRepository struct {                            │
│      db *gorm.DB                     // ✅ 只依赖 DB        │
│  }                                                       │
└────────────────────┬─────────────────────────────────────┘
                     ↓
┌──────────────────────────────────────────────────────────┐
│                    Database (MySQL)                      │
│  - file_uploads                                          │
│  - file_downloads                                        │
│  - file_metadata                                         │
│  - file_references                                       │
└──────────────────────────────────────────────────────────┘
```

---

## 📊 依赖关系

### ✅ 正确的依赖方向

```
API → Service → Repository → DB
                ↓
              MinIO
```

**职责分离**：
- **API 层**：只负责 HTTP 协议，不涉及业务逻辑和数据访问
- **Service 层**：只负责业务逻辑，不直接操作数据库
- **Repository 层**：只负责数据访问，不涉及业务逻辑

### ❌ 错误的依赖（已修复）

```
API → DB (直接操作)  ❌ 违反分层原则
    → Service
```

**问题**：
- API 层直接依赖 DB，职责混乱
- 难以测试（需要 mock DB）
- 难以替换数据库实现
- 违反单一职责原则

---

## 📁 目录结构

```
core/app-storage/
├── api/v1/                     # API 层
│   └── storage.go              # HTTP Handler
│       └── NewStorage(storageService)  ✅ 只依赖 Service
│
├── service/                    # Service 层
│   └── storage_service.go      # 业务逻辑
│       └── NewStorageService(client, cfg, fileRepo)  ✅ 依赖 Repository
│
├── repository/                 # Repository 层 ✨ 新增
│   └── file_repository.go      # 数据访问
│       └── NewFileRepository(db)  ✅ 只依赖 DB
│
├── model/                      # 数据模型
│   ├── file.go                 # 数据表定义
│   └── init.go                 # 表初始化
│
├── server/                     # 服务器启动
│   ├── server.go               # 初始化和依赖注入
│   └── router.go               # 路由注册
│
└── cmd/app/                    # 程序入口
    └── main.go
```

---

## 🔧 依赖注入流程

### 1. Server 初始化

```go
// server/server.go
func NewServer(cfg *config.AppStorageConfig) (*Server, error) {
    // 1. 初始化数据库
    s.initDatabase()
    
    // 2. 初始化 MinIO 客户端
    s.initMinIO()
    
    // 3. 初始化服务（依赖注入）
    s.initServices()
    
    // 4. 初始化路由
    s.initRouter()
}
```

### 2. Service 初始化

```go
// server/server.go - initServices()
func (s *Server) initServices(ctx context.Context) error {
    // 初始化 Repository 层
    var fileRepo *repository.FileRepository
    if s.db != nil {
        fileRepo = repository.NewFileRepository(s.db)  // ✅ 注入 DB
    }
    
    // 初始化 Service 层
    s.storageService = service.NewStorageService(
        s.minioClient,  // MinIO 客户端
        s.cfg,          // 配置
        fileRepo,       // ✅ 注入 Repository
    )
}
```

### 3. Handler 初始化

```go
// server/router.go - setupRoutes()
func (s *Server) setupRoutes() {
    storageHandler := v1.NewStorage(s.storageService)  // ✅ 只注入 Service
}
```

---

## 🎯 各层职责

### API Layer (api/v1/storage.go)

**职责**：
- ✅ 处理 HTTP 请求/响应
- ✅ 参数验证和绑定
- ✅ 从 JWT 中提取用户信息
- ✅ 调用 Service 层方法
- ✅ 统一响应格式

**示例**：
```go
func (s *Storage) GetUploadToken(c *gin.Context) {
    // 1. 参数验证
    var req dto.GetUploadTokenReq
    if err := c.ShouldBindJSON(&req); err != nil {
        response.FailWithMessage(c, "参数错误")
        return
    }
    
    // 2. 提取用户信息（JWT）
    userID, _ := c.Get("user_id")
    username, _ := c.Get("username")
    
    // 3. 调用 Service（业务逻辑）
    url, key, expire, err := s.storageService.GenerateUploadToken(...)
    
    // 4. 记录审计（通过 Service）
    uploadRecord := &model.FileUpload{...}
    s.storageService.RecordUpload(ctx, uploadRecord)
    
    // 5. 统一响应
    response.OkWithData(c, resp)
}
```

**不应该做的**：
- ❌ 直接操作数据库（`s.db.Create()`）
- ❌ 编写业务逻辑
- ❌ 直接调用 MinIO

---

### Service Layer (service/storage_service.go)

**职责**：
- ✅ 业务逻辑处理
- ✅ 调用 Repository 层
- ✅ 调用外部服务（MinIO）
- ✅ 事务管理
- ✅ 数据转换

**示例**：
```go
// 业务方法：生成上传凭证
func (s *StorageService) GenerateUploadToken(...) (url, key, expire, error) {
    // 1. 业务规则：校验文件大小
    if fileSize > s.cfg.MinIO.Upload.MaxSize {
        return "", "", time.Time{}, fmt.Errorf("文件过大")
    }
    
    // 2. 生成文件 Key（业务逻辑）
    key = s.generateFileKey(router, fileName)
    
    // 3. 调用外部服务（MinIO）
    presignedURL, err := s.client.PresignedPutObject(...)
    
    return presignedURL.String(), key, expire, nil
}

// 业务方法：记录上传
func (s *StorageService) RecordUpload(ctx context.Context, record *model.FileUpload) error {
    if s.fileRepo == nil {
        return nil // 审计未启用
    }
    // 调用 Repository 层
    return s.fileRepo.CreateUploadRecord(ctx, record)
}
```

**不应该做的**：
- ❌ 直接操作 `gorm.DB`
- ❌ 处理 HTTP 请求/响应
- ❌ 依赖 `gin.Context`

---

### Repository Layer (repository/file_repository.go)

**职责**：
- ✅ 封装数据库操作
- ✅ CRUD 方法
- ✅ 查询构建
- ✅ 数据持久化

**示例**：
```go
// 数据访问：创建上传记录
func (r *FileRepository) CreateUploadRecord(ctx context.Context, record *model.FileUpload) error {
    return r.db.WithContext(ctx).Create(record).Error
}

// 数据访问：更新上传状态
func (r *FileRepository) UpdateUploadStatus(ctx context.Context, fileKey string, status string) error {
    return r.db.WithContext(ctx).
        Model(&model.FileUpload{}).
        Where("file_key = ?", fileKey).
        Update("status", status).Error
}

// 数据访问：统计查询
func (r *FileRepository) GetStorageStatsByUser(ctx context.Context, userID int64) (fileCount, totalSize int64, err error) {
    err = r.db.WithContext(ctx).
        Model(&model.FileUpload{}).
        Where("user_id = ? AND status = ?", userID, "completed").
        Select("COUNT(*) as file_count, SUM(file_size) as total_size").
        Row().
        Scan(&fileCount, &totalSize)
    return
}
```

**不应该做的**：
- ❌ 业务逻辑判断
- ❌ 调用外部服务
- ❌ 依赖 Service 层

---

## ✅ 架构优势

### 1. **职责清晰**

每一层只做自己的事：
- API：HTTP 协议
- Service：业务逻辑
- Repository：数据访问

### 2. **易于测试**

```go
// 测试 Service 层
func TestStorageService_RecordUpload(t *testing.T) {
    // Mock Repository
    mockRepo := &MockFileRepository{}
    service := NewStorageService(minioClient, cfg, mockRepo)
    
    // 测试业务逻辑
    err := service.RecordUpload(ctx, record)
    assert.NoError(t, err)
}

// 测试 Repository 层
func TestFileRepository_CreateUploadRecord(t *testing.T) {
    // Mock DB
    db := setupTestDB()
    repo := NewFileRepository(db)
    
    // 测试数据访问
    err := repo.CreateUploadRecord(ctx, record)
    assert.NoError(t, err)
}
```

### 3. **易于替换**

- 可以轻松切换数据库（MySQL → PostgreSQL）
- 可以切换存储后端（MinIO → S3）
- Repository 接口化后可以 Mock

### 4. **符合 SOLID 原则**

- ✅ **单一职责原则**（SRP）：每层只有一个变化的理由
- ✅ **开闭原则**（OCP）：对扩展开放，对修改关闭
- ✅ **依赖倒置原则**（DIP）：依赖抽象（接口），不依赖具体实现

---

## 📊 对比

| 架构 | API 依赖 | Service 依赖 | Repository 依赖 | 评价 |
|------|----------|--------------|-----------------|------|
| **修复后** | Service | Repository | DB | ✅ 标准三层架构 |
| **修复前** | Service + DB | MinIO | - | ❌ 职责混乱 |

---

## 🎯 总结

通过引入 **Repository 层**，我们实现了：

1. ✅ **标准三层架构**：API → Service → Repository
2. ✅ **职责清晰**：每层只做自己的事
3. ✅ **易于测试**：每层可以独立测试
4. ✅ **易于维护**：修改数据访问不影响业务逻辑
5. ✅ **易于扩展**：可以轻松添加新的数据源

这是**企业级应用的标准架构**，感谢指正！🎉

