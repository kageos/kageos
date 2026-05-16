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
	Server AgentServerServerConfig   `mapstructure:"server"`
	DB     DBConfig                  `mapstructure:"db"`
	LLMs   AgentServerLLMSeedsConfig `mapstructure:"llms"`
	// 注意：Control Service 配置已移至全局配置，不再在此处配置
	// 数据库配置保留在服务配置中，因为微服务后续每个服务一个库
}

// AgentServerServerConfig agent-server 服务器配置
type AgentServerServerConfig struct {
	Port        int    `mapstructure:"port"`
	ListenHost  string `mapstructure:"listen_host"`
	LogLevel    string `mapstructure:"log_level"`
	Debug       bool   `mapstructure:"debug"`
	EnablePprof *bool  `mapstructure:"enable_pprof"`
}

type AgentServerLLMSeedsConfig struct {
	Default string                     `mapstructure:"default"`
	Configs []AgentServerLLMSeedConfig `mapstructure:"configs"`
}

type AgentServerLLMSeedConfig struct {
	Code        string `mapstructure:"code"`
	Name        string `mapstructure:"name"`
	Provider    string `mapstructure:"provider"`
	Model       string `mapstructure:"model"`
	APIKey      string `mapstructure:"api_key"`
	APIKeyEnv   string `mapstructure:"api_key_env"`
	APIBase     string `mapstructure:"api_base"`
	Timeout     int    `mapstructure:"timeout"`
	MaxTokens   int    `mapstructure:"max_tokens"`
	ExtraConfig string `mapstructure:"extra_config"`
	UseThinking bool   `mapstructure:"use_thinking"`
	IsDefault   bool   `mapstructure:"is_default"`
	Visibility  int    `mapstructure:"visibility"`
	Admin       string `mapstructure:"admin"`
}

// 便捷访问方法
func (c *AgentServerConfig) GetPort() int { return c.Server.Port }
func (c *AgentServerConfig) GetListenHost() string {
	if c == nil {
		return normalizeListenHost("")
	}
	return normalizeListenHost(c.Server.ListenHost)
}
func (c *AgentServerConfig) GetLogLevel() string { return c.Server.LogLevel }
func (c *AgentServerConfig) IsDebug() bool       { return c.Server.Debug }
func (c *AgentServerConfig) IsPprofEnabled() bool {
	if c == nil {
		return true
	}
	return boolConfigValue(c.Server.EnablePprof, true)
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
