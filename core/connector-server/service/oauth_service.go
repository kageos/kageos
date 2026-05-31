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
	resourcePath := defaultConnectorResourcePath(req.ResourcePath)
	if _, _, err := parseConnectorBindingScope(resourcePath); err != nil {
		return nil, err
	}
	state, err := newOAuthState()
	if err != nil {
		return nil, err
	}
	codeVerifier, codeChallenge, err := newPKCEVerifier()
	if err != nil {
		return nil, err
	}
	scopes := s.effectiveOAuthScopes(ctx, owner, provider, resourcePath, req.Scopes)
	expiresAt := time.Now().Add(s.oauthStateTTL())
	oauthState := &model.ConnectorOAuthState{
		State:         state,
		OwnerUsername: owner,
		Provider:      provider.Code,
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
		AuthorizeURL: s.oauth.BuildAuthURL(provider, callbackURL, state, codeChallenge, scopes),
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
	tokenPayload, err := s.oauth.Exchange(ctx, provider, s.oauthCallbackURL(), code, oauthState.CodeVerifier, scopes)
	if err != nil {
		return nil, oauthState.RedirectAfter, err
	}
	userInfo, err := fetchProviderUserInfo(ctx, provider, tokenPayload.AccessToken)
	if err != nil {
		return nil, oauthState.RedirectAfter, err
	}
	externalID, providerDisplayName := extractProviderIdentity(provider, userInfo)
	displayName := firstNonEmpty(oauthState.DisplayName, providerDisplayName, provider.Name, provider.Code)
	metadata := map[string]interface{}{
		"oauth":       true,
		"scopes":      scopes,
		"provider":    provider.Code,
		"external_id": externalID,
	}
	if provider.ProviderAccountURL != "" {
		metadata["provider_account_url"] = provider.ProviderAccountURL
	}
	connection, err := s.createConnectionForOwner(ctx, oauthState.OwnerUsername, dto.CreateConnectorConnectionReq{
		Provider:          provider.Code,
		DisplayName:       displayName,
		ExternalAccountID: externalID,
		Metadata:          metadata,
	})
	if err != nil {
		return nil, oauthState.RedirectAfter, err
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
	payload, err := s.oauth.Refresh(ctx, provider, refreshToken, splitScopes(tokenRow.Scopes))
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

func (s *ConnectorService) effectiveOAuthScopes(ctx context.Context, owner string, provider config.ConnectorOAuthProviderConfig, resourcePath string, requested []string) []string {
	scopes := make([]string, 0, len(provider.Scopes)+len(requested))
	scopes = append(scopes, provider.Scopes...)
	if existingScopes := s.existingOAuthScopesForResource(ctx, owner, resourcePath, provider.Code); len(existingScopes) > 0 {
		scopes = append(scopes, existingScopes...)
	}
	scopes = append(scopes, requested...)
	return cleanScopes(scopes)
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

func missingScopes(provider string, granted, required []string) []string {
	granted = cleanScopes(granted)
	required = cleanScopes(required)
	if len(required) == 0 {
		return nil
	}
	missing := make([]string, 0)
	for _, scope := range required {
		if connectorScopeGranted(provider, granted, scope) {
			continue
		}
		missing = append(missing, scope)
	}
	return missing
}

func connectorScopeGranted(provider string, granted []string, required string) bool {
	required = strings.TrimSpace(required)
	if required == "" {
		return true
	}
	grantedSet := make(map[string]struct{}, len(granted))
	for _, scope := range granted {
		scope = strings.TrimSpace(scope)
		if scope != "" {
			grantedSet[scope] = struct{}{}
		}
	}
	if _, ok := grantedSet[required]; ok {
		return true
	}
	if normalizeProvider(provider) == "github" {
		return githubScopeImplies(grantedSet, required)
	}
	return false
}

func githubScopeImplies(granted map[string]struct{}, required string) bool {
	if _, ok := granted["repo"]; ok {
		switch required {
		case "public_repo", "repo:status", "repo_deployment", "repo:invite", "security_events", "admin:repo_hook", "write:repo_hook", "read:repo_hook":
			return true
		}
	}
	if _, ok := granted["user"]; ok {
		switch required {
		case "read:user", "user:email", "user:follow":
			return true
		}
	}
	if _, ok := granted["admin:org"]; ok {
		switch required {
		case "write:org", "read:org":
			return true
		}
	}
	if _, ok := granted["write:org"]; ok && required == "read:org" {
		return true
	}
	return false
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
