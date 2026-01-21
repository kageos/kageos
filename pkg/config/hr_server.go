package config

import (
	"fmt"
	"os"
	"sync"
)

var (
	hrServerConfig *HRServerConfig
	hrServerOnce   sync.Once
	hrServerMu     sync.RWMutex
)

// GetHRServerConfig 获取 hr-server 配置
func GetHRServerConfig() *HRServerConfig {
	hrServerOnce.Do(func() {
		cfg := &HRServerConfig{}
		if err := loadYAMLConfig("hr-server.yaml", cfg); err != nil {
			// 配置文件不存在或加载失败，返回空配置
			fmt.Printf("Failed to load hr-server config: %v\n", err)
			cfg = &HRServerConfig{}
		}

		hrServerMu.Lock()
		hrServerConfig = cfg
		hrServerMu.Unlock()
	})

	hrServerMu.RLock()
	defer hrServerMu.RUnlock()
	return hrServerConfig
}

// HRServerConfig hr-server 配置
type HRServerConfig struct {
	Server     HRServerServerConfig `mapstructure:"server"`
	Email      EmailConfig          `mapstructure:"email"`
	DB         DBConfig             `mapstructure:"db"`
	SystemUser SystemUserConfig     `mapstructure:"system_user"` // ⭐ 系统账号配置
	// 注意：JWT 配置已移至全局配置，不再在此处配置
	// 数据库配置保留在服务配置中，因为微服务后续每个服务一个库
}

// SystemUserConfig 系统账号配置
type SystemUserConfig struct {
	Password string `mapstructure:"password"` // 系统账号密码（可选，如果为空则生成随机密码）
}

// HRServerServerConfig hr-server 服务器配置
type HRServerServerConfig struct {
	Port     int    `mapstructure:"port"`
	LogLevel string `mapstructure:"log_level"`
	Debug    bool   `mapstructure:"debug"`
}

// 常用便捷访问方法
func (c *HRServerConfig) GetPort() int {
	if c.Server.Port == 0 {
		return 9091 // 默认端口 9091
	}
	return c.Server.Port
}

func (c *HRServerConfig) GetLogLevel() string {
	if c.Server.LogLevel == "" {
		return "info"
	}
	return c.Server.LogLevel
}

func (c *HRServerConfig) IsDebug() bool {
	return c.Server.Debug
}

// 数据库配置便捷访问方法
func (c *HRServerConfig) GetDBLogLevel() string {
	if c.DB.LogLevel == "" {
		return "warn" // 默认日志级别
	}
	return c.DB.LogLevel
}

func (c *HRServerConfig) GetDBSlowThreshold() int {
	if c.DB.SlowThreshold == 0 {
		return 200 // 默认200毫秒
	}
	return c.DB.SlowThreshold
}

func (c *HRServerConfig) IsDBLogEnabled() bool {
	return c.DB.LogLevel != "silent"
}

// GetDB 获取数据库配置
func (c *HRServerConfig) GetDB() DBConfig {
	return c.DB
}

// GetDatabaseDSN 获取数据库连接字符串（与 app-server 共享数据库）
func (c *HRServerConfig) GetDatabaseDSN() string {
	db := c.DB
	if db.Type == "" {
		db.Type = "mysql"
	}
	if db.Host == "" {
		db.Host = "localhost"
	}
	if db.Port == 0 {
		db.Port = 3306
	}
	if db.User == "" {
		db.User = "root"
	}
	if db.Name == "" {
		db.Name = "ai_agent_os"
	}

	// MySQL DSN 格式: user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		db.User, db.Password, db.Host, db.Port, db.Name)
}

// GetJWT 获取 JWT 配置（从全局配置获取）
func (c *HRServerConfig) GetJWT() JWTConfig {
	return GetGlobalSharedConfig().JWT
}

// GetEmail 获取邮箱配置
func (c *HRServerConfig) GetEmail() EmailConfig {
	return c.Email
}

// GetSystemUserPassword 获取系统账号密码（优先从环境变量获取，其次从配置文件）
func (c *HRServerConfig) GetSystemUserPassword() string {
	// 优先从环境变量获取（容器化部署推荐）
	if envPassword := os.Getenv("SYSTEM_USER_PASSWORD"); envPassword != "" {
		return envPassword
	}
	// 其次从配置文件获取
	return c.SystemUser.Password
}
