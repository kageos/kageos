package llms

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// TestDeepSeekDebugDetailed 详细调试DeepSeek流式响应
func TestDeepSeekDebugDetailed(t *testing.T) {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		t.Skip("DEEPSEEK_API_KEY not set")
	}

	// 构造DeepSeek API请求
	apiReq := map[string]interface{}{
		"model":       "deepseek-chat",
		"messages":    []Message{{Role: "user", Content: "你好"}},
		"max_tokens":  50,
		"temperature": 0.7,
		"stream":      true,
	}

	jsonData, err := json.Marshal(apiReq)
	if err != nil {
		t.Fatalf("序列化请求失败: %v", err)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequest("POST", "https://api.deepseek.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("创建HTTP请求失败: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("HTTP请求失败，状态码: %d", resp.StatusCode)
	}

	// 模拟我们的流式解析代码
	scanner := bufio.NewScanner(resp.Body)
	var finalUsage *Usage
	chunkCount := 0

	t.Logf("开始模拟我们的流式解析代码...")

	for scanner.Scan() {
		line := scanner.Text()

		// 跳过SSE格式的注释行和空行
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// 处理SSE数据行
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// 检查是否是结束标记
			if data == "[DONE]" {
				t.Logf("=== 第 %d 个数据块 (结束标记) ===", chunkCount+1)
				t.Logf("发送完成信号")
				break
			}

			// 解析JSON数据
			var streamResp DeepSeekStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				t.Logf("JSON解析失败: %v", err)
				continue
			}

			chunkCount++
			t.Logf("=== 第 %d 个数据块 ===", chunkCount)
			t.Logf("原始数据: %s", data)

			// 检查错误
			if streamResp.Error != nil {
				t.Logf("API错误: %s - %s", streamResp.Error.Code, streamResp.Error.Message)
				break
			}

			// 处理选择内容
			if len(streamResp.Choices) > 0 {
				choice := streamResp.Choices[0]
				finishReason := ""
				if choice.FinishReason != nil {
					finishReason = *choice.FinishReason
				}
				t.Logf("Choice: delta.content='%s', finish_reason='%s'", choice.Delta.Content, finishReason)

				// 发送内容片段
				if choice.Delta.Content != "" {
					t.Logf("发送内容片段: '%s'", choice.Delta.Content)
				} else {
					t.Logf("内容为空，跳过")
				}

				// 检查是否完成
				if choice.FinishReason != nil && *choice.FinishReason != "" {
					t.Logf("检测到完成信号: %s", *choice.FinishReason)

					// 保存使用统计
					if streamResp.Usage != nil {
						finalUsage = &Usage{
							PromptTokens:     int(streamResp.Usage.PromptTokens),
							CompletionTokens: int(streamResp.Usage.CompletionTokens),
							TotalTokens:      int(streamResp.Usage.TotalTokens),
						}
						t.Logf("使用统计: %+v", finalUsage)
					}

					// 发送完成信号
					t.Logf("发送完成信号")
					break
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("读取流式响应失败: %v", err)
	}

	t.Logf("总共处理了 %d 个数据块", chunkCount)
}
