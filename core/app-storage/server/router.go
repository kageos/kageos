package server

import (
	v1 "github.com/ai-agent-os/ai-agent-os/core/app-storage/api/v1"
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

	// 存储相关路由（需要JWT验证）
	storage := apiV1.Group("/storage")
	storage.Use(middleware2.JWTAuth()) // 存储管理需要JWT认证
	storageHandler := v1.NewStorage(s.storageService)
	
	// 上传相关
	storage.POST("/upload_token", storageHandler.GetUploadToken)
	storage.POST("/upload_complete", storageHandler.UploadComplete)  // 上传完成通知
	
	// 文件操作（key 包含斜杠，使用 *key 匹配）
	storage.GET("/download/*key", storageHandler.GetFileURL)
	storage.GET("/info/*key", storageHandler.GetFileInfo)  // ✅ info 在前，避免 catch-all 冲突
	storage.DELETE("/files/*key", storageHandler.DeleteFile)
	
	// 批量操作（按函数路径）
	storage.GET("/files", storageHandler.ListFiles)           // 列举文件
	storage.GET("/stats", storageHandler.GetStorageStats)     // 存储统计
	storage.POST("/batch_delete", storageHandler.DeleteFilesByRouter) // 批量删除
}

