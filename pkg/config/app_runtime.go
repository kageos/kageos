package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	Timeouts  AppRuntimeTimeoutConfig `mapstructure:"timeouts"`
	AppManage AppManageServiceConfig  `mapstructure:"app_manage"`
	Container ContainerServiceConfig  `mapstructure:"container"`
	// 注意：NATS 配置已移至全局配置，不再在此处配置
}

// AppRuntimeTimeoutConfig App Runtime 超时配置
type AppRuntimeTimeoutConfig struct {
	FunctionServerRequest int `mapstructure:"function_server_request"` // app-server 请求处理超时时间（秒）
	ContainerStartup      int `mapstructure:"container_startup"`       // 容器启动等待时间（秒）
	ContainerCleanup      int `mapstructure:"container_cleanup"`       // 容器清理等待时间（秒）
}

// RuntimeConfig 运行时配置
type RuntimeConfig struct {
	Port     int    `mapstructure:"port"`
	LogLevel string `mapstructure:"log_level"`
	Debug    bool   `mapstructure:"debug"`
	// 注意：gateway_url 已移除，改为从全局配置读取（GetGatewayURL()）
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
	OutputDir        string `mapstructure:"output_dir"`
	BinaryNameFormat string `mapstructure:"binary_name_format"`
}

// GitConfig Git 配置
type GitConfig struct {
	EmailSuffix string `mapstructure:"email_suffix"` // Git 邮箱后缀（如 "ai-agent-os.com"）
}

// ContainerServiceConfig 容器服务配置
type ContainerServiceConfig struct {
	Runtime string `mapstructure:"runtime"` // podman, docker
	Socket  string `mapstructure:"socket"`  // 容器运行时 socket 路径
	Timeout int    `mapstructure:"timeout"` // 连接超时时间（秒）
	// LSM 模式：为后续内核级安全（如防删 code/workplace）做准备。
	// - auto: 启动时检测宿主机 LSM（同机读 /sys，Mac/Win 起临时容器探测），结果缓存，后续只启用匹配的一种。
	// - apparmor / selinux: 强制使用该 LSM（不检测）。
	// - none: 不使用 LSM 相关安全选项。
	LSMMode         string `mapstructure:"lsm_mode"`          // auto / apparmor / selinux / none
	AppArmorProfile string `mapstructure:"apparmor_profile"`  // AppArmor 环境下使用的 profile 名（如 ai-agent-os-app），空则不启用
	Image           ImageConfig `mapstructure:"image"`
}

// ImageConfig 镜像配置
type ImageConfig struct {
	BaseImage     string   `mapstructure:"base_image"`
	ContainerPath string   `mapstructure:"container_path"`
	Command       []string `mapstructure:"command"`
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

// loadYAMLConfig 加载 YAML 配置文件（仅从 configs/dev 或 configs/prod 解析，由 APP_ENV 决定）。
// 加载成功时打印实际使用的配置路径，避免糊涂账。
func loadYAMLConfig(filename string, config interface{}) error {
	configPath := findConfigFile(filename)
	if configPath == "" {
		return fmt.Errorf("config file not found: %s", filename)
	}
	absPath, _ := filepath.Abs(configPath)
	env := getConfigEnv()
	fmt.Printf("[Config] APP_ENV=%s  %s <- %s\n", env, filepath.Base(filename), absPath)

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

// getConfigEnv 返回当前配置环境：仅 dev 为开发，其余（含未设）均为 prod
// 本机开发时设置 APP_ENV=dev 用 configs/dev/；不设或 APP_ENV=prod 用 configs/prod/
func getConfigEnv() string {
	e := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	if e == "dev" {
		return "dev"
	}
	return "prod" // 未设或任意其他值都当 prod
}

// findProjectRoot 从 dir 向上查找包含 go.mod 或 configs 的目录作为项目根
func findProjectRoot(dir string) string {
	dir, _ = filepath.Abs(dir)
	for {
		if dir == "" || dir == "/" || len(dir) <= 1 {
			return ""
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, "configs")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
}

// findConfigFile 查找配置文件
// 只从 configs/dev/ 或 configs/prod/ 读；先按 cwd 相对路径试，再按项目根（向上找 go.mod/configs）试
func findConfigFile(filename string) string {
	env := getConfigEnv()
	baseName := filepath.Base(filename)
	wantPath := filepath.Join("configs", env, baseName)

	// 1. 相对 cwd 的常见层级
	for _, rel := range []string{"", "..", "../..", "../../..", "../../../..", "../../../../.."} {
		var path string
		if rel == "" {
			path = wantPath
		} else {
			path = filepath.Join(rel, wantPath)
		}
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// 2. 从 cwd 向上找到项目根，再找 configs/{env}/baseName
	if cwd, _ := os.Getwd(); cwd != "" {
		if root := findProjectRoot(cwd); root != "" {
			path := filepath.Join(root, wantPath)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	return ""
}
