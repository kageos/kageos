package config

import (
	"fmt"
	"strings"
	"sync"
)

var (
	messageServerConfig *MessageServerConfig
	messageServerOnce   sync.Once
	messageServerMu     sync.RWMutex
)

func GetMessageServerConfig() *MessageServerConfig {
	messageServerOnce.Do(func() {
		cfg := &MessageServerConfig{}
		if err := loadYAMLConfig("message-server.yaml", cfg); err != nil {
			fmt.Printf("Failed to load message-server config: %v\n", err)
			cfg = &MessageServerConfig{}
		}
		messageServerMu.Lock()
		messageServerConfig = cfg
		messageServerMu.Unlock()
	})

	messageServerMu.RLock()
	defer messageServerMu.RUnlock()
	return messageServerConfig
}

type MessageServerConfig struct {
	Server MessageServerHTTPConfig `mapstructure:"server"`
	DB     DBConfig                `mapstructure:"db"`
}

type MessageServerHTTPConfig struct {
	Port                     int    `mapstructure:"port"`
	ListenHost               string `mapstructure:"listen_host"`
	LogLevel                 string `mapstructure:"log_level"`
	Debug                    bool   `mapstructure:"debug"`
	EnablePprof              *bool  `mapstructure:"enable_pprof"`
	AllowNATSDegradedStartup bool   `mapstructure:"allow_nats_degraded_startup"`
}

func (c *MessageServerConfig) GetPort() int {
	if c == nil || c.Server.Port <= 0 {
		return 9099
	}
	return c.Server.Port
}

func (c *MessageServerConfig) GetListenHost() string {
	if c == nil {
		return normalizeListenHost("")
	}
	return normalizeListenHost(c.Server.ListenHost)
}

func (c *MessageServerConfig) GetLogLevel() string {
	if c == nil || c.Server.LogLevel == "" {
		return "info"
	}
	return c.Server.LogLevel
}

func (c *MessageServerConfig) IsDebug() bool {
	return c != nil && c.Server.Debug
}

func (c *MessageServerConfig) IsPprofEnabled() bool {
	if c == nil {
		return true
	}
	return boolConfigValue(c.Server.EnablePprof, true)
}

func (c *MessageServerConfig) AllowNATSDegradedStartup() bool {
	return c != nil && c.Server.AllowNATSDegradedStartup
}

func (c *MessageServerConfig) GetDB() DBConfig {
	if c == nil {
		return DBConfig{Type: "sqlite", Name: "data/message-server.db"}
	}
	return normalizeMessageDB(c.DB, "message-server")
}

func normalizeMessageDB(db DBConfig, defaultName string) DBConfig {
	if db.Type == "" {
		if strings.HasSuffix(strings.ToLower(strings.TrimSpace(db.Name)), ".db") {
			db.Type = "sqlite"
		} else {
			db.Type = "mysql"
		}
	}
	if db.Type == "sqlite" {
		if db.Name == "" {
			db.Name = "data/" + defaultName + ".db"
		}
		return db
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
		db.Name = defaultName
	}
	return db
}
