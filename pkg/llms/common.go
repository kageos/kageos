package llms

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// createHTTPClient 创建HTTP客户端（公共函数）
func createHTTPClient(options *ClientOptions, timeout time.Duration) *http.Client {
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

// validateRequest 验证请求参数（公共函数）
func validateRequest(ctx context.Context, apiKey string, req *ChatRequest) error {
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
		if msg.Content == "" {
			return fmt.Errorf("消息 %d 的 content 不能为空", i)
		}
	}
	return nil
}
