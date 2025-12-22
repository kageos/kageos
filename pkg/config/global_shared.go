package config

import (
	"fmt"
	"sync"
)

var (
	globalSharedConfig *GlobalSharedConfig
	globalSharedOnce   sync.Once
	globalSharedMu     sync.RWMutex
)

// GetGlobalSharedConfig 获取全局共享配置
func GetGlobalSharedConfig() *GlobalSharedConfig {
	globalSharedOnce.Do(func() {
		cfg := &GlobalSharedConfig{}
		if err := loadYAMLConfig("global.yaml", cfg); err != nil {
			// 配置文件不存在或加载失败，返回空配置（使用默认值）
			fmt.Printf("Failed to load global config: %v\n", err)
			cfg = &GlobalSharedConfig{}
		}
		globalSharedMu.Lock()
		globalSharedConfig = cfg
		globalSharedMu.Unlock()
	})

	globalSharedMu.RLock()
	defer globalSharedMu.RUnlock()
	return globalSharedConfig
}

// GlobalSharedConfig 全局共享配置
type GlobalSharedConfig struct {
	Gateway        GatewayConfig                `mapstructure:"gateway"`
	Database       DBConfig                     `mapstructure:"database"`
	Nats           NatsConfig                   `mapstructure:"nats"`
	JWT            JWTConfig                    `mapstructure:"jwt"`
	ControlService ControlServiceClientConfig   `mapstructure:"control_service"`
}

// GatewayConfig 网关配置
type GatewayConfig struct {
	Host        string `mapstructure:"host"`         // 网关主机（如 localhost）
	Port        int    `mapstructure:"port"`         // 网关端口（如 9090）
	Domain      string `mapstructure:"domain"`       // 网关域名（如 api.example.com）
	BaseURL     string `mapstructure:"base_url"`     // 网关基础 URL（如 http://localhost:9090）
	InternalURL string `mapstructure:"internal_url"` // 内部服务访问地址（如 http://localhost:9090）
}

// GetBaseURL 获取网关基础 URL
// 优先级：base_url > domain > host+port > 默认值
func (g *GatewayConfig) GetBaseURL() string {
	if g.BaseURL != "" {
		return g.BaseURL
	}
	if g.Domain != "" {
		return fmt.Sprintf("https://%s", g.Domain)
	}
	if g.Host != "" && g.Port > 0 {
		return fmt.Sprintf("http://%s:%d", g.Host, g.Port)
	}
	return "http://localhost:9090" // 默认值
}

// GetInternalURL 获取内部服务访问地址
// 优先级：internal_url > base_url > host+port > 默认值
func (g *GatewayConfig) GetInternalURL() string {
	if g.InternalURL != "" {
		return g.InternalURL
	}
	return g.GetBaseURL()
}

