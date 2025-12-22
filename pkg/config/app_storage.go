package config

import (
	"fmt"
	"sync"
)

// AppStorageConfig app-storage 配置
type AppStorageConfig struct {
	Server struct {
		Port     int    `mapstructure:"port"`
		LogLevel string `mapstructure:"log_level"`
		Debug    bool   `mapstructure:"debug"`
	} `mapstructure:"server"`

	JWT struct {
		Secret string `mapstructure:"secret"`
		Issuer string `mapstructure:"issuer"`
	} `mapstructure:"jwt"`

	Audit struct {
		UploadTracking struct {
			Enabled bool `mapstructure:"enabled"`
		} `mapstructure:"upload_tracking"`
		DownloadTracking struct {
			Enabled       bool `mapstructure:"enabled"`
			RetentionDays int  `mapstructure:"retention_days"`
		} `mapstructure:"download_tracking"`
	} `mapstructure:"audit"`

	Storage struct {
		Type string `mapstructure:"type"` // 存储类型：minio | tencentcos | aliyunoss | awss3 | local

		MinIO struct {
			Endpoint       string `mapstructure:"endpoint"`        // 浏览器上传用的 endpoint（宿主机访问）
			ServerEndpoint string `mapstructure:"server_endpoint"` // ✨ 服务端上传用的 endpoint（容器内访问）
			AccessKey      string `mapstructure:"access_key"`
			SecretKey      string `mapstructure:"secret_key"`
			UseSSL         bool   `mapstructure:"use_ssl"`
			Region         string `mapstructure:"region"`
			DefaultBucket  string `mapstructure:"default_bucket"`
			CDNDomain      string `mapstructure:"cdn_domain"` // ✨ CDN 域名（可选，用于加速访问）
		} `mapstructure:"minio"`

		TencentCOS struct {
			Endpoint      string `mapstructure:"endpoint"`
			SecretID      string `mapstructure:"secret_id"`
			SecretKey     string `mapstructure:"secret_key"`
			Region        string `mapstructure:"region"`
			DefaultBucket string `mapstructure:"default_bucket"`
			CDNDomain     string `mapstructure:"cdn_domain"` // ✨ CDN 域名（可选）
		} `mapstructure:"tencentcos"`

		AliyunOSS struct {
			Endpoint        string `mapstructure:"endpoint"`
			AccessKeyID      string `mapstructure:"access_key_id"`
			AccessKeySecret  string `mapstructure:"access_key_secret"`
			Region           string `mapstructure:"region"`
			DefaultBucket    string `mapstructure:"default_bucket"`
			CDNDomain        string `mapstructure:"cdn_domain"` // ✨ CDN 域名（可选）
		} `mapstructure:"aliyunoss"`

		AWSS3 struct {
			Endpoint      string `mapstructure:"endpoint"`
			AccessKey     string `mapstructure:"access_key"`
			SecretKey     string `mapstructure:"secret_key"`
			Region        string `mapstructure:"region"`
			DefaultBucket string `mapstructure:"default_bucket"`
			CDNDomain     string `mapstructure:"cdn_domain"` // ✨ CDN 域名（可选）
		} `mapstructure:"awss3"`

		Upload struct {
			MaxSize      int64    `mapstructure:"max_size"`
			TokenExpire  int      `mapstructure:"token_expire"`
			AllowedTypes []string `mapstructure:"allowed_types"`
		} `mapstructure:"upload"`

		Deduplication struct {
			Enabled       bool   `mapstructure:"enabled"`
			HashAlgorithm string `mapstructure:"hash_algorithm"`
		} `mapstructure:"deduplication"`

		Cache struct {
			Enabled bool `mapstructure:"enabled"`
			MaxAge  int  `mapstructure:"max_age"`
		} `mapstructure:"cache"`
	} `mapstructure:"storage"`

	DB DBConfig `mapstructure:"db"`
}

var (
	appStorageConfig     *AppStorageConfig
	appStorageConfigOnce sync.Once
)

// GetAppStorageConfig 获取 app-storage 配置（单例）
func GetAppStorageConfig() *AppStorageConfig {
	appStorageConfigOnce.Do(func() {
		appStorageConfig = loadAppStorageConfig()
	})
	return appStorageConfig
}

// loadAppStorageConfig 加载 app-storage 配置
func loadAppStorageConfig() *AppStorageConfig {
	cfg := &AppStorageConfig{}
	if err := loadYAMLConfig("app-storage.yaml", cfg); err != nil {
		// 配置文件不存在或加载失败，返回空配置
		fmt.Printf("Failed to load app-storage config: %v\n", err)
		cfg = &AppStorageConfig{}
	}

	// 获取全局共享配置
	global := GetGlobalSharedConfig()

	// 合并数据库配置（如果服务配置了，使用服务配置；否则使用全局配置）
	if cfg.DB.Host == "" && cfg.DB.Type == "" {
		// 服务完全没有配置数据库，使用全局配置
		cfg.DB = global.Database
	} else {
		// 服务配置了部分字段，合并未配置的字段
		cfg.DB = mergeDBConfig(global.Database, cfg.DB)
	}

	// 合并 JWT 配置
	if cfg.JWT.Secret == "" {
		// 服务没有配置 JWT，使用全局配置
		cfg.JWT.Secret = global.JWT.Secret
		cfg.JWT.Issuer = global.JWT.Issuer
	} else {
		// 服务配置了部分字段，合并未配置的字段
		if cfg.JWT.Issuer == "" {
			cfg.JWT.Issuer = global.JWT.Issuer
		}
	}

	return cfg
}

// GetPort 获取端口
func (c *AppStorageConfig) GetPort() int {
	if c.Server.Port == 0 {
		return 8083
	}
	return c.Server.Port
}

// GetLogLevel 获取日志级别
func (c *AppStorageConfig) GetLogLevel() string {
	if c.Server.LogLevel == "" {
		return "info"
	}
	return c.Server.LogLevel
}

// IsDebug 是否调试模式
func (c *AppStorageConfig) IsDebug() bool {
	return c.Server.Debug
}

