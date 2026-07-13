package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/config"
	"golang.org/x/oauth2"
)

func (r *OAuthProviderRegistry) Exchange(ctx context.Context, provider config.ConnectorOAuthProviderConfig, redirectURL, code, codeVerifier string, scopes []string) (*OAuthTokenPayload, error) {
	return connectorAdapterFor(provider.Code).ExchangeToken(ctx, provider, redirectURL, code, codeVerifier, scopes)
}

func exchangeOAuthToken(ctx context.Context, provider config.ConnectorOAuthProviderConfig, redirectURL, code, codeVerifier string, scopes []string) (*OAuthTokenPayload, error) {
	switch strings.ToLower(strings.TrimSpace(provider.TokenRequestMode)) {
	case tokenRequestModeJSON:
		return exchangeJSONToken(ctx, provider, redirectURL, code, codeVerifier, scopes)
	case tokenRequestModeJSONBasic:
		return exchangeJSONBasicToken(ctx, provider, redirectURL, code, codeVerifier, scopes)
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
	return connectorAdapterFor(provider.Code).RefreshToken(ctx, provider, refreshToken, scopes)
}

func refreshOAuthToken(ctx context.Context, provider config.ConnectorOAuthProviderConfig, refreshToken string, scopes []string) (*OAuthTokenPayload, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, fmt.Errorf("refresh_token 为空，无法刷新")
	}
	switch strings.ToLower(strings.TrimSpace(provider.TokenRequestMode)) {
	case tokenRequestModeJSON:
		return refreshJSONToken(ctx, provider, refreshToken, scopes)
	case tokenRequestModeJSONBasic:
		return refreshJSONBasicToken(ctx, provider, refreshToken, scopes)
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

func exchangeJSONToken(ctx context.Context, provider config.ConnectorOAuthProviderConfig, redirectURL, code, codeVerifier string, scopes []string) (*OAuthTokenPayload, error) {
	body := jsonTokenBody(provider, "authorization_code", true)
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
	body := jsonTokenBody(provider, "refresh_token", true)
	body[paramName(provider.RefreshTokenParam, "refresh_token")] = refreshToken
	for key, value := range provider.ExtraTokenParams {
		if strings.TrimSpace(key) != "" {
			body[key] = value
		}
	}
	return postJSONToken(ctx, provider.TokenURL, body, scopes)
}

func exchangeJSONBasicToken(ctx context.Context, provider config.ConnectorOAuthProviderConfig, redirectURL, code, codeVerifier string, scopes []string) (*OAuthTokenPayload, error) {
	body := jsonTokenBody(provider, "authorization_code", false)
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
	return postJSONBasicToken(ctx, provider, provider.TokenURL, body, scopes)
}

func refreshJSONBasicToken(ctx context.Context, provider config.ConnectorOAuthProviderConfig, refreshToken string, scopes []string) (*OAuthTokenPayload, error) {
	body := jsonTokenBody(provider, "refresh_token", false)
	body[paramName(provider.RefreshTokenParam, "refresh_token")] = refreshToken
	for key, value := range provider.ExtraTokenParams {
		if strings.TrimSpace(key) != "" {
			body[key] = value
		}
	}
	return postJSONBasicToken(ctx, provider, provider.TokenURL, body, scopes)
}

func jsonTokenBody(provider config.ConnectorOAuthProviderConfig, grantType string, includeCredentials bool) map[string]string {
	body := map[string]string{
		paramName(provider.GrantTypeParam, "grant_type"): grantType,
	}
	if includeCredentials {
		body[paramName(provider.ClientIDParam, "client_id")] = provider.ClientID
		body[paramName(provider.ClientSecretParam, "client_secret")] = provider.ClientSecret
	}
	return body
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
		return nil, oauthTokenEndpointError(resp.StatusCode, data)
	}
	return parseTokenPayload(data, scopes)
}

func postJSONBasicToken(ctx context.Context, provider config.ConnectorOAuthProviderConfig, tokenURL string, body map[string]string, scopes []string) (*OAuthTokenPayload, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	credentials := base64.StdEncoding.EncodeToString([]byte(provider.ClientID + ":" + provider.ClientSecret))
	req.Header.Set("Authorization", "Basic "+credentials)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	decorateProviderAPIRequest(provider.Code, req)
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
		return nil, oauthTokenEndpointError(resp.StatusCode, data)
	}
	return parseTokenPayload(data, scopes)
}

func oauthTokenEndpointError(statusCode int, body []byte) error {
	body = []byte(strings.TrimSpace(string(body)))
	if len(body) == 0 {
		return fmt.Errorf("oauth token endpoint returned %d", statusCode)
	}
	return fmt.Errorf("oauth token endpoint returned %d (response body hidden, %d bytes)", statusCode, len(body))
}
