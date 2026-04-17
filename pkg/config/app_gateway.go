package config

import (
	"fmt"
	"sync"
)

var (
	apiGatewayConfig *APIGatewayConfig
	apiGatewayOnce   sync.Once
	apiGatewayMu     sync.RWMutex
)

// GetAPIGatewayConfig 获取 api-gateway 配置（与其它服务保持一致的访问方式）
func GetAPIGatewayConfig() *APIGatewayConfig {
	apiGatewayOnce.Do(func() {
		cfg := &APIGatewayConfig{}
		if err := loadYAMLConfig("api-gateway.yaml", cfg); err != nil {
			// 配置文件不存在或加载失败，返回空配置
			fmt.Printf("Failed to load api-gateway config: %v\n", err)
			cfg = &APIGatewayConfig{}
		}
		apiGatewayMu.Lock()
		apiGatewayConfig = cfg
		apiGatewayMu.Unlock()
	})

	apiGatewayMu.RLock()
	defer apiGatewayMu.RUnlock()
	return apiGatewayConfig
}

// APIGatewayConfig API Gateway 配置
type APIGatewayConfig struct {
	Server   GatewayServerConfig  `mapstructure:"server"`
	Routes   []RouteConfig        `mapstructure:"routes"` // ✨ 路由配置
	Timeouts GatewayTimeoutConfig `mapstructure:"timeouts"`
}

// GatewayServerConfig 网关服务器配置
type GatewayServerConfig struct {
	Port                     int    `mapstructure:"port"`
	ListenHost               string `mapstructure:"listen_host"`
	LogLevel                 string `mapstructure:"log_level"`
	Debug                    bool   `mapstructure:"debug"`
	EnablePprof              *bool  `mapstructure:"enable_pprof"`
	AllowNATSDegradedStartup bool   `mapstructure:"allow_nats_degraded_startup"`
}

// RouteConfig 路由配置
type RouteConfig struct {
	Path        string          `mapstructure:"path"`         // 路径前缀（如 /api/v1/storage）
	Targets     []BackendConfig `mapstructure:"targets"`      // 后端服务列表（至少一个；多个 targets 当前仅回退使用第一个）
	Timeout     int             `mapstructure:"timeout"`      // 超时时间（秒，0 表示使用默认值）
	RewritePath string          `mapstructure:"rewrite_path"` // 路径重写（可选，如 /api/v1，将去掉路由前缀后替换为此路径）
	ServiceName string          `mapstructure:"service_name"` // 服务名称（必须，用于 Swagger 文档聚合，必须显式配置）
}

// BackendConfig 后端服务配置
type BackendConfig struct {
	URL string `mapstructure:"url"` // 后端服务地址（如 http://localhost:9091）
}

// GatewayTimeoutConfig 网关超时配置
type GatewayTimeoutConfig struct {
	Default int `mapstructure:"default"` // 默认超时时间（秒）
}

// GetPort 获取端口
func (c *APIGatewayConfig) GetPort() int {
	if c.Server.Port == 0 {
		return 9090
	}
	return c.Server.Port
}

// GetListenHost 获取监听地址
func (c *APIGatewayConfig) GetListenHost() string {
	if c == nil {
		return normalizeListenHost("")
	}
	return normalizeListenHost(c.Server.ListenHost)
}

// GetLogLevel 获取日志级别
func (c *APIGatewayConfig) GetLogLevel() string {
	if c.Server.LogLevel == "" {
		return "info"
	}
	return c.Server.LogLevel
}

// IsDebug 是否调试模式
func (c *APIGatewayConfig) IsDebug() bool {
	return c.Server.Debug
}

// IsPprofEnabled 是否启用 pprof。
// 默认为 true，保持开发环境向后兼容；生产模板应显式关闭。
func (c *APIGatewayConfig) IsPprofEnabled() bool {
	if c == nil {
		return true
	}
	return boolConfigValue(c.Server.EnablePprof, true)
}

// AllowNATSDegradedStartup 是否允许在 NATS 初始化失败时继续启动 HTTP 网关。
// 默认值为 false，避免 token 失效链路静默漂移。
func (c *APIGatewayConfig) AllowNATSDegradedStartup() bool {
	return c.Server.AllowNATSDegradedStartup
}
