package config

// StorageConfigAdapter 存储配置适配器
// 实现 storage.Config 接口，根据配置的存储类型返回对应的配置
type StorageConfigAdapter struct {
	cfg *AppStorageConfig
}

// NewStorageConfigAdapter 创建存储配置适配器
func NewStorageConfigAdapter(cfg *AppStorageConfig) *StorageConfigAdapter {
	return &StorageConfigAdapter{cfg: cfg}
}

// GetEndpoint 获取 Endpoint
func (a *StorageConfigAdapter) GetEndpoint() string {
	switch a.cfg.Storage.Type {
	case "minio":
		return a.cfg.Storage.MinIO.Endpoint
	case "tencentcos":
		return a.cfg.Storage.TencentCOS.Endpoint
	case "aliyunoss":
		return a.cfg.Storage.AliyunOSS.Endpoint
	case "awss3":
		return a.cfg.Storage.AWSS3.Endpoint
	default:
		return a.cfg.Storage.MinIO.Endpoint
	}
}

// GetAccessKey 获取 AccessKey
func (a *StorageConfigAdapter) GetAccessKey() string {
	switch a.cfg.Storage.Type {
	case "minio":
		return a.cfg.Storage.MinIO.AccessKey
	case "tencentcos":
		return a.cfg.Storage.TencentCOS.SecretID
	case "aliyunoss":
		return a.cfg.Storage.AliyunOSS.AccessKeyID
	case "awss3":
		return a.cfg.Storage.AWSS3.AccessKey
	default:
		return a.cfg.Storage.MinIO.AccessKey
	}
}

// GetSecretKey 获取 SecretKey
func (a *StorageConfigAdapter) GetSecretKey() string {
	switch a.cfg.Storage.Type {
	case "minio":
		return a.cfg.Storage.MinIO.SecretKey
	case "tencentcos":
		return a.cfg.Storage.TencentCOS.SecretKey
	case "aliyunoss":
		return a.cfg.Storage.AliyunOSS.AccessKeySecret
	case "awss3":
		return a.cfg.Storage.AWSS3.SecretKey
	default:
		return a.cfg.Storage.MinIO.SecretKey
	}
}

// GetRegion 获取 Region
func (a *StorageConfigAdapter) GetRegion() string {
	switch a.cfg.Storage.Type {
	case "minio":
		return a.cfg.Storage.MinIO.Region
	case "tencentcos":
		return a.cfg.Storage.TencentCOS.Region
	case "aliyunoss":
		return a.cfg.Storage.AliyunOSS.Region
	case "awss3":
		return a.cfg.Storage.AWSS3.Region
	default:
		return a.cfg.Storage.MinIO.Region
	}
}

// GetUseSSL 获取 UseSSL
func (a *StorageConfigAdapter) GetUseSSL() bool {
	switch a.cfg.Storage.Type {
	case "minio":
		return a.cfg.Storage.MinIO.UseSSL
	default:
		// 云存储默认使用 HTTPS
		return true
	}
}

// GetDefaultBucket 获取默认 Bucket
func (a *StorageConfigAdapter) GetDefaultBucket() string {
	switch a.cfg.Storage.Type {
	case "minio":
		return a.cfg.Storage.MinIO.DefaultBucket
	case "tencentcos":
		return a.cfg.Storage.TencentCOS.DefaultBucket
	case "aliyunoss":
		return a.cfg.Storage.AliyunOSS.DefaultBucket
	case "awss3":
		return a.cfg.Storage.AWSS3.DefaultBucket
	default:
		return a.cfg.Storage.MinIO.DefaultBucket
	}
}

// GetCDNDomain 获取 CDN 域名
func (a *StorageConfigAdapter) GetCDNDomain() string {
	switch a.cfg.Storage.Type {
	case "minio":
		return a.cfg.Storage.MinIO.CDNDomain
	case "tencentcos":
		return a.cfg.Storage.TencentCOS.CDNDomain
	case "aliyunoss":
		return a.cfg.Storage.AliyunOSS.CDNDomain
	case "awss3":
		return a.cfg.Storage.AWSS3.CDNDomain
	default:
		return a.cfg.Storage.MinIO.CDNDomain
	}
}

