package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/contextx"
	"golang.org/x/sync/singleflight"
)

const (
	accessTokenActiveCacheTTL   = time.Minute
	accessTokenRejectedCacheTTL = 30 * time.Second
	accessTokenValidateTimeout  = 5 * time.Second
	accessTokenCacheMaxEntries  = 20_000
)

var errAccessTokenAuthorityRejected = errors.New("access token authority rejected token")

type cachedAccessPrincipal struct {
	principal  *auth.AccessTokenPrincipal
	validUntil time.Time
}

// AccessTokenValidator keeps a short-lived gateway cache while HR user_session
// remains the persistent source of truth.
type AccessTokenValidator struct {
	validationURL string
	httpClient    *http.Client

	mu       sync.RWMutex
	active   map[string]cachedAccessPrincipal
	rejected map[string]time.Time
	group    singleflight.Group
}

func NewAccessTokenValidator(cfg *config.APIGatewayConfig) (*AccessTokenValidator, error) {
	validationURL, err := resolveAccessTokenValidationURL(cfg)
	if err != nil {
		return nil, err
	}
	return &AccessTokenValidator{
		validationURL: validationURL,
		httpClient:    &http.Client{Timeout: accessTokenValidateTimeout},
		active:        make(map[string]cachedAccessPrincipal),
		rejected:      make(map[string]time.Time),
	}, nil
}

func resolveAccessTokenValidationURL(cfg *config.APIGatewayConfig) (string, error) {
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
		return strings.TrimRight(route.Targets[0].URL, "/") + "/hr/api/v1/auth/access_token/validate", nil
	}
	return "", errors.New("hr route is required for access token validation")
}

func (v *AccessTokenValidator) Validate(ctx context.Context, rawToken string) (*auth.AccessTokenPrincipal, error) {
	rawToken = strings.TrimSpace(rawToken)
	if v == nil || strings.TrimSpace(v.validationURL) == "" {
		return nil, errors.New("access token validator is not configured")
	}
	claims, err := auth.NewJWTService().ValidateAccessToken(rawToken)
	if err != nil {
		return nil, err
	}
	return v.validateAccessWithClaims(ctx, rawToken, claims)
}

// ValidateGatewayToken routes signed gateway tokens by purpose. Normal access
// tokens keep the HR session check; short-lived scheduled tokens are validated
// locally and never become login sessions.
func (v *AccessTokenValidator) ValidateGatewayToken(ctx context.Context, rawToken string) (*auth.AccessTokenPrincipal, error) {
	rawToken = strings.TrimSpace(rawToken)
	if v == nil {
		return nil, errors.New("access token validator is not configured")
	}
	claims, err := auth.NewJWTService().ValidateGatewayToken(rawToken)
	if err != nil {
		return nil, err
	}
	if claims.TokenUse == auth.TokenUseScheduled {
		principal := &auth.AccessTokenPrincipal{
			UserID:   claims.UserID,
			Username: strings.TrimSpace(claims.Username),
			Email:    strings.TrimSpace(claims.Email),
		}
		if claims.DepartmentFullPath != nil {
			principal.DepartmentFullPath = strings.TrimSpace(*claims.DepartmentFullPath)
		}
		return principal, nil
	}
	if strings.TrimSpace(v.validationURL) == "" {
		return nil, errors.New("access token validator is not configured")
	}
	return v.validateAccessWithClaims(ctx, rawToken, claims)
}

func (v *AccessTokenValidator) validateAccessWithClaims(ctx context.Context, rawToken string, claims *auth.JWTClaims) (*auth.AccessTokenPrincipal, error) {

	tokenHash := hashAccessToken(rawToken)
	if v.isRejected(tokenHash) {
		return nil, errAccessTokenAuthorityRejected
	}
	if principal := v.cachedPrincipal(tokenHash); principal != nil {
		return principal, nil
	}

	value, err, _ := v.group.Do(tokenHash, func() (interface{}, error) {
		if v.isRejected(tokenHash) {
			return nil, errAccessTokenAuthorityRejected
		}
		if principal := v.cachedPrincipal(tokenHash); principal != nil {
			return principal, nil
		}
		principal, validateErr := v.validateWithHR(ctx, rawToken)
		if validateErr != nil {
			if errors.Is(validateErr, errAccessTokenAuthorityRejected) {
				v.rejectUntil(tokenHash, time.Now().Add(accessTokenRejectedCacheTTL))
			}
			return nil, validateErr
		}
		if principal.UserID != claims.UserID || principal.Username != strings.TrimSpace(claims.Username) {
			v.rejectUntil(tokenHash, time.Now().Add(accessTokenRejectedCacheTTL))
			return nil, errors.New("access token principal does not match signed claims")
		}
		v.cachePrincipal(tokenHash, principal, claims.ExpiresAt)
		return principal, nil
	})
	if err != nil {
		return nil, err
	}
	principal, ok := value.(*auth.AccessTokenPrincipal)
	if !ok || principal == nil {
		return nil, errors.New("access token validation returned no principal")
	}
	return principal, nil
}

func (v *AccessTokenValidator) MarkRevoked(tokenHash string, expiresAtUnix int64) {
	if v == nil || strings.TrimSpace(tokenHash) == "" {
		return
	}
	until := time.Now().Add(accessTokenRejectedCacheTTL)
	if expiresAtUnix > 0 {
		until = time.Unix(expiresAtUnix, 0)
	}
	v.mu.Lock()
	delete(v.active, tokenHash)
	v.pruneLocked(time.Now(), accessTokenCacheMaxEntries-1)
	v.rejected[tokenHash] = until
	v.mu.Unlock()
}

func (v *AccessTokenValidator) cachedPrincipal(tokenHash string) *auth.AccessTokenPrincipal {
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

func (v *AccessTokenValidator) cachePrincipal(tokenHash string, principal *auth.AccessTokenPrincipal, tokenExpiresAt *jwt.NumericDate) {
	validUntil := time.Now().Add(accessTokenActiveCacheTTL)
	if tokenExpiresAt != nil && tokenExpiresAt.Time.Before(validUntil) {
		validUntil = tokenExpiresAt.Time
	}
	copy := *principal
	v.mu.Lock()
	delete(v.rejected, tokenHash)
	v.pruneLocked(time.Now(), accessTokenCacheMaxEntries-1)
	v.active[tokenHash] = cachedAccessPrincipal{principal: &copy, validUntil: validUntil}
	v.mu.Unlock()
}

func (v *AccessTokenValidator) isRejected(tokenHash string) bool {
	now := time.Now()
	v.mu.RLock()
	until, ok := v.rejected[tokenHash]
	v.mu.RUnlock()
	if !ok {
		return false
	}
	if until.After(now) {
		return true
	}
	v.mu.Lock()
	delete(v.rejected, tokenHash)
	v.mu.Unlock()
	return false
}

func (v *AccessTokenValidator) rejectUntil(tokenHash string, until time.Time) {
	v.mu.Lock()
	delete(v.active, tokenHash)
	v.pruneLocked(time.Now(), accessTokenCacheMaxEntries-1)
	v.rejected[tokenHash] = until
	v.mu.Unlock()
}

func (v *AccessTokenValidator) pruneLocked(now time.Time, maxEntries int) {
	if len(v.active)+len(v.rejected) <= maxEntries {
		return
	}
	for tokenHash, entry := range v.active {
		if !entry.validUntil.After(now) {
			delete(v.active, tokenHash)
		}
	}
	for tokenHash, until := range v.rejected {
		if !until.After(now) {
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

func (v *AccessTokenValidator) validateWithHR(ctx context.Context, rawToken string) (*auth.AccessTokenPrincipal, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.validationURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set(contextx.TokenHeader, rawToken)
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("access token authority unavailable: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code int                       `json:"code"`
		Msg  string                    `json:"msg"`
		Data auth.AccessTokenPrincipal `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode access token authority response: %w", err)
	}
	if resp.StatusCode != http.StatusOK || result.Code != 0 {
		return nil, fmt.Errorf("%w: %s", errAccessTokenAuthorityRejected, strings.TrimSpace(result.Msg))
	}
	if result.Data.UserID <= 0 || strings.TrimSpace(result.Data.Username) == "" {
		return nil, errors.New("access token authority returned an empty principal")
	}
	return &result.Data, nil
}

func hashAccessToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
