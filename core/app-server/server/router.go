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

	// Workspace 路由组（统一使用 /workspace/api/v1 开头，方便网关代理）
	workspace := s.httpServer.Group("/workspace")
	apiV1 := workspace.Group("/api/v1")

	// 认证相关路由（不需要JWT验证）
	auth := apiV1.Group("/auth")
	authHandler := v1.NewAuth(s.authService, s.emailService)
	auth.POST("/send_email_code", authHandler.SendEmailCode)
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.RefreshToken)
	auth.POST("/logout", authHandler.Logout)

	// 应用管理路由（需要JWT验证）
	app := apiV1.Group("/app")
	app.Use(middleware2.JWTAuth()) // 应用管理需要JWT认证
	appHandler := v1.NewApp(s.appService)
	app.GET("/list", appHandler.GetApps)
	app.POST("/create", appHandler.CreateApp)
	app.POST("/update/:app", appHandler.UpdateApp)
	app.DELETE("/delete/:app", appHandler.DeleteApp)
	// 支持所有 HTTP 方法的请求应用接口
	request := apiV1.Group("/run")
	request.Use(middleware2.JWTAuth())
	request.Any("/*router", appHandler.RequestApp)

	callback := apiV1.Group("/callback")
	callback.Use(middleware2.JWTAuth())
	callback.POST("/*router", appHandler.CallbackApp)

	// 服务目录管理路由（需要JWT验证）
	serviceTree := apiV1.Group("/service_tree")
	serviceTreeHandler := v1.NewServiceTree(s.serviceTreeService, s.functionGenService)

	// 需要JWT验证的路由
	serviceTreeAuth := serviceTree.Group("")
	serviceTreeAuth.Use(middleware2.JWTAuth()) // 服务目录管理需要JWT认证
	serviceTreeAuth.POST("", serviceTreeHandler.CreateServiceTree)
	serviceTreeAuth.GET("", serviceTreeHandler.GetServiceTree)
	serviceTreeAuth.PUT("", serviceTreeHandler.UpdateServiceTree)
	serviceTreeAuth.DELETE("", serviceTreeHandler.DeleteServiceTree)
	serviceTreeAuth.POST("/copy", serviceTreeHandler.CopyServiceTree)                 // 复制服务目录
	serviceTreeAuth.POST("/publish_to_hub", serviceTreeHandler.PublishDirectoryToHub) // 发布目录到 Hub
	serviceTreeAuth.POST("/push_to_hub", serviceTreeHandler.PushDirectoryToHub)       // 推送目录到 Hub（更新已发布的目录）
	serviceTreeAuth.GET("/hub_info", serviceTreeHandler.GetHubInfo)                    // 获取目录的 Hub 信息
	serviceTreeAuth.POST("/pull_from_hub", serviceTreeHandler.PullDirectoryFromHub)    // 从 Hub 拉取目录

	// 服务间调用路由（不需要JWT验证）
	serviceTree.POST("/add_functions", serviceTreeHandler.AddFunctions) // 向服务目录添加函数（agent-server -> workspace）

	// 函数管理路由（需要JWT验证）
	function := apiV1.Group("/function")
	function.Use(middleware2.JWTAuth()) // 函数管理需要JWT认证
	functionHandler := v1.NewFunction(s.functionService)
	function.GET("/get", functionHandler.GetFunction)
	function.GET("/list", functionHandler.GetFunctionsByApp)

	// 用户管理路由（需要JWT验证）
	user := apiV1.Group("/user")
	user.Use(middleware2.JWTAuth()) // 用户管理需要JWT认证
	userHandler := v1.NewUser(s.userService)
	user.GET("/info", userHandler.GetUserInfo)
	user.GET("/query", userHandler.QueryUser)
	user.GET("/search_fuzzy", userHandler.SearchUsersFuzzy)
	user.PUT("/update", userHandler.UpdateUser)

	// 批量获取用户（需要JWT验证）
	users := apiV1.Group("/users")
	users.Use(middleware2.JWTAuth())
	users.POST("", userHandler.GetUsersByUsernames)

	// 操作日志路由（需要JWT验证 + 操作日志功能鉴权）
	operateLog := apiV1.Group("/operate_log")
	operateLog.Use(middleware2.JWTAuth())                                    // JWT 认证
	operateLog.Use(middleware2.RequireFeature(enterprise.FeatureOperateLog)) // 操作日志功能鉴权（企业版）
	operateLogHandler := v1.NewOperateLog(s.operateLogService)
	operateLog.GET("/table", operateLogHandler.GetTableOperateLogs) // 查询 Table 操作日志
	operateLog.GET("/form", operateLogHandler.GetFormOperateLogs)   // 查询 Form 操作日志

	// 目录更新历史路由（需要JWT验证）
	directoryUpdateHistory := apiV1.Group("/directory_update_history")
	directoryUpdateHistory.Use(middleware2.JWTAuth()) // 目录更新历史需要JWT认证
	directoryUpdateHistoryHandler := v1.NewDirectoryUpdateHistory(s.directoryUpdateHistoryService)
	directoryUpdateHistory.GET("/app_version", directoryUpdateHistoryHandler.GetAppVersionUpdateHistory) // 获取应用版本更新历史（App视角）
	directoryUpdateHistory.GET("/directory", directoryUpdateHistoryHandler.GetDirectoryUpdateHistory)    // 获取目录更新历史（目录视角）

}
