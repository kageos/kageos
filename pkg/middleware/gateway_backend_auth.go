package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/kageos/kageos/pkg/ginx/response"
)

const (
	gatewayBackendSignatureMaxAge      = 30 * time.Second
	gatewayBackendSignatureFutureSkew  = 10 * time.Second
	maxSignedGatewayBackendRequestBody = 10 << 20
)

// GatewayOrCredentialAuth accepts exactly one verified trust source: a
// gateway backend HMAC, an access JWT, or an authoritative OpenAPI token.
// Loopback and caller-supplied identity headers never establish identity.
func GatewayOrCredentialAuth(gatewayVerifiers ...*controlauth.Verifier) gin.HandlerFunc {
	verifier := resolveGatewayBackendVerifier(gatewayVerifiers)
	return func(c *gin.Context) {
		if controlauth.HasHTTPMetadata(c.Request.Header) {
			if err := verifyGatewayBackendIdentity(c, verifier); err != nil {
				rejectGatewayOrCredential(c)
				return
			}
			c.Next()
			return
		}
		if !applyStrictExternalCredential(c) {
			rejectGatewayOrCredential(c)
			return
		}
		c.Next()
	}
}

// GatewayBackendAuth is the HMAC-only form used by internal Agent endpoints
// that intentionally do not accept direct end-user credentials.
func GatewayBackendAuth(gatewayVerifiers ...*controlauth.Verifier) gin.HandlerFunc {
	verifier := resolveGatewayBackendVerifier(gatewayVerifiers)
	return func(c *gin.Context) {
		if !controlauth.HasHTTPMetadata(c.Request.Header) {
			rejectGatewayBackend(c)
			return
		}
		if err := verifyGatewayBackendIdentity(c, verifier); err != nil {
			rejectGatewayBackend(c)
			return
		}
		c.Next()
	}
}

func resolveGatewayBackendVerifier(provided []*controlauth.Verifier) *controlauth.Verifier {
	if len(provided) > 0 {
		return provided[0]
	}
	secret, err := config.GetControlPlaneSecret()
	if err != nil {
		return nil
	}
	verifier, err := controlauth.NewVerifier(
		secret,
		controlauth.HTTPGatewayDelegatedBackendScope,
		controlauth.VerifierOptions{
			MaxAge:        gatewayBackendSignatureMaxAge,
			MaxFutureSkew: gatewayBackendSignatureFutureSkew,
		},
	)
	if err != nil {
		return nil
	}
	return verifier
}

func verifyGatewayBackendIdentity(c *gin.Context, verifier *controlauth.Verifier) error {
	if c == nil || c.Request == nil || verifier == nil {
		return controlauth.ErrInvalidConfig
	}
	req := c.Request
	claimedIdentity := contextx.CaptureTrustedIdentityHeaders(req.Header)
	body, err := readAndRestoreGatewayBackendBody(req, maxSignedGatewayBackendRequestBody)
	if err != nil {
		clearStrictCredentialIdentity(c)
		return err
	}
	if err := controlauth.VerifyHTTPRequest(req, body, gatewayBackendProtectedHeaders(), verifier); err != nil {
		clearStrictCredentialIdentity(c)
		return err
	}
	requestUser := strings.TrimSpace(claimedIdentity[contextx.RequestUserHeader])
	if requestUser == "" {
		clearStrictCredentialIdentity(c)
		return fmt.Errorf("gateway backend identity is missing request user")
	}

	clearStrictCredentialIdentity(c)
	contextx.ApplyTrustedIdentityHeaders(req.Header, claimedIdentity)
	for name, value := range claimedIdentity {
		if value = strings.TrimSpace(value); value != "" {
			c.Set(name, value)
		}
	}
	c.Set(contextx.RequestUserHeader, requestUser)
	c.Set("username", requestUser)
	c.Set("email", strings.TrimSpace(claimedIdentity[contextx.UserEmailHeader]))
	if userID, parseErr := strconv.ParseInt(strings.TrimSpace(claimedIdentity[contextx.UserIDHeader]), 10, 64); parseErr == nil && userID > 0 {
		c.Set("user_id", userID)
	}
	return nil
}

func gatewayBackendProtectedHeaders() []string {
	names := contextx.TrustedIdentityHeaderNames()
	return append(names, contextx.TraceIdHeader)
}

func readAndRestoreGatewayBackendBody(req *http.Request, maxBytes int64) ([]byte, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, nil
	}
	body, err := io.ReadAll(io.LimitReader(req.Body, maxBytes+1))
	_ = req.Body.Close()
	req.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("read signed gateway request body: %w", err)
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("signed gateway request body exceeds %d bytes", maxBytes)
	}
	return body, nil
}

func rejectGatewayOrCredential(c *gin.Context) {
	clearStrictCredentialIdentity(c)
	response.NoAuth(c, "valid access/OpenAPI credential or gateway signature is required")
	c.Abort()
}

func rejectGatewayBackend(c *gin.Context) {
	clearStrictCredentialIdentity(c)
	response.NoAuth(c, "valid gateway signature is required")
	c.Abort()
}
