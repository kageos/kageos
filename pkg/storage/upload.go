package storage

import (
	"context"
	"io"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

// UploadResult 上传结果
type UploadResult struct {
	Key              string // 文件 Key
	ETag             string // 存储服务返回的 ETag（可能为空，取决于存储引擎）
	Hash             string // 文件 SHA256 hash（上传前计算）
	Size             int64  // 文件大小
	ContentType      string // 文件类型
	DownloadURL      string // ✨ 外部访问的下载地址（前端使用）
	ServerDownloadURL string // ✨ 内部访问的下载地址（服务端使用）
}

// Uploader 上传接口（根据不同的存储引擎实现）
// 所有存储引擎的上传器都必须实现此接口
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

// UploaderFactory 上传器工厂（根据 storage 字段创建对应的上传器）
type UploaderFactory struct{}

// NewUploader 根据 storage 类型创建对应的上传器
// storage: 存储引擎类型（minio/qiniu/tencentcos/aliyunoss/awss3等）
func (f *UploaderFactory) NewUploader(storage string) (Uploader, error) {
	switch storage {
	case "minio":
		return NewMinIOUploader(), nil
	case "qiniu":
		// TODO: 实现七牛云上传器
		return nil, ErrNotImplemented
	case "tencentcos":
		// TODO: 实现腾讯云COS上传器
		return nil, ErrNotImplemented
	case "aliyunoss":
		// TODO: 实现阿里云OSS上传器
		return nil, ErrNotImplemented
	case "awss3":
		// TODO: 实现AWS S3上传器
		return nil, ErrNotImplemented
	default:
		return nil, ErrUnsupportedStorage
	}
}

// GetDefaultFactory 获取默认的上传器工厂
func GetDefaultFactory() *UploaderFactory {
	return &UploaderFactory{}
}
