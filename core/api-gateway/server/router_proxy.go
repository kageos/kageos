package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/response"
)

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
	// ⭐ 对于 SSE 流式接口，需要更长的 ResponseHeaderTimeout
	// 由于 Transport 是共享的，我们为 SSE 请求创建单独的 Transport
	// 但为了性能，我们仍然使用共享 Transport，超时由 Context 控制
	proxy.Transport = s.sharedTransport

	// 自定义请求修改（支持路径重写和TraceId传递）。
	// ReverseProxy 会转发剩余 header；可信身份头已在 ServeHTTP 前由
	// prepareProxyIdentity 统一清洗并从可验证凭据重建。
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		// 保存原始 Host；若上游已传 X-Forwarded-Host（如 Vite 代理传的浏览器 Host），保留不覆盖
		originalHost := req.Host
		existingForwardedHost := req.Header.Get("X-Forwarded-Host")
		originalDirector(req)
		if existingForwardedHost != "" {
			// 上游（如 Vite）已传浏览器真实 Host，保留给 app-storage 做预签名
			req.Header.Set("X-Forwarded-Host", existingForwardedHost)
		} else if originalHost != "" && originalHost != target.Host {
			req.Header.Set("X-Forwarded-Host", originalHost)
			if idx := strings.Index(originalHost, ":"); idx >= 0 && idx < len(originalHost)-1 {
				req.Header.Set("X-Forwarded-Port", originalHost[idx+1:])
			}
			if strings.Contains(req.URL.Path, "/storage/") {
				logger.Infof(s.ctx, "[Proxy] X-Forwarded-Host set for presign: originalHost=%q, target.Host=%q, path=%s", originalHost, target.Host, req.URL.Path)
			}
		}
		req.Host = target.Host

		// ✨ 传递 TraceId 到后端服务
		// 如果请求 header 中已有 TraceId（由 gin handler 设置），直接使用
		// 否则保持原样（可能客户端已提供）
		if traceId := req.Header.Get(contextx.TraceIdHeader); traceId == "" {
			// 如果 header 中没有，说明需要从其他地方获取（这种情况不应该发生，因为 gin handler 已设置）
			logger.Debugf(s.ctx, "[Proxy] TraceId not found in request header")
		}

		// ⭐ 注意：JWT Token 解析和用户信息设置已移至 gin handler 中（在调用 proxy.ServeHTTP 之前）
		// 这样可以确保 header 被正确传递，就像 TraceId 一样

		// 注意：X-Token 和其他请求头会被 httputil.ReverseProxy 自动转发，无需手动处理

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

		// A verified message-server workspace action has no end-user JWT. Once
		// target host and path rewriting are final, mint a new gateway->Agent
		// signature over the exact backend request. Ordinary browser/OpenAPI
		// requests retain their own credentials and are not signed here.
		if route != nil && route.ServiceName == "agent" {
			s.signVerifiedAgentBackendRequest(req)
		}
		if route != nil && route.ServiceName == "timer" {
			s.signVerifiedTimerBackendRequest(req)
		}
		// General Agent delegation may target workspace, HR, storage, or future
		// explicitly allowlisted routes. The private verified context marker makes
		// this a no-op for ordinary browser/OpenAPI traffic.
		s.signVerifiedDelegatedBackendRequest(req)
	}

	// 移除后端服务设置的 CORS 头，避免与网关的 CORS 中间件重复
	// 网关的 CORS 中间件会统一处理所有响应
	proxy.ModifyResponse = func(resp *http.Response) error {
		// ⭐ 新增：检查 token 是否在黑名单中，如果是则返回 401
		if resp.Request.Header.Get("X-Token-Blacklisted") == "true" {
			logger.Warnf(s.ctx, "[Proxy] Token is blacklisted, returning 401")
			resp.StatusCode = http.StatusUnauthorized
			resp.Status = "401 Unauthorized"
			// 设置响应头
			resp.Header = make(http.Header)
			resp.Header.Set("Content-Type", "application/json")
			// 返回 JSON 错误响应（使用定义的常量，前端会根据 code 跳转到登录页）
			errorResp := response.GetTokenBlacklistedResponse()
			errorBody, _ := json.Marshal(errorResp)
			resp.Body = io.NopCloser(bytes.NewReader(errorBody))
			resp.ContentLength = int64(len(errorBody))
			return nil
		}

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
		proxyStart := time.Now()

		// ✨ 将 TraceId 从 gin context 设置到请求 header，供后端服务使用
		// WithTraceId 中间件已经将 TraceId 设置到 gin context 中（使用常量 TraceIdHeader）
		traceId := c.GetString(contextx.TraceIdHeader) // ⭐ 使用常量 TraceIdHeader
		if traceId != "" {
			// 设置到请求 header，这样 proxy.Director 就能读取并传递给后端
			c.Request.Header.Set(contextx.TraceIdHeader, traceId)
		}

		// The gateway is the external trust boundary: client-supplied identity,
		// role, department and provenance headers are always cleared first.
		// Only a verified JWT/anonymous mechanism or a short-lived signed core
		// request may rebuild them.
		if err := s.prepareProxyIdentity(c); err != nil {
			switch {
			case errors.Is(err, errProxyTokenBlacklisted):
				logger.Warnf(s.ctx, "[Proxy] Token is blacklisted, rejecting request")
				c.JSON(http.StatusUnauthorized, response.GetTokenBlacklistedResponse())
			default:
				logger.Warnf(s.ctx, "[Proxy] Rejected untrusted internal identity metadata: path=%s error=%v", c.Request.URL.Path, err)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid internal request authentication"})
			}
			c.Abort()
			return
		}

		// ✅ 创建带超时的 Context，避免高并发时请求堆积
		// ⭐ 特殊处理：对于 SSE 流式接口（如 /chat/stream），使用更长的超时时间或不设置超时
		var ctx context.Context
		var cancel context.CancelFunc

		// 检测是否是 SSE 流式接口（通过路径判断）
		isStreamingRequest := strings.Contains(c.Request.URL.Path, "/stream") ||
			strings.Contains(c.Request.URL.Path, "/chat/stream")

		if isStreamingRequest {
			// SSE 流式接口：使用更长的超时时间（30分钟）或不设置超时
			// 使用 30 分钟超时，避免无限等待，但足够长以支持长时间流式响应
			ctx, cancel = context.WithTimeout(c.Request.Context(), 30*time.Minute)
			logger.Debugf(s.ctx, "[Proxy] SSE streaming request detected, using extended timeout (30min): %s", c.Request.URL.Path)
		} else {
			// 普通请求：使用配置的超时时间
			ctx, cancel = context.WithTimeout(c.Request.Context(), time.Duration(timeout)*time.Second)
		}
		defer cancel()

		logger.Infof(s.ctx, "[Proxy] start: traceId=%s, method=%s, path=%s, target=%s, timeout=%ds",
			traceId, c.Request.Method, c.Request.URL.Path, targetURL, timeout)

		// ✅ 使用带超时的 Context 创建新请求
		// ⭐ 注意：WithContext 会创建一个新请求，但 Header 是共享的（引用类型）
		// 所以之前设置的 header（TraceId、X-Request-User 等）会被正确传递到后端服务
		req := c.Request.WithContext(ctx)
		proxy.ServeHTTP(c.Writer, req)

		logger.Infof(s.ctx, "[Proxy] done: traceId=%s, path=%s, status=%d, elapsed=%s",
			traceId, c.Request.URL.Path, c.Writer.Status(), time.Since(proxyStart).Truncate(time.Millisecond))
	}
}

// createLoadBalanceProxy 处理多 target 路由。
// 当前并未实现真正的负载均衡，只回退使用第一个 target。
func (s *Server) createLoadBalanceProxy(route *config.RouteConfig) gin.HandlerFunc {
	logger.Warnf(s.ctx, "[LoadBalance] Load balance not implemented yet, using first target: %s", route.Targets[0].URL)
	timeout := s.getTimeout(route.Timeout)
	return s.createProxy(route.Targets[0].URL, timeout, route)
}
