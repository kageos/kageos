package llms

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// createHTTPClient 创建HTTP客户端（公共函数）
func createHTTPClient(options *ClientOptions, timeout time.Duration) *http.Client {
	options = normalizeClientOptions(options)
	if timeout <= 0 {
		timeout = options.Timeout
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:       options.MaxIdleConns,
			IdleConnTimeout:    options.IdleConnTimeout,
			DisableCompression: true,
		},
	}
}

func normalizeClientOptions(options *ClientOptions) *ClientOptions {
	if options == nil {
		return DefaultClientOptions()
	}
	return options
}

func resolveRequestTimeout(options *ClientOptions, req *ChatRequest) time.Duration {
	options = normalizeClientOptions(options)
	timeout := options.Timeout
	if req != nil && req.Timeout != nil && *req.Timeout > 0 {
		timeout = *req.Timeout
	}
	return timeout
}

func sendStreamChunk(ctx context.Context, chunkChan chan<- *StreamChunk, chunk *StreamChunk) bool {
	select {
	case chunkChan <- chunk:
		return true
	case <-ctx.Done():
		return false
	}
}

// validateRequest 验证请求参数（公共函数）
func validateRequest(ctx context.Context, apiKey string, req *ChatRequest) error {
	if req == nil {
		return fmt.Errorf("request 不能为空")
	}
	if apiKey == "" {
		return fmt.Errorf("API Key 不能为空")
	}
	if len(req.Messages) == 0 {
		return fmt.Errorf("messages 不能为空")
	}
	// 验证消息格式
	for i, msg := range req.Messages {
		if msg.Role == "" {
			return fmt.Errorf("消息 %d 的 role 不能为空", i)
		}
		switch strings.ToLower(strings.TrimSpace(msg.Role)) {
		case "assistant":
			if msg.Content == "" && len(msg.ToolCalls) == 0 {
				return fmt.Errorf("消息 %d 的 content 或 tool_calls 不能同时为空", i)
			}
		case "tool":
			if msg.ToolCallID == "" {
				return fmt.Errorf("消息 %d 的 tool_call_id 不能为空", i)
			}
		default:
			if msg.Content == "" {
				return fmt.Errorf("消息 %d 的 content 不能为空", i)
			}
		}
	}
	if err := validateToolMessageOrder(req.Messages); err != nil {
		return err
	}
	return nil
}

func validateToolMessageOrder(messages []Message) error {
	pending := make(map[string]struct{})
	pendingOrder := make([]string, 0)
	for i, msg := range messages {
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		if len(pending) > 0 {
			if role != "tool" {
				return fmt.Errorf("消息 %d 之前的 assistant tool_calls 缺少 tool 结果: %s", i, strings.Join(pendingOrder, ", "))
			}
			id := strings.TrimSpace(msg.ToolCallID)
			if _, ok := pending[id]; !ok {
				return fmt.Errorf("消息 %d 的 tool_call_id %q 未匹配上一条 assistant tool_calls", i, msg.ToolCallID)
			}
			delete(pending, id)
			pendingOrder = removePendingToolCallID(pendingOrder, id)
			continue
		}
		if role == "tool" {
			return fmt.Errorf("消息 %d 的 tool 结果没有紧邻的 assistant tool_calls", i)
		}
		if role != "assistant" || len(msg.ToolCalls) == 0 {
			continue
		}
		for _, tc := range msg.ToolCalls {
			id := strings.TrimSpace(tc.ID)
			if id == "" {
				return fmt.Errorf("消息 %d 的 assistant tool_calls 包含空 tool_call id", i)
			}
			if _, exists := pending[id]; exists {
				return fmt.Errorf("消息 %d 的 assistant tool_calls 包含重复 tool_call id %q", i, id)
			}
			pending[id] = struct{}{}
			pendingOrder = append(pendingOrder, id)
		}
	}
	if len(pending) > 0 {
		return fmt.Errorf("最后一条 assistant tool_calls 缺少 tool 结果: %s", strings.Join(pendingOrder, ", "))
	}
	return nil
}

func removePendingToolCallID(ids []string, id string) []string {
	for i, v := range ids {
		if v == id {
			return append(ids[:i], ids[i+1:]...)
		}
	}
	return ids
}
