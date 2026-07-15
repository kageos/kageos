package server

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSetupRoutesRegistersCanonicalRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	s := &Server{httpServer: gin.New()}
	s.setupRoutes()

	routes := make(map[string]struct{})
	for _, route := range s.httpServer.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	expectedRoutes := []string{
		"GET /workspace/api/v1/app/detail",
		"GET /workspace/api/v1/app/tree",
		"GET /workspace/api/v1/service_tree/detail",
		"POST /workspace/api/v1/service_tree/batch_detail",
		"GET /workspace/api/v1/function/info/:func-type/*full-code-path",
	}
	for _, route := range expectedRoutes {
		if _, ok := routes[route]; !ok {
			t.Fatalf("expected route %s to be registered", route)
		}
	}

	removedRoutes := []string{
		"GET /workspace/api/v1/app/detail/:app",
		"GET /workspace/api/v1/app/:user/:app/tree",
		"POST /workspace/api/v1/app/update/:user/:app",
		"PUT /workspace/api/v1/app/workspace/:user/:app",
		"DELETE /workspace/api/v1/app/delete/:app",
		"GET /workspace/api/v1/service_tree/package_info",
		"GET /workspace/api/v1/function/list",
		"POST /workspace/api/v1/service_tree",
		"GET /workspace/api/v1/service_tree",
		"PUT /workspace/api/v1/service_tree",
		"DELETE /workspace/api/v1/service_tree",
		"POST /workspace/api/v1/permission/request/create",
		"POST /workspace/api/v1/role/user",
		"POST /workspace/api/v1/role/department",
	}
	for _, route := range removedRoutes {
		if _, ok := routes[route]; ok {
			t.Fatalf("did not expect removed route %s to be registered", route)
		}
	}
}
