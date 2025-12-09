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

// TestDeepSeekRawDebug 调试DeepSeek原始流式响应
func TestDeepSeekRawDebug(t *testing.T) {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		t.Skip("DEEPSEEK_API_KEY not set")
	}

	// 构造DeepSeek API请求
	apiReq := map[string]interface{}{
		"model":       "deepseek-chat",
		"messages":    []Message{{Role: "user", Content: "请简单介绍一下你自己"}},
		"max_tokens":  100,
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

	// 读取原始响应
	scanner := bufio.NewScanner(resp.Body)
	lineCount := 0

	t.Logf("开始读取DeepSeek原始流式响应...")

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		t.Logf("=== 第 %d 行 ===", lineCount)
		t.Logf("原始内容: %s", line)

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, ":") {
			t.Logf("跳过空行或注释")
			continue
		}

		// 处理SSE数据行
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			t.Logf("SSE数据: %s", data)

			if data == "[DONE]" {
				t.Logf("流式响应结束")
				break
			}

			// 尝试解析JSON
			var streamResp map[string]interface{}
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				t.Logf("JSON解析失败: %v", err)
				continue
			}

			t.Logf("解析后的JSON: %+v", streamResp)

			// 检查choices
			if choices, ok := streamResp["choices"].([]interface{}); ok && len(choices) > 0 {
				choice := choices[0].(map[string]interface{})
				t.Logf("Choice: %+v", choice)

				if delta, ok := choice["delta"].(map[string]interface{}); ok {
					t.Logf("Delta: %+v", delta)
					if content, ok := delta["content"].(string); ok {
						t.Logf("Delta Content: '%s'", content)
					}
				}

				if finishReason, ok := choice["finish_reason"].(string); ok {
					t.Logf("Finish Reason: '%s'", finishReason)
				}
			}

			// 检查usage
			if usage, ok := streamResp["usage"].(map[string]interface{}); ok {
				t.Logf("Usage: %+v", usage)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("读取流式响应失败: %v", err)
	}

	t.Logf("总共读取了 %d 行", lineCount)
}
