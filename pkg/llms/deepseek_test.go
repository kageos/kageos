package llms

import (
	"context"
	"testing"
	"time"
)

// TestDeepSeekChat 测试DeepSeek普通聊天功能
func TestDeepSeekChat(t *testing.T) {
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

	resp, err := client.Chat(ctx, req)
	if err != nil {
		t.Fatalf("DeepSeek聊天失败: %v", err)
	}

	if resp.Content == "" {
		t.Error("响应内容不能为空")
	}

	if resp.Usage == nil {
		t.Error("使用统计不能为空")
	}

	t.Logf("DeepSeek响应: %s", resp.Content)
	t.Logf("使用统计: %+v", resp.Usage)
}

// TestDeepSeekChatStream 测试DeepSeek流式聊天功能
func TestDeepSeekChatStream(t *testing.T) {
	client := NewDeepSeekClient("")

	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请写一首关于春天的短诗"},
		},
		MaxTokens:   200,
		Temperature: 0.8,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := client.ChatStream(ctx, req)
	if err != nil {
		t.Fatalf("创建流式请求失败: %v", err)
	}

	var content string
	var chunkCount int
	var firstChunkTime time.Duration
	startTime := time.Now()

	for chunk := range stream {
		if chunk.Error != "" {
			t.Logf("流式响应错误: %s", chunk.Error)
			break
		}

		if chunk.Content != "" {
			if firstChunkTime == 0 {
				firstChunkTime = time.Since(startTime)
			}
			content += chunk.Content
			chunkCount++
			t.Logf("收到内容片段 %d: %s", chunkCount, chunk.Content)
		}

		if chunk.Done {
			totalTime := time.Since(startTime)
			t.Logf("流式响应完成")
			t.Logf("首字响应时间: %v", firstChunkTime)
			t.Logf("总响应时间: %v", totalTime)
			t.Logf("接收片段数: %d", chunkCount)
			t.Logf("完整内容: %s", content)

			if chunk.Usage != nil {
				t.Logf("使用统计: %+v", chunk.Usage)
			}
			break
		}
	}

	if content == "" {
		t.Error("流式响应内容不能为空")
	}

	if chunkCount == 0 {
		t.Error("应该接收到至少一个内容片段")
	}
}

// TestDeepSeekChatWithSystemMessage 测试带系统消息的聊天
func TestDeepSeekChatWithSystemMessage(t *testing.T) {
	client := NewDeepSeekClient("")

	req := &ChatRequest{
		Messages: []Message{
			{Role: "system", Content: "你是一个专业的编程助手，请用简洁明了的语言回答问题"},
			{Role: "user", Content: "什么是Go语言的goroutine？"},
		},
		MaxTokens:   300,
		Temperature: 0.5,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, req)
	if err != nil {
		t.Fatalf("DeepSeek聊天失败: %v", err)
	}

	if resp.Content == "" {
		t.Error("响应内容不能为空")
	}

	t.Logf("DeepSeek响应: %s", resp.Content)
}

// TestDeepSeekChatStreamWithSystemMessage 测试带系统消息的流式聊天
func TestDeepSeekChatStreamWithSystemMessage(t *testing.T) {
	client := NewDeepSeekClient("")

	req := &ChatRequest{
		Messages: []Message{
			{Role: "system", Content: "你是一个创意写作助手，请用富有想象力的语言创作"},
			{Role: "user", Content: "写一个关于未来科技的科幻故事开头"},
		},
		MaxTokens:   400,
		Temperature: 0.9,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := client.ChatStream(ctx, req)
	if err != nil {
		t.Fatalf("创建流式请求失败: %v", err)
	}

	var content string
	var chunkCount int

	for chunk := range stream {
		if chunk.Error != "" {
			t.Logf("流式响应错误: %s", chunk.Error)
			break
		}

		if chunk.Content != "" {
			content += chunk.Content
			chunkCount++
		}

		if chunk.Done {
			break
		}
	}

	if content == "" {
		t.Error("流式响应内容不能为空")
	}

	t.Logf("完整内容: %s", content)
}

// TestDeepSeekChatTimeout 测试超时处理
func TestDeepSeekChatTimeout(t *testing.T) {
	client := NewDeepSeekClient("")

	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请详细解释量子计算原理"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	// 设置很短的超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := client.Chat(ctx, req)
	if err == nil {
		t.Error("应该因为超时而失败")
	}

	t.Logf("预期的超时错误: %v", err)
}

// TestDeepSeekChatStreamTimeout 测试流式聊天超时处理
func TestDeepSeekChatStreamTimeout(t *testing.T) {
	client := NewDeepSeekClient("")

	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请写一篇关于人工智能的长篇论文"},
		},
		MaxTokens:   2000,
		Temperature: 0.7,
	}

	// 设置很短的超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := client.ChatStream(ctx, req)
	if err != nil {
		t.Fatalf("创建流式请求失败: %v", err)
	}

	var content string
	timeout := false

	for chunk := range stream {
		if chunk.Error != "" {
			t.Logf("流式响应错误: %s", chunk.Error)
			timeout = true
			break
		}

		if chunk.Content != "" {
			content += chunk.Content
		}

		if chunk.Done {
			break
		}
	}

	if !timeout && content == "" {
		t.Error("应该因为超时而失败或接收到内容")
	}

	t.Logf("接收到的内容长度: %d", len(content))
}

// TestDeepSeekChatInvalidAPIKey 测试无效API密钥
func TestDeepSeekChatInvalidAPIKey(t *testing.T) {
	client := NewDeepSeekClient("invalid-api-key")

	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := client.Chat(ctx, req)
	if err == nil {
		t.Error("应该因为无效API密钥而失败")
	}

	t.Logf("预期的API密钥错误: %v", err)
}

// TestDeepSeekChatStreamInvalidAPIKey 测试流式聊天无效API密钥
func TestDeepSeekChatStreamInvalidAPIKey(t *testing.T) {
	client := NewDeepSeekClient("invalid-api-key")

	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.ChatStream(ctx, req)
	if err != nil {
		t.Fatalf("创建流式请求失败: %v", err)
	}

	chunk := <-stream
	if chunk.Error == "" {
		t.Error("应该返回API密钥错误")
	}

	t.Logf("预期的API密钥错误: %s", chunk.Error)
}

// TestDeepSeekChatPerformance 测试性能对比
func TestDeepSeekChatPerformance(t *testing.T) {
	client := NewDeepSeekClient("")

	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请简单介绍一下Go语言的特点"},
		},
		MaxTokens:   200,
		Temperature: 0.7,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 测试普通聊天性能
	t.Run("Normal_Chat", func(t *testing.T) {
		start := time.Now()
		resp, err := client.Chat(ctx, req)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("普通聊天失败: %v", err)
		}

		t.Logf("普通聊天耗时: %v", duration)
		t.Logf("响应长度: %d", len(resp.Content))
		if resp.Usage != nil {
			t.Logf("Token使用: %d", resp.Usage.TotalTokens)
		}
	})

	// 测试流式聊天性能
	t.Run("Stream_Chat", func(t *testing.T) {
		start := time.Now()
		stream, err := client.ChatStream(ctx, req)
		if err != nil {
			t.Fatalf("创建流式请求失败: %v", err)
		}

		var content string
		var firstChunkTime time.Duration
		var chunkCount int

		for chunk := range stream {
			if chunk.Error != "" {
				t.Logf("流式响应错误: %s", chunk.Error)
				break
			}

			if chunk.Content != "" {
				if firstChunkTime == 0 {
					firstChunkTime = time.Since(start)
				}
				content += chunk.Content
				chunkCount++
			}

			if chunk.Done {
				totalTime := time.Since(start)
				t.Logf("流式聊天总耗时: %v", totalTime)
				t.Logf("首字响应时间: %v", firstChunkTime)
				t.Logf("响应长度: %d", len(content))
				t.Logf("接收片段数: %d", chunkCount)
				break
			}
		}
	})
}
