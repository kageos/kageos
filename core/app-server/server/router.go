package server

import (
	v1 "github.com/ai-agent-os/ai-agent-os/core/app-server/api/v1"
	middleware2 "github.com/ai-agent-os/ai-agent-os/pkg/middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// 健康检查
	s.httpServer.GET("/health", s.healthHandler)

	// Swagger 文档路由
	s.httpServer.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 路由组
	apiV1 := s.httpServer.Group("/api/v1")

	// 认证相关路由（不需要JWT验证）
	auth := apiV1.Group("/auth")
	authHandler := v1.NewAuth()
	auth.POST("/send_email_code", authHandler.SendEmailCode)
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.RefreshToken)
	auth.POST("/logout", authHandler.Logout)

	// 应用管理路由（需要JWT验证）
	app := apiV1.Group("/app")
	app.Use(middleware2.JWTAuth()) // 应用管理需要JWT认证
	appHandler := v1.NewApp(s.appService)
	app.POST("/create", appHandler.CreateApp)
	app.POST("/update/:app", appHandler.UpdateApp)
	app.DELETE("/delete/:app", appHandler.DeleteApp)
	// 支持所有 HTTP 方法的请求应用接口
	app.Any("/request/:app/*router", appHandler.RequestApp)
}
