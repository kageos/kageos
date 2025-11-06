package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOStorage MinIO 存储实现
type MinIOStorage struct {
	client         *minio.Client
	cdnDomain      string // CDN 域名（可选）
	endpoint       string // MinIO endpoint（用于构建公开URL）
	useSSL         bool   // 是否使用SSL
	accessKey      string // Access Key（用于创建临时客户端）
	secretKey      string // Secret Key（用于创建临时客户端）
	region         string // Region（用于创建临时客户端）
	serverEndpoint string // ✨ 服务端上传用的 endpoint（容器内访问）
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

	// ✨ 获取 server_endpoint（如果配置了）
	serverEndpoint := ""
	// 通过类型断言获取 serverEndpoint（如果 StorageConfigAdapter 实现了 GetServerEndpoint）
	if adapter, ok := cfg.(interface{ GetServerEndpoint() string }); ok {
		serverEndpoint = adapter.GetServerEndpoint()
	}

	return &MinIOStorage{
		client:         client,
		cdnDomain:      cfg.GetCDNDomain(),
		endpoint:       cfg.GetEndpoint(),
		useSSL:         cfg.GetUseSSL(),
		accessKey:      cfg.GetAccessKey(),
		secretKey:      cfg.GetSecretKey(),
		region:         cfg.GetRegion(),
		serverEndpoint: serverEndpoint,
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

// GetUploadEndpoint 获取上传用的 endpoint
// uploadSource: 上传来源（browser 或 server）
// 统一逻辑：如果配置了 server_endpoint 且 upload_source 是 server，返回 server_endpoint；否则返回默认 endpoint
func (s *MinIOStorage) GetUploadEndpoint(uploadSource string) string {
	if uploadSource == "server" && s.serverEndpoint != "" {
		return s.serverEndpoint
	}
	// 如果地址相同，server_endpoint 可以不配置，返回默认 endpoint
	return s.endpoint
}

// GenerateUploadCredentials 生成上传凭证（同时生成外部和内部访问的URL）
func (s *MinIOStorage) GenerateUploadCredentials(ctx context.Context, bucket, key, contentType string, expire time.Duration) (*UploadCredentials, error) {
	// ✨ 生成两个URL：外部访问（默认endpoint）和内部访问（server_endpoint）
	// 1. 生成外部访问的URL（前端使用）
	externalURL, err := s.client.PresignedPutObject(ctx, bucket, key, expire)
	if err != nil {
		logger.Errorf(ctx, "[MinIOStorage] Failed to generate external upload URL: %v", err)
		return nil, fmt.Errorf("生成上传凭证失败: %w", err)
	}

	// 2. 生成内部访问的URL（服务端使用，如果配置了server_endpoint）
	var serverURL string
	if s.serverEndpoint != "" && s.serverEndpoint != s.endpoint {
		// 创建临时 client 生成内部访问的预签名 URL
		tempClient, err := minio.New(s.serverEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(s.accessKey, s.secretKey, ""),
			Secure: s.useSSL,
			Region: s.region,
		})
		if err != nil {
			logger.Errorf(ctx, "[MinIOStorage] Failed to create temp client for server endpoint %s: %v", s.serverEndpoint, err)
			// 如果创建失败，使用外部URL（降级处理）
			serverURL = externalURL.String()
		} else {
			internalURL, err := tempClient.PresignedPutObject(ctx, bucket, key, expire)
			if err != nil {
				logger.Errorf(ctx, "[MinIOStorage] Failed to generate internal upload URL: %v", err)
				// 如果生成失败，使用外部URL（降级处理）
				serverURL = externalURL.String()
			} else {
				serverURL = internalURL.String()
			}
		}
	} else {
		// 如果地址相同或未配置server_endpoint，使用外部URL
		serverURL = externalURL.String()
	}

	// 解析上传域名信息（使用外部URL）
	uploadHost, uploadDomain := s.extractDomainInfo(externalURL.String())

	return &UploadCredentials{
		Method:    UploadMethodPresignedURL,
		URL:       externalURL.String(), // 外部访问URL
		ServerURL: serverURL,            // 内部访问URL
		Headers: map[string]string{
			"Content-Type": contentType,
		},
		UploadHost:   uploadHost,   // ✨ 上传目标 host（用于 CORS、进度监听）
		UploadDomain: uploadDomain, // ✨ 上传完整域名（用于日志、调试）
	}, nil
}

// GenerateUploadCredentialsWithEndpoint 生成上传凭证（支持指定 endpoint）
// uploadEndpoint: 上传用的 endpoint（如果与默认 endpoint 相同，则使用默认 client）
// uploadSource: 上传来源（browser 或 server），用于决定是否返回SDK配置
func (s *MinIOStorage) GenerateUploadCredentialsWithEndpoint(ctx context.Context, bucket, key, contentType string, expire time.Duration, uploadEndpoint string, uploadSource string) (*UploadCredentials, error) {
	var presignedURL *url.URL
	var err error

	// ✨ 如果指定的 endpoint 与默认 endpoint 相同，直接使用默认 client（避免创建重复的 client）
	if uploadEndpoint == "" || uploadEndpoint == s.endpoint {
		// 使用默认 client
		presignedURL, err = s.client.PresignedPutObject(ctx, bucket, key, expire)
	} else {
		// endpoint 不同，创建临时 client 生成预签名 URL
		var tempClient *minio.Client
		tempClient, err = minio.New(uploadEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(s.accessKey, s.secretKey, ""),
			Secure: s.useSSL,
			Region: s.region,
		})
		if err != nil {
			logger.Errorf(ctx, "[MinIOStorage] Failed to create temp client for endpoint %s: %v", uploadEndpoint, err)
			return nil, fmt.Errorf("创建临时客户端失败: %w", err)
		}
		presignedURL, err = tempClient.PresignedPutObject(ctx, bucket, key, expire)
	}

	if err != nil {
		logger.Errorf(ctx, "[MinIOStorage] Failed to generate upload credentials: %v", err)
		return nil, fmt.Errorf("生成上传凭证失败: %w", err)
	}

	// 解析上传域名信息
	uploadURL := presignedURL.String()
	uploadHost, uploadDomain := s.extractDomainInfo(uploadURL)

	creds := &UploadCredentials{
		Method: UploadMethodPresignedURL,
		URL:    uploadURL,
		Headers: map[string]string{
			"Content-Type": contentType,
		},
		UploadHost:   uploadHost,   // ✨ 上传目标 host（用于 CORS、进度监听）
		UploadDomain: uploadDomain, // ✨ 上传完整域名（用于日志、调试）
	}

	// ✨ 如果是服务端上传，在SDKConfig中放入MinIO连接信息（用于SDK直接上传）
	if uploadSource == "server" {
		creds.SDKConfig = map[string]interface{}{
			"endpoint":   uploadEndpoint, // 使用实际的上传endpoint（可能是server_endpoint）
			"access_key": s.accessKey,
			"secret_key": s.secretKey,
			"region":     s.region,
			"use_ssl":    s.useSSL,
			"bucket":     bucket,
		}
		logger.Infof(ctx, "[MinIOStorage] Added SDK config for server upload: endpoint=%s", uploadEndpoint)
	}

	return creds, nil
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

// GenerateDownloadURL 生成下载 URL（返回外部访问URL，用于兼容旧接口）
func (s *MinIOStorage) GenerateDownloadURL(ctx context.Context, bucket, key string, expire time.Duration, cacheControl map[string]string) (string, error) {
	externalURL, _, err := s.GenerateDownloadURLs(ctx, bucket, key, expire, cacheControl)
	return externalURL, err
}

// GenerateDownloadURLs 生成下载 URL（同时生成外部和内部访问的URL）
func (s *MinIOStorage) GenerateDownloadURLs(ctx context.Context, bucket, key string, expire time.Duration, cacheControl map[string]string) (externalURL string, serverURL string, err error) {
	// 构建外部访问URL
	scheme := "http"
	if s.useSSL {
		scheme = "https"
	}

	// 如果配置了CDN域名，使用CDN
	if s.cdnDomain != "" {
		cdnURL := s.cdnDomain
		if !strings.HasPrefix(cdnURL, "http://") && !strings.HasPrefix(cdnURL, "https://") {
			if s.useSSL {
				cdnURL = "https://" + cdnURL
			} else {
				cdnURL = "http://" + cdnURL
			}
		}
		externalURL = fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(cdnURL, "/"), bucket, key)
	} else {
		// 直接返回公开URL（bucket已设置为public-read）
		externalURL = fmt.Sprintf("%s://%s/%s/%s", scheme, s.endpoint, bucket, key)
	}

	// 构建内部访问URL（如果配置了server_endpoint且与endpoint不同）
	if s.serverEndpoint != "" && s.serverEndpoint != s.endpoint {
		serverURL = fmt.Sprintf("%s://%s/%s/%s", scheme, s.serverEndpoint, bucket, key)
	} else {
		// 如果地址相同或未配置server_endpoint，使用外部URL
		serverURL = externalURL
	}

	return externalURL, serverURL, nil
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
	// 先测试连接，列出所有bucket来验证权限
	_, err := s.client.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("无法连接到MinIO或权限不足，请检查access_key和secret_key配置: %w", err)
	}

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

		// 设置bucket策略为public-read，允许直接访问
		policy := `{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Principal": {"AWS": ["*"]},
					"Action": ["s3:GetObject"],
					"Resource": ["arn:aws:s3:::` + bucket + `/*"]
				}
			]
		}`
		err = s.client.SetBucketPolicy(ctx, bucket, policy)
		if err != nil {
			logger.Warnf(ctx, "[MinIOStorage] Failed to set bucket policy for %s: %v", bucket, err)
		} else {
			logger.Infof(ctx, "[MinIOStorage] Set bucket policy for %s to allow public read", bucket)
		}
	} else {
		// bucket已存在，强制设置策略（确保策略正确）
		policy := `{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Principal": {"AWS": ["*"]},
					"Action": ["s3:GetObject"],
					"Resource": ["arn:aws:s3:::` + bucket + `/*"]
				}
			]
		}`
		err = s.client.SetBucketPolicy(ctx, bucket, policy)
		if err != nil {
			logger.Warnf(ctx, "[MinIOStorage] Failed to set bucket policy for existing bucket %s: %v", bucket, err)
		} else {
			logger.Infof(ctx, "[MinIOStorage] Updated bucket policy for %s to allow public read", bucket)
		}
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
