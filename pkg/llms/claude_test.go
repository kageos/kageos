package llms

import (
	"context"
	"testing"
	"time"
)

// TestClaudeClientCreation 测试Claude客户端创建
func TestClaudeClientCreation(t *testing.T) {
	client, err := NewClaudeClientFromEnv()
	if err != nil {
		t.Fatalf("Claude客户端创建失败: %v", err)
	}

	if client == nil {
		t.Fatal("Claude客户端创建失败")
	}

	// 验证基本信息
	if client.GetModelName() != "claude-sonnet-4-20250514" {
		t.Errorf("期望模型名称: claude-sonnet-4-20250514, 实际: %s", client.GetModelName())
	}

	// 检查提供商名称（允许大小写差异）
	provider := client.GetProvider()
	if provider != string(ProviderClaude) && provider != "Claude" {
		t.Errorf("期望提供商: %s 或 Claude, 实际: %s", ProviderClaude, provider)
	}
}

// TestClaudeSupportedModels 测试支持的模型列表
func TestClaudeSupportedModels(t *testing.T) {
	client, err := NewClaudeClientFromEnv()
	if err != nil {
		t.Fatalf("Claude客户端创建失败: %v", err)
	}

	// 使用类型断言来访问具体实现的方法
	if claudeClient, ok := client.(*ClaudeClient); ok {
		models := claudeClient.GetSupportedModels()

		if len(models) == 0 {
			t.Fatal("支持的模型列表为空")
		}

		// 检查默认模型是否在列表中
		found := false
		for _, model := range models {
			if model == "claude-sonnet-4-20250514" {
				found = true
				break
			}
		}

		if !found {
			t.Error("默认模型 claude-sonnet-4-20250514 不在支持的模型列表中")
		}
	} else {
		t.Skip("无法获取Claude客户端的具体实现")
	}
}

// TestClaudePricingInfo 测试价格信息
func TestClaudePricingInfo(t *testing.T) {
	client, err := NewClaudeClientFromEnv()
	if err != nil {
		t.Fatalf("Claude客户端创建失败: %v", err)
	}

	// 使用类型断言来访问具体实现的方法
	if claudeClient, ok := client.(*ClaudeClient); ok {
		pricing := claudeClient.GetPricingInfo()

		if len(pricing) == 0 {
			t.Fatal("价格信息为空")
		}

		// 检查关键价格信息
		if pricing["model"] != "claude-sonnet-4-20250514" {
			t.Errorf("期望模型: claude-sonnet-4-20250514, 实际: %v", pricing["model"])
		}
	} else {
		t.Skip("无法获取Claude客户端的具体实现")
	}
}

// TestClaudeChat 测试聊天功能
func TestClaudeChat(t *testing.T) {
	client, err := NewClaudeClientFromEnv()
	if err != nil {
		t.Fatalf("Claude客户端创建失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &ChatRequest{
		Messages: []Message{
			{
				Role:    "user",
				Content: "你好！请用一句话介绍你自己。",
			},
		},
		MaxTokens:   50,
		Temperature: 0.7,
	}

	resp, err := client.Chat(ctx, req)
	if err != nil {
		t.Fatalf("聊天请求失败: %v", err)
	}

	if resp.Content == "" {
		t.Fatal("响应内容为空")
	}

	t.Logf("Claude回复: %s", resp.Content)

	if resp.Usage != nil {
		t.Logf("Token使用: 输入=%d, 输出=%d, 总计=%d",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}
}

// TestClaudeCodeGeneration 测试代码生成功能
func TestClaudeCodeGeneration(t *testing.T) {
	client, err := NewClaudeClientFromEnv()
	if err != nil {
		t.Fatalf("Claude客户端创建失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req := &ChatRequest{
		Messages: []Message{
			{
				Role:    "user",
				Content: "请用Go语言写一个快速排序函数，包含测试用例。",
			},
		},
		MaxTokens:   500,
		Temperature: 0.3, // 降低温度以获得更稳定的代码
	}

	t.Log("正在生成代码...")
	resp, err := client.Chat(ctx, req)
	if err != nil {
		t.Fatalf("代码生成失败: %v", err)
	}

	if resp.Content == "" {
		t.Fatal("生成的代码为空")
	}

	// 检查是否包含Go代码的关键元素
	content := resp.Content
	if !containsGoCodeClaude(content) {
		t.Logf("生成的代码可能不是标准的Go代码格式: %s", content[:100])
	}

	t.Logf("生成的代码长度: %d 字符", len(content))
	t.Logf("代码预览: %s", content[:200])

	if resp.Usage != nil {
		t.Logf("Token使用: 输入=%d, 输出=%d, 总计=%d",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}
}

// containsGoCodeClaude 检查内容是否包含Go代码的关键元素
func containsGoCodeClaude(content string) bool {
	goKeywords := []string{"func ", "package ", "import ", "return ", "if ", "for ", "range "}
	for _, keyword := range goKeywords {
		if contains(content, keyword) {
			return true
		}
	}
	return false
}

// contains 检查字符串是否包含子字符串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr)))
}

// containsSubstring 检查字符串中间是否包含子字符串
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
