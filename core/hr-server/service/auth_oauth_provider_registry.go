package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/oauth2"
)

type OAuthProfile struct {
	ProviderCode      string
	ExternalID        string
	Email             string
	EmailVerified     bool
	PreferredUsername string
	Nickname          string
	Avatar            string
}

type OAuthProviderFactory struct {
	OAuth2Config      func(values map[string]string) (*oauth2.Config, error)
	FetchProfile      func(ctx context.Context, client *http.Client) (*OAuthProfile, error)
	AuthCodeOptions   func(values map[string]string) []oauth2.AuthCodeOption
	BuildAuthorizeURL func(values map[string]string, state string) (string, error)
	ExchangeProfile   func(ctx context.Context, values map[string]string, code string) (*OAuthProfile, error)
	DisplayName       string
	ShortCode         string
	RegisterType      string
	UsePKCE           bool
}

// OAuthLoginProvider bundles login configuration metadata with runtime OAuth behavior.
type OAuthLoginProvider struct {
	Seed    AuthProviderSeed
	Factory OAuthProviderFactory
	Aliases []string
}

var oauthProviderRegistry = struct {
	sync.RWMutex
	providers map[string]OAuthProviderFactory
	aliases   map[string]string
}{
	providers: make(map[string]OAuthProviderFactory),
	aliases:   make(map[string]string),
}

// RegisterOAuthLoginProvider registers both the admin-facing login provider seed
// and the runtime OAuth provider factory. Prefer this for OAuth/OIDC login providers.
func RegisterOAuthLoginProvider(provider OAuthLoginProvider) {
	provider.Seed.Code = normalizeProviderCode(provider.Seed.Code)
	if provider.Seed.Code == "" || !validOAuthProviderFactory(provider.Factory) {
		panic("oauth login provider requires a seed code and a complete standard or custom OAuth flow")
	}
	RegisterAuthProviderSeed(provider.Seed)
	RegisterOAuthProvider(provider.Seed.Code, provider.Factory, provider.Aliases...)
}

// RegisterOAuthProvider registers only the runtime OAuth behavior.
func RegisterOAuthProvider(code string, factory OAuthProviderFactory, aliases ...string) {
	code = normalizeProviderCode(code)
	if code == "" || !validOAuthProviderFactory(factory) {
		panic("oauth provider requires a code and a complete standard or custom OAuth flow")
	}
	oauthProviderRegistry.Lock()
	defer oauthProviderRegistry.Unlock()
	if _, exists := oauthProviderRegistry.providers[code]; exists {
		panic(fmt.Sprintf("oauth provider %s already registered", code))
	}
	oauthProviderRegistry.providers[code] = factory
	oauthProviderRegistry.aliases[code] = code
	for _, alias := range aliases {
		alias = normalizeProviderCode(alias)
		if alias != "" {
			if existing, exists := oauthProviderRegistry.aliases[alias]; exists && existing != code {
				panic(fmt.Sprintf("oauth provider alias %s already registered for %s", alias, existing))
			}
			oauthProviderRegistry.aliases[alias] = code
		}
	}
}

func validOAuthProviderFactory(factory OAuthProviderFactory) bool {
	standard := factory.OAuth2Config != nil && factory.FetchProfile != nil
	custom := factory.BuildAuthorizeURL != nil && factory.ExchangeProfile != nil
	return standard || custom
}

func GetOAuthProvider(code string) (OAuthProviderFactory, bool) {
	code = normalizeProviderCode(code)
	oauthProviderRegistry.RLock()
	defer oauthProviderRegistry.RUnlock()
	factory, ok := oauthProviderRegistry.providers[code]
	return factory, ok
}

func LookupOAuthProviderCode(alias string) (string, bool) {
	alias = normalizeProviderCode(alias)
	oauthProviderRegistry.RLock()
	defer oauthProviderRegistry.RUnlock()
	code, ok := oauthProviderRegistry.aliases[alias]
	return code, ok
}

func init() {
	RegisterOAuthLoginProvider(OAuthLoginProvider{
		Seed: AuthProviderSeed{
			Code:          ProviderGoogleOAuth,
			Name:          "Google 登录",
			Description:   "使用 Google OAuth 账号登录。",
			Action:        ProviderActionRedirect,
			AuthorizePath: "/hr/api/v1/auth/google/authorize",
			CallbackPath:  "/hr/api/v1/auth/google/callback",
			DocsURL:       "https://developers.google.com/identity/protocols/oauth2",
			SortOrder:     10,
			Fields: []AuthProviderFieldDef{
				{Key: "client_id", Label: "Client ID", Type: "text", Required: true},
				{Key: "client_secret", Label: "Client Secret", Type: "password", Required: true, Secret: true},
				{Key: "redirect_url", Label: "Redirect URL", Type: "url", Required: true, Help: "需要和 Google Cloud Console 中配置的回调地址一致。"},
			},
		},
		Factory: OAuthProviderFactory{
			OAuth2Config: googleOAuth2Config,
			FetchProfile: fetchGoogleProfile,
			AuthCodeOptions: func(map[string]string) []oauth2.AuthCodeOption {
				return []oauth2.AuthCodeOption{oauth2.SetAuthURLParam("prompt", "select_account")}
			},
			DisplayName:  "Google",
			ShortCode:    "google",
			RegisterType: "google",
		},
		Aliases: []string{"google"},
	})

	RegisterOAuthLoginProvider(OAuthLoginProvider{
		Seed: AuthProviderSeed{
			Code:          ProviderGitHubOAuth,
			Name:          "GitHub 登录",
			Description:   "使用 GitHub OAuth 账号登录。",
			Action:        ProviderActionRedirect,
			AuthorizePath: "/hr/api/v1/auth/github/authorize",
			CallbackPath:  "/hr/api/v1/auth/github/callback",
			DocsURL:       "https://docs.github.com/apps/oauth-apps",
			SortOrder:     20,
			Fields: []AuthProviderFieldDef{
				{Key: "client_id", Label: "Client ID", Type: "text", Required: true},
				{Key: "client_secret", Label: "Client Secret", Type: "password", Required: true, Secret: true},
				{Key: "redirect_url", Label: "Redirect URL", Type: "url", Required: true, Help: "需要和 GitHub OAuth App 中配置的 Authorization callback URL 一致。"},
				{Key: "scopes", Label: "Scopes", Type: "text", Required: false, Placeholder: "read:user user:email"},
			},
		},
		Factory: OAuthProviderFactory{
			OAuth2Config: githubOAuth2Config,
			FetchProfile: fetchGitHubProfile,
			DisplayName:  "GitHub",
			ShortCode:    "github",
			RegisterType: "github",
		},
		Aliases: []string{"github"},
	})

	RegisterOAuthLoginProvider(OAuthLoginProvider{
		Seed: AuthProviderSeed{
			Code:          ProviderKageOSAuth,
			Name:          "kageos 微信扫码登录",
			Description:   "通过 kageos 第一方统一认证服务使用微信公众号扫码登录。",
			Action:        ProviderActionRedirect,
			AuthorizePath: "/hr/api/v1/auth/kageos/authorize",
			CallbackPath:  "/hr/api/v1/auth/kageos/callback",
			DocsURL:       "https://auth.kageos.com",
			SortOrder:     30,
			Fields: []AuthProviderFieldDef{
				{Key: "client_id", Label: "Client ID", Type: "text", Required: true},
				{Key: "client_secret", Label: "Client Secret", Type: "password", Required: true, Secret: true},
				{Key: "redirect_url", Label: "回调地址", Type: "url", Required: true, Help: "必须是 kageos Auth 白名单中登记的精确回调地址。"},
			},
		},
		Factory: OAuthProviderFactory{
			OAuth2Config: kageosAuthOAuth2Config,
			FetchProfile: fetchKageOSAuthProfile,
			DisplayName:  "kageos Auth",
			ShortCode:    "kageos",
			RegisterType: "kageos_auth",
			UsePKCE:      true,
		},
		Aliases: []string{"kageos", "kageos-auth"},
	})

	RegisterOAuthLoginProvider(OAuthLoginProvider{
		Seed: AuthProviderSeed{
			Code:          ProviderWechatOpenOAuth,
			Name:          "微信开放平台登录",
			Description:   "使用微信开放平台网站应用授权登录。",
			Action:        ProviderActionRedirect,
			AuthorizePath: "/hr/api/v1/auth/wechat-open/authorize",
			CallbackPath:  "/hr/api/v1/auth/wechat-open/callback",
			DocsURL:       "https://open.weixin.qq.com/",
			SortOrder:     40,
			Fields: []AuthProviderFieldDef{
				{Key: "app_id", Label: "AppID", Type: "text", Required: true},
				{Key: "app_secret", Label: "AppSecret", Type: "password", Required: true, Secret: true},
				{Key: "redirect_url", Label: "回调地址", Type: "url", Required: true, Help: "需要和微信开放平台网站应用配置的授权回调域一致。"},
			},
		},
		Factory: OAuthProviderFactory{
			BuildAuthorizeURL: buildWechatOpenAuthorizeURL,
			ExchangeProfile:   exchangeWechatOpenProfile,
			DisplayName:       "微信",
			ShortCode:         "wechat",
			RegisterType:      "wechat",
		},
		Aliases: []string{"wechat-open", "wechat_open"},
	})

	RegisterAuthProviderSeed(AuthProviderSeed{
		Code:          ProviderWechatOfficial,
		Name:          "微信公众号扫码登录",
		Description:   "使用已认证公众号的带参二维码登录；未关注用户关注后也可完成登录。",
		Action:        ProviderActionQRCode,
		AuthorizePath: "/hr/api/v1/auth/wechat/attempts",
		CallbackPath:  "/hr/api/v1/auth/wechat/callback",
		DocsURL:       "https://mp.weixin.qq.com/",
		SortOrder:     50,
		Fields: []AuthProviderFieldDef{
			{Key: "app_id", Label: "AppID", Type: "text", Required: true},
			{Key: "app_secret", Label: "AppSecret", Type: "password", Required: true, Secret: true},
			{Key: "message_token", Label: "消息 Token", Type: "password", Required: true, Secret: true, Help: "必须和公众号服务器配置中的 Token 完全一致；首版使用明文消息模式。"},
		},
	})
}

func kageosAuthOAuth2Config(values map[string]string) (*oauth2.Config, error) {
	clientID, clientSecret, redirectURL, err := oauthClientValues(values)
	if err != nil {
		return nil, err
	}
	return &oauth2.Config{
		ClientID: clientID, ClientSecret: clientSecret, RedirectURL: redirectURL,
		Scopes: []string{"openid", "profile"},
		Endpoint: oauth2.Endpoint{
			AuthURL: "https://auth.kageos.com/api/v1/oauth/authorize", TokenURL: "https://auth.kageos.com/api/v1/oauth/token",
			AuthStyle: oauth2.AuthStyleInHeader,
		},
	}, nil
}

func fetchKageOSAuthProfile(ctx context.Context, client *http.Client) (*OAuthProfile, error) {
	var payload struct {
		Subject           string `json:"sub"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		Picture           string `json:"picture"`
	}
	if err := getJSON(ctx, client, "https://auth.kageos.com/api/v1/oauth/userinfo", &payload); err != nil {
		return nil, fmt.Errorf("获取 kageos Auth 用户信息失败: %w", err)
	}
	if strings.TrimSpace(payload.Subject) == "" {
		return nil, fmt.Errorf("kageos Auth 未返回有效用户标识")
	}
	return &OAuthProfile{
		ProviderCode: ProviderKageOSAuth, ExternalID: payload.Subject, PreferredUsername: payload.PreferredUsername,
		Nickname: payload.Name, Avatar: payload.Picture,
	}, nil
}

func googleOAuth2Config(values map[string]string) (*oauth2.Config, error) {
	clientID, clientSecret, redirectURL, err := oauthClientValues(values)
	if err != nil {
		return nil, err
	}
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}, nil
}

func githubOAuth2Config(values map[string]string) (*oauth2.Config, error) {
	clientID, clientSecret, redirectURL, err := oauthClientValues(values)
	if err != nil {
		return nil, err
	}
	scopes := strings.Fields(values["scopes"])
	if len(scopes) == 0 {
		scopes = []string{"read:user", "user:email"}
	}
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://github.com/login/oauth/authorize",
			TokenURL: "https://github.com/login/oauth/access_token",
		},
	}, nil
}

func oauthClientValues(values map[string]string) (string, string, string, error) {
	clientID := strings.TrimSpace(values["client_id"])
	clientSecret := strings.TrimSpace(values["client_secret"])
	redirectURL := strings.TrimSpace(values["redirect_url"])
	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return "", "", "", fmt.Errorf("授权配置不完整")
	}
	return clientID, clientSecret, redirectURL, nil
}
