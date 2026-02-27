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

	// Hub 目录管理路由
	hubDirectory := apiV1.Group("/directories")
	hubDirectoryHandler := v1.NewDirectory(s.hubDirectoryService)

	// 公开接口（不需要认证）
	hubDirectory.GET("", hubDirectoryHandler.GetDirectoryList)
	hubDirectory.GET("/detail", hubDirectoryHandler.GetDirectoryDetail)
	hubDirectory.GET("/versions", hubDirectoryHandler.GetDirectoryVersions)
	hubDirectory.POST("/increment_download", hubDirectoryHandler.IncrementDownloadCount)

	// 发布/推送接口（支持 JWT 或 Pub Key 认证）
	hubDirectoryPub := hubDirectory.Group("")
	hubDirectoryPub.Use(middleware2.JWTOrPubKeyAuth(s.pubKeyService.ValidateKey))
	hubDirectoryPub.POST("/publish", hubDirectoryHandler.PublishDirectory)
	hubDirectoryPub.PUT("/update", hubDirectoryHandler.UpdateDirectory)

	// 其他需要认证的接口（仅 JWT）
	hubDirectoryAuth := hubDirectory.Group("")
	hubDirectoryAuth.Use(middleware2.JWTAuth())
	hubDirectoryAuth.POST("/:id/star", hubDirectoryHandler.Star)
	hubDirectoryAuth.DELETE("/:id/star", hubDirectoryHandler.Unstar)
	hubDirectoryAuth.DELETE("/:id", hubDirectoryHandler.DeleteDirectory)

	// Pub Key 管理路由（需要 JWT 认证，用户在 Hub 前端登录后管理自己的密钥）
	pubKeyGroup := apiV1.Group("/pub_key")
	pubKeyGroup.Use(middleware2.JWTAuth())
	pubKeyHandler := v1.NewPubKey(s.pubKeyService)
	pubKeyGroup.POST("/generate", pubKeyHandler.Generate)
	pubKeyGroup.GET("/list", pubKeyHandler.List)
	pubKeyGroup.DELETE("/:id", pubKeyHandler.Delete)
}
