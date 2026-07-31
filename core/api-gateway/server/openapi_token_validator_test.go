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
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/openapitoken"
)

func TestOpenAPITokenValidatorCachesAuthorityResultAndRevocation(t *testing.T) {
	var authorityCalls atomic.Int64
	authority := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorityCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0,
			"msg":  "成功",
			"data": openapitoken.Principal{
				TokenID:  7,
				UserID:   42,
				Username: "alice",
			},
		})
	}))
	defer authority.Close()

	validator := &OpenAPITokenValidator{
		validationURL: authority.URL,
		httpClient:    authority.Client(),
		active:        make(map[string]cachedOpenAPIPrincipal),
		rejected:      make(map[string]time.Time),
	}
	token, err := auth.NewJWTService().GenerateOpenAPITokenWithContext(auth.UserTokenContext{
		UserID:   42,
		Username: "alice",
	}, nil)
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

	validator.MarkRevoked(openapitoken.HashToken(token), 0)
	if _, err := validator.Validate(context.Background(), token); err == nil {
		t.Fatal("revoked cached token should be rejected")
	}
	if got := authorityCalls.Load(); got != 1 {
		t.Fatalf("revoked token should not call authority again, calls = %d", got)
	}
}

func TestResolveOpenAPITokenValidationURL(t *testing.T) {
	got, err := resolveOpenAPITokenValidationURL(&config.APIGatewayConfig{
		Routes: []config.RouteConfig{{
			Path:        "/hr",
			ServiceName: "hr",
			Targets:     []config.BackendConfig{{URL: "http://hr:9097/"}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "http://hr:9097/hr/api/v1/auth/openapi_token/validate"
	if got != want {
		t.Fatalf("url = %q, want %q", got, want)
	}
}

func TestStripUntrustedIdentityHeaders(t *testing.T) {
	header := make(http.Header)
	header.Set(contextx.RequestUserHeader, "admin")
	header.Set("X-Username", "admin")
	header.Set(contextx.DepartmentFullPathHeader, "/org/root")
	header.Set(contextx.TokenHeader, "web-token")

	stripUntrustedIdentityHeaders(header)

	for _, key := range []string{
		contextx.RequestUserHeader,
		"X-Username",
		contextx.DepartmentFullPathHeader,
	} {
		if value := header.Get(key); value != "" {
			t.Fatalf("%s was not stripped: %q", key, value)
		}
	}
	if header.Get(contextx.TokenHeader) != "web-token" {
		t.Fatal("credential header should remain available for authentication")
	}
}

func TestOpenAPITokenValidatorPrunesExpiredAndBoundsCache(t *testing.T) {
	now := time.Now()
	validator := &OpenAPITokenValidator{
		active:   make(map[string]cachedOpenAPIPrincipal),
		rejected: make(map[string]time.Time),
	}
	validator.active["expired"] = cachedOpenAPIPrincipal{validUntil: now.Add(-time.Second)}
	validator.rejected["rejected-expired"] = now.Add(-time.Second)
	validator.rejected["permanent"] = time.Time{}
	for i := 0; i < openAPITokenCacheMaxEntries; i++ {
		validator.rejected[fmt.Sprintf("rejected-%d", i)] = now.Add(time.Hour)
	}

	validator.mu.Lock()
	validator.pruneLocked(now, openAPITokenCacheMaxEntries-1)
	validator.active["new"] = cachedOpenAPIPrincipal{validUntil: now.Add(time.Minute)}
	total := len(validator.active) + len(validator.rejected)
	_, activeExpired := validator.active["expired"]
	_, rejectedExpired := validator.rejected["rejected-expired"]
	validator.mu.Unlock()

	if total > openAPITokenCacheMaxEntries {
		t.Fatalf("cache entries = %d, want <= %d", total, openAPITokenCacheMaxEntries)
	}
	if activeExpired || rejectedExpired {
		t.Fatal("expired cache entries should be pruned")
	}
}
