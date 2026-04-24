package config

import (
	"fmt"
	"sync"
	"time"
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
	Server    AgentServerServerConfig    `mapstructure:"server"`
	Scheduler AgentServerSchedulerConfig `mapstructure:"scheduler"`
	DB        DBConfig                   `mapstructure:"db"`
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

// AgentServerSchedulerConfig 定时 Agent 会话调度配置。
type AgentServerSchedulerConfig struct {
	PollIntervalSeconds   int `mapstructure:"poll_interval_seconds"`
	BatchSize             int `mapstructure:"batch_size"`
	LeaseDurationSeconds  int `mapstructure:"lease_duration_seconds"`
	MaxConcurrency        int `mapstructure:"max_concurrency"`
	DefaultTimeoutSeconds int `mapstructure:"default_timeout_seconds"`
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

func (c *AgentServerConfig) GetSchedulerPollInterval() time.Duration {
	if c == nil || c.Scheduler.PollIntervalSeconds <= 0 {
		return 5 * time.Second
	}
	return time.Duration(c.Scheduler.PollIntervalSeconds) * time.Second
}

func (c *AgentServerConfig) GetSchedulerBatchSize() int {
	if c == nil || c.Scheduler.BatchSize <= 0 {
		return 20
	}
	return c.Scheduler.BatchSize
}

func (c *AgentServerConfig) GetSchedulerLeaseDuration() time.Duration {
	if c == nil || c.Scheduler.LeaseDurationSeconds <= 0 {
		return 10 * time.Minute
	}
	return time.Duration(c.Scheduler.LeaseDurationSeconds) * time.Second
}

func (c *AgentServerConfig) GetSchedulerMaxConcurrency() int {
	if c == nil || c.Scheduler.MaxConcurrency <= 0 {
		return 3
	}
	return c.Scheduler.MaxConcurrency
}

func (c *AgentServerConfig) GetSchedulerDefaultTimeout() time.Duration {
	if c == nil || c.Scheduler.DefaultTimeoutSeconds <= 0 {
		return 5 * time.Minute
	}
	return time.Duration(c.Scheduler.DefaultTimeoutSeconds) * time.Second
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
