package middleware

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/openapitoken"
	"github.com/kageos/kageos/pkg/publicshare"
)

// StrictCredentialAuth never trusts loopback or identity headers. It accepts
// exactly one real credential: an access_* JWT in X-Token or an openapi_* JWT
// in Authorization Bearer, then rebuilds all trusted identity metadata.
func StrictCredentialAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Any control metadata is an attempted internal authentication. This
		// middleware does not accept that scope, and must not erase a partial or
		// invalid signature before falling back to a user credential.
		if controlauth.HasHTTPMetadata(c.Request.Header) || !applyStrictExternalCredential(c) {
			rejectStrictCredential(c)
			return
		}
		c.Next()
	}
}

func applyStrictExternalCredential(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	xToken := strings.TrimSpace(c.GetHeader(contextx.TokenHeader))
	bearerToken := openapitoken.BearerToken(c.GetHeader("Authorization"))
	clearStrictCredentialIdentity(c)

	if (xToken == "") == (bearerToken == "") {
		return false
	}
	if xToken != "" {
		claims, err := auth.NewJWTService().ValidateToken(xToken)
		if err != nil || !isAccessTokenClaims(claims) || strings.TrimSpace(claims.Username) == "" {
			return false
		}
		identity := map[string]string{
			contextx.RequestUserHeader:    strings.TrimSpace(claims.Username),
			contextx.UserIDHeader:         strconv.FormatInt(claims.UserID, 10),
			contextx.UserEmailHeader:      strings.TrimSpace(claims.Email),
			contextx.CompanyCodeHeader:    strings.TrimSpace(claims.CompanyCode),
			contextx.CompanyNameHeader:    strings.TrimSpace(claims.CompanyName),
			contextx.CompanyLogoURLHeader: strings.TrimSpace(claims.CompanyLogoURL),
			contextx.ClientSourceHeader:   contextx.ClientSourceBrowser,
		}
		if claims.DepartmentFullPath != nil {
			identity[contextx.DepartmentFullPathHeader] = strings.TrimSpace(*claims.DepartmentFullPath)
		}
		if claims.LeaderUsername != nil {
			identity[contextx.LeaderUsernameHeader] = strings.TrimSpace(*claims.LeaderUsername)
		}
		applyStrictCredentialIdentity(c, xToken, claims.UserID, claims.Email, identity, 0)
		return true
	}

	principal, err := openapitoken.Validate(bearerToken, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil || principal == nil || strings.TrimSpace(principal.Username) == "" {
		return false
	}
	identity := map[string]string{
		contextx.RequestUserHeader:        strings.TrimSpace(principal.Username),
		contextx.UserIDHeader:             strconv.FormatInt(principal.UserID, 10),
		contextx.UserEmailHeader:          strings.TrimSpace(principal.Email),
		contextx.DepartmentFullPathHeader: strings.TrimSpace(principal.DepartmentFullPath),
		contextx.CompanyCodeHeader:        strings.TrimSpace(principal.CompanyCode),
		contextx.CompanyNameHeader:        strings.TrimSpace(principal.CompanyName),
		contextx.CompanyLogoURLHeader:     strings.TrimSpace(principal.CompanyLogoURL),
		contextx.ClientSourceHeader:       contextx.ClientSourceOpenAPI,
		contextx.SourceTypeHeader:         contextx.SourceTypeOpenAPIToken,
		contextx.SourceRefHeader:          strings.TrimSpace(principal.Username),
	}
	applyStrictCredentialIdentity(c, bearerToken, principal.UserID, principal.Email, identity, principal.TokenID)
	return true
}

func clearStrictCredentialIdentity(c *gin.Context) {
	contextx.ClearTrustedIdentityHeaders(c.Request.Header)
	controlauth.ClearHTTPMetadata(c.Request.Header)
	c.Request.Header.Del(contextx.TokenHeader)
	c.Request.Header.Del("Authorization")
	c.Request.Header.Del(contextx.PubKeyHerder)
	c.Request.Header.Del(publicshare.AnonymousTokenHeader)
	for _, name := range contextx.TrustedIdentityHeaderNames() {
		c.Set(name, "")
	}
	c.Set(contextx.TokenHeader, "")
	c.Set("user_id", int64(0))
	c.Set("username", "")
	c.Set("email", "")
	c.Set("openapi_token_id", int64(0))
}

func applyStrictCredentialIdentity(
	c *gin.Context,
	rawToken string,
	userID int64,
	email string,
	identity map[string]string,
	openAPITokenID int64,
) {
	contextx.ApplyTrustedIdentityHeaders(c.Request.Header, identity)
	c.Request.Header.Set(contextx.TokenHeader, rawToken)
	c.Set(contextx.TokenHeader, rawToken)
	c.Set("user_id", userID)
	c.Set("username", identity[contextx.RequestUserHeader])
	c.Set("email", strings.TrimSpace(email))
	if openAPITokenID > 0 {
		c.Set("openapi_token_id", openAPITokenID)
	}
	for name, value := range identity {
		if value = strings.TrimSpace(value); value != "" {
			c.Set(name, value)
		}
	}
}

func rejectStrictCredential(c *gin.Context) {
	clearStrictCredentialIdentity(c)
	response.NoAuth(c, "valid access or OpenAPI token is required")
	c.Abort()
}
