package middleware

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/openapitoken"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestJWTAuthAcceptsAccessTokenAndRejectsRefreshToken(t *testing.T) {
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
	refreshToken, err := auth.NewJWTService().GenerateRefreshTokenWithContext(auth.UserTokenContext{
		UserID:      42,
		Username:    "alice",
		Email:       "alice@example.com",
		CompanyCode: "acme",
	})
	if err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	handlerCalls := 0
	gotUser := ""
	router := gin.New()
	router.GET("/protected", JWTAuth(), func(c *gin.Context) {
		handlerCalls++
		gotUser = ginString(c, contextx.RequestUserHeader)
		c.Status(http.StatusNoContent)
	})

	accessRequest := httptest.NewRequest(http.MethodGet, "/protected", nil)
	accessRequest.Header.Set(contextx.TokenHeader, accessToken)
	accessRequest.Header.Set("X-Forwarded-For", "203.0.113.10")
	accessRecorder := httptest.NewRecorder()
	router.ServeHTTP(accessRecorder, accessRequest)
	if accessRecorder.Code != http.StatusNoContent {
		t.Fatalf("access token status = %d, want 204; body=%s", accessRecorder.Code, accessRecorder.Body.String())
	}
	if gotUser != "alice" {
		t.Fatalf("access token user = %q, want alice", gotUser)
	}

	refreshRequest := httptest.NewRequest(http.MethodGet, "/protected", nil)
	refreshRequest.Header.Set(contextx.TokenHeader, refreshToken)
	refreshRequest.Header.Set("X-Forwarded-For", "203.0.113.10")
	refreshRecorder := httptest.NewRecorder()
	router.ServeHTTP(refreshRecorder, refreshRequest)
	if refreshRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("refresh token status = %d, want 401; body=%s", refreshRecorder.Code, refreshRecorder.Body.String())
	}
	if handlerCalls != 1 {
		t.Fatalf("protected handler calls = %d, want 1", handlerCalls)
	}
}

func TestOptionalIdentityMiddlewareIgnoresRefreshToken(t *testing.T) {
	restore := useMiddlewareJWTTestConfig(t)
	defer restore()
	refreshToken, err := auth.NewJWTService().GenerateRefreshTokenWithContext(auth.UserTokenContext{
		UserID:             42,
		Username:           "alice",
		Email:              "alice@example.com",
		CompanyCode:        "acme",
		DepartmentFullPath: "/acme/engineering",
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		middleware gin.HandlerFunc
	}{
		{name: "JWTAuthOptional", middleware: JWTAuthOptional()},
		{name: "WithUserInfo", middleware: WithUserInfo()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.GET("/optional", tt.middleware, func(c *gin.Context) {
				if got := ginString(c, contextx.RequestUserHeader); got != "" {
					t.Errorf("refresh token established request user %q", got)
				}
				if got := ginString(c, contextx.CompanyCodeHeader); got != "" {
					t.Errorf("refresh token established company %q", got)
				}
				if got := ginString(c, contextx.DepartmentFullPathHeader); got != "" {
					t.Errorf("refresh token established department %q", got)
				}
				c.Status(http.StatusNoContent)
			})
			req := httptest.NewRequest(http.MethodGet, "/optional", nil)
			req.Header.Set(contextx.TokenHeader, refreshToken)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusNoContent {
				t.Fatalf("optional middleware status = %d, want 204", recorder.Code)
			}
		})
	}
}

func TestOpenAPIBearerSubjectIsEnforced(t *testing.T) {
	restore := useMiddlewareJWTTestConfig(t)
	defer restore()
	jwtService := auth.NewJWTService()
	user := auth.UserTokenContext{UserID: 42, Username: "alice", Email: "alice@example.com"}
	accessToken, err := jwtService.GenerateAccessTokenWithContext(user)
	if err != nil {
		t.Fatal(err)
	}
	refreshToken, err := jwtService.GenerateRefreshTokenWithContext(user)
	if err != nil {
		t.Fatal(err)
	}
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "middleware-openapi.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := openapitoken.SetDB(database); err != nil {
		t.Fatal(err)
	}
	created, err := openapitoken.Create(openapitoken.CreateInput{
		OwnerUserID:   user.UserID,
		OwnerUsername: user.Username,
		OwnerEmail:    user.Email,
		Name:          "middleware subject test",
	})
	if err != nil {
		t.Fatal(err)
	}
	openAPIToken := created.Secret

	gin.SetMode(gin.TestMode)
	protectedCalls := 0
	protectedUser := ""
	protectedRouter := gin.New()
	protectedRouter.GET("/protected", JWTAuth(), func(c *gin.Context) {
		protectedCalls++
		protectedUser = ginString(c, contextx.RequestUserHeader)
		c.Status(http.StatusNoContent)
	})
	for name, token := range map[string]string{"access": accessToken, "refresh": refreshToken} {
		t.Run("JWTAuth rejects "+name+" bearer", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			recorder := httptest.NewRecorder()
			protectedRouter.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("%s bearer status = %d, want 401; body=%s", name, recorder.Code, recorder.Body.String())
			}
		})
	}
	openAPIRequest := httptest.NewRequest(http.MethodGet, "/protected", nil)
	openAPIRequest.Header.Set("Authorization", "Bearer "+openAPIToken)
	openAPIRecorder := httptest.NewRecorder()
	protectedRouter.ServeHTTP(openAPIRecorder, openAPIRequest)
	if openAPIRecorder.Code != http.StatusNoContent {
		t.Fatalf("OpenAPI bearer status = %d, want 204; body=%s", openAPIRecorder.Code, openAPIRecorder.Body.String())
	}
	if protectedCalls != 1 || protectedUser != "alice" {
		t.Fatalf("OpenAPI protected result: calls=%d user=%q", protectedCalls, protectedUser)
	}

	for name, test := range map[string]struct {
		token    string
		wantUser string
	}{
		"access":  {token: accessToken},
		"refresh": {token: refreshToken},
		"openapi": {token: openAPIToken, wantUser: "alice"},
	} {
		t.Run("WithUserInfo "+name+" bearer", func(t *testing.T) {
			gotUser := ""
			router := gin.New()
			router.GET("/optional", WithUserInfo(), func(c *gin.Context) {
				gotUser = ginString(c, contextx.RequestUserHeader)
				c.Status(http.StatusNoContent)
			})
			req := httptest.NewRequest(http.MethodGet, "/optional", nil)
			req.Header.Set("Authorization", "Bearer "+test.token)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusNoContent {
				t.Fatalf("WithUserInfo %s bearer status = %d, want 204", name, recorder.Code)
			}
			if gotUser != test.wantUser {
				t.Fatalf("WithUserInfo %s bearer user = %q, want %q", name, gotUser, test.wantUser)
			}
		})
	}
}

func TestJWTOrPubKeyAuthRejectsRefreshTokenJWTBranch(t *testing.T) {
	restore := useMiddlewareJWTTestConfig(t)
	defer restore()
	refreshToken, err := auth.NewJWTService().GenerateRefreshToken(42, "alice", "alice@example.com")
	if err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	handlerCalled := false
	router := gin.New()
	router.GET("/protected", JWTOrPubKeyAuth(nil), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(contextx.TokenHeader, refreshToken)
	req.Header.Set("X-Forwarded-For", "203.0.113.10")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("JWTOrPubKey refresh status = %d, want 401; body=%s", recorder.Code, recorder.Body.String())
	}
	if handlerCalled {
		t.Fatal("JWTOrPubKey protected handler ran for a refresh token")
	}
}

func ginString(c *gin.Context, key string) string {
	value, exists := c.Get(key)
	if !exists {
		return ""
	}
	text, _ := value.(string)
	return text
}

func useMiddlewareJWTTestConfig(t *testing.T) func() {
	t.Helper()
	global := config.GetGlobalSharedConfig()
	previous := global.JWT
	global.JWT = config.JWTConfig{
		Secret:             "middleware-jwt-subject-test-secret",
		Issuer:             "middleware-jwt-subject-test",
		AccessTokenExpire:  300,
		RefreshTokenExpire: 300,
	}
	return func() {
		global.JWT = previous
	}
}
