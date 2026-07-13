package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/kageos/kageos/pkg/logger"
)

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
			remainingPath := path[len(swaggerPrefix):]
			if idx := strings.Index(remainingPath, "/"); idx >= 0 {
				// 提取服务名称后的路径部分
				newPath := swaggerPrefix + remainingPath[idx+1:]
				logger.Infof(s.ctx, "[Swagger] Path rewrite: %s -> %s", path, newPath)
				req.URL.Path = newPath
			} else {
				// 如果没有后续路径，直接使用 /swagger
				logger.Infof(s.ctx, "[Swagger] Path rewrite: %s -> /swagger", path)
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

	// 添加错误处理
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		logger.Errorf(s.ctx, "[Swagger] Proxy error for %s -> %s: %v", req.URL.Path, targetURL, err)
		traceID := req.Header.Get("trace-id")
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(rw).Encode(gin.H{
			"error":    "Swagger service unavailable",
			"trace_id": traceID,
			"details":  fmt.Sprintf("Failed to proxy to %s: %v", targetURL, err),
		})
	}

	return func(c *gin.Context) {
		// 记录请求日志
		originalPath := c.Request.URL.Path
		logger.Infof(s.ctx, "[Swagger] Proxying request: %s -> %s", originalPath, targetURL)
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
