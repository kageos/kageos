package server

import (
	"context"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/middleware"
	"github.com/ai-agent-os/ai-agent-os/pkg/pprof"
	"github.com/ai-agent-os/ai-agent-os/pkg/serverx"
)

// initRouter 初始化路由
func (s *Server) initRouter(ctx context.Context) error {
	logger.Infof(ctx, "[Control Service] Initializing router...")

	s.httpServer = serverx.NewGin(
		serverx.WithDebug(s.cfg.IsDebug()),
		serverx.WithRecovery(),
		serverx.WithLogger(),
	)

	s.httpServer.GET("/health", s.healthHandler)

	// 注册 pprof 路由（性能分析）
	if s.cfg.IsPprofEnabled() {
		pprof.RegisterPprofRoutes(s.httpServer)
	}

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
