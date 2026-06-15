package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	Runtime     RuntimeConfig           `mapstructure:"runtime"`
	Timeouts    AppRuntimeTimeoutConfig `mapstructure:"timeouts"`
	AppManage   AppManageServiceConfig  `mapstructure:"app_manage"`
	Container   ContainerServiceConfig  `mapstructure:"container"`
	AppDatabase AppDatabaseConfig       `mapstructure:"app_database"`
	// 注意：NATS 配置已移至全局配置，不再在此处配置
}

var defaultRuntimeAppDirStructure = []string{
	"code",
	"code/api",
	"code/cmd/app",
	"workplace",
	"workplace/bin",
	"workplace/bin/releases",
	"workplace/api-logs",
	"workplace/data",
	"workplace/logs",
	"workplace/metadata",
}

const (
	defaultRuntimeAppBasePath      = "namespace"
	defaultRuntimeBuildOutputDir   = "workplace/bin/releases"
	defaultRuntimeBinaryNameFormat = "{user}_{app}_{version}"
	defaultRuntimeGitEmailSuffix   = "kageos.com"
	defaultContainerRuntime        = "podman"
	defaultContainerLSMMode        = "auto"
	defaultAppArmorProfile         = "kageos-app"
	defaultContainerBaseImage      = "kagebase:latest"
	defaultContainerPath           = "/app"
)

const (
	defaultAppDatabaseDialect      = "mysql"
	defaultAppDatabaseHost         = "127.0.0.1"
	defaultAppDatabasePort         = 3306
	defaultAppDatabaseGrantHost    = "%"
	defaultAppDatabaseNamePrefix   = "kgo_"
	defaultAppDatabaseUserPrefix   = "kgu_"
	defaultAppDatabaseMaxOpenConns = 2
	defaultAppDatabaseMaxIdleConns = 0
	defaultAppDatabaseMaxIdleTime  = 30
	defaultAppDatabaseMaxLifetime  = 600
)

// AppRuntimeTimeoutConfig App Runtime 超时配置
type AppRuntimeTimeoutConfig struct {
	AppServerRequest       int `mapstructure:"app_server_request"`       // app-server 请求处理超时时间（秒）
	ContainerStartup       int `mapstructure:"container_startup"`        // 容器启动等待时间（秒）
	AppStartupNotification int `mapstructure:"app_startup_notification"` // 应用启动通知等待时间（秒）
	ContainerCleanup       int `mapstructure:"container_cleanup"`        // 容器清理等待时间（秒）
}

// RuntimeConfig 运行时配置
type RuntimeConfig struct {
	Port       int    `mapstructure:"port"`
	ListenHost string `mapstructure:"listen_host"`
	LogLevel   string `mapstructure:"log_level"`
	Debug      bool   `mapstructure:"debug"`
	InstanceID string `mapstructure:"instance_id"`
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
	EmailSuffix string `mapstructure:"email_suffix"` // Git 邮箱后缀（如 "kageos.com"）
}

// AppDatabaseConfig controls runtime-managed per-package databases for SDK
// apps. Admin credentials stay in app-runtime and are never injected into app
// containers.
type AppDatabaseConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	Dialect        string `mapstructure:"dialect"`
	Host           string `mapstructure:"host"`
	Port           int    `mapstructure:"port"`
	AdminUser      string `mapstructure:"admin_user"`
	AdminPassword  string `mapstructure:"admin_password"`
	GrantHost      string `mapstructure:"grant_host"`
	SecretKey      string `mapstructure:"secret_key"`
	ClusterKey     string `mapstructure:"cluster_key"`
	DatabasePrefix string `mapstructure:"database_prefix"`
	UserPrefix     string `mapstructure:"user_prefix"`
	MaxOpenConns   int    `mapstructure:"max_open_conns"`
	MaxIdleConns   int    `mapstructure:"max_idle_conns"`
	MaxIdleTime    int    `mapstructure:"max_idle_time"` // seconds
	MaxLifetime    int    `mapstructure:"max_lifetime"`  // seconds
}

func (c *AppManageServiceConfig) GetBasePath() string {
	if c == nil {
		return defaultRuntimeAppBasePath
	}
	if v := strings.TrimSpace(c.AppDir.BasePath); v != "" {
		return filepath.Clean(v)
	}
	return defaultRuntimeAppBasePath
}

func (c *AppManageServiceConfig) GetStructure() []string {
	source := defaultRuntimeAppDirStructure
	if c != nil && len(c.AppDir.Structure) > 0 {
		source = c.AppDir.Structure
	}

	normalized := make([]string, 0, len(source))
	for _, dir := range source {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		normalized = append(normalized, filepath.Clean(dir))
	}
	if len(normalized) == 0 {
		return append([]string(nil), defaultRuntimeAppDirStructure...)
	}
	return normalized
}

func (c *AppManageServiceConfig) GetBuildOutputDir() string {
	if c == nil {
		return defaultRuntimeBuildOutputDir
	}
	if v := strings.TrimSpace(c.Build.OutputDir); v != "" {
		return filepath.Clean(v)
	}
	return defaultRuntimeBuildOutputDir
}

func (c *AppManageServiceConfig) GetBinaryNameFormat() string {
	if c == nil {
		return defaultRuntimeBinaryNameFormat
	}
	if v := strings.TrimSpace(c.Build.BinaryNameFormat); v != "" {
		return v
	}
	return defaultRuntimeBinaryNameFormat
}

func (c *AppManageServiceConfig) GetGitEmailSuffix() string {
	if c == nil {
		return defaultRuntimeGitEmailSuffix
	}
	if v := strings.TrimSpace(c.Git.EmailSuffix); v != "" {
		return v
	}
	return defaultRuntimeGitEmailSuffix
}

// ContainerServiceConfig 容器服务配置
type ContainerServiceConfig struct {
	Runtime string `mapstructure:"runtime"` // 当前仅支持 podman
	Socket  string `mapstructure:"socket"`  // 容器运行时 socket 路径
	Timeout int    `mapstructure:"timeout"` // 连接超时时间（秒）
	// LSM 模式：为后续内核级安全（如防删 code/workplace）做准备。
	// - auto: 启动时检测宿主机 LSM（同机读 /sys，Mac/Win 起临时容器探测），结果缓存，后续只启用匹配的一种。
	// - apparmor / selinux: 强制使用该 LSM（不检测）。
	// - none: 不使用 LSM 相关安全选项。
	LSMMode         string      `mapstructure:"lsm_mode"`         // auto / apparmor / selinux / none
	AppArmorProfile string      `mapstructure:"apparmor_profile"` // AppArmor 环境下使用的 profile 名；未配置时默认 kageos-app
	Image           ImageConfig `mapstructure:"image"`
}

// ImageConfig 镜像配置
type ImageConfig struct {
	BaseImage     string `mapstructure:"base_image"`
	ContainerPath string `mapstructure:"container_path"`
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
	runtimeName := c.Container.GetRuntime()
	if runtimeName != defaultContainerRuntime {
		return fmt.Errorf("unsupported container runtime: %s (only %s is supported)", runtimeName, defaultContainerRuntime)
	}
	if c.Container.Timeout <= 0 {
		return fmt.Errorf("container timeout must be positive")
	}
	c.Container.Runtime = runtimeName
	c.Container.Socket = c.Container.GetSocket()
	c.Container.LSMMode = c.Container.GetLSMMode()
	c.Container.AppArmorProfile = c.Container.GetAppArmorProfile()
	c.Container.Image.BaseImage = c.Container.GetBaseImage()
	c.Container.Image.ContainerPath = c.Container.GetContainerPath()

	// 验证应用管理配置
	basePath := c.AppManage.GetBasePath()
	if !filepath.IsAbs(basePath) {
		absPath, err := filepath.Abs(basePath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for base_path: %w", err)
		}
		basePath = absPath
	}
	c.AppManage.AppDir.BasePath = filepath.Clean(basePath)

	return nil
}

// GetAppServerRequestTimeout 获取 app-server 请求处理超时时间
func (c *AppRuntimeConfig) GetAppServerRequestTimeout() int {
	if c.Timeouts.AppServerRequest <= 0 {
		return 30 // 默认 30 秒
	}
	return c.Timeouts.AppServerRequest
}

// GetContainerStartupTimeout 获取容器启动等待时间
func (c *AppRuntimeConfig) GetContainerStartupTimeout() int {
	if c.Timeouts.ContainerStartup <= 0 {
		return 2 // 默认 2 秒
	}
	return c.Timeouts.ContainerStartup
}

// GetAppStartupNotificationTimeout 获取应用启动通知等待时间
func (c *AppRuntimeConfig) GetAppStartupNotificationTimeout() int {
	if c.Timeouts.AppStartupNotification <= 0 {
		return 300 // 默认 300 秒
	}
	return c.Timeouts.AppStartupNotification
}

// GetContainerCleanupTimeout 获取容器清理等待时间
func (c *AppRuntimeConfig) GetContainerCleanupTimeout() int {
	if c.Timeouts.ContainerCleanup <= 0 {
		return 10 // 默认 10 秒
	}
	return c.Timeouts.ContainerCleanup
}

// GetRuntimeInstanceID 获取 runtime 实例 ID。
// 优先使用配置；未配置时回退为基于 hostname 的稳定 ID。
func (c *AppRuntimeConfig) GetRuntimeInstanceID() string {
	if id := strings.TrimSpace(c.Runtime.InstanceID); id != "" {
		return id
	}

	hostname, err := os.Hostname()
	if err == nil {
		hostname = strings.TrimSpace(hostname)
		if hostname != "" {
			hostname = strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(hostname)
			return "runtime-" + hostname
		}
	}

	return "runtime-local"
}

func (c *AppRuntimeConfig) GetListenHost() string {
	if c == nil {
		return normalizeListenHost("")
	}
	return normalizeListenHost(c.Runtime.ListenHost)
}

func (c *AppRuntimeConfig) GetContainerBaseImage() string {
	if c == nil {
		return (&ContainerServiceConfig{}).GetBaseImage()
	}
	return c.Container.GetBaseImage()
}

func (c *AppRuntimeConfig) GetContainerRuntime() string {
	if c == nil {
		return (&ContainerServiceConfig{}).GetRuntime()
	}
	return c.Container.GetRuntime()
}

func (c *AppRuntimeConfig) GetContainerSocket() string {
	if c == nil {
		return (&ContainerServiceConfig{}).GetSocket()
	}
	return c.Container.GetSocket()
}

func (c *AppRuntimeConfig) GetContainerPath() string {
	if c == nil {
		return (&ContainerServiceConfig{}).GetContainerPath()
	}
	return c.Container.GetContainerPath()
}

func (c *AppRuntimeConfig) GetAppDatabaseConfig() AppDatabaseConfig {
	if c == nil {
		return AppDatabaseConfig{}
	}
	return c.AppDatabase.WithDefaults()
}

func (c AppDatabaseConfig) WithDefaults() AppDatabaseConfig {
	if envEnabled := strings.TrimSpace(os.Getenv("KAGEOS_APP_DB_ENABLED")); envEnabled != "" {
		c.Enabled = strings.EqualFold(envEnabled, "true") || envEnabled == "1" || strings.EqualFold(envEnabled, "yes")
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_APP_DB_HOST")); v != "" {
		c.Host = v
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_APP_DB_PORT")); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Port = port
		}
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_APP_DB_ADMIN_USER")); v != "" {
		c.AdminUser = v
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_APP_DB_ADMIN_PASSWORD")); v != "" {
		c.AdminPassword = v
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_APP_DB_SECRET_KEY")); v != "" {
		c.SecretKey = v
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_APP_DB_CLUSTER_KEY")); v != "" {
		c.ClusterKey = v
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_APP_DB_GRANT_HOST")); v != "" {
		c.GrantHost = v
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_APP_DB_DATABASE_PREFIX")); v != "" {
		c.DatabasePrefix = v
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_APP_DB_USER_PREFIX")); v != "" {
		c.UserPrefix = v
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_APP_DB_MAX_OPEN_CONNS")); v != "" {
		if value, err := strconv.Atoi(v); err == nil {
			c.MaxOpenConns = value
		}
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_APP_DB_MAX_IDLE_CONNS")); v != "" {
		if value, err := strconv.Atoi(v); err == nil {
			c.MaxIdleConns = value
		}
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_APP_DB_MAX_IDLE_TIME")); v != "" {
		if value, err := strconv.Atoi(v); err == nil {
			c.MaxIdleTime = value
		}
	}
	if v := strings.TrimSpace(os.Getenv("KAGEOS_APP_DB_MAX_LIFETIME")); v != "" {
		if value, err := strconv.Atoi(v); err == nil {
			c.MaxLifetime = value
		}
	}

	if c.Dialect == "" {
		c.Dialect = defaultAppDatabaseDialect
	}
	if c.Host == "" {
		c.Host = defaultAppDatabaseHost
	}
	if c.Port <= 0 {
		c.Port = defaultAppDatabasePort
	}
	if c.GrantHost == "" {
		c.GrantHost = defaultAppDatabaseGrantHost
	}
	if c.ClusterKey == "" {
		c.ClusterKey = defaultAppDatabaseClusterKey(c.Host, c.Port)
	}
	if c.DatabasePrefix == "" {
		c.DatabasePrefix = defaultAppDatabaseNamePrefix
	}
	if c.UserPrefix == "" {
		c.UserPrefix = defaultAppDatabaseUserPrefix
	}
	if c.MaxOpenConns <= 0 {
		c.MaxOpenConns = defaultAppDatabaseMaxOpenConns
	}
	if c.MaxIdleConns < 0 {
		c.MaxIdleConns = defaultAppDatabaseMaxIdleConns
	}
	if c.MaxIdleTime <= 0 {
		c.MaxIdleTime = defaultAppDatabaseMaxIdleTime
	}
	if c.MaxLifetime <= 0 {
		c.MaxLifetime = defaultAppDatabaseMaxLifetime
	}
	return c
}

func defaultAppDatabaseClusterKey(host string, port int) string {
	value := fmt.Sprintf("%s:%d", strings.ToLower(strings.TrimSpace(host)), port)
	sum := sha256.Sum256([]byte(value))
	return "mysql_" + hex.EncodeToString(sum[:])[:12]
}

func (c *ContainerServiceConfig) GetRuntime() string {
	if c == nil {
		return defaultContainerRuntime
	}
	if v := strings.ToLower(strings.TrimSpace(c.Runtime)); v != "" {
		return v
	}
	return defaultContainerRuntime
}

func (c *ContainerServiceConfig) GetSocket() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.Socket)
}

func (c *ContainerServiceConfig) GetLSMMode() string {
	if c == nil {
		return defaultContainerLSMMode
	}
	if v := strings.ToLower(strings.TrimSpace(c.LSMMode)); v != "" {
		return v
	}
	return defaultContainerLSMMode
}

func (c *ContainerServiceConfig) GetAppArmorProfile() string {
	if c == nil {
		return defaultAppArmorProfile
	}
	if v := strings.TrimSpace(c.AppArmorProfile); v != "" {
		return v
	}
	return defaultAppArmorProfile
}

func (c *ContainerServiceConfig) GetBaseImage() string {
	if v := strings.TrimSpace(os.Getenv("KAGEOS_APP_BASE_IMAGE")); v != "" {
		return v
	}
	if c == nil {
		return defaultContainerBaseImage
	}
	if v := strings.TrimSpace(c.Image.BaseImage); v != "" {
		return v
	}
	return defaultContainerBaseImage
}

func (c *ContainerServiceConfig) GetContainerPath() string {
	if c == nil {
		return defaultContainerPath
	}
	if v := strings.TrimSpace(c.Image.ContainerPath); v != "" {
		return v
	}
	return defaultContainerPath
}

// loadYAMLConfig 加载 YAML 配置文件。
// 当前优先级：
//
//	dev  -> .kageos/dev/config/<file>           -> fallback: deploy/dev/config/<file>
//	prod -> .kageos/prod/generated/config/<file> -> fallback: deploy/prod/config/runtime/<file> -> deploy/prod/config/template/<file>
//
// 加载成功时打印实际使用的配置路径，避免糊涂账。
func loadYAMLConfig(filename string, config interface{}) error {
	configPath := findConfigFile(filename)
	if configPath == "" {
		return fmt.Errorf("config file not found: %s", filename)
	}
	absPath, _ := filepath.Abs(configPath)
	env := getConfigEnv()
	fmt.Printf("[Config] mode=%s  %s <- %s\n", env, filepath.Base(filename), absPath)

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

func configPathsForEnv(root, env, baseName string) []string {
	root = filepath.Clean(root)
	if env == "dev" {
		return []string{
			filepath.Join(root, ".kageos", "dev", "config", baseName),
			filepath.Join(root, "deploy", "dev", "config", baseName),
		}
	}
	return []string{
		filepath.Join(root, ".kageos", "prod", "generated", "config", baseName),
		filepath.Join(root, "deploy", "prod", "config", "runtime", baseName),
		filepath.Join(root, "deploy", "prod", "config", "template", baseName),
	}
}

// findConfigFile 查找配置文件：只使用官方结构。
func findConfigFile(filename string) string {
	env := getConfigEnv()
	baseName := filepath.Base(filename)

	tryPrefix := func(prefix string) string {
		for _, p := range configPathsForEnv(prefix, env, baseName) {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		return ""
	}

	// 0. 显式根目录
	if root := GetKageosRoot(); root != "" {
		if p := tryPrefix(root); p != "" {
			return p
		}
	}

	// 1. 相对 cwd 的常见层级
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		for _, rel := range []string{".", "..", "../..", "../../..", "../../../..", "../../../../.."} {
			var base string
			if rel == "." {
				base = cwd
			} else {
				base = filepath.Join(cwd, rel)
			}
			base, _ = filepath.Abs(base)
			if p := tryPrefix(base); p != "" {
				return p
			}
		}
	}

	// 2. 从 cwd 逐级向上，在每个祖先目录尝试（不依赖预先解析的根）
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		dir, _ := filepath.Abs(cwd)
		for dir != "" {
			if p := tryPrefix(dir); p != "" {
				return p
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	return ""
}
