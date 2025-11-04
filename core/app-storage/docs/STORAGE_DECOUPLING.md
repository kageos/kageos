# 存储引擎解耦设计

## 🎯 设计目标

**与底层存储引擎解耦，支持多种存储后端（MinIO、腾讯云 COS、阿里云 OSS、AWS S3 等），无需修改业务代码。**

---

## 🏗️ 架构设计

### 核心思想：依赖倒置原则（DIP）

```
高层模块（Service）不应该依赖低层模块（MinIO），
两者都应该依赖抽象（Storage Interface）
```

### 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                       Business Logic                             │
│                     (StorageService)                             │
│                           ↓                                      │
│              依赖抽象接口，不依赖具体实现                           │
└─────────────────────┬───────────────────────────────────────────┘
                      ↓
┌─────────────────────────────────────────────────────────────────┐
│              Storage Interface (storage.Storage)                 │
│  ┌───────────────────────────────────────────────────────┐      │
│  │  GenerateUploadURL()                                   │      │
│  │  GenerateDownloadURL()                                 │      │
│  │  DeleteObject()                                        │      │
│  │  GetObjectInfo()                                       │      │
│  │  ListObjects()                                         │      │
│  │  EnsureBucket()                                        │      │
│  └───────────────────────────────────────────────────────┘      │
└─────────────────────┬───────────────────────────────────────────┘
                      ↓
          ┌───────────┴─────────────┐
          │   Storage Factory        │  (工厂模式)
          │   (根据配置创建实例)      │
          └───────────┬──────────────┘
                      ↓
    ┌────────────────┼────────────────┬────────────────┐
    ↓                ↓                ↓                ↓
┌────────┐    ┌────────────┐   ┌──────────┐   ┌──────────┐
│ MinIO  │    │ TencentCOS │   │ AliyunOSS│   │  AWS S3  │
│ Impl   │    │   Impl     │   │   Impl   │   │   Impl   │
└────────┘    └────────────┘   └──────────┘   └──────────┘
```

---

## 📊 核心组件

### 1. Storage Interface (storage/interface.go)

**定义统一的存储接口**

```go
type Storage interface {
    // 生成上传预签名 URL
    GenerateUploadURL(ctx context.Context, bucket, key, contentType string, expire time.Duration) (url string, err error)
    
    // 生成下载预签名 URL
    GenerateDownloadURL(ctx context.Context, bucket, key string, expire time.Duration, cacheControl map[string]string) (url string, err error)
    
    // 删除对象
    DeleteObject(ctx context.Context, bucket, key string) error
    
    // 获取对象信息
    GetObjectInfo(ctx context.Context, bucket, key string) (*ObjectInfo, error)
    
    // 列举对象
    ListObjects(ctx context.Context, bucket, prefix string, recursive bool) ([]ObjectInfo, error)
    
    // 确保 Bucket 存在
    EnsureBucket(ctx context.Context, bucket, region string) error
    
    // 直接上传对象（用于代理上传）
    UploadObject(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) error
    
    // 直接下载对象（用于代理下载）
    DownloadObject(ctx context.Context, bucket, key string) (io.ReadCloser, error)
}
```

**为什么需要接口？**
- ✅ **解耦**：业务逻辑不依赖具体实现
- ✅ **扩展**：新增存储后端无需修改业务代码
- ✅ **测试**：可以 Mock 接口进行单元测试
- ✅ **灵活**：可以在运行时切换存储后端

---

### 2. MinIOStorage Implementation (storage/minio.go)

**MinIO 的具体实现**

```go
type MinIOStorage struct {
    client *minio.Client
}

func NewMinIOStorage(cfg Config) (*MinIOStorage, error) {
    client, err := minio.New(cfg.GetEndpoint(), &minio.Options{
        Creds:  credentials.NewStaticV4(cfg.GetAccessKey(), cfg.GetSecretKey(), ""),
        Secure: cfg.GetUseSSL(),
        Region: cfg.GetRegion(),
    })
    return &MinIOStorage{client: client}, err
}

// 实现所有 Storage 接口方法
func (s *MinIOStorage) GenerateUploadURL(...) { ... }
func (s *MinIOStorage) GenerateDownloadURL(...) { ... }
func (s *MinIOStorage) DeleteObject(...) { ... }
// ...
```

---

### 3. Storage Factory (storage/factory.go)

**根据配置创建存储实例**

```go
type StorageType string

const (
    StorageTypeMinIO      StorageType = "minio"
    StorageTypeTencentCOS StorageType = "tencentcos"
    StorageTypeAliyunOSS  StorageType = "aliyunoss"
    StorageTypeAWSS3      StorageType = "awss3"
)

func (f *Factory) CreateStorage(storageType string, cfg Config) (Storage, error) {
    switch StorageType(storageType) {
    case StorageTypeMinIO:
        return NewMinIOStorage(cfg)
    case StorageTypeTencentCOS:
        return NewTencentCOSStorage(cfg)  // TODO: 实现
    case StorageTypeAliyunOSS:
        return NewAliyunOSSStorage(cfg)   // TODO: 实现
    case StorageTypeAWSS3:
        return NewAWSS3Storage(cfg)       // TODO: 实现
    default:
        return nil, fmt.Errorf("不支持的存储类型: %s", storageType)
    }
}
```

---

### 4. Config Adapter (pkg/config/storage_adapter.go)

**适配不同存储的配置**

```go
type StorageConfigAdapter struct {
    cfg *AppStorageConfig
}

func (a *StorageConfigAdapter) GetEndpoint() string {
    switch a.cfg.Storage.Type {
    case "minio":
        return a.cfg.Storage.MinIO.Endpoint
    case "tencentcos":
        return a.cfg.Storage.TencentCOS.Endpoint
    // ...
    }
}
```

**为什么需要适配器？**
- ✅ 不同存储的配置字段名不同（AccessKey vs SecretID）
- ✅ 统一的接口访问配置
- ✅ 符合适配器模式

---

### 5. Business Service (service/storage_service.go)

**业务层只依赖接口**

```go
type StorageService struct {
    storage  storage.Storage  // ✅ 依赖抽象接口
    cfg      *config.AppStorageConfig
    fileRepo *repository.FileRepository
}

func (s *StorageService) GenerateUploadToken(...) {
    // 业务逻辑
    key = s.generateFileKey(router, fileName)
    
    // 调用存储接口（不关心具体实现）
    url, err := s.storage.GenerateUploadURL(ctx, bucket, key, contentType, expiry)
    
    return url, key, expire, err
}
```

---

## 🎨 配置示例

### app-storage.yaml

```yaml
# 存储配置
storage:
  # 存储类型：minio | tencentcos | aliyunoss | awss3
  type: "minio"
  
  # MinIO 配置
  minio:
    endpoint: "localhost:9000"
    access_key: "minioadmin"
    secret_key: "minioadmin123"
    use_ssl: false
    region: "us-east-1"
    default_bucket: "ai-agent-os"
  
  # 腾讯云 COS 配置
  tencentcos:
    endpoint: "cos.ap-guangzhou.myqcloud.com"
    secret_id: "your-secret-id"
    secret_key: "your-secret-key"
    region: "ap-guangzhou"
    default_bucket: "your-bucket"
  
  # 阿里云 OSS 配置
  aliyunoss:
    endpoint: "oss-cn-hangzhou.aliyuncs.com"
    access_key_id: "your-access-key-id"
    access_key_secret: "your-access-key-secret"
    region: "oss-cn-hangzhou"
    default_bucket: "your-bucket"
```

---

## 🔧 切换存储后端

### 方式 1：修改配置文件

```yaml
# 从 MinIO 切换到腾讯云 COS
storage:
  type: "tencentcos"  # 只需修改这一行
  
  tencentcos:
    endpoint: "cos.ap-guangzhou.myqcloud.com"
    secret_id: "..."
    secret_key: "..."
    region: "ap-guangzhou"
    default_bucket: "my-bucket"
```

**无需修改任何代码！** 重启服务即可。

### 方式 2：环境变量（生产环境）

```bash
export STORAGE_TYPE=tencentcos
export TENCENTCOS_SECRET_ID=xxx
export TENCENTCOS_SECRET_KEY=yyy
```

---

## 🚀 新增存储后端

### 示例：添加腾讯云 COS 支持

#### Step 1: 实现 Storage 接口

```go
// core/app-storage/storage/tencentcos.go
package storage

import (
    "github.com/tencentyun/cos-go-sdk-v5"
)

type TencentCOSStorage struct {
    client *cos.Client
}

func NewTencentCOSStorage(cfg Config) (*TencentCOSStorage, error) {
    u, _ := url.Parse(fmt.Sprintf("https://%s", cfg.GetEndpoint()))
    
    client := cos.NewClient(&cos.BaseURL{BucketURL: u}, &http.Client{
        Transport: &cos.AuthorizationTransport{
            SecretID:  cfg.GetAccessKey(),
            SecretKey: cfg.GetSecretKey(),
        },
    })
    
    return &TencentCOSStorage{client: client}, nil
}

// 实现 Storage 接口
func (s *TencentCOSStorage) GenerateUploadURL(ctx context.Context, bucket, key, contentType string, expire time.Duration) (string, error) {
    presignedURL, err := s.client.Object.GetPresignedURL(
        ctx,
        http.MethodPut,
        key,
        s.cfg.GetAccessKey(),
        s.cfg.GetSecretKey(),
        expire,
        nil,
    )
    return presignedURL.String(), err
}

// ... 实现其他方法
```

#### Step 2: 注册到工厂

```go
// storage/factory.go
func (f *Factory) CreateStorage(storageType string, cfg Config) (Storage, error) {
    switch StorageType(storageType) {
    case StorageTypeMinIO:
        return NewMinIOStorage(cfg)
    case StorageTypeTencentCOS:
        return NewTencentCOSStorage(cfg)  // ✅ 添加这一行
    // ...
    }
}
```

#### Step 3: 完成！

**无需修改任何业务代码！** 只需修改配置即可使用腾讯云 COS。

---

## ✅ 优势对比

| 特性 | 耦合实现（旧） | 解耦实现（新） |
|------|---------------|---------------|
| **扩展性** | ❌ 需修改业务代码 | ✅ 只需添加实现类 |
| **维护性** | ❌ 代码分散，难以维护 | ✅ 职责清晰，易于维护 |
| **测试性** | ❌ 需要真实 MinIO | ✅ 可以 Mock 接口 |
| **灵活性** | ❌ 固定 MinIO | ✅ 运行时切换 |
| **依赖** | ❌ 强依赖 MinIO SDK | ✅ 依赖抽象接口 |

---

## 🎯 SOLID 原则体现

### 1. **单一职责原则（SRP）**
- ✅ `StorageService`：只负责业务逻辑
- ✅ `MinIOStorage`：只负责 MinIO 操作
- ✅ `Factory`：只负责创建实例

### 2. **开闭原则（OCP）**
- ✅ 对扩展开放：可以新增存储实现
- ✅ 对修改关闭：无需修改业务代码

### 3. **里氏替换原则（LSP）**
- ✅ 所有 Storage 实现都可以互换

### 4. **接口隔离原则（ISP）**
- ✅ Storage 接口只包含必要方法

### 5. **依赖倒置原则（DIP）** ⭐
- ✅ 高层模块（Service）依赖抽象（Interface）
- ✅ 低层模块（MinIOStorage）依赖抽象（Interface）

---

## 📊 未来扩展

### 已预留支持

1. ✅ **腾讯云 COS**
2. ✅ **阿里云 OSS**
3. ✅ **AWS S3**
4. ✅ **本地文件系统**（开发环境）

### 扩展方向

1. **多存储混合**：不同租户使用不同存储
2. **存储迁移**：平滑迁移存储后端
3. **存储代理**：智能选择最优存储
4. **存储备份**：多存储冗余备份

---

## 🎉 总结

通过引入存储抽象层，我们实现了：

1. ✅ **完全解耦**：业务代码与存储引擎解耦
2. ✅ **易于扩展**：新增存储只需添加实现类
3. ✅ **配置驱动**：切换存储无需修改代码
4. ✅ **符合SOLID**：遵循最佳实践
5. ✅ **未来可期**：为多存储、存储迁移等高级功能打下基础

**这是企业级系统的标准架构！** 🎉

