package server

import (
	v1 "github.com/ai-agent-os/ai-agent-os/core/agent-server/api/v1"
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

	// Agent 路由组（统一使用 /agent/api/v1 开头，方便网关代理）
	agent := s.httpServer.Group("/agent")

	// API v1 路由组
	apiV1 := agent.Group("/api/v1")

	// 添加用户信息中间件
	apiV1.Use(middleware2.WithUserInfo())

	// 智能体管理路由
	agents := apiV1.Group("/agents")
	agentHandler := v1.NewAgent(s.agentService, s.cfg)
	agents.GET("/list", agentHandler.List)        // 获取智能体列表（前端调用）
	agents.GET("/get", agentHandler.Get)          // 获取智能体详情
	agents.POST("/create", agentHandler.Create)   // 创建智能体
	agents.POST("/update", agentHandler.Update)   // 更新智能体
	agents.POST("/delete", agentHandler.Delete)   // 删除智能体
	agents.POST("/enable", agentHandler.Enable)   // 启用智能体
	agents.POST("/disable", agentHandler.Disable) // 禁用智能体

	// 知识库管理路由已废弃，使用服务树管理文档

	// LLM 配置管理路由
	llm := apiV1.Group("/llm")
	llmHandler := v1.NewLLM(s.llmService)
	llm.GET("/list", llmHandler.List)               // 获取LLM配置列表
	llm.GET("/get", llmHandler.Get)                 // 获取LLM配置详情
	llm.GET("/get_default", llmHandler.GetDefault)  // 获取默认LLM配置
	llm.POST("/create", llmHandler.Create)          // 创建LLM配置
	llm.POST("/update", llmHandler.Update)          // 更新LLM配置
	llm.POST("/delete", llmHandler.Delete)          // 删除LLM配置
	llm.POST("/set_default", llmHandler.SetDefault) // 设置默认LLM配置

	// 插件管理路由
	plugins := apiV1.Group("/plugins")
	pluginHandler := v1.NewPlugin(s.pluginService, s.cfg)
	plugins.GET("/list", pluginHandler.List)            // 获取插件列表
	plugins.GET("/:id", pluginHandler.Get)              // 获取插件详情
	plugins.POST("", pluginHandler.Create)              // 创建插件
	plugins.PUT("/:id", pluginHandler.Update)           // 更新插件
	plugins.DELETE("/:id", pluginHandler.Delete)        // 删除插件
	plugins.POST("/:id/enable", pluginHandler.Enable)   // 启用插件
	plugins.POST("/:id/disable", pluginHandler.Disable) // 禁用插件

	// 智能体聊天路由（按 chat_type 区分）
	agentChatHandler := v1.NewAgentChat(s.agentChatService)
	chat := apiV1.Group("/chat")
	chat.POST("/function_gen", agentChatHandler.FunctionGenChat)            // 智能体聊天 - 函数生成类型
	chat.GET("/function_gen/status", agentChatHandler.GetFunctionGenStatus) // 查询代码生成状态
	chat.GET("/sessions", agentChatHandler.ListSessions)                    // 获取会话列表
	chat.GET("/messages", agentChatHandler.ListMessages)                    // 获取消息列表

	// 工作空间相关路由（服务间调用，不需要JWT验证，但需要用户信息中间件）
	workspace := apiV1.Group("/workspace")
	workspaceHandler := v1.NewFunctionGen(s.functionGenService)
	workspace.POST("/update/callback", workspaceHandler.ReceiveCallback) // 接收工作空间更新回调（app-server -> agent-server）

	// 智能工作台（Step 3 接入 WorkspaceChatService）
	workspaceChatHandler := v1.NewWorkspace(s.toolRegistry, s.modeRepo, s.workspaceChatService)
	workspace.GET("/tools", workspaceChatHandler.ListTools)           // 列出工具
	workspace.GET("/tools/names", workspaceChatHandler.ListToolNames) // 工具名列表（供模式配置多选）
	workspace.POST("/call_tool", workspaceChatHandler.CallTool)       // 执行工具（临时，验证 call_tool）
	workspace.GET("/sessions", workspaceChatHandler.ListSessions)    // 获取会话列表（根据 full_code_path）
	workspace.GET("/messages", workspaceChatHandler.ListMessages)    // 获取会话消息列表（根据 session_id）
	workspace.POST("/chat/stream", workspaceChatHandler.ChatStream)
	// 工作台模式 CRUD
	workspace.GET("/modes", workspaceChatHandler.ListModes)                    // 模式列表
	workspace.GET("/modes/by-code", workspaceChatHandler.GetModeByCode)        // 按 code 获取模式
	workspace.GET("/modes/:id", workspaceChatHandler.GetMode)                   // 按 id 获取模式
	workspace.POST("/modes", workspaceChatHandler.CreateMode)                   // 创建模式
	workspace.PUT("/modes/:id", workspaceChatHandler.UpdateMode)                // 更新模式
	workspace.DELETE("/modes/:id", workspaceChatHandler.DeleteMode)              // 删除模式（内置不可删）
}
