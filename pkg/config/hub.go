package config

import (
	"fmt"
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
			"../../config/hub.yaml",
		}

		var err error
		for _, path := range configPaths {
			if err = loadYAMLConfig(path, cfg); err == nil {
				break
			}
		}

		if err != nil {
			// 配置文件不存在或加载失败，返回空配置
			fmt.Printf("Failed to load hub config from any path, using defaults: %v\n", err)
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
	Server HubServerConfig `mapstructure:"server"`
	DB     DBConfig        `mapstructure:"db"`
	OS     OSConfig        `mapstructure:"os"`
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
