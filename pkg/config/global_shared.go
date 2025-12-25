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
	Gateway        GatewayConfig              `mapstructure:"gateway"`
	Nats           NatsConfig                 `mapstructure:"nats"`
	JWT            JWTConfig                  `mapstructure:"jwt"`
	ControlService ControlServiceClientConfig `mapstructure:"control_service"`
	SDK            SDKConfig                  `mapstructure:"sdk"`
	// 注意：数据库配置不在全局配置中，每个服务可以单独配置自己的数据库
}

// GatewayConfig 网关配置
// 注意：服务运行在裸机上，使用 localhost 访问
type GatewayConfig struct {
	Host        string `mapstructure:"host"`         // 网关主机（裸机服务访问，如 localhost）
	Port        int    `mapstructure:"port"`         // 网关端口（如 9090）
	Domain      string `mapstructure:"domain"`       // 网关域名（生产环境使用，如 api.example.com）
	BaseURL     string `mapstructure:"base_url"`     // 网关基础 URL（裸机服务访问，如 http://localhost:9090）
	InternalURL string `mapstructure:"internal_url"` // 内部服务访问地址（裸机服务之间访问，如 http://localhost:9090）
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
	return "http://localhost:9090" // 默认值（裸机服务访问）
}

// GetInternalURL 获取内部服务访问地址
// 优先级：internal_url > base_url > host+port > 默认值
func (g *GatewayConfig) GetInternalURL() string {
	if g.InternalURL != "" {
		return g.InternalURL
	}
	return g.GetBaseURL()
}

// GetGatewayURL 获取网关地址（全局函数，从全局配置读取）
// 用于注入到 SDK 容器中等场景
func GetGatewayURL() string {
	global := GetGlobalSharedConfig()
	return global.Gateway.GetBaseURL()
}

// SDKConfig SDK 配置（专门用于 runtime 构建 SDK app 时注入到容器中）
// 注意：SDK app 运行在容器中，需要使用 host.containers.internal 访问宿主机服务
// 这些配置会在构建时注入为环境变量：
//   - nats_url -> NATS_URL 环境变量
//   - gateway_url -> GATEWAY_URL 环境变量
//   - env_vars 中的键值对 -> 对应的环境变量
type SDKConfig struct {
	NatsURL    string            `mapstructure:"nats_url"`    // NATS 地址（容器内访问，如 nats://host.containers.internal:4222），注入为 NATS_URL 环境变量
	GatewayURL string            `mapstructure:"gateway_url"` // 网关地址（容器内访问，如 http://host.containers.internal:9090），注入为 GATEWAY_URL 环境变量
	EnvVars    map[string]string `mapstructure:"env_vars"`    // 额外的环境变量映射（键值对），会直接注入到容器中
}

// GetNatsURL 获取 SDK NATS 地址（容器内访问）
func (s *SDKConfig) GetNatsURL() string {
	if s.NatsURL != "" {
		return s.NatsURL
	}
	return "nats://host.containers.internal:4222" // 默认值（容器内访问宿主机 NATS）
}

// GetGatewayURL 获取 SDK 网关地址（容器内访问）
func (s *SDKConfig) GetGatewayURL() string {
	if s.GatewayURL != "" {
		return s.GatewayURL
	}
	return "http://host.containers.internal:9090" // 默认值（容器内访问宿主机网关）
}

// GetEnvVars 获取所有环境变量（包括固定字段和 env_vars 中的）
// 返回 map[string]string，键为环境变量名，值为环境变量值
func (s *SDKConfig) GetEnvVars() map[string]string {
	envVars := make(map[string]string)

	// 注入固定字段（向后兼容）
	if s.NatsURL != "" {
		envVars["NATS_URL"] = s.NatsURL
	} else {
		envVars["NATS_URL"] = "nats://host.containers.internal:4222" // 默认值
	}

	if s.GatewayURL != "" {
		envVars["GATEWAY_URL"] = s.GatewayURL
	} else {
		envVars["GATEWAY_URL"] = "http://host.containers.internal:9090" // 默认值
	}

	// 注入 env_vars 中的额外环境变量（会覆盖固定字段，如果键名相同）
	if s.EnvVars != nil {
		for k, v := range s.EnvVars {
			envVars[k] = v
		}
	}

	return envVars
}

// GetSDKConfig 获取 SDK 配置（全局函数）
// 用于 runtime 构建 SDK app 时注入到容器中
func GetSDKConfig() SDKConfig {
	global := GetGlobalSharedConfig()
	return global.SDK
}
