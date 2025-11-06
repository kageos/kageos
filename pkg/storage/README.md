# 存储上传接口

统一的上传接口，支持多种存储引擎（MinIO、腾讯云COS、阿里云OSS、AWS S3、七牛云等）。

## 设计理念

1. **统一接口**：所有存储引擎实现相同的 `Uploader` 接口
2. **工厂模式**：根据 `storage` 字段动态创建对应的上传器
3. **可扩展性**：新增存储引擎只需实现 `Uploader` 接口并注册到工厂

## 关于 Hash 的处理

### MinIO 的 ETag vs SHA256 Hash

**重要区别**：
- **ETag**：MinIO 服务器返回的标识符
  - 小文件（<5MB）：ETag = MD5（不是 SHA256）
  - 大文件（multipart 上传）：ETag = `hash-分片数`（用于验证分片完整性）
  - **不是完整文件的 SHA256 hash**

- **SHA256 Hash**：需要上传方在上传前计算
  - 用于秒传和去重功能
  - 必须在上传前计算（因为需要检查文件是否已存在）

### Hash 计算性能

| 文件大小 | 计算时间（估算） | 说明 |
|---------|----------------|------|
| < 1MB   | < 10ms        | 几乎无感知 |
| 1-10MB  | 10-100ms      | 可以接受 |
| 10-100MB | 100ms-1s      | 稍慢，但可接受 |
| 100MB-1GB | 1-5s          | 较慢，建议异步处理 |
| > 1GB   | 5-20s         | 很慢，建议分块计算或异步 |

**建议**：
- 小文件（<10MB）：同步计算，上传前计算
- 大文件（>10MB）：可以考虑异步计算或在后台计算

### 实现方式

```go
// 1. 上传前计算 SHA256 hash（用于秒传）
hash := calculateSHA256(fileReader)

// 2. 上传文件（hash 作为参数传入）
result, err := uploader.Upload(ctx, creds, fileReader, fileSize, hash)

// 3. 上传后获取 ETag（从响应头获取，用于验证）
etag := result.ETag  // MinIO 返回的 ETag
hash := result.Hash  // 上传前计算的 SHA256
```

## 使用示例

```go
import "github.com/ai-agent-os/ai-agent-os/pkg/storage"

// 1. 从存储服务获取上传凭证
creds := &dto.GetUploadTokenResp{
    Storage: "minio",
    Method:  dto.UploadMethodPresignedURL,
    URL:     "http://localhost:9000/...",
    // ...
}

// 2. 创建对应的上传器
factory := storage.GetDefaultFactory()
uploader, err := factory.NewUploader(creds.Storage)
if err != nil {
    return err
}

// 3. 计算文件 hash（上传前）
hash := calculateSHA256(fileReader)

// 4. 上传文件
result, err := uploader.Upload(ctx, creds, fileReader, fileSize, hash)
if err != nil {
    return err
}

// 5. 使用上传结果
fmt.Printf("Key: %s\n", result.Key)
fmt.Printf("ETag: %s\n", result.ETag)      // MinIO 返回的 ETag
fmt.Printf("Hash: %s\n", result.Hash)      // 上传前计算的 SHA256
fmt.Printf("DownloadURL: %s\n", result.DownloadURL)
```

## 支持的存储引擎

- ✅ **MinIO** (`minio`) - 使用 presigned_url 方式
- ⏳ **七牛云** (`qiniu`) - 使用 form_upload 方式（待实现）
- ⏳ **腾讯云COS** (`tencentcos`) - 使用 presigned_url 方式（待实现）
- ⏳ **阿里云OSS** (`aliyunoss`) - 使用 presigned_url 方式（待实现）
- ⏳ **AWS S3** (`awss3`) - 使用 presigned_url 方式（待实现）

## 扩展新的存储引擎

1. 实现 `Uploader` 接口：

```go
type MyStorageUploader struct{}

func (u *MyStorageUploader) Upload(ctx context.Context, creds *dto.GetUploadTokenResp, fileReader io.Reader, fileSize int64, hash string) (*UploadResult, error) {
    // 实现上传逻辑
    // ...
}
```

2. 在工厂中注册：

```go
func (f *UploaderFactory) NewUploader(storage string) (Uploader, error) {
    switch storage {
    case "mystorage":
        return NewMyStorageUploader(), nil
    // ...
    }
}
```

## 接口定义

```go
type Uploader interface {
    Upload(ctx context.Context, creds *dto.GetUploadTokenResp, fileReader io.Reader, fileSize int64, hash string) (*UploadResult, error)
}

type UploadResult struct {
    Key         string // 文件 Key
    ETag        string // 存储服务返回的 ETag
    Hash        string // 文件 SHA256 hash（上传前计算）
    Size        int64  // 文件大小
    ContentType string // 文件类型
    DownloadURL string // 下载地址
}
```

