package service

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/config"
)

const (
	tokenRequestModeJSON      = "json"
	tokenRequestModeJSONBasic = "json_basic"
)

type OAuthProviderRegistry struct {
	definitions map[string]ConnectorDefinition
	providers   map[string]config.ConnectorOAuthProviderConfig
}

type OAuthTokenPayload struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	Scopes       string
	Expiry       *time.Time
	RawResponse  string
}

func NewOAuthProviderRegistry(cfg config.ConnectorOAuthConfig) *OAuthProviderRegistry {
	definitions := builtInConnectorDefinitions()
	providers := oauthProvidersFromDefinitions(definitions)
	for _, provider := range cfg.Providers {
		provider.Code = normalizeProvider(provider.Code)
		if provider.Code == "" {
			continue
		}
		base := providers[provider.Code]
		providers[provider.Code] = mergeOAuthProvider(base, provider)
		definitions[provider.Code] = ConnectorDefinition{
			Provider:     providers[provider.Code],
			Capabilities: connectorProviderCapabilities(provider.Code),
		}
	}
	return &OAuthProviderRegistry{definitions: definitions, providers: providers}
}

func (r *OAuthProviderRegistry) Get(code string) (config.ConnectorOAuthProviderConfig, error) {
	code = normalizeProvider(code)
	provider, ok := r.providers[code]
	if !ok {
		return config.ConnectorOAuthProviderConfig{}, fmt.Errorf("未配置 OAuth provider: %s", code)
	}
	provider.Code = code
	return validateOAuthProvider(provider)
}

func (r *OAuthProviderRegistry) Lookup(code string) (config.ConnectorOAuthProviderConfig, bool) {
	code = normalizeProvider(code)
	provider, ok := r.providers[code]
	if !ok {
		return config.ConnectorOAuthProviderConfig{}, false
	}
	provider.Code = code
	return provider, true
}

func (r *OAuthProviderRegistry) List() map[string]config.ConnectorOAuthProviderConfig {
	out := make(map[string]config.ConnectorOAuthProviderConfig, len(r.providers))
	for code, provider := range r.providers {
		provider.Code = code
		out[code] = provider
	}
	return out
}

func (r *OAuthProviderRegistry) Definitions() map[string]ConnectorDefinition {
	out := make(map[string]ConnectorDefinition, len(r.definitions))
	for code, definition := range r.definitions {
		definition.Provider.Code = code
		out[code] = definition
	}
	return out
}

func validateOAuthProvider(provider config.ConnectorOAuthProviderConfig) (config.ConnectorOAuthProviderConfig, error) {
	authType, err := normalizeConnectorAuthType(provider.AuthType)
	if err != nil {
		return config.ConnectorOAuthProviderConfig{}, err
	}
	provider.AuthType = authType
	provider.ClientID = strings.TrimSpace(firstNonEmpty(provider.ClientID, os.Getenv(provider.ClientIDEnv)))
	provider.ClientSecret = strings.TrimSpace(firstNonEmpty(provider.ClientSecret, os.Getenv(provider.ClientSecretEnv)))
	if provider.ClientID == "" {
		return config.ConnectorOAuthProviderConfig{}, fmt.Errorf("OAuth provider %s 缺少 client_id", provider.Code)
	}
	if provider.ClientSecret == "" {
		return config.ConnectorOAuthProviderConfig{}, fmt.Errorf("OAuth provider %s 缺少 client_secret", provider.Code)
	}
	if provider.AuthURL == "" || provider.TokenURL == "" {
		return config.ConnectorOAuthProviderConfig{}, fmt.Errorf("OAuth provider %s 缺少 auth_url 或 token_url", provider.Code)
	}
	return provider, nil
}
