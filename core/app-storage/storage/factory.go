package storage

import (
	"fmt"
	"strings"
)

// StorageType 存储类型
type StorageType string

const (
	StorageTypeMinIO      StorageType = "minio"       // MinIO
	StorageTypeTencentCOS StorageType = "tencentcos"  // 腾讯云 COS
	StorageTypeAliyunOSS  StorageType = "aliyunoss"   // 阿里云 OSS
	StorageTypeAWSS3      StorageType = "awss3"       // AWS S3
	StorageTypeLocal      StorageType = "local"       // 本地存储（开发环境）
)

// Factory 存储工厂
type Factory struct{}

// NewFactory 创建存储工厂
func NewFactory() *Factory {
	return &Factory{}
}

// CreateStorage 根据类型创建存储实例
func (f *Factory) CreateStorage(storageType string, cfg Config) (Storage, error) {
	switch StorageType(strings.ToLower(storageType)) {
	case StorageTypeMinIO:
		return NewMinIOStorage(cfg)
	
	case StorageTypeTencentCOS:
		// TODO: 实现腾讯云 COS
		return nil, fmt.Errorf("腾讯云 COS 存储尚未实现")
	
	case StorageTypeAliyunOSS:
		// TODO: 实现阿里云 OSS
		return nil, fmt.Errorf("阿里云 OSS 存储尚未实现")
	
	case StorageTypeAWSS3:
		// TODO: 实现 AWS S3（可以使用 MinIO SDK，S3 兼容）
		return nil, fmt.Errorf("AWS S3 存储尚未实现")
	
	case StorageTypeLocal:
		// TODO: 实现本地文件系统存储
		return nil, fmt.Errorf("本地存储尚未实现")
	
	default:
		return nil, fmt.Errorf("不支持的存储类型: %s", storageType)
	}
}

