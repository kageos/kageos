package apicall

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/serviceconfig"
)

type ApiResult[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

type callConfig struct {
	headers map[string]string
}

type callOption func(*callConfig)

// httpClient 通用 HTTP 客户端（复用连接，提高性能）。
var httpClient = &http.Client{
	Timeout: 300 * time.Second,
}

func callAPI[T any](ctx context.Context, method, path string, reqBody interface{}) (*ApiResult[T], error) {
	fullURL := serviceconfig.BuildGatewayURL(path)
	return callAPIWithOptions[T](ctx, method, fullURL, reqBody)
}

// CallAPI calls a gateway API and decodes only the response data field into
// respData. It is the non-generic entry used by SDK code generated inside
// workspaces.
func CallAPI(ctx context.Context, method, path string, reqBody interface{}, respData interface{}) error {
	fullURL := strings.TrimSpace(path)
	if !isHTTPURL(fullURL) {
		fullURL = serviceconfig.BuildGatewayURL(fullURL)
	}
	result, err := callAPIWithOptions[json.RawMessage](ctx, method, fullURL, reqBody)
	if err != nil {
		return err
	}
	if respData == nil || len(result.Data) == 0 || string(result.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(result.Data, respData); err != nil {
		return fmt.Errorf("解析响应 data 失败: %w", err)
	}
	return nil
}

// CallAPIWithURL 使用完整 URL 调用 API。
// 注意：这里直接使用完整 URL，不通过网关地址拼接。
func CallAPIWithURL[T any](ctx context.Context, method, fullURL string, reqBody interface{}) (*ApiResult[T], error) {
	return callAPIWithOptions[T](ctx, method, fullURL, reqBody)
}

func callAPIWithOptions[T any](ctx context.Context, method, fullURL string, reqBody interface{}, options ...callOption) (*ApiResult[T], error) {
	if ctx == nil {
		ctx = context.Background()
	}

	config := buildCallConfig(options...)
	bodyReader, err := buildRequestBody(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	applyCommonHeaders(req, ctx)
	for k, v := range config.headers {
		req.Header.Set(k, v)
	}

	return doAPIRequest[T](req)
}

func buildCallConfig(options ...callOption) callConfig {
	config := callConfig{}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	return config
}

func withHeader(key, value string) callOption {
	return func(config *callConfig) {
		if config.headers == nil {
			config.headers = make(map[string]string)
		}
		config.headers[key] = value
	}
}

func buildRequestBody(reqBody interface{}) (io.Reader, error) {
	if reqBody == nil {
		return nil, nil
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}
	return bytes.NewReader(bodyBytes), nil
}

func applyCommonHeaders(req *http.Request, ctx context.Context) {
	req.Header.Set("Content-Type", "application/json")

	if token := contextx.GetToken(ctx); token != "" {
		req.Header.Set(contextx.TokenHeader, token)
	}
	if traceID := contextx.GetTraceId(ctx); traceID != "" {
		req.Header.Set(contextx.TraceIdHeader, traceID)
	}
	if requestUser := contextx.GetRequestUser(ctx); requestUser != "" {
		req.Header.Set(contextx.RequestUserHeader, requestUser)
	}
	if departmentFullPath := contextx.GetRequestDepartmentFullPath(ctx); departmentFullPath != "" {
		req.Header.Set(contextx.DepartmentFullPathHeader, departmentFullPath)
	}
	if clientSource := contextx.GetClientSource(ctx); clientSource != "" {
		req.Header.Set(contextx.ClientSourceHeader, clientSource)
	}
	if sourceType := contextx.GetSourceType(ctx); sourceType != "" {
		req.Header.Set(contextx.SourceTypeHeader, sourceType)
	}
	if sourceRef := contextx.GetSourceRef(ctx); sourceRef != "" {
		req.Header.Set(contextx.SourceRefHeader, sourceRef)
	}
}

func isHTTPURL(raw string) bool {
	raw = strings.ToLower(strings.TrimSpace(raw))
	return strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://")
}

func doAPIRequest[T any](req *http.Request) (*ApiResult[T], error) {
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, formatHTTPError(resp, bodyBytes)
	}

	var result ApiResult[T]
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(bodyBytes))
	}

	if result.Code != 0 {
		return &result, fmt.Errorf("业务错误 [%d]: %s", result.Code, result.Msg)
	}

	return &result, nil
}

func formatHTTPError(resp *http.Response, bodyBytes []byte) error {
	body := strings.TrimSpace(string(bodyBytes))
	return fmt.Errorf("HTTP错误: %d %s, 响应: %s", resp.StatusCode, resp.Status, body)
}
