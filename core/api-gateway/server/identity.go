package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/publicshare"
)

const (
	internalHTTPMaxAge           = 30 * time.Second
	internalHTTPMaxFutureSkew    = 10 * time.Second
	maxSignedInternalRequestBody = 10 << 20
	workspaceActionPath          = "/agent/api/v1/workspace/chat/stream"
	externalAccessTokenPrefix    = "access_"
)

var (
	errProxyTokenBlacklisted = errors.New("gateway: token is blacklisted")
	errInvalidInternalAuth   = errors.New("gateway: invalid internal request authentication")
)

type verifiedWorkspaceActionContextKey struct{}

type verifiedWorkspaceActionRequest struct {
	body []byte
}

type verifiedTimerDelegationContextKey struct{}

type verifiedTimerDelegationRequest struct {
	body []byte
}

type verifiedAgentDelegationContextKey struct{}

type verifiedAgentDelegationRequest struct {
	body []byte
}

// prepareProxyIdentity is the gateway trust-boundary operation. It never
// forwards client-provided identity metadata as-is: signed internal requests
// are verified before their signed snapshot is restored, while external
// requests are rebuilt only from credentials the gateway can verify locally.
func (s *Server) prepareProxyIdentity(c *gin.Context) error {
	req := c.Request
	claimedIdentity := contextx.CaptureTrustedIdentityHeaders(req.Header)

	if controlauth.HasHTTPMetadata(req.Header) {
		return s.verifyAndRestoreInternalIdentity(c, claimedIdentity)
	}

	contextx.ClearTrustedIdentityHeaders(req.Header)
	controlauth.ClearHTTPMetadata(req.Header)
	return s.applyVerifiedExternalIdentity(c)
}

func (s *Server) verifyAndRestoreInternalIdentity(c *gin.Context, claimedIdentity map[string]string) error {
	req := c.Request
	defer controlauth.ClearHTTPMetadata(req.Header)

	isWorkspaceAction := req.Method == http.MethodPost && req.URL.Path == workspaceActionPath
	isTimerDelegation := false
	isAgentDelegation := false
	verifier := s.workspaceActionVerifier
	switch {
	case isWorkspaceAction:
	case isAllowedAgentDelegatedAPI(req.Method, req.URL.Path):
		verifier = s.agentDelegationVerifier
		isAgentDelegation = true
	case isAllowedAgentDelegatedTimer(req.Method, req.URL.Path):
		verifier = s.agentTimerVerifier
		isTimerDelegation = true
	default:
		contextx.ClearTrustedIdentityHeaders(req.Header)
		return fmt.Errorf("%w: internal scope is not allowed for %s %s", errInvalidInternalAuth, req.Method, req.URL.Path)
	}

	body, err := readAndRestoreSignedBody(req, maxSignedInternalRequestBody)
	if err != nil {
		contextx.ClearTrustedIdentityHeaders(req.Header)
		return fmt.Errorf("%w: %v", errInvalidInternalAuth, err)
	}
	if err := controlauth.VerifyHTTPRequest(
		req,
		body,
		internalHTTPProtectedHeaders(),
		verifier,
	); err != nil {
		contextx.ClearTrustedIdentityHeaders(req.Header)
		return fmt.Errorf("%w: %v", errInvalidInternalAuth, err)
	}
	if strings.TrimSpace(claimedIdentity[contextx.RequestUserHeader]) == "" {
		contextx.ClearTrustedIdentityHeaders(req.Header)
		return fmt.Errorf("%w: signed request user is required", errInvalidInternalAuth)
	}

	// The signed identity is authoritative. Remove unrelated credentials so a
	// downstream middleware cannot choose a second, conflicting principal.
	req.Header.Del(contextx.TokenHeader)
	req.Header.Del("Authorization")
	req.Header.Del(contextx.PubKeyHerder)
	req.Header.Del(publicshare.AnonymousTokenHeader)
	contextx.ApplyTrustedIdentityHeaders(req.Header, claimedIdentity)
	setGinIdentityContext(c, claimedIdentity)
	if isWorkspaceAction {
		c.Request = req.WithContext(context.WithValue(
			req.Context(),
			verifiedWorkspaceActionContextKey{},
			verifiedWorkspaceActionRequest{body: append([]byte(nil), body...)},
		))
	} else if isTimerDelegation {
		c.Request = req.WithContext(context.WithValue(
			req.Context(),
			verifiedTimerDelegationContextKey{},
			verifiedTimerDelegationRequest{body: append([]byte(nil), body...)},
		))
	} else if isAgentDelegation {
		c.Request = req.WithContext(context.WithValue(
			req.Context(),
			verifiedAgentDelegationContextKey{},
			verifiedAgentDelegationRequest{body: append([]byte(nil), body...)},
		))
	}
	return nil
}

// signVerifiedDelegatedBackendRequest applies a fresh signature after the
// reverse proxy has finalized the backend Host and rewritten path. The typed
// context marker exists only after the gateway verified the Agent delegation
// against its exact method/path allowlist.
func (s *Server) signVerifiedDelegatedBackendRequest(req *http.Request) {
	verified, ok := req.Context().Value(verifiedAgentDelegationContextKey{}).(verifiedAgentDelegationRequest)
	if !ok {
		return
	}
	if err := controlauth.SignHTTPRequest(
		req,
		verified.body,
		internalHTTPProtectedHeaders(),
		s.delegatedBackendSigner,
	); err == nil {
		return
	} else {
		logger.Errorf(s.ctx, "[Proxy] Failed to sign verified Agent delegation for backend: %v", err)
	}
	clearFailedBackendAuthentication(req)
}

func (s *Server) signVerifiedTimerBackendRequest(req *http.Request) {
	verified, ok := req.Context().Value(verifiedTimerDelegationContextKey{}).(verifiedTimerDelegationRequest)
	if !ok {
		return
	}
	if err := controlauth.SignHTTPRequest(
		req,
		verified.body,
		internalHTTPProtectedHeaders(),
		s.timerBackendSigner,
	); err == nil {
		return
	} else {
		logger.Errorf(s.ctx, "[Proxy] Failed to sign verified Agent delegation for timer backend: %v", err)
	}
	clearFailedBackendAuthentication(req)
}

// signVerifiedAgentBackendRequest applies a fresh, scope-separated signature
// only to a workspace action that was authenticated on the gateway's inbound
// boundary. The private context value cannot be supplied over HTTP.
func (s *Server) signVerifiedAgentBackendRequest(req *http.Request) {
	verified, ok := req.Context().Value(verifiedWorkspaceActionContextKey{}).(verifiedWorkspaceActionRequest)
	if !ok {
		return
	}
	if err := controlauth.SignHTTPRequest(
		req,
		verified.body,
		internalHTTPProtectedHeaders(),
		s.agentBackendSigner,
	); err == nil {
		return
	} else {
		logger.Errorf(s.ctx, "[Proxy] Failed to sign verified workspace action for agent backend: %v", err)
	}

	// Director cannot return an error. Remove every usable identity/credential
	// and send an unsigned request; agent-server's strict boundary will reject
	// it with 401 instead of executing under a spoofable identity.
	clearFailedBackendAuthentication(req)
}

func clearFailedBackendAuthentication(req *http.Request) {
	if req == nil {
		return
	}
	contextx.ClearTrustedIdentityHeaders(req.Header)
	controlauth.ClearHTTPMetadata(req.Header)
	req.Header.Del(contextx.TokenHeader)
	req.Header.Del("Authorization")
	req.Header.Del(contextx.PubKeyHerder)
	req.Header.Del(publicshare.AnonymousTokenHeader)
}

func (s *Server) applyVerifiedExternalIdentity(c *gin.Context) error {
	req := c.Request
	token := strings.TrimSpace(req.Header.Get(contextx.TokenHeader))
	if token != "" {
		if s.tokenBlacklist != nil && s.tokenBlacklist.IsBlacklisted(token) {
			return errProxyTokenBlacklisted
		}

		claims, err := auth.NewJWTService().ValidateToken(token)
		if err == nil && strings.HasPrefix(claims.Subject, externalAccessTokenPrefix) {
			identity := identityFromJWTClaims(claims)
			contextx.ApplyTrustedIdentityHeaders(req.Header, identity)
			setGinIdentityContext(c, identity)
			logger.Infof(s.ctx, "[Proxy] Extracted verified username from token: %s, Path: %s", claims.Username, req.URL.Path)
			return nil
		}
		if err == nil {
			logger.Warnf(s.ctx, "[Proxy] Refused non-access JWT as an external API credential - Path: %s, Subject: %s",
				req.URL.Path, claims.Subject)
			return nil
		}
		// Public and optional routes may legitimately receive an expired token.
		// Forward the credential for their own policy, but never rebuild identity
		// from it at the gateway.
		logger.Warnf(s.ctx, "[Proxy] Failed to parse token - Path: %s, Error: %v, TokenLength: %d",
			req.URL.Path, err, len(token))
	}

	// Anonymous public-share identity is derived later from the share id and
	// verified session token. The gateway can safely establish provenance, but
	// must not invent X-Request-User without the share context.
	if anonymousToken := strings.TrimSpace(req.Header.Get(publicshare.AnonymousTokenHeader)); anonymousToken != "" {
		if _, err := publicshare.ValidateAnonymousToken(anonymousToken); err == nil {
			identity := map[string]string{
				contextx.ClientSourceHeader: contextx.ClientSourcePublicShare,
				contextx.SourceTypeHeader:   contextx.SourceTypePublicShare,
			}
			contextx.ApplyTrustedIdentityHeaders(req.Header, identity)
			setGinIdentityContext(c, identity)
		}
	}
	return nil
}

func identityFromJWTClaims(claims *auth.JWTClaims) map[string]string {
	identity := map[string]string{
		contextx.RequestUserHeader:    strings.TrimSpace(claims.Username),
		contextx.UserIDHeader:         strconv.FormatInt(claims.UserID, 10),
		contextx.UserEmailHeader:      strings.TrimSpace(claims.Email),
		contextx.ClientSourceHeader:   contextx.ClientSourceBrowser,
		contextx.CompanyCodeHeader:    strings.TrimSpace(claims.CompanyCode),
		contextx.CompanyNameHeader:    strings.TrimSpace(claims.CompanyName),
		contextx.CompanyLogoURLHeader: strings.TrimSpace(claims.CompanyLogoURL),
	}
	if claims.DepartmentFullPath != nil {
		identity[contextx.DepartmentFullPathHeader] = strings.TrimSpace(*claims.DepartmentFullPath)
	}
	if claims.LeaderUsername != nil {
		identity[contextx.LeaderUsernameHeader] = strings.TrimSpace(*claims.LeaderUsername)
	}
	return identity
}

func internalHTTPProtectedHeaders() []string {
	names := contextx.TrustedIdentityHeaderNames()
	names = append(names, contextx.TraceIdHeader)
	return names
}

func setGinIdentityContext(c *gin.Context, identity map[string]string) {
	for _, name := range contextx.TrustedIdentityHeaderNames() {
		if value := strings.TrimSpace(identity[name]); value != "" {
			c.Set(name, value)
		}
	}
}

func readAndRestoreSignedBody(req *http.Request, maxBytes int64) ([]byte, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, nil
	}
	limited := io.LimitReader(req.Body, maxBytes+1)
	body, err := io.ReadAll(limited)
	_ = req.Body.Close()
	req.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("read signed request body: %w", err)
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("signed request body exceeds %d bytes", maxBytes)
	}
	return body, nil
}
