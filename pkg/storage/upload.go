package storage

import (
	"context"
	"io"

	"github.com/kageos/kageos/dto"
)

// UploadResult 上传结果
type UploadResult struct {
	Key               string // 文件 Key
	ETag              string // 存储服务返回的 ETag（可能为空，取决于存储引擎）
	Hash              string // 文件 SHA256 hash（上传前计算）
	Size              int64  // 文件大小
	ContentType       string // 文件类型
	DownloadURL       string // ✨ 外部访问的下载地址（前端使用）
	ServerDownloadURL string // ✨ 内部访问的下载地址（服务端使用）
}

// Uploader 服务端文件上传接口。
type Uploader interface {
	// Upload 上传文件
	// ctx: 上下文
	// creds: 上传凭证（从存储服务获取）
	// fileReader: 文件读取器
	// fileSize: 文件大小（字节）
	// hash: 文件SHA256 hash（上传前已计算，用于秒传）
	// 返回: 上传结果，包含Key、ETag、Hash等信息
	Upload(ctx context.Context, creds *dto.GetUploadTokenResp, fileReader io.Reader, fileSize int64, hash string) (*UploadResult, error)
}

// NewUploader 根据 storage 类型创建对应的上传器。
// 当前仅支持 MinIO。
func NewUploader(storage string) (Uploader, error) {
	switch storage {
	case "", "minio":
		return NewMinIOUploader(), nil
	default:
		return nil, ErrUnsupportedStorage
	}
}
