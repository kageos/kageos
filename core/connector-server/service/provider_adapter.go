package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/config"
)

type ConnectorProvider interface {
	Code() string
	Definition() ConnectorDefinition
	Adapter() ConnectorProviderAdapter
}

type ConnectorProviderAdapter interface {
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

var builtInConnectorProviderAdapters = map[string]ConnectorProviderAdapter{
	"github": githubProviderAdapter{defaultProviderAdapter: defaultProviderAdapter{code: "github"}},
	"notion": notionProviderAdapter{defaultProviderAdapter: defaultProviderAdapter{code: "notion"}},
}

var (
	connectorProviderRegistryMu         sync.RWMutex
	registeredConnectorProviderDefs     = map[string]ConnectorDefinition{}
	registeredConnectorProviderAdapters = map[string]ConnectorProviderAdapter{}
)

func RegisterConnectorProvider(provider ConnectorProvider) error {
	if provider == nil {
		return fmt.Errorf("connector provider 不能为空")
	}
	code := normalizeProvider(provider.Code())
	if code == "" {
		return fmt.Errorf("connector provider code 不能为空")
	}
	definition := provider.Definition()
	if definition.Provider.Code == "" {
		definition.Provider.Code = code
	}
	definition.Provider.Code = normalizeProvider(definition.Provider.Code)
	if definition.Provider.Code != code {
		return fmt.Errorf("connector provider code 不匹配: provider=%s definition=%s", code, definition.Provider.Code)
	}
	adapter := provider.Adapter()
	if adapter != nil {
		adapterCode := normalizeProvider(adapter.Code())
		if adapterCode == "" {
			return fmt.Errorf("connector provider adapter code 不能为空")
		}
		if adapterCode != code {
			return fmt.Errorf("connector provider code 不匹配: provider=%s adapter=%s", code, adapterCode)
		}
		if connectorProviderCapabilitiesEmpty(definition.Capabilities) {
			definition.Capabilities = adapter.Capabilities()
		}
	}
	if connectorProviderCapabilitiesEmpty(definition.Capabilities) {
		definition.Capabilities = NewDefaultConnectorProviderAdapter(code).Capabilities()
	}

	connectorProviderRegistryMu.Lock()
	defer connectorProviderRegistryMu.Unlock()
	registeredConnectorProviderDefs[code] = definition
	if adapter != nil {
		registeredConnectorProviderAdapters[code] = adapter
	}
	return nil
}

func RegisterConnectorProviderDefinition(definition ConnectorDefinition) error {
	code := normalizeProvider(definition.Provider.Code)
	if code == "" {
		return fmt.Errorf("connector provider definition code 不能为空")
	}
	definition.Provider.Code = code
	if connectorProviderCapabilitiesEmpty(definition.Capabilities) {
		definition.Capabilities = connectorAdapterFor(code).Capabilities()
	}
	connectorProviderRegistryMu.Lock()
	defer connectorProviderRegistryMu.Unlock()
	registeredConnectorProviderDefs[code] = definition
	return nil
}

func RegisterConnectorProviderAdapter(adapter ConnectorProviderAdapter) error {
	if adapter == nil {
		return fmt.Errorf("connector provider adapter 不能为空")
	}
	code := normalizeProvider(adapter.Code())
	if code == "" {
		return fmt.Errorf("connector provider adapter code 不能为空")
	}
	connectorProviderRegistryMu.Lock()
	defer connectorProviderRegistryMu.Unlock()
	registeredConnectorProviderAdapters[code] = adapter
	return nil
}

func registeredConnectorDefinitions() map[string]ConnectorDefinition {
	connectorProviderRegistryMu.RLock()
	defer connectorProviderRegistryMu.RUnlock()
	out := make(map[string]ConnectorDefinition, len(registeredConnectorProviderDefs))
	for code, definition := range registeredConnectorProviderDefs {
		definition.Provider.Code = code
		out[code] = definition
	}
	return out
}

func registeredConnectorAdapterFor(provider string) (ConnectorProviderAdapter, bool) {
	code := normalizeProvider(provider)
	connectorProviderRegistryMu.RLock()
	defer connectorProviderRegistryMu.RUnlock()
	adapter, ok := registeredConnectorProviderAdapters[code]
	return adapter, ok
}

func unregisterConnectorProviderForTest(code string) {
	code = normalizeProvider(code)
	connectorProviderRegistryMu.Lock()
	defer connectorProviderRegistryMu.Unlock()
	delete(registeredConnectorProviderDefs, code)
	delete(registeredConnectorProviderAdapters, code)
}

func connectorAdapterFor(provider string) ConnectorProviderAdapter {
	code := normalizeProvider(provider)
	if adapter, ok := registeredConnectorAdapterFor(code); ok {
		return adapter
	}
	if adapter, ok := builtInConnectorProviderAdapters[code]; ok {
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

type DefaultConnectorProviderAdapter struct {
	code string
}

type defaultProviderAdapter = DefaultConnectorProviderAdapter

func NewDefaultConnectorProviderAdapter(code string) DefaultConnectorProviderAdapter {
	return DefaultConnectorProviderAdapter{code: normalizeProvider(code)}
}

func (a DefaultConnectorProviderAdapter) Code() string {
	return a.code
}

func (a DefaultConnectorProviderAdapter) Capabilities() dto.ConnectorProviderCapabilities {
	return dto.ConnectorProviderCapabilities{
		OAuthSupported: true,
	}
}

func (a DefaultConnectorProviderAdapter) ProxyBaseURL() string {
	return ""
}

func (a DefaultConnectorProviderAdapter) UseAccessTypeOffline() bool {
	return true
}

func (a DefaultConnectorProviderAdapter) BuildAuthorizeURL(provider config.ConnectorOAuthProviderConfig, redirectURL, state, codeChallenge string, scopes []string) string {
	return buildOAuthAuthorizeURL(provider, redirectURL, state, codeChallenge, scopes)
}

func (a DefaultConnectorProviderAdapter) ExchangeToken(ctx context.Context, provider config.ConnectorOAuthProviderConfig, redirectURL, code, codeVerifier string, scopes []string) (*OAuthTokenPayload, error) {
	return exchangeOAuthToken(ctx, provider, redirectURL, code, codeVerifier, scopes)
}

func (a DefaultConnectorProviderAdapter) RefreshToken(ctx context.Context, provider config.ConnectorOAuthProviderConfig, refreshToken string, scopes []string) (*OAuthTokenPayload, error) {
	return refreshOAuthToken(ctx, provider, refreshToken, scopes)
}

func (a DefaultConnectorProviderAdapter) MergeReconnectScopes(provider config.ConnectorOAuthProviderConfig, existingScopes, requestedScopes []string) []string {
	scopes := make([]string, 0, len(provider.Scopes)+len(existingScopes)+len(requestedScopes))
	scopes = append(scopes, provider.Scopes...)
	scopes = append(scopes, existingScopes...)
	scopes = append(scopes, requestedScopes...)
	return cleanScopes(scopes)
}

func (a DefaultConnectorProviderAdapter) MissingScopes(granted, required []string) []string {
	return defaultMissingScopes(granted, required, defaultScopeGranted)
}

func (a DefaultConnectorProviderAdapter) BuildProxyRequest(ctx context.Context, method, targetURL, accessToken string, req dto.ConnectorProxyReq) (*http.Request, error) {
	provider := a.Code()
	if provider == "" {
		provider = req.Provider
	}
	return buildDefaultConnectorProxyRequest(ctx, method, targetURL, accessToken, provider, req)
}

func (a DefaultConnectorProviderAdapter) TranslateProxyError(statusCode int, body []byte) error {
	return fmt.Errorf("%s API 返回 %d: %s", a.Code(), statusCode, strings.TrimSpace(string(body)))
}

func (a DefaultConnectorProviderAdapter) DecorateAPIRequest(req *http.Request) {}

func (a DefaultConnectorProviderAdapter) EnrichOAuthProfile(ctx context.Context, provider config.ConnectorOAuthProviderConfig, tokenPayload *OAuthTokenPayload, userInfo map[string]interface{}, profile *dto.ConnectorConnectionProfile, metadata map[string]interface{}) error {
	return nil
}

func connectorProviderCapabilitiesEmpty(capabilities dto.ConnectorProviderCapabilities) bool {
	return !capabilities.OAuthSupported &&
		!capabilities.ProxySupported &&
		!capabilities.ProfileSupported &&
		!capabilities.ResourceSummarySupported
}

func mergeConnectorProviderCapabilities(base, override dto.ConnectorProviderCapabilities) dto.ConnectorProviderCapabilities {
	return dto.ConnectorProviderCapabilities{
		OAuthSupported:           base.OAuthSupported || override.OAuthSupported,
		ProxySupported:           base.ProxySupported || override.ProxySupported,
		ProfileSupported:         base.ProfileSupported || override.ProfileSupported,
		ResourceSummarySupported: base.ResourceSummarySupported || override.ResourceSummarySupported,
	}
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
