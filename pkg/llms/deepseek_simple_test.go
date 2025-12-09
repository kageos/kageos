package llms

import (
	"context"
	"testing"
	"time"
)

// TestDeepSeekSimple 简单的DeepSeek流式测试
func TestDeepSeekSimple(t *testing.T) {
	client := NewDeepSeekClient("")

	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "你好"},
		},
		MaxTokens:   50,
		Temperature: 0.7,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := client.ChatStream(ctx, req)
	if err != nil {
		t.Fatalf("创建流式请求失败: %v", err)
	}

	t.Logf("开始接收DeepSeek流式响应...")

	chunkCount := 0
	for chunk := range stream {
		chunkCount++
		t.Logf("=== 第 %d 个数据块 ===", chunkCount)
		t.Logf("Content: '%s'", chunk.Content)
		t.Logf("Done: %v", chunk.Done)
		t.Logf("Error: '%s'", chunk.Error)
		if chunk.Usage != nil {
			t.Logf("Usage: %+v", chunk.Usage)
		}

		if chunk.Done {
			t.Logf("流式响应完成")
			break
		}
	}

	t.Logf("总共收到 %d 个数据块", chunkCount)
}
