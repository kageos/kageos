package config

import (
	"fmt"
	"strings"
	"sync"
)

var (
	agentServerConfig *AgentServerConfig
	agentServerOnce   sync.Once
	agentServerMu     sync.RWMutex
)

// GetAgentServerConfig 获取 agent-server 配置
func GetAgentServerConfig() *AgentServerConfig {
	agentServerOnce.Do(func() {
		cfg := &AgentServerConfig{}
		if err := loadYAMLConfig("agent-server.yaml", cfg); err != nil {
			// 配置文件不存在或加载失败，返回空配置
			fmt.Printf("Failed to load agent-server config: %v\n", err)
			cfg = &AgentServerConfig{}
		}

		agentServerMu.Lock()
		agentServerConfig = cfg
		agentServerMu.Unlock()
	})

	agentServerMu.RLock()
	defer agentServerMu.RUnlock()
	return agentServerConfig
}

// AgentServerConfig agent-server 配置
type AgentServerConfig struct {
	Server AgentServerServerConfig `mapstructure:"server"`
	Agent  AgentConfig            `mapstructure:"agent"`
	DB     DBConfig               `mapstructure:"db"`
	// 注意：Control Service 配置已移至全局配置，不再在此处配置
	// 数据库配置保留在服务配置中，因为微服务后续每个服务一个库
}

// AgentServerServerConfig agent-server 服务器配置
type AgentServerServerConfig struct {
	Port     int    `mapstructure:"port"`
	LogLevel string `mapstructure:"log_level"`
	Debug    bool   `mapstructure:"debug"`
}

// AgentConfig 智能体配置
type AgentConfig struct {
	Timeout int `mapstructure:"timeout"`
	// 注意：NATS 配置已移至全局配置，不再在此处配置
}

// 便捷访问方法
func (c *AgentServerConfig) GetPort() int         { return c.Server.Port }
func (c *AgentServerConfig) GetLogLevel() string  { return c.Server.LogLevel }
func (c *AgentServerConfig) IsDebug() bool        { return c.Server.Debug }
func (c *AgentServerConfig) GetAgentTimeout() int { return c.Agent.Timeout }
// GetNatsHost 获取 NATS 地址（从全局配置读取）
func (c *AgentServerConfig) GetNatsHost() string {
	global := GetGlobalSharedConfig()
	if global.Nats.URL != "" {
		// 从 nats://127.0.0.1:4222 格式中提取 host:port
		url := global.Nats.URL
		if strings.HasPrefix(url, "nats://") {
			return strings.TrimPrefix(url, "nats://")
		}
		return url
	}
	return "127.0.0.1:4222" // 默认值
}

// GetNatsTimeout 获取 NATS 请求超时时间（秒）
func (c *AgentServerConfig) GetNatsTimeout() int {
	// 默认 600 秒
	return 600
}

// 数据库配置便捷访问方法
func (c *AgentServerConfig) GetDBLogLevel() string {
	if c.DB.LogLevel == "" {
		return "warn"
	}
	return c.DB.LogLevel
}

func (c *AgentServerConfig) GetDBSlowThreshold() int {
	if c.DB.SlowThreshold == 0 {
		return 200
	}
	return c.DB.SlowThreshold
}

func (c *AgentServerConfig) IsDBLogEnabled() bool {
	return c.DB.LogLevel != "silent"
}

// GetDB 获取数据库配置
func (c *AgentServerConfig) GetDB() DBConfig {
	return c.DB
}

// GetControlService 获取 Control Service 配置（从全局配置获取）
func (c *AgentServerConfig) GetControlService() ControlServiceClientConfig {
	return GetGlobalSharedConfig().ControlService
}
