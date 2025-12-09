package llms

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestChatStream 测试流式聊天功能
func TestChatStream(t *testing.T) {
	// 测试GLM流式功能
	t.Run("GLM_Stream", func(t *testing.T) {
		// 使用空字符串，让客户端自动从环境变量获取API密钥
		client := NewGLMClient("")

		req := &ChatRequest{
			Messages: []Message{
				{Role: "user", Content: "你好，请简单介绍一下自己"},
			},
			MaxTokens:   100,
			Temperature: 0.7,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		stream, err := client.ChatStream(ctx, req)
		if err != nil {
			t.Fatalf("创建流式请求失败: %v", err)
		}

		// 收集流式响应
		var content string
		var done bool

		for chunk := range stream {
			if chunk.Error != "" {
				t.Logf("流式响应错误: %s", chunk.Error)
				// 对于测试环境，错误是预期的（没有真实的API密钥）
				break
			}

			if chunk.Content != "" {
				content += chunk.Content
				t.Logf("收到内容片段: %s", chunk.Content)
			}

			if chunk.Done {
				done = true
				if chunk.Usage != nil {
					t.Logf("使用统计: %+v", chunk.Usage)
				}
				break
			}
		}

		if !done {
			t.Error("流式响应未正常结束")
		}

		t.Logf("完整内容: %s", content)
	})

	// 测试DeepSeek流式功能
	t.Run("DeepSeek_Stream", func(t *testing.T) {
		// 使用空字符串，让客户端自动从环境变量获取API密钥
		client := NewDeepSeekClient("")

		req := &ChatRequest{
			Messages: []Message{
				{Role: "user", Content: "Hello, please introduce yourself briefly"},
			},
			MaxTokens:   100,
			Temperature: 0.7,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		stream, err := client.ChatStream(ctx, req)
		if err != nil {
			t.Fatalf("创建流式请求失败: %v", err)
		}

		// 收集流式响应
		var content string
		var done bool

		for chunk := range stream {
			if chunk.Error != "" {
				t.Logf("流式响应错误: %s", chunk.Error)
				// 对于测试环境，错误是预期的（没有真实的API密钥）
				break
			}

			if chunk.Content != "" {
				content += chunk.Content
				t.Logf("收到内容片段: %s", chunk.Content)
			}

			if chunk.Done {
				done = true
				if chunk.Usage != nil {
					t.Logf("使用统计: %+v", chunk.Usage)
				}
				break
			}
		}

		if !done {
			t.Error("流式响应未正常结束")
		}

		t.Logf("完整内容: %s", content)
	})

	// 测试千问流式功能
	t.Run("Qwen_Stream", func(t *testing.T) {
		// 使用空字符串，让客户端自动从环境变量获取API密钥
		client := NewQwenClient("")

		req := &ChatRequest{
			Messages: []Message{
				{Role: "user", Content: "你好，请简单介绍一下自己"},
			},
			MaxTokens:   100,
			Temperature: 0.7,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		stream, err := client.ChatStream(ctx, req)
		if err != nil {
			t.Fatalf("创建流式请求失败: %v", err)
		}

		// 收集流式响应
		var content string
		var done bool

		for chunk := range stream {
			if chunk.Error != "" {
				t.Logf("流式响应错误: %s", chunk.Error)
				// 对于测试环境，错误是预期的（没有真实的API密钥）
				break
			}

			if chunk.Content != "" {
				content += chunk.Content
				t.Logf("收到内容片段: %s", chunk.Content)
			}

			if chunk.Done {
				done = true
				if chunk.Usage != nil {
					t.Logf("使用统计: %+v", chunk.Usage)
				}
				break
			}
		}

		if !done {
			t.Error("流式响应未正常结束")
		}

		t.Logf("完整内容: %s", content)
	})

	// 测试不支持流式的客户端
	t.Run("Claude_Stream_Not_Supported", func(t *testing.T) {
		client := NewClaudeClient("test-api-key")

		req := &ChatRequest{
			Messages: []Message{
				{Role: "user", Content: "Hello"},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		stream, err := client.ChatStream(ctx, req)
		if err != nil {
			t.Fatalf("创建流式请求失败: %v", err)
		}

		// 应该收到不支持的错误
		chunk := <-stream
		if chunk.Error == "" {
			t.Error("应该返回不支持流式的错误")
		}

		if !chunk.Done {
			t.Error("应该标记为完成")
		}

		t.Logf("预期的错误信息: %s", chunk.Error)
	})
}

// TestStreamChunk 测试流式数据块结构
func TestStreamChunk(t *testing.T) {
	// 测试内容片段
	contentChunk := &StreamChunk{
		Content: "Hello",
		Done:    false,
	}

	if contentChunk.Content != "Hello" {
		t.Error("内容片段内容不正确")
	}

	if contentChunk.Done {
		t.Error("内容片段不应该标记为完成")
	}

	// 测试完成片段
	doneChunk := &StreamChunk{
		Content: "",
		Done:    true,
		Usage: &Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	if !doneChunk.Done {
		t.Error("完成片段应该标记为完成")
	}

	if doneChunk.Usage == nil {
		t.Error("完成片段应该包含使用统计")
	}

	// 测试错误片段
	errorChunk := &StreamChunk{
		Error: "API错误",
		Done:  true,
	}

	if errorChunk.Error == "" {
		t.Error("错误片段应该包含错误信息")
	}

	if !errorChunk.Done {
		t.Error("错误片段应该标记为完成")
	}
}

// TestStreamInterface 测试流式接口实现
func TestStreamInterface(t *testing.T) {
	// 测试所有客户端都实现了ChatStream方法
	clients := []LLMClient{
		NewGLMClient(""),
		NewDeepSeekClient(""),
		NewQwenClient(""),
		NewClaudeClient(""),
		NewKimiClient(""),
		NewDouBaoClient(""),
		NewGeminiClient(""),
		NewQwen3CoderClient(""),
	}

	for _, client := range clients {
		t.Run(fmt.Sprintf("Test_%s_Stream", client.GetProvider()), func(t *testing.T) {
			req := &ChatRequest{
				Messages: []Message{
					{Role: "user", Content: "Test"},
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			stream, err := client.ChatStream(ctx, req)
			if err != nil {
				t.Fatalf("创建流式请求失败: %v", err)
			}

			// 至少应该能创建流式通道
			if stream == nil {
				t.Error("流式通道不应该为nil")
			}

			// 读取一个数据块
			select {
			case chunk := <-stream:
				if chunk == nil {
					t.Error("流式数据块不应该为nil")
				}
			case <-time.After(2 * time.Second):
				t.Error("流式响应超时")
			}
		})
	}
}
