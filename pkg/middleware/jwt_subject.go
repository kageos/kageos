package middleware

import (
	"strings"

	"github.com/kageos/kageos/pkg/auth"
)

const accessTokenSubjectPrefix = "access_"

// isAccessTokenClaims distinguishes user-facing access credentials from
// refresh, password-reset, and long-lived OpenAPI JWTs. ValidateToken proves
// the signature, but the token subject still determines what it may authorize.
func isAccessTokenClaims(claims *auth.JWTClaims) bool {
	return claims != nil && strings.HasPrefix(strings.TrimSpace(claims.Subject), accessTokenSubjectPrefix)
}
