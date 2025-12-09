package llms

import (
	"context"
	"testing"
	"time"
)

// TestDeepSeekDebug 调试DeepSeek流式响应
func TestDeepSeekDebug(t *testing.T) {
	client := NewDeepSeekClient("")

	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请简单介绍一下你自己"},
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
	chunkCount := 0

	t.Logf("开始接收DeepSeek流式响应...")

	for chunk := range stream {
		chunkCount++
		t.Logf("=== 收到第 %d 个数据块 ===", chunkCount)
		t.Logf("Content: '%s'", chunk.Content)
		t.Logf("Done: %v", chunk.Done)
		t.Logf("Error: '%s'", chunk.Error)
		if chunk.Usage != nil {
			t.Logf("Usage: %+v", chunk.Usage)
		}

		if chunk.Error != "" {
			t.Logf("流式响应错误: %s", chunk.Error)
			break
		}

		if chunk.Content != "" {
			content += chunk.Content
			t.Logf("累积内容: %s", content)
		}

		if chunk.Done {
			done = true
			t.Logf("流式响应完成")
			break
		}
	}

	t.Logf("=== 流式响应结束 ===")
	t.Logf("总数据块数: %d", chunkCount)
	t.Logf("是否完成: %v", done)
	t.Logf("完整内容: '%s'", content)
	t.Logf("内容长度: %d", len(content))

	if !done {
		t.Error("流式响应未正常结束")
	}

	if content == "" {
		t.Error("流式响应内容为空")
	}
}
