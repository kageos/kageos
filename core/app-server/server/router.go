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
	serviceTree.Use(middleware2.JWTAuth()) // 服务目录管理需要JWT认证
	serviceTreeHandler := v1.NewServiceTree(s.serviceTreeService)
	serviceTree.POST("", serviceTreeHandler.CreateServiceTree)
	serviceTree.GET("", serviceTreeHandler.GetServiceTree)
	serviceTree.PUT("", serviceTreeHandler.UpdateServiceTree)
	serviceTree.DELETE("", serviceTreeHandler.DeleteServiceTree)

	// 函数管理路由（需要JWT验证）
	function := apiV1.Group("/function")
	function.Use(middleware2.JWTAuth()) // 函数管理需要JWT认证
	functionHandler := v1.NewFunction(s.functionService)
	function.GET("/get", functionHandler.GetFunction)
	function.GET("/list", functionHandler.GetFunctionsByApp)
	function.POST("/fork", functionHandler.ForkFunctionGroup)

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
}
