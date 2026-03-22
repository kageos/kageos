package server

import (
	"context"

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
	pprof.RegisterPprofRoutes(s.httpServer)

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
	app.GET("/detail/:app", appHandler.GetAppDetail)
	// ⭐ 服务树接口：使用 user 和 app 参数（从 full-code-path 中解析）
	// 格式：/workspace/api/v1/app/{user}/{app}/tree
	app.GET("/:user/:app/tree", middleware2.Gzip(), appHandler.GetAppWithServiceTree)
	app.POST("/create", appHandler.CreateApp)
	// ⭐ 更新应用接口（debug 功能，生产环境不存在，无需权限检查）
	app.POST("/update/:user/:app", appHandler.UpdateApp)
	// ⭐ 更新工作空间接口（只更新 MySQL 记录，不涉及容器更新，需要 app:admin 权限）
	//app.PUT("/workspace/:user/:app", middleware2.CheckWorkspaceUpdate(), appHandler.UpdateWorkspace)
	app.PUT("/workspace/:user/:app", appHandler.UpdateWorkspace)
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
	serviceTreeHandler := v1.NewServiceTree(s.serviceTreeService)

	// 需要JWT验证的路由
	serviceTreeAuth := serviceTree.Group("")
	serviceTreeAuth.Use(middleware2.JWTAuth()) // 服务目录管理需要JWT认证
	serviceTreeAuth.POST("", serviceTreeHandler.CreateServiceTree)
	// 服务树接口使用 gzip 压缩
	serviceTreeAuth.GET("", middleware2.Gzip(), serviceTreeHandler.GetServiceTree)
	serviceTreeAuth.GET("/detail", serviceTreeHandler.GetServiceTreeDetail)      // ⭐ 获取服务目录详情（包含权限，兼容旧接口）
	serviceTreeAuth.GET("/package_info", serviceTreeHandler.GetPackageInfo)      // ⭐ 获取目录信息（仅用于获取目录权限，函数权限从函数详情接口获取）
	serviceTreeAuth.GET("/search_functions", serviceTreeHandler.SearchFunctions) // ⭐ 搜索函数
	serviceTreeAuth.PUT("", serviceTreeHandler.UpdateServiceTree)
	serviceTreeAuth.DELETE("", serviceTreeHandler.DeleteServiceTree)
	serviceTreeAuth.POST("/copy", serviceTreeHandler.CopyServiceTree)                 // 复制服务目录
	serviceTreeAuth.POST("/publish_to_hub", serviceTreeHandler.PublishDirectoryToHub) // 发布目录到 Hub
	serviceTreeAuth.POST("/push_to_hub", serviceTreeHandler.PushDirectoryToHub)       // 推送目录到 Hub（更新已发布的目录）
	serviceTreeAuth.GET("/hub_push_form_info", serviceTreeHandler.GetHubPushFormInfo) // 获取推送表单信息（预填 + 下一版本号）
	serviceTreeAuth.GET("/hub_info", serviceTreeHandler.GetHubInfo)                   // 获取目录的 Hub 信息
	serviceTreeAuth.POST("/pull_from_hub", serviceTreeHandler.PullDirectoryFromHub)           // 从 Hub 拉取目录
	serviceTreeAuth.POST("/import_hub_bundle", serviceTreeHandler.ImportHubDirectoryBundle)   // 从离线 JSON 包安装目录

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
	postsAuth.GET("", middleware2.CheckBoardRead(), boardPostHandler.ListPosts)                                    // GET 列表：query full_code_path
	postsAuth.GET("/:id", middleware2.CheckBoardReadFromPostID(getPostPath), boardPostHandler.GetPost)             // GET 详情
	postsAuth.POST("", middleware2.CheckBoardWrite(), boardPostHandler.CreatePost)                                 // POST 发帖：body full_code_path
	postsAuth.PUT("/:id", middleware2.CheckBoardUpdateFromPostID(getPostPath), boardPostHandler.UpdatePost)        // PUT 更新
	postsAuth.DELETE("/:id", middleware2.CheckBoardDeleteFromPostID(getPostPath), boardPostHandler.DeletePost)    // DELETE 删除

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
	// ⭐ /list 路由放在通配符路由之前，避免路由冲突
	function.GET("/list", functionHandler.GetFunctionsByApp)
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

	// ⭐ 权限管理路由（需要JWT验证 + 权限管理功能鉴权）
	permission := apiV1.Group("/permission")
	permission.Use(middleware2.JWTAuth())                                    // JWT 认证
	permission.Use(middleware2.RequireFeature(enterprise.FeaturePermission)) // 权限管理功能鉴权（企业版）
	permissionHandler := v1.NewPermission(s.permissionService)
	permission.POST("/apply", permissionHandler.ApplyPermission)            // 权限申请（角色申请）
	permission.GET("/workspace", permissionHandler.GetWorkspacePermissions) // 获取工作空间所有权限
	permission.GET("/resource", permissionHandler.GetResourcePermissions)   // 查询资源的所有权限分配

	// ⭐ 权限申请和审批路由（新权限系统）
	permission.POST("/request/create", permissionHandler.CreatePermissionRequest)   // 创建权限申请
	permission.POST("/request/approve", permissionHandler.ApprovePermissionRequest) // 审批通过
	permission.POST("/request/reject", permissionHandler.RejectPermissionRequest)   // 审批拒绝
	permission.GET("/requests", permissionHandler.GetPermissionRequests)            // 获取权限申请列表

	// ⭐ 角色管理路由（需要JWT验证 + 权限管理功能鉴权）
	role := apiV1.Group("/role")
	role.Use(middleware2.JWTAuth())                                    // JWT 认证
	role.Use(middleware2.RequireFeature(enterprise.FeaturePermission)) // 权限管理功能鉴权（企业版）
	// 角色管理使用与 Permission 一致的 permissionService（从 Server 注入）
	roleHandler := v1.NewRoleHandlerFromPermissionService(s.permissionService)
	role.GET("", roleHandler.GetRoles)                                    // 获取所有角色
	role.GET("/:id", roleHandler.GetRole)                                 // 获取角色详情
	role.POST("", roleHandler.CreateRole)                                 // 创建角色
	role.PUT("/:id", roleHandler.UpdateRole)                              // 更新角色
	role.DELETE("/:id", roleHandler.DeleteRole)                           // 删除角色
	role.POST("/assign/user", roleHandler.AssignRoleToUser)               // 给用户分配角色
	role.POST("/assign/department", roleHandler.AssignRoleToDepartment)   // 给组织架构分配角色
	role.POST("/remove/user", roleHandler.RemoveRoleFromUser)             // 移除用户角色
	role.POST("/remove/department", roleHandler.RemoveRoleFromDepartment) // 移除组织架构角色
	role.POST("/user", roleHandler.GetUserRoles)                          // 获取用户角色
	role.POST("/department", roleHandler.GetDepartmentRoles)              // 获取组织架构角色
	role.GET("/for_request", roleHandler.GetRolesForPermissionRequest)    // 获取可用于权限申请的角色列表（根据节点类型过滤）

	// 定时任务（atime/cron/every + 执行记录）
	scheduledTask := apiV1.Group("/scheduled_tasks")
	scheduledTask.Use(middleware2.JWTAuth())
	scheduledTaskHandler := v1.NewScheduledTask(s.scheduledTaskService)
	scheduledTask.POST("", scheduledTaskHandler.Create)              // 创建定时任务
	scheduledTask.GET("", scheduledTaskHandler.List)                 // 列表
	scheduledTask.DELETE("/:id", scheduledTaskHandler.Cancel)        // 取消
	scheduledTask.GET("/:id/executions", scheduledTaskHandler.ListExecutions) // 执行记录
}
