package server

import (
	v1 "github.com/ai-agent-os/ai-agent-os/core/app-server/api/v1"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	middleware2 "github.com/ai-agent-os/ai-agent-os/pkg/middleware"
	"github.com/ai-agent-os/ai-agent-os/pkg/pprof"
	"github.com/gin-gonic/gin"
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

	// Workspace 路由组（统一使用 /workspace/api/v1 开头，方便网关代理）
	workspace := s.httpServer.Group("/workspace")
	apiV1 := workspace.Group("/api/v1")

	// ⭐ 统一添加用户信息中间件，所有接口都需要（网关会透传 token，解析后设置到 X-Request-User header）
	apiV1.Use(middleware2.WithUserInfo())

	// 应用管理路由（需要JWT验证）
	app := apiV1.Group("/app")
	app.Use(middleware2.JWTAuth()) // 应用管理需要JWT认证
	appHandler := v1.NewApp(s.appService, s.serviceTreeService)
	app.GET("/list", appHandler.GetApps)
	app.GET("/detail", appHandler.GetAppDetail)
	app.GET("/tree", middleware2.Gzip(), appHandler.GetAppWithServiceTree)
	app.DELETE("/delete", middleware2.CheckAppDelete(), appHandler.DeleteApp)
	app.POST("/create", appHandler.CreateApp)
	app.POST("/update", appHandler.UpdateApp)
	app.PUT("/workspace", appHandler.UpdateWorkspace)

	// 服务目录管理路由（需要JWT验证）
	serviceTree := apiV1.Group("/service_tree")
	serviceTreeHandler := v1.NewServiceTree(s.serviceTreeService)

	// 需要JWT验证的路由
	serviceTreeAuth := serviceTree.Group("")
	serviceTreeAuth.Use(middleware2.JWTAuth())                                   // 服务目录管理需要JWT认证
	serviceTreeAuth.GET("/detail", serviceTreeHandler.GetServiceTreeDetail)      // ⭐ 获取服务目录详情（包含权限，兼容旧接口）
	serviceTreeAuth.GET("/search_functions", serviceTreeHandler.SearchFunctions) // ⭐ 搜索函数
	serviceTreeAuth.GET("/search_resources", serviceTreeHandler.SearchResources) // 全站资源搜索（目录/函数/文档/讨论区）
	serviceTreeAuth.POST("/copy", serviceTreeHandler.CopyServiceTree)            // 复制服务目录
	serviceTreeAuth.GET("/export_capability_bundle", serviceTreeHandler.ExportCapabilityBundle)
	serviceTreeAuth.POST("/export_capability_bundle", serviceTreeHandler.ExportCapabilityBundle)
	serviceTreeAuth.POST("/install_capability_bundle", serviceTreeHandler.InstallCapabilityBundle)

	// ⭐ 按类型分离的 CRUD 接口（推荐使用）
	// ==================== Package 类型接口 ====================
	packagesAuth := apiV1.Group("/packages")
	packagesAuth.Use(middleware2.JWTAuth())
	packagesAuth.POST("", serviceTreeHandler.CreatePackage)       // POST /api/v1/packages
	packagesAuth.PUT("/:id", serviceTreeHandler.UpdatePackage)    // PUT /api/v1/packages/:id
	packagesAuth.DELETE("/:id", serviceTreeHandler.DeletePackage) // DELETE /api/v1/packages/:id

	// ==================== Function 类型接口 ====================
	functionsAuth := apiV1.Group("/functions")
	functionsAuth.Use(middleware2.JWTAuth())
	functionsAuth.POST("", serviceTreeHandler.CreateFunction)       // POST /api/v1/functions
	functionsAuth.PUT("/:id", serviceTreeHandler.UpdateFunction)    // PUT /api/v1/functions/:id
	functionsAuth.DELETE("/:id", serviceTreeHandler.DeleteFunction) // DELETE /api/v1/functions/:id

	// ==================== Docs 类型接口 ====================
	// ⭐ docs CRUD 接口（使用 /docs/crud 避免与文档管理路由冲突）
	docsCrudAuth := apiV1.Group("/docs/crud")
	docsCrudAuth.Use(middleware2.JWTAuth())
	docsCrudAuth.POST("", serviceTreeHandler.CreateDocs)       // POST /api/v1/docs/crud
	docsCrudAuth.PUT("/:id", serviceTreeHandler.UpdateDocs)    // PUT /api/v1/docs/crud/:id
	docsCrudAuth.DELETE("/:id", serviceTreeHandler.DeleteDocs) // DELETE /api/v1/docs/crud/:id

	// ==================== Board 类型接口（版块/讨论区） ====================
	boardsCrudAuth := apiV1.Group("/boards/crud")
	boardsCrudAuth.Use(middleware2.JWTAuth())
	boardsCrudAuth.POST("", serviceTreeHandler.CreateBoard)       // POST /api/v1/boards/crud
	boardsCrudAuth.PUT("/:id", serviceTreeHandler.UpdateBoard)    // PUT /api/v1/boards/crud/:id
	boardsCrudAuth.DELETE("/:id", serviceTreeHandler.DeleteBoard) // DELETE /api/v1/boards/crud/:id

	// ==================== 版块帖子接口（讨论区鉴权：board:read/write/update/delete） ====================
	postsAuth := apiV1.Group("/posts")
	postsAuth.Use(middleware2.JWTAuth())
	boardPostHandler := v1.NewBoardPost(s.boardService)
	getPostPath := func(c *gin.Context, id int64) (string, error) {
		return s.boardService.GetPostPath(contextx.ToContext(c), id)
	}
	postsAuth.GET("", middleware2.CheckBoardRead(), boardPostHandler.ListPosts)                                // GET 列表：query full_code_path
	postsAuth.GET("/:id", middleware2.CheckBoardReadFromPostID(getPostPath), boardPostHandler.GetPost)         // GET 详情
	postsAuth.POST("", middleware2.CheckBoardWrite(), boardPostHandler.CreatePost)                             // POST 发帖：body full_code_path
	postsAuth.PUT("/:id", middleware2.CheckBoardUpdateFromPostID(getPostPath), boardPostHandler.UpdatePost)    // PUT 更新
	postsAuth.DELETE("/:id", middleware2.CheckBoardDeleteFromPostID(getPostPath), boardPostHandler.DeletePost) // DELETE 删除

	// ⭐ 文档管理路由（基于完整路径，与 table/form/chart 风格一致）
	docs := apiV1.Group("/docs")
	docs.Use(middleware2.JWTAuth())
	docHandler := v1.NewDoc(s.docService)
	docs.GET("/search", docHandler.SearchDocs)                                               // 搜索文档（模糊搜索）
	docs.GET("/batch", docHandler.BatchGetDocs)                                              // 批量获取文档（精确查询）
	docs.GET("/info/*full-code-path", middleware2.CheckDocRead(), docHandler.GetDoc)         // 获取文档
	docs.PUT("/info/*full-code-path", middleware2.CheckDocWrite(), docHandler.UpdateDoc)     // 更新文档
	docs.DELETE("/info/*full-code-path", middleware2.CheckDocDelete(), docHandler.DeleteDoc) // 删除文档

	// ==================== 服务间调用路由 ====================
	// 服务间调用路由（不需要JWT验证，但用户信息中间件已在 apiV1 级别统一添加）
	serviceTree.POST("/add_functions", serviceTreeHandler.AddFunctions) // 向服务目录添加函数（agent-server -> workspace）

	// 工作台环境信息路由（不需要JWT验证，但用户信息中间件已在 apiV1 级别统一添加）
	workspaceGroup := apiV1.Group("/workspace")
	workspaceGroup.GET("/context", serviceTreeHandler.GetWorkspaceContext)       // 获取工作台环境信息（agent-server -> workspace）
	workspaceGroup.POST("/files/replace", serviceTreeHandler.ReplaceFileContent) // 工作台文件 search-replace（实时写盘）
	workspaceGroup.POST("/files/delete", serviceTreeHandler.DeleteFile)          // 工作台删除文件（删磁盘+删节点）
	workspaceGroup.POST("/logs/read", serviceTreeHandler.ReadAppLog)             // 工作台读取应用日志（支持 version/keyword）

	// 函数管理路由（需要JWT验证）
	function := apiV1.Group("/function")
	function.Use(middleware2.JWTAuth()) // 函数管理需要JWT认证
	functionHandler := v1.NewFunction(s.functionService)
	// ⭐ 使用 /info/:func-type/*full-code-path 作为路径参数，函数类型直接从 URL 路径获取
	// ⭐ 这样后端无需查询数据库即可构造权限点（table:read、form:read、chart:read）
	function.GET("/info/:func-type/*full-code-path", middleware2.CheckFunctionRead(), functionHandler.GetFunction)

	// 操作日志路由（需要JWT验证 + 操作日志功能鉴权）
	operateLog := apiV1.Group("/operate_log")
	operateLog.Use(middleware2.JWTAuth())                                    // JWT 认证
	operateLog.Use(middleware2.RequireFeature(enterprise.FeatureOperateLog)) // 操作日志功能鉴权（企业版）
	operateLogHandler := v1.NewOperateLog()                                  // 使用企业版接口，无需传入服务
	operateLog.GET("/table", operateLogHandler.GetTableOperateLogs)          // 查询 Table 操作日志
	operateLog.GET("/form", operateLogHandler.GetFormOperateLogs)            // 查询 Form 操作日志

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
	table.GET("/search/*full-code-path", middleware2.CheckTableSearch(), standardAPI.TableSearch)            // Table 查询
	table.GET("/template/*full-code-path", middleware2.CheckTableRead(), standardAPI.TableTemplate)          // Table 下载导入模板
	table.POST("/create/*full-code-path", middleware2.CheckTableWrite(), standardAPI.TableCreate)            // Table 新增
	table.POST("/batch-create/*full-code-path", middleware2.CheckTableWrite(), standardAPI.TableBatchCreate) // Table 批量导入
	table.PUT("/update/*full-code-path", middleware2.CheckTableUpdate(), standardAPI.TableUpdate)            // Table 更新
	table.DELETE("/delete/*full-code-path", middleware2.CheckTableDelete(), standardAPI.TableDelete)         // Table 删除

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

	advancedPermission := middleware2.RequireFeature(enterprise.FeaturePermission)

	// ⭐ 权限管理路由：基础用户授权社区版可用，高级申请/审批能力需要企业版
	permission := apiV1.Group("/permission")
	permission.Use(middleware2.JWTAuth()) // JWT 认证
	permissionHandler := v1.NewPermission(s.permissionService)
	permission.POST("/apply", advancedPermission, permissionHandler.ApplyPermission) // 权限申请（角色申请，企业版）
	permission.GET("/workspace", permissionHandler.GetWorkspacePermissions)          // 获取工作空间所有权限
	permission.GET("/resource", permissionHandler.GetResourcePermissions)            // 查询资源的所有权限分配

	// ⭐ 权限申请和审批路由（新权限系统）
	permission.POST("/request/create", advancedPermission, permissionHandler.CreatePermissionRequest)   // 创建权限申请（企业版）
	permission.POST("/request/approve", advancedPermission, permissionHandler.ApprovePermissionRequest) // 审批通过（企业版）
	permission.POST("/request/reject", advancedPermission, permissionHandler.RejectPermissionRequest)   // 审批拒绝（企业版）
	permission.GET("/requests", advancedPermission, permissionHandler.GetPermissionRequests)            // 获取权限申请列表（企业版）

	// ⭐ 角色管理路由：预设角色查询/用户授权社区版可用，自定义角色/组织架构/申请辅助需要企业版
	role := apiV1.Group("/role")
	role.Use(middleware2.JWTAuth()) // JWT 认证
	// 角色管理使用与 Permission 一致的 permissionService（从 Server 注入）
	roleHandler := v1.NewRoleHandlerFromPermissionService(s.permissionService)
	role.GET("", roleHandler.GetRoles)                                                        // 获取所有角色
	role.GET("/for_request", roleHandler.GetRolesForPermissionRequest)                        // 获取可用于权限申请/赋权的角色列表
	role.GET("/:id", roleHandler.GetRole)                                                     // 获取角色详情
	role.POST("", advancedPermission, roleHandler.CreateRole)                                 // 创建自定义角色（企业版）
	role.PUT("/:id", advancedPermission, roleHandler.UpdateRole)                              // 更新角色（企业版）
	role.DELETE("/:id", advancedPermission, roleHandler.DeleteRole)                           // 删除角色（企业版）
	role.POST("/assign/user", roleHandler.AssignRoleToUser)                                   // 给用户分配角色
	role.POST("/assign/department", advancedPermission, roleHandler.AssignRoleToDepartment)   // 给组织架构分配角色（企业版）
	role.POST("/remove/user", roleHandler.RemoveRoleFromUser)                                 // 移除用户角色
	role.POST("/remove/department", advancedPermission, roleHandler.RemoveRoleFromDepartment) // 移除组织架构角色（企业版）
	role.POST("/user", roleHandler.GetUserRoles)                                              // 获取用户角色
	role.POST("/department", advancedPermission, roleHandler.GetDepartmentRoles)              // 获取组织架构角色（企业版）

	// 定时任务（atime/cron/every + 执行记录）
	scheduledTask := apiV1.Group("/scheduled_tasks")
	scheduledTask.Use(middleware2.JWTAuth())
	scheduledTaskHandler := v1.NewScheduledTask(s.scheduledTaskService)
	scheduledTask.POST("", scheduledTaskHandler.Create)                                   // 创建定时任务
	scheduledTask.GET("", scheduledTaskHandler.List)                                      // 列表
	scheduledTask.GET("/:id", scheduledTaskHandler.Get)                                   // 详情
	scheduledTask.POST("/:id/cancel", scheduledTaskHandler.Cancel)                        // 取消
	scheduledTask.DELETE("/:id", scheduledTaskHandler.Delete)                             // 删除
	scheduledTask.GET("/:id/executions", scheduledTaskHandler.ListExecutions)             // 执行记录
	scheduledTask.GET("/:id/executions/:execution_id", scheduledTaskHandler.GetExecution) // 执行记录详情
}
