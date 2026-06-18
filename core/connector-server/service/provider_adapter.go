package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/config"
)

type connectorProviderAdapter interface {
	Code() string
	Capabilities() dto.ConnectorProviderCapabilities
	ProxyBaseURL() string
	UseAccessTypeOffline() bool
	BuildAuthorizeURL(provider config.ConnectorOAuthProviderConfig, redirectURL, state, codeChallenge string, scopes []string) string
	ExchangeToken(ctx context.Context, provider config.ConnectorOAuthProviderConfig, redirectURL, code, codeVerifier string, scopes []string) (*OAuthTokenPayload, error)
	RefreshToken(ctx context.Context, provider config.ConnectorOAuthProviderConfig, refreshToken string, scopes []string) (*OAuthTokenPayload, error)
	MergeReconnectScopes(provider config.ConnectorOAuthProviderConfig, existingScopes, requestedScopes []string) []string
	MissingScopes(granted, required []string) []string
	BuildProxyRequest(ctx context.Context, method, targetURL, accessToken string, req dto.ConnectorProxyReq) (*http.Request, error)
	TranslateProxyError(statusCode int, body []byte) error
	DecorateAPIRequest(req *http.Request)
	EnrichOAuthProfile(ctx context.Context, provider config.ConnectorOAuthProviderConfig, tokenPayload *OAuthTokenPayload, userInfo map[string]interface{}, profile *dto.ConnectorConnectionProfile, metadata map[string]interface{}) error
}

var connectorProviderAdapters = map[string]connectorProviderAdapter{
	"github": githubProviderAdapter{defaultProviderAdapter: defaultProviderAdapter{code: "github"}},
	"notion": notionProviderAdapter{defaultProviderAdapter: defaultProviderAdapter{code: "notion"}},
}

func connectorAdapterFor(provider string) connectorProviderAdapter {
	code := normalizeProvider(provider)
	if adapter, ok := connectorProviderAdapters[code]; ok {
		return adapter
	}
	return defaultProviderAdapter{code: code}
}

func connectorProviderCapabilities(provider string) dto.ConnectorProviderCapabilities {
	return connectorAdapterFor(provider).Capabilities()
}

func decorateProviderAPIRequest(provider string, req *http.Request) {
	if req == nil {
		return
	}
	connectorAdapterFor(provider).DecorateAPIRequest(req)
}

type defaultProviderAdapter struct {
	code string
}

func (a defaultProviderAdapter) Code() string {
	return a.code
}

func (a defaultProviderAdapter) Capabilities() dto.ConnectorProviderCapabilities {
	return dto.ConnectorProviderCapabilities{
		OAuthSupported: true,
	}
}

func (a defaultProviderAdapter) ProxyBaseURL() string {
	return ""
}

func (a defaultProviderAdapter) UseAccessTypeOffline() bool {
	return true
}

func (a defaultProviderAdapter) BuildAuthorizeURL(provider config.ConnectorOAuthProviderConfig, redirectURL, state, codeChallenge string, scopes []string) string {
	return buildOAuthAuthorizeURL(provider, redirectURL, state, codeChallenge, scopes)
}

func (a defaultProviderAdapter) ExchangeToken(ctx context.Context, provider config.ConnectorOAuthProviderConfig, redirectURL, code, codeVerifier string, scopes []string) (*OAuthTokenPayload, error) {
	return exchangeOAuthToken(ctx, provider, redirectURL, code, codeVerifier, scopes)
}

func (a defaultProviderAdapter) RefreshToken(ctx context.Context, provider config.ConnectorOAuthProviderConfig, refreshToken string, scopes []string) (*OAuthTokenPayload, error) {
	return refreshOAuthToken(ctx, provider, refreshToken, scopes)
}

func (a defaultProviderAdapter) MergeReconnectScopes(provider config.ConnectorOAuthProviderConfig, existingScopes, requestedScopes []string) []string {
	scopes := make([]string, 0, len(provider.Scopes)+len(existingScopes)+len(requestedScopes))
	scopes = append(scopes, provider.Scopes...)
	scopes = append(scopes, existingScopes...)
	scopes = append(scopes, requestedScopes...)
	return cleanScopes(scopes)
}

func (a defaultProviderAdapter) MissingScopes(granted, required []string) []string {
	return defaultMissingScopes(granted, required, defaultScopeGranted)
}

func (a defaultProviderAdapter) BuildProxyRequest(ctx context.Context, method, targetURL, accessToken string, req dto.ConnectorProxyReq) (*http.Request, error) {
	provider := a.Code()
	if provider == "" {
		provider = req.Provider
	}
	return buildDefaultConnectorProxyRequest(ctx, method, targetURL, accessToken, provider, req)
}

func (a defaultProviderAdapter) TranslateProxyError(statusCode int, body []byte) error {
	return fmt.Errorf("%s API 返回 %d: %s", a.Code(), statusCode, strings.TrimSpace(string(body)))
}

func (a defaultProviderAdapter) DecorateAPIRequest(req *http.Request) {}

func (a defaultProviderAdapter) EnrichOAuthProfile(ctx context.Context, provider config.ConnectorOAuthProviderConfig, tokenPayload *OAuthTokenPayload, userInfo map[string]interface{}, profile *dto.ConnectorConnectionProfile, metadata map[string]interface{}) error {
	return nil
}

func defaultMissingScopes(granted, required []string, grantedFunc func(map[string]struct{}, string) bool) []string {
	granted = cleanScopes(granted)
	required = cleanScopes(required)
	if len(required) == 0 {
		return nil
	}
	grantedSet := make(map[string]struct{}, len(granted))
	for _, scope := range granted {
		scope = strings.TrimSpace(scope)
		if scope != "" {
			grantedSet[scope] = struct{}{}
		}
	}
	missing := make([]string, 0)
	for _, scope := range required {
		if grantedFunc(grantedSet, strings.TrimSpace(scope)) {
			continue
		}
		missing = append(missing, scope)
	}
	return missing
}

func defaultScopeGranted(granted map[string]struct{}, required string) bool {
	if required == "" {
		return true
	}
	_, ok := granted[required]
	return ok
}
