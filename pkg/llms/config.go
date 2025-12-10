package llms

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Config LLM配置结构
type Config struct {
	Providers map[Provider]ProviderConfig `json:"providers"`
	Default   Provider                    `json:"default"`
}

// ProviderConfig 单个提供商配置
type ProviderConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url,omitempty"`
	Timeout int    `json:"timeout,omitempty"` // 超时时间（秒）
}

// GlobalConfig 全局配置实例
var (
	globalConfig *Config
	configMutex  sync.RWMutex
)

// LoadConfig 从文件加载配置
func LoadConfig(configPath string) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	globalConfig = &config
	return nil
}

// GetConfig 获取全局配置
func GetConfig() *Config {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return globalConfig
}

// GetProviderConfig 获取指定提供商的配置
func GetProviderConfig(provider Provider) (*ProviderConfig, error) {
	config := GetConfig()
	if config == nil {
		return nil, fmt.Errorf("配置未加载")
	}

	providerConfig, exists := config.Providers[provider]
	if !exists {
		return nil, fmt.Errorf("提供商 %s 的配置不存在", provider)
	}

	return &providerConfig, nil
}

// GetDefaultProvider 获取默认提供商
func GetDefaultProvider() Provider {
	config := GetConfig()
	if config == nil {
		return ProviderDeepSeek // 默认使用DeepSeek
	}
	return config.Default
}

// CreateClientFromConfig 从配置创建客户端
func CreateClientFromConfig(provider Provider) (LLMClient, error) {
	providerConfig, err := GetProviderConfig(provider)
	if err != nil {
		return nil, err
	}

	// 🎯 修复：将配置文件中的秒转换为time.Duration
	options := DefaultClientOptions()
	if providerConfig.Timeout > 0 {
		options.Timeout = time.Duration(providerConfig.Timeout) * time.Second
	}
	if providerConfig.BaseURL != "" {
		options.BaseURL = providerConfig.BaseURL
	}

	client, err := NewLLMClientWithOptions(provider, providerConfig.APIKey, options)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// CreateDefaultClient 创建默认客户端
func CreateDefaultClient() (LLMClient, error) {
	defaultProvider := GetDefaultProvider()
	return CreateClientFromConfig(defaultProvider)
}

// SaveConfig 保存配置到文件
func SaveConfig(configPath string) error {
	configMutex.RLock()
	defer configMutex.RUnlock()

	if globalConfig == nil {
		return fmt.Errorf("没有配置可保存")
	}

	data, err := json.MarshalIndent(globalConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	return nil
}

// UpdateProviderConfig 更新提供商配置
func UpdateProviderConfig(provider Provider, config ProviderConfig) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	if globalConfig == nil {
		globalConfig = &Config{
			Providers: make(map[Provider]ProviderConfig),
		}
	}

	globalConfig.Providers[provider] = config
	return nil
}

// SetDefaultProvider 设置默认提供商
func SetDefaultProvider(provider Provider) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	if globalConfig == nil {
		globalConfig = &Config{
			Providers: make(map[Provider]ProviderConfig),
		}
	}

	globalConfig.Default = provider
	return nil
}
