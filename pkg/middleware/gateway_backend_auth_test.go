package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/controlauth"
)

const gatewayBackendAuthTestSecret = "gateway-backend-auth-test-secret-32-bytes"

func TestJWTAuthRejectsLoopbackForgedIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handlerCalls := 0
	router := gin.New()
	router.GET("/protected", JWTAuth(newGatewayBackendTestVerifier(t)), func(c *gin.Context) {
		handlerCalls++
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "http://backend.internal/protected", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set(contextx.RequestUserHeader, "system")
	req.Header.Set(contextx.UsernameHeader, "system")
	req.Header.Set(contextx.CompanyCodeHeader, "victim-company")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", recorder.Code, recorder.Body.String())
	}
	if handlerCalls != 0 {
		t.Fatalf("handler calls = %d, want 0", handlerCalls)
	}
}

func TestJWTAuthAcceptsGatewaySignatureAndRebuildsIdentity(t *testing.T) {
	body := []byte(`{"value":"ok"}`)
	verifier := newGatewayBackendTestVerifier(t)
	signer := newGatewayBackendTestSigner(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/protected", JWTAuth(verifier), func(c *gin.Context) {
		if got := c.GetHeader(contextx.RequestUserHeader); got != "alice" {
			t.Errorf("request user = %q, want alice", got)
		}
		if got := c.GetHeader(contextx.UserIDHeader); got != "42" {
			t.Errorf("user id = %q, want 42", got)
		}
		if got := c.GetHeader(contextx.CompanyCodeHeader); got != "acme" {
			t.Errorf("company code = %q, want acme", got)
		}
		if got, exists := c.Get("user_id"); !exists || got != int64(42) {
			t.Errorf("context user_id = %#v, exists=%v", got, exists)
		}
		if controlauth.HasHTTPMetadata(c.Request.Header) {
			t.Error("gateway signature metadata survived backend verification")
		}
		gotBody, err := io.ReadAll(c.Request.Body)
		if err != nil {
			t.Errorf("read restored body: %v", err)
		} else if !bytes.Equal(gotBody, body) {
			t.Errorf("restored body = %q, want %q", gotBody, body)
		}
		c.Status(http.StatusNoContent)
	})

	req := newSignedGatewayBackendTestRequest(t, http.MethodPost, "/protected", body, signer)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestJWTAuthRejectsTamperedReplayedAndPartialGatewaySignature(t *testing.T) {
	body := []byte(`{"value":"ok"}`)
	signer := newGatewayBackendTestSigner(t)
	verifier := newGatewayBackendTestVerifier(t)

	gin.SetMode(gin.TestMode)
	handlerCalls := 0
	router := gin.New()
	router.POST("/protected", JWTAuth(verifier), func(c *gin.Context) {
		handlerCalls++
		c.Status(http.StatusNoContent)
	})

	signed := newSignedGatewayBackendTestRequest(t, http.MethodPost, "/protected", body, signer)
	signedHeaders := signed.Header.Clone()

	tampered := httptest.NewRequest(http.MethodPost, "http://backend.internal/protected", bytes.NewReader(body))
	tampered.Header = signedHeaders.Clone()
	tampered.Header.Set(contextx.RequestUserHeader, "mallory")
	assertMiddlewareStatus(t, router, tampered, http.StatusUnauthorized)

	first := httptest.NewRequest(http.MethodPost, "http://backend.internal/protected", bytes.NewReader(body))
	first.Header = signedHeaders.Clone()
	assertMiddlewareStatus(t, router, first, http.StatusNoContent)

	replay := httptest.NewRequest(http.MethodPost, "http://backend.internal/protected", bytes.NewReader(body))
	replay.Header = signedHeaders.Clone()
	assertMiddlewareStatus(t, router, replay, http.StatusUnauthorized)

	restore := useMiddlewareJWTTestConfig(t)
	defer restore()
	accessToken, err := auth.NewJWTService().GenerateAccessTokenWithContext(auth.UserTokenContext{UserID: 42, Username: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	partial := httptest.NewRequest(http.MethodPost, "http://backend.internal/protected", bytes.NewReader(body))
	partial.Header.Set(contextx.TokenHeader, accessToken)
	partial.Header.Set(controlauth.HTTPNonceHeader, "partial-control-auth")
	assertMiddlewareStatus(t, router, partial, http.StatusUnauthorized)

	if handlerCalls != 1 {
		t.Fatalf("handler calls = %d, want only the first valid request", handlerCalls)
	}
}

func TestGatewayBackendAuthRequiresHMACNotAccessJWT(t *testing.T) {
	restore := useMiddlewareJWTTestConfig(t)
	defer restore()
	accessToken, err := auth.NewJWTService().GenerateAccessTokenWithContext(auth.UserTokenContext{UserID: 42, Username: "alice"})
	if err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/internal", GatewayBackendAuth(newGatewayBackendTestVerifier(t)), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	credentialOnly := httptest.NewRequest(http.MethodGet, "http://backend.internal/internal", nil)
	credentialOnly.Header.Set(contextx.TokenHeader, accessToken)
	assertMiddlewareStatus(t, router, credentialOnly, http.StatusUnauthorized)

	signed := newSignedGatewayBackendTestRequest(t, http.MethodGet, "/internal", nil, newGatewayBackendTestSigner(t))
	assertMiddlewareStatus(t, router, signed, http.StatusNoContent)
}

func TestOptionalAndPubKeyMiddlewareNeverTrustBareIdentity(t *testing.T) {
	for _, tc := range []struct {
		name       string
		middleware gin.HandlerFunc
		wantStatus int
	}{
		{name: "WithUserInfo", middleware: WithUserInfo(), wantStatus: http.StatusNoContent},
		{name: "JWTAuthOptional", middleware: JWTAuthOptional(), wantStatus: http.StatusNoContent},
		{name: "JWTOrPubKeyAuth", middleware: JWTOrPubKeyAuth(nil), wantStatus: http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			handlerCalls := 0
			router := gin.New()
			router.GET("/probe", tc.middleware, func(c *gin.Context) {
				handlerCalls++
				if got := c.GetHeader(contextx.RequestUserHeader); got != "" {
					t.Errorf("bare request user survived: %q", got)
				}
				if got, exists := c.Get(contextx.RequestUserHeader); exists && got != "" {
					t.Errorf("bare context user survived: %#v", got)
				}
				c.Status(http.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodGet, "http://backend.internal/probe", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			req.Header.Set(contextx.RequestUserHeader, "system")
			req.Header.Set(contextx.UsernameHeader, "system")
			req.Header.Set(contextx.CompanyCodeHeader, "victim-company")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			if recorder.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, tc.wantStatus, recorder.Body.String())
			}
			if tc.wantStatus == http.StatusNoContent && handlerCalls != 1 {
				t.Fatalf("optional handler calls = %d, want 1", handlerCalls)
			}
			if tc.wantStatus != http.StatusNoContent && handlerCalls != 0 {
				t.Fatalf("protected handler calls = %d, want 0", handlerCalls)
			}
		})
	}
}

func newGatewayBackendTestSigner(t *testing.T) *controlauth.Signer {
	t.Helper()
	signer, err := controlauth.NewSigner(gatewayBackendAuthTestSecret, controlauth.HTTPGatewayDelegatedBackendScope)
	if err != nil {
		t.Fatal(err)
	}
	return signer
}

func newGatewayBackendTestVerifier(t *testing.T) *controlauth.Verifier {
	t.Helper()
	verifier, err := controlauth.NewVerifier(
		gatewayBackendAuthTestSecret,
		controlauth.HTTPGatewayDelegatedBackendScope,
		controlauth.VerifierOptions{MaxAge: time.Minute, MaxFutureSkew: time.Second},
	)
	if err != nil {
		t.Fatal(err)
	}
	return verifier
}

func newSignedGatewayBackendTestRequest(t *testing.T, method, path string, body []byte, signer *controlauth.Signer) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, "http://backend.internal"+path, bytes.NewReader(body))
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set(contextx.RequestUserHeader, "alice")
	req.Header.Set(contextx.UserIDHeader, "42")
	req.Header.Set(contextx.UserEmailHeader, "alice@example.com")
	req.Header.Set(contextx.CompanyCodeHeader, "acme")
	req.Header.Set(contextx.ClientSourceHeader, contextx.ClientSourceAgent)
	req.Header.Set(contextx.TraceIdHeader, "trace-gateway-backend-test")
	if err := controlauth.SignHTTPRequest(req, body, gatewayBackendProtectedHeaders(), signer); err != nil {
		t.Fatal(err)
	}
	return req
}

func assertMiddlewareStatus(t *testing.T, router http.Handler, req *http.Request, want int) {
	t.Helper()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != want {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, want, strings.TrimSpace(recorder.Body.String()))
	}
}
