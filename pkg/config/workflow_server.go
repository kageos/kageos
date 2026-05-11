package config

import (
	"fmt"
	"sync"
)

var (
	workflowServerConfig *WorkflowServerConfig
	workflowServerOnce   sync.Once
	workflowServerMu     sync.RWMutex
)

func GetWorkflowServerConfig() *WorkflowServerConfig {
	workflowServerOnce.Do(func() {
		cfg := &WorkflowServerConfig{}
		if err := loadYAMLConfig("workflow-server.yaml", cfg); err != nil {
			fmt.Printf("Failed to load workflow-server config: %v\n", err)
			cfg = &WorkflowServerConfig{}
		}
		workflowServerMu.Lock()
		workflowServerConfig = cfg
		workflowServerMu.Unlock()
	})

	workflowServerMu.RLock()
	defer workflowServerMu.RUnlock()
	return workflowServerConfig
}

type WorkflowServerConfig struct {
	Server WorkflowServerHTTPConfig `mapstructure:"server"`
	DB     DBConfig                 `mapstructure:"db"`
}

type WorkflowServerHTTPConfig struct {
	Port       int    `mapstructure:"port"`
	ListenHost string `mapstructure:"listen_host"`
	LogLevel   string `mapstructure:"log_level"`
	Debug      bool   `mapstructure:"debug"`
}

func (c *WorkflowServerConfig) GetPort() int {
	if c == nil || c.Server.Port <= 0 {
		return 9110
	}
	return c.Server.Port
}

func (c *WorkflowServerConfig) GetListenHost() string {
	if c == nil {
		return normalizeListenHost("")
	}
	return normalizeListenHost(c.Server.ListenHost)
}

func (c *WorkflowServerConfig) GetLogLevel() string {
	if c == nil || c.Server.LogLevel == "" {
		return "info"
	}
	return c.Server.LogLevel
}

func (c *WorkflowServerConfig) IsDebug() bool {
	return c != nil && c.Server.Debug
}

func (c *WorkflowServerConfig) GetDB() DBConfig {
	if c == nil {
		return DBConfig{Type: "mysql", Name: "workflow-server"}
	}
	db := c.DB
	if db.Type == "" {
		db.Type = "mysql"
	}
	if db.Host == "" {
		db.Host = "127.0.0.1"
	}
	if db.Port == 0 {
		db.Port = 3306
	}
	if db.User == "" {
		db.User = "root"
	}
	if db.Name == "" {
		db.Name = "workflow-server"
	}
	return db
}
