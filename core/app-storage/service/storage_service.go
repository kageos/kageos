package service

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-storage/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-storage/repository"
	"github.com/ai-agent-os/ai-agent-os/core/app-storage/storage"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/google/uuid"
)

// StorageService 存储服务
type StorageService struct {
	storage  storage.Storage               // ✅ 依赖抽象接口，不依赖具体实现
	cfg      *config.AppStorageConfig
	fileRepo *repository.FileRepository
}

// NewStorageService 创建存储服务
func NewStorageService(storage storage.Storage, cfg *config.AppStorageConfig, fileRepo *repository.FileRepository) *StorageService {
	return &StorageService{
		storage:  storage,
		cfg:      cfg,
		fileRepo: fileRepo,
	}
}

// GenerateUploadToken 生成上传凭证
func (s *StorageService) GenerateUploadToken(ctx context.Context, router string, fileName string, contentType string, fileSize int64) (creds *storage.UploadCredentials, key string, expire time.Time, err error) {
	// 校验文件大小
	if fileSize > s.cfg.Storage.Upload.MaxSize {
		return nil, "", time.Time{}, fmt.Errorf("文件大小超过限制（最大 %d MB）", s.cfg.Storage.Upload.MaxSize/1024/1024)
	}

	// 生成唯一的文件 Key（包含函数路径）
	key = s.generateFileKey(router, fileName)

	// 通过存储接口生成上传凭证
	bucket := s.getDefaultBucket()
	expiry := time.Duration(s.cfg.Storage.Upload.TokenExpire) * time.Second

	creds, err = s.storage.GenerateUploadCredentials(ctx, bucket, key, contentType, expiry)
	if err != nil {
		logger.Errorf(ctx, "Failed to generate upload credentials: %v", err)
		return nil, "", time.Time{}, fmt.Errorf("生成上传凭证失败")
	}

	expire = time.Now().Add(expiry)
	logger.Infof(ctx, "Generated upload token for file: %s, key: %s, method: %s", fileName, key, creds.Method)

	return creds, key, expire, nil
}

// GetFileURL 获取文件访问 URL
func (s *StorageService) GetFileURL(ctx context.Context, key string) (downloadURL string, expire time.Time, err error) {
	// 生成预签名下载 URL（7 天有效）
	bucket := s.getDefaultBucket()
	expiry := 7 * 24 * time.Hour

	// ✅ 注意：MinIO/S3 不支持 response-cache-control 和 response-expires 作为查询参数
	// 这些响应头应该在存储对象时通过元数据设置，或者在代理层添加
	// 暂时不添加查询参数，确保预签名 URL 能正常工作
	cacheControl := make(map[string]string)
	// TODO: 未来可以通过对象元数据或在代理层设置 Cache-Control 响应头

	presignedURL, err := s.storage.GenerateDownloadURL(ctx, bucket, key, expiry, cacheControl)
	if err != nil {
		logger.Errorf(ctx, "Failed to generate download URL: %v", err)
		return "", time.Time{}, fmt.Errorf("生成下载链接失败")
	}

	expire = time.Now().Add(expiry)
	logger.Infof(ctx, "Generated download URL for key: %s", key)
	return presignedURL, expire, nil
}

// DeleteFile 删除文件
func (s *StorageService) DeleteFile(ctx context.Context, key string) error {
	bucket := s.getDefaultBucket()
	err := s.storage.DeleteObject(ctx, bucket, key)
	if err != nil {
		logger.Errorf(ctx, "Failed to delete file %s: %v", key, err)
		return fmt.Errorf("删除文件失败")
	}
	logger.Infof(ctx, "Deleted file: %s", key)
	return nil
}

// GetFileInfo 获取文件信息
func (s *StorageService) GetFileInfo(ctx context.Context, key string) (*storage.ObjectInfo, error) {
	bucket := s.getDefaultBucket()
	info, err := s.storage.GetObjectInfo(ctx, bucket, key)
	if err != nil {
		logger.Errorf(ctx, "Failed to get file info for %s: %v", key, err)
		return nil, fmt.Errorf("获取文件信息失败")
	}
	return info, nil
}

// GetBucketName 获取默认 Bucket 名称
func (s *StorageService) GetBucketName() string {
	return s.getDefaultBucket()
}

// GetCDNDomain 获取 CDN 域名
func (s *StorageService) GetCDNDomain() string {
	return s.storage.GetCDNDomain()
}

// GetStorage 获取存储接口（用于直接访问存储方法）
func (s *StorageService) GetStorage() storage.Storage {
	return s.storage
}

// GetStorageType 获取存储引擎类型
func (s *StorageService) GetStorageType() string {
	return s.cfg.Storage.Type
}

// getDefaultBucket 获取默认 Bucket（内部方法）
func (s *StorageService) getDefaultBucket() string {
	switch s.cfg.Storage.Type {
	case "minio":
		return s.cfg.Storage.MinIO.DefaultBucket
	case "tencentcos":
		return s.cfg.Storage.TencentCOS.DefaultBucket
	case "aliyunoss":
		return s.cfg.Storage.AliyunOSS.DefaultBucket
	case "awss3":
		return s.cfg.Storage.AWSS3.DefaultBucket
	default:
		return s.cfg.Storage.MinIO.DefaultBucket
	}
}

// generateFileKey 生成文件存储路径
// 格式：{router}/{date}/{uuid}.{ext}
// 例如：luobei/test88888/tools/cashier_desk/2025/01/03/xxx-xxx.jpg
func (s *StorageService) generateFileKey(router string, filename string) string {
	// ✅ 清理 router 前后的斜杠
	router = filepath.Clean(router)
	if router == "." {
		router = ""
	}
	// 移除前导斜杠
	router = trimLeadingSlash(router)

	// 生成 UUID
	id := uuid.New().String()

	// 获取文件扩展名
	ext := filepath.Ext(filename)

	// 按函数路径和日期分组
	now := time.Now()
	key := fmt.Sprintf("%s/%d/%02d/%02d/%s%s",
		router, now.Year(), now.Month(), now.Day(), id, ext)

	return key
}

// trimLeadingSlash 移除前导斜杠
func trimLeadingSlash(s string) string {
	for len(s) > 0 && s[0] == '/' {
		s = s[1:]
	}
	return s
}

// ListFilesByRouter 列举某个函数路径下的所有文件
func (s *StorageService) ListFilesByRouter(ctx context.Context, router string) ([]string, error) {
	bucket := s.getDefaultBucket()
	prefix := router + "/"

	objects, err := s.storage.ListObjects(ctx, bucket, prefix, true)
	if err != nil {
		logger.Errorf(ctx, "Failed to list objects: %v", err)
		return nil, fmt.Errorf("列举文件失败")
	}

	var files []string
	for _, obj := range objects {
		files = append(files, obj.Key)
	}

	return files, nil
}

// GetStorageStats 获取存储统计信息
func (s *StorageService) GetStorageStats(ctx context.Context, router string) (fileCount int, totalSize int64, err error) {
	bucket := s.getDefaultBucket()
	prefix := router + "/"

	objects, err := s.storage.ListObjects(ctx, bucket, prefix, true)
	if err != nil {
		logger.Errorf(ctx, "Failed to get storage stats: %v", err)
		return 0, 0, fmt.Errorf("获取统计信息失败")
	}

	fileCount = len(objects)
	for _, obj := range objects {
		totalSize += obj.Size
	}

	return fileCount, totalSize, nil
}

// DeleteFilesByRouter 删除某个函数路径下的所有文件
func (s *StorageService) DeleteFilesByRouter(ctx context.Context, router string) (int, error) {
	bucket := s.getDefaultBucket()
	prefix := router + "/"

	objects, err := s.storage.ListObjects(ctx, bucket, prefix, true)
	if err != nil {
		logger.Errorf(ctx, "Failed to list objects for deletion: %v", err)
		return 0, err
	}

	deletedCount := 0
	for _, obj := range objects {
		err := s.storage.DeleteObject(ctx, bucket, obj.Key)
		if err != nil {
			logger.Errorf(ctx, "Failed to delete file %s: %v", obj.Key, err)
			continue
		}
		deletedCount++
	}

	logger.Infof(ctx, "Deleted %d files under router: %s", deletedCount, router)
	return deletedCount, nil
}

// RecordUpload 记录上传
func (s *StorageService) RecordUpload(ctx context.Context, record *model.FileUpload) error {
	if s.fileRepo == nil {
		return nil // 审计功能未启用
	}
	return s.fileRepo.CreateUploadRecord(ctx, record)
}

// UpdateUploadStatus 更新上传状态
func (s *StorageService) UpdateUploadStatus(ctx context.Context, fileKey string, status string) error {
	if s.fileRepo == nil {
		return nil // 审计功能未启用
	}
	return s.fileRepo.UpdateUploadStatus(ctx, fileKey, status)
}

// RecordDownload 记录下载
func (s *StorageService) RecordDownload(ctx context.Context, record *model.FileDownload) error {
	// ✅ 检查是否启用了下载记录
	if !s.cfg.Audit.DownloadTracking.Enabled {
		return nil // 下载记录未启用
	}
	
	if s.fileRepo == nil {
		return nil // 审计功能未启用
	}
	
	return s.fileRepo.CreateDownloadRecord(ctx, record)
}
