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
		// 与全项目一致：仅 deploy/config/{dev|prod}/hub.yaml（APP_ENV）
		if err := loadYAMLConfig("hub.yaml", cfg); err != nil {
			// 配置文件不存在或加载失败，返回空配置；copy_url 会回退为请求 Host（如 localhost:9090）
			fmt.Printf("Failed to load hub config, using defaults: %v (set APP_ENV=dev or AI_AGENT_OS_ROOT; expect deploy/config/{dev|prod}/hub.yaml)\n", err)
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

// OSConfig 主站相关配置（用于 Hub 试用跳转等）
type OSConfig struct {
	BaseURL string `mapstructure:"base_url"` // 主站前端地址（如 http://localhost:5173 或 http://125.122.96.207:8999），不是后端 API 地址
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

// GetMainSiteURL 获取主站前端地址，用于 Hub 点击「试用」时跳转（主站前端 + /workspace + 目录路径）。注意是前端地址不是后端 API。
func (c *HubConfig) GetMainSiteURL() string {
	return strings.TrimRight(strings.TrimSpace(c.OS.BaseURL), "/")
}
