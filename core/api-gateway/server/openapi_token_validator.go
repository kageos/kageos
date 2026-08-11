package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/openapitoken"
	"golang.org/x/sync/singleflight"
)

const (
	openAPITokenActiveCacheTTL   = time.Minute
	openAPITokenRejectedCacheTTL = 5 * time.Minute
	openAPITokenValidateTimeout  = 5 * time.Second
	openAPITokenCacheMaxEntries  = 10_000
)

var errOpenAPITokenAuthorityRejected = errors.New("OpenAPI Token authority rejected token")

type cachedOpenAPIPrincipal struct {
	principal  *openapitoken.Principal
	validUntil time.Time
}

type OpenAPITokenValidator struct {
	validationURL string
	httpClient    *http.Client

	mu       sync.RWMutex
	active   map[string]cachedOpenAPIPrincipal
	rejected map[string]time.Time
	group    singleflight.Group
}

func NewOpenAPITokenValidator(cfg *config.APIGatewayConfig) (*OpenAPITokenValidator, error) {
	validationURL, err := resolveOpenAPITokenValidationURL(cfg)
	if err != nil {
		return nil, err
	}
	return &OpenAPITokenValidator{
		validationURL: validationURL,
		httpClient: &http.Client{
			Timeout: openAPITokenValidateTimeout,
		},
		active:   make(map[string]cachedOpenAPIPrincipal),
		rejected: make(map[string]time.Time),
	}, nil
}

func resolveOpenAPITokenValidationURL(cfg *config.APIGatewayConfig) (string, error) {
	if cfg == nil {
		return "", errors.New("api gateway config is nil")
	}
	for _, route := range cfg.Routes {
		if route.ServiceName != "hr" && route.Path != "/hr" {
			continue
		}
		if len(route.Targets) == 0 || strings.TrimSpace(route.Targets[0].URL) == "" {
			break
		}
		return strings.TrimRight(route.Targets[0].URL, "/") + "/hr/api/v1/auth/openapi_token/validate", nil
	}
	return "", errors.New("hr route is required for OpenAPI Token validation")
}

func (v *OpenAPITokenValidator) Validate(ctx context.Context, rawToken string) (*openapitoken.Principal, error) {
	rawToken = strings.TrimSpace(rawToken)
	if v == nil || strings.TrimSpace(v.validationURL) == "" {
		return nil, errors.New("OpenAPI Token validator is not configured")
	}
	claims, err := auth.NewJWTService().ValidateOpenAPIToken(rawToken)
	if err != nil {
		return nil, err
	}

	tokenHash := openapitoken.HashToken(rawToken)
	if v.isRejected(tokenHash) {
		return nil, openapitoken.ErrTokenRevoked
	}
	if principal := v.cachedPrincipal(tokenHash); principal != nil {
		return principal, nil
	}

	value, err, _ := v.group.Do(tokenHash, func() (interface{}, error) {
		if v.isRejected(tokenHash) {
			return nil, openapitoken.ErrTokenRevoked
		}
		if principal := v.cachedPrincipal(tokenHash); principal != nil {
			return principal, nil
		}
		principal, validateErr := v.validateWithHR(ctx, rawToken)
		if validateErr != nil {
			if errors.Is(validateErr, errOpenAPITokenAuthorityRejected) {
				v.rejectUntil(tokenHash, time.Now().Add(openAPITokenRejectedCacheTTL))
			}
			return nil, validateErr
		}
		if principal.Username != strings.TrimSpace(claims.Username) ||
			(principal.UserID != 0 && principal.UserID != claims.UserID) {
			v.rejectUntil(tokenHash, time.Now().Add(openAPITokenRejectedCacheTTL))
			return nil, errors.New("OpenAPI Token principal does not match signed claims")
		}
		v.cachePrincipal(tokenHash, principal, claims.ExpiresAt)
		return principal, nil
	})
	if err != nil {
		return nil, err
	}
	principal, ok := value.(*openapitoken.Principal)
	if !ok || principal == nil {
		return nil, errors.New("OpenAPI Token validation returned no principal")
	}
	return principal, nil
}

func (v *OpenAPITokenValidator) MarkRevoked(tokenHash string, expiresAtUnix int64) {
	if v == nil || strings.TrimSpace(tokenHash) == "" {
		return
	}
	var expiresAt time.Time
	if expiresAtUnix > 0 {
		expiresAt = time.Unix(expiresAtUnix, 0)
	}
	v.mu.Lock()
	delete(v.active, tokenHash)
	v.pruneLocked(time.Now(), openAPITokenCacheMaxEntries-1)
	v.rejected[tokenHash] = expiresAt
	v.mu.Unlock()
}

func (v *OpenAPITokenValidator) cachedPrincipal(tokenHash string) *openapitoken.Principal {
	now := time.Now()
	v.mu.RLock()
	entry, ok := v.active[tokenHash]
	v.mu.RUnlock()
	if !ok {
		return nil
	}
	if !entry.validUntil.After(now) {
		v.mu.Lock()
		delete(v.active, tokenHash)
		v.mu.Unlock()
		return nil
	}
	copy := *entry.principal
	return &copy
}

func (v *OpenAPITokenValidator) cachePrincipal(tokenHash string, principal *openapitoken.Principal, tokenExpiresAt *jwt.NumericDate) {
	validUntil := time.Now().Add(openAPITokenActiveCacheTTL)
	if tokenExpiresAt != nil && tokenExpiresAt.Time.Before(validUntil) {
		validUntil = tokenExpiresAt.Time
	}
	copy := *principal
	v.mu.Lock()
	delete(v.rejected, tokenHash)
	v.pruneLocked(time.Now(), openAPITokenCacheMaxEntries-1)
	v.active[tokenHash] = cachedOpenAPIPrincipal{principal: &copy, validUntil: validUntil}
	v.mu.Unlock()
}

func (v *OpenAPITokenValidator) isRejected(tokenHash string) bool {
	now := time.Now()
	v.mu.RLock()
	until, ok := v.rejected[tokenHash]
	v.mu.RUnlock()
	if !ok {
		return false
	}
	if until.IsZero() || until.After(now) {
		return true
	}
	v.mu.Lock()
	delete(v.rejected, tokenHash)
	v.mu.Unlock()
	return false
}

func (v *OpenAPITokenValidator) rejectUntil(tokenHash string, until time.Time) {
	v.mu.Lock()
	delete(v.active, tokenHash)
	v.pruneLocked(time.Now(), openAPITokenCacheMaxEntries-1)
	v.rejected[tokenHash] = until
	v.mu.Unlock()
}

func (v *OpenAPITokenValidator) pruneLocked(now time.Time, maxEntries int) {
	if len(v.active)+len(v.rejected) <= maxEntries {
		return
	}
	for tokenHash, entry := range v.active {
		if !entry.validUntil.After(now) {
			delete(v.active, tokenHash)
		}
	}
	for tokenHash, until := range v.rejected {
		if !until.IsZero() && !until.After(now) {
			delete(v.rejected, tokenHash)
		}
	}
	for len(v.active)+len(v.rejected) > maxEntries {
		removed := false
		for tokenHash := range v.active {
			delete(v.active, tokenHash)
			removed = true
			break
		}
		if removed {
			continue
		}
		for tokenHash := range v.rejected {
			delete(v.rejected, tokenHash)
			removed = true
			break
		}
		if !removed {
			break
		}
	}
}

func (v *OpenAPITokenValidator) validateWithHR(ctx context.Context, rawToken string) (*openapitoken.Principal, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.validationURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+rawToken)
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OpenAPI Token authority unavailable: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code int                    `json:"code"`
		Msg  string                 `json:"msg"`
		Data openapitoken.Principal `json:"data"`
	}
	if err := decodeAuthorityResponse(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("decode OpenAPI Token authority response: %w", err)
	}
	if resp.StatusCode != http.StatusOK || result.Code != 0 {
		return nil, fmt.Errorf("%w: %s", errOpenAPITokenAuthorityRejected, strings.TrimSpace(result.Msg))
	}
	if strings.TrimSpace(result.Data.Username) == "" {
		return nil, errors.New("OpenAPI Token authority returned an empty principal")
	}
	return &result.Data, nil
}
