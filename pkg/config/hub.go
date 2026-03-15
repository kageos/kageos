package config

import (
	"fmt"
	"strings"
	"sync"
)

var (
	hubConfig *HubConfig
	hubOnce   sync.Once
	hubMu     sync.RWMutex
)

// GetHubConfig 获取 Hub 配置
func GetHubConfig() *HubConfig {
	hubOnce.Do(func() {
		cfg := &HubConfig{}
		// 尝试从多个路径加载配置文件
		configPaths := []string{
			"configs/hub.yaml",
			"enterprise_impl/hub/backend/config/hub.yaml",
			"../../config/hub.yaml",
		}

		var err error
		for _, path := range configPaths {
			if err = loadYAMLConfig(path, cfg); err == nil {
				break
			}
		}

		if err != nil {
			// 配置文件不存在或加载失败，返回空配置；copy_url 会回退为请求 Host（如 localhost:9090）
			fmt.Printf("Failed to load hub config from any path, using defaults: %v (set APP_ENV=dev and run from project root or ensure configs/dev/hub.yaml exists)\n", err)
			cfg = &HubConfig{}
		}
		hubMu.Lock()
		hubConfig = cfg
		hubMu.Unlock()
	})

	hubMu.RLock()
	defer hubMu.RUnlock()
	return hubConfig
}

// HubConfig Hub 配置
type HubConfig struct {
	Server     HubServerConfig `mapstructure:"server"`
	DB         DBConfig        `mapstructure:"db"`
	OS         OSConfig        `mapstructure:"os"`
	PublicHost string          `mapstructure:"public_host"` // 主站（用户访问入口）的 host:port，用于生成 copy_url（hub://主站host/路径@版本）；不配则回退请求头
}

// HubServerConfig Hub 服务器配置
type HubServerConfig struct {
	Port     int    `mapstructure:"port"`
	LogLevel string `mapstructure:"log_level"`
	Debug    bool   `mapstructure:"debug"`
}

// OSConfig OS 平台配置
type OSConfig struct {
	BaseURL string `mapstructure:"base_url"` // OS 平台基础 URL
}

// GetPort 获取端口
func (c *HubConfig) GetPort() int {
	if c.Server.Port == 0 {
		return 9094 // 默认端口
	}
	return c.Server.Port
}

// GetLogLevel 获取日志级别
func (c *HubConfig) GetLogLevel() string {
	if c.Server.LogLevel == "" {
		return "info"
	}
	return c.Server.LogLevel
}

// IsDebug 是否调试模式
func (c *HubConfig) IsDebug() bool {
	return c.Server.Debug
}

// GetPublicHost 获取主站 host:port（用于 copy_url），返回配置的 public_host，可能为空
func (c *HubConfig) GetPublicHost() string {
	return strings.TrimSpace(c.PublicHost)
}
