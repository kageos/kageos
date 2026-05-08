package config

import (
	"fmt"
	"sync"
)

var (
	appServerConfig *AppServerConfig
	appServerOnce   sync.Once
	appServerMu     sync.RWMutex
)

// GetAppServerConfig 获取 app-server 配置
func GetAppServerConfig() *AppServerConfig {
	appServerOnce.Do(func() {
		cfg := &AppServerConfig{}
		if err := loadYAMLConfig("app-server.yaml", cfg); err != nil {
			// 配置文件不存在或加载失败，返回空配置
			fmt.Printf("Failed to load app-server config: %v\n", err)
			cfg = &AppServerConfig{}
		}

		appServerMu.Lock()
		appServerConfig = cfg
		appServerMu.Unlock()
	})

	appServerMu.RLock()
	defer appServerMu.RUnlock()
	return appServerConfig
}

// AppServerConfig app-server 配置
type AppServerConfig struct {
	Server          AppServerServerConfig      `mapstructure:"server"`
	TimerWorker     AppServerTimerWorkerConfig `mapstructure:"timer_worker"`
	Timeouts        AppServerTimeoutCfg        `mapstructure:"timeouts"`
	DB              DBConfig                   `mapstructure:"db"`
	ScheduledTaskDB DBConfig                   `mapstructure:"scheduled_task_db"`
	// 注意：NATS、JWT、Control Service 配置已移至全局配置，不再在此处配置
	// 数据库配置保留在服务配置中，因为微服务后续每个服务一个库
}

// AppServerServerConfig app-server 服务器配置
type AppServerServerConfig struct {
	Port        int    `mapstructure:"port"`
	ListenHost  string `mapstructure:"listen_host"`
	LogLevel    string `mapstructure:"log_level"`
	Debug       bool   `mapstructure:"debug"`
	EnablePprof *bool  `mapstructure:"enable_pprof"`
}

// AppServerTimerWorkerConfig app-server 定时任务执行器配置。
type AppServerTimerWorkerConfig struct {
	MaxConcurrency int `mapstructure:"max_concurrency"`
}

// AppServerTimeoutCfg 超时配置
type AppServerTimeoutCfg struct {
	AppRequest  int `mapstructure:"app_request"`  // 应用请求超时（秒）
	NatsRequest int `mapstructure:"nats_request"` // NATS 请求超时（秒）
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret             string `mapstructure:"secret"`
	AccessTokenExpire  int    `mapstructure:"access_token_expire"`
	RefreshTokenExpire int    `mapstructure:"refresh_token_expire"`
	Issuer             string `mapstructure:"issuer"`
}

// DBConfig 数据库配置
type DBConfig struct {
	Type         string `mapstructure:"type" json:"type"` // mysql, postgres, sqlite
	Host         string `mapstructure:"host" json:"host"`
	Port         int    `mapstructure:"port" json:"port"`
	User         string `mapstructure:"user" json:"user"`
	Password     string `mapstructure:"password" json:"password"`
	Name         string `mapstructure:"name" json:"name"`
	MaxIdleConns int    `mapstructure:"max_idle_conns" json:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns" json:"max_open_conns"`
	MaxLifetime  int    `mapstructure:"max_lifetime" json:"max_lifetime"` // seconds

	// 数据库日志配置
	LogLevel      string `mapstructure:"log_level" json:"log_level"`           // silent, error, warn, info
	SlowThreshold int    `mapstructure:"slow_threshold" json:"slow_threshold"` // 慢查询阈值（毫秒）
}

// 常用便捷访问方法（可选）
func (c *AppServerConfig) GetPort() int { return c.Server.Port }
func (c *AppServerConfig) GetListenHost() string {
	if c == nil {
		return normalizeListenHost("")
	}
	return normalizeListenHost(c.Server.ListenHost)
}
func (c *AppServerConfig) GetLogLevel() string { return c.Server.LogLevel }
func (c *AppServerConfig) IsDebug() bool       { return c.Server.Debug }
func (c *AppServerConfig) IsPprofEnabled() bool {
	if c == nil {
		return true
	}
	return boolConfigValue(c.Server.EnablePprof, true)
}
func (c *AppServerConfig) GetAppRequestTimeout() int {
	if c.Timeouts.AppRequest <= 0 {
		return 300 // 默认 300 秒（5分钟）
	}
	return c.Timeouts.AppRequest
}
func (c *AppServerConfig) GetNatsRequestTimeout() int {
	if c.Timeouts.NatsRequest <= 0 {
		return 300 // 默认 300 秒（5分钟）
	}
	return c.Timeouts.NatsRequest
}

func (c *AppServerConfig) GetTimerWorkerMaxConcurrency() int {
	if c.TimerWorker.MaxConcurrency <= 0 {
		return 4
	}
	return c.TimerWorker.MaxConcurrency
}

// 数据库配置便捷访问方法
func (c *AppServerConfig) GetDBLogLevel() string {
	if c.DB.LogLevel == "" {
		return "warn" // 默认日志级别
	}
	return c.DB.LogLevel
}

func (c *AppServerConfig) GetDBSlowThreshold() int {
	if c.DB.SlowThreshold == 0 {
		return 200 // 默认200毫秒
	}
	return c.DB.SlowThreshold
}

func (c *AppServerConfig) IsDBLogEnabled() bool {
	return c.DB.LogLevel != "silent"
}

// GetDB 获取数据库配置
func (c *AppServerConfig) GetDB() DBConfig {
	return c.DB
}

// GetScheduledTaskDB 获取 app-server 业务定时任务数据库配置。
func (c *AppServerConfig) GetScheduledTaskDB() DBConfig {
	db := c.DB
	if c.ScheduledTaskDB.Type != "" {
		db.Type = c.ScheduledTaskDB.Type
	}
	if c.ScheduledTaskDB.Host != "" {
		db.Host = c.ScheduledTaskDB.Host
	}
	if c.ScheduledTaskDB.Port > 0 {
		db.Port = c.ScheduledTaskDB.Port
	}
	if c.ScheduledTaskDB.User != "" {
		db.User = c.ScheduledTaskDB.User
	}
	if c.ScheduledTaskDB.Password != "" {
		db.Password = c.ScheduledTaskDB.Password
	}
	if c.ScheduledTaskDB.Name != "" {
		db.Name = c.ScheduledTaskDB.Name
	} else {
		db.Name = "app-scheduled-task"
	}
	if c.ScheduledTaskDB.MaxIdleConns > 0 {
		db.MaxIdleConns = c.ScheduledTaskDB.MaxIdleConns
	}
	if c.ScheduledTaskDB.MaxOpenConns > 0 {
		db.MaxOpenConns = c.ScheduledTaskDB.MaxOpenConns
	}
	if c.ScheduledTaskDB.MaxLifetime > 0 {
		db.MaxLifetime = c.ScheduledTaskDB.MaxLifetime
	}
	if c.ScheduledTaskDB.LogLevel != "" {
		db.LogLevel = c.ScheduledTaskDB.LogLevel
	}
	if c.ScheduledTaskDB.SlowThreshold > 0 {
		db.SlowThreshold = c.ScheduledTaskDB.SlowThreshold
	}
	return db
}

// GetNats 获取 NATS 配置（从全局配置获取）
func (c *AppServerConfig) GetNats() NatsConfig {
	return GetGlobalSharedConfig().Nats
}

// GetJWT 获取 JWT 配置（从全局配置获取）
func (c *AppServerConfig) GetJWT() JWTConfig {
	return GetGlobalSharedConfig().JWT
}

// GetControlService 获取 Control Service 配置（从全局配置获取）
func (c *AppServerConfig) GetControlService() ControlServiceClientConfig {
	return GetGlobalSharedConfig().ControlService
}
