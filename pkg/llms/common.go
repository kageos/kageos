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
	return nil
}
