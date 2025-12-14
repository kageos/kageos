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
	Server         AppServerServerConfig      `mapstructure:"server"`
	Nats           NatsConfig                 `mapstructure:"nats"`
	Timeouts       AppServerTimeoutCfg        `mapstructure:"timeouts"`
	DB             DBConfig                   `mapstructure:"db"`
	Email          EmailConfig                `mapstructure:"email"`
	JWT            JWTConfig                  `mapstructure:"jwt"`
	ControlService ControlServiceClientConfig `mapstructure:"control_service"`
}

// AppServerServerConfig app-server 服务器配置
type AppServerServerConfig struct {
	Port     int    `mapstructure:"port"`
	LogLevel string `mapstructure:"log_level"`
	Debug    bool   `mapstructure:"debug"`
}

// AppServerTimeoutCfg 超时配置
type AppServerTimeoutCfg struct {
	AppRequest  int `mapstructure:"app_request"`  // 应用请求超时（秒）
	NatsRequest int `mapstructure:"nats_request"` // NATS 请求超时（秒）
}

// EmailConfig 邮箱配置
type EmailConfig struct {
	SMTP         EmailSMTPConfig         `mapstructure:"smtp"`
	Verification EmailVerificationConfig `mapstructure:"verification"`
	Register     EmailRegisterConfig     `mapstructure:"register"`
}

// EmailSMTPConfig SMTP配置
type EmailSMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	FromName string `mapstructure:"from_name"`
}

// EmailVerificationConfig 邮箱验证配置
type EmailVerificationConfig struct {
	CodeLength int    `mapstructure:"code_length"`
	CodeExpire int    `mapstructure:"code_expire"`
	Template   string `mapstructure:"template"`
}

// EmailRegisterConfig 注册邮件配置
type EmailRegisterConfig struct {
	Subject  string `mapstructure:"subject"`
	Template string `mapstructure:"template"`
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
	LogFile       string `mapstructure:"log_file" json:"log_file"`             // 日志文件路径
	Colorful      bool   `mapstructure:"colorful" json:"colorful"`             // 是否彩色输出
}

// 常用便捷访问方法（可选）
func (c *AppServerConfig) GetPort() int               { return c.Server.Port }
func (c *AppServerConfig) GetLogLevel() string        { return c.Server.LogLevel }
func (c *AppServerConfig) IsDebug() bool              { return c.Server.Debug }
func (c *AppServerConfig) GetAppRequestTimeout() int  { return c.Timeouts.AppRequest }
func (c *AppServerConfig) GetNatsRequestTimeout() int { return c.Timeouts.NatsRequest }

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

func (c *AppServerConfig) IsDBLogColorful() bool {
	return c.DB.Colorful
}
