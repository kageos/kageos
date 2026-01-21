package server

import (
	v1 "github.com/ai-agent-os/ai-agent-os/core/hr-server/api/v1"
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

	// HR 路由组（统一使用 /hr/api/v1 开头，方便网关代理）
	hr := s.httpServer.Group("/hr")
	apiV1 := hr.Group("/api/v1")

	// 认证相关路由（不需要JWT验证）
	auth := apiV1.Group("/auth")
	authHandler := v1.NewAuth(s.authService, s.emailService, s.userService, s.departmentService)
	auth.POST("/send_email_code", authHandler.SendEmailCode)
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.RefreshToken)
	auth.POST("/logout", authHandler.Logout)
	auth.POST("/forgot_password", authHandler.ForgotPassword)

	// 用户管理路由（需要JWT验证）
	user := apiV1.Group("/user")
	user.Use(middleware2.JWTAuth()) // 用户管理需要JWT认证
	userHandler := v1.NewUser(s.userService, s.departmentService)
	user.GET("/info", userHandler.GetUserInfo)
	user.GET("/query", userHandler.QueryUser)
	user.GET("/search_fuzzy", userHandler.SearchUsersFuzzy)
	user.PUT("/update", userHandler.UpdateUser)

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

	// 用户分配路由（需要JWT验证）
	userAllocationHandler := v1.NewUserAllocation(s.userService, s.departmentService)
	user.POST("/assign", userAllocationHandler.AssignUser)
	user.GET("/department", userAllocationHandler.GetUsersByDepartment)
}
