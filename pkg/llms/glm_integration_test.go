package llms

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestGLMIntegration 测试GLM集成功能
func TestGLMIntegration(t *testing.T) {
	// 检查环境变量
	apiKey := os.Getenv("GLM_API_KEY")
	if apiKey == "" {
		t.Skip("跳过GLM集成测试：未设置GLM_API_KEY环境变量")
	}

	// 创建GLM客户端
	client, err := NewGLMClientFromEnv()
	if err != nil {
		t.Fatalf("创建GLM客户端失败: %v", err)
	}

	// 测试基本功能
	t.Run("基本聊天功能", func(t *testing.T) {
		req := &ChatRequest{
			Messages: []Message{
				{Role: "user", Content: "你好，请简单介绍一下你自己。"},
			},
			MaxTokens:   500,
			Temperature: 0.7,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := client.Chat(ctx, req)
		if err != nil {
			t.Fatalf("聊天调用失败: %v", err)
		}

		if resp.Error != "" {
			t.Fatalf("API返回错误: %s", resp.Error)
		}

		if resp.Content == "" {
			t.Fatal("返回内容为空")
		}

		fmt.Printf("GLM回复: %s\n", resp.Content)
		if resp.Usage != nil {
			fmt.Printf("Token使用: 输入=%d, 输出=%d, 总计=%d\n",
				resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
		}
	})

	// 测试GLM特殊功能
	t.Run("GLM特殊功能", func(t *testing.T) {
		glmClient, ok := client.(*GLMClient)
		if !ok {
			t.Fatal("客户端类型转换失败")
		}

		// 测试模型切换
		originalModel := glmClient.GetModelName()
		glmClient.SetModel("glm-4.5-air")
		if glmClient.GetModelName() != "glm-4.5-air" {
			t.Errorf("期望模型为 glm-4.5-air，实际为 %s", glmClient.GetModelName())
		}
		glmClient.SetModel(originalModel) // 恢复原模型

		// 测试支持的模型列表
		models := glmClient.GetSupportedModels()
		if len(models) == 0 {
			t.Fatal("支持的模型列表为空")
		}
		fmt.Printf("支持的模型: %v\n", models)

		// 测试思考模式支持
		if !glmClient.IsThinkingEnabled() {
			t.Error("GLM-4.5系列应该支持思考模式")
		}
	})

	// 测试思考模式
	t.Run("思考模式", func(t *testing.T) {
		glmClient, ok := client.(*GLMClient)
		if !ok {
			t.Fatal("客户端类型转换失败")
		}

		req := &ChatRequest{
			Messages: []Message{
				{Role: "user", Content: "请分析一下Go语言和Python语言在并发处理方面的区别。"},
			},
			MaxTokens:   1000,
			Temperature: 0.7,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// 使用思考模式
		resp, err := glmClient.ChatWithThinking(ctx, req, true)
		if err != nil {
			t.Fatalf("思考模式调用失败: %v", err)
		}

		if resp.Error != "" {
			t.Fatalf("思考模式API返回错误: %s", resp.Error)
		}

		if resp.Content == "" {
			t.Fatal("思考模式返回内容为空")
		}

		fmt.Printf("思考模式回复: %s\n", resp.Content)
		if resp.Usage != nil {
			fmt.Printf("思考模式Token使用: 输入=%d, 输出=%d, 总计=%d\n",
				resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
		}
	})

	// 测试不同模型
	t.Run("不同模型对比", func(t *testing.T) {
		glmClient, ok := client.(*GLMClient)
		if !ok {
			t.Fatal("客户端类型转换失败")
		}

		models := []string{"glm-4.5", "glm-4.5-air", "glm-4.5-flash"}

		for _, model := range models {
			t.Run(fmt.Sprintf("模型_%s", model), func(t *testing.T) {
				glmClient.SetModel(model)

				req := &ChatRequest{
					Messages: []Message{
						{Role: "user", Content: "请用一句话介绍Go语言的特点。"},
					},
					MaxTokens:   100,
					Temperature: 0.7,
				}

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				resp, err := glmClient.Chat(ctx, req)
				if err != nil {
					t.Errorf("模型 %s 调用失败: %v", model, err)
					return
				}

				if resp.Error != "" {
					t.Errorf("模型 %s API返回错误: %s", model, resp.Error)
					return
				}

				fmt.Printf("模型 %s 回复: %s\n", model, resp.Content)
				if resp.Usage != nil {
					fmt.Printf("模型 %s Token使用: %d\n", model, resp.Usage.TotalTokens)
				}
			})
		}
	})
}

// TestGLMFactoryIntegrationReal 测试工厂模式集成（真实API调用）
func TestGLMFactoryIntegrationReal(t *testing.T) {
	// 检查环境变量
	apiKey := os.Getenv("GLM_API_KEY")
	if apiKey == "" {
		t.Skip("跳过GLM工厂集成测试：未设置GLM_API_KEY环境变量")
	}

	// 通过工厂创建客户端
	client, err := NewLLMClient(ProviderGLM, "")
	if err != nil {
		t.Fatalf("通过工厂创建GLM客户端失败: %v", err)
	}

	// 验证客户端类型
	glmClient, ok := client.(*GLMClient)
	if !ok {
		t.Fatal("工厂返回的客户端类型不是GLMClient")
	}

	// 验证提供商
	if client.GetProvider() != "GLM" {
		t.Errorf("期望提供商为 GLM，实际为 %s", client.GetProvider())
	}

	// 验证模型
	if glmClient.GetModelName() != "glm-4.5" {
		t.Errorf("期望默认模型为 glm-4.5，实际为 %s", glmClient.GetModelName())
	}

	fmt.Printf("工厂创建成功 - 提供商: %s, 模型: %s\n",
		client.GetProvider(), glmClient.GetModelName())
}

// TestGLMProviderConstantsReal 测试提供商常量（真实API调用）
func TestGLMProviderConstantsReal(t *testing.T) {
	// 测试提供商常量
	if ProviderGLM != "glm" {
		t.Errorf("期望ProviderGLM为 'glm'，实际为 %s", ProviderGLM)
	}

	// 测试提供商列表包含GLM
	providers := GetSupportedProviders()
	found := false
	for _, provider := range providers {
		if provider == ProviderGLM {
			found = true
			break
		}
	}
	if !found {
		t.Error("期望支持的提供商列表包含GLM")
	}

	// 测试显示名称
	displayName := GetProviderDisplayName(ProviderGLM)
	if displayName != "GLM" {
		t.Errorf("期望GLM显示名称为 'GLM'，实际为 %s", displayName)
	}

	fmt.Printf("提供商常量测试通过 - 常量: %s, 显示名称: %s\n", ProviderGLM, displayName)
}
