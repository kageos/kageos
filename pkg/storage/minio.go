package storage

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/netprobe"
)

// MinIOUploader MinIO 上传器实现
// 使用 presigned_url 方式上传（PUT请求）
type MinIOUploader struct{}

var minIOUploadHTTPClient = &http.Client{
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true,
		WriteBufferSize:     256 * 1024,
		ReadBufferSize:      256 * 1024,
	},
	Timeout: 10 * time.Minute,
}

// NewMinIOUploader 创建 MinIO 上传器
func NewMinIOUploader() *MinIOUploader {
	return &MinIOUploader{}
}

// Upload 上传文件到 MinIO
// Upload 使用服务端可达的短期预签名 URL 上传，不接收对象存储长期凭据。
func (u *MinIOUploader) Upload(ctx context.Context, creds *dto.GetUploadTokenResp, fileReader io.Reader, fileSize int64, hash string) (*UploadResult, error) {
	// ✨ 性能监控：记录开始时间
	startTime := time.Now()

	return u.uploadWithHTTP(ctx, creds, fileReader, fileSize, hash, startTime)
}

// uploadWithHTTP 使用 HTTP PUT 方式上传（原有逻辑）
func (u *MinIOUploader) uploadWithHTTP(ctx context.Context, creds *dto.GetUploadTokenResp, fileReader io.Reader, fileSize int64, hash string, startTime time.Time) (*UploadResult, error) {
	prepareStart := startTime

	if creds.Method != dto.UploadMethodPresignedURL {
		return nil, fmt.Errorf("MinIO 只支持 presigned_url 上传方式，当前方式: %s", creds.Method)
	}

	// ✨ 服务端上传使用 ServerURL（内部访问URL）
	uploadURL := resolveUploadURL(ctx, creds)

	// 验证必要的字段
	if uploadURL == "" {
		return nil, ErrInvalidCredentials
	}

	prepareTime := time.Since(prepareStart)
	logger.Debugf(ctx, "[MinIOUploader] 准备阶段耗时: %v, 文件大小: %d bytes (使用HTTP PUT)", prepareTime, fileSize)

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
	logger.Debugf(ctx, "[MinIOUploader] 创建请求耗时: %v", reqTime)

	// 执行上传
	// ✨ 优化HTTP客户端配置，提升上传性能
	uploadStart := time.Now()
	logger.Debugf(ctx, "[MinIOUploader] 开始上传(HTTP PUT): Size=%d bytes", fileSize)

	resp, err := minIOUploadHTTPClient.Do(req)
	if err != nil {
		uploadTime := time.Since(uploadStart)
		logger.Errorf(ctx, "[MinIOUploader] 上传失败: 耗时=%v, 错误=%v", uploadTime, err)
		return nil, fmt.Errorf("%w: 上传请求失败: %v", ErrUploadFailed, err)
	}
	defer resp.Body.Close()

	uploadTime := time.Since(uploadStart)
	uploadSpeed := float64(fileSize) / uploadTime.Seconds()
	logger.Debugf(ctx, "[MinIOUploader] 上传完成(HTTP PUT): 耗时=%v, 速度=%.2f MB/s (%.2f bytes/s)",
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
	logUploadSummary(ctx, "HTTP PUT", fileSize, totalTime, prepareTime, reqTime, uploadTime, uploadSpeed)

	result := &UploadResult{
		Key:               creds.Key,
		ETag:              etag,
		Hash:              hash,
		Size:              fileSize,
		ContentType:       getUploadContentType(creds),
		DownloadURL:       creds.DownloadURL,
		ServerDownloadURL: creds.ServerDownloadURL,
	}

	logger.Debugf(ctx, "[MinIOUploader] Upload successful (HTTP PUT): key=%s, etag=%s, hash=%s", creds.Key, etag, hash)
	return result, nil
}

func logUploadSummary(ctx context.Context, mode string, fileSize int64, totalTime, prepareTime, setupTime, uploadTime time.Duration, uploadSpeed float64) {
	const message = "[MinIOUploader] upload summary(%s): total=%v prepare=%v setup=%v upload=%v size=%d speed=%.2f MB/s"
	if threshold := slowUploadThreshold(); threshold > 0 && totalTime >= threshold {
		logger.Warnf(ctx, message, mode, totalTime, prepareTime, setupTime, uploadTime, fileSize, uploadSpeed/(1024*1024))
		return
	}
	logger.Debugf(ctx, message, mode, totalTime, prepareTime, setupTime, uploadTime, fileSize, uploadSpeed/(1024*1024))
}

func slowUploadThreshold() time.Duration {
	const defaultThreshold = 5 * time.Second
	raw := strings.TrimSpace(os.Getenv("KAGEOS_SDK_SLOW_UPLOAD_MS"))
	if raw == "" {
		return defaultThreshold
	}
	ms, err := strconv.Atoi(raw)
	if err != nil {
		return defaultThreshold
	}
	if ms <= 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

func resolveUploadURL(ctx context.Context, creds *dto.GetUploadTokenResp) string {
	if ctx == nil {
		ctx = context.Background()
	}
	if creds == nil {
		return ""
	}
	var uploadURL string
	if creds.ServerUploadURL != "" {
		uploadURL = creds.ServerUploadURL
	} else {
		uploadURL = creds.UploadURL
	}
	if resolvedURL, err := netprobe.ResolveHTTPURLHostCached(ctx, "minio-http-upload", uploadURL, time.Second); err == nil && resolvedURL != "" {
		return resolvedURL
	}
	return uploadURL
}

func getUploadContentType(creds *dto.GetUploadTokenResp) string {
	if creds.Headers == nil {
		return ""
	}
	return creds.Headers["Content-Type"]
}
