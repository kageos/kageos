package server

import (
	middleware2 "github.com/ai-agent-os/ai-agent-os/pkg/middleware"
	v1 "github.com/ai-agent-os/hub/backend/api/v1"
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
	hub := s.httpServer.Group("/hub")
	apiV1 := hub.Group("/api/v1")

	// TODO: 认证相关路由（调用 OS API 进行认证）
	// auth := apiV1.Group("/auth")
	// authHandler := v1.NewAuth(s.authService)
	// auth.POST("/login", authHandler.Login)
	// auth.POST("/refresh", authHandler.RefreshToken)
	// auth.POST("/logout", authHandler.Logout)

	// Hub 目录管理路由
	hubDirectory := apiV1.Group("/directories")
	hubDirectoryHandler := v1.NewDirectory(s.hubDirectoryService)

	// 公开接口（不需要认证）
	hubDirectory.GET("", hubDirectoryHandler.GetDirectoryList)          // 获取目录列表
	hubDirectory.GET("/detail", hubDirectoryHandler.GetDirectoryDetail) // 获取目录详情 ?hub_directory_id=xxx

	// 需要认证的接口
	hubDirectoryAuth := hubDirectory.Group("")
	hubDirectoryAuth.Use(middleware2.JWTAuth())
	hubDirectoryAuth.POST("/publish", hubDirectoryHandler.PublishDirectory) // 发布目录
	hubDirectoryAuth.PUT("/update", hubDirectoryHandler.UpdateDirectory)    // 更新目录（push）

	// TODO: 服务费支付路由（需要JWT验证）
	// payment := apiV1.Group("/payments")
	// payment.Use(middleware2.JWTAuth())
	// paymentHandler := v1.NewPayment(s.paymentService)
	// payment.POST("/:hub_app_id", paymentHandler.CreatePayment)
	// payment.POST("/:payment_id/callback", paymentHandler.PaymentCallback)
}
