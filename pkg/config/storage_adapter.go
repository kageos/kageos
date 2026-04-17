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
	return a.cfg.Storage.MinIO.Endpoint
}

// GetAccessKey 获取 AccessKey
func (a *StorageConfigAdapter) GetAccessKey() string {
	return a.cfg.Storage.MinIO.AccessKey
}

// GetSecretKey 获取 SecretKey
func (a *StorageConfigAdapter) GetSecretKey() string {
	return a.cfg.Storage.MinIO.SecretKey
}

// GetRegion 获取 Region
func (a *StorageConfigAdapter) GetRegion() string {
	return a.cfg.Storage.MinIO.Region
}

// GetUseSSL 获取 UseSSL
func (a *StorageConfigAdapter) GetUseSSL() bool {
	return a.cfg.Storage.MinIO.UseSSL
}

// GetDefaultBucket 获取默认 Bucket
func (a *StorageConfigAdapter) GetDefaultBucket() string {
	return a.cfg.Storage.MinIO.DefaultBucket
}

// GetCDNDomain 获取 CDN 域名
func (a *StorageConfigAdapter) GetCDNDomain() string {
	return a.cfg.Storage.MinIO.CDNDomain
}

// GetServerEndpoint 获取服务端 endpoint（仅用于 MinIO，容器内访问）
func (a *StorageConfigAdapter) GetServerEndpoint() string {
	return a.cfg.Storage.MinIO.ServerEndpoint
}
