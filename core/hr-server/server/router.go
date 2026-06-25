package server

import (
	v1 "github.com/kageos/kageos/core/hr-server/api/v1"
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

	// HR 路由组（统一使用 /hr/api/v1 开头，方便网关代理）
	hr := s.httpServer.Group("/hr")
	apiV1 := hr.Group("/api/v1")

	// 认证相关路由（不需要JWT验证）
	auth := apiV1.Group("/auth")
	authHandler := v1.NewAuth(s.authService, s.authOAuthService, s.emailService, s.settingsService, s.userService, s.departmentService)
	auth.POST("/send_email_code", authHandler.SendEmailCode)
	auth.POST("/register", authHandler.Register)
	auth.GET("/companies/search", authHandler.SearchCompanies)
	auth.GET("/oauth/registration/:ticket", authHandler.GetOAuthRegistrationIntent)
	auth.POST("/oauth/registration/:ticket/confirm", authHandler.ConfirmOAuthRegistration)
	auth.GET("/:provider/authorize", authHandler.OAuthAuthorize)
	auth.GET("/:provider/callback", authHandler.OAuthCallback)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.RefreshToken)
	auth.POST("/logout", authHandler.Logout)
	auth.POST("/forgot_password", authHandler.ForgotPassword)
	authProviderHandler := v1.NewAuthLoginProvider(s.authProviderService)
	auth.GET("/methods", authProviderHandler.PublicMethods)

	// 用户管理路由（需要JWT验证）
	user := apiV1.Group("/user")
	user.Use(middleware2.JWTAuth()) // 用户管理需要JWT认证
	userHandler := v1.NewUser(s.userService, s.departmentService)
	user.GET("/info", userHandler.GetUserInfo)
	user.GET("/query", userHandler.QueryUser)
	user.GET("/search_fuzzy", userHandler.SearchUsersFuzzy)
	user.PUT("/update", userHandler.UpdateUser)
	user.GET("/openapi_tokens", userHandler.ListOpenAPITokens)
	user.POST("/openapi_tokens", userHandler.CreateOpenAPIToken)
	user.POST("/openapi_tokens/revoke", userHandler.RevokeOpenAPIToken)

	systemSettings := apiV1.Group("/system/settings")
	systemSettings.Use(middleware2.JWTAuth())
	systemSettingsHandler := v1.NewSystemSettings(s.settingsService)
	systemSettings.GET("", systemSettingsHandler.Get)
	systemSettings.PUT("", systemSettingsHandler.Update)
	systemSettings.POST("/email/test", systemSettingsHandler.TestEmail)
	systemSettings.GET("/tls", systemSettingsHandler.GetTLS)
	systemSettings.PUT("/tls", systemSettingsHandler.UpdateTLS)
	systemSettings.POST("/tls/reload", systemSettingsHandler.ReloadTLS)

	systemAuthProviders := apiV1.Group("/system/auth/providers")
	systemAuthProviders.Use(middleware2.JWTAuth())
	systemAuthProviders.GET("", authProviderHandler.List)
	systemAuthProviders.PUT("/:code/config", authProviderHandler.UpdateConfig)
	systemAuthProviders.PUT("/:code/enabled", authProviderHandler.SetEnabled)

	systemUsers := apiV1.Group("/system/users")
	systemUsers.Use(middleware2.JWTAuth())
	systemUserHandler := v1.NewSystemUser(s.userService, s.departmentService)
	systemUsers.GET("", systemUserHandler.List)
	systemUsers.POST("", systemUserHandler.Create)
	systemUsers.PUT("/:username", systemUserHandler.Update)
	systemUsers.POST("/:username/password", systemUserHandler.ResetPassword)
	systemUsers.PUT("/:username/status", systemUserHandler.UpdateStatus)

	// 批量获取用户（需要JWT验证）
	users := apiV1.Group("/users")
	users.Use(middleware2.JWTAuth())
	users.POST("", userHandler.GetUsersByUsernames)

	// 部门管理路由（需要JWT验证）
	department := apiV1.Group("/department")
	department.Use(middleware2.JWTAuth())
	departmentHandler := v1.NewDepartment(s.departmentService)
	department.POST("", departmentHandler.CreateDepartment)
	department.GET("/tree", departmentHandler.GetDepartmentTree)
	department.GET("/:id", departmentHandler.GetDepartmentByID)
	department.PUT("/:id", departmentHandler.UpdateDepartment)
	department.DELETE("/:id", departmentHandler.DeleteDepartment)

	// 批量获取部门路由（需要JWT验证）
	departments := apiV1.Group("/departments")
	departments.Use(middleware2.JWTAuth())
	departments.GET("", departmentHandler.GetDepartmentsByPaths)

	// 用户分配路由（需要JWT验证）
	userAllocationHandler := v1.NewUserAllocation(s.userService, s.departmentService)
	user.POST("/assign", userAllocationHandler.AssignUser)
	user.GET("/department", userAllocationHandler.GetUsersByDepartment)

	serverx.ApplyRouteRegistrars(serverx.ServiceHRServer, s.httpServer)
}
