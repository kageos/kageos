package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/kageos/kageos/pkg/openapitoken"
	"github.com/kageos/kageos/pkg/publicshare"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const agentAuthTestSecret = "0123456789abcdef0123456789abcdef"

func TestAgentAPILoopbackForgedIdentityIsUnauthorized(t *testing.T) {
	s := newAgentAuthTestServer(t)
	router := newAgentAuthProbeRouter(s, nil)
	req := httptest.NewRequest(http.MethodGet, "http://agent.internal/agent/api/v1/probe", nil)
	req.RemoteAddr = "127.0.0.1:43210"
	setAgentForgedIdentity(req.Header)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("loopback forged request status = %d, want 401; body=%s", recorder.Code, recorder.Body.String())
	}
	assertAgentTrustedIdentityEmpty(t, req.Header)
}

func TestAgentAPIValidJWTOverwritesForgedIdentity(t *testing.T) {
	restore := useAgentAuthTestJWTConfig(t)
	defer restore()
	token, err := auth.NewJWTService().GenerateAccessTokenWithContext(auth.UserTokenContext{
		UserID:             42,
		Username:           "alice",
		Email:              "alice@example.com",
		CompanyCode:        "acme",
		CompanyName:        "Acme",
		DepartmentFullPath: "/acme/engineering",
	})
	if err != nil {
		t.Fatal(err)
	}

	var gotIdentity http.Header
	s := newAgentAuthTestServer(t)
	router := newAgentAuthProbeRouter(s, func(c *gin.Context) {
		gotIdentity = c.Request.Header.Clone()
	})
	req := httptest.NewRequest(http.MethodGet, "http://agent.internal/agent/api/v1/probe", nil)
	req.RemoteAddr = "127.0.0.1:43210"
	setAgentForgedIdentity(req.Header)
	req.Header.Set(contextx.TokenHeader, token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("valid JWT status = %d, want 204; body=%s", recorder.Code, recorder.Body.String())
	}
	assertAgentHeader(t, gotIdentity, contextx.RequestUserHeader, "alice")
	assertAgentHeader(t, gotIdentity, contextx.DepartmentFullPathHeader, "/acme/engineering")
	assertAgentHeader(t, gotIdentity, contextx.CompanyCodeHeader, "acme")
	assertAgentHeader(t, gotIdentity, contextx.ClientSourceHeader, contextx.ClientSourceBrowser)
	if got := gotIdentity.Get(contextx.SourceTypeHeader); got != "" {
		t.Fatalf("forged source type survived JWT rebuild: %q", got)
	}
}

func TestAgentAPIValidOpenAPIBearerRebuildsIdentity(t *testing.T) {
	restore := useAgentAuthTestJWTConfig(t)
	defer restore()
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "agent-openapi.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := openapitoken.SetDB(database); err != nil {
		t.Fatal(err)
	}
	created, err := openapitoken.Create(openapitoken.CreateInput{
		OwnerUserID:   77,
		OwnerUsername: "automation",
		OwnerEmail:    "automation@example.com",
		CompanyCode:   "acme",
		Name:          "agent auth test",
	})
	if err != nil {
		t.Fatal(err)
	}
	token := created.Secret

	var gotIdentity http.Header
	s := newAgentAuthTestServer(t)
	router := newAgentAuthProbeRouter(s, func(c *gin.Context) {
		gotIdentity = c.Request.Header.Clone()
	})
	req := httptest.NewRequest(http.MethodGet, "http://agent.internal/agent/api/v1/probe", nil)
	setAgentForgedIdentity(req.Header)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("valid OpenAPI bearer status = %d, want 204; body=%s", recorder.Code, recorder.Body.String())
	}
	assertAgentHeader(t, gotIdentity, contextx.RequestUserHeader, "automation")
	assertAgentHeader(t, gotIdentity, contextx.ClientSourceHeader, contextx.ClientSourceOpenAPI)
	assertAgentHeader(t, gotIdentity, contextx.SourceTypeHeader, contextx.SourceTypeOpenAPIToken)
	assertAgentHeader(t, gotIdentity, contextx.SourceRefHeader, "automation")
	assertAgentHeader(t, gotIdentity, contextx.TokenHeader, token)
}

func TestAgentAPIAcceptsGatewayWorkspaceActionSignature(t *testing.T) {
	body := []byte(`{"full_code_path":"/alice/ops","message":{"content":"run"}}`)
	signer := newAgentBackendTestSigner(t)
	s := newAgentAuthTestServer(t)
	var gotIdentity http.Header
	router := newAgentAuthProbeRouterAt(s, agentWorkspaceActionPath, func(c *gin.Context) {
		gotIdentity = c.Request.Header.Clone()
	})
	req := newSignedAgentWorkspaceActionRequest(t, body, signer)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("signed workspace action status = %d, want 204; body=%s", recorder.Code, recorder.Body.String())
	}
	assertAgentHeader(t, gotIdentity, contextx.RequestUserHeader, "bob")
	assertAgentHeader(t, gotIdentity, contextx.ClientSourceHeader, "mobile_action")
	if controlauth.HasHTTPMetadata(gotIdentity) {
		t.Fatal("Agent handler observed reusable gateway authentication metadata")
	}
}

func TestAgentAPIRejectsMessageServerFirstHopScope(t *testing.T) {
	body := []byte(`{"message":"run"}`)
	firstHopSigner, err := controlauth.NewSigner(agentAuthTestSecret, controlauth.HTTPWorkspaceActionScope)
	if err != nil {
		t.Fatal(err)
	}
	s := newAgentAuthTestServer(t)
	router := newAgentAuthProbeRouterAt(s, agentWorkspaceActionPath, nil)
	req := newSignedAgentWorkspaceActionRequest(t, body, firstHopSigner)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("message-server first-hop signature status = %d, want 401", recorder.Code)
	}
	assertAgentTrustedIdentityEmpty(t, req.Header)
}

func TestAgentAPIRejectsTamperedAndReplayedGatewaySignature(t *testing.T) {
	body := []byte(`{"message":"run"}`)
	signer := newAgentBackendTestSigner(t)
	s := newAgentAuthTestServer(t)
	handlerCalls := 0
	router := newAgentAuthProbeRouterAt(s, agentWorkspaceActionPath, func(_ *gin.Context) {
		handlerCalls++
	})
	signed := newSignedAgentWorkspaceActionRequest(t, body, signer)
	signedHeaders := signed.Header.Clone()

	tampered := httptest.NewRequest(http.MethodPost, "http://agent.internal"+agentWorkspaceActionPath, strings.NewReader(`{"message":"tampered"}`))
	tampered.Header = signedHeaders.Clone()
	tamperedRecorder := httptest.NewRecorder()
	router.ServeHTTP(tamperedRecorder, tampered)
	if tamperedRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("tampered signature status = %d, want 401", tamperedRecorder.Code)
	}
	assertAgentTrustedIdentityEmpty(t, tampered.Header)
	if controlauth.HasHTTPMetadata(tampered.Header) {
		t.Fatal("tampered request retained authentication metadata")
	}

	first := httptest.NewRequest(http.MethodPost, "http://agent.internal"+agentWorkspaceActionPath, bytes.NewReader(body))
	first.Header = signedHeaders.Clone()
	firstRecorder := httptest.NewRecorder()
	router.ServeHTTP(firstRecorder, first)
	if firstRecorder.Code != http.StatusNoContent {
		t.Fatalf("first signed request status = %d, want 204; body=%s", firstRecorder.Code, firstRecorder.Body.String())
	}

	replay := httptest.NewRequest(http.MethodPost, "http://agent.internal"+agentWorkspaceActionPath, bytes.NewReader(body))
	replay.Header = signedHeaders.Clone()
	replayRecorder := httptest.NewRecorder()
	router.ServeHTTP(replayRecorder, replay)
	if replayRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("replayed signature status = %d, want 401", replayRecorder.Code)
	}
	if handlerCalls != 1 {
		t.Fatalf("authenticated handler calls = %d, want 1", handlerCalls)
	}
}

func TestAgentSignedWorkspaceCanDelegateFirstWorkspaceContextCall(t *testing.T) {
	delegationVerifier, err := controlauth.NewVerifier(
		agentAuthTestSecret,
		controlauth.HTTPAgentDelegatedAPIScope,
		controlauth.VerifierOptions{MaxAge: time.Minute, MaxFutureSkew: time.Second},
	)
	if err != nil {
		t.Fatal(err)
	}
	fakeGateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		body, readErr := io.ReadAll(req.Body)
		if readErr != nil {
			t.Errorf("read delegated body: %v", readErr)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := controlauth.VerifyHTTPRequest(req, body, agentHTTPProtectedHeaders(), delegationVerifier); err != nil {
			t.Errorf("verify delegated workspace request: %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if req.URL.Path != "/workspace/api/v1/workspace/context" {
			t.Errorf("delegated path = %q", req.URL.Path)
		}
		if req.Header.Get(contextx.RequestUserHeader) != "bob" {
			t.Errorf("delegated request user = %q", req.Header.Get(contextx.RequestUserHeader))
		}
		if req.Header.Get(contextx.TokenHeader) != "" || req.Header.Get("Authorization") != "" {
			t.Error("delegated workspace request propagated a user credential")
		}
		controlauth.ClearHTTPMetadata(req.Header)
		if controlauth.HasHTTPMetadata(req.Header) {
			t.Error("delegation metadata survived gateway consumption")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0,
			"msg":  "ok",
			"data": map[string]interface{}{
				"directory": map[string]interface{}{"id": 1, "name": "Ops", "code": "ops"},
			},
		})
	}))
	defer fakeGateway.Close()
	t.Setenv("GATEWAY_URL", fakeGateway.URL)

	apiSigner, err := controlauth.NewSigner(agentAuthTestSecret, controlauth.HTTPAgentDelegatedAPIScope)
	if err != nil {
		t.Fatal(err)
	}
	timerSigner, err := controlauth.NewSigner(agentAuthTestSecret, controlauth.HTTPAgentDelegatedTimerScope)
	if err != nil {
		t.Fatal(err)
	}
	requestSigner, err := newAgentDelegatedHTTPRequestSigner(fakeGateway.URL, apiSigner, timerSigner)
	if err != nil {
		t.Fatal(err)
	}
	s := newAgentAuthTestServer(t)
	s.agentDelegationSigner = requestSigner
	var workspaceErr error
	var directoryName string
	router := newAgentAuthProbeRouterAt(s, agentWorkspaceActionPath, func(c *gin.Context) {
		workspace, err := apicall.GetWorkspaceContext(contextx.ToContext(c), "/bob/ops", "")
		workspaceErr = err
		if workspace != nil {
			directoryName = workspace.Directory.Name
		}
	})
	req := newSignedAgentWorkspaceActionRequest(t, []byte(`{"message":"run"}`), newAgentBackendTestSigner(t))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("Agent probe status = %d, want 204; body=%s", recorder.Code, recorder.Body.String())
	}
	if workspaceErr != nil || directoryName != "Ops" {
		t.Fatalf("first GetWorkspaceContext result: name=%q err=%v", directoryName, workspaceErr)
	}
}

func TestAgentDelegationDoesNotLeakCredentialsToExternalOrigin(t *testing.T) {
	apiSigner, err := controlauth.NewSigner(agentAuthTestSecret, controlauth.HTTPAgentDelegatedAPIScope)
	if err != nil {
		t.Fatal(err)
	}
	timerSigner, err := controlauth.NewSigner(agentAuthTestSecret, controlauth.HTTPAgentDelegatedTimerScope)
	if err != nil {
		t.Fatal(err)
	}
	requestSigner, err := newAgentDelegatedHTTPRequestSigner("http://gateway.internal", apiSigner, timerSigner)
	if err != nil {
		t.Fatal(err)
	}
	ctx := controlauth.WithDelegatedHTTPRequestSigner(context.Background(), requestSigner)
	req := httptest.NewRequest(http.MethodPost, "https://external.example/upload", strings.NewReader(`{"secret":true}`)).WithContext(ctx)
	req.Header.Set(contextx.TokenHeader, "access-secret")
	req.Header.Set("Authorization", "Bearer openapi-secret")
	req.Header.Set(contextx.PubKeyHerder, "pub-secret")
	req.Header.Set(publicshare.AnonymousTokenHeader, "anonymous-secret")
	setAgentForgedIdentity(req.Header)
	delegated, err := controlauth.ApplyDelegatedHTTPRequestSignature(req, []byte(`{"secret":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if delegated {
		t.Fatal("external origin received a delegation signature")
	}
	assertAgentTrustedIdentityEmpty(t, req.Header)
	for _, name := range []string{contextx.TokenHeader, "Authorization", contextx.PubKeyHerder, publicshare.AnonymousTokenHeader} {
		if got := req.Header.Get(name); got != "" {
			t.Fatalf("external request credential %s = %q", name, got)
		}
	}
	if controlauth.HasHTTPMetadata(req.Header) {
		t.Fatal("external origin received control authentication metadata")
	}
}

func TestAgentBusinessRoutesUseStrictAuthentication(t *testing.T) {
	s := newAgentAuthTestServer(t)
	s.cfg = &config.AgentServerConfig{}
	s.httpServer = gin.New()
	s.setupRoutes()

	healthRecorder := httptest.NewRecorder()
	s.httpServer.ServeHTTP(healthRecorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	if healthRecorder.Code != http.StatusOK {
		t.Fatalf("public health status = %d, want 200", healthRecorder.Code)
	}
	swaggerRecorder := httptest.NewRecorder()
	s.httpServer.ServeHTTP(swaggerRecorder, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
	if swaggerRecorder.Code != http.StatusOK {
		t.Fatalf("public Swagger status = %d, want 200", swaggerRecorder.Code)
	}

	businessRequest := httptest.NewRequest(http.MethodGet, "/agent/api/v1/state/runtime-summary", nil)
	businessRequest.RemoteAddr = "127.0.0.1:12345"
	businessRequest.Header.Set(contextx.RequestUserHeader, "forged")
	businessRecorder := httptest.NewRecorder()
	s.httpServer.ServeHTTP(businessRecorder, businessRequest)
	if businessRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated business route status = %d, want 401; body=%s", businessRecorder.Code, businessRecorder.Body.String())
	}
}

func newAgentAuthTestServer(t *testing.T) *Server {
	t.Helper()
	verifier, err := controlauth.NewVerifier(
		agentAuthTestSecret,
		controlauth.HTTPGatewayAgentBackendScope,
		controlauth.VerifierOptions{MaxAge: time.Minute, MaxFutureSkew: time.Second},
	)
	if err != nil {
		t.Fatal(err)
	}
	return &Server{agentBackendVerifier: verifier}
}

func newAgentBackendTestSigner(t *testing.T) *controlauth.Signer {
	t.Helper()
	signer, err := controlauth.NewSigner(agentAuthTestSecret, controlauth.HTTPGatewayAgentBackendScope)
	if err != nil {
		t.Fatal(err)
	}
	return signer
}

func newAgentAuthProbeRouter(s *Server, inspect func(*gin.Context)) *gin.Engine {
	return newAgentAuthProbeRouterAt(s, "/agent/api/v1/probe", inspect)
}

func newAgentAuthProbeRouterAt(s *Server, path string, inspect func(*gin.Context)) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(http.MethodGet, path, s.requireAgentAPIAuthentication(), func(c *gin.Context) {
		if inspect != nil {
			inspect(c)
		}
		c.Status(http.StatusNoContent)
	})
	router.Handle(http.MethodPost, path, s.requireAgentAPIAuthentication(), func(c *gin.Context) {
		if inspect != nil {
			inspect(c)
		}
		c.Status(http.StatusNoContent)
	})
	return router
}

func newSignedAgentWorkspaceActionRequest(t *testing.T, body []byte, signer *controlauth.Signer) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "http://agent.internal"+agentWorkspaceActionPath, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(contextx.RequestUserHeader, "bob")
	req.Header.Set(contextx.ClientSourceHeader, "mobile_action")
	req.Header.Set(contextx.SourceTypeHeader, "message_action")
	req.Header.Set(contextx.TraceIdHeader, "trace-agent-auth-1")
	if err := controlauth.SignHTTPRequest(req, body, agentHTTPProtectedHeaders(), signer); err != nil {
		t.Fatal(err)
	}
	return req
}

func setAgentForgedIdentity(header http.Header) {
	for _, name := range contextx.TrustedIdentityHeaderNames() {
		header.Set(name, "forged")
	}
}

func assertAgentTrustedIdentityEmpty(t *testing.T, header http.Header) {
	t.Helper()
	for _, name := range contextx.TrustedIdentityHeaderNames() {
		if got := header.Get(name); got != "" {
			t.Fatalf("trusted identity header %s = %q, want empty", name, got)
		}
	}
}

func assertAgentHeader(t *testing.T, header http.Header, name, want string) {
	t.Helper()
	if got := header.Get(name); got != want {
		t.Fatalf("header %s = %q, want %q", name, got, want)
	}
}

func useAgentAuthTestJWTConfig(t *testing.T) func() {
	t.Helper()
	global := config.GetGlobalSharedConfig()
	previous := global.JWT
	global.JWT = config.JWTConfig{
		Secret:            "agent-auth-test-jwt-secret",
		Issuer:            "agent-auth-test",
		AccessTokenExpire: 300,
	}
	return func() {
		global.JWT = previous
	}
}
