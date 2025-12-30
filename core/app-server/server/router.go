package server

import (
	"context"

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
	appHandler := v1.NewApp(s.appService, s.serviceTreeService)
	app.GET("/list", appHandler.GetApps)
	app.GET("/detail/:app", appHandler.GetAppDetail)
	// ⭐ 服务树接口：使用 user 和 app 参数（从 full-code-path 中解析）
	// 格式：/workspace/api/v1/app/{user}/{app}/tree
	app.GET("/:user/:app/tree", middleware2.Gzip(), appHandler.GetAppWithServiceTree)
	app.POST("/create", appHandler.CreateApp)
	// ⭐ 添加应用更新权限检查
	app.POST("/update/:app", middleware2.CheckAppUpdate(), appHandler.UpdateApp)
	// ⭐ 添加应用删除权限检查
	app.DELETE("/delete/:app", middleware2.CheckAppDelete(), appHandler.DeleteApp)
	// 支持所有 HTTP 方法的请求应用接口
	request := apiV1.Group("/run")
	request.Use(middleware2.JWTAuth())
	// ⭐ 添加权限检查中间件（动态根据函数类型和HTTP方法确定权限点）
	request.Use(middleware2.CheckFunctionExecute(func(ctx context.Context, fullCodePath string) (string, error) {
		// 根据 full-code-path 获取服务树节点（包含 template_type）
		serviceTree, err := s.serviceTreeService.GetServiceTreeByFullPath(ctx, fullCodePath)
		if err != nil {
			// 如果查询失败，返回空字符串（使用默认的 function:manage 权限）
			return "", nil
		}
		return serviceTree.TemplateType, nil
	}))
	request.Any("/*router", appHandler.RequestApp)

	// ⭐ 旧的回调接口（已注释，改用标准接口）
	// callback := apiV1.Group("/callback")
	// callback.Use(middleware2.JWTAuth())
	// // ⭐ 添加回调接口权限检查
	// callback.Use(middleware2.CheckCallback())
	// callback.POST("/*router", appHandler.CallbackApp)

	// 服务目录管理路由（需要JWT验证）
	serviceTree := apiV1.Group("/service_tree")
	serviceTreeHandler := v1.NewServiceTree(s.serviceTreeService, s.functionGenService)

	// 需要JWT验证的路由
	serviceTreeAuth := serviceTree.Group("")
	serviceTreeAuth.Use(middleware2.JWTAuth()) // 服务目录管理需要JWT认证
	serviceTreeAuth.POST("", serviceTreeHandler.CreateServiceTree)
	// 服务树接口使用 gzip 压缩
	serviceTreeAuth.GET("", middleware2.Gzip(), serviceTreeHandler.GetServiceTree)
	serviceTreeAuth.GET("/detail", serviceTreeHandler.GetServiceTreeDetail) // ⭐ 获取服务目录详情（包含权限，兼容旧接口）
	serviceTreeAuth.GET("/package_info", serviceTreeHandler.GetPackageInfo) // ⭐ 获取目录信息（仅用于获取目录权限，函数权限从函数详情接口获取）
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

	// ⭐ 标准接口路由（使用 full-code-path，便于权限控制）
	standardAPI := v1.NewStandardAPI(s.appService)

	// Table 函数接口
	table := apiV1.Group("/table")
	table.Use(middleware2.JWTAuth())
	table.GET("/search/*full-code-path", middleware2.CheckTableSearch(), standardAPI.TableSearch)     // Table 查询
	table.POST("/create/*full-code-path", middleware2.CheckTableWrite(), standardAPI.TableCreate)    // Table 新增
	table.PUT("/update/*full-code-path", middleware2.CheckTableUpdate(), standardAPI.TableUpdate)     // Table 更新
	table.DELETE("/delete/*full-code-path", middleware2.CheckTableDelete(), standardAPI.TableDelete) // Table 删除

	// Form 函数接口
	form := apiV1.Group("/form")
	form.Use(middleware2.JWTAuth())
	form.POST("/submit/*full-code-path", middleware2.CheckFormWrite(), standardAPI.FormSubmit) // Form 提交

	// Chart 函数接口
	chart := apiV1.Group("/chart")
	chart.Use(middleware2.JWTAuth())
	chart.GET("/query/*full-code-path", middleware2.CheckChartQuery(), standardAPI.ChartQuery) // Chart 查询

	// Callback 接口（不需要权限检查，因为这是内部回调）
	callbackStandard := apiV1.Group("/callback")
	callbackStandard.Use(middleware2.JWTAuth())
	callbackStandard.POST("/on_select_fuzzy/*full-code-path", standardAPI.CallbackOnSelectFuzzy) // 模糊搜索回调

	// ⭐ 权限管理路由（需要JWT验证 + 权限管理功能鉴权）
	permission := apiV1.Group("/permission")
	permission.Use(middleware2.JWTAuth())                                    // JWT 认证
	permission.Use(middleware2.RequireFeature(enterprise.FeaturePermission)) // 权限管理功能鉴权（企业版）
	permissionHandler := v1.NewPermission(s.permissionService)
	permission.POST("/add", permissionHandler.AddPermission)                // 添加权限（内部使用，被 ApplyPermission 调用）
	permission.POST("/apply", permissionHandler.ApplyPermission)             // 权限申请
	permission.GET("/workspace", permissionHandler.GetWorkspacePermissions)  // 获取工作空间所有权限

}
