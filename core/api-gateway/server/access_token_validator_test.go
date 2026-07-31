package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/config"
)

func TestAccessTokenValidatorCachesAuthorityResultAndRevocation(t *testing.T) {
	var authorityCalls atomic.Int64
	authority := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorityCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0,
			"msg":  "成功",
			"data": auth.AccessTokenPrincipal{
				UserID:   42,
				Username: "alice",
			},
		})
	}))
	defer authority.Close()

	validator := &AccessTokenValidator{
		validationURL: authority.URL,
		httpClient:    authority.Client(),
		active:        make(map[string]cachedAccessPrincipal),
		rejected:      make(map[string]time.Time),
	}
	token, err := auth.NewJWTService().GenerateAccessToken(42, "alice", "")
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		principal, err := validator.Validate(context.Background(), token)
		if err != nil {
			t.Fatal(err)
		}
		if principal.Username != "alice" {
			t.Fatalf("principal = %#v", principal)
		}
	}
	if got := authorityCalls.Load(); got != 1 {
		t.Fatalf("authority calls = %d, want 1", got)
	}

	validator.MarkRevoked(hashAccessToken(token), time.Now().Add(time.Hour).Unix())
	if _, err := validator.Validate(context.Background(), token); err == nil {
		t.Fatal("revoked cached token should be rejected")
	}
	if got := authorityCalls.Load(); got != 1 {
		t.Fatalf("revoked token should not call authority again, calls = %d", got)
	}
}

func TestResolveAccessTokenValidationURL(t *testing.T) {
	got, err := resolveAccessTokenValidationURL(&config.APIGatewayConfig{
		Routes: []config.RouteConfig{{
			Path:        "/hr",
			ServiceName: "hr",
			Targets:     []config.BackendConfig{{URL: "http://hr:9097/"}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "http://hr:9097/hr/api/v1/auth/access_token/validate"
	if got != want {
		t.Fatalf("url = %q, want %q", got, want)
	}
}

func TestAccessTokenValidatorRejectsRefreshTokenBeforeAuthorityCall(t *testing.T) {
	var authorityCalls atomic.Int64
	authority := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorityCalls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer authority.Close()
	validator := &AccessTokenValidator{
		validationURL: authority.URL,
		httpClient:    authority.Client(),
		active:        make(map[string]cachedAccessPrincipal),
		rejected:      make(map[string]time.Time),
	}
	token, err := auth.NewJWTService().GenerateRefreshToken(42, "alice", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validator.Validate(context.Background(), token); err == nil {
		t.Fatal("refresh token must not validate as access token")
	}
	if authorityCalls.Load() != 0 {
		t.Fatal("wrong-purpose token must be rejected before authority call")
	}
}

func TestAccessTokenValidatorPrunesExpiredAndBoundsCache(t *testing.T) {
	now := time.Now()
	validator := &AccessTokenValidator{
		active:   make(map[string]cachedAccessPrincipal),
		rejected: make(map[string]time.Time),
	}
	validator.active["expired"] = cachedAccessPrincipal{validUntil: now.Add(-time.Second)}
	validator.rejected["rejected-expired"] = now.Add(-time.Second)
	for i := 0; i < accessTokenCacheMaxEntries; i++ {
		validator.rejected[fmt.Sprintf("rejected-%d", i)] = now.Add(time.Hour)
	}

	validator.mu.Lock()
	validator.pruneLocked(now, accessTokenCacheMaxEntries-1)
	validator.active["new"] = cachedAccessPrincipal{validUntil: now.Add(time.Minute)}
	total := len(validator.active) + len(validator.rejected)
	_, activeExpired := validator.active["expired"]
	_, rejectedExpired := validator.rejected["rejected-expired"]
	validator.mu.Unlock()

	if total > accessTokenCacheMaxEntries {
		t.Fatalf("cache entries = %d, want <= %d", total, accessTokenCacheMaxEntries)
	}
	if activeExpired || rejectedExpired {
		t.Fatal("expired cache entries should be pruned")
	}
}
