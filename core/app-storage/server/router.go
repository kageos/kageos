package server

import (
	v1 "github.com/ai-agent-os/ai-agent-os/core/app-storage/api/v1"
	middleware2 "github.com/ai-agent-os/ai-agent-os/pkg/middleware"
	"github.com/ai-agent-os/ai-agent-os/pkg/pprof"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// 健康检查
	s.httpServer.GET("/health", s.healthHandler)

	// 注册 pprof 路由（性能分析）
	pprof.RegisterPprofRoutes(s.httpServer)

	// Swagger 文档路由
	s.httpServer.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Storage 路由组（统一使用 /storage/api/v1 开头，方便网关代理）
	storage := s.httpServer.Group("/storage")

	// API v1 路由组
	apiV1 := storage.Group("/api/v1")

	// 存储相关路由（需要JWT验证）
	storageGroup := apiV1
	storageGroup.Use(middleware2.JWTAuth()) // 存储管理需要JWT认证
	storageHandler := v1.NewStorage(s.storageService)

	// 上传相关
	storageGroup.POST("/upload_token", storageHandler.GetUploadToken)
	storageGroup.POST("/batch_upload_token", storageHandler.BatchGetUploadToken)    // ✨ 批量获取上传凭证
	storageGroup.POST("/upload_complete", storageHandler.UploadComplete)            // 上传完成通知
	storageGroup.POST("/batch_upload_complete", storageHandler.BatchUploadComplete) // ✨ 批量上传完成通知

	// 文件操作（key 包含斜杠，使用 *key 匹配）
	storageGroup.GET("/download/*key", storageHandler.GetFileURL)
	storageGroup.GET("/info/*key", storageHandler.GetFileInfo) // ✅ info 在前，避免 catch-all 冲突
	storageGroup.DELETE("/files/*key", storageHandler.DeleteFile)

	// 批量操作（按函数路径）
	storageGroup.GET("/files", storageHandler.ListFiles)                   // 列举文件
	storageGroup.GET("/stats", storageHandler.GetStorageStats)             // 存储统计
	storageGroup.POST("/batch_delete", storageHandler.DeleteFilesByRouter) // 批量删除
}
