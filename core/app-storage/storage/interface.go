package storage

import (
	"context"
	"io"
	"time"
)

// UploadMethod 上传方式
type UploadMethod string

const (
	UploadMethodPresignedURL UploadMethod = "presigned_url" // 预签名 URL（标准 S3 协议）
	UploadMethodFormUpload   UploadMethod = "form_upload"   // 表单上传（七牛云等）
	UploadMethodSDKUpload    UploadMethod = "sdk_upload"    // SDK 上传（特殊云存储）
)

// UploadCredentials 上传凭证（统一结构）
type UploadCredentials struct {
	Method UploadMethod // 上传方式
	
	// 预签名 URL 上传（MinIO、COS、OSS、S3）
	URL       string            // ✨ 外部访问的预签名 URL（前端使用）
	ServerURL string            // ✨ 内部访问的预签名 URL（服务端/SDK使用）
	Headers   map[string]string // 请求头
	
	// 上传域名信息 ✨ 新增
	UploadHost string // 上传目标域名（例如：localhost:9000 或 cdn.example.com）
	UploadDomain string // 上传完整域名（例如：http://localhost:9000 或 https://cdn.example.com）
	
	// 表单上传（七牛云、又拍云等）
	FormData map[string]string // 表单字段
	PostURL  string            // POST 地址
	
	// SDK 上传（特殊云存储）
	SDKConfig map[string]interface{} // SDK 配置
}

// Storage 存储接口（抽象层）
// 所有存储实现（MinIO、腾讯云 COS、阿里云 OSS、AWS S3 等）都必须实现此接口
type Storage interface {
	// GetUploadMethod 获取上传方式 ✨ 新增
	GetUploadMethod() UploadMethod

	// GetCDNDomain 获取 CDN 域名 ✨ 新增
	GetCDNDomain() string

	// GetUploadEndpoint 获取上传用的 endpoint
	// uploadSource: 上传来源（browser 或 server）
	// 返回: 上传用的 endpoint（如果为空则使用默认 endpoint）
	// 统一逻辑：如果配置了 server_endpoint 且 upload_source 是 server，返回 server_endpoint；否则返回空字符串（使用默认）
	GetUploadEndpoint(uploadSource string) string

	// GenerateUploadCredentials 生成上传凭证（统一接口）✨ 新增
	GenerateUploadCredentials(ctx context.Context, bucket, key, contentType string, expire time.Duration) (*UploadCredentials, error)

	// GenerateUploadURL 生成上传预签名 URL（兼容旧接口，内部调用 GenerateUploadCredentials）
	GenerateUploadURL(ctx context.Context, bucket, key, contentType string, expire time.Duration) (url string, err error)

	// GenerateDownloadURL 生成下载预签名 URL
	GenerateDownloadURL(ctx context.Context, bucket, key string, expire time.Duration, cacheControl map[string]string) (url string, err error)

	// DeleteObject 删除对象
	DeleteObject(ctx context.Context, bucket, key string) error

	// GetObjectInfo 获取对象信息
	GetObjectInfo(ctx context.Context, bucket, key string) (*ObjectInfo, error)

	// ListObjects 列举对象
	ListObjects(ctx context.Context, bucket, prefix string, recursive bool) ([]ObjectInfo, error)

	// EnsureBucket 确保 Bucket 存在（如果不存在则创建）
	EnsureBucket(ctx context.Context, bucket, region string) error

	// UploadObject 直接上传对象（用于代理上传场景）
	UploadObject(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) error

	// DownloadObject 直接下载对象（用于代理下载场景）
	DownloadObject(ctx context.Context, bucket, key string) (io.ReadCloser, error)
}

// ObjectInfo 对象信息
type ObjectInfo struct {
	Key          string    // 对象 Key
	Size         int64     // 文件大小（字节）
	ContentType  string    // MIME 类型
	ETag         string    // ETag
	LastModified time.Time // 最后修改时间
}

// Config 存储配置接口
type Config interface {
	GetEndpoint() string
	GetAccessKey() string
	GetSecretKey() string
	GetRegion() string
	GetUseSSL() bool
	GetDefaultBucket() string
	GetCDNDomain() string  // ✨ 新增：获取 CDN 域名
}

