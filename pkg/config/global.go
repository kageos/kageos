package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// GlobalConfig 全局配置
type GlobalConfig struct {
	BuildMode string                   `mapstructure:"build_mode"`
	Logging   LoggingConfig            `mapstructure:"logging"`
	Runtime   GlobalRuntimeConfig      `mapstructure:"runtime"`
	Services  map[string]ServiceConfig `mapstructure:"services"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string                      `mapstructure:"level"`
	Output     string                      `mapstructure:"output"`
	TimeFormat string                      `mapstructure:"time_format"`
	ShowCaller bool                        `mapstructure:"show_caller"`
	Services   map[string]ServiceLogConfig `mapstructure:"services"`
}

// ServiceLogConfig 服务日志配置
type ServiceLogConfig struct {
	Level string `mapstructure:"level"`
}

// GlobalRuntimeConfig 全局运行时配置
type GlobalRuntimeConfig struct {
	RunDir    string `mapstructure:"run_dir"`
	ConfigDir string `mapstructure:"config_dir"`
}

// ServiceConfig 服务配置
type ServiceConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port"`
	Config  string `mapstructure:"config"`
}

var (
	globalConfig     *GlobalConfig
	globalConfigOnce sync.Once
	globalConfigMu   sync.RWMutex
)

// GetGlobalConfig 获取全局配置
func GetGlobalConfig() *GlobalConfig {
	globalConfigOnce.Do(func() {
		config := &GlobalConfig{}

		// 尝试加载配置文件
		if err := loadYAMLConfig("ai-agent-os.yaml", config); err != nil {
			// 配置文件不存在或加载失败，返回空配置
			fmt.Printf("Failed to load global config: %v\n", err)
			config = &GlobalConfig{}
		}

		globalConfigMu.Lock()
		globalConfig = config
		globalConfigMu.Unlock()
	})

	globalConfigMu.RLock()
	defer globalConfigMu.RUnlock()
	return globalConfig
}

// IsIntegratedBuild 是否为集成编译
func (gc *GlobalConfig) IsIntegratedBuild() bool {
	return gc.BuildMode == "integrated"
}

// IsSeparateBuild 是否为分开编译
func (gc *GlobalConfig) IsSeparateBuild() bool {
	return gc.BuildMode == "separate"
}

// GetServiceLogConfig 获取服务日志配置
func (gc *GlobalConfig) GetServiceLogConfig(serviceName string) ServiceLogConfig {
	if config, exists := gc.Logging.Services[serviceName]; exists {
		return config
	}

	// 返回默认配置
	return ServiceLogConfig{
		Level: gc.Logging.Level,
	}
}

// GetLogFilePath 获取日志文件路径
func (gc *GlobalConfig) GetLogFilePath(serviceName string) string {
	runDir := gc.GetRunDir()
	if gc.IsSeparateBuild() {
		// 分开编译：每个服务有自己的日志文件
		return filepath.Join(runDir, serviceName+".log")
	}
	// 集成编译：所有服务共享一个日志文件
	return filepath.Join(runDir, "ai-agent-os.log")
}

// GetRunDir 获取运行时目录
func (gc *GlobalConfig) GetRunDir() string {
	runDir := gc.Runtime.RunDir
	if runDir == "~/.ai-agent-os" {
		// 展开用户目录
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "./ai-agent-os" // 如果获取用户目录失败，使用当前目录
		}
		return filepath.Join(homeDir, ".ai-agent-os")
	}
	return runDir
}

// GetConfigDir 获取配置目录
func (gc *GlobalConfig) GetConfigDir() string {
	return gc.Runtime.ConfigDir
}
