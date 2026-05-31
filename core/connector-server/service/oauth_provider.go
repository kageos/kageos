package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/config"
	"golang.org/x/oauth2"
)

const tokenRequestModeJSON = "json"

type OAuthProviderRegistry struct {
	providers map[string]config.ConnectorOAuthProviderConfig
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
	providers := builtInOAuthProviders()
	for _, provider := range cfg.Providers {
		provider.Code = normalizeProvider(provider.Code)
		if provider.Code == "" {
			continue
		}
		base := providers[provider.Code]
		providers[provider.Code] = mergeOAuthProvider(base, provider)
	}
	return &OAuthProviderRegistry{providers: providers}
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

func validateOAuthProvider(provider config.ConnectorOAuthProviderConfig) (config.ConnectorOAuthProviderConfig, error) {
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

func (r *OAuthProviderRegistry) BuildAuthURL(provider config.ConnectorOAuthProviderConfig, redirectURL, state, codeChallenge string, scopes []string) string {
	conf := oauth2Config(provider, redirectURL, scopes)
	opts := []oauth2.AuthCodeOption{oauth2.AccessTypeOffline}
	if providerUsesPKCE(provider) {
		opts = append(opts,
			oauth2.SetAuthURLParam("code_challenge", codeChallenge),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		)
	}
	for key, value := range provider.ExtraAuthParams {
		if strings.TrimSpace(key) != "" && strings.TrimSpace(value) != "" {
			opts = append(opts, oauth2.SetAuthURLParam(key, value))
		}
	}
	return conf.AuthCodeURL(state, opts...)
}

func (r *OAuthProviderRegistry) Exchange(ctx context.Context, provider config.ConnectorOAuthProviderConfig, redirectURL, code, codeVerifier string, scopes []string) (*OAuthTokenPayload, error) {
	if strings.EqualFold(provider.TokenRequestMode, tokenRequestModeJSON) {
		return exchangeJSONToken(ctx, provider, redirectURL, code, codeVerifier, scopes)
	}
	conf := oauth2Config(provider, redirectURL, scopes)
	opts := []oauth2.AuthCodeOption{}
	if providerUsesPKCE(provider) {
		opts = append(opts, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	}
	token, err := conf.Exchange(ctx, code, opts...)
	if err != nil {
		return nil, err
	}
	return oauth2TokenPayload(token, scopes), nil
}

func (r *OAuthProviderRegistry) Refresh(ctx context.Context, provider config.ConnectorOAuthProviderConfig, refreshToken string, scopes []string) (*OAuthTokenPayload, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, fmt.Errorf("refresh_token 为空，无法刷新")
	}
	if strings.EqualFold(provider.TokenRequestMode, tokenRequestModeJSON) {
		return refreshJSONToken(ctx, provider, refreshToken, scopes)
	}
	conf := oauth2Config(provider, "", scopes)
	expired := &oauth2.Token{
		RefreshToken: refreshToken,
		Expiry:       time.Now().Add(-time.Hour),
	}
	token, err := conf.TokenSource(ctx, expired).Token()
	if err != nil {
		return nil, err
	}
	return oauth2TokenPayload(token, scopes), nil
}

func oauth2Config(provider config.ConnectorOAuthProviderConfig, redirectURL string, scopes []string) *oauth2.Config {
	if len(scopes) == 0 {
		scopes = provider.Scopes
	}
	return &oauth2.Config{
		ClientID:     provider.ClientID,
		ClientSecret: provider.ClientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  provider.AuthURL,
			TokenURL: provider.TokenURL,
		},
	}
}

func newOAuthState() (string, error) {
	return randomBase64URL(32)
}

func newPKCEVerifier() (string, string, error) {
	verifier, err := randomBase64URL(32)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func randomBase64URL(size int) (string, error) {
	data := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func exchangeJSONToken(ctx context.Context, provider config.ConnectorOAuthProviderConfig, redirectURL, code, codeVerifier string, scopes []string) (*OAuthTokenPayload, error) {
	body := jsonTokenBody(provider, "authorization_code")
	body[paramName(provider.CodeParam, "code")] = code
	if redirectURL != "" {
		body[paramName(provider.RedirectURIParam, "redirect_uri")] = redirectURL
	}
	if providerUsesPKCE(provider) {
		body["code_verifier"] = codeVerifier
	}
	for key, value := range provider.ExtraTokenParams {
		if strings.TrimSpace(key) != "" {
			body[key] = value
		}
	}
	return postJSONToken(ctx, provider.TokenURL, body, scopes)
}

func refreshJSONToken(ctx context.Context, provider config.ConnectorOAuthProviderConfig, refreshToken string, scopes []string) (*OAuthTokenPayload, error) {
	body := jsonTokenBody(provider, "refresh_token")
	body[paramName(provider.RefreshTokenParam, "refresh_token")] = refreshToken
	for key, value := range provider.ExtraTokenParams {
		if strings.TrimSpace(key) != "" {
			body[key] = value
		}
	}
	return postJSONToken(ctx, provider.TokenURL, body, scopes)
}

func jsonTokenBody(provider config.ConnectorOAuthProviderConfig, grantType string) map[string]string {
	return map[string]string{
		paramName(provider.ClientIDParam, "client_id"):         provider.ClientID,
		paramName(provider.ClientSecretParam, "client_secret"): provider.ClientSecret,
		paramName(provider.GrantTypeParam, "grant_type"):       grantType,
	}
}

func postJSONToken(ctx context.Context, tokenURL string, body map[string]string, scopes []string) (*OAuthTokenPayload, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("oauth token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return parseTokenPayload(data, scopes)
}

func fetchProviderUserInfo(ctx context.Context, provider config.ConnectorOAuthProviderConfig, accessToken string) (map[string]interface{}, error) {
	if strings.TrimSpace(provider.UserInfoURL) == "" || strings.TrimSpace(accessToken) == "" {
		return nil, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("oauth userinfo endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if dataObj, ok := raw["data"].(map[string]interface{}); ok {
		return dataObj, nil
	}
	return raw, nil
}

func oauth2TokenPayload(token *oauth2.Token, scopes []string) *OAuthTokenPayload {
	if token == nil {
		return nil
	}
	var expiry *time.Time
	if !token.Expiry.IsZero() {
		t := token.Expiry
		expiry = &t
	}
	scopeText := strings.Join(scopes, " ")
	if value := token.Extra("scope"); value != nil {
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				scopeText = typed
			}
		case []string:
			if len(typed) > 0 {
				scopeText = strings.Join(typed, " ")
			}
		}
	}
	raw := map[string]interface{}{
		"token_type":      token.TokenType,
		"expiry":          token.Expiry,
		"scopes":          scopeText,
		"access_present":  token.AccessToken != "",
		"refresh_present": token.RefreshToken != "",
	}
	rawData, _ := json.Marshal(raw)
	return &OAuthTokenPayload{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Scopes:       scopeText,
		Expiry:       expiry,
		RawResponse:  string(rawData),
	}
}

func parseTokenPayload(data []byte, scopes []string) (*OAuthTokenPayload, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	tokenRoot := raw
	if dataObj, ok := raw["data"].(map[string]interface{}); ok {
		tokenRoot = dataObj
	}
	accessToken := stringField(tokenRoot, "access_token", "accessToken")
	if accessToken == "" {
		return nil, fmt.Errorf("oauth token response 缺少 access_token")
	}
	expiry := tokenExpiry(tokenRoot)
	scopeText := stringField(tokenRoot, "scope", "scopes")
	if scopeText == "" {
		scopeText = strings.Join(scopes, " ")
	}
	return &OAuthTokenPayload{
		AccessToken:  accessToken,
		RefreshToken: stringField(tokenRoot, "refresh_token", "refreshToken"),
		TokenType:    stringField(tokenRoot, "token_type", "tokenType"),
		Scopes:       scopeText,
		Expiry:       expiry,
		RawResponse:  string(data),
	}, nil
}

func tokenExpiry(raw map[string]interface{}) *time.Time {
	seconds := numberField(raw, "expires_in", "expiresIn", "expire_in", "expireIn")
	if seconds <= 0 {
		return nil
	}
	t := time.Now().Add(time.Duration(seconds) * time.Second)
	return &t
}

func stringField(raw map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v, ok := raw[key]; ok {
			if value := oauthValueString(v); value != "" {
				return value
			}
		}
	}
	return ""
}

func numberField(raw map[string]interface{}, keys ...string) int64 {
	for _, key := range keys {
		if v, ok := raw[key]; ok {
			switch typed := v.(type) {
			case float64:
				return int64(typed)
			case int64:
				return typed
			case int:
				return int64(typed)
			case string:
				n, _ := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
				return n
			}
		}
	}
	return 0
}

func extractProviderIdentity(provider config.ConnectorOAuthProviderConfig, userInfo map[string]interface{}) (externalID, displayName string) {
	if len(userInfo) == 0 {
		return "", ""
	}
	externalID = stringFieldByPath(userInfo, provider.ExternalIDField)
	if externalID == "" {
		externalID = stringField(userInfo, "id", "sub", "openid", "open_id", "unionid", "union_id", "login", "email")
	}
	displayName = stringFieldByPath(userInfo, provider.DisplayNameField)
	if displayName == "" {
		displayName = stringField(userInfo, "name", "display_name", "displayName", "login", "email", "nickname")
	}
	return externalID, displayName
}

func stringFieldByPath(raw map[string]interface{}, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	current := interface{}(raw)
	for _, part := range strings.Split(path, ".") {
		obj, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		next, ok := obj[part]
		if !ok || next == nil {
			return ""
		}
		current = next
	}
	return oauthValueString(current)
}

func oauthValueString(value interface{}) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case float32:
		return strconv.FormatInt(int64(typed), 10)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case int32:
		return strconv.FormatInt(int64(typed), 10)
	case json.Number:
		return typed.String()
	case []interface{}:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if value := oauthValueString(item); value != "" {
				values = append(values, value)
			}
		}
		return strings.Join(values, " ")
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func providerUsesPKCE(provider config.ConnectorOAuthProviderConfig) bool {
	if provider.UsePKCE == nil {
		return true
	}
	return *provider.UsePKCE
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
	base.Code = override.Code
	return base
}

func builtInOAuthProviders() map[string]config.ConnectorOAuthProviderConfig {
	trueValue := true
	falseValue := false
	return map[string]config.ConnectorOAuthProviderConfig{
		"github": {
			Code:             "github",
			Name:             "GitHub",
			ClientIDEnv:      "KAGEOS_OAUTH_GITHUB_CLIENT_ID",
			ClientSecretEnv:  "KAGEOS_OAUTH_GITHUB_CLIENT_SECRET",
			AuthURL:          "https://github.com/login/oauth/authorize",
			TokenURL:         "https://github.com/login/oauth/access_token",
			UserInfoURL:      "https://api.github.com/user",
			UsePKCE:          &trueValue,
			ExternalIDField:  "id",
			DisplayNameField: "login",
		},
		"gitlab": {
			Code:             "gitlab",
			Name:             "GitLab",
			ClientIDEnv:      "KAGEOS_OAUTH_GITLAB_CLIENT_ID",
			ClientSecretEnv:  "KAGEOS_OAUTH_GITLAB_CLIENT_SECRET",
			AuthURL:          "https://gitlab.com/oauth/authorize",
			TokenURL:         "https://gitlab.com/oauth/token",
			UserInfoURL:      "https://gitlab.com/api/v4/user",
			Scopes:           []string{"read_user"},
			UsePKCE:          &trueValue,
			ExternalIDField:  "id",
			DisplayNameField: "username",
		},
		"google": {
			Code:             "google",
			Name:             "Google",
			ClientIDEnv:      "KAGEOS_OAUTH_GOOGLE_CLIENT_ID",
			ClientSecretEnv:  "KAGEOS_OAUTH_GOOGLE_CLIENT_SECRET",
			AuthURL:          "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:         "https://oauth2.googleapis.com/token",
			UserInfoURL:      "https://openidconnect.googleapis.com/v1/userinfo",
			Scopes:           []string{"openid", "email", "profile"},
			UsePKCE:          &trueValue,
			ExtraAuthParams:  map[string]string{"prompt": "consent"},
			ExternalIDField:  "sub",
			DisplayNameField: "email",
		},
		"microsoft": {
			Code:             "microsoft",
			Name:             "Microsoft",
			ClientIDEnv:      "KAGEOS_OAUTH_MICROSOFT_CLIENT_ID",
			ClientSecretEnv:  "KAGEOS_OAUTH_MICROSOFT_CLIENT_SECRET",
			AuthURL:          "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL:         "https://login.microsoftonline.com/common/oauth2/v2.0/token",
			UserInfoURL:      "https://graph.microsoft.com/v1.0/me",
			Scopes:           []string{"openid", "offline_access", "profile", "email", "User.Read"},
			UsePKCE:          &trueValue,
			ExternalIDField:  "id",
			DisplayNameField: "userPrincipalName",
		},
		"slack": {
			Code:            "slack",
			Name:            "Slack",
			ClientIDEnv:     "KAGEOS_OAUTH_SLACK_CLIENT_ID",
			ClientSecretEnv: "KAGEOS_OAUTH_SLACK_CLIENT_SECRET",
			AuthURL:         "https://slack.com/oauth/v2/authorize",
			TokenURL:        "https://slack.com/api/oauth.v2.access",
			Scopes:          []string{"identity.basic"},
			UsePKCE:         &falseValue,
		},
		"feishu": {
			Code:             "feishu",
			Name:             "飞书",
			ClientIDEnv:      "KAGEOS_OAUTH_FEISHU_CLIENT_ID",
			ClientSecretEnv:  "KAGEOS_OAUTH_FEISHU_CLIENT_SECRET",
			AuthURL:          "https://accounts.feishu.cn/open-apis/authen/v1/authorize",
			TokenURL:         "https://open.feishu.cn/open-apis/authen/v2/oauth/token",
			TokenRequestMode: tokenRequestModeJSON,
			UsePKCE:          &trueValue,
			ExternalIDField:  "open_id",
			DisplayNameField: "name",
		},
		"lark": {
			Code:             "lark",
			Name:             "Lark",
			ClientIDEnv:      "KAGEOS_OAUTH_LARK_CLIENT_ID",
			ClientSecretEnv:  "KAGEOS_OAUTH_LARK_CLIENT_SECRET",
			AuthURL:          "https://accounts.larksuite.com/open-apis/authen/v1/authorize",
			TokenURL:         "https://open.larksuite.com/open-apis/authen/v2/oauth/token",
			TokenRequestMode: tokenRequestModeJSON,
			UsePKCE:          &trueValue,
			ExternalIDField:  "open_id",
			DisplayNameField: "name",
		},
		"dingtalk": {
			Code:              "dingtalk",
			Name:              "钉钉",
			ClientIDEnv:       "KAGEOS_OAUTH_DINGTALK_CLIENT_ID",
			ClientSecretEnv:   "KAGEOS_OAUTH_DINGTALK_CLIENT_SECRET",
			AuthURL:           "https://login.dingtalk.com/oauth2/auth",
			TokenURL:          "https://api.dingtalk.com/v1.0/oauth2/userAccessToken",
			TokenRequestMode:  tokenRequestModeJSON,
			ClientIDParam:     "clientId",
			ClientSecretParam: "clientSecret",
			GrantTypeParam:    "grantType",
			RefreshTokenParam: "refreshToken",
			UsePKCE:           &falseValue,
			ExternalIDField:   "unionId",
			DisplayNameField:  "nick",
		},
	}
}

func safeRedirectAfter(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return ""
	}
	if parsed, err := url.Parse(value); err == nil && parsed.IsAbs() {
		return ""
	}
	return value
}
