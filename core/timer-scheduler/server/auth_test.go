package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/timer-scheduler/model"
	timerservice "github.com/kageos/kageos/core/timer-scheduler/service"
	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/openapitoken"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTimerAPILoopbackAndForgedIdentityStillRequireToken(t *testing.T) {
	router := NewRouter(newTimerHTTPTestService(t))
	req := httptest.NewRequest(http.MethodGet, "/timer/api/v1/tasks", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set(contextx.RequestUserHeader, "system")
	req.Header.Set(contextx.DepartmentFullPathHeader, "/forged/admin")
	req.Header.Set(contextx.ClientSourceHeader, contextx.ClientSourceAgent)
	req.Header.Set(contextx.TokenHeader, "forged-token")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("loopback forged request status = %d, want 401; body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestTimerAPIValidOpenAPITokenWorks(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "timer-openapi.db")), &gorm.Config{})
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
		Name:          "timer auth test",
	})
	if err != nil {
		t.Fatal(err)
	}
	token := created.Secret
	router := NewRouter(newTimerHTTPTestService(t))
	req := httptest.NewRequest(http.MethodGet, "/timer/api/v1/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(contextx.RequestUserHeader, "forged-user")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("OpenAPI token status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestTimerAPIValidAccessTokenWorksAndRebuildsIdentity(t *testing.T) {
	token, err := auth.NewJWTService().GenerateAccessTokenWithContext(auth.UserTokenContext{
		UserID:             42,
		Username:           "alice",
		Email:              "alice@example.com",
		CompanyCode:        "acme",
		DepartmentFullPath: "/acme/engineering",
	})
	if err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/probe", requireTimerAPIAuthentication(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user":    c.GetHeader(contextx.RequestUserHeader),
			"company": c.GetHeader(contextx.CompanyCodeHeader),
			"dept":    c.GetHeader(contextx.DepartmentFullPathHeader),
			"source":  c.GetHeader(contextx.ClientSourceHeader),
		})
	})
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set(contextx.TokenHeader, token)
	req.Header.Set(contextx.RequestUserHeader, "mallory")
	req.Header.Set(contextx.CompanyCodeHeader, "forged")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("valid token status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	for _, want := range []string{`"user":"alice"`, `"company":"acme"`, `"dept":"/acme/engineering"`, `"source":"browser"`} {
		if !strings.Contains(recorder.Body.String(), want) {
			t.Fatalf("response %s missing %s", recorder.Body.String(), want)
		}
	}

	apiRouter := NewRouter(newTimerHTTPTestService(t))
	apiReq := httptest.NewRequest(http.MethodGet, "/timer/api/v1/tasks", nil)
	apiReq.Header.Set(contextx.TokenHeader, token)
	apiRecorder := httptest.NewRecorder()
	apiRouter.ServeHTTP(apiRecorder, apiReq)
	if apiRecorder.Code != http.StatusOK {
		t.Fatalf("authenticated timer API status = %d, want 200; body=%s", apiRecorder.Code, apiRecorder.Body.String())
	}
}

func TestTimerAPIRejectsNonAccessJWT(t *testing.T) {
	refreshToken, err := auth.NewJWTService().GenerateRefreshToken(42, "alice", "alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	router := NewRouter(newTimerHTTPTestService(t))
	req := httptest.NewRequest(http.MethodGet, "/timer/api/v1/tasks", nil)
	req.Header.Set(contextx.TokenHeader, refreshToken)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("refresh token status = %d, want 401; body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestTimerExecutionHTTPWritesAreNotRegistered(t *testing.T) {
	router := NewRouter(newTimerHTTPTestService(t))
	for _, path := range []string{
		"/timer/api/v1/executions/started",
		"/timer/api/v1/executions/heartbeat",
		"/timer/api/v1/executions/finished",
	} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		req.RemoteAddr = "127.0.0.1:12345"
		req.Header.Set(contextx.RequestUserHeader, "system")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("legacy execution endpoint %s status = %d, want 404; body=%s", path, recorder.Code, recorder.Body.String())
		}
	}
}

func newTimerHTTPTestService(t *testing.T) *timerservice.Service {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "timer-http-test.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := model.InitTables(db); err != nil {
		t.Fatal(err)
	}
	return timerservice.NewService(db, timerservice.Options{})
}
