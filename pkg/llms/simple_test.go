package llms

import (
	"context"
	"testing"
	"time"
)

// TestSimpleChat 测试简单的非流式聊天
func TestSimpleChat(t *testing.T) {
	// 测试GLM非流式功能
	t.Run("GLM_Simple_Chat", func(t *testing.T) {
		client := NewGLMClient("")

		req := &ChatRequest{
			Messages: []Message{
				{Role: "user", Content: "你好"},
			},
			MaxTokens:   50,
			Temperature: 0.7,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := client.Chat(ctx, req)
		if err != nil {
			t.Logf("GLM非流式请求错误: %v", err)
			return
		}

		if resp.Content == "" {
			t.Error("响应内容为空")
		}

		t.Logf("GLM响应: %s", resp.Content)
		if resp.Usage != nil {
			t.Logf("使用统计: %+v", resp.Usage)
		}
	})

	// 测试DeepSeek非流式功能
	t.Run("DeepSeek_Simple_Chat", func(t *testing.T) {
		client := NewDeepSeekClient("")

		req := &ChatRequest{
			Messages: []Message{
				{Role: "user", Content: "Hello"},
			},
			MaxTokens:   50,
			Temperature: 0.7,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := client.Chat(ctx, req)
		if err != nil {
			t.Logf("DeepSeek非流式请求错误: %v", err)
			return
		}

		if resp.Content == "" {
			t.Error("响应内容为空")
		}

		t.Logf("DeepSeek响应: %s", resp.Content)
		if resp.Usage != nil {
			t.Logf("使用统计: %+v", resp.Usage)
		}
	})

	// 测试Qwen非流式功能
	t.Run("Qwen_Simple_Chat", func(t *testing.T) {
		client := NewQwenClient("")

		req := &ChatRequest{
			Messages: []Message{
				{Role: "user", Content: "你好"},
			},
			MaxTokens:   50,
			Temperature: 0.7,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := client.Chat(ctx, req)
		if err != nil {
			t.Logf("Qwen非流式请求错误: %v", err)
			return
		}

		if resp.Content == "" {
			t.Error("响应内容为空")
		}

		t.Logf("Qwen响应: %s", resp.Content)
		if resp.Usage != nil {
			t.Logf("使用统计: %+v", resp.Usage)
		}
	})
}
