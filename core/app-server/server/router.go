package server

import (
	v1 "github.com/ai-agent-os/ai-agent-os/core/app-server/api/v1"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
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

	// API v1 路由组
	apiV1 := s.httpServer.Group("/api/v1")

	// Workspace 路由组（统一使用 /api/v1/workspace 开头，方便网关代理）
	workspace := apiV1.Group("/workspace")

	// 认证相关路由（不需要JWT验证）
	auth := workspace.Group("/auth")
	authHandler := v1.NewAuth(s.authService, s.emailService)
	auth.POST("/send_email_code", authHandler.SendEmailCode)
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.RefreshToken)
	auth.POST("/logout", authHandler.Logout)

	// 应用管理路由（需要JWT验证）
	app := workspace.Group("/app")
	app.Use(middleware2.JWTAuth()) // 应用管理需要JWT认证
	appHandler := v1.NewApp(s.appService)
	app.GET("/list", appHandler.GetApps)
	app.POST("/create", appHandler.CreateApp)
	app.POST("/update/:app", appHandler.UpdateApp)
	app.DELETE("/delete/:app", appHandler.DeleteApp)
	// 支持所有 HTTP 方法的请求应用接口
	request := workspace.Group("/run")
	request.Use(middleware2.JWTAuth())
	request.Any("/*router", appHandler.RequestApp)

	callback := workspace.Group("/callback")
	callback.Use(middleware2.JWTAuth())
	callback.POST("/*router", appHandler.CallbackApp)

	// 服务目录管理路由（需要JWT验证）
	serviceTree := workspace.Group("/service_tree")
	serviceTree.Use(middleware2.JWTAuth()) // 服务目录管理需要JWT认证
	serviceTreeHandler := v1.NewServiceTree(s.serviceTreeService)
	serviceTree.POST("", serviceTreeHandler.CreateServiceTree)
	serviceTree.GET("", serviceTreeHandler.GetServiceTree)
	serviceTree.PUT("", serviceTreeHandler.UpdateServiceTree)
	serviceTree.DELETE("", serviceTreeHandler.DeleteServiceTree)
	serviceTree.POST("/copy", serviceTreeHandler.CopyServiceTree)                 // 复制服务目录
	serviceTree.POST("/publish_to_hub", serviceTreeHandler.PublishDirectoryToHub) // 发布目录到 Hub

	// 函数管理路由（需要JWT验证）
	function := workspace.Group("/function")
	function.Use(middleware2.JWTAuth()) // 函数管理需要JWT认证
	functionHandler := v1.NewFunction(s.functionService)
	function.GET("/get", functionHandler.GetFunction)
	function.GET("/list", functionHandler.GetFunctionsByApp)

	// 用户管理路由（需要JWT验证）
	user := workspace.Group("/user")
	user.Use(middleware2.JWTAuth()) // 用户管理需要JWT认证
	userHandler := v1.NewUser(s.userService)
	user.GET("/info", userHandler.GetUserInfo)
	user.GET("/query", userHandler.QueryUser)
	user.GET("/search_fuzzy", userHandler.SearchUsersFuzzy)
	user.PUT("/update", userHandler.UpdateUser)

	// 批量获取用户（需要JWT验证）
	users := workspace.Group("/users")
	users.Use(middleware2.JWTAuth())
	users.POST("", userHandler.GetUsersByUsernames)

	// 操作日志路由（需要JWT验证 + 操作日志功能鉴权）
	operateLog := workspace.Group("/operate_log")
	operateLog.Use(middleware2.JWTAuth())                                    // JWT 认证
	operateLog.Use(middleware2.RequireFeature(enterprise.FeatureOperateLog)) // 操作日志功能鉴权（企业版）
	operateLogHandler := v1.NewOperateLog(s.operateLogService)
	operateLog.GET("/table", operateLogHandler.GetTableOperateLogs) // 查询 Table 操作日志
	operateLog.GET("/form", operateLogHandler.GetFormOperateLogs)   // 查询 Form 操作日志

	// 目录更新历史路由（需要JWT验证）
	directoryUpdateHistory := workspace.Group("/directory_update_history")
	directoryUpdateHistory.Use(middleware2.JWTAuth()) // 目录更新历史需要JWT认证
	directoryUpdateHistoryHandler := v1.NewDirectoryUpdateHistory(s.directoryUpdateHistoryService)
	directoryUpdateHistory.GET("/app_version", directoryUpdateHistoryHandler.GetAppVersionUpdateHistory) // 获取应用版本更新历史（App视角）
	directoryUpdateHistory.GET("/directory", directoryUpdateHistoryHandler.GetDirectoryUpdateHistory)    // 获取目录更新历史（目录视角）
}
