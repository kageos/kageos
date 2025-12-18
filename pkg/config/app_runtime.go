package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"
)

var (
	appRuntimeConfig *AppRuntimeConfig
	appRuntimeOnce   sync.Once
	appRuntimeMu     sync.RWMutex
)

// GetAppRuntimeConfig 获取 app-runtime 配置
func GetAppRuntimeConfig() *AppRuntimeConfig {
	appRuntimeOnce.Do(func() {
		config := &AppRuntimeConfig{}

		// 尝试加载配置文件
		if err := loadYAMLConfig("app-runtime.yaml", config); err != nil {
			// 配置文件不存在或加载失败，返回空配置
			fmt.Printf("Failed to load app-runtime config: %v\n", err)
			config = &AppRuntimeConfig{}
		}

		appRuntimeMu.Lock()
		appRuntimeConfig = config
		appRuntimeMu.Unlock()
	})

	appRuntimeMu.RLock()
	defer appRuntimeMu.RUnlock()
	return appRuntimeConfig
}

// NatsConfig NATS 配置
type NatsConfig struct {
	URL string `mapstructure:"url"`
}

// AppRuntimeConfig app-runtime 配置
type AppRuntimeConfig struct {
	Runtime   RuntimeConfig           `mapstructure:"runtime"`
	Nats      NatsConfig              `mapstructure:"nats"`
	Timeouts  AppRuntimeTimeoutConfig `mapstructure:"timeouts"`
	AppManage AppManageServiceConfig  `mapstructure:"app_manage"`
	Container ContainerServiceConfig  `mapstructure:"container"`
}

// AppRuntimeTimeoutConfig App Runtime 超时配置
type AppRuntimeTimeoutConfig struct {
	FunctionServerRequest int `mapstructure:"function_server_request"` // app-server 请求处理超时时间（秒）
	ContainerStartup      int `mapstructure:"container_startup"`       // 容器启动等待时间（秒）
	ContainerCleanup      int `mapstructure:"container_cleanup"`       // 容器清理等待时间（秒）
}

// RuntimeConfig 运行时配置
type RuntimeConfig struct {
	Port       int    `mapstructure:"port"`
	LogLevel   string `mapstructure:"log_level"`
	Debug      bool   `mapstructure:"debug"`
	GatewayURL string `mapstructure:"gateway_url"` // 网关地址（会注入到 SDK 容器中）
}

// AppManageServiceConfig App Manage 服务配置
type AppManageServiceConfig struct {
	AppDir AppDirConfig `mapstructure:"app_dir"`
	Build  BuildConfig  `mapstructure:"build"`
	Git    GitConfig    `mapstructure:"git"` // Git 配置
}

// AppDirConfig 应用目录配置
type AppDirConfig struct {
	BasePath  string   `mapstructure:"base_path"`
	Structure []string `mapstructure:"structure"`
}

// BuildConfig 编译配置
type BuildConfig struct {
	Platform         string `mapstructure:"platform"`
	OutputDir        string `mapstructure:"output_dir"`
	BinaryNameFormat string `mapstructure:"binary_name_format"`
}

// GitConfig Git 配置
type GitConfig struct {
	EmailSuffix string `mapstructure:"email_suffix"` // Git 邮箱后缀（如 "ai-agent-os.com"）
}

// ContainerServiceConfig 容器服务配置
type ContainerServiceConfig struct {
	Runtime        string               `mapstructure:"runtime"` // podman, docker
	Socket         string               `mapstructure:"socket"`  // 容器运行时 socket 路径
	Timeout        int                  `mapstructure:"timeout"` // 连接超时时间（秒）
	Image          ImageConfig          `mapstructure:"image"`
	Environment    []string             `mapstructure:"environment"`
	Infrastructure InfrastructureConfig `mapstructure:"infrastructure"`
}

// ImageConfig 镜像配置
type ImageConfig struct {
	BaseImage     string   `mapstructure:"base_image"`
	ContainerPath string   `mapstructure:"container_path"`
	Command       []string `mapstructure:"command"`
	RestartPolicy string   `mapstructure:"restart_policy"`
}

// InfrastructureConfig 基础设施配置
type InfrastructureConfig struct {
	AutoStartContainers []string            `mapstructure:"auto_start_containers"`
	Nats                ContainerNatsConfig `mapstructure:"nats"`
}

// ContainerNatsConfig 容器 NATS 配置
type ContainerNatsConfig struct {
	Image         string   `mapstructure:"image"`
	Ports         []string `mapstructure:"ports"`
	RestartPolicy string   `mapstructure:"restart_policy"`
}

// Validate 验证配置
func (c *AppRuntimeConfig) Validate() error {
	// 验证运行时配置
	if c.Runtime.Port <= 0 || c.Runtime.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Runtime.Port)
	}
	if c.Runtime.LogLevel == "" {
		return fmt.Errorf("log level cannot be empty")
	}

	// 验证容器配置
	if c.Container.Runtime == "" {
		return fmt.Errorf("container runtime cannot be empty")
	}
	if c.Container.Timeout <= 0 {
		return fmt.Errorf("container timeout must be positive")
	}

	// 验证应用管理配置
	if c.AppManage.AppDir.BasePath == "" {
		return fmt.Errorf("app directory base path cannot be empty")
	}

	// 将相对路径转换为绝对路径
	if !filepath.IsAbs(c.AppManage.AppDir.BasePath) {
		absPath, err := filepath.Abs(c.AppManage.AppDir.BasePath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for base_path: %w", err)
		}
		c.AppManage.AppDir.BasePath = absPath
	}

	if c.AppManage.Build.Platform == "" {
		return fmt.Errorf("build platform cannot be empty")
	}

	return nil
}

// GetFunctionServerRequestTimeout 获取 app-server 请求处理超时时间
func (c *AppRuntimeConfig) GetFunctionServerRequestTimeout() int {
	if c.Timeouts.FunctionServerRequest <= 0 {
		return 30 // 默认 30 秒
	}
	return c.Timeouts.FunctionServerRequest
}

// GetContainerStartupTimeout 获取容器启动等待时间
func (c *AppRuntimeConfig) GetContainerStartupTimeout() int {
	if c.Timeouts.ContainerStartup <= 0 {
		return 2 // 默认 2 秒
	}
	return c.Timeouts.ContainerStartup
}

// GetContainerCleanupTimeout 获取容器清理等待时间
func (c *AppRuntimeConfig) GetContainerCleanupTimeout() int {
	if c.Timeouts.ContainerCleanup <= 0 {
		return 10 // 默认 10 秒
	}
	return c.Timeouts.ContainerCleanup
}

// loadYAMLConfig 加载 YAML 配置文件
func loadYAMLConfig(filename string, config interface{}) error {
	// 查找配置文件
	configPath := findConfigFile(filename)
	if configPath == "" {
		return fmt.Errorf("config file not found: %s", filename)
	}

	// 读取文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析 YAML
	var rawConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// 转换为结构体
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "mapstructure",
		Result:  config,
	})
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}

	if err := decoder.Decode(rawConfig); err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}

	return nil
}

// findConfigFile 查找配置文件
func findConfigFile(filename string) string {
	// 按优先级查找配置文件
	searchPaths := []string{
		filename,                                 // 当前目录
		filepath.Join("configs", filename),       // configs 目录
		filepath.Join("..", "configs", filename), // 上级目录的 configs
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}
