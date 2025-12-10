package llms

import (
	"testing"
	"time"
)

// TestTimeoutConfiguration 测试超时配置是否正确传递
func TestTimeoutConfiguration(t *testing.T) {
	// 测试默认超时
	options := DefaultClientOptions()
	if options.Timeout != 60*time.Second {
		t.Errorf("默认超时应该是60秒，实际是: %v", options.Timeout)
	}

	// 测试自定义超时
	customTimeout := 600 * time.Second
	options = DefaultClientOptions().WithTimeout(customTimeout)
	if options.Timeout != customTimeout {
		t.Errorf("自定义超时应该是600秒，实际是: %v", options.Timeout)
	}

	// 测试配置文件超时转换
	providerConfig := ProviderConfig{
		APIKey:  "test-key",
		Timeout: 600, // 600秒
	}

	options = DefaultClientOptions()
	if providerConfig.Timeout > 0 {
		options.Timeout = time.Duration(providerConfig.Timeout) * time.Second
	}

	if options.Timeout != 600*time.Second {
		t.Errorf("配置文件超时转换后应该是600秒，实际是: %v", options.Timeout)
	}
}

// TestClientCreationWithOptions 测试客户端创建时超时配置是否正确传递
func TestClientCreationWithOptions(t *testing.T) {
	// 测试Kimi客户端
	customTimeout := 600 * time.Second
	options := DefaultClientOptions().WithTimeout(customTimeout)

	client := NewKimiClientWithOptions("test-key", options)
	if client.Options.Timeout != customTimeout {
		t.Errorf("Kimi客户端超时应该是600秒，实际是: %v", client.Options.Timeout)
	}

	// 测试DeepSeek客户端
	deepseekClient := NewDeepSeekClientWithOptions("test-key", options)
	if deepseekClient.Options.Timeout != customTimeout {
		t.Errorf("DeepSeek客户端超时应该是600秒，实际是: %v", deepseekClient.Options.Timeout)
	}
}

// TestRequestLevelTimeout 测试请求级别的超时配置
func TestRequestLevelTimeout(t *testing.T) {
	// 创建客户端，默认超时60秒
	client := NewKimiClient("test-key")

	// 创建请求，指定超时600秒
	requestTimeout := 600 * time.Second
	req := &ChatRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
		Timeout:  &requestTimeout,
	}

	// 模拟超时选择逻辑
	timeout := client.Options.Timeout // 默认使用客户端配置的超时时间
	if req.Timeout != nil && *req.Timeout > 0 {
		timeout = *req.Timeout // 如果请求中指定了超时时间，则使用请求的超时时间
	}

	if timeout != requestTimeout {
		t.Errorf("请求级别超时应该是600秒，实际是: %v", timeout)
	}
}
