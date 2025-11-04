package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOStorage MinIO 存储实现
type MinIOStorage struct {
	client    *minio.Client
	cdnDomain string  // CDN 域名（可选）
}

// NewMinIOStorage 创建 MinIO 存储
func NewMinIOStorage(cfg Config) (*MinIOStorage, error) {
	client, err := minio.New(cfg.GetEndpoint(), &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.GetAccessKey(), cfg.GetSecretKey(), ""),
		Secure: cfg.GetUseSSL(),
		Region: cfg.GetRegion(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	return &MinIOStorage{
		client:    client,
		cdnDomain: cfg.GetCDNDomain(),
	}, nil
}

// GetCDNDomain 获取 CDN 域名
func (s *MinIOStorage) GetCDNDomain() string {
	return s.cdnDomain
}

// GetUploadMethod 获取上传方式
func (s *MinIOStorage) GetUploadMethod() UploadMethod {
	return UploadMethodPresignedURL
}

// GenerateUploadCredentials 生成上传凭证
func (s *MinIOStorage) GenerateUploadCredentials(ctx context.Context, bucket, key, contentType string, expire time.Duration) (*UploadCredentials, error) {
	presignedURL, err := s.client.PresignedPutObject(ctx, bucket, key, expire)
	if err != nil {
		logger.Errorf(ctx, "[MinIOStorage] Failed to generate upload credentials: %v", err)
		return nil, fmt.Errorf("生成上传凭证失败: %w", err)
	}
	
	// 解析上传域名信息
	uploadURL := presignedURL.String()
	uploadHost, uploadDomain := s.extractDomainInfo(uploadURL)
	
	return &UploadCredentials{
		Method: UploadMethodPresignedURL,
		URL:    uploadURL,
		Headers: map[string]string{
			"Content-Type": contentType,
		},
		UploadHost:   uploadHost,   // ✨ 上传目标 host（用于 CORS、进度监听）
		UploadDomain: uploadDomain, // ✨ 上传完整域名（用于日志、调试）
	}, nil
}

// extractDomainInfo 从 URL 中提取域名信息
func (s *MinIOStorage) extractDomainInfo(uploadURL string) (host string, domain string) {
	parsedURL, err := url.Parse(uploadURL)
	if err != nil {
		logger.Errorf(context.Background(), "[MinIOStorage] Failed to parse upload URL: %v", err)
		return "", ""
	}
	
	// 提取 host（hostname:port）
	host = parsedURL.Host
	
	// 提取完整域名（scheme://host）
	scheme := parsedURL.Scheme
	if scheme == "" {
		scheme = "http" // 默认 http
	}
	domain = fmt.Sprintf("%s://%s", scheme, host)
	
	return host, domain
}

// GenerateUploadURL 生成上传预签名 URL（兼容旧接口）
func (s *MinIOStorage) GenerateUploadURL(ctx context.Context, bucket, key, contentType string, expire time.Duration) (string, error) {
	creds, err := s.GenerateUploadCredentials(ctx, bucket, key, contentType, expire)
	if err != nil {
		return "", err
	}
	return creds.URL, nil
}

// GenerateDownloadURL 生成下载预签名 URL
func (s *MinIOStorage) GenerateDownloadURL(ctx context.Context, bucket, key string, expire time.Duration, cacheControl map[string]string) (string, error) {
	// 构建查询参数
	reqParams := make(url.Values)
	for k, v := range cacheControl {
		reqParams.Set(k, v)
	}

	presignedURL, err := s.client.PresignedGetObject(ctx, bucket, key, expire, reqParams)
	if err != nil {
		logger.Errorf(ctx, "[MinIOStorage] Failed to generate download URL: %v", err)
		return "", fmt.Errorf("生成下载链接失败: %w", err)
	}
	return presignedURL.String(), nil
}

// DeleteObject 删除对象
func (s *MinIOStorage) DeleteObject(ctx context.Context, bucket, key string) error {
	err := s.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		logger.Errorf(ctx, "[MinIOStorage] Failed to delete object %s: %v", key, err)
		return fmt.Errorf("删除文件失败: %w", err)
	}
	return nil
}

// GetObjectInfo 获取对象信息
func (s *MinIOStorage) GetObjectInfo(ctx context.Context, bucket, key string) (*ObjectInfo, error) {
	stat, err := s.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		logger.Errorf(ctx, "[MinIOStorage] Failed to get object info for %s: %v", key, err)
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	return &ObjectInfo{
		Key:          stat.Key,
		Size:         stat.Size,
		ContentType:  stat.ContentType,
		ETag:         stat.ETag,
		LastModified: stat.LastModified,
	}, nil
}

// ListObjects 列举对象
func (s *MinIOStorage) ListObjects(ctx context.Context, bucket, prefix string, recursive bool) ([]ObjectInfo, error) {
	var objects []ObjectInfo

	objectCh := s.client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: recursive,
	})

	for object := range objectCh {
		if object.Err != nil {
			logger.Errorf(ctx, "[MinIOStorage] Failed to list objects: %v", object.Err)
			return nil, fmt.Errorf("列举文件失败: %w", object.Err)
		}

		objects = append(objects, ObjectInfo{
			Key:          object.Key,
			Size:         object.Size,
			ContentType:  object.ContentType,
			ETag:         object.ETag,
			LastModified: object.LastModified,
		})
	}

	return objects, nil
}

// EnsureBucket 确保 Bucket 存在
func (s *MinIOStorage) EnsureBucket(ctx context.Context, bucket, region string) error {
	exists, err := s.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("检查 Bucket 是否存在失败: %w", err)
	}

	if !exists {
		err = s.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: region})
		if err != nil {
			return fmt.Errorf("创建 Bucket 失败: %w", err)
		}
		logger.Infof(ctx, "[MinIOStorage] Created bucket: %s", bucket)
	}

	return nil
}

// UploadObject 直接上传对象
func (s *MinIOStorage) UploadObject(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		logger.Errorf(ctx, "[MinIOStorage] Failed to upload object %s: %v", key, err)
		return fmt.Errorf("上传文件失败: %w", err)
	}
	return nil
}

// DownloadObject 直接下载对象
func (s *MinIOStorage) DownloadObject(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	object, err := s.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		logger.Errorf(ctx, "[MinIOStorage] Failed to download object %s: %v", key, err)
		return nil, fmt.Errorf("下载文件失败: %w", err)
	}
	return object, nil
}

