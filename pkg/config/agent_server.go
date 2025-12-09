package config

import (
	"fmt"
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
	Server   AgentServerServerConfig `mapstructure:"server"`
	DB       DBConfig                 `mapstructure:"db"`
	Agent    AgentConfig              `mapstructure:"agent"`
	CodeGen  CodeGenConfig            `mapstructure:"code_gen"`
	Builder  BuilderConfig            `mapstructure:"builder"`
}

// AgentServerServerConfig agent-server 服务器配置
type AgentServerServerConfig struct {
	Port     int    `mapstructure:"port"`
	LogLevel string `mapstructure:"log_level"`
	Debug    bool   `mapstructure:"debug"`
}

// AgentConfig 智能体配置
type AgentConfig struct {
	Timeout int            `mapstructure:"timeout"`
	Retry   RetryConfig    `mapstructure:"retry"`
	Nats    AgentNatsConfig `mapstructure:"nats"`
}

// AgentNatsConfig Agent Server NATS 配置
type AgentNatsConfig struct {
	Host    string `mapstructure:"host"`    // NATS 服务器地址，例如：127.0.0.1:4222
	Timeout int    `mapstructure:"timeout"` // NATS 请求超时时间（秒）
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxAttempts int    `mapstructure:"max_attempts"`
	Backoff     string `mapstructure:"backoff"` // exponential, linear, fixed
}

// CodeGenConfig 代码生成配置
type CodeGenConfig struct {
	SDKVersion  string `mapstructure:"sdk_version"`
	TemplatePath string `mapstructure:"template_path"`
	OutputPath   string `mapstructure:"output_path"`
}

// BuilderConfig 应用构建配置
type BuilderConfig struct {
	Timeout      int    `mapstructure:"timeout"`
	WorkspacePath string `mapstructure:"workspace_path"`
}

// 便捷访问方法
func (c *AgentServerConfig) GetPort() int        { return c.Server.Port }
func (c *AgentServerConfig) GetLogLevel() string  { return c.Server.LogLevel }
func (c *AgentServerConfig) IsDebug() bool        { return c.Server.Debug }
func (c *AgentServerConfig) GetAgentTimeout() int { return c.Agent.Timeout }
func (c *AgentServerConfig) GetBuilderTimeout() int { return c.Builder.Timeout }
func (c *AgentServerConfig) GetNatsHost() string {
	if c.Agent.Nats.Host == "" {
		return "127.0.0.1:4222" // 默认值
	}
	return c.Agent.Nats.Host
}
func (c *AgentServerConfig) GetNatsTimeout() int {
	if c.Agent.Nats.Timeout == 0 {
		return 600 // 默认 600 秒
	}
	return c.Agent.Nats.Timeout
}

// 数据库配置便捷访问方法（复用 DBConfig 的方法）
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

func (c *AgentServerConfig) IsDBLogColorful() bool {
	return c.DB.Colorful
}

