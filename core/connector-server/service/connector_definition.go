package service

import (
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/config"
)

type ConnectorDefinition struct {
	Provider     config.ConnectorOAuthProviderConfig
	Capabilities dto.ConnectorProviderCapabilities
}

func (d ConnectorDefinition) OAuthProvider() config.ConnectorOAuthProviderConfig {
	provider := d.Provider
	provider.Code = normalizeProvider(provider.Code)
	return provider
}

func oauthProvidersFromDefinitions(definitions map[string]ConnectorDefinition) map[string]config.ConnectorOAuthProviderConfig {
	providers := make(map[string]config.ConnectorOAuthProviderConfig, len(definitions))
	for code, definition := range definitions {
		provider := definition.OAuthProvider()
		if provider.Code == "" {
			provider.Code = code
		}
		providers[provider.Code] = provider
	}
	return providers
}

func (r *OAuthProviderRegistry) Capabilities(code string) dto.ConnectorProviderCapabilities {
	code = normalizeProvider(code)
	adapterCapabilities := connectorProviderCapabilities(code)
	if definition, ok := r.definitions[code]; ok {
		return mergeConnectorProviderCapabilities(definition.Capabilities, adapterCapabilities)
	}
	return adapterCapabilities
}

func builtInConnectorDefinitions() map[string]ConnectorDefinition {
	trueValue := true
	falseValue := false
	definitions := map[string]ConnectorDefinition{
		"github": {
			Provider: config.ConnectorOAuthProviderConfig{
				Code:               "github",
				Name:               "GitHub",
				ClientIDEnv:        "KAGEOS_OAUTH_GITHUB_CLIENT_ID",
				ClientSecretEnv:    "KAGEOS_OAUTH_GITHUB_CLIENT_SECRET",
				AuthURL:            "https://github.com/login/oauth/authorize",
				TokenURL:           "https://github.com/login/oauth/access_token",
				UserInfoURL:        "https://api.github.com/user",
				UsePKCE:            &trueValue,
				ExternalIDField:    "id",
				DisplayNameField:   "login",
				ProviderAccountURL: "https://github.com",
				LogoURL:            "https://github.githubassets.com/favicons/favicon.svg",
				BrandColor:         "#24292f",
			},
			Capabilities: connectorProviderCapabilities("github"),
		},
		"notion": {
			Provider: config.ConnectorOAuthProviderConfig{
				Code:               "notion",
				Name:               "Notion",
				ClientIDEnv:        "KAGEOS_OAUTH_NOTION_CLIENT_ID",
				ClientSecretEnv:    "KAGEOS_OAUTH_NOTION_CLIENT_SECRET",
				AuthURL:            "https://api.notion.com/v1/oauth/authorize",
				TokenURL:           "https://api.notion.com/v1/oauth/token",
				UserInfoURL:        "https://api.notion.com/v1/users/me",
				TokenRequestMode:   tokenRequestModeJSONBasic,
				UsePKCE:            &falseValue,
				ExtraAuthParams:    map[string]string{"owner": "user"},
				ExternalIDField:    "id",
				DisplayNameField:   "name",
				ProviderAccountURL: "https://www.notion.so",
				LogoURL:            "https://www.notion.so/images/favicon.ico",
				BrandColor:         "#000000",
			},
			Capabilities: connectorProviderCapabilities("notion"),
		},
		"gitlab": {
			Provider: config.ConnectorOAuthProviderConfig{
				Code:               "gitlab",
				Name:               "GitLab",
				ClientIDEnv:        "KAGEOS_OAUTH_GITLAB_CLIENT_ID",
				ClientSecretEnv:    "KAGEOS_OAUTH_GITLAB_CLIENT_SECRET",
				AuthURL:            "https://gitlab.com/oauth/authorize",
				TokenURL:           "https://gitlab.com/oauth/token",
				UserInfoURL:        "https://gitlab.com/api/v4/user",
				Scopes:             []string{"read_user"},
				UsePKCE:            &trueValue,
				ExternalIDField:    "id",
				DisplayNameField:   "username",
				ProviderAccountURL: "https://gitlab.com",
				LogoURL:            "https://about.gitlab.com/ico/favicon.ico",
				BrandColor:         "#fc6d26",
			},
			Capabilities: connectorProviderCapabilities("gitlab"),
		},
		"google": {
			Provider: config.ConnectorOAuthProviderConfig{
				Code:               "google",
				Name:               "Google",
				ClientIDEnv:        "KAGEOS_OAUTH_GOOGLE_CLIENT_ID",
				ClientSecretEnv:    "KAGEOS_OAUTH_GOOGLE_CLIENT_SECRET",
				AuthURL:            "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL:           "https://oauth2.googleapis.com/token",
				UserInfoURL:        "https://openidconnect.googleapis.com/v1/userinfo",
				Scopes:             []string{"openid", "email", "profile"},
				UsePKCE:            &trueValue,
				ExtraAuthParams:    map[string]string{"prompt": "consent"},
				ExternalIDField:    "sub",
				DisplayNameField:   "email",
				ProviderAccountURL: "https://myaccount.google.com",
				LogoURL:            "https://www.google.com/favicon.ico",
				BrandColor:         "#4285f4",
			},
			Capabilities: connectorProviderCapabilities("google"),
		},
		"microsoft": {
			Provider: config.ConnectorOAuthProviderConfig{
				Code:               "microsoft",
				Name:               "Microsoft",
				ClientIDEnv:        "KAGEOS_OAUTH_MICROSOFT_CLIENT_ID",
				ClientSecretEnv:    "KAGEOS_OAUTH_MICROSOFT_CLIENT_SECRET",
				AuthURL:            "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
				TokenURL:           "https://login.microsoftonline.com/common/oauth2/v2.0/token",
				UserInfoURL:        "https://graph.microsoft.com/v1.0/me",
				Scopes:             []string{"openid", "offline_access", "profile", "email", "User.Read"},
				UsePKCE:            &trueValue,
				ExternalIDField:    "id",
				DisplayNameField:   "userPrincipalName",
				ProviderAccountURL: "https://account.microsoft.com",
				LogoURL:            "https://www.microsoft.com/favicon.ico",
				BrandColor:         "#5e5e5e",
			},
			Capabilities: connectorProviderCapabilities("microsoft"),
		},
		"slack": {
			Provider: config.ConnectorOAuthProviderConfig{
				Code:               "slack",
				Name:               "Slack",
				ClientIDEnv:        "KAGEOS_OAUTH_SLACK_CLIENT_ID",
				ClientSecretEnv:    "KAGEOS_OAUTH_SLACK_CLIENT_SECRET",
				AuthURL:            "https://slack.com/oauth/v2/authorize",
				TokenURL:           "https://slack.com/api/oauth.v2.access",
				Scopes:             []string{"identity.basic"},
				UsePKCE:            &falseValue,
				ProviderAccountURL: "https://slack.com",
				LogoURL:            "https://a.slack-edge.com/80588/marketing/img/meta/favicon-32.png",
				BrandColor:         "#4a154b",
			},
			Capabilities: connectorProviderCapabilities("slack"),
		},
		"feishu": {
			Provider: config.ConnectorOAuthProviderConfig{
				Code:               "feishu",
				Name:               "飞书",
				ClientIDEnv:        "KAGEOS_OAUTH_FEISHU_CLIENT_ID",
				ClientSecretEnv:    "KAGEOS_OAUTH_FEISHU_CLIENT_SECRET",
				AuthURL:            "https://accounts.feishu.cn/open-apis/authen/v1/authorize",
				TokenURL:           "https://open.feishu.cn/open-apis/authen/v2/oauth/token",
				TokenRequestMode:   tokenRequestModeJSON,
				UsePKCE:            &trueValue,
				ExternalIDField:    "open_id",
				DisplayNameField:   "name",
				ProviderAccountURL: "https://www.feishu.cn",
				BrandColor:         "#00d6b9",
			},
			Capabilities: connectorProviderCapabilities("feishu"),
		},
		"lark": {
			Provider: config.ConnectorOAuthProviderConfig{
				Code:               "lark",
				Name:               "Lark",
				ClientIDEnv:        "KAGEOS_OAUTH_LARK_CLIENT_ID",
				ClientSecretEnv:    "KAGEOS_OAUTH_LARK_CLIENT_SECRET",
				AuthURL:            "https://accounts.larksuite.com/open-apis/authen/v1/authorize",
				TokenURL:           "https://open.larksuite.com/open-apis/authen/v2/oauth/token",
				TokenRequestMode:   tokenRequestModeJSON,
				UsePKCE:            &trueValue,
				ExternalIDField:    "open_id",
				DisplayNameField:   "name",
				ProviderAccountURL: "https://www.larksuite.com",
				BrandColor:         "#00d6b9",
			},
			Capabilities: connectorProviderCapabilities("lark"),
		},
		"dingtalk": {
			Provider: config.ConnectorOAuthProviderConfig{
				Code:               "dingtalk",
				Name:               "钉钉",
				ClientIDEnv:        "KAGEOS_OAUTH_DINGTALK_CLIENT_ID",
				ClientSecretEnv:    "KAGEOS_OAUTH_DINGTALK_CLIENT_SECRET",
				AuthURL:            "https://login.dingtalk.com/oauth2/auth",
				TokenURL:           "https://api.dingtalk.com/v1.0/oauth2/userAccessToken",
				TokenRequestMode:   tokenRequestModeJSON,
				ClientIDParam:      "clientId",
				ClientSecretParam:  "clientSecret",
				GrantTypeParam:     "grantType",
				RefreshTokenParam:  "refreshToken",
				UsePKCE:            &falseValue,
				ExternalIDField:    "unionId",
				DisplayNameField:   "nick",
				ProviderAccountURL: "https://www.dingtalk.com",
				BrandColor:         "#1677ff",
			},
			Capabilities: connectorProviderCapabilities("dingtalk"),
		},
	}
	for code, definition := range registeredConnectorDefinitions() {
		definition.Capabilities = mergeConnectorProviderCapabilities(definition.Capabilities, connectorProviderCapabilities(code))
		definitions[code] = definition
	}
	return definitions
}
