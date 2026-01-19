package server

import (
	"context"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/middleware"
	"github.com/ai-agent-os/ai-agent-os/pkg/pprof"
	"github.com/gin-gonic/gin"
)

// initRouter 初始化路由
func (s *Server) initRouter(ctx context.Context) error {
	logger.Infof(ctx, "[Control Service] Initializing router...")

	gin.SetMode(gin.ReleaseMode)
	s.httpServer = gin.New()
	s.httpServer.Use(gin.Recovery())
	s.httpServer.Use(gin.Logger())

	// 注册 pprof 路由（性能分析）
	pprof.RegisterPprofRoutes(s.httpServer)

	// Control 路由组（统一使用 /control/api/v1 开头，方便网关代理）
	control := s.httpServer.Group("/control")

	// API v1 路由组
	// ⭐ 添加用户信息中间件，从 header 中读取用户信息（网关已解析 token 并设置到 header）
	apiV1 := control.Group("/api/v1")
	apiV1.Use(middleware.WithUserInfo())

	// License 相关路由
	license := apiV1.Group("/license")
	{
		license.GET("/status", s.licenseAPI.GetStatus)
		license.POST("/activate", s.licenseAPI.Activate)
		license.POST("/deactivate", s.licenseAPI.Deactivate)
	}

	return nil
}
