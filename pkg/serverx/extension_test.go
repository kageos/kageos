package serverx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/license"
)

func TestRegisteredMiddlewaresUsesGlobalThenService(t *testing.T) {
	resetExtensionRegistryForTest(t)

	RegisterMiddleware(ServiceGlobal, func() gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Set("order", "global")
			c.Next()
		}
	})
	RegisterMiddleware(ServiceHRServer, func() gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Set("order", c.GetString("order")+",service")
			c.Next()
		}
	})

	if got := len(RegisteredMiddlewares(ServiceAppServer)); got != 1 {
		t.Fatalf("app-server middleware count = %d, want 1", got)
	}
	if got := len(RegisteredMiddlewares(ServiceHRServer)); got != 2 {
		t.Fatalf("hr-server middleware count = %d, want 2", got)
	}
}

func TestApplyRouteRegistrarsUsesGlobalThenService(t *testing.T) {
	resetExtensionRegistryForTest(t)
	gin.SetMode(gin.TestMode)

	RegisterRoutes(ServiceGlobal, func(engine *gin.Engine) {
		engine.GET("/global", func(c *gin.Context) {
			c.String(http.StatusOK, "global")
		})
	})
	RegisterRoutes(ServiceAppServer, func(engine *gin.Engine) {
		engine.GET("/service", func(c *gin.Context) {
			c.String(http.StatusOK, "service")
		})
	})

	engine := gin.New()
	ApplyRouteRegistrars(ServiceAppServer, engine)

	assertRouteStatus(t, engine, "/global", http.StatusOK)
	assertRouteStatus(t, engine, "/service", http.StatusOK)
	assertRouteStatus(t, engine, "/runtime/info", http.StatusOK)
	assertRouteStatus(t, engine, "/missing", http.StatusNotFound)
}

func TestApplyRouteRegistrarsAddsRuntimeInfoRoute(t *testing.T) {
	resetExtensionRegistryForTest(t)
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	ApplyRouteRegistrars(ServiceAppServer, engine)

	req := httptest.NewRequest(http.MethodGet, "/runtime/info", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /runtime/info status = %d, want %d", rec.Code, http.StatusOK)
	}

	var info license.Info
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
		t.Fatalf("decode runtime info: %v", err)
	}
	if info.Service != string(ServiceAppServer) {
		t.Fatalf("service = %q, want %q", info.Service, ServiceAppServer)
	}
	if info.Edition != license.EditionCommunity {
		t.Fatalf("edition = %q, want %q", info.Edition, license.EditionCommunity)
	}
}

func resetExtensionRegistryForTest(t *testing.T) {
	t.Helper()
	extensionRegistry.Lock()
	extensionRegistry.middleware = make(map[string][]MiddlewareFactory)
	extensionRegistry.routes = make(map[string][]RouteRegistrar)
	extensionRegistry.Unlock()
	t.Cleanup(func() {
		extensionRegistry.Lock()
		extensionRegistry.middleware = make(map[string][]MiddlewareFactory)
		extensionRegistry.routes = make(map[string][]RouteRegistrar)
		extensionRegistry.Unlock()
	})
}

func assertRouteStatus(t *testing.T, engine *gin.Engine, path string, want int) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != want {
		t.Fatalf("GET %s status = %d, want %d", path, rec.Code, want)
	}
}
