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
	Nats     NatsConfig           `mapstructure:"nats"`
	Timeouts GatewayTimeoutConfig `mapstructure:"timeouts"`
}

// GatewayServerConfig 网关服务器配置
type GatewayServerConfig struct {
	Port     int    `mapstructure:"port"`
	LogLevel string `mapstructure:"log_level"`
	Debug    bool   `mapstructure:"debug"`
}

// RouteConfig 路由配置
type RouteConfig struct {
	Path        string             `mapstructure:"path"`         // 路径前缀（如 /api/v1/storage）
	Targets     []BackendConfig    `mapstructure:"targets"`      // 后端服务列表（至少一个）
	LoadBalance *LoadBalanceConfig `mapstructure:"load_balance"` // 负载均衡配置（多个 targets 时生效）
	Timeout     int                `mapstructure:"timeout"`      // 超时时间（秒，0 表示使用默认值）
	RewritePath string             `mapstructure:"rewrite_path"` // 路径重写（可选，如 /api/v1，将去掉路由前缀后替换为此路径）
	ServiceName string             `mapstructure:"service_name"` // 服务名称（必须，用于 Swagger 文档聚合，必须显式配置）
}

// BackendConfig 后端服务配置
type BackendConfig struct {
	URL    string `mapstructure:"url"`    // 后端服务地址（如 http://localhost:9091）
	Weight int    `mapstructure:"weight"` // 权重（用于加权负载均衡，默认 1，仅当 load_balance.strategy=weighted 时生效）
}

// LoadBalanceConfig 负载均衡配置
type LoadBalanceConfig struct {
	Strategy    string             `mapstructure:"strategy"`     // 策略：round_robin, weighted, least_connections, ip_hash（默认 round_robin）
	HealthCheck *HealthCheckConfig `mapstructure:"health_check"` // 健康检查配置（可选）
}

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
	Enabled  bool   `mapstructure:"enabled"`  // 是否启用健康检查（默认 false）
	Path     string `mapstructure:"path"`     // 健康检查路径（默认 /health）
	Interval int    `mapstructure:"interval"` // 检查间隔（秒，默认 10）
	Timeout  int    `mapstructure:"timeout"`  // 超时时间（秒，默认 3）
	Retries  int    `mapstructure:"retries"`  // 失败重试次数（默认 2）
}

// GatewayTimeoutConfig 网关超时配置
type GatewayTimeoutConfig struct {
	Default          int `mapstructure:"default"`            // 默认超时时间（秒）
	AppServerRequest int `mapstructure:"app_server_request"` // app-server 请求超时时间（秒）
	AppRequest       int `mapstructure:"app_request"`        // 应用请求超时时间（秒）
	NatsRequest      int `mapstructure:"nats_request"`       // NATS 请求超时时间（秒）
}

// GetPort 获取端口
func (c *APIGatewayConfig) GetPort() int {
	if c.Server.Port == 0 {
		return 9090
	}
	return c.Server.Port
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
