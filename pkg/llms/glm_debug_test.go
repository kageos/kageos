package llms

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// TestGLMDebugStream 调试GLM流式响应
func TestGLMDebugStream(t *testing.T) {
	client := NewGLMClient("")

	// 启用思考模式的流式请求
	enableThinking := true
	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请简单介绍一下Go语言"},
		},
		MaxTokens:   100,
		Temperature: 0.7,
		UseThinking: &enableThinking,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := client.ChatStream(ctx, req)
	if err != nil {
		t.Fatalf("创建GLM流式请求失败: %v", err)
	}

	var content string
	var chunkCount int
	var rawResponses []string

	for chunk := range stream {
		t.Logf("收到chunk: %+v", chunk)

		if chunk.Error != "" {
			t.Logf("流式响应错误: %s", chunk.Error)
			break
		}

		if chunk.Content != "" {
			content += chunk.Content
			chunkCount++
			t.Logf("内容片段 %d: %s", chunkCount, chunk.Content)
		}

		if chunk.Done {
			t.Logf("流式响应完成")
			t.Logf("总接收片段数: %d", chunkCount)
			t.Logf("完整内容长度: %d", len(content))

			if chunk.Usage != nil {
				t.Logf("使用统计: %+v", chunk.Usage)
			}
			break
		}
	}

	t.Logf("原始响应数据: %v", rawResponses)

	if content == "" {
		t.Error("流式响应内容不能为空")
	} else {
		t.Logf("最终内容: %s", content)
	}
}

// TestGLMStreamRawResponse 测试GLM原始流式响应
func TestGLMStreamRawResponse(t *testing.T) {
	// 模拟GLM流式响应数据
	testCases := []struct {
		name     string
		response string
		expected string
	}{
		{
			name:     "标准流式响应",
			response: `{"choices":[{"delta":{"content":"Hello"},"finish_reason":""}]}`,
			expected: "Hello",
		},
		{
			name:     "思考模式响应",
			response: `{"choices":[{"delta":{"content":"<thinking>让我想想</thinking>"},"finish_reason":""}]}`,
			expected: "<thinking>让我想想</thinking>",
		},
		{
			name:     "完成响应",
			response: `{"choices":[{"delta":{"content":"World"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`,
			expected: "World",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var streamResp GLMStreamResponse
			err := json.Unmarshal([]byte(tc.response), &streamResp)
			if err != nil {
				t.Fatalf("解析响应失败: %v", err)
			}

			t.Logf("解析后的响应: %+v", streamResp)

			if len(streamResp.Choices) > 0 {
				choice := streamResp.Choices[0]
				t.Logf("Delta内容: %s", choice.Delta.Content)
				t.Logf("完成原因: %s", choice.FinishReason)

				if choice.Delta.Content != tc.expected {
					t.Errorf("期望内容 %s, 实际内容 %s", tc.expected, choice.Delta.Content)
				}
			} else {
				t.Error("没有选择内容")
			}
		})
	}
}
