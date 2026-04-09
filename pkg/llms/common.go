package llms

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
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

func newJSONRequest(ctx context.Context, rawURL string, jsonData []byte, options *ClientOptions) (*http.Request, error) {
	options = normalizeClientOptions(options)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if options.UserAgent != "" {
		req.Header.Set("User-Agent", options.UserAgent)
	}
	return req, nil
}

func newBearerJSONRequest(ctx context.Context, rawURL, apiKey string, jsonData []byte, options *ClientOptions) (*http.Request, error) {
	req, err := newJSONRequest(ctx, rawURL, jsonData, options)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	return req, nil
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
