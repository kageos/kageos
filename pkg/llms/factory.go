package llms

import (
	"fmt"
	"os"
)

// Provider 提供商类型
type Provider string

const (
	ProviderDeepSeek   Provider = "deepseek"
	ProviderQwen       Provider = "qwen"
	ProviderQwen3Coder Provider = "qwen3-coder"
	ProviderDouBao     Provider = "doubao"
	ProviderKimi       Provider = "kimi"
	ProviderClaude     Provider = "claude"
	ProviderGemini     Provider = "gemini"
	ProviderGLM        Provider = "glm"
)

// NewLLMClient 创建LLM客户端
func NewLLMClient(provider Provider, apiKey string) (LLMClient, error) {
	// 如果API Key为空，尝试从环境变量获取
	if apiKey == "" {
		apiKey = getAPIKeyFromEnv(provider)
		if apiKey == "" {
			return nil, fmt.Errorf("未提供API Key且环境变量中未找到 %s 的配置", provider)
		}
	}

	return NewLLMClientWithOptions(provider, apiKey, DefaultClientOptions())
}

// NewLLMClientWithOptions 创建带自定义配置的LLM客户端
func NewLLMClientWithOptions(provider Provider, apiKey string, options *ClientOptions) (LLMClient, error) {
	// 如果API Key为空，尝试从环境变量获取
	if apiKey == "" {
		apiKey = getAPIKeyFromEnv(provider)
		if apiKey == "" {
			return nil, fmt.Errorf("未提供API Key且环境变量中未找到 %s 的配置", provider)
		}
	}

	// 如果options为空，使用默认配置
	if options == nil {
		options = DefaultClientOptions()
	}

	switch provider {
	case ProviderDeepSeek:
		return NewDeepSeekClientWithOptions(apiKey, options), nil
	case ProviderQwen:
		return NewQwenClientWithOptions(apiKey, options), nil
	case ProviderQwen3Coder:
		return NewQwen3CoderClientWithOptions(apiKey, options), nil
	case ProviderDouBao:
		return NewDouBaoClientWithOptions(apiKey, options), nil
	case ProviderKimi:
		return NewKimiClientWithOptions(apiKey, options), nil
	case ProviderClaude:
		return NewClaudeClientWithOptions(apiKey, options), nil
	case ProviderGemini:
		return NewGeminiClientWithOptions(apiKey, options), nil
	case ProviderGLM:
		return NewGLMClientWithOptions(apiKey, options), nil
	default:
		return nil, fmt.Errorf("不支持的提供商: %s", provider)
	}
}

// getAPIKeyFromEnv 从环境变量获取API Key
func getAPIKeyFromEnv(provider Provider) string {
	switch provider {
	case ProviderDeepSeek:
		return os.Getenv("DEEPSEEK_API_KEY")
	case ProviderQwen, ProviderQwen3Coder:
		// 千问和千问3 Coder使用同一个API Key
		return os.Getenv("QIANWEN_API_KEY")
	case ProviderDouBao:
		return os.Getenv("DOUBAO_API_KEY")
	case ProviderKimi:
		return os.Getenv("KIMI_API_KEY")
	case ProviderClaude:
		return os.Getenv("CLAUDE_API_KEY")
	case ProviderGemini:
		return os.Getenv("GEMINI_API_KEY")
	case ProviderGLM:
		return os.Getenv("GLM_API_KEY")
	default:
		return ""
	}
}

// NewLLMClientFromEnv 从环境变量创建LLM客户端（推荐使用）
func NewLLMClientFromEnv(provider Provider) (LLMClient, error) {
	return NewLLMClient(provider, "")
}

// NewQwenClientFromEnv 从环境变量创建千问客户端
func NewQwenClientFromEnv() (LLMClient, error) {
	return NewLLMClient(ProviderQwen, "")
}

// NewQwen3CoderClientFromEnv 从环境变量创建千问3 Coder客户端
func NewQwen3CoderClientFromEnv() (LLMClient, error) {
	return NewLLMClient(ProviderQwen3Coder, "")
}

// NewDeepSeekClientFromEnv 从环境变量创建DeepSeek客户端
func NewDeepSeekClientFromEnv() (LLMClient, error) {
	return NewLLMClient(ProviderDeepSeek, "")
}

// NewKimiClientFromEnv 从环境变量创建Kimi客户端
func NewKimiClientFromEnv() (LLMClient, error) {
	return NewLLMClient(ProviderKimi, "")
}

// NewDouBaoClientFromEnv 从环境变量创建豆包客户端
func NewDouBaoClientFromEnv() (LLMClient, error) {
	return NewLLMClient(ProviderDouBao, "")
}

// NewClaudeClientFromEnv 从环境变量创建Claude客户端
func NewClaudeClientFromEnv() (LLMClient, error) {
	return NewLLMClient(ProviderClaude, "")
}

// NewGeminiClientFromEnv 从环境变量创建Gemini客户端
func NewGeminiClientFromEnv() (LLMClient, error) {
	return NewLLMClient(ProviderGemini, "")
}

// NewGLMClientFromEnv 从环境变量创建GLM客户端
func NewGLMClientFromEnv() (LLMClient, error) {
	return NewLLMClient(ProviderGLM, "")
}

// GetSupportedProviders 获取支持的提供商列表
func GetSupportedProviders() []Provider {
	return []Provider{
		ProviderDeepSeek,
		ProviderQwen,
		ProviderQwen3Coder,
		ProviderDouBao,
		ProviderKimi,
		ProviderClaude,
		ProviderGemini,
		ProviderGLM,
	}
}

// GetProviderDisplayName 获取提供商显示名称
func GetProviderDisplayName(provider Provider) string {
	switch provider {
	case ProviderDeepSeek:
		return "DeepSeek"
	case ProviderQwen:
		return "千问"
	case ProviderQwen3Coder:
		return "千问3-Coder"
	case ProviderDouBao:
		return "豆包"
	case ProviderKimi:
		return "Kimi"
	case ProviderClaude:
		return "Claude"
	case ProviderGemini:
		return "Gemini"
	case ProviderGLM:
		return "GLM"
	default:
		return string(provider)
	}
}
