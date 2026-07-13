package service

import (
	"net/url"
	"strings"

	"github.com/kageos/kageos/pkg/config"
)

func providerUsesPKCE(provider config.ConnectorOAuthProviderConfig) bool {
	if provider.UsePKCE == nil {
		return true
	}
	return *provider.UsePKCE
}

func providerUsesAccessTypeOffline(provider config.ConnectorOAuthProviderConfig) bool {
	return connectorAdapterFor(provider.Code).UseAccessTypeOffline()
}

func paramName(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func mergeOAuthProvider(base, override config.ConnectorOAuthProviderConfig) config.ConnectorOAuthProviderConfig {
	if override.Name != "" {
		base.Name = override.Name
	}
	if override.AuthType != "" {
		base.AuthType = override.AuthType
	}
	if override.ClientID != "" {
		base.ClientID = override.ClientID
	}
	if override.ClientSecret != "" {
		base.ClientSecret = override.ClientSecret
	}
	if override.ClientIDEnv != "" {
		base.ClientIDEnv = override.ClientIDEnv
	}
	if override.ClientSecretEnv != "" {
		base.ClientSecretEnv = override.ClientSecretEnv
	}
	if override.AuthURL != "" {
		base.AuthURL = override.AuthURL
	}
	if override.TokenURL != "" {
		base.TokenURL = override.TokenURL
	}
	if override.UserInfoURL != "" {
		base.UserInfoURL = override.UserInfoURL
	}
	if len(override.Scopes) > 0 {
		base.Scopes = override.Scopes
	}
	if override.UsePKCE != nil {
		base.UsePKCE = override.UsePKCE
	}
	if override.TokenRequestMode != "" {
		base.TokenRequestMode = override.TokenRequestMode
	}
	if override.ClientIDParam != "" {
		base.ClientIDParam = override.ClientIDParam
	}
	if override.ClientSecretParam != "" {
		base.ClientSecretParam = override.ClientSecretParam
	}
	if override.GrantTypeParam != "" {
		base.GrantTypeParam = override.GrantTypeParam
	}
	if override.CodeParam != "" {
		base.CodeParam = override.CodeParam
	}
	if override.RefreshTokenParam != "" {
		base.RefreshTokenParam = override.RefreshTokenParam
	}
	if override.RedirectURIParam != "" {
		base.RedirectURIParam = override.RedirectURIParam
	}
	if len(override.ExtraAuthParams) > 0 {
		base.ExtraAuthParams = override.ExtraAuthParams
	}
	if len(override.ExtraTokenParams) > 0 {
		base.ExtraTokenParams = override.ExtraTokenParams
	}
	if override.ExternalIDField != "" {
		base.ExternalIDField = override.ExternalIDField
	}
	if override.DisplayNameField != "" {
		base.DisplayNameField = override.DisplayNameField
	}
	if override.ProviderAccountURL != "" {
		base.ProviderAccountURL = override.ProviderAccountURL
	}
	if override.LogoURL != "" {
		base.LogoURL = override.LogoURL
	}
	if override.BrandColor != "" {
		base.BrandColor = override.BrandColor
	}
	base.Code = override.Code
	return base
}

func safeRedirectAfter(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	if parsed.IsAbs() {
		if isLocalOAuthRedirectHost(parsed) {
			return value
		}
		return ""
	}
	if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return ""
	}
	return value
}

func isLocalOAuthRedirectHost(u *url.URL) bool {
	if u == nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	host := strings.ToLower(u.Hostname())
	switch host {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}
