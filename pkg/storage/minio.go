package storage

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOUploader MinIO 上传器实现
// 使用 presigned_url 方式上传（PUT请求）
type MinIOUploader struct{}

// NewMinIOUploader 创建 MinIO 上传器
func NewMinIOUploader() *MinIOUploader {
	return &MinIOUploader{}
}

// Upload 上传文件到 MinIO
// 优先使用 MinIO SDK 直接上传（如果提供了SDKConfig），否则使用 presigned_url + HTTP PUT
func (u *MinIOUploader) Upload(ctx context.Context, creds *dto.GetUploadTokenResp, fileReader io.Reader, fileSize int64, hash string) (*UploadResult, error) {
	// ✨ 性能监控：记录开始时间
	startTime := time.Now()
	prepareStart := startTime

	// ✨ 如果提供了SDKConfig，使用MinIO SDK直接上传（性能更好）
	if creds.SDKConfig != nil && len(creds.SDKConfig) > 0 {
		return u.uploadWithSDK(ctx, creds, fileReader, fileSize, hash, startTime)
	}

	// 否则使用HTTP PUT方式（原有逻辑）
	// 验证上传方式
	if creds.Method != dto.UploadMethodPresignedURL {
		return nil, fmt.Errorf("MinIO 只支持 presigned_url 上传方式，当前方式: %s", creds.Method)
	}

	// ✨ 服务端上传使用 ServerURL（内部访问URL）
	uploadURL := creds.ServerURL
	if uploadURL == "" {
		// 如果ServerURL为空，降级使用URL（兼容处理）
		uploadURL = creds.URL
	}
	
	// 验证必要的字段
	if uploadURL == "" {
		return nil, ErrInvalidCredentials
	}

	prepareTime := time.Since(prepareStart)
	logger.Infof(ctx, "[MinIOUploader] 准备阶段耗时: %v, 文件大小: %d bytes (使用HTTP PUT)", prepareTime, fileSize)

	// 创建 PUT 请求（使用服务端URL）
	reqStart := time.Now()
	req, err := http.NewRequestWithContext(ctx, "PUT", uploadURL, fileReader)
	if err != nil {
		logger.Errorf(ctx, "[MinIOUploader] Failed to create request: %v", err)
		return nil, fmt.Errorf("%w: 创建请求失败: %v", ErrUploadFailed, err)
	}

	// 设置请求头
	req.ContentLength = fileSize
	if creds.Headers != nil {
		for key, value := range creds.Headers {
			req.Header.Set(key, value)
		}
	}
	reqTime := time.Since(reqStart)
	logger.Infof(ctx, "[MinIOUploader] 创建请求耗时: %v", reqTime)

	// 执行上传
	// ✨ 优化HTTP客户端配置，提升上传性能
	// 使用自定义Transport，配置合适的超时和连接池参数
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second, // 连接超时
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,              // 最大空闲连接数
		MaxIdleConnsPerHost: 10,               // 每个host的最大空闲连接数
		IdleConnTimeout:     90 * time.Second, // 空闲连接超时
		DisableCompression:  true,             // 禁用压缩（上传文件通常已压缩）
		// 使用更大的写缓冲区以提升IO性能
		WriteBufferSize: 256 * 1024, // 256KB
		ReadBufferSize:  256 * 1024, // 256KB
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Minute, // 总超时时间（大文件上传需要更长时间）
	}

	// ✨ 性能监控：记录实际上传开始时间
	uploadStart := time.Now()
	logger.Infof(ctx, "[MinIOUploader] 开始上传: URL=%s, Size=%d bytes", uploadURL, fileSize)

	resp, err := client.Do(req)
	if err != nil {
		uploadTime := time.Since(uploadStart)
		logger.Errorf(ctx, "[MinIOUploader] 上传失败: 耗时=%v, 错误=%v", uploadTime, err)
		return nil, fmt.Errorf("%w: 上传请求失败: %v", ErrUploadFailed, err)
	}
	defer resp.Body.Close()

	// ✨ 性能监控：记录上传完成时间
	uploadTime := time.Since(uploadStart)
	uploadSpeed := float64(fileSize) / uploadTime.Seconds()
	logger.Infof(ctx, "[MinIOUploader] 上传完成: 耗时=%v, 速度=%.2f MB/s (%.2f bytes/s)", 
		uploadTime, uploadSpeed/(1024*1024), uploadSpeed)

	// 检查响应状态
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		// 读取错误响应体（可选，用于调试）
		bodyBytes := make([]byte, 1024)
		resp.Body.Read(bodyBytes)
		logger.Errorf(ctx, "[MinIOUploader] Upload failed: status=%d, body=%s", resp.StatusCode, string(bodyBytes))
		return nil, fmt.Errorf("%w: 上传失败，状态码: %d", ErrUploadFailed, resp.StatusCode)
	}

	// 从响应头获取 ETag（MinIO 会在响应头中返回 ETag）
	etag := resp.Header.Get("ETag")
	// 移除 ETag 的引号（如果存在）
	etag = strings.Trim(etag, `"`)

	// ✨ 性能监控：总耗时
	totalTime := time.Since(startTime)
	logger.Infof(ctx, "[MinIOUploader] 总耗时: %v (准备:%v, 创建请求:%v, 上传:%v), 平均速度: %.2f MB/s", 
		totalTime, prepareTime, reqTime, uploadTime, uploadSpeed/(1024*1024))

	// 构建结果
	result := &UploadResult{
		Key:              creds.Key,
		ETag:             etag,                    // MinIO 返回的 ETag（对于小文件是 MD5，大文件是 multipart 标识）
		Hash:             hash,                     // 上传前计算的 SHA256 hash（用于秒传）
		Size:             fileSize,
		ContentType:      creds.Headers["Content-Type"],
		DownloadURL:      creds.DownloadURL,        // ✨ 外部访问的下载地址（前端使用）
		ServerDownloadURL: creds.ServerDownloadURL, // ✨ 内部访问的下载地址（服务端使用）
	}

	logger.Infof(ctx, "[MinIOUploader] Upload successful: key=%s, etag=%s, hash=%s", creds.Key, etag, hash)
	return result, nil
}

// uploadWithSDK 使用 MinIO SDK 直接上传（性能更好）
func (u *MinIOUploader) uploadWithSDK(ctx context.Context, creds *dto.GetUploadTokenResp, fileReader io.Reader, fileSize int64, hash string, startTime time.Time) (*UploadResult, error) {
	prepareStart := startTime

	// 从SDKConfig中提取MinIO连接信息
	endpoint, _ := creds.SDKConfig["endpoint"].(string)
	accessKey, _ := creds.SDKConfig["access_key"].(string)
	secretKey, _ := creds.SDKConfig["secret_key"].(string)
	region, _ := creds.SDKConfig["region"].(string)
	useSSL, _ := creds.SDKConfig["use_ssl"].(bool)
	bucket, _ := creds.SDKConfig["bucket"].(string)

	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		logger.Warnf(ctx, "[MinIOUploader] SDKConfig不完整，降级使用HTTP PUT方式")
		// 降级使用HTTP PUT
		return u.uploadWithHTTP(ctx, creds, fileReader, fileSize, hash, startTime)
	}

	if region == "" {
		region = "us-east-1" // 默认region
	}

	prepareTime := time.Since(prepareStart)
	logger.Infof(ctx, "[MinIOUploader] 准备阶段耗时: %v, 文件大小: %d bytes (使用MinIO SDK)", prepareTime, fileSize)

	// 创建MinIO客户端
	clientStart := time.Now()
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
		Region: region,
	})
	if err != nil {
		logger.Errorf(ctx, "[MinIOUploader] 创建MinIO客户端失败: %v", err)
		return nil, fmt.Errorf("%w: 创建MinIO客户端失败: %v", ErrUploadFailed, err)
	}
	clientTime := time.Since(clientStart)
	logger.Infof(ctx, "[MinIOUploader] 创建客户端耗时: %v", clientTime)

	// 执行上传
	uploadStart := time.Now()
	contentType := "application/octet-stream"
	if creds.Headers != nil && creds.Headers["Content-Type"] != "" {
		contentType = creds.Headers["Content-Type"]
	}

	logger.Infof(ctx, "[MinIOUploader] 开始上传(MinIO SDK): endpoint=%s, bucket=%s, key=%s, Size=%d bytes", endpoint, bucket, creds.Key, fileSize)

	info, err := client.PutObject(ctx, bucket, creds.Key, fileReader, fileSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		uploadTime := time.Since(uploadStart)
		logger.Errorf(ctx, "[MinIOUploader] 上传失败: 耗时=%v, 错误=%v", uploadTime, err)
		return nil, fmt.Errorf("%w: 上传失败: %v", ErrUploadFailed, err)
	}

	uploadTime := time.Since(uploadStart)
	uploadSpeed := float64(fileSize) / uploadTime.Seconds()
	logger.Infof(ctx, "[MinIOUploader] 上传完成(MinIO SDK): 耗时=%v, 速度=%.2f MB/s (%.2f bytes/s)", 
		uploadTime, uploadSpeed/(1024*1024), uploadSpeed)

	// 获取ETag
	etag := info.ETag
	etag = strings.Trim(etag, `"`)

	// ✨ 性能监控：总耗时
	totalTime := time.Since(startTime)
	logger.Infof(ctx, "[MinIOUploader] 总耗时(MinIO SDK): %v (准备:%v, 创建客户端:%v, 上传:%v), 平均速度: %.2f MB/s", 
		totalTime, prepareTime, clientTime, uploadTime, uploadSpeed/(1024*1024))

	// 构建结果
	result := &UploadResult{
		Key:              creds.Key,
		ETag:             etag,
		Hash:             hash,
		Size:             fileSize,
		ContentType:      contentType,
		DownloadURL:      creds.DownloadURL,
		ServerDownloadURL: creds.ServerDownloadURL,
	}

	logger.Infof(ctx, "[MinIOUploader] Upload successful (MinIO SDK): key=%s, etag=%s, hash=%s", creds.Key, etag, hash)
	return result, nil
}

// uploadWithHTTP 使用 HTTP PUT 方式上传（原有逻辑）
func (u *MinIOUploader) uploadWithHTTP(ctx context.Context, creds *dto.GetUploadTokenResp, fileReader io.Reader, fileSize int64, hash string, startTime time.Time) (*UploadResult, error) {
	prepareStart := startTime

	// ✨ 服务端上传使用 ServerURL（内部访问URL）
	uploadURL := creds.ServerURL
	if uploadURL == "" {
		// 如果ServerURL为空，降级使用URL（兼容处理）
		uploadURL = creds.URL
	}
	
	// 验证必要的字段
	if uploadURL == "" {
		return nil, ErrInvalidCredentials
	}

	prepareTime := time.Since(prepareStart)
	logger.Infof(ctx, "[MinIOUploader] 准备阶段耗时: %v, 文件大小: %d bytes (使用HTTP PUT)", prepareTime, fileSize)

	// 创建 PUT 请求（使用服务端URL）
	reqStart := time.Now()
	req, err := http.NewRequestWithContext(ctx, "PUT", uploadURL, fileReader)
	if err != nil {
		logger.Errorf(ctx, "[MinIOUploader] Failed to create request: %v", err)
		return nil, fmt.Errorf("%w: 创建请求失败: %v", ErrUploadFailed, err)
	}

	// 设置请求头
	req.ContentLength = fileSize
	if creds.Headers != nil {
		for key, value := range creds.Headers {
			req.Header.Set(key, value)
		}
	}
	reqTime := time.Since(reqStart)
	logger.Infof(ctx, "[MinIOUploader] 创建请求耗时: %v", reqTime)

	// 执行上传
	// ✨ 优化HTTP客户端配置，提升上传性能
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true,
		WriteBufferSize: 256 * 1024,
		ReadBufferSize:  256 * 1024,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Minute,
	}

	uploadStart := time.Now()
	logger.Infof(ctx, "[MinIOUploader] 开始上传(HTTP PUT): URL=%s, Size=%d bytes", uploadURL, fileSize)

	resp, err := client.Do(req)
	if err != nil {
		uploadTime := time.Since(uploadStart)
		logger.Errorf(ctx, "[MinIOUploader] 上传失败: 耗时=%v, 错误=%v", uploadTime, err)
		return nil, fmt.Errorf("%w: 上传请求失败: %v", ErrUploadFailed, err)
	}
	defer resp.Body.Close()

	uploadTime := time.Since(uploadStart)
	uploadSpeed := float64(fileSize) / uploadTime.Seconds()
	logger.Infof(ctx, "[MinIOUploader] 上传完成(HTTP PUT): 耗时=%v, 速度=%.2f MB/s (%.2f bytes/s)", 
		uploadTime, uploadSpeed/(1024*1024), uploadSpeed)

	// 检查响应状态
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyBytes := make([]byte, 1024)
		resp.Body.Read(bodyBytes)
		logger.Errorf(ctx, "[MinIOUploader] Upload failed: status=%d, body=%s", resp.StatusCode, string(bodyBytes))
		return nil, fmt.Errorf("%w: 上传失败，状态码: %d", ErrUploadFailed, resp.StatusCode)
	}

	// 从响应头获取 ETag
	etag := resp.Header.Get("ETag")
	etag = strings.Trim(etag, `"`)

	totalTime := time.Since(startTime)
	logger.Infof(ctx, "[MinIOUploader] 总耗时(HTTP PUT): %v (准备:%v, 创建请求:%v, 上传:%v), 平均速度: %.2f MB/s", 
		totalTime, prepareTime, reqTime, uploadTime, uploadSpeed/(1024*1024))

	result := &UploadResult{
		Key:              creds.Key,
		ETag:             etag,
		Hash:             hash,
		Size:             fileSize,
		ContentType:      creds.Headers["Content-Type"],
		DownloadURL:      creds.DownloadURL,
		ServerDownloadURL: creds.ServerDownloadURL,
	}

	logger.Infof(ctx, "[MinIOUploader] Upload successful (HTTP PUT): key=%s, etag=%s, hash=%s", creds.Key, etag, hash)
	return result, nil
}

