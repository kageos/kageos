package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	pathpkg "path"
	"strings"
	"time"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"gorm.io/gorm"
)

const connectorProxyMaxBodyBytes = 4 << 20

var connectorProxyHTTPClient = &http.Client{Timeout: 30 * time.Second}

func (s *ConnectorService) Proxy(ctx context.Context, req dto.ConnectorProxyReq) (*dto.ConnectorProxyResp, error) {
	if err := s.ensureOAuthReady(); err != nil {
		return nil, err
	}
	owner := contextx.GetRequestUser(ctx)
	if owner == "" {
		return nil, fmt.Errorf("未提供用户信息")
	}
	provider := normalizeProvider(req.Provider)
	if provider == "" {
		return nil, fmt.Errorf("provider 不能为空")
	}
	adapter := connectorAdapterFor(provider)
	baseURL := strings.TrimSpace(adapter.ProxyBaseURL())
	if baseURL == "" {
		return nil, fmt.Errorf("connector proxy 暂不支持 provider: %s", provider)
	}
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}
	if !allowedConnectorProxyMethod(method) {
		return nil, fmt.Errorf("connector proxy 不支持 HTTP 方法: %s", method)
	}
	if _, err := s.resolveOAuthProvider(ctx, provider); err != nil {
		return nil, err
	}
	resolved, err := s.ResolveDirectoryBinding(ctx, req.ResourcePath, provider)
	if err != nil {
		return nil, err
	}
	tokenRow, err := s.repo.GetOwnedOAuthToken(ctx, owner, resolved.Connection.ConnectionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("连接器没有 OAuth token")
		}
		return nil, err
	}
	if tokenRow.Expiry != nil && time.Until(*tokenRow.Expiry) <= time.Minute {
		refreshed, err := s.RefreshOAuthToken(ctx, resolved.Connection.ConnectionID)
		if err != nil {
			return nil, err
		}
		tokenRow, err = s.repo.GetOwnedOAuthToken(ctx, owner, refreshed.Connection.ConnectionID)
		if err != nil {
			return nil, err
		}
	}
	accessToken, err := s.tokenVault.Open(tokenRow.AccessTokenCipher)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(accessToken) == "" {
		return nil, fmt.Errorf("连接器 access_token 为空，需重新授权")
	}
	targetURL, err := buildConnectorProxyURL(baseURL, req.Path, req.Query)
	if err != nil {
		return nil, err
	}
	req.Provider = provider
	outReq, err := adapter.BuildProxyRequest(ctx, method, targetURL, accessToken, req)
	if err != nil {
		return nil, err
	}
	outResp, err := connectorProxyHTTPClient.Do(outReq)
	if err != nil {
		return nil, fmt.Errorf("调用 %s API 失败: %w", provider, err)
	}
	defer outResp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(outResp.Body, connectorProxyMaxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("读取 %s API 响应失败: %w", provider, err)
	}
	if outResp.StatusCode < http.StatusOK || outResp.StatusCode >= http.StatusMultipleChoices {
		return nil, adapter.TranslateProxyError(outResp.StatusCode, data)
	}
	return &dto.ConnectorProxyResp{
		Provider:      provider,
		StatusCode:    outResp.StatusCode,
		Headers:       pickConnectorProxyHeaders(outResp.Header),
		Body:          json.RawMessage(data),
		ResolvedFrom:  resolved.ResolvedFrom,
		RequestedPath: resolved.RequestedPath,
	}, nil
}

func buildConnectorProxyURL(baseURL, rawPath string, query map[string]string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return "", fmt.Errorf("connector proxy base url 非法")
	}
	escapedPath, err := cleanConnectorProxyPath(rawPath)
	if err != nil {
		return "", err
	}
	target := *base
	target.Path = strings.TrimRight(base.EscapedPath(), "/") + escapedPath
	values := target.Query()
	for key, value := range query {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		values.Set(key, value)
	}
	target.RawQuery = values.Encode()
	return target.String(), nil
}

func cleanConnectorProxyPath(rawPath string) (string, error) {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return "", fmt.Errorf("connector proxy path 不能为空")
	}
	if strings.Contains(rawPath, "\\") || strings.Contains(rawPath, "?") || strings.Contains(rawPath, "#") {
		return "", fmt.Errorf("connector proxy path 只能是相对 API 路径")
	}
	parsed, err := url.Parse(rawPath)
	if err != nil {
		return "", fmt.Errorf("connector proxy path 非法: %w", err)
	}
	if parsed.IsAbs() || parsed.Host != "" || !strings.HasPrefix(parsed.Path, "/") {
		return "", fmt.Errorf("connector proxy path 必须以 / 开头，且不能包含域名")
	}
	if parsed.Path != pathpkg.Clean(parsed.Path) {
		return "", fmt.Errorf("connector proxy path 不能包含 . 或 .. 路径段")
	}
	segments := strings.Split(parsed.EscapedPath(), "/")
	for _, segment := range segments {
		unescaped, err := url.PathUnescape(segment)
		if err != nil {
			return "", fmt.Errorf("connector proxy path 编码非法: %w", err)
		}
		if unescaped == "." || unescaped == ".." {
			return "", fmt.Errorf("connector proxy path 不能包含 . 或 .. 路径段")
		}
	}
	return parsed.Path, nil
}

func buildDefaultConnectorProxyRequest(ctx context.Context, method, targetURL, accessToken, provider string, req dto.ConnectorProxyReq) (*http.Request, error) {
	var body io.Reader
	if len(req.Body) > 0 && method != http.MethodGet {
		body = bytes.NewReader(req.Body)
	}
	outReq, err := http.NewRequestWithContext(ctx, method, targetURL, body)
	if err != nil {
		return nil, fmt.Errorf("创建 connector proxy 请求失败: %w", err)
	}
	outReq.Header.Set("Authorization", "Bearer "+accessToken)
	outReq.Header.Set("Accept", "application/json")
	outReq.Header.Set("User-Agent", "KageOS-Connector")
	if method != http.MethodGet && len(req.Body) > 0 {
		outReq.Header.Set("Content-Type", "application/json")
	}
	decorateProviderAPIRequest(provider, outReq)
	for key, value := range req.Headers {
		switch http.CanonicalHeaderKey(strings.TrimSpace(key)) {
		case "Accept", "If-None-Match":
			if strings.TrimSpace(value) != "" {
				outReq.Header.Set(http.CanonicalHeaderKey(key), strings.TrimSpace(value))
			}
		}
	}
	return outReq, nil
}

func allowedConnectorProxyMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func pickConnectorProxyHeaders(headers http.Header) map[string]string {
	keys := []string{
		"Link",
		"X-RateLimit-Limit",
		"X-RateLimit-Remaining",
		"X-RateLimit-Reset",
		"ETag",
	}
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value := headers.Get(key); value != "" {
			out[key] = value
		}
	}
	return out
}
