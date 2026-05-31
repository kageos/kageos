package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/kageos/kageos/core/connector-server/model"
	"github.com/kageos/kageos/core/connector-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/contextx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDirectoryBindingIsScopedToRequestUser(t *testing.T) {
	svc := newTestConnectorService(t)

	aliceCtx := contextx.WithRequestUser(context.Background(), "alice")
	conn, err := svc.CreateConnection(aliceCtx, dto.CreateConnectorConnectionReq{
		Provider:          "GitHub",
		DisplayName:       "Alice GitHub",
		ExternalAccountID: "alice",
	})
	if err != nil {
		t.Fatalf("create connection: %v", err)
	}

	if _, err := svc.BindDirectory(aliceCtx, dto.BindConnectorDirectoryReq{
		ResourcePath: "/alice/app",
		Provider:     "github",
		ConnectionID: conn.ConnectionID,
	}); err != nil {
		t.Fatalf("bind directory: %v", err)
	}

	resolved, err := svc.ResolveDirectoryBinding(aliceCtx, "/alice/app/folder/file", "github")
	if err != nil {
		t.Fatalf("resolve alice binding: %v", err)
	}
	if resolved.Connection.ConnectionID != conn.ConnectionID {
		t.Fatalf("resolved connection = %s, want %s", resolved.Connection.ConnectionID, conn.ConnectionID)
	}
	if resolved.ResolvedFrom != "/alice/app" {
		t.Fatalf("resolved from = %s, want /alice/app", resolved.ResolvedFrom)
	}

	bobCtx := contextx.WithRequestUser(context.Background(), "bob")
	if _, err := svc.ResolveDirectoryBinding(bobCtx, "/alice/app/folder/file", "github"); err == nil {
		t.Fatal("expected bob to be unable to resolve alice binding")
	}
	if _, err := svc.BindDirectory(bobCtx, dto.BindConnectorDirectoryReq{
		ResourcePath: "/bob/app",
		Provider:     "github",
		ConnectionID: conn.ConnectionID,
	}); err == nil || !strings.Contains(err.Error(), "不属于当前用户") {
		t.Fatalf("expected bob to be unable to bind alice connection, got %v", err)
	}
}

func TestGlobalDirectoryBindingResolvesEveryWorkspaceForOwner(t *testing.T) {
	svc := newTestConnectorService(t)

	aliceCtx := contextx.WithRequestUser(context.Background(), "alice")
	conn, err := svc.CreateConnection(aliceCtx, dto.CreateConnectorConnectionReq{
		Provider:    "github",
		DisplayName: "Alice GitHub",
	})
	if err != nil {
		t.Fatalf("create connection: %v", err)
	}

	binding, err := svc.BindDirectory(aliceCtx, dto.BindConnectorDirectoryReq{
		ResourcePath: "/",
		Provider:     "github",
		ConnectionID: conn.ConnectionID,
	})
	if err != nil {
		t.Fatalf("bind global directory: %v", err)
	}
	if binding.ResourcePath != "/" || binding.TenantUser != "*" || binding.App != "*" {
		t.Fatalf("global binding not normalized: %#v", binding)
	}

	resolved, err := svc.ResolveDirectoryBinding(aliceCtx, "/someone/else/folder/file", "github")
	if err != nil {
		t.Fatalf("resolve global binding: %v", err)
	}
	if resolved.ResolvedFrom != "/" || resolved.Connection.ConnectionID != conn.ConnectionID {
		t.Fatalf("resolved global binding mismatch: %#v", resolved)
	}

	bobCtx := contextx.WithRequestUser(context.Background(), "bob")
	if _, err := svc.ResolveDirectoryBinding(bobCtx, "/someone/else/folder/file", "github"); err == nil {
		t.Fatal("expected bob to be unable to resolve alice global binding")
	}
}

func TestWildcardResourcePathIsRejected(t *testing.T) {
	svc := newTestConnectorService(t)

	ctx := contextx.WithRequestUser(context.Background(), "alice")
	conn, err := svc.CreateConnection(ctx, dto.CreateConnectorConnectionReq{
		Provider:    "github",
		DisplayName: "Alice GitHub",
	})
	if err != nil {
		t.Fatalf("create connection: %v", err)
	}

	_, err = svc.BindDirectory(ctx, dto.BindConnectorDirectoryReq{
		ResourcePath: "/*",
		Provider:     "github",
		ConnectionID: conn.ConnectionID,
	})
	if err == nil || !strings.Contains(err.Error(), "不支持通配符") {
		t.Fatalf("expected wildcard bind to be rejected, got %v", err)
	}

	if _, err := svc.ResolveDirectoryBinding(ctx, "/*", "github"); err == nil || !strings.Contains(err.Error(), "不支持通配符") {
		t.Fatalf("expected wildcard resolve to be rejected, got %v", err)
	}
}

func TestDirectoryBindingCanBeReboundAfterDelete(t *testing.T) {
	svc := newTestConnectorService(t)
	ctx := contextx.WithRequestUser(context.Background(), "alice")

	conn, err := svc.CreateConnection(ctx, dto.CreateConnectorConnectionReq{Provider: "github"})
	if err != nil {
		t.Fatalf("create connection: %v", err)
	}
	req := dto.BindConnectorDirectoryReq{
		ResourcePath: "/alice/app",
		Provider:     "github",
		ConnectionID: conn.ConnectionID,
	}
	if _, err := svc.BindDirectory(ctx, req); err != nil {
		t.Fatalf("bind directory: %v", err)
	}
	if err := svc.DeleteDirectoryBinding(ctx, "/alice/app", "github"); err != nil {
		t.Fatalf("delete binding: %v", err)
	}
	if _, err := svc.BindDirectory(ctx, req); err != nil {
		t.Fatalf("rebind directory after delete: %v", err)
	}
}

func TestOAuthCallbackStoresEncryptedTokenAndBinding(t *testing.T) {
	var tokenRequests int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			tokenRequests++
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token":  "access-secret",
				"refresh_token": "refresh-secret",
				"token_type":    "Bearer",
				"expires_in":    3600,
				"scope":         "repo user:email",
			})
		case "/user":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":    "alice-ext",
				"login": "alice",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	svc := newOAuthTestConnectorService(t, config.ConnectorOAuthConfig{
		CallbackBaseURL: "http://kageos.test",
		Providers: []config.ConnectorOAuthProviderConfig{{
			Code:             "test",
			Name:             "Test Provider",
			ClientID:         "client-id",
			ClientSecret:     "client-secret",
			AuthURL:          upstream.URL + "/authorize",
			TokenURL:         upstream.URL + "/token",
			UserInfoURL:      upstream.URL + "/user",
			Scopes:           []string{"repo"},
			ExternalIDField:  "id",
			DisplayNameField: "login",
		}},
	})

	ctx := contextx.WithRequestUser(context.Background(), "alice")
	started, err := svc.StartOAuth(ctx, dto.StartConnectorOAuthReq{
		Provider:     "test",
		ResourcePath: "/alice/app",
	})
	if err != nil {
		t.Fatalf("start oauth: %v", err)
	}
	if !strings.Contains(started.AuthorizeURL, "state="+started.State) {
		t.Fatalf("authorize URL missing state: %s", started.AuthorizeURL)
	}

	completed, _, err := svc.CompleteOAuthCallback(context.Background(), "code-123", started.State, "")
	if err != nil {
		t.Fatalf("complete oauth callback: %v", err)
	}
	if tokenRequests != 1 {
		t.Fatalf("token requests = %d, want 1", tokenRequests)
	}
	if completed.Connection.ExternalAccountID != "alice-ext" {
		t.Fatalf("external account = %s, want alice-ext", completed.Connection.ExternalAccountID)
	}
	if completed.Binding == nil || completed.Binding.ResourcePath != "/alice/app" {
		t.Fatalf("expected /alice/app binding, got %#v", completed.Binding)
	}
	if !completed.Token.HasAccess || !completed.Token.HasRefresh {
		t.Fatalf("token summary missing access/refresh: %#v", completed.Token)
	}

	row, err := svc.repo.GetOwnedOAuthToken(context.Background(), "alice", completed.Connection.ConnectionID)
	if err != nil {
		t.Fatalf("get token row: %v", err)
	}
	if strings.Contains(row.AccessTokenCipher, "access-secret") || strings.Contains(row.RefreshTokenCipher, "refresh-secret") {
		t.Fatalf("token row stores plaintext token: %#v", row)
	}
	accessToken, err := svc.tokenVault.Open(row.AccessTokenCipher)
	if err != nil {
		t.Fatalf("decrypt access token: %v", err)
	}
	if accessToken != "access-secret" {
		t.Fatalf("decrypted access token = %s, want access-secret", accessToken)
	}
	if err := svc.RevokeConnection(ctx, completed.Connection.ConnectionID); err != nil {
		t.Fatalf("revoke connection: %v", err)
	}
	if _, err := svc.repo.GetOwnedOAuthToken(context.Background(), "alice", completed.Connection.ConnectionID); err == nil {
		t.Fatal("expected token row to be deleted after revoke")
	}
	if _, err := svc.ResolveDirectoryBinding(ctx, "/alice/app/folder", "test"); err == nil {
		t.Fatal("expected binding to be removed after revoke")
	}
}

func TestRefreshOAuthTokenUpdatesEncryptedAccessToken(t *testing.T) {
	var refreshSeen bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "alice-ext"})
			return
		}
		if r.URL.Path != "/token" {
			http.NotFound(w, r)
			return
		}
		_ = r.ParseForm()
		accessToken := "access-initial"
		refreshToken := "refresh-secret"
		if r.Form.Get("grant_type") == "refresh_token" {
			refreshSeen = true
			accessToken = "access-refreshed"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer upstream.Close()

	svc := newOAuthTestConnectorService(t, config.ConnectorOAuthConfig{
		CallbackBaseURL: "http://kageos.test",
		Providers: []config.ConnectorOAuthProviderConfig{{
			Code:         "test",
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			AuthURL:      upstream.URL + "/authorize",
			TokenURL:     upstream.URL + "/token",
			UserInfoURL:  upstream.URL + "/user",
		}},
	})

	ctx := contextx.WithRequestUser(context.Background(), "alice")
	started, err := svc.StartOAuth(ctx, dto.StartConnectorOAuthReq{Provider: "test"})
	if err != nil {
		t.Fatalf("start oauth: %v", err)
	}
	completed, _, err := svc.CompleteOAuthCallback(context.Background(), "code-123", started.State, "")
	if err != nil {
		t.Fatalf("complete oauth callback: %v", err)
	}
	if completed.Binding == nil || completed.Binding.ResourcePath != "/" {
		t.Fatalf("expected default global binding, got %#v", completed.Binding)
	}
	resolved, err := svc.ResolveDirectoryBinding(ctx, "/someone/else/folder", "test")
	if err != nil {
		t.Fatalf("resolve default global oauth binding: %v", err)
	}
	if resolved.ResolvedFrom != "/" || resolved.Connection.ConnectionID != completed.Connection.ConnectionID {
		t.Fatalf("default global oauth binding mismatch: %#v", resolved)
	}

	refreshed, err := svc.RefreshOAuthToken(ctx, completed.Connection.ConnectionID)
	if err != nil {
		t.Fatalf("refresh oauth token: %v", err)
	}
	if !refreshSeen {
		t.Fatal("token endpoint did not receive refresh_token grant")
	}
	row, err := svc.repo.GetOwnedOAuthToken(context.Background(), "alice", refreshed.Connection.ConnectionID)
	if err != nil {
		t.Fatalf("get token row: %v", err)
	}
	accessToken, err := svc.tokenVault.Open(row.AccessTokenCipher)
	if err != nil {
		t.Fatalf("decrypt access token: %v", err)
	}
	if accessToken != "access-refreshed" {
		t.Fatalf("decrypted access token = %s, want access-refreshed", accessToken)
	}
}

func TestOAuthProviderManagementEncryptsSecretAndOverridesRuntimeConfig(t *testing.T) {
	svc := newOAuthTestConnectorService(t, config.ConnectorOAuthConfig{
		CallbackBaseURL: "http://kageos.test",
	})
	ctx := contextx.WithRequestUser(context.Background(), "system")
	enabled := true

	info, err := svc.UpsertOAuthProvider(ctx, dto.UpsertConnectorOAuthProviderReq{
		Code:         "custom",
		Name:         "Custom OAuth",
		ClientID:     "custom-client",
		ClientSecret: "custom-secret",
		AuthURL:      "https://example.com/oauth/authorize",
		TokenURL:     "https://example.com/oauth/token",
		Scopes:       []string{"read", "write"},
		Enabled:      &enabled,
	})
	if err != nil {
		t.Fatalf("upsert oauth provider: %v", err)
	}
	if !info.Managed || !info.HasClientSecret {
		t.Fatalf("provider info should be managed with secret: %#v", info)
	}
	if info.Code != "custom" || info.ClientID != "custom-client" {
		t.Fatalf("unexpected provider info: %#v", info)
	}

	row, err := svc.repo.GetOAuthProviderSetting(context.Background(), "custom")
	if err != nil {
		t.Fatalf("get provider setting: %v", err)
	}
	if strings.Contains(row.ClientSecretCipher, "custom-secret") {
		t.Fatalf("provider secret stored in plaintext: %#v", row)
	}
	decrypted, err := svc.tokenVault.Open(row.ClientSecretCipher)
	if err != nil {
		t.Fatalf("decrypt provider secret: %v", err)
	}
	if decrypted != "custom-secret" {
		t.Fatalf("provider secret = %s, want custom-secret", decrypted)
	}

	started, err := svc.StartOAuth(ctx, dto.StartConnectorOAuthReq{Provider: "custom"})
	if err != nil {
		t.Fatalf("start custom oauth: %v", err)
	}
	if !strings.HasPrefix(started.AuthorizeURL, "https://example.com/oauth/authorize?") {
		t.Fatalf("authorize URL did not use managed provider: %s", started.AuthorizeURL)
	}
	if !strings.Contains(started.AuthorizeURL, "client_id=custom-client") {
		t.Fatalf("authorize URL should use managed client id: %s", started.AuthorizeURL)
	}

	list, err := svc.ListOAuthProviders(ctx)
	if err != nil {
		t.Fatalf("list providers: %v", err)
	}
	found := false
	for _, item := range list {
		if item.Code == "custom" {
			found = item.Managed && item.HasClientSecret
		}
	}
	if !found {
		t.Fatalf("managed provider not found in list: %#v", list)
	}

	if err := svc.DeleteOAuthProvider(ctx, "custom"); err != nil {
		t.Fatalf("delete provider: %v", err)
	}
	if _, err := svc.StartOAuth(ctx, dto.StartConnectorOAuthReq{Provider: "custom"}); err == nil {
		t.Fatal("expected deleted custom provider to be unavailable")
	}
}

func TestSeedOAuthProviderSettingsInitializesBuiltInProviders(t *testing.T) {
	for _, env := range []string{
		"KAGEOS_OAUTH_GITHUB_CLIENT_ID",
		"KAGEOS_OAUTH_GITHUB_CLIENT_SECRET",
		"KAGEOS_OAUTH_GITLAB_CLIENT_ID",
		"KAGEOS_OAUTH_GITLAB_CLIENT_SECRET",
		"KAGEOS_OAUTH_GOOGLE_CLIENT_ID",
		"KAGEOS_OAUTH_GOOGLE_CLIENT_SECRET",
		"KAGEOS_OAUTH_MICROSOFT_CLIENT_ID",
		"KAGEOS_OAUTH_MICROSOFT_CLIENT_SECRET",
		"KAGEOS_OAUTH_SLACK_CLIENT_ID",
		"KAGEOS_OAUTH_SLACK_CLIENT_SECRET",
		"KAGEOS_OAUTH_FEISHU_CLIENT_ID",
		"KAGEOS_OAUTH_FEISHU_CLIENT_SECRET",
		"KAGEOS_OAUTH_LARK_CLIENT_ID",
		"KAGEOS_OAUTH_LARK_CLIENT_SECRET",
		"KAGEOS_OAUTH_DINGTALK_CLIENT_ID",
		"KAGEOS_OAUTH_DINGTALK_CLIENT_SECRET",
	} {
		t.Setenv(env, "")
	}

	svc := newOAuthTestConnectorService(t, config.ConnectorOAuthConfig{
		CallbackBaseURL: "http://kageos.test",
	})
	ctx := contextx.WithRequestUser(context.Background(), "system")

	if err := svc.SeedOAuthProviderSettings(ctx); err != nil {
		t.Fatalf("seed oauth provider settings: %v", err)
	}
	providers, err := svc.ListOAuthProviders(ctx)
	if err != nil {
		t.Fatalf("list oauth providers: %v", err)
	}

	byCode := make(map[string]dto.ConnectorOAuthProviderInfo, len(providers))
	for _, provider := range providers {
		byCode[provider.Code] = provider
	}
	for _, code := range []string{"github", "gitlab", "google", "microsoft", "slack", "feishu", "lark", "dingtalk"} {
		provider, ok := byCode[code]
		if !ok {
			t.Fatalf("seeded provider %s not found: %#v", code, providers)
		}
		if !provider.Managed {
			t.Fatalf("provider %s should be initialized as managed: %#v", code, provider)
		}
		if !provider.Enabled {
			t.Fatalf("provider %s should be enabled by default: %#v", code, provider)
		}
		if provider.Active {
			t.Fatalf("provider %s should wait for client credentials before becoming active: %#v", code, provider)
		}
	}

	github := byCode["github"]
	if len(github.Scopes) != 0 {
		t.Fatalf("github built-in provider should not request broad scopes by default: %#v", github.Scopes)
	}
	updated, err := svc.UpsertOAuthProvider(ctx, dto.UpsertConnectorOAuthProviderReq{
		Code:         github.Code,
		ClientID:     "github-client",
		ClientSecret: "github-secret",
	})
	if err != nil {
		t.Fatalf("configure github provider credentials: %v", err)
	}
	if !updated.Active {
		t.Fatalf("configured github should become active: %#v", updated)
	}
	if updated.AuthURL != github.AuthURL || updated.TokenURL != github.TokenURL {
		t.Fatalf("configured github should preserve initialized endpoints: %#v", updated)
	}
}

func TestResolveDirectoryBindingReportsMissingScopes(t *testing.T) {
	svc := newOAuthTestConnectorService(t, config.ConnectorOAuthConfig{
		CallbackBaseURL: "http://kageos.test",
		Providers: []config.ConnectorOAuthProviderConfig{{
			Code:         "github",
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			AuthURL:      "https://github.test/login/oauth/authorize",
			TokenURL:     "https://github.test/login/oauth/access_token",
		}},
	})
	ctx := contextx.WithRequestUser(context.Background(), "alice")

	conn, err := svc.CreateConnection(ctx, dto.CreateConnectorConnectionReq{Provider: "github"})
	if err != nil {
		t.Fatalf("create connection: %v", err)
	}
	if _, err := svc.BindDirectory(ctx, dto.BindConnectorDirectoryReq{
		ResourcePath: "/alice/app",
		Provider:     "github",
		ConnectionID: conn.ConnectionID,
	}); err != nil {
		t.Fatalf("bind directory: %v", err)
	}
	if _, err := svc.storeOAuthToken(ctx, "alice", "github", conn.ConnectionID, &OAuthTokenPayload{
		AccessToken: "access-token",
		TokenType:   "Bearer",
		Scopes:      "user",
	}); err != nil {
		t.Fatalf("store token: %v", err)
	}

	resolved, err := svc.ResolveDirectoryBindingWithScopes(ctx, "/alice/app/github/orgs.table", "github", []string{"read:user", "user:email", "read:org"})
	if err != nil {
		t.Fatalf("resolve binding: %v", err)
	}
	if resolved.Connection.ConnectionID != conn.ConnectionID {
		t.Fatalf("resolved connection should be present: %#v", resolved)
	}
	if got := strings.Join(resolved.MissingScopes, ","); got != "read:org" {
		t.Fatalf("missing scopes = %q, want read:org", got)
	}
	if resolved.ScopeSatisfied {
		t.Fatalf("scope_satisfied should be false when read:org is missing: %#v", resolved)
	}
}

func TestStartOAuthMergesProviderExistingAndRequestedScopes(t *testing.T) {
	svc := newOAuthTestConnectorService(t, config.ConnectorOAuthConfig{
		CallbackBaseURL: "http://kageos.test",
		Providers: []config.ConnectorOAuthProviderConfig{{
			Code:         "test",
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			AuthURL:      "https://provider.test/oauth/authorize",
			TokenURL:     "https://provider.test/oauth/token",
			Scopes:       []string{"identity"},
		}},
	})
	ctx := contextx.WithRequestUser(context.Background(), "alice")
	conn, err := svc.CreateConnection(ctx, dto.CreateConnectorConnectionReq{Provider: "test"})
	if err != nil {
		t.Fatalf("create connection: %v", err)
	}
	if _, err := svc.BindDirectory(ctx, dto.BindConnectorDirectoryReq{
		ResourcePath: "/",
		Provider:     "test",
		ConnectionID: conn.ConnectionID,
	}); err != nil {
		t.Fatalf("bind directory: %v", err)
	}
	if _, err := svc.storeOAuthToken(ctx, "alice", "test", conn.ConnectionID, &OAuthTokenPayload{
		AccessToken: "access-token",
		TokenType:   "Bearer",
		Scopes:      "old.scope",
	}); err != nil {
		t.Fatalf("store token: %v", err)
	}

	started, err := svc.StartOAuth(ctx, dto.StartConnectorOAuthReq{
		Provider:     "test",
		ResourcePath: "/alice/app/github/orgs.table",
		Scopes:       []string{"new.scope"},
	})
	if err != nil {
		t.Fatalf("start oauth: %v", err)
	}
	authorizeURL, err := url.Parse(started.AuthorizeURL)
	if err != nil {
		t.Fatalf("parse authorize url: %v", err)
	}
	scopes := strings.Fields(authorizeURL.Query().Get("scope"))
	for _, want := range []string{"identity", "old.scope", "new.scope"} {
		if !containsString(scopes, want) {
			t.Fatalf("authorize scope %q missing %q", authorizeURL.Query().Get("scope"), want)
		}
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func newTestConnectorService(t *testing.T) *ConnectorService {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := model.InitTables(db); err != nil {
		t.Fatalf("migrate connector tables: %v", err)
	}
	return NewConnectorService(repository.NewConnectorRepository(db))
}

func newOAuthTestConnectorService(t *testing.T, oauthCfg config.ConnectorOAuthConfig) *ConnectorService {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := model.InitTables(db); err != nil {
		t.Fatalf("migrate connector tables: %v", err)
	}
	return NewConnectorService(
		repository.NewConnectorRepository(db),
		WithOAuthConfig(oauthCfg, "test-token-secret"),
	)
}
