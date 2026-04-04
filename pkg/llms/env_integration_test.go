//go:build integration
// +build integration

package llms

import (
	"context"
	"testing"
	"time"
)

// TestEnvironmentVariableIntegration 测试环境变量集成的完整功能
func TestEnvironmentVariableIntegration(t *testing.T) {
	t.Run("测试千问3 Coder环境变量代码生成", func(t *testing.T) {
		// 从环境变量创建客户端
		client, err := NewQwen3CoderClientFromEnv()
		if err != nil {
			t.Fatalf("从环境变量创建千问3 Coder客户端失败: %v", err)
		}

		// 测试代码生成
		req := &ChatRequest{
			Messages: []Message{
				{
					Role:    "system",
					Content: "你是一个专业的Go语言开发助手，请生成简洁的代码示例",
				},
				{
					Role:    "user",
					Content: "请用Go语言编写一个简单的Hello World函数",
				},
			},
			MaxTokens:   500,
			Temperature: 0.1,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := client.Chat(ctx, req)
		if err != nil {
			t.Fatalf("代码生成失败: %v", err)
		}

		if resp.Error != "" {
			t.Logf("API返回错误: %s", resp.Error)
			return
		}

		if resp.Content == "" {
			t.Error("响应内容为空")
			return
		}

		t.Logf("✅ 千问3 Coder环境变量代码生成成功！")
		t.Logf("📊 Token使用: 输入%d, 输出%d, 总计%d",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)

		// 显示生成的代码（截取前200字符）
		content := resp.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		t.Logf("📄 生成的代码: %s", content)
	})

	t.Run("测试千问环境变量代码生成", func(t *testing.T) {
		// 从环境变量创建客户端
		client, err := NewQwenClientFromEnv()
		if err != nil {
			t.Fatalf("从环境变量创建千问客户端失败: %v", err)
		}

		// 测试代码生成
		req := &ChatRequest{
			Messages: []Message{
				{
					Role:    "system",
					Content: "你是一个专业的Go语言开发助手，请生成简洁的代码示例",
				},
				{
					Role:    "user",
					Content: "请用Go语言编写一个简单的Hello World函数",
				},
			},
			MaxTokens:   500,
			Temperature: 0.1,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := client.Chat(ctx, req)
		if err != nil {
			t.Fatalf("代码生成失败: %v", err)
		}

		if resp.Error != "" {
			t.Logf("API返回错误: %s", resp.Error)
			return
		}

		if resp.Content == "" {
			t.Error("响应内容为空")
			return
		}

		t.Logf("✅ 千问环境变量代码生成成功！")
		t.Logf("📊 Token使用: 输入%d, 输出%d, 总计%d",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)

		// 显示生成的代码（截取前200字符）
		content := resp.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		t.Logf("📄 生成的代码: %s", content)
	})

	t.Run("测试DeepSeek环境变量代码生成", func(t *testing.T) {
		// 从环境变量创建客户端
		client, err := NewDeepSeekClientFromEnv()
		if err != nil {
			t.Fatalf("从环境变量创建DeepSeek客户端失败: %v", err)
		}

		// 测试代码生成
		req := &ChatRequest{
			Messages: []Message{
				{
					Role:    "system",
					Content: "你是一个专业的Go语言开发助手，请生成简洁的代码示例",
				},
				{
					Role:    "user",
					Content: "请用Go语言编写一个简单的Hello World函数",
				},
			},
			MaxTokens:   500,
			Temperature: 0.1,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := client.Chat(ctx, req)
		if err != nil {
			t.Fatalf("代码生成失败: %v", err)
		}

		if resp.Error != "" {
			t.Logf("API返回错误: %s", resp.Error)
			return
		}

		if resp.Content == "" {
			t.Error("响应内容为空")
			return
		}

		t.Logf("✅ DeepSeek环境变量代码生成成功！")
		t.Logf("📊 Token使用: 输入%d, 输出%d, 总计%d",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)

		// 显示生成的代码（截取前200字符）
		content := resp.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		t.Logf("📄 生成的代码: %s", content)
	})
}
