package serverx

import (
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/license"
)

type ServiceName string

const (
	ServiceGlobal          ServiceName = "*"
	ServiceAPIGateway      ServiceName = "api-gateway"
	ServiceHRServer        ServiceName = "hr-server"
	ServiceAppServer       ServiceName = "app-server"
	ServiceMessageServer   ServiceName = "message-server"
	ServiceAgentServer     ServiceName = "agent-server"
	ServiceAppStorage      ServiceName = "app-storage"
	ServiceConnectorServer ServiceName = "connector-server"
	ServiceTimerScheduler  ServiceName = "timer-scheduler"
)

const globalServiceKey = string(ServiceGlobal)

// MiddlewareFactory builds a Gin middleware for a named service.
type MiddlewareFactory func() gin.HandlerFunc

// RouteRegistrar attaches routes to a Gin engine for a named service.
type RouteRegistrar func(*gin.Engine)

var extensionRegistry = struct {
	sync.RWMutex
	middleware map[string][]MiddlewareFactory
	routes     map[string][]RouteRegistrar
}{
	middleware: make(map[string][]MiddlewareFactory),
	routes:     make(map[string][]RouteRegistrar),
}

// RegisterMiddleware registers a middleware factory for a service.kage
// Use service "*" to apply it to every service that opts into registered middleware.
func RegisterMiddleware(service ServiceName, factory MiddlewareFactory) {
	if factory == nil {
		panic("serverx middleware factory is nil")
	}
	key := normalizeServiceKey(service)
	extensionRegistry.Lock()
	defer extensionRegistry.Unlock()
	extensionRegistry.middleware[key] = append(extensionRegistry.middleware[key], factory)
}

// RegisteredMiddlewares builds registered middleware for a service.
func RegisteredMiddlewares(service ServiceName) []gin.HandlerFunc {
	keys := serviceKeys(service)
	extensionRegistry.RLock()
	factories := make([]MiddlewareFactory, 0)
	for _, key := range keys {
		factories = append(factories, extensionRegistry.middleware[key]...)
	}
	extensionRegistry.RUnlock()

	out := make([]gin.HandlerFunc, 0, len(factories))
	for _, factory := range factories {
		if handler := factory(); handler != nil {
			out = append(out, handler)
		}
	}
	return out
}

// WithRegisteredMiddlewares appends registered middleware for a service.
func WithRegisteredMiddlewares(service ServiceName) GinOption {
	return WithMiddleware(RegisteredMiddlewares(service)...)
}

// RegisterRoutes registers a route registrar for a service.
// Use service "*" to apply it to every service that calls ApplyRouteRegistrars.
func RegisterRoutes(service ServiceName, registrar RouteRegistrar) {
	if registrar == nil {
		panic("serverx route registrar is nil")
	}
	key := normalizeServiceKey(service)
	extensionRegistry.Lock()
	defer extensionRegistry.Unlock()
	extensionRegistry.routes[key] = append(extensionRegistry.routes[key], registrar)
}

// RegisteredRouteRegistrars returns route registrars for a service.
func RegisteredRouteRegistrars(service ServiceName) []RouteRegistrar {
	keys := serviceKeys(service)
	extensionRegistry.RLock()
	defer extensionRegistry.RUnlock()

	out := make([]RouteRegistrar, 0)
	for _, key := range keys {
		out = append(out, extensionRegistry.routes[key]...)
	}
	return out
}

// ApplyRouteRegistrars attaches registered routes to an engine.
func ApplyRouteRegistrars(service ServiceName, engine *gin.Engine) {
	if engine == nil {
		return
	}
	registerRuntimeInfoRoute(service, engine)
	for _, registrar := range RegisteredRouteRegistrars(service) {
		registrar(engine)
	}
}

func registerRuntimeInfoRoute(service ServiceName, engine *gin.Engine) {
	engine.GET("/runtime/info", func(c *gin.Context) {
		c.JSON(http.StatusOK, license.Snapshot(string(normalizeServiceKey(service))))
	})
}

func normalizeServiceKey(service ServiceName) string {
	key := strings.ToLower(strings.TrimSpace(string(service)))
	if key == "" {
		return globalServiceKey
	}
	return key
}

func serviceKeys(service ServiceName) []string {
	key := normalizeServiceKey(service)
	if key == globalServiceKey {
		return []string{globalServiceKey}
	}
	return []string{globalServiceKey, key}
}
