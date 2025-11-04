package config

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// AppStorageConfig app-storage 配置
type AppStorageConfig struct {
	Server struct {
		Port     int    `yaml:"port"`
		LogLevel string `yaml:"log_level"`
		Debug    bool   `yaml:"debug"`
	} `yaml:"server"`

	JWT struct {
		Secret string `yaml:"secret"`
		Issuer string `yaml:"issuer"`
	} `yaml:"jwt"`

	Audit struct {
		UploadTracking struct {
			Enabled bool `yaml:"enabled"`
		} `yaml:"upload_tracking"`
		DownloadTracking struct {
			Enabled       bool `yaml:"enabled"`
			RetentionDays int  `yaml:"retention_days"`
		} `yaml:"download_tracking"`
	} `yaml:"audit"`

	Storage struct {
		Type string `yaml:"type"` // 存储类型：minio | tencentcos | aliyunoss | awss3 | local

		MinIO struct {
			Endpoint      string `yaml:"endpoint"`
			AccessKey     string `yaml:"access_key"`
			SecretKey     string `yaml:"secret_key"`
			UseSSL        bool   `yaml:"use_ssl"`
			Region        string `yaml:"region"`
			DefaultBucket string `yaml:"default_bucket"`
			CDNDomain     string `yaml:"cdn_domain"` // ✨ CDN 域名（可选，用于加速访问）
		} `yaml:"minio"`

		TencentCOS struct {
			Endpoint      string `yaml:"endpoint"`
			SecretID      string `yaml:"secret_id"`
			SecretKey     string `yaml:"secret_key"`
			Region        string `yaml:"region"`
			DefaultBucket string `yaml:"default_bucket"`
			CDNDomain     string `yaml:"cdn_domain"` // ✨ CDN 域名（可选）
		} `yaml:"tencentcos"`

		AliyunOSS struct {
			Endpoint        string `yaml:"endpoint"`
			AccessKeyID     string `yaml:"access_key_id"`
			AccessKeySecret string `yaml:"access_key_secret"`
			Region          string `yaml:"region"`
			DefaultBucket   string `yaml:"default_bucket"`
			CDNDomain       string `yaml:"cdn_domain"` // ✨ CDN 域名（可选）
		} `yaml:"aliyunoss"`

		AWSS3 struct {
			Endpoint      string `yaml:"endpoint"`
			AccessKey     string `yaml:"access_key"`
			SecretKey     string `yaml:"secret_key"`
			Region        string `yaml:"region"`
			DefaultBucket string `yaml:"default_bucket"`
			CDNDomain     string `yaml:"cdn_domain"` // ✨ CDN 域名（可选）
		} `yaml:"awss3"`

		Upload struct {
			MaxSize      int64    `yaml:"max_size"`
			TokenExpire  int      `yaml:"token_expire"`
			AllowedTypes []string `yaml:"allowed_types"`
		} `yaml:"upload"`

		Deduplication struct {
			Enabled       bool   `yaml:"enabled"`
			HashAlgorithm string `yaml:"hash_algorithm"`
		} `yaml:"deduplication"`

		Cache struct {
			Enabled bool `yaml:"enabled"`
			MaxAge  int  `yaml:"max_age"`
		} `yaml:"cache"`
	} `yaml:"storage"`

	DB struct {
		Type         string `yaml:"type"`
		Host         string `yaml:"host"`
		Port         int    `yaml:"port"`
		User         string `yaml:"user"`
		Password     string `yaml:"password"`
		Name         string `yaml:"name"`
		MaxIdleConns int    `yaml:"max_idle_conns"`
		MaxOpenConns int    `yaml:"max_open_conns"`
		LogLevel     string `yaml:"log_level"`
	} `yaml:"db"`
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
	configFile := "configs/app-storage.yaml"

	data, err := os.ReadFile(configFile)
	if err != nil {
		panic(fmt.Sprintf("Failed to read app-storage config file: %v", err))
	}

	var cfg AppStorageConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		panic(fmt.Sprintf("Failed to parse app-storage config: %v", err))
	}

	return &cfg
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

