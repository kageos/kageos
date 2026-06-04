package apicall

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/kageos/kageos/dto"
)

const ConnectorGlobalResourcePath = "/"

type ConnectorOAuthOption func(*dto.StartConnectorOAuthReq)

func WithConnectorResourcePath(resourcePath string) ConnectorOAuthOption {
	return func(req *dto.StartConnectorOAuthReq) {
		req.ResourcePath = strings.TrimSpace(resourcePath)
	}
}

func WithConnectorScopes(scopes ...string) ConnectorOAuthOption {
	return func(req *dto.StartConnectorOAuthReq) {
		req.Scopes = append([]string(nil), scopes...)
	}
}

func WithConnectorDisplayName(displayName string) ConnectorOAuthOption {
	return func(req *dto.StartConnectorOAuthReq) {
		req.DisplayName = strings.TrimSpace(displayName)
	}
}

func WithConnectorRedirectAfter(redirectAfter string) ConnectorOAuthOption {
	return func(req *dto.StartConnectorOAuthReq) {
		req.RedirectAfter = strings.TrimSpace(redirectAfter)
	}
}

func StartConnectorOAuth(ctx context.Context, req *dto.StartConnectorOAuthReq) (*dto.StartConnectorOAuthResp, error) {
	if req == nil {
		return nil, fmt.Errorf("connector oauth request 不能为空")
	}
	payload := *req
	if strings.TrimSpace(payload.ResourcePath) == "" {
		payload.ResourcePath = ConnectorGlobalResourcePath
	}
	return PostAPI[*dto.StartConnectorOAuthReq, *dto.StartConnectorOAuthResp](ctx, "/connector/api/v1/oauth/authorize", &payload)
}

func StartGlobalConnectorOAuth(ctx context.Context, provider string, opts ...ConnectorOAuthOption) (*dto.StartConnectorOAuthResp, error) {
	req := &dto.StartConnectorOAuthReq{
		Provider:     strings.TrimSpace(provider),
		ResourcePath: ConnectorGlobalResourcePath,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(req)
		}
	}
	if strings.TrimSpace(req.ResourcePath) == "" {
		req.ResourcePath = ConnectorGlobalResourcePath
	}
	return StartConnectorOAuth(ctx, req)
}

func ResolveConnectorBinding(ctx context.Context, provider, resourcePath string) (*dto.ResolveConnectorBindingResp, error) {
	return ResolveConnectorBindingWithScopes(ctx, provider, resourcePath, nil)
}

func ResolveConnectorBindingWithScopes(ctx context.Context, provider, resourcePath string, requiredScopes []string) (*dto.ResolveConnectorBindingResp, error) {
	resourcePath = strings.TrimSpace(resourcePath)
	if resourcePath == "" {
		resourcePath = ConnectorGlobalResourcePath
	}
	return GetAPI[*dto.ResolveConnectorBindingResp](ctx, "/connector/api/v1/resolve", buildQueryParams(
		func(values url.Values) {
			values.Set("provider", strings.TrimSpace(provider))
		},
		func(values url.Values) {
			values.Set("resource_path", resourcePath)
		},
		withCSVQueryValue("required_scopes", requiredScopes),
	))
}

func GetConnectorOAuthProvider(ctx context.Context, provider string) (*dto.GetConnectorOAuthProviderResp, error) {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return nil, fmt.Errorf("provider 不能为空")
	}
	return GetAPI[*dto.GetConnectorOAuthProviderResp](ctx, "/connector/api/v1/oauth/providers/"+url.PathEscape(provider), nil)
}

func ListConnectorConnections(ctx context.Context, provider string) (*dto.ListConnectorConnectionsResp, error) {
	return GetAPI[*dto.ListConnectorConnectionsResp](ctx, "/connector/api/v1/connections", buildQueryParams(
		withTrimmedQueryValue("provider", provider),
	))
}

func ProxyConnector(ctx context.Context, req *dto.ConnectorProxyReq) (*dto.ConnectorProxyResp, error) {
	if req == nil {
		return nil, fmt.Errorf("connector proxy request 不能为空")
	}
	payload := *req
	if strings.TrimSpace(payload.ResourcePath) == "" {
		payload.ResourcePath = ConnectorGlobalResourcePath
	}
	return PostAPI[*dto.ConnectorProxyReq, *dto.ConnectorProxyResp](ctx, "/connector/api/v1/proxy", &payload)
}
