package server

import (
	v1 "github.com/kageos/kageos/core/app-server/api/v1"
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

	publicShareHandler := v1.NewPublicShareAPI(s.publicShareService, s.appService, s.teamAccessService)
	public := s.httpServer.Group("/public/api")
	public.POST("/anonymous-token", publicShareHandler.AnonymousToken)
	public.GET("/s/:share_id", publicShareHandler.View)
	public.POST("/s/:share_id/submit", publicShareHandler.Submit)
	public.POST("/s/:share_id/selection-options", publicShareHandler.CallbackOnSelectFuzzy)

	// Workspace 路由组（统一使用 /workspace/api/v1 开头，方便网关代理）
	workspace := s.httpServer.Group("/workspace")
	apiV1 := workspace.Group("/api/v1")

	// 应用管理路由（需要JWT验证）
	app := apiV1.Group("/apps")
	app.Use(middleware2.JWTAuth()) // 应用管理需要JWT认证
	appHandler := v1.NewApp(s.appService, s.serviceTreeService, s.teamAccessService)
	app.GET("", appHandler.GetApps)
	app.GET("/detail", appHandler.GetAppDetail)
	app.GET("/tree", middleware2.Gzip(), appHandler.GetAppWithServiceTree)
	app.DELETE("", appHandler.DeleteApp)
	app.POST("", appHandler.CreateApp)
	app.POST("/personal-workspace", appHandler.BootstrapPersonalWorkspace)
	app.POST("/builds", appHandler.UpdateApp)
	app.PATCH("/workspace", appHandler.UpdateWorkspace)

	teamAccess := apiV1.Group("/access")
	teamAccess.Use(middleware2.JWTAuth())
	teamAccessHandler := v1.NewTeamAccess(s.teamAccessService)
	teamAccess.GET("/members", teamAccessHandler.ListMembers)
	teamAccess.POST("/assignments", teamAccessHandler.Assign)
	teamAccess.POST("/assignments/batch", teamAccessHandler.BatchAssign)
	teamAccess.DELETE("/assignments", teamAccessHandler.Remove)
	teamAccess.GET("/permissions", teamAccessHandler.MyPermissions)

	publicShares := apiV1.Group("/public-shares")
	publicShares.Use(middleware2.JWTAuth())
	publicShares.POST("", publicShareHandler.Create)
	publicShares.GET("", publicShareHandler.List)
	publicShares.DELETE("/:share_id", publicShareHandler.Disable)

	// 服务目录管理路由（需要JWT验证）
	serviceTreeHandler := v1.NewServiceTree(s.serviceTreeService, s.teamAccessService)
	directoryResources := apiV1.Group("")
	directoryResources.Use(middleware2.JWTAuth())
	directoryResources.GET("/directories", serviceTreeHandler.GetServiceTreeDetail)
	directoryResources.POST("/directory-queries", serviceTreeHandler.BatchGetServiceTreeDetails)
	directoryResources.GET("/directory-overviews", serviceTreeHandler.GetDirectoryOverview)
	directoryResources.GET("/function-search-results", serviceTreeHandler.SearchFunctions)
	directoryResources.GET("/resource-search-results", serviceTreeHandler.SearchResources)
	directoryResources.POST("/directory-copies", serviceTreeHandler.CopyServiceTree)
	directoryResources.GET("/capability-bundle-exports", serviceTreeHandler.ExportCapabilityBundle)
	directoryResources.POST("/capability-bundle-exports", serviceTreeHandler.ExportCapabilityBundle)
	directoryResources.POST("/capability-bundle-installations", serviceTreeHandler.InstallCapabilityBundle)

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

	// ⭐ 文档管理路由（基于完整路径，与 table/form/chart 风格一致）
	docs := apiV1.Group("/docs")
	docs.Use(middleware2.JWTAuth())
	docHandler := v1.NewDoc(s.docService, s.teamAccessService)
	docs.GET("/search", docHandler.SearchDocs)                 // 搜索文档（模糊搜索）
	docs.GET("/batch", docHandler.BatchGetDocs)                // 批量获取文档（精确查询）
	docs.GET("/info/*full-code-path", docHandler.GetDoc)       // 获取文档
	docs.PUT("/info/*full-code-path", docHandler.UpdateDoc)    // 更新文档
	docs.DELETE("/info/*full-code-path", docHandler.DeleteDoc) // 删除文档

	// ==================== Agent 委托的内部调用 ====================
	// These routes intentionally do not accept end-user credentials. Agent must
	// pass the Gateway's exact allowlist; Gateway then re-signs the rewritten
	// backend request. Host-network Apps cannot substitute identity headers.
	agentDelegated := apiV1.Group("")
	agentDelegated.Use(middleware2.GatewayBackendAuth())
	agentDelegated.POST("/functions/batch", serviceTreeHandler.AddFunctions)

	workspaceGroup := agentDelegated.Group("/workspace")
	workspaceGroup.GET("/context", serviceTreeHandler.GetWorkspaceContext)       // 获取工作台环境信息（agent-server -> workspace）
	workspaceGroup.POST("/files/write", serviceTreeHandler.WriteFileContent)     // 工作台写入单个文本文件（实时写盘）
	workspaceGroup.POST("/files/replace", serviceTreeHandler.ReplaceFileContent) // 工作台文件 search-replace（实时写盘）
	workspaceGroup.POST("/files/delete", serviceTreeHandler.DeleteFile)          // 工作台删除文件（删磁盘+删节点）
	workspaceGroup.POST("/logs/read", serviceTreeHandler.ReadAppLog)             // 工作台读取应用日志（支持 version/keyword）

	// 函数管理路由（需要JWT验证）
	function := apiV1.Group("/function")
	function.Use(middleware2.JWTAuth()) // 函数管理需要JWT认证
	functionHandler := v1.NewFunction(s.functionService)
	function.GET("/info/:func-type/*full-code-path", functionHandler.GetFunction)

	// 操作日志路由（需要JWT验证）
	operateLog := apiV1.Group("/operate_log")
	operateLog.Use(middleware2.JWTAuth())                                           // JWT 认证
	operateLogHandler := v1.NewOperateLog(s.operateLogService, s.teamAccessService) // 查询统一操作日志
	operateLog.GET("/general", operateLogHandler.GetOperateLogs)                    // 查询通用操作日志

	// 目录更新历史路由（需要JWT验证）
	directoryUpdateHistory := apiV1.Group("/directory_update_history")
	directoryUpdateHistory.Use(middleware2.JWTAuth()) // 目录更新历史需要JWT认证
	directoryUpdateHistoryHandler := v1.NewDirectoryUpdateHistory(s.directoryUpdateHistoryService)
	directoryUpdateHistory.GET("/app_version", directoryUpdateHistoryHandler.GetAppVersionUpdateHistory) // 获取应用版本更新历史（App视角）
	directoryUpdateHistory.GET("/directory", directoryUpdateHistoryHandler.GetDirectoryUpdateHistory)    // 获取目录更新历史（目录视角）

	// ⭐ 标准接口路由（使用 full-code-path）
	standardAPI := v1.NewStandardAPI(s.appService, s.teamAccessService)

	// Workspace function resources. The function path remains the catch-all resource ID.
	tables := apiV1.Group("/tables")
	tables.Use(middleware2.JWTAuth())
	tables.GET("/*full-code-path", standardAPI.TableSearch)
	tables.POST("/*full-code-path", standardAPI.TableCreate)
	tables.PUT("/*full-code-path", standardAPI.TableUpdate)
	tables.DELETE("/*full-code-path", standardAPI.TableDelete)

	tableImportTemplates := apiV1.Group("/table-import-templates")
	tableImportTemplates.Use(middleware2.JWTAuth())
	tableImportTemplates.GET("/*full-code-path", standardAPI.TableTemplate)

	formSubmissions := apiV1.Group("/form-submissions")
	formSubmissions.Use(middleware2.JWTAuth())
	formSubmissions.POST("/*full-code-path", standardAPI.FormSubmit)

	// 工作台私有 runtime 接口（agent tool -> 当前 workspace app）
	pythonExecutions := apiV1.Group("/python-executions")
	pythonExecutions.Use(middleware2.JWTAuth())
	pythonExecutions.POST("/*full-code-path", standardAPI.RuntimePython)

	// Chart 函数接口
	charts := apiV1.Group("/charts")
	charts.Use(middleware2.JWTAuth())
	charts.GET("/*full-code-path", standardAPI.ChartQuery)

	// Callback 接口（不需要权限检查，因为这是内部回调）
	selectionOptions := apiV1.Group("/selection-options")
	selectionOptions.Use(middleware2.JWTAuth())
	selectionOptions.POST("/*full-code-path", standardAPI.CallbackOnSelectFuzzy)

	serverx.ApplyRouteRegistrars(serverx.ServiceAppServer, s.httpServer)
}
