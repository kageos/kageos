package config

import (
	"fmt"
	"sync"
)

var (
	appGatewayConfig *AppGatewayConfig
	appGatewayOnce   sync.Once
	appGatewayMu     sync.RWMutex
)

// GetAppGatewayConfig 获取 app-gateway 配置（与其它服务保持一致的访问方式）
func GetAppGatewayConfig() *AppGatewayConfig {
	appGatewayOnce.Do(func() {
		cfg := &AppGatewayConfig{}
		if err := loadYAMLConfig("app-gateway.yaml", cfg); err != nil {
			// 配置文件不存在或加载失败，返回空配置
			fmt.Printf("Failed to load app-gateway config: %v\n", err)
			cfg = &AppGatewayConfig{}
		}
		appGatewayMu.Lock()
		appGatewayConfig = cfg
		appGatewayMu.Unlock()
	})

	appGatewayMu.RLock()
	defer appGatewayMu.RUnlock()
	return appGatewayConfig
}

// AppGatewayConfig App Gateway 配置（精简版）
type AppGatewayConfig struct {
	Server   ServerConfig         `mapstructure:"server"`
	Nats     NatsConfig           `mapstructure:"nats"`
	Timeouts GatewayTimeoutConfig `mapstructure:"timeouts"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port     int    `mapstructure:"port"`
	LogLevel string `mapstructure:"log_level"`
	Debug    bool   `mapstructure:"debug"`
}

// GatewayTimeoutConfig 网关超时配置
type GatewayTimeoutConfig struct {
	AppServerRequest int `mapstructure:"app_server_request"` // app-server 请求超时时间（秒）
	AppRequest       int `mapstructure:"app_request"`        // 应用请求超时时间（秒）
	NatsRequest      int `mapstructure:"nats_request"`       // NATS 请求超时时间（秒）
}
