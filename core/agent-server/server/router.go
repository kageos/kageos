package server

import (
	v1 "github.com/ai-agent-os/ai-agent-os/core/agent-server/api/v1"
	"github.com/ai-agent-os/ai-agent-os/pkg/pprof"
	middleware2 "github.com/ai-agent-os/ai-agent-os/pkg/middleware"
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
	agent.Use(middleware2.WithUserInfo())

	// 智能体管理路由
	agents := agent.Group("/agents")
	agentHandler := v1.NewAgent(s.agentService, s.cfg)
	agents.GET("/list", agentHandler.List)        // 获取智能体列表（前端调用）
	agents.GET("/get", agentHandler.Get)          // 获取智能体详情
	agents.POST("/create", agentHandler.Create)   // 创建智能体
	agents.POST("/update", agentHandler.Update)   // 更新智能体
	agents.POST("/delete", agentHandler.Delete)   // 删除智能体
	agents.POST("/enable", agentHandler.Enable)   // 启用智能体
	agents.POST("/disable", agentHandler.Disable) // 禁用智能体

	// 知识库管理路由
	knowledge := agent.Group("/knowledge")
	knowledgeHandler := v1.NewKnowledge(s.knowledgeService)
	knowledge.GET("/list", knowledgeHandler.List)                      // 获取知识库列表
	knowledge.GET("/get", knowledgeHandler.Get)                        // 获取知识库详情
	knowledge.POST("/create", knowledgeHandler.Create)                 // 创建知识库
	knowledge.POST("/update", knowledgeHandler.Update)                 // 更新知识库
	knowledge.POST("/delete", knowledgeHandler.Delete)                 // 删除知识库
	knowledge.POST("/add_document", knowledgeHandler.AddDocument)          // 添加文档
	knowledge.GET("/list_documents", knowledgeHandler.ListDocuments)      // 获取文档列表（平铺）
	knowledge.GET("/get_documents_tree", knowledgeHandler.GetDocumentsTree) // 获取文档树（目录结构）
	knowledge.GET("/get_document", knowledgeHandler.GetDocument)            // 获取文档详情
	knowledge.POST("/update_document", knowledgeHandler.UpdateDocument)    // 更新文档
	knowledge.POST("/update_documents_sort", knowledgeHandler.UpdateDocumentsSort) // 批量更新文档排序
	knowledge.POST("/delete_document", knowledgeHandler.DeleteDocument)     // 删除文档

	// LLM 配置管理路由
	llm := agent.Group("/llm")
	llmHandler := v1.NewLLM(s.llmService)
	llm.GET("/list", llmHandler.List)              // 获取LLM配置列表
	llm.GET("/get", llmHandler.Get)                // 获取LLM配置详情
	llm.GET("/get_default", llmHandler.GetDefault) // 获取默认LLM配置
	llm.POST("/create", llmHandler.Create)         // 创建LLM配置
	llm.POST("/update", llmHandler.Update)         // 更新LLM配置
	llm.POST("/delete", llmHandler.Delete)         // 删除LLM配置
	llm.POST("/set_default", llmHandler.SetDefault) // 设置默认LLM配置

	// 插件管理路由
	plugins := agent.Group("/plugins")
	pluginHandler := v1.NewPlugin(s.pluginService, s.cfg)
	plugins.GET("/list", pluginHandler.List)              // 获取插件列表
	plugins.GET("/:id", pluginHandler.Get)                // 获取插件详情
	plugins.POST("", pluginHandler.Create)                // 创建插件
	plugins.PUT("/:id", pluginHandler.Update)             // 更新插件
	plugins.DELETE("/:id", pluginHandler.Delete)          // 删除插件
	plugins.POST("/:id/enable", pluginHandler.Enable)     // 启用插件
	plugins.POST("/:id/disable", pluginHandler.Disable)    // 禁用插件

	// 智能体聊天路由（按 chat_type 区分）
	agentChatHandler := v1.NewAgentChat(s.agentChatService)
	agent.POST("/chat/function_gen", agentChatHandler.FunctionGenChat)              // 智能体聊天 - 函数生成类型
	agent.GET("/chat/function_gen/status", agentChatHandler.GetFunctionGenStatus)    // 查询代码生成状态
	agent.GET("/chat/sessions", agentChatHandler.ListSessions)                      // 获取会话列表
	agent.GET("/chat/messages", agentChatHandler.ListMessages)                      // 获取消息列表
}
