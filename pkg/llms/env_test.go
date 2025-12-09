package llms

import (
	"os"
	"testing"
)

// TestEnvironmentVariableFallback 测试环境变量回退功能
func TestEnvironmentVariableFallback(t *testing.T) {
	// 保存原始环境变量
	originalGLM := os.Getenv("GLM_API_KEY")
	originalDeepSeek := os.Getenv("DEEPSEEK_API_KEY")
	originalQwen := os.Getenv("QWEN_API_KEY")

	// 清理环境变量
	defer func() {
		if originalGLM != "" {
			os.Setenv("GLM_API_KEY", originalGLM)
		} else {
			os.Unsetenv("GLM_API_KEY")
		}
		if originalDeepSeek != "" {
			os.Setenv("DEEPSEEK_API_KEY", originalDeepSeek)
		} else {
			os.Unsetenv("DEEPSEEK_API_KEY")
		}
		if originalQwen != "" {
			os.Setenv("QWEN_API_KEY", originalQwen)
		} else {
			os.Unsetenv("QWEN_API_KEY")
		}
	}()

	// 测试GLM客户端
	t.Run("GLM_Environment_Fallback", func(t *testing.T) {
		// 设置测试环境变量
		testKey := "test-glm-key-from-env"
		os.Setenv("GLM_API_KEY", testKey)

		// 使用空字符串创建客户端，应该从环境变量获取
		client := NewGLMClient("")
		if client.APIKey != testKey {
			t.Errorf("期望API密钥为 %s，实际为 %s", testKey, client.APIKey)
		}
	})

	// 测试DeepSeek客户端
	t.Run("DeepSeek_Environment_Fallback", func(t *testing.T) {
		// 设置测试环境变量
		testKey := "test-deepseek-key-from-env"
		os.Setenv("DEEPSEEK_API_KEY", testKey)

		// 使用空字符串创建客户端，应该从环境变量获取
		client := NewDeepSeekClient("")
		if client.APIKey != testKey {
			t.Errorf("期望API密钥为 %s，实际为 %s", testKey, client.APIKey)
		}
	})

	// 测试Qwen客户端
	t.Run("Qwen_Environment_Fallback", func(t *testing.T) {
		// 设置测试环境变量
		testKey := "test-qwen-key-from-env"
		os.Setenv("QWEN_API_KEY", testKey)

		// 使用空字符串创建客户端，应该从环境变量获取
		client := NewQwenClient("")
		if client.APIKey != testKey {
			t.Errorf("期望API密钥为 %s，实际为 %s", testKey, client.APIKey)
		}
	})

	// 测试优先级：传入的API密钥应该优先于环境变量
	t.Run("API_Key_Priority", func(t *testing.T) {
		// 设置环境变量
		envKey := "env-key"
		os.Setenv("GLM_API_KEY", envKey)

		// 传入的API密钥应该优先
		passedKey := "passed-key"
		client := NewGLMClient(passedKey)
		if client.APIKey != passedKey {
			t.Errorf("期望API密钥为 %s，实际为 %s", passedKey, client.APIKey)
		}
	})

	// 测试环境变量为空的情况
	t.Run("Empty_Environment_Variable", func(t *testing.T) {
		// 确保环境变量为空
		os.Unsetenv("GLM_API_KEY")

		// 使用空字符串创建客户端
		client := NewGLMClient("")
		if client.APIKey != "" {
			t.Errorf("期望API密钥为空，实际为 %s", client.APIKey)
		}
	})
}
