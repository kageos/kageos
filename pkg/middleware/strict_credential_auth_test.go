package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/controlauth"
)

func TestStrictCredentialAuthRejectsForgedIdentityAndNonAccessJWT(t *testing.T) {
	restore := useMiddlewareJWTTestConfig(t)
	defer restore()
	refreshToken, err := auth.NewJWTService().GenerateRefreshTokenWithContext(auth.UserTokenContext{
		UserID: 42, Username: "alice", Email: "alice@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	handlerCalls := 0
	router := gin.New()
	router.GET("/protected", StrictCredentialAuth(), func(c *gin.Context) {
		handlerCalls++
		c.Status(http.StatusNoContent)
	})

	for _, tc := range []struct {
		name  string
		setup func(*http.Request)
	}{
		{
			name: "loopback forged identity",
			setup: func(req *http.Request) {
				req.Header.Set("X-Forwarded-For", "127.0.0.1")
				req.Header.Set(contextx.RequestUserHeader, "victim")
				req.Header.Set(contextx.CompanyCodeHeader, "victim-company")
			},
		},
		{
			name: "refresh token",
			setup: func(req *http.Request) {
				req.Header.Set(contextx.TokenHeader, refreshToken)
			},
		},
		{
			name: "competing credentials",
			setup: func(req *http.Request) {
				req.Header.Set(contextx.TokenHeader, refreshToken)
				req.Header.Set("Authorization", "Bearer "+refreshToken)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			tc.setup(req)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401; body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
	if handlerCalls != 0 {
		t.Fatalf("protected handler calls = %d, want 0", handlerCalls)
	}
}

func TestStrictCredentialAuthRejectsPartialControlMetadata(t *testing.T) {
	restore := useMiddlewareJWTTestConfig(t)
	defer restore()
	accessToken, err := auth.NewJWTService().GenerateAccessTokenWithContext(auth.UserTokenContext{
		UserID:      42,
		Username:    "alice",
		Email:       "alice@example.com",
		CompanyCode: "acme",
	})
	if err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlerCalls := 0
	router.GET("/protected", StrictCredentialAuth(), func(c *gin.Context) {
		handlerCalls++
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(contextx.TokenHeader, accessToken)
	req.Header.Set(contextx.RequestUserHeader, "victim")
	req.Header.Set(contextx.CompanyCodeHeader, "victim-company")
	req.Header.Set(controlauth.HTTPNonceHeader, "forged-partial-metadata")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", recorder.Code, recorder.Body.String())
	}
	if handlerCalls != 0 {
		t.Fatalf("handler calls = %d, want 0", handlerCalls)
	}
}
