package config

import (
	"fmt"
	"sync"
)

var (
	controlServiceConfig *ControlServiceConfig
	controlServiceOnce   sync.Once
	controlServiceMu     sync.RWMutex
)

// GetControlServiceConfig 获取 control-service 配置
func GetControlServiceConfig() *ControlServiceConfig {
	controlServiceOnce.Do(func() {
		cfg := &ControlServiceConfig{}
		if err := loadYAMLConfig("control-service.yaml", cfg); err != nil {
			// 配置文件不存在或加载失败，返回空配置
			fmt.Printf("Failed to load control-service config: %v\n", err)
			cfg = &ControlServiceConfig{}
		}
		controlServiceMu.Lock()
		controlServiceConfig = cfg
		controlServiceMu.Unlock()
	})

	controlServiceMu.RLock()
	defer controlServiceMu.RUnlock()
	return controlServiceConfig
}

// ControlServiceConfig control-service 配置
type ControlServiceConfig struct {
	Server      ControlServiceServerConfig `mapstructure:"server"`
	Nats        NatsConfig                 `mapstructure:"nats"`
	License     LicenseConfig              `mapstructure:"license"`
}

// ControlServiceServerConfig control-service 服务器配置
type ControlServiceServerConfig struct {
	Port     int    `mapstructure:"port"`
	LogLevel string  `mapstructure:"log_level"`
	Debug   bool    `mapstructure:"debug"`
}

// LicenseConfig License 配置
type LicenseConfig struct {
	Path          string `mapstructure:"path"`           // License 文件路径
	EncryptionKey string `mapstructure:"encryption_key"` // License 加密密钥（32字节）
	PublishInterval int  `mapstructure:"publish_interval"` // 发布间隔（秒，默认300秒）
}

// NatsConfig NATS 配置
type NatsConfig struct {
	URL string `mapstructure:"url"` // NATS 服务器地址
}

// GetPort 获取端口
func (c *ControlServiceConfig) GetPort() int {
	if c.Server.Port == 0 {
		return 9096 // 默认端口（9090-9095 已被占用）
	}
	return c.Server.Port
}

// GetLogLevel 获取日志级别
func (c *ControlServiceConfig) GetLogLevel() string {
	if c.Server.LogLevel == "" {
		return "info"
	}
	return c.Server.LogLevel
}

// IsDebug 是否调试模式
func (c *ControlServiceConfig) IsDebug() bool {
	return c.Server.Debug
}

// GetNatsURL 获取 NATS URL
func (c *ControlServiceConfig) GetNatsURL() string {
	if c.Nats.URL == "" {
		return "nats://127.0.0.1:4222"
	}
	return c.Nats.URL
}

// GetLicensePath 获取 License 文件路径
func (c *ControlServiceConfig) GetLicensePath() string {
	if c.License.Path == "" {
		return "./license.json"
	}
	return c.License.Path
}

// GetEncryptionKey 获取加密密钥
func (c *ControlServiceConfig) GetEncryptionKey() []byte {
	key := c.License.EncryptionKey
	if len(key) != 32 {
		// 如果密钥长度不对，返回默认密钥（生产环境应该报错）
		return []byte("ai-agent-os-license-key-32bytes!!")
	}
	return []byte(key)
}

// GetPublishInterval 获取发布间隔（秒）
func (c *ControlServiceConfig) GetPublishInterval() int {
	if c.License.PublishInterval == 0 {
		return 300 // 默认5分钟
	}
	return c.License.PublishInterval
}
