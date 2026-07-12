package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/kageos/kageos/pkg/openapitoken"
	"github.com/kageos/kageos/pkg/publicshare"
)

const (
	accessTokenSubjectPrefix  = "access_"
	openAPITokenSubjectPrefix = "openapi_"
	timerVerifiedLeaderKey    = "timer.verified.leader_username"
	maxSignedTimerRequestBody = 10 << 20
)

// requireTimerAPIAuthentication is deliberately local to timer-scheduler.
// It never trusts loopback, proxy-added identity headers, or X-Request-User by
// itself: every request must carry a cryptographically valid access/OpenAPI
// token, after which identity and audit headers are rebuilt from its claims.
func requireTimerAPIAuthentication(gatewayVerifiers ...*controlauth.Verifier) gin.HandlerFunc {
	var gatewayVerifier *controlauth.Verifier
	if len(gatewayVerifiers) > 0 {
		gatewayVerifier = gatewayVerifiers[0]
	}
	return func(c *gin.Context) {
		if controlauth.HasHTTPMetadata(c.Request.Header) {
			authenticateSignedTimerBackendRequest(c, gatewayVerifier)
			return
		}
		xToken := strings.TrimSpace(c.GetHeader(contextx.TokenHeader))
		bearerToken := openapitoken.BearerToken(c.GetHeader("Authorization"))

		clearTimerRequestAuthentication(c)

		if xToken != "" && bearerToken != "" && xToken != bearerToken {
			rejectTimerAPIAuthentication(c)
			return
		}
		rawToken := xToken
		if rawToken == "" {
			rawToken = bearerToken
		}
		if rawToken == "" {
			rejectTimerAPIAuthentication(c)
			return
		}

		claims, err := auth.NewJWTService().ValidateToken(rawToken)
		if err != nil || claims == nil || strings.TrimSpace(claims.Username) == "" {
			rejectTimerAPIAuthentication(c)
			return
		}
		if claims.LeaderUsername != nil && strings.TrimSpace(*claims.LeaderUsername) != "" {
			c.Set(timerVerifiedLeaderKey, strings.TrimSpace(*claims.LeaderUsername))
		}

		switch {
		case strings.HasPrefix(claims.Subject, accessTokenSubjectPrefix):
			applyTimerJWTIdentity(c, rawToken, claims)
		case strings.HasPrefix(claims.Subject, openAPITokenSubjectPrefix):
			principal, validateErr := openapitoken.Validate(rawToken, c.ClientIP(), c.GetHeader("User-Agent"))
			if validateErr != nil || principal == nil || strings.TrimSpace(principal.Username) == "" {
				rejectTimerAPIAuthentication(c)
				return
			}
			applyTimerOpenAPIIdentity(c, rawToken, principal)
		default:
			// Refresh, password-reset, and other JWTs are not API credentials.
			rejectTimerAPIAuthentication(c)
			return
		}

		c.Next()
	}
}

func authenticateSignedTimerBackendRequest(c *gin.Context, verifier *controlauth.Verifier) {
	req := c.Request
	claimedIdentity := contextx.CaptureTrustedIdentityHeaders(req.Header)
	verificationRequest := req.Clone(req.Context())
	verificationRequest.Header = req.Header.Clone()
	clearTimerRequestAuthentication(c)

	const tasksPath = "/timer/api/v1/tasks"
	if req.URL == nil || (req.URL.Path != tasksPath && !strings.HasPrefix(req.URL.Path, tasksPath+"/")) {
		rejectTimerAPIAuthentication(c)
		return
	}
	body, err := readAndRestoreSignedTimerBody(req, maxSignedTimerRequestBody)
	if err != nil {
		rejectTimerAPIAuthentication(c)
		return
	}
	if err := controlauth.VerifyHTTPRequest(verificationRequest, body, timerHTTPProtectedHeaders(), verifier); err != nil {
		rejectTimerAPIAuthentication(c)
		return
	}
	if strings.TrimSpace(claimedIdentity[contextx.RequestUserHeader]) == "" {
		rejectTimerAPIAuthentication(c)
		return
	}
	contextx.ApplyTrustedIdentityHeaders(req.Header, claimedIdentity)
	if userID, err := strconv.ParseInt(strings.TrimSpace(claimedIdentity[contextx.UserIDHeader]), 10, 64); err == nil && userID > 0 {
		c.Set("user_id", userID)
	}
	if email := strings.TrimSpace(claimedIdentity[contextx.UserEmailHeader]); email != "" {
		c.Set("email", email)
	}
	if leader := strings.TrimSpace(claimedIdentity[contextx.LeaderUsernameHeader]); leader != "" {
		c.Set(timerVerifiedLeaderKey, leader)
	}
	for name, value := range claimedIdentity {
		if value = strings.TrimSpace(value); value != "" {
			c.Set(name, value)
		}
	}
	c.Next()
}

func clearTimerRequestAuthentication(c *gin.Context) {
	contextx.ClearTrustedIdentityHeaders(c.Request.Header)
	controlauth.ClearHTTPMetadata(c.Request.Header)
	c.Request.Header.Del(contextx.TokenHeader)
	c.Request.Header.Del("Authorization")
	c.Request.Header.Del(contextx.PubKeyHerder)
	c.Request.Header.Del(publicshare.AnonymousTokenHeader)
	for _, name := range contextx.TrustedIdentityHeaderNames() {
		c.Set(name, "")
	}
}

func timerHTTPProtectedHeaders() []string {
	names := contextx.TrustedIdentityHeaderNames()
	return append(names, contextx.TraceIdHeader)
}

func readAndRestoreSignedTimerBody(req *http.Request, maxBytes int64) ([]byte, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, nil
	}
	body, err := io.ReadAll(io.LimitReader(req.Body, maxBytes+1))
	_ = req.Body.Close()
	req.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("read signed timer request body: %w", err)
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("signed timer request body exceeds %d bytes", maxBytes)
	}
	return body, nil
}

func applyTimerJWTIdentity(c *gin.Context, rawToken string, claims *auth.JWTClaims) {
	identity := map[string]string{
		contextx.RequestUserHeader:    strings.TrimSpace(claims.Username),
		contextx.CompanyCodeHeader:    strings.TrimSpace(claims.CompanyCode),
		contextx.CompanyNameHeader:    strings.TrimSpace(claims.CompanyName),
		contextx.CompanyLogoURLHeader: strings.TrimSpace(claims.CompanyLogoURL),
		contextx.ClientSourceHeader:   contextx.ClientSourceBrowser,
	}
	if claims.DepartmentFullPath != nil {
		identity[contextx.DepartmentFullPathHeader] = strings.TrimSpace(*claims.DepartmentFullPath)
	}
	applyTimerAuthenticatedIdentity(c, rawToken, claims.UserID, claims.Email, identity, 0)
}

func applyTimerOpenAPIIdentity(c *gin.Context, rawToken string, principal *openapitoken.Principal) {
	identity := map[string]string{
		contextx.RequestUserHeader:        strings.TrimSpace(principal.Username),
		contextx.DepartmentFullPathHeader: strings.TrimSpace(principal.DepartmentFullPath),
		contextx.CompanyCodeHeader:        strings.TrimSpace(principal.CompanyCode),
		contextx.CompanyNameHeader:        strings.TrimSpace(principal.CompanyName),
		contextx.CompanyLogoURLHeader:     strings.TrimSpace(principal.CompanyLogoURL),
		contextx.ClientSourceHeader:       contextx.ClientSourceOpenAPI,
		contextx.SourceTypeHeader:         contextx.SourceTypeOpenAPIToken,
		contextx.SourceRefHeader:          strings.TrimSpace(principal.Username),
	}
	applyTimerAuthenticatedIdentity(c, rawToken, principal.UserID, principal.Email, identity, principal.TokenID)
}

func applyTimerAuthenticatedIdentity(c *gin.Context, rawToken string, userID int64, email string, identity map[string]string, openAPITokenID int64) {
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
		if value != "" {
			c.Set(name, value)
		}
	}
}

func rejectTimerAPIAuthentication(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "valid access or OpenAPI token is required"})
}
