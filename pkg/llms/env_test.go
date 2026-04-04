package llms

import (
	"testing"
)

// TestEnvironmentVariableFallback 测试环境变量回退功能
func TestEnvironmentVariableFallback(t *testing.T) {
	// 测试GLM客户端
	t.Run("GLM_Environment_Fallback", func(t *testing.T) {
		testKey := "test-glm-key-from-env"
		t.Setenv("GLM_API_KEY", testKey)

		client := NewGLMClient("")
		if client.APIKey != testKey {
			t.Errorf("期望API密钥为 %s，实际为 %s", testKey, client.APIKey)
		}
	})

	// 测试DeepSeek客户端
	t.Run("DeepSeek_Environment_Fallback", func(t *testing.T) {
		testKey := "test-deepseek-key-from-env"
		t.Setenv("DEEPSEEK_API_KEY", testKey)

		client := NewDeepSeekClient("")
		if client.APIKey != testKey {
			t.Errorf("期望API密钥为 %s，实际为 %s", testKey, client.APIKey)
		}
	})

	// 测试Qwen客户端
	t.Run("Qwen_Environment_Fallback", func(t *testing.T) {
		testKey := "test-qwen-key-from-env"
		t.Setenv("QIANWEN_API_KEY", testKey)

		client := NewQwenClient("")
		if client.APIKey != testKey {
			t.Errorf("期望API密钥为 %s，实际为 %s", testKey, client.APIKey)
		}
	})

	// 测试优先级：传入的API密钥应该优先于环境变量
	t.Run("API_Key_Priority", func(t *testing.T) {
		envKey := "env-key"
		t.Setenv("GLM_API_KEY", envKey)

		passedKey := "passed-key"
		client := NewGLMClient(passedKey)
		if client.APIKey != passedKey {
			t.Errorf("期望API密钥为 %s，实际为 %s", passedKey, client.APIKey)
		}
	})

	// 测试环境变量为空的情况
	t.Run("Empty_Environment_Variable", func(t *testing.T) {
		t.Setenv("GLM_API_KEY", "")

		client := NewGLMClient("")
		if client.APIKey != "" {
			t.Errorf("期望API密钥为空，实际为 %s", client.APIKey)
		}
	})
}
