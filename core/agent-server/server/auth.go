package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/kageos/kageos/pkg/openapitoken"
	"github.com/kageos/kageos/pkg/publicshare"
)

const (
	agentBackendHTTPMaxAge           = 30 * time.Second
	agentBackendHTTPMaxFutureSkew    = 10 * time.Second
	maxSignedAgentBackendRequestBody = 10 << 20
	agentWorkspaceActionPath         = "/agent/api/v1/workspace/chat/stream"
	accessTokenSubjectPrefix         = "access_"
	openAPITokenSubjectPrefix        = "openapi_"
)

// requireAgentAPIAuthentication is the Agent HTTP trust boundary. Loopback,
// host-network callers, and gateway-added identity headers are never trusted
// on their own. A request must either carry its own user credential or a fresh
// gateway->Agent signature produced for a verified workspace action.
func (s *Server) requireAgentAPIAuthentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		if controlauth.HasHTTPMetadata(c.Request.Header) {
			s.authenticateSignedAgentBackendRequest(c)
			return
		}
		s.authenticateAgentCredential(c)
	}
}

func (s *Server) authenticateSignedAgentBackendRequest(c *gin.Context) {
	req := c.Request
	claimedIdentity := contextx.CaptureTrustedIdentityHeaders(req.Header)
	verificationRequest := req.Clone(req.Context())
	verificationRequest.Header = req.Header.Clone()

	// Clear the live request before verification. Only the private clone retains
	// the claimed values needed to validate the signature; handlers can never
	// observe them unless verification succeeds and they are rebuilt below.
	clearAgentRequestAuthentication(c)

	if req.Method != http.MethodPost || req.URL.Path != agentWorkspaceActionPath {
		rejectAgentAPIAuthentication(c)
		return
	}
	body, err := readAndRestoreAgentSignedBody(req, maxSignedAgentBackendRequestBody)
	if err != nil {
		rejectAgentAPIAuthentication(c)
		return
	}
	if err := controlauth.VerifyHTTPRequest(
		verificationRequest,
		body,
		agentHTTPProtectedHeaders(),
		s.agentBackendVerifier,
	); err != nil {
		rejectAgentAPIAuthentication(c)
		return
	}
	if strings.TrimSpace(claimedIdentity[contextx.RequestUserHeader]) == "" {
		rejectAgentAPIAuthentication(c)
		return
	}

	applyAgentIdentity(c, claimedIdentity)
	s.continueAuthenticatedAgentRequest(c)
}

func (s *Server) authenticateAgentCredential(c *gin.Context) {
	xToken := strings.TrimSpace(c.GetHeader(contextx.TokenHeader))
	bearerToken := openapitoken.BearerToken(c.GetHeader("Authorization"))
	clearAgentRequestAuthentication(c)

	// Access JWTs use X-Token; long-lived OpenAPI credentials use Bearer. Do
	// not guess when both mechanisms are present, because two principals could
	// otherwise compete for the same request.
	if (xToken == "") == (bearerToken == "") {
		rejectAgentAPIAuthentication(c)
		return
	}

	if xToken != "" {
		claims, err := auth.NewJWTService().ValidateToken(xToken)
		if err != nil || claims == nil ||
			!strings.HasPrefix(claims.Subject, accessTokenSubjectPrefix) ||
			strings.TrimSpace(claims.Username) == "" {
			rejectAgentAPIAuthentication(c)
			return
		}
		applyAgentJWTIdentity(c, xToken, claims)
		s.continueAuthenticatedAgentRequest(c)
		return
	}

	claims, err := auth.NewJWTService().ValidateToken(bearerToken)
	if err != nil || claims == nil || !strings.HasPrefix(claims.Subject, openAPITokenSubjectPrefix) {
		rejectAgentAPIAuthentication(c)
		return
	}
	principal, err := openapitoken.Validate(bearerToken, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil || principal == nil || strings.TrimSpace(principal.Username) == "" {
		rejectAgentAPIAuthentication(c)
		return
	}
	applyAgentOpenAPIIdentity(c, bearerToken, principal)
	s.continueAuthenticatedAgentRequest(c)
}

func (s *Server) continueAuthenticatedAgentRequest(c *gin.Context) {
	if s.agentDelegationSigner != nil {
		ctx := controlauth.WithDelegatedHTTPRequestSigner(c.Request.Context(), s.agentDelegationSigner)
		c.Request = c.Request.WithContext(ctx)
	}
	c.Next()
}

func newAgentDelegatedHTTPRequestSigner(
	gatewayBaseURL string,
	apiSigner *controlauth.Signer,
	timerSigner *controlauth.Signer,
) (controlauth.DelegatedHTTPRequestSigner, error) {
	gatewayURL, err := url.Parse(strings.TrimSpace(gatewayBaseURL))
	if err != nil {
		return nil, fmt.Errorf("parse gateway URL: %w", err)
	}
	if gatewayURL.Scheme == "" || gatewayURL.Host == "" {
		return nil, fmt.Errorf("gateway URL must include scheme and host")
	}
	gatewayScheme := strings.ToLower(gatewayURL.Scheme)
	gatewayHost := strings.ToLower(gatewayURL.Host)

	return func(req *http.Request, body []byte) (bool, error) {
		if req == nil || req.URL == nil {
			return false, nil
		}
		// Delegation carries only the signed identity snapshot. Long-lived
		// OpenAPI tokens and end-user access credentials must not propagate to
		// downstream services or any absolute external URL.
		credential := strings.TrimSpace(req.Header.Get(contextx.TokenHeader))
		if credential == "" {
			credential = openapitoken.BearerToken(req.Header.Get("Authorization"))
		}
		applyAgentDelegatedCredentialHeaders(req.Header, credential)
		req.Header.Del(contextx.TokenHeader)
		req.Header.Del("Authorization")
		req.Header.Del(contextx.PubKeyHerder)
		req.Header.Del(publicshare.AnonymousTokenHeader)
		if strings.ToLower(req.URL.Scheme) != gatewayScheme ||
			strings.ToLower(req.URL.Host) != gatewayHost {
			contextx.ClearTrustedIdentityHeaders(req.Header)
			controlauth.ClearHTTPMetadata(req.Header)
			return false, nil
		}

		signer := apiSigner
		if req.URL.Path == "/timer/api/v1/tasks" || strings.HasPrefix(req.URL.Path, "/timer/api/v1/tasks/") {
			signer = timerSigner
		}
		if err := controlauth.SignHTTPRequest(req, body, agentHTTPProtectedHeaders(), signer); err != nil {
			contextx.ClearTrustedIdentityHeaders(req.Header)
			controlauth.ClearHTTPMetadata(req.Header)
			return false, err
		}
		return true, nil
	}, nil
}

func applyAgentDelegatedCredentialHeaders(header http.Header, rawToken string) {
	if strings.TrimSpace(rawToken) == "" {
		return
	}
	claims, err := auth.NewJWTService().ValidateToken(rawToken)
	if err != nil || claims == nil {
		return
	}
	if claims.UserID > 0 {
		header.Set(contextx.UserIDHeader, strconv.FormatInt(claims.UserID, 10))
	}
	if email := strings.TrimSpace(claims.Email); email != "" {
		header.Set(contextx.UserEmailHeader, email)
	}
	if claims.LeaderUsername != nil {
		if leader := strings.TrimSpace(*claims.LeaderUsername); leader != "" {
			header.Set(contextx.LeaderUsernameHeader, leader)
		}
	}
}

func applyAgentJWTIdentity(c *gin.Context, rawToken string, claims *auth.JWTClaims) {
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
	applyAgentCredentialIdentity(c, rawToken, claims.UserID, claims.Email, identity, 0)
}

func applyAgentOpenAPIIdentity(c *gin.Context, rawToken string, principal *openapitoken.Principal) {
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
	applyAgentCredentialIdentity(c, rawToken, principal.UserID, principal.Email, identity, principal.TokenID)
}

func applyAgentCredentialIdentity(
	c *gin.Context,
	rawToken string,
	userID int64,
	email string,
	identity map[string]string,
	openAPITokenID int64,
) {
	applyAgentIdentity(c, identity)
	c.Request.Header.Set(contextx.TokenHeader, rawToken)
	c.Set(contextx.TokenHeader, rawToken)
	c.Set("user_id", userID)
	c.Set("username", identity[contextx.RequestUserHeader])
	c.Set("email", strings.TrimSpace(email))
	if openAPITokenID > 0 {
		c.Set("openapi_token_id", openAPITokenID)
	}
}

func applyAgentIdentity(c *gin.Context, identity map[string]string) {
	contextx.ApplyTrustedIdentityHeaders(c.Request.Header, identity)
	for name, value := range identity {
		if value = strings.TrimSpace(value); value != "" {
			c.Set(name, value)
		}
	}
}

func clearAgentRequestAuthentication(c *gin.Context) {
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

func rejectAgentAPIAuthentication(c *gin.Context) {
	clearAgentRequestAuthentication(c)
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": "valid access/OpenAPI token or gateway signature is required",
	})
}

func agentHTTPProtectedHeaders() []string {
	names := contextx.TrustedIdentityHeaderNames()
	names = append(names, contextx.TraceIdHeader)
	return names
}

func readAndRestoreAgentSignedBody(req *http.Request, maxBytes int64) ([]byte, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, nil
	}
	limited := io.LimitReader(req.Body, maxBytes+1)
	body, err := io.ReadAll(limited)
	_ = req.Body.Close()
	req.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("read signed Agent request body: %w", err)
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("signed Agent request body exceeds %d bytes", maxBytes)
	}
	return body, nil
}
