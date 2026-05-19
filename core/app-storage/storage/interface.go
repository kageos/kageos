package storage

import (
	"context"
	"io"
	"time"
)

// UploadMethod 上传方式
type UploadMethod string

const (
	UploadMethodPresignedURL UploadMethod = "presigned_url" // 预签名 URL（当前官方实际使用）
	UploadMethodFormUpload   UploadMethod = "form_upload"   // 预留：当前未使用
	UploadMethodSDKUpload    UploadMethod = "sdk_upload"    // 预留：当前未使用
)

// UploadCredentials 上传凭证（统一结构）
type UploadCredentials struct {
	Method UploadMethod // 上传方式

	// 预签名 URL 上传（当前官方仅 MinIO）
	UploadURL       string            // 外部访问的预签名上传地址（前端使用）
	ServerUploadURL string            // 内部访问的预签名上传地址（服务端/SDK使用）
	Headers         map[string]string // 请求头

	// 上传域名信息
	UploadHost   string // 上传目标域名（例如：localhost:9000 或 cdn.example.com）
	UploadDomain string // 上传完整域名（例如：http://localhost:9000 或 https://cdn.example.com）

	// 表单上传（预留）
	FormData map[string]string // 表单字段
	PostURL  string            // POST 地址

	// SDK 上传（预留）
	SDKConfig map[string]interface{} // SDK 配置
}

// Storage 存储接口（抽象层）
// 当前官方只落地了 MinIO；接口保留是为了后续扩展时不重写业务层。
type Storage interface {
	// GetUploadMethod 获取上传方式
	GetUploadMethod() UploadMethod

	// GetCDNDomain 获取 CDN 域名
	GetCDNDomain() string

	// GetUploadEndpoint 获取上传用的 endpoint
	// uploadSource: 上传来源（browser 或 server）
	// 返回: 上传用的 endpoint（如果为空则使用默认 endpoint）
	// 统一逻辑：如果配置了 server_endpoint 且 upload_source 是 server，返回 server_endpoint；否则返回空字符串（使用默认）
	GetUploadEndpoint(uploadSource string) string

	// GenerateUploadCredentials 生成上传凭证（统一接口）
	// uploadSource: 上传来源（browser 或 server），用于决定是否返回SDK配置
	GenerateUploadCredentials(ctx context.Context, bucket, key, contentType string, expire time.Duration, uploadSource string) (*UploadCredentials, error)

	// GenerateDownloadURLs 生成下载 URL（同时返回外部和内部访问的URL）
	GenerateDownloadURLs(ctx context.Context, bucket, key string, expire time.Duration, cacheControl map[string]string) (externalURL string, serverURL string, err error)

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
	GetCDNDomain() string
	GetServerEndpoint() string // 获取服务端 endpoint（仅用于 MinIO，容器内访问）
}
