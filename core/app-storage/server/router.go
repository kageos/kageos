package server

import (
	v1 "github.com/kageos/kageos/core/app-storage/api/v1"
	middleware2 "github.com/kageos/kageos/pkg/middleware"
	"github.com/kageos/kageos/pkg/pprof"
	"github.com/kageos/kageos/pkg/serverx"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// 健康检查
	s.httpServer.GET("/health", s.healthHandler)

	// 注册 pprof 路由（性能分析）
	if s.cfg.IsPprofEnabled() {
		pprof.RegisterPprofRoutes(s.httpServer)
	}

	// Swagger 文档路由
	s.httpServer.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Storage 路由组（统一使用 /storage/api/v1 开头，方便网关代理）
	storage := s.httpServer.Group("/storage")

	// API v1 路由组
	apiV1 := storage.Group("/api/v1")

	public := apiV1.Group("/public")
	storageHandler := v1.NewStorage(s.storageService)
	public.POST("/share/:share_id/upload_token", storageHandler.PublicShareGetUploadToken)
	public.POST("/share/:share_id/upload_complete", storageHandler.PublicShareUploadComplete)
	public.POST("/share/:share_id/batch_upload_complete", storageHandler.PublicShareBatchUploadComplete)
	public.POST("/share/:share_id/files/resolve", storageHandler.PublicShareResolveFileRefs)

	// 存储相关路由（需要JWT验证）
	storageGroup := apiV1
	storageGroup.Use(middleware2.JWTAuth()) // 存储管理需要JWT认证

	// 上传相关
	storageGroup.POST("/upload_token", storageHandler.GetUploadToken)
	storageGroup.POST("/batch_upload_token", storageHandler.BatchGetUploadToken)    // ✨ 批量获取上传凭证
	storageGroup.POST("/upload_complete", storageHandler.UploadComplete)            // 上传完成通知
	storageGroup.POST("/batch_upload_complete", storageHandler.BatchUploadComplete) // ✨ 批量上传完成通知
	storageGroup.POST("/files/resolve", storageHandler.ResolveFileRefs)             // 批量解析 files ref，返回元数据和直连 URL
	storageGroup.POST("/files/description", storageHandler.UpdateFileDescription)   // 更新文件描述元数据

	// 文件操作（key 包含斜杠，使用 *key 匹配）
	storageGroup.GET("/download/*key", storageHandler.DownloadFile)
	storageGroup.GET("/info/*key", storageHandler.GetFileInfo) // ✅ info 在前，避免 catch-all 冲突
	storageGroup.DELETE("/files/*key", storageHandler.DeleteFile)

	// 批量操作（按函数路径）
	storageGroup.GET("/files", storageHandler.ListFiles)                   // 列举文件
	storageGroup.GET("/stats", storageHandler.GetStorageStats)             // 存储统计
	storageGroup.POST("/batch_delete", storageHandler.DeleteFilesByRouter) // 批量删除

	serverx.ApplyRouteRegistrars(serverx.ServiceAppStorage, s.httpServer)
}
