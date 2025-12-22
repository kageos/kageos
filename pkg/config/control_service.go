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
	Server  ControlServiceServerConfig `mapstructure:"server"`
	Nats    ControlServiceNatsConfig   `mapstructure:"nats"` // 已废弃，改为从全局配置读取
	License LicenseConfig              `mapstructure:"license"`
}

// ControlServiceServerConfig control-service 服务器配置
type ControlServiceServerConfig struct {
	Port     int    `mapstructure:"port"`
	LogLevel string `mapstructure:"log_level"`
	Debug    bool   `mapstructure:"debug"`
}

// LicenseConfig License 配置
type LicenseConfig struct {
	Path            string `mapstructure:"path"`             // License 文件路径
	EncryptionKey   string `mapstructure:"encryption_key"`   // License 传输加密密钥（32字节字符串，用于AES-256-GCM加密传输）
	PublishInterval int    `mapstructure:"publish_interval"` // 过期检查间隔（秒，默认300秒=5分钟，用于定期检查License是否过期）
}

// ControlServiceNatsConfig Control Service NATS 配置
type ControlServiceNatsConfig struct {
	URL string `mapstructure:"url"` // NATS 服务器地址
}

// ControlServiceClientConfig Control Service 客户端配置
// 用于各服务连接到 Control Service 获取 License
type ControlServiceClientConfig struct {
	Enabled       bool   `mapstructure:"enabled"`        // 是否启用 Control Service 客户端（默认：true）
	NatsURL       string `mapstructure:"nats_url"`       // Control Service 的 NATS 地址（如果为空，使用主 NATS 配置）
	EncryptionKey string `mapstructure:"encryption_key"` // License 传输加密密钥（32字节字符串，必须与 Control Service 相同）
	KeyPath       string `mapstructure:"key_path"`       // 本地密钥文件路径（可选，默认：~/.ai-agent-os/license.key）
}

// IsEnabled 是否启用 Control Service 客户端
func (c *ControlServiceClientConfig) IsEnabled() bool {
	// 如果显式设置为 false，则禁用
	if !c.Enabled {
		return false
	}
	// 如果配置了加密密钥，则认为启用（即使 enabled 字段未设置）
	if c.EncryptionKey != "" {
		return true
	}
	// 默认启用（向后兼容）
	return true
}

// GetNatsURL 获取 NATS URL
func (c *ControlServiceClientConfig) GetNatsURL() string {
	if c.NatsURL != "" {
		return c.NatsURL
	}
	return "" // 返回空字符串，表示使用主 NATS 配置
}

// GetEncryptionKey 获取加密密钥（用于AES-256-GCM加密传输）
func (c *ControlServiceClientConfig) GetEncryptionKey() []byte {
	key := c.EncryptionKey
	if len(key) != 32 {
		// 如果密钥长度不对，返回默认密钥（生产环境应该报错）
		return []byte("ai-agent-os-license-key-32bytes!!")
	}
	return []byte(key)
}

// GetKeyPath 获取密钥文件路径
func (c *ControlServiceClientConfig) GetKeyPath() string {
	if c.KeyPath != "" {
		return c.KeyPath
	}
	return "" // 返回空字符串，使用默认路径
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

// GetNatsURL 获取 NATS URL（从全局配置读取）
func (c *ControlServiceConfig) GetNatsURL() string {
	// 优先使用全局配置的 NATS
	global := GetGlobalSharedConfig()
	if global.Nats.URL != "" {
		return global.Nats.URL
	}
	// 如果全局配置为空，使用服务配置（向后兼容）
	if c.Nats.URL != "" {
		return c.Nats.URL
	}
	// 默认值
	return "nats://127.0.0.1:4222"
}

// GetLicensePath 获取 License 文件路径
func (c *ControlServiceConfig) GetLicensePath() string {
	if c.License.Path == "" {
		return "./license.json"
	}
	return c.License.Path
}

// GetEncryptionKey 获取加密密钥（用于AES-256-GCM加密传输）
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
