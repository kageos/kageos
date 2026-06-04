package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/kageos/kageos/core/connector-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/contextx"
	"gorm.io/gorm"
)

const defaultOAuthStateTTL = 10 * time.Minute

func WithOAuthConfig(oauthCfg config.ConnectorOAuthConfig, encryptionSecret string) ConnectorServiceOption {
	return func(s *ConnectorService) {
		s.oauth = NewOAuthProviderRegistry(oauthCfg)
		s.callbackBase = strings.TrimSpace(oauthCfg.CallbackBaseURL)
		if oauthCfg.StateTTLSeconds > 0 {
			s.stateTTL = time.Duration(oauthCfg.StateTTLSeconds) * time.Second
		}
		s.admins = adminSet(oauthCfg.ProviderAdmins)
		vault, err := NewTokenVault(firstNonEmpty(oauthCfg.TokenEncryptionSecret, encryptionSecret))
		if err != nil {
			s.oauthInitErr = err
			return
		}
		s.tokenVault = vault
	}
}

func adminSet(users []string) map[string]struct{} {
	if len(users) == 0 {
		users = []string{"system"}
	}
	out := make(map[string]struct{}, len(users))
	for _, user := range users {
		user = strings.TrimSpace(user)
		if user != "" {
			out[user] = struct{}{}
		}
	}
	if len(out) == 0 {
		out["system"] = struct{}{}
	}
	return out
}

func (s *ConnectorService) StartOAuth(ctx context.Context, req dto.StartConnectorOAuthReq) (*dto.StartConnectorOAuthResp, error) {
	if err := s.ensureOAuthReady(); err != nil {
		return nil, err
	}
	owner := contextx.GetRequestUser(ctx)
	if owner == "" {
		return nil, fmt.Errorf("未提供用户信息")
	}
	providerCode := normalizeProvider(req.Provider)
	provider, err := s.resolveOAuthProvider(ctx, providerCode)
	if err != nil {
		return nil, err
	}
	if hasConnectorWildcardResourcePath(req.ResourcePath) {
		return nil, fmt.Errorf("resource_path 不支持通配符，请使用 / 表示全局连接器")
	}
	resourcePath := connectorGlobalResourcePath
	if _, _, err := parseConnectorBindingScope(resourcePath); err != nil {
		return nil, err
	}
	connectionID := strings.TrimSpace(req.ConnectionID)
	if connectionID != "" {
		conn, err := s.repo.GetOwnedConnection(ctx, owner, connectionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("连接器不存在或不属于当前用户")
			}
			return nil, err
		}
		if normalizeProvider(conn.Provider) != provider.Code {
			return nil, fmt.Errorf("连接器 provider 不匹配: got %s, want %s", conn.Provider, provider.Code)
		}
	}
	state, err := newOAuthState()
	if err != nil {
		return nil, err
	}
	codeVerifier, codeChallenge, err := newPKCEVerifier()
	if err != nil {
		return nil, err
	}
	adapter := connectorAdapterFor(provider.Code)
	scopes := s.effectiveOAuthScopes(ctx, owner, provider, resourcePath, req.Scopes, connectionID)
	expiresAt := time.Now().Add(s.oauthStateTTL())
	oauthState := &model.ConnectorOAuthState{
		State:         state,
		OwnerUsername: owner,
		Provider:      provider.Code,
		ConnectionID:  connectionID,
		ResourcePath:  resourcePath,
		Scopes:        strings.Join(scopes, " "),
		DisplayName:   strings.TrimSpace(req.DisplayName),
		RedirectAfter: safeRedirectAfter(req.RedirectAfter),
		CodeVerifier:  codeVerifier,
		Status:        model.OAuthStateStatusPending,
		ExpiresAt:     expiresAt,
	}
	oauthState.CreatedBy = owner
	oauthState.UpdatedBy = owner
	if err := s.repo.CreateOAuthState(ctx, oauthState); err != nil {
		return nil, err
	}
	callbackURL := s.oauthCallbackURL()
	return &dto.StartConnectorOAuthResp{
		Provider:     provider.Code,
		AuthorizeURL: adapter.BuildAuthorizeURL(provider, callbackURL, state, codeChallenge, scopes),
		State:        state,
		ExpiresAt:    expiresAt.Format(time.RFC3339),
		CallbackURL:  callbackURL,
	}, nil
}

func (s *ConnectorService) CompleteOAuthCallback(ctx context.Context, code, state, providerError string) (*dto.ConnectorOAuthCallbackResp, string, error) {
	if err := s.ensureOAuthReady(); err != nil {
		return nil, "", err
	}
	state = strings.TrimSpace(state)
	if state == "" {
		return nil, "", fmt.Errorf("state 不能为空")
	}
	oauthState, err := s.repo.GetPendingOAuthState(ctx, state)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", fmt.Errorf("OAuth state 不存在、已使用或已过期")
		}
		return nil, "", err
	}
	if providerError != "" {
		return nil, oauthState.RedirectAfter, fmt.Errorf("OAuth provider 返回错误: %s", providerError)
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, oauthState.RedirectAfter, fmt.Errorf("code 不能为空")
	}
	if err := s.repo.MarkOAuthStateUsed(ctx, state); err != nil {
		return nil, oauthState.RedirectAfter, err
	}
	provider, err := s.resolveOAuthProvider(ctx, oauthState.Provider)
	if err != nil {
		return nil, oauthState.RedirectAfter, err
	}
	scopes := splitScopes(oauthState.Scopes)
	adapter := connectorAdapterFor(provider.Code)
	tokenPayload, err := adapter.ExchangeToken(ctx, provider, s.oauthCallbackURL(), code, oauthState.CodeVerifier, scopes)
	if err != nil {
		return nil, oauthState.RedirectAfter, err
	}
	profile, err := buildOAuthConnectionProfile(ctx, provider, tokenPayload)
	if err != nil {
		return nil, oauthState.RedirectAfter, err
	}
	displayName := firstNonEmpty(oauthState.DisplayName, profile.DisplayName, provider.Name, provider.Code)
	var connection *dto.ConnectorConnectionInfo
	if strings.TrimSpace(oauthState.ConnectionID) != "" {
		existing, err := s.repo.GetOwnedConnection(ctx, oauthState.OwnerUsername, oauthState.ConnectionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, oauthState.RedirectAfter, fmt.Errorf("重新授权的连接器不存在或不属于当前用户")
			}
			return nil, oauthState.RedirectAfter, err
		}
		if normalizeProvider(existing.Provider) != provider.Code {
			return nil, oauthState.RedirectAfter, fmt.Errorf("重新授权的连接器 provider 不匹配: got %s, want %s", existing.Provider, provider.Code)
		}
		displayName = firstNonEmpty(oauthState.DisplayName, profile.DisplayName, existing.DisplayName, provider.Name, provider.Code)
		connection, err = s.updateConnectionProfileForOwner(ctx, oauthState.OwnerUsername, existing.ConnectionID, provider.Code, displayName, profile.ExternalAccountID, profile.Metadata)
		if err != nil {
			return nil, oauthState.RedirectAfter, err
		}
	} else {
		connection, err = s.createConnectionForOwner(ctx, oauthState.OwnerUsername, dto.CreateConnectorConnectionReq{
			Provider:          provider.Code,
			AuthType:          model.ConnectorAuthTypeOAuth2User,
			DisplayName:       displayName,
			ExternalAccountID: profile.ExternalAccountID,
			Metadata:          profile.Metadata,
		})
		if err != nil {
			return nil, oauthState.RedirectAfter, err
		}
	}
	tokenInfo, err := s.storeOAuthToken(ctx, oauthState.OwnerUsername, provider.Code, connection.ConnectionID, tokenPayload)
	if err != nil {
		return nil, oauthState.RedirectAfter, err
	}
	var binding *dto.ConnectorDirectoryBindingInfo
	bound, err := s.BindDirectory(contextx.WithRequestUser(ctx, oauthState.OwnerUsername), dto.BindConnectorDirectoryReq{
		ResourcePath: oauthState.ResourcePath,
		Provider:     provider.Code,
		ConnectionID: connection.ConnectionID,
	})
	if err != nil {
		return nil, oauthState.RedirectAfter, err
	}
	binding = bound
	return &dto.ConnectorOAuthCallbackResp{
		Connection: *connection,
		Token:      *tokenInfo,
		Binding:    binding,
	}, oauthState.RedirectAfter, nil
}

func (s *ConnectorService) RefreshOAuthToken(ctx context.Context, connectionID string) (*dto.RefreshConnectorOAuthTokenResp, error) {
	if err := s.ensureOAuthReady(); err != nil {
		return nil, err
	}
	owner := contextx.GetRequestUser(ctx)
	if owner == "" {
		return nil, fmt.Errorf("未提供用户信息")
	}
	conn, err := s.repo.GetOwnedConnection(ctx, owner, connectionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("连接器不存在或不属于当前用户")
		}
		return nil, err
	}
	tokenRow, err := s.repo.GetOwnedOAuthToken(ctx, owner, conn.ConnectionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("连接器没有 OAuth token")
		}
		return nil, err
	}
	refreshToken, err := s.tokenVault.Open(tokenRow.RefreshTokenCipher)
	if err != nil {
		return nil, err
	}
	if refreshToken == "" {
		return nil, fmt.Errorf("连接器没有 refresh_token，需重新授权")
	}
	provider, err := s.resolveOAuthProvider(ctx, conn.Provider)
	if err != nil {
		return nil, err
	}
	payload, err := connectorAdapterFor(provider.Code).RefreshToken(ctx, provider, refreshToken, splitScopes(tokenRow.Scopes))
	if err != nil {
		return nil, err
	}
	if payload.RefreshToken == "" {
		payload.RefreshToken = refreshToken
	}
	tokenInfo, err := s.storeOAuthToken(ctx, owner, conn.Provider, conn.ConnectionID, payload)
	if err != nil {
		return nil, err
	}
	return &dto.RefreshConnectorOAuthTokenResp{
		Connection: *connectionToInfo(conn),
		Token:      *tokenInfo,
	}, nil
}

func (s *ConnectorService) ensureOAuthReady() error {
	if s.oauthInitErr != nil {
		return s.oauthInitErr
	}
	if s.oauth == nil || s.tokenVault == nil {
		return fmt.Errorf("connector OAuth 未初始化")
	}
	return nil
}

func (s *ConnectorService) storeOAuthToken(ctx context.Context, owner, provider, connectionID string, payload *OAuthTokenPayload) (*dto.ConnectorTokenInfo, error) {
	if payload == nil || strings.TrimSpace(payload.AccessToken) == "" {
		return nil, fmt.Errorf("OAuth token 为空")
	}
	accessCipher, err := s.tokenVault.Seal(payload.AccessToken)
	if err != nil {
		return nil, err
	}
	refreshCipher, err := s.tokenVault.Seal(payload.RefreshToken)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	row := &model.ConnectorOAuthToken{
		ConnectionID:       connectionID,
		OwnerUsername:      owner,
		Provider:           provider,
		AccessTokenCipher:  accessCipher,
		RefreshTokenCipher: refreshCipher,
		TokenType:          payload.TokenType,
		Scopes:             payload.Scopes,
		Expiry:             payload.Expiry,
		LastRefreshAt:      &now,
		RawResponse:        payload.RawResponse,
	}
	row.CreatedBy = owner
	row.UpdatedBy = owner
	if err := s.repo.UpsertOAuthToken(ctx, row); err != nil {
		return nil, err
	}
	return tokenToInfo(row), nil
}

func (s *ConnectorService) oauthCallbackURL() string {
	base := strings.TrimRight(s.callbackBase, "/")
	if base == "" {
		base = strings.TrimRight(config.GetGlobalSharedConfig().Gateway.GetBaseURL(), "/")
	}
	return base + "/connector/oauth/callback"
}

func (s *ConnectorService) oauthStateTTL() time.Duration {
	if s.stateTTL > 0 {
		return s.stateTTL
	}
	return defaultOAuthStateTTL
}

func tokenToInfo(token *model.ConnectorOAuthToken) *dto.ConnectorTokenInfo {
	if token == nil {
		return nil
	}
	info := &dto.ConnectorTokenInfo{
		ConnectionID: token.ConnectionID,
		Provider:     token.Provider,
		TokenType:    token.TokenType,
		Scopes:       token.Scopes,
		HasAccess:    token.AccessTokenCipher != "",
		HasRefresh:   token.RefreshTokenCipher != "",
	}
	if token.Expiry != nil {
		info.Expiry = token.Expiry.Format(time.RFC3339)
	}
	if token.LastRefreshAt != nil {
		info.LastRefreshAt = token.LastRefreshAt.Format(time.RFC3339)
	}
	return info
}

func effectiveScopes(requested, defaults []string) []string {
	if len(requested) > 0 {
		return cleanScopes(requested)
	}
	return cleanScopes(defaults)
}

func (s *ConnectorService) effectiveOAuthScopes(ctx context.Context, owner string, provider config.ConnectorOAuthProviderConfig, resourcePath string, requested []string, connectionID string) []string {
	var existingScopes []string
	if strings.TrimSpace(connectionID) != "" {
		existingScopes = s.existingOAuthScopesForConnection(ctx, owner, connectionID)
	}
	if len(existingScopes) == 0 {
		existingScopes = s.existingOAuthScopesForResource(ctx, owner, resourcePath, provider.Code)
	}
	return connectorAdapterFor(provider.Code).MergeReconnectScopes(provider, existingScopes, requested)
}

func (s *ConnectorService) existingOAuthScopesForConnection(ctx context.Context, owner, connectionID string) []string {
	if owner == "" || strings.TrimSpace(connectionID) == "" {
		return nil
	}
	tokenRow, err := s.repo.GetOwnedOAuthToken(ctx, owner, connectionID)
	if err != nil {
		return nil
	}
	return splitScopes(tokenRow.Scopes)
}

func (s *ConnectorService) existingOAuthScopesForResource(ctx context.Context, owner, resourcePath, provider string) []string {
	provider = normalizeProvider(provider)
	if owner == "" || provider == "" {
		return nil
	}
	resolveCtx := contextx.WithRequestUser(ctx, owner)
	resolved, err := s.ResolveDirectoryBinding(resolveCtx, resourcePath, provider)
	if err != nil || resolved == nil {
		return nil
	}
	tokenRow, err := s.repo.GetOwnedOAuthToken(ctx, owner, resolved.Connection.ConnectionID)
	if err != nil {
		return nil
	}
	return splitScopes(tokenRow.Scopes)
}

func cleanScopes(scopes []string) []string {
	out := make([]string, 0, len(scopes))
	seen := map[string]struct{}{}
	for _, scope := range scopes {
		for _, part := range strings.Fields(strings.ReplaceAll(scope, ",", " ")) {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if _, ok := seen[part]; ok {
				continue
			}
			seen[part] = struct{}{}
			out = append(out, part)
		}
	}
	return out
}

func splitScopes(scopeText string) []string {
	return cleanScopes([]string{scopeText})
}

func appendOAuthCallbackQuery(redirectAfter, status, connectionID, message string) string {
	if redirectAfter == "" {
		return ""
	}
	u, err := url.Parse(redirectAfter)
	if err != nil {
		return ""
	}
	q := u.Query()
	q.Set("connector_status", status)
	if connectionID != "" {
		q.Set("connection_id", connectionID)
	}
	if message != "" {
		q.Set("message", message)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func OAuthCallbackRedirect(redirectAfter, status, connectionID, message string) string {
	return appendOAuthCallbackQuery(redirectAfter, status, connectionID, message)
}
