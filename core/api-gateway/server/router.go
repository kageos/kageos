package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	v1 "github.com/ai-agent-os/ai-agent-os/core/api-gateway/api/v1"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// 健康检查
	s.httpServer.GET("/health", s.healthHandler)

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
}

// createRouteProxy 创建路由代理（统一入口）
func (s *Server) createRouteProxy(route *config.RouteConfig) gin.HandlerFunc {
	// 创建代理（支持负载均衡）
	if len(route.Targets) == 1 {
		// 单个目标，使用简单代理
		return s.createProxy(route.Targets[0].URL, route.Timeout, route)
	} else {
		// 多个目标，使用负载均衡代理
		return s.createLoadBalanceProxy(route)
	}
}

// createProxy 创建反向代理（单个目标）
func (s *Server) createProxy(targetURL string, timeout int, route *config.RouteConfig) gin.HandlerFunc {
	// 解析目标 URL
	target, err := url.Parse(targetURL)
	if err != nil {
		logger.Errorf(s.ctx, "[Proxy] Invalid target URL: %s, error: %v", targetURL, err)
		return func(c *gin.Context) {
			traceID := c.GetString("trace-id")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":    "Invalid gateway configuration",
				"trace_id": traceID,
				"details":  fmt.Sprintf("Invalid target URL: %s", targetURL),
			})
		}
	}

	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(target)

	// 从配置读取超时时间（使用统一方法）
	timeout = s.getTimeout(timeout)

	// 使用共享 Transport（提高性能）
	// 注意：ResponseHeaderTimeout 需要根据每个路由的超时时间动态设置
	// 由于 Transport 是共享的，我们使用配置的超时时间，但实际超时由 Context 控制
	proxy.Transport = s.sharedTransport

	// 自定义请求修改（支持路径重写）
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host

		// 路径重写：如果配置了 rewrite_path，替换路径前缀
		if route != nil && route.RewritePath != "" {
			originalPath := req.URL.Path
			routePath := route.Path

			// 如果请求路径以路由路径开头，进行重写
			if strings.HasPrefix(originalPath, routePath) {
				// 提取路径的后半部分（去掉路由前缀）
				suffix := originalPath[len(routePath):]
				// 拼接新的路径
				rewritePath := route.RewritePath
				if !strings.HasSuffix(rewritePath, "/") && suffix != "" && !strings.HasPrefix(suffix, "/") {
					rewritePath += "/"
				}
				req.URL.Path = rewritePath + suffix

				logger.Debugf(s.ctx, "[Proxy] Path rewrite: %s -> %s (route: %s, rewrite: %s)",
					originalPath, req.URL.Path, routePath, route.RewritePath)
			}
		}
	}

	// 移除后端服务设置的 CORS 头，避免与网关的 CORS 中间件重复
	// 网关的 CORS 中间件会统一处理所有响应
	proxy.ModifyResponse = func(resp *http.Response) error {
		// 移除后端服务设置的 CORS 头，避免重复
		resp.Header.Del("Access-Control-Allow-Origin")
		resp.Header.Del("Access-Control-Allow-Methods")
		resp.Header.Del("Access-Control-Allow-Headers")
		resp.Header.Del("Access-Control-Allow-Credentials")
		resp.Header.Del("Access-Control-Expose-Headers")
		// 网关的 CORS 中间件会在响应返回前统一添加 CORS 头
		return nil
	}

	// 错误处理
	// 注意：不需要在 ErrorHandler 中设置 CORS 头
	// 因为网关的 CORS 中间件会在所有响应（包括错误响应）中添加 CORS 头
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Errorf(s.ctx, "[Proxy] Proxy error to %s: %v", targetURL, err)
		http.Error(w, fmt.Sprintf("Gateway error: %v", err), http.StatusBadGateway)
	}

	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// createLoadBalanceProxy 创建负载均衡代理
// 当前实现：使用第一个目标（负载均衡功能待实现）
// 未来实现：
//   - 根据 strategy 选择负载均衡算法（round_robin, weighted, least_connections, ip_hash）
//   - 健康检查（如果启用）
//   - 失败重试和故障转移
func (s *Server) createLoadBalanceProxy(route *config.RouteConfig) gin.HandlerFunc {
	logger.Warnf(s.ctx, "[LoadBalance] Load balance not implemented yet, using first target: %s", route.Targets[0].URL)
	timeout := s.getTimeout(route.Timeout)
	return s.createProxy(route.Targets[0].URL, timeout, route)
}

// setupSwaggerRoutes 设置 Swagger 文档路由（聚合所有服务）
func (s *Server) setupSwaggerRoutes() {
	cfg := s.cfg

	// Swagger 聚合首页（列出所有服务的文档链接）
	s.httpServer.GET("/swagger", s.swaggerIndexHandler)
	s.httpServer.GET("/swagger/index.html", s.swaggerIndexHandler)

	// 根据配置路由动态创建 Swagger 代理
	// 从路由配置中提取服务地址，创建对应的 Swagger 代理
	serviceMap := make(map[string]string) // service -> target

	// 解析路由配置，提取服务名称和目标地址（使用第一个 target）
	// 注意：必须显式配置 service_name，不支持自动提取
	for _, route := range cfg.Routes {
		if len(route.Targets) == 0 {
			continue
		}
		// 必须配置 service_name，否则跳过
		if route.ServiceName == "" {
			logger.Warnf(s.ctx, "[Swagger] Route %s missing service_name, skipping Swagger proxy", route.Path)
			continue
		}
		serviceMap[route.ServiceName] = route.Targets[0].URL
	}

	// 如果没有配置路由，无法创建 Swagger 代理
	if len(serviceMap) == 0 {
		logger.Warnf(s.ctx, "[Swagger] No routes configured, cannot setup Swagger proxy")
		return
	}

	// 为每个服务创建 Swagger 代理路由
	for serviceName, target := range serviceMap {
		swaggerProxy := s.createSwaggerProxy(target)
		swaggerPath := fmt.Sprintf("/swagger/%s/*path", serviceName)
		s.httpServer.Any(swaggerPath, swaggerProxy)
		logger.Infof(s.ctx, "[Swagger] Registered: %s -> %s/swagger", swaggerPath, target)
	}
}

// createSwaggerProxy 创建 Swagger 文档代理
func (s *Server) createSwaggerProxy(targetURL string) gin.HandlerFunc {
	target, err := url.Parse(targetURL)
	if err != nil {
		logger.Errorf(s.ctx, "[Swagger] Invalid target URL: %s", targetURL)
		return func(c *gin.Context) {
			traceID := c.GetString("trace-id")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":    "Invalid Swagger target",
				"trace_id": traceID,
				"details":  fmt.Sprintf("Invalid target URL: %s", targetURL),
			})
		}
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// 使用共享 Transport
	proxy.Transport = s.sharedTransport

	// 自定义路径处理：将 /swagger/serviceName/* 转换为目标服务的 /swagger/*
	// 例如：/swagger/server/index.html -> /swagger/index.html
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host

		// 移除服务名称前缀：/swagger/serviceName/xxx -> /swagger/xxx
		path := req.URL.Path
		const swaggerPrefix = "/swagger/"
		if strings.HasPrefix(path, swaggerPrefix) {
			// 找到服务名称后的第一个 /（即第二个 /）
			// 例如：/swagger/server/index.html -> /swagger/index.html
			if idx := strings.Index(path[len(swaggerPrefix):], "/"); idx >= 0 {
				req.URL.Path = swaggerPrefix + path[len(swaggerPrefix)+idx+1:]
			} else {
				// 如果没有后续路径，直接使用 /swagger
				req.URL.Path = "/swagger"
			}
		}
	}

	// 移除后端服务设置的 CORS 头，避免与网关的 CORS 中间件重复
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("Access-Control-Allow-Origin")
		resp.Header.Del("Access-Control-Allow-Methods")
		resp.Header.Del("Access-Control-Allow-Headers")
		resp.Header.Del("Access-Control-Allow-Credentials")
		resp.Header.Del("Access-Control-Expose-Headers")
		return nil
	}

	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// swaggerIndexHandler Swagger 聚合首页
func (s *Server) swaggerIndexHandler(c *gin.Context) {
	cfg := s.cfg
	gatewayURL := fmt.Sprintf("http://%s", c.Request.Host)

	services := []map[string]string{}

	// 首先添加网关自己的文档
	services = append(services, map[string]string{
		"name":    "gateway",
		"path":    "/",
		"swagger": fmt.Sprintf("%s/swagger/gateway/index.html", gatewayURL),
		"target":  "localhost:9090",
	})

	// 从路由配置中提取服务（必须显式配置 service_name）
	for _, route := range cfg.Routes {
		if len(route.Targets) == 0 {
			continue
		}
		// 必须配置 service_name，否则跳过
		if route.ServiceName == "" {
			logger.Warnf(s.ctx, "[Swagger] Route %s missing service_name, skipping", route.Path)
			continue
		}
		// 显示所有 targets（如果是负载均衡）
		targetStr := route.Targets[0].URL
		if len(route.Targets) > 1 {
			targetStr = fmt.Sprintf("%d targets", len(route.Targets))
		}
		services = append(services, map[string]string{
			"name":    route.ServiceName,
			"path":    route.Path,
			"swagger": fmt.Sprintf("%s/swagger/%s/index.html", gatewayURL, route.ServiceName),
			"target":  targetStr,
		})
		logger.Infof(s.ctx, "[Swagger] Registered service: %s (path: %s)", route.ServiceName, route.Path)
	}

	// 返回 HTML 页面
	html := s.generateSwaggerIndexHTML(services)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// SwaggerURL Swagger URL 配置
type SwaggerURL struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

// generateSwaggerIndexHTML 生成 Swagger 聚合首页 HTML（使用 Swagger UI 的 Select a definition 功能）
func (s *Server) generateSwaggerIndexHTML(services []map[string]string) string {
	// 构建 Swagger JSON URLs 数组（使用 encoding/json 安全序列化）
	urls := make([]SwaggerURL, 0, len(services))
	for _, service := range services {
		// 使用服务的 swagger.json 路径（gin-swagger 默认路径是 /swagger/doc.json）
		// 从 /swagger/serviceName/index.html 提取出 /swagger/serviceName，然后拼接 /doc.json
		swaggerPath := service["swagger"]
		if len(swaggerPath) < len("/index.html") {
			logger.Warnf(s.ctx, "[Swagger] Invalid swagger path: %s", swaggerPath)
			continue
		}
		swaggerBasePath := swaggerPath[:len(swaggerPath)-len("/index.html")]
		swaggerJSONURL := fmt.Sprintf("%s/doc.json", swaggerBasePath)
		urls = append(urls, SwaggerURL{
			URL:  swaggerJSONURL,
			Name: service["name"],
		})
	}

	// 使用 encoding/json 安全序列化
	urlsJSONBytes, err := json.Marshal(urls)
	if err != nil {
		logger.Errorf(s.ctx, "[Swagger] Failed to marshal URLs: %v", err)
		// 降级处理：返回空数组
		urlsJSONBytes = []byte("[]")
	}
	urlsJSON := string(urlsJSONBytes)

	html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>API Gateway - Swagger 文档聚合</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.9.0/swagger-ui.css" />
    <style>
        html {
            box-sizing: border-box;
            overflow: -moz-scrollbars-vertical;
            overflow-y: scroll;
        }
        *, *:before, *:after {
            box-sizing: inherit;
        }
        body {
            margin:0;
            background: #fafafa;
        }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.9.0/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5.9.0/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const urls = ` + urlsJSON + `;
            
            // 使用 Swagger UI 的 urls 配置，支持 Select a definition 下拉选择
            const ui = SwaggerUIBundle({
                urls: urls,
                "urls.primaryName": urls.length > 0 ? urls[0].name : "",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                validatorUrl: null  // 禁用验证器，避免加载错误
            });
        }
    </script>
</body>
</html>`
	return html
}
