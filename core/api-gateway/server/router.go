package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	v1 "github.com/kageos/kageos/core/api-gateway/api/v1"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/pprof"
	"github.com/kageos/kageos/pkg/serverx"
)

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// 健康检查
	s.httpServer.GET("/health", s.healthHandler)

	// 注册 pprof 路由（性能分析）
	if s.cfg.IsPprofEnabled() {
		pprof.RegisterPprofRoutes(s.httpServer)
	}

	// 配置接口（本地处理）
	configHandler := v1.NewConfig()
	s.httpServer.GET("/api/v1/config", configHandler.GetConfig)

	// 网关自己的 Swagger 文档（直接服务，不通过代理）
	// 注意：必须在 setupSwaggerRoutes 之前注册，避免路由冲突
	s.httpServer.GET("/swagger/gateway/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Swagger 文档聚合
	s.setupSwaggerRoutes()

	// 从配置文件读取路由并注册代理
	cfg := s.cfg
	routes := cfg.Routes

	// 检查路由配置
	if len(routes) == 0 {
		logger.Errorf(s.ctx, "[Router] No routes configured in config file")
		return
	}

	// 分离路由：具体的路径和通用路径
	var specificRoutes []config.RouteConfig // 具体路径（如 /api/v1/storage）
	var catchAllRoutes []config.RouteConfig // 通用路径（如 /api）

	for _, route := range routes {
		if route.Path == "" {
			logger.Warnf(s.ctx, "[Router] Invalid route config: path is empty")
			continue
		}

		if len(route.Targets) == 0 {
			logger.Warnf(s.ctx, "[Router] Invalid route config: path=%s, no targets configured", route.Path)
			continue
		}

		// 验证至少有一个有效的 target URL
		hasValidTarget := false
		for i, target := range route.Targets {
			if target.URL == "" {
				logger.Warnf(s.ctx, "[Router] Invalid route config: path=%s, target[%d] url is empty", route.Path, i)
			} else {
				hasValidTarget = true
			}
		}
		if !hasValidTarget {
			logger.Warnf(s.ctx, "[Router] Skipping route: path=%s, no valid targets", route.Path)
			continue
		}

		// 判断是否为通用路由（catch-all）：检查是否有其他更具体的路径以当前路径为前缀
		// 例如：/api 是通用路由（因为 /api/v1/storage 以它开头）
		//      /api/v1/storage 是具体路由（因为它是具体的子路径）
		isCatchAll := false
		for _, otherRoute := range routes {
			if otherRoute.Path != route.Path &&
				len(otherRoute.Path) > len(route.Path) &&
				strings.HasPrefix(otherRoute.Path, route.Path) {
				// 检查是否是子路径（如 /api 是 /api/v1/storage 的前缀）
				if otherRoute.Path[len(route.Path)] == '/' {
					isCatchAll = true
					break
				}
			}
		}

		if isCatchAll {
			catchAllRoutes = append(catchAllRoutes, route)
		} else {
			specificRoutes = append(specificRoutes, route)
		}
	}

	// 先注册具体路径
	for _, route := range specificRoutes {
		proxy := s.createRouteProxy(&route)
		pathPattern := route.Path + "/*path"
		s.httpServer.Any(pathPattern, proxy)
		logger.Infof(s.ctx, "[Router] Registered route: %s -> %s (timeout: %ds)",
			pathPattern, route.Targets[0].URL, route.Timeout)
	}

	// 处理通用路径（兜底）- 使用 NoRoute 避免路由冲突
	if len(catchAllRoutes) > 0 {
		// 只支持一个通用路由作为兜底
		catchAllRoute := catchAllRoutes[0]
		if len(catchAllRoutes) > 1 {
			logger.Warnf(s.ctx, "[Router] Multiple catch-all routes found, only using first: %s", catchAllRoute.Path)
			for i := 1; i < len(catchAllRoutes); i++ {
				logger.Warnf(s.ctx, "[Router] Ignored catch-all route: %s", catchAllRoutes[i].Path)
			}
		}
		proxy := s.createRouteProxy(&catchAllRoute)

		// 保存通用路由信息，供 NoRoute 使用
		catchAllPrefix := catchAllRoute.Path

		// 使用 NoRoute 处理所有未匹配的请求
		s.httpServer.NoRoute(func(c *gin.Context) {
			// 只处理以 catchAllPrefix 开头的请求（如 /api）
			requestPath := c.Request.URL.Path
			if len(requestPath) >= len(catchAllPrefix) &&
				requestPath[:len(catchAllPrefix)] == catchAllPrefix {
				// 匹配成功，执行代理
				proxy(c)
			} else {
				// 不匹配，返回 404
				c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
			}
		})

		logger.Infof(s.ctx, "[Router] Registered catch-all route: %s -> %s (timeout: %ds)",
			catchAllRoute.Path+"/*", catchAllRoute.Targets[0].URL, catchAllRoute.Timeout)
	}

	serverx.ApplyRouteRegistrars(serverx.ServiceAPIGateway, s.httpServer)
}
