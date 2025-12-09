package llms

import (
	"context"
	"testing"
	"time"
)

// TestGLMThinkingStream 测试GLM思考过程的流式内容
func TestGLMThinkingStream(t *testing.T) {
	client := NewGLMClient("")

	// 启用思考模式的流式请求
	enableThinking := true
	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请详细分析一下Go语言的并发模型，包括goroutine、channel和select的工作原理"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
		UseThinking: &enableThinking, // 启用思考模式
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	stream, err := client.ChatStream(ctx, req)
	if err != nil {
		t.Fatalf("创建GLM思考流式请求失败: %v", err)
	}

	var content string
	var chunkCount int
	var thinkingContent string
	var finalContent string
	var inThinkingMode bool

	for chunk := range stream {
		if chunk.Error != "" {
			t.Logf("流式响应错误: %s", chunk.Error)
			break
		}

		if chunk.Content != "" {
			content += chunk.Content
			chunkCount++

			// 检测思考过程内容（通常以特定标记开始）
			if chunk.Content == "<thinking>" || chunk.Content == "```thinking" || chunk.Content == "**思考过程**" {
				inThinkingMode = true
				t.Logf("🧠 开始思考过程...")
			} else if chunk.Content == "</thinking>" || chunk.Content == "```" || chunk.Content == "**回答**" {
				inThinkingMode = false
				t.Logf("💡 思考过程结束，开始回答...")
			}

			if inThinkingMode {
				thinkingContent += chunk.Content
				t.Logf("🧠 思考内容: %s", chunk.Content)
			} else {
				finalContent += chunk.Content
				t.Logf("💬 回答内容: %s", chunk.Content)
			}
		}

		if chunk.Done {
			t.Logf("流式响应完成")
			t.Logf("总接收片段数: %d", chunkCount)
			t.Logf("思考内容长度: %d", len(thinkingContent))
			t.Logf("最终回答长度: %d", len(finalContent))
			t.Logf("完整内容长度: %d", len(content))

			if chunk.Usage != nil {
				t.Logf("使用统计: %+v", chunk.Usage)
			}
			break
		}
	}

	if content == "" {
		t.Error("流式响应内容不能为空")
	}

	// 检查是否包含思考过程
	if thinkingContent == "" {
		t.Log("⚠️ 未检测到明显的思考过程内容，可能模型没有返回思考标记")
	} else {
		t.Logf("✅ 检测到思考过程内容: %s", thinkingContent[:minInt(200, len(thinkingContent))])
	}
}

// TestGLMThinkingComparison 测试GLM思考模式对比
func TestGLMThinkingComparison(t *testing.T) {
	client := NewGLMClient("")

	baseReq := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请解释一下什么是微服务架构，以及它的优缺点"},
		},
		MaxTokens:   800,
		Temperature: 0.7,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 测试启用思考模式
	t.Run("With_Thinking", func(t *testing.T) {
		enableThinking := true
		req := *baseReq
		req.UseThinking = &enableThinking

		start := time.Now()
		stream, err := client.ChatStream(ctx, &req)
		if err != nil {
			t.Fatalf("创建思考流式请求失败: %v", err)
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

		duration := time.Since(start)
		t.Logf("思考模式 - 耗时: %v, 内容长度: %d, 片段数: %d", duration, len(content), chunkCount)
	})

	// 测试禁用思考模式
	t.Run("Without_Thinking", func(t *testing.T) {
		disableThinking := false
		req := *baseReq
		req.UseThinking = &disableThinking

		start := time.Now()
		stream, err := client.ChatStream(ctx, &req)
		if err != nil {
			t.Fatalf("创建非思考流式请求失败: %v", err)
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

		duration := time.Since(start)
		t.Logf("非思考模式 - 耗时: %v, 内容长度: %d, 片段数: %d", duration, len(content), chunkCount)
	})
}

// TestDeepSeekThinkingStream 测试DeepSeek是否支持思考过程流式内容
func TestDeepSeekThinkingStream(t *testing.T) {
	client := NewDeepSeekClient("")

	// DeepSeek可能不支持思考模式，但我们可以测试一下
	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请详细分析一下Go语言的并发模型，包括goroutine、channel和select的工作原理"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
		// DeepSeek可能不支持UseThinking参数
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	stream, err := client.ChatStream(ctx, req)
	if err != nil {
		t.Fatalf("创建DeepSeek流式请求失败: %v", err)
	}

	var content string
	var chunkCount int
	var thinkingContent string
	var finalContent string
	var inThinkingMode bool

	for chunk := range stream {
		if chunk.Error != "" {
			t.Logf("流式响应错误: %s", chunk.Error)
			break
		}

		if chunk.Content != "" {
			content += chunk.Content
			chunkCount++

			// 检测可能的思考过程内容
			if chunk.Content == "<thinking>" || chunk.Content == "```thinking" || chunk.Content == "**思考过程**" || chunk.Content == "Let me think" {
				inThinkingMode = true
				t.Logf("🧠 检测到思考过程开始...")
			} else if chunk.Content == "</thinking>" || chunk.Content == "```" || chunk.Content == "**回答**" {
				inThinkingMode = false
				t.Logf("💡 思考过程结束，开始回答...")
			}

			if inThinkingMode {
				thinkingContent += chunk.Content
				t.Logf("🧠 思考内容: %s", chunk.Content)
			} else {
				finalContent += chunk.Content
				t.Logf("💬 回答内容: %s", chunk.Content)
			}
		}

		if chunk.Done {
			t.Logf("流式响应完成")
			t.Logf("总接收片段数: %d", chunkCount)
			t.Logf("思考内容长度: %d", len(thinkingContent))
			t.Logf("最终回答长度: %d", len(finalContent))
			t.Logf("完整内容长度: %d", len(content))

			if chunk.Usage != nil {
				t.Logf("使用统计: %+v", chunk.Usage)
			}
			break
		}
	}

	if content == "" {
		t.Error("流式响应内容不能为空")
	}

	// 检查是否包含思考过程
	if thinkingContent == "" {
		t.Log("⚠️ DeepSeek未检测到明显的思考过程内容")
	} else {
		t.Logf("✅ DeepSeek检测到思考过程内容: %s", thinkingContent[:minInt(200, len(thinkingContent))])
	}
}

// TestThinkingStreamDetection 测试思考过程内容检测
func TestThinkingStreamDetection(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected bool
	}{
		{"标准思考标记", "<thinking>这是思考内容</thinking>", true},
		{"代码块思考", "```thinking\n这是思考内容\n```", true},
		{"Markdown思考", "**思考过程**\n这是思考内容", true},
		{"英文思考", "Let me think about this...", true},
		{"普通内容", "这是普通的回答内容", false},
		{"空内容", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasThinking := detectThinkingContent(tc.content)
			if hasThinking != tc.expected {
				t.Errorf("检测结果不匹配: 期望 %v, 实际 %v", tc.expected, hasThinking)
			}
		})
	}
}

// detectThinkingContent 检测内容是否包含思考过程
func detectThinkingContent(content string) bool {
	thinkingMarkers := []string{
		"<thinking>",
		"</thinking>",
		"```thinking",
		"**思考过程**",
		"Let me think",
		"思考一下",
		"让我想想",
	}

	for _, marker := range thinkingMarkers {
		if containsString(content, marker) {
			return true
		}
	}
	return false
}

// containsString 检查字符串是否包含子字符串
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			indexOf(s, substr) >= 0)))
}

// indexOf 查找子字符串位置
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// minInt 返回两个整数中的较小值
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
