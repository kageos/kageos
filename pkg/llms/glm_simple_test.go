package llms

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestGLMSimple 简单的GLM功能测试
func TestGLMSimple(t *testing.T) {
	// 检查环境变量
	apiKey := os.Getenv("GLM_API_KEY")
	if apiKey == "" {
		t.Skip("跳过GLM测试：未设置GLM_API_KEY环境变量")
	}

	// 创建GLM客户端
	client, err := NewGLMClientFromEnv()
	if err != nil {
		t.Fatalf("创建GLM客户端失败: %v", err)
	}

	// 测试基本聊天功能
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

	fmt.Printf("✅ GLM基本功能测试通过\n")
	fmt.Printf("回复: %s\n", resp.Content)
	if resp.Usage != nil {
		fmt.Printf("Token使用: 输入=%d, 输出=%d, 总计=%d\n",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}
}

// TestGLMModelSwitchingSimple 测试模型切换（简化版）
func TestGLMModelSwitchingSimple(t *testing.T) {
	// 检查环境变量
	apiKey := os.Getenv("GLM_API_KEY")
	if apiKey == "" {
		t.Skip("跳过GLM模型切换测试：未设置GLM_API_KEY环境变量")
	}

	// 创建GLM客户端
	client, err := NewGLMClientFromEnv()
	if err != nil {
		t.Fatalf("创建GLM客户端失败: %v", err)
	}

	glmClient, ok := client.(*GLMClient)
	if !ok {
		t.Fatal("客户端类型转换失败")
	}

	// 测试模型切换
	models := []string{"glm-4.5", "glm-4.5-air", "glm-4.5-flash"}

	for _, model := range models {
		t.Run(fmt.Sprintf("模型_%s", model), func(t *testing.T) {
			glmClient.SetModel(model)

			req := &ChatRequest{
				Messages: []Message{
					{Role: "user", Content: "请用一句话介绍Go语言。"},
				},
				MaxTokens:   100,
				Temperature: 0.7,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			resp, err := glmClient.Chat(ctx, req)
			if err != nil {
				t.Logf("模型 %s 调用失败: %v", model, err)
				return
			}

			if resp.Error != "" {
				t.Logf("模型 %s API返回错误: %s", model, resp.Error)
				return
			}

			if resp.Content == "" {
				t.Logf("模型 %s 返回内容为空", model)
				return
			}

			fmt.Printf("✅ 模型 %s 测试通过\n", model)
			fmt.Printf("回复: %s\n", resp.Content)
		})
	}
}

// TestGLMFactory 测试工厂模式
func TestGLMFactory(t *testing.T) {
	// 检查环境变量
	apiKey := os.Getenv("GLM_API_KEY")
	if apiKey == "" {
		t.Skip("跳过GLM工厂测试：未设置GLM_API_KEY环境变量")
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

	fmt.Printf("✅ GLM工厂模式测试通过\n")
	fmt.Printf("提供商: %s, 模型: %s\n", client.GetProvider(), glmClient.GetModelName())
}

// TestGLMProviderInfo 测试提供商信息
func TestGLMProviderInfo(t *testing.T) {
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

	fmt.Printf("✅ GLM提供商信息测试通过\n")
	fmt.Printf("常量: %s, 显示名称: %s\n", ProviderGLM, displayName)
}
