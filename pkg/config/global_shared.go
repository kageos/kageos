package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
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
	Site    SiteConfig    `mapstructure:"site"`
	Gateway GatewayConfig `mapstructure:"gateway"`
	Nats    NatsConfig    `mapstructure:"nats"`
	JWT     JWTConfig     `mapstructure:"jwt"`
	SDK     SDKConfig     `mapstructure:"sdk"`
	// 注意：数据库配置不在全局配置中，每个服务可以单独配置自己的数据库
}

// SiteConfig 是用户浏览器可访问的主站配置。
// 它用于通知卡片、分享链接、邮件链接等“给人点击”的绝对 URL。
type SiteConfig struct {
	BaseURL string `mapstructure:"base_url"`
}

func (g *GlobalSharedConfig) GetPublicSiteBaseURL() string {
	if g != nil {
		if v := normalizePublicBaseURL(g.Site.BaseURL); v != "" {
			return v
		}
	}
	if v := normalizePublicBaseURL(os.Getenv(EnvCanonicalBaseURL)); v != "" {
		return v
	}
	if v := normalizePublicBaseURL(os.Getenv("KAGEOS_BASE_URL")); v != "" {
		return v
	}
	if g != nil {
		if v := normalizePublicBaseURL(g.Gateway.GetBaseURL()); v != "" && !isLocalServiceBaseURL(v) {
			return v
		}
	}
	return "http://localhost:5173"
}

func GetPublicSiteBaseURL() string {
	return GetGlobalSharedConfig().GetPublicSiteBaseURL()
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

func normalizePublicBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}

func isLocalServiceBaseURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(strings.Trim(parsed.Hostname(), "[]"))
	switch host {
	case "", "localhost", "0.0.0.0":
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsUnspecified()
}

// SDKConfig SDK 配置（专门用于 runtime 构建 SDK app 时连接平台服务）
// SDK app 会在自身网络命名空间内自动探测本地候选地址，因此这里可以保存部署器渲染出的默认地址。
// NATS URL 包含认证信息时属于敏感配置，只能通过 Podman Secret 挂载；
// gateway_url 和 env_vars 中的非敏感配置仍作为环境变量注入。
type SDKConfig struct {
	NatsURL    string            `mapstructure:"nats_url"`    // NATS 地址，由 app-runtime 挂载到 /run/secrets/kageos-nats
	GatewayURL string            `mapstructure:"gateway_url"` // 网关地址（SDK 进程会自动探测本地候选地址），注入为 GATEWAY_URL 环境变量
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

// GetEnvVars 获取允许注入 App 容器的非敏感环境变量。
// NATS_URL 和 Secret 路径是 runtime 保留配置，不能通过 env_vars 覆盖。
// 返回 map[string]string，键为环境变量名，值为环境变量值
func (s *SDKConfig) GetEnvVars() map[string]string {
	envVars := make(map[string]string)

	if s.GatewayURL != "" {
		envVars["GATEWAY_URL"] = s.GatewayURL
	} else {
		envVars["GATEWAY_URL"] = "http://host.containers.internal:9090" // 默认值
	}

	// 注入 env_vars 中的额外环境变量（会覆盖固定字段，如果键名相同）
	if s.EnvVars != nil {
		for k, v := range s.EnvVars {
			switch strings.ToUpper(strings.TrimSpace(k)) {
			case "NATS_URL", "KAGEOS_NATS_CREDENTIALS_FILE":
				continue
			}
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
