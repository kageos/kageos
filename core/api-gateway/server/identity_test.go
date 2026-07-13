package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/kageos/kageos/pkg/publicshare"
)

const gatewayIdentityTestSecret = "0123456789abcdef0123456789abcdef"

func TestPrepareProxyIdentityClearsUnsignedForgedHeaders(t *testing.T) {
	s := newIdentityTestServer(t)
	c := newIdentityTestContext(http.MethodGet, "/workspace/api/v1/apps", nil)
	setAllForgedIdentityHeaders(c.Request.Header)
	c.Request.Header.Set(contextx.TokenHeader, "invalid-token-must-be-forwarded-but-not-trusted")

	if err := s.prepareProxyIdentity(c); err != nil {
		t.Fatal(err)
	}
	assertTrustedIdentityHeadersEmpty(t, c.Request.Header)
	if got := c.Request.Header.Get(contextx.TokenHeader); got != "invalid-token-must-be-forwarded-but-not-trusted" {
		t.Fatalf("credential header = %q", got)
	}
}

func TestPrepareProxyIdentityOverwritesForgedHeadersFromJWT(t *testing.T) {
	restoreJWTConfig := useIdentityTestJWTConfig(t)
	defer restoreJWTConfig()

	token, err := auth.NewJWTService().GenerateAccessTokenWithContext(auth.UserTokenContext{
		UserID:             7,
		Username:           "alice",
		Email:              "alice@example.com",
		DepartmentFullPath: "/org/engineering",
		CompanyCode:        "acme",
		CompanyName:        "Acme",
		CompanyLogoURL:     "https://assets.example.com/acme.png",
	})
	if err != nil {
		t.Fatal(err)
	}

	s := newIdentityTestServer(t)
	c := newIdentityTestContext(http.MethodGet, "/workspace/api/v1/apps", nil)
	setAllForgedIdentityHeaders(c.Request.Header)
	c.Request.Header.Set(contextx.TokenHeader, token)

	if err := s.prepareProxyIdentity(c); err != nil {
		t.Fatal(err)
	}
	assertHeader(t, c.Request.Header, contextx.RequestUserHeader, "alice")
	assertHeader(t, c.Request.Header, contextx.DepartmentFullPathHeader, "/org/engineering")
	assertHeader(t, c.Request.Header, contextx.CompanyCodeHeader, "acme")
	assertHeader(t, c.Request.Header, contextx.CompanyNameHeader, "Acme")
	assertHeader(t, c.Request.Header, contextx.ClientSourceHeader, contextx.ClientSourceBrowser)
	for _, name := range []string{
		contextx.UsernameHeader,
		contextx.SourceTypeHeader,
		contextx.SourceRefHeader,
		contextx.WorkspaceRoleHeader,
		contextx.InitiatorUserHeader,
		contextx.ToolCallIDHeader,
	} {
		if got := c.Request.Header.Get(name); got != "" {
			t.Fatalf("unverified header %s survived JWT rebuild: %q", name, got)
		}
	}
}

func TestPrepareProxyIdentityDoesNotTrustRefreshToken(t *testing.T) {
	restoreJWTConfig := useIdentityTestJWTConfig(t)
	defer restoreJWTConfig()

	token, err := auth.NewJWTService().GenerateRefreshTokenWithContext(auth.UserTokenContext{
		UserID:             7,
		Username:           "alice",
		Email:              "alice@example.com",
		DepartmentFullPath: "/org/engineering",
		CompanyCode:        "acme",
	})
	if err != nil {
		t.Fatal(err)
	}

	s := newIdentityTestServer(t)
	c := newIdentityTestContext(http.MethodGet, "/workspace/api/v1/apps", nil)
	setAllForgedIdentityHeaders(c.Request.Header)
	c.Request.Header.Set(contextx.TokenHeader, token)

	if err := s.prepareProxyIdentity(c); err != nil {
		t.Fatal(err)
	}
	assertTrustedIdentityHeadersEmpty(t, c.Request.Header)
	if got := c.Request.Header.Get(contextx.TokenHeader); got != token {
		t.Fatalf("refresh credential was not preserved for downstream policy: %q", got)
	}
}

func TestPrepareProxyIdentityRebuildsOnlyAnonymousProvenance(t *testing.T) {
	restoreJWTConfig := useIdentityTestJWTConfig(t)
	defer restoreJWTConfig()
	anonymousToken, _, err := publicshare.IssueAnonymousToken()
	if err != nil {
		t.Fatal(err)
	}

	s := newIdentityTestServer(t)
	c := newIdentityTestContext(http.MethodPost, "/public/api/s/share-1/submit", []byte(`{"value":"ok"}`))
	setAllForgedIdentityHeaders(c.Request.Header)
	c.Request.Header.Set(publicshare.AnonymousTokenHeader, anonymousToken)

	if err := s.prepareProxyIdentity(c); err != nil {
		t.Fatal(err)
	}
	if got := c.Request.Header.Get(contextx.RequestUserHeader); got != "" {
		t.Fatalf("gateway invented anonymous request user %q", got)
	}
	assertHeader(t, c.Request.Header, contextx.ClientSourceHeader, contextx.ClientSourcePublicShare)
	assertHeader(t, c.Request.Header, contextx.SourceTypeHeader, contextx.SourceTypePublicShare)
	if got := c.Request.Header.Get(contextx.SourceRefHeader); got != "" {
		t.Fatalf("unverified anonymous source ref survived: %q", got)
	}
	if got := c.Request.Header.Get(contextx.WorkspaceRoleHeader); got != "" {
		t.Fatalf("unverified anonymous role survived: %q", got)
	}
}

func TestPrepareProxyIdentityRestoresVerifiedInternalIdentityAndBody(t *testing.T) {
	body := []byte(`{"full_code_path":"/alice/ops","message":{"content":"run"}}`)
	signer := newIdentityTestSigner(t)
	s := newIdentityTestServer(t)
	c := newIdentityTestContext(http.MethodPost, workspaceActionPath, body)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set(contextx.RequestUserHeader, "bob")
	c.Request.Header.Set(contextx.ClientSourceHeader, "mobile_action")
	c.Request.Header.Set(contextx.SourceTypeHeader, "message_action")
	c.Request.Header.Set(contextx.WorkspaceRoleHeader, "app_developer")
	c.Request.Header.Set(contextx.TraceIdHeader, "trace-internal-1")
	if err := controlauth.SignHTTPRequest(c.Request, body, internalHTTPProtectedHeaders(), signer); err != nil {
		t.Fatal(err)
	}

	if err := s.prepareProxyIdentity(c); err != nil {
		t.Fatal(err)
	}
	assertHeader(t, c.Request.Header, contextx.RequestUserHeader, "bob")
	assertHeader(t, c.Request.Header, contextx.ClientSourceHeader, "mobile_action")
	assertHeader(t, c.Request.Header, contextx.SourceTypeHeader, "message_action")
	assertHeader(t, c.Request.Header, contextx.WorkspaceRoleHeader, "app_developer")
	if controlauth.HasHTTPMetadata(c.Request.Header) {
		t.Fatal("internal authentication metadata was forwarded")
	}
	restoredBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(restoredBody, body) {
		t.Fatalf("restored body = %q, want %q", restoredBody, body)
	}
}

func TestPrepareProxyIdentityRejectsTamperedOrReplayedInternalIdentity(t *testing.T) {
	body := []byte(`{"message":"run"}`)
	signer := newIdentityTestSigner(t)
	s := newIdentityTestServer(t)

	signed := newIdentityTestContext(http.MethodPost, workspaceActionPath, body)
	signed.Request.Header.Set(contextx.RequestUserHeader, "bob")
	signed.Request.Header.Set(contextx.SourceTypeHeader, "message_action")
	if err := controlauth.SignHTTPRequest(signed.Request, body, internalHTTPProtectedHeaders(), signer); err != nil {
		t.Fatal(err)
	}
	signedHeaders := signed.Request.Header.Clone()

	tampered := newIdentityTestContext(http.MethodPost, workspaceActionPath, body)
	tampered.Request.Header = signedHeaders.Clone()
	tampered.Request.Header.Set(contextx.RequestUserHeader, "mallory")
	if err := s.prepareProxyIdentity(tampered); !errors.Is(err, errInvalidInternalAuth) {
		t.Fatalf("tampered identity error = %v", err)
	}
	assertTrustedIdentityHeadersEmpty(t, tampered.Request.Header)

	first := newIdentityTestContext(http.MethodPost, workspaceActionPath, body)
	first.Request.Header = signedHeaders.Clone()
	if err := s.prepareProxyIdentity(first); err != nil {
		t.Fatalf("first signed request: %v", err)
	}
	second := newIdentityTestContext(http.MethodPost, workspaceActionPath, body)
	second.Request.Header = signedHeaders.Clone()
	if err := s.prepareProxyIdentity(second); !errors.Is(err, errInvalidInternalAuth) {
		t.Fatalf("replayed identity error = %v", err)
	}
	assertTrustedIdentityHeadersEmpty(t, second.Request.Header)
}

func TestCreateProxyDoesNotForwardUnsignedForgedIdentity(t *testing.T) {
	received := make(chan http.Header, 1)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Header.Clone()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer backend.Close()

	s := newIdentityTestServer(t)
	s.cfg = &config.APIGatewayConfig{Timeouts: config.GatewayTimeoutConfig{Default: 5}}
	s.sharedTransport = http.DefaultTransport.(*http.Transport).Clone()
	defer s.sharedTransport.CloseIdleConnections()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	proxy := s.createProxy(backend.URL, 5, &config.RouteConfig{Path: "/workspace"})
	engine.Any("/workspace/*path", proxy)

	req := httptest.NewRequest(http.MethodGet, "/workspace/demo", nil)
	setAllForgedIdentityHeaders(req.Header)
	res := httptest.NewRecorder()
	engine.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("proxy status = %d, body=%s", res.Code, res.Body.String())
	}
	select {
	case header := <-received:
		assertTrustedIdentityHeadersEmpty(t, header)
	case <-time.After(time.Second):
		t.Fatal("backend did not receive proxied request")
	}
}

func TestCreateProxyResignsVerifiedWorkspaceActionAfterBackendRewrite(t *testing.T) {
	type receipt struct {
		header http.Header
		path   string
		body   []byte
		err    error
	}
	received := make(chan receipt, 1)
	backendVerifier, err := controlauth.NewVerifier(
		gatewayIdentityTestSecret,
		controlauth.HTTPGatewayAgentBackendScope,
		controlauth.VerifierOptions{MaxAge: time.Minute, MaxFutureSkew: time.Second},
	)
	if err != nil {
		t.Fatal(err)
	}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, readErr := io.ReadAll(r.Body)
		verifyErr := readErr
		if verifyErr == nil {
			verifyErr = controlauth.VerifyHTTPRequest(r, body, internalHTTPProtectedHeaders(), backendVerifier)
		}
		received <- receipt{header: r.Header.Clone(), path: r.URL.Path, body: body, err: verifyErr}
		if verifyErr != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer backend.Close()

	s := newIdentityTestServer(t)
	s.cfg = &config.APIGatewayConfig{Timeouts: config.GatewayTimeoutConfig{Default: 5}}
	s.sharedTransport = http.DefaultTransport.(*http.Transport).Clone()
	defer s.sharedTransport.CloseIdleConnections()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	route := &config.RouteConfig{
		Path:        "/agent",
		RewritePath: "/backend-agent",
		ServiceName: "agent",
	}
	engine.Any("/agent/*path", s.createProxy(backend.URL, 5, route))

	body := []byte(`{"full_code_path":"/alice/ops","message":{"content":"run"}}`)
	req := newSignedGatewayWorkspaceActionRequest(t, body)
	inboundSignature := req.Header.Get(controlauth.HTTPSignatureHeader)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("resigned proxy status = %d, want 204; body=%s", recorder.Code, recorder.Body.String())
	}

	select {
	case got := <-received:
		if got.err != nil {
			t.Fatalf("verify gateway->Agent signature: %v", got.err)
		}
		if got.path != "/backend-agent/api/v1/workspace/chat/stream" {
			t.Fatalf("rewritten backend path = %q", got.path)
		}
		if !bytes.Equal(got.body, body) {
			t.Fatalf("backend body = %q, want %q", got.body, body)
		}
		assertHeader(t, got.header, contextx.RequestUserHeader, "bob")
		if got.header.Get(controlauth.HTTPSignatureHeader) == inboundSignature {
			t.Fatal("gateway forwarded the first-hop signature instead of minting a scope-separated signature")
		}
	case <-time.After(time.Second):
		t.Fatal("Agent backend did not receive resigned request")
	}
}

func TestCreateProxyResignsVerifiedAgentDelegationAfterBackendRewrite(t *testing.T) {
	type receipt struct {
		header http.Header
		path   string
		body   []byte
		err    error
	}
	received := make(chan receipt, 1)
	backendVerifier, err := controlauth.NewVerifier(
		gatewayIdentityTestSecret,
		controlauth.HTTPGatewayDelegatedBackendScope,
		controlauth.VerifierOptions{MaxAge: time.Minute, MaxFutureSkew: time.Second},
	)
	if err != nil {
		t.Fatal(err)
	}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, readErr := io.ReadAll(r.Body)
		verifyErr := readErr
		if verifyErr == nil {
			verifyErr = controlauth.VerifyHTTPRequest(r, body, internalHTTPProtectedHeaders(), backendVerifier)
		}
		received <- receipt{header: r.Header.Clone(), path: r.URL.Path, body: body, err: verifyErr}
		if verifyErr != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer backend.Close()

	s := newIdentityTestServer(t)
	s.cfg = &config.APIGatewayConfig{Timeouts: config.GatewayTimeoutConfig{Default: 5}}
	s.sharedTransport = http.DefaultTransport.(*http.Transport).Clone()
	defer s.sharedTransport.CloseIdleConnections()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	route := &config.RouteConfig{
		Path:        "/workspace",
		RewritePath: "/backend-workspace",
		ServiceName: "workspace",
	}
	engine.Any("/workspace/*path", s.createProxy(backend.URL, 5, route))

	body := []byte(`{"full_code_path":"/alice/ops","content":"package main"}`)
	req := newSignedGatewayAgentDelegationRequest(t, http.MethodPost, "/workspace/api/v1/workspace/files/write", body)
	inboundSignature := req.Header.Get(controlauth.HTTPSignatureHeader)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("resigned proxy status = %d, want 204; body=%s", recorder.Code, recorder.Body.String())
	}

	select {
	case got := <-received:
		if got.err != nil {
			t.Fatalf("verify gateway->backend signature: %v", got.err)
		}
		if got.path != "/backend-workspace/api/v1/workspace/files/write" {
			t.Fatalf("rewritten backend path = %q", got.path)
		}
		if !bytes.Equal(got.body, body) {
			t.Fatalf("backend body = %q, want %q", got.body, body)
		}
		assertHeader(t, got.header, contextx.RequestUserHeader, "bob")
		if got.header.Get(controlauth.HTTPSignatureHeader) == inboundSignature {
			t.Fatal("gateway forwarded the Agent first-hop signature instead of minting a backend signature")
		}
	case <-time.After(time.Second):
		t.Fatal("workspace backend did not receive resigned request")
	}
}

func TestCreateProxySigningFailureClearsIdentityAndLetsAgentReject(t *testing.T) {
	type receipt struct {
		header http.Header
	}
	received := make(chan receipt, 1)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- receipt{header: r.Header.Clone()}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer backend.Close()

	s := newIdentityTestServer(t)
	s.agentBackendSigner = nil
	s.cfg = &config.APIGatewayConfig{Timeouts: config.GatewayTimeoutConfig{Default: 5}}
	s.sharedTransport = http.DefaultTransport.(*http.Transport).Clone()
	defer s.sharedTransport.CloseIdleConnections()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	route := &config.RouteConfig{Path: "/agent", ServiceName: "agent"}
	engine.Any("/agent/*path", s.createProxy(backend.URL, 5, route))

	req := newSignedGatewayWorkspaceActionRequest(t, []byte(`{"message":"run"}`))
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("signing failure proxy status = %d, want Agent 401", recorder.Code)
	}

	select {
	case got := <-received:
		assertTrustedIdentityHeadersEmpty(t, got.header)
		if controlauth.HasHTTPMetadata(got.header) {
			t.Fatal("failed backend signing forwarded authentication metadata")
		}
		if got.header.Get(contextx.TokenHeader) != "" || got.header.Get("Authorization") != "" {
			t.Fatal("failed backend signing forwarded a usable credential")
		}
	case <-time.After(time.Second):
		t.Fatal("Agent backend did not receive fail-closed request")
	}
}

func newIdentityTestServer(t *testing.T) *Server {
	t.Helper()
	verifier, err := controlauth.NewVerifier(
		gatewayIdentityTestSecret,
		controlauth.HTTPWorkspaceActionScope,
		controlauth.VerifierOptions{MaxAge: time.Minute, MaxFutureSkew: time.Second},
	)
	if err != nil {
		t.Fatal(err)
	}
	agentBackendSigner, err := controlauth.NewSigner(
		gatewayIdentityTestSecret,
		controlauth.HTTPGatewayAgentBackendScope,
	)
	if err != nil {
		t.Fatal(err)
	}
	agentDelegationVerifier, err := controlauth.NewVerifier(
		gatewayIdentityTestSecret,
		controlauth.HTTPAgentDelegatedAPIScope,
		controlauth.VerifierOptions{MaxAge: time.Minute, MaxFutureSkew: time.Second},
	)
	if err != nil {
		t.Fatal(err)
	}
	delegatedBackendSigner, err := controlauth.NewSigner(
		gatewayIdentityTestSecret,
		controlauth.HTTPGatewayDelegatedBackendScope,
	)
	if err != nil {
		t.Fatal(err)
	}
	agentTimerVerifier, err := controlauth.NewVerifier(
		gatewayIdentityTestSecret,
		controlauth.HTTPAgentDelegatedTimerScope,
		controlauth.VerifierOptions{MaxAge: time.Minute, MaxFutureSkew: time.Second},
	)
	if err != nil {
		t.Fatal(err)
	}
	timerBackendSigner, err := controlauth.NewSigner(
		gatewayIdentityTestSecret,
		controlauth.HTTPGatewayTimerBackendScope,
	)
	if err != nil {
		t.Fatal(err)
	}
	return &Server{
		ctx:                     context.Background(),
		tokenBlacklist:          &TokenBlacklist{blacklist: make(map[string]int64)},
		workspaceActionVerifier: verifier,
		agentBackendSigner:      agentBackendSigner,
		agentDelegationVerifier: agentDelegationVerifier,
		delegatedBackendSigner:  delegatedBackendSigner,
		agentTimerVerifier:      agentTimerVerifier,
		timerBackendSigner:      timerBackendSigner,
	}
}

func newIdentityTestSigner(t *testing.T) *controlauth.Signer {
	t.Helper()
	signer, err := controlauth.NewSigner(gatewayIdentityTestSecret, controlauth.HTTPWorkspaceActionScope)
	if err != nil {
		t.Fatal(err)
	}
	return signer
}

func newIdentityTestContext(method, path string, body []byte) *gin.Context {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, "http://gateway.internal"+path, bytes.NewReader(body))
	return c
}

func newSignedGatewayWorkspaceActionRequest(t *testing.T, body []byte) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "http://gateway.internal"+workspaceActionPath, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(contextx.RequestUserHeader, "bob")
	req.Header.Set(contextx.ClientSourceHeader, "mobile_action")
	req.Header.Set(contextx.SourceTypeHeader, "message_action")
	req.Header.Set(contextx.TraceIdHeader, "trace-gateway-resign-1")
	if err := controlauth.SignHTTPRequest(req, body, internalHTTPProtectedHeaders(), newIdentityTestSigner(t)); err != nil {
		t.Fatal(err)
	}
	return req
}

func newSignedGatewayAgentDelegationRequest(t *testing.T, method, path string, body []byte) *http.Request {
	t.Helper()
	signer, err := controlauth.NewSigner(gatewayIdentityTestSecret, controlauth.HTTPAgentDelegatedAPIScope)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, "http://gateway.internal"+path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(contextx.RequestUserHeader, "bob")
	req.Header.Set(contextx.UserIDHeader, "42")
	req.Header.Set(contextx.ClientSourceHeader, contextx.ClientSourceAgent)
	req.Header.Set(contextx.SourceTypeHeader, contextx.SourceTypeAgentTool)
	req.Header.Set(contextx.TraceIdHeader, "trace-agent-delegation-resign-1")
	if err := controlauth.SignHTTPRequest(req, body, internalHTTPProtectedHeaders(), signer); err != nil {
		t.Fatal(err)
	}
	return req
}

func setAllForgedIdentityHeaders(header http.Header) {
	for _, name := range contextx.TrustedIdentityHeaderNames() {
		header.Set(name, "forged")
	}
}

func assertTrustedIdentityHeadersEmpty(t *testing.T, header http.Header) {
	t.Helper()
	for _, name := range contextx.TrustedIdentityHeaderNames() {
		if got := header.Get(name); got != "" {
			t.Fatalf("trusted identity header %s = %q, want empty", name, got)
		}
	}
}

func assertHeader(t *testing.T, header http.Header, name, want string) {
	t.Helper()
	if got := header.Get(name); got != want {
		t.Fatalf("header %s = %q, want %q", name, got, want)
	}
}

func useIdentityTestJWTConfig(t *testing.T) func() {
	t.Helper()
	global := config.GetGlobalSharedConfig()
	previous := global.JWT
	global.JWT = config.JWTConfig{
		Secret:             "identity-test-jwt-secret",
		Issuer:             "identity-test",
		AccessTokenExpire:  300,
		RefreshTokenExpire: 300,
	}
	return func() {
		global.JWT = previous
	}
}

func TestReadAndRestoreSignedBodyEnforcesLimit(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, workspaceActionPath, strings.NewReader("12345"))
	if _, err := readAndRestoreSignedBody(req, 4); err == nil {
		t.Fatal("oversized signed body was accepted")
	}
	got, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "12345" {
		t.Fatalf("restored oversized body = %q", got)
	}
}
