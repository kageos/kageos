package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOStorage MinIO 存储实现
type MinIOStorage struct {
	client         *minio.Client // 主客户端：连接 endpoint（服务自身直连 MinIO），用于实际读写操作
	externalClient *minio.Client // 外部客户端：基于 cdn_domain 生成浏览器预签名 URL（无 cdn_domain 时 = client）
	serverClient   *minio.Client // 容器内访问用客户端：基于 server_endpoint 生成预签名 GET（DownloadFiles 用，无则=nil）
	cdnDomain      string        // CDN/代理域名（浏览器访问文件的公开地址，如 Nginx 反向代理域名）
	endpoint       string        // MinIO endpoint（服务自身连接 MinIO 的地址）
	useSSL         bool
	accessKey      string
	secretKey      string
	region         string
	serverEndpoint string // 容器内部访问 MinIO 的地址（用于生成 SDK/容器的内部 URL）
}

// NewMinIOStorage 创建 MinIO 存储
func NewMinIOStorage(cfg Config) (*MinIOStorage, error) {
	// 主客户端：使用 endpoint（服务自身能连通的 MinIO 地址）
	// 本地开发: localhost:9000, Docker 部署: minio:9000
	client, err := minio.New(cfg.GetEndpoint(), &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.GetAccessKey(), cfg.GetSecretKey(), ""),
		Secure: cfg.GetUseSSL(),
		Region: cfg.GetRegion(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// 外部客户端：基于 cdn_domain 生成浏览器可访问的预签名 URL
	// cdn_domain 可以是 Nginx 反向代理的域名，如 "your-domain.com"
	// PresignedPutObject 是纯签名计算，不发起网络请求
	var externalClient *minio.Client
	browserEndpoint, browserUseSSL, hasBrowserEndpoint := endpointFromCDN(cfg.GetCDNDomain(), cfg.GetUseSSL())
	if hasBrowserEndpoint && (browserEndpoint != cfg.GetEndpoint() || browserUseSSL != cfg.GetUseSSL()) {
		ec, err := minio.New(browserEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.GetAccessKey(), cfg.GetSecretKey(), ""),
			Secure: browserUseSSL,
			Region: cfg.GetRegion(),
		})
		if err != nil {
			logger.Warnf(context.Background(), "[MinIOStorage] Failed to create external client for %s, falling back to main client: %v", browserEndpoint, err)
			externalClient = client
		} else {
			externalClient = ec
		}
	} else {
		externalClient = client
	}

	serverEndpoint := cfg.GetServerEndpoint()
	// 容器内下载用客户端：用 server_endpoint 生成预签名 GET，桶保持私有
	var serverClient *minio.Client
	if serverEndpoint != "" && serverEndpoint != cfg.GetEndpoint() {
		sc, err := minio.New(serverEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.GetAccessKey(), cfg.GetSecretKey(), ""),
			Secure: cfg.GetUseSSL(),
			Region: cfg.GetRegion(),
		})
		if err != nil {
			logger.Warnf(context.Background(), "[MinIOStorage] Failed to create server client for %s: %v", serverEndpoint, err)
		} else {
			serverClient = sc
		}
	}

	return &MinIOStorage{
		client:         client,
		externalClient: externalClient,
		serverClient:   serverClient,
		cdnDomain:      cfg.GetCDNDomain(),
		endpoint:       cfg.GetEndpoint(),
		useSSL:         cfg.GetUseSSL(),
		accessKey:      cfg.GetAccessKey(),
		secretKey:      cfg.GetSecretKey(),
		region:         cfg.GetRegion(),
		serverEndpoint: serverEndpoint,
	}, nil
}

// endpointFromCDN 从 cdn_domain 提取 MinIO endpoint，并在带 scheme 时让 presign URL 跟随该 scheme。
func endpointFromCDN(cdnDomain string, defaultUseSSL bool) (endpoint string, useSSL bool, ok bool) {
	cdnDomain = strings.TrimSpace(cdnDomain)
	if cdnDomain == "" {
		return "", defaultUseSSL, false
	}
	if strings.HasPrefix(cdnDomain, "http://") || strings.HasPrefix(cdnDomain, "https://") {
		parsed, err := url.Parse(cdnDomain)
		if err != nil || parsed.Host == "" {
			return "", defaultUseSSL, false
		}
		return parsed.Host, parsed.Scheme == "https", true
	}
	return cdnDomain, defaultUseSSL, true
}

// GetCDNDomain 获取 CDN 域名
func (s *MinIOStorage) GetCDNDomain() string {
	return s.cdnDomain
}

// GetUploadEndpoint 获取上传用的 endpoint
// uploadSource: 上传来源（browser 或 server）
// 统一逻辑：如果配置了 server_endpoint 且 upload_source 是 server，返回 server_endpoint；否则返回默认 endpoint
func (s *MinIOStorage) GetUploadEndpoint(uploadSource string) string {
	if uploadSource == UploadSourceServer && s.serverEndpoint != "" {
		return s.serverEndpoint
	}
	// 如果地址相同，server_endpoint 可以不配置，返回默认 endpoint
	return s.endpoint
}

// GenerateUploadCredentials 生成上传凭证（统一接口）
// 浏览器上传时：优先用配置的 cdn_domain（固定域名，本地/线上各配各的），未配置时才用请求 Host，保证签名与 PUT 的 Host 一致
func (s *MinIOStorage) GenerateUploadCredentials(ctx context.Context, bucket, key, contentType string, expire time.Duration, uploadSource string) (*UploadCredentials, error) {
	// 1. 生成外部访问的 URL（前端浏览器使用）
	// 优先级：配置的 cdn_domain > 请求 Host（PresignHost）。配置后本地/线上都稳定，不依赖网关传 Host
	var externalURL *url.URL
	useConfigHost := s.cdnDomain != "" && uploadSource == UploadSourceBrowser
	if useConfigHost {
		// 用已根据 cdn_domain 建好的 externalClient 生成，签名与配置的域名一致
		var err error
		externalURL, err = s.externalClient.PresignedPutObject(ctx, bucket, key, expire)
		if err != nil {
			logger.Errorf(ctx, "[MinIOStorage] Failed to generate external upload URL (cdn_domain): %v", err)
			return nil, fmt.Errorf("生成上传凭证失败: %w", err)
		}
		logger.Infof(ctx, "[MinIOStorage] GenerateUploadCredentials: using cdn_domain for presign (uploadSource=%s)", uploadSource)
	}
	if externalURL == nil {
		presignHost := contextx.GetPresignHost(ctx)
		logger.Infof(ctx, "[MinIOStorage] GenerateUploadCredentials: uploadSource=%s, presignHost=%q", uploadSource, presignHost)
		if uploadSource == UploadSourceBrowser && presignHost != "" {
			clientForHost, err := minio.New(presignHost, &minio.Options{
				Creds:  credentials.NewStaticV4(s.accessKey, s.secretKey, ""),
				Secure: s.useSSL,
				Region: s.region,
			})
			if err != nil {
				logger.Warnf(ctx, "[MinIOStorage] Failed to create presign client for host %s, fallback to externalClient: %v", presignHost, err)
			} else {
				u, err := clientForHost.PresignedPutObject(ctx, bucket, key, expire)
				if err != nil {
					logger.Warnf(ctx, "[MinIOStorage] PresignedPutObject with request host failed: %v", err)
				} else {
					externalURL = u
				}
			}
		}
	}
	if externalURL == nil {
		var err error
		externalURL, err = s.externalClient.PresignedPutObject(ctx, bucket, key, expire)
		if err != nil {
			logger.Errorf(ctx, "[MinIOStorage] Failed to generate external upload URL: %v", err)
			return nil, fmt.Errorf("生成上传凭证失败: %w", err)
		}
	}

	// 2. 生成内部访问的URL（服务端/SDK使用，用 server_endpoint 生成 → 容器内可达 MinIO）
	var serverURL string
	if s.serverEndpoint != "" && s.serverEndpoint != s.endpoint {
		if s.serverClient == nil {
			logger.Errorf(ctx, "[MinIOStorage] Server client unavailable for server endpoint %s", s.serverEndpoint)
			serverURL = externalURL.String()
		} else if internalURL, err := s.serverClient.PresignedPutObject(ctx, bucket, key, expire); err != nil {
			logger.Errorf(ctx, "[MinIOStorage] Failed to generate internal upload URL: %v", err)
			serverURL = externalURL.String()
		} else {
			serverURL = internalURL.String()
		}
	} else {
		serverURL = externalURL.String()
	}

	// 解析上传域名信息（使用外部URL）
	uploadURLStr := externalURL.String()
	uploadHost, uploadDomain := s.extractDomainInfo(uploadURLStr)

	creds := &UploadCredentials{
		Method:          UploadMethodPresignedURL,
		UploadURL:       uploadURLStr,
		ServerUploadURL: serverURL,
		Headers: map[string]string{
			ContentTypeHeader: contentType,
		},
		UploadHost:   uploadHost,
		UploadDomain: uploadDomain,
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

// GenerateDownloadURLs 生成下载 URL（同时生成外部和内部访问的URL）
func (s *MinIOStorage) GenerateDownloadURLs(ctx context.Context, bucket, key string, expire time.Duration, cacheControl map[string]string) (externalURL string, serverURL string, err error) {
	scheme := "http"
	if s.useSSL {
		scheme = "https"
	}

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
		// 返回相对路径，前端通过 Nginx 反向代理（location /{bucket}/）访问 MinIO
		// 好处：零配置、无跨域、开发/生产一致
		externalURL = fmt.Sprintf("/%s/%s", bucket, key)
	}

	// 内部访问URL（容器/SDK 用，DownloadFiles 会 GET 此 URL）
	// 使用预签名 GET，桶可保持私有，容器内无需带鉴权即可下载
	if s.serverClient != nil {
		u, err := s.serverClient.PresignedGetObject(ctx, bucket, key, expire, nil)
		if err != nil {
			logger.Warnf(ctx, "[MinIOStorage] PresignedGetObject for serverURL failed: %v, fallback to plain URL", err)
			serverURL = fmt.Sprintf("%s://%s/%s/%s", scheme, s.serverEndpoint, bucket, key)
		} else {
			serverURL = u.String()
		}
	} else if s.serverEndpoint != "" && s.serverEndpoint != s.endpoint {
		serverURL = fmt.Sprintf("%s://%s/%s/%s", scheme, s.serverEndpoint, bucket, key)
	} else {
		serverURL = fmt.Sprintf("%s://%s/%s/%s", scheme, s.endpoint, bucket, key)
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
		if isMinIOTimeSkewError(err) {
			currentTime := time.Now().Format("2006-01-02 15:04:05 MST")
			logger.Errorf(ctx, "[MinIOStorage] 时间同步错误 - 当前系统时间: %s", currentTime)
			return fmt.Errorf("时间同步错误：客户端与 MinIO 服务器的时间差过大（通常超过15分钟）。当前系统时间: %s。请同步宿主机和容器运行时 VM 的时间；macOS 可执行 `sudo sntp -sS time.apple.com`，Podman 本地开发可执行 `podman machine stop && podman machine start`，Linux 可执行 `sudo timedatectl set-ntp true`。注意：TZ 或 /etc/localtime 只影响时区显示，不能修正绝对时间偏移: %w", currentTime, err)
		}
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

func isMinIOTimeSkewError(err error) bool {
	if err == nil {
		return false
	}

	var minioErr minio.ErrorResponse
	if errors.As(err, &minioErr) && minioErr.Code == "RequestTimeTooSkewed" {
		return true
	}

	errMsg := err.Error()
	return strings.Contains(errMsg, "difference between the request time") ||
		strings.Contains(errMsg, "time is too large") ||
		strings.Contains(errMsg, "RequestTimeTooSkewed")
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
