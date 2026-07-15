package server

import (
	v1 "github.com/kageos/kageos/core/agent-server/api/v1"
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

	// Agent 路由组（统一使用 /agent/api/v1 开头，方便网关代理）
	agent := s.httpServer.Group("/agent")

	// API v1 路由组
	apiV1 := agent.Group("/api/v1")

	// 添加用户信息中间件
	apiV1.Use(middleware2.WithUserInfo())

	// 服务端运行态。当前由 agent-server 内存维护，后续可替换为分布式实现。
	state := apiV1.Group("/state")
	stateHandler := v1.NewState(s.runtimeStateStore)
	state.GET("/runtime-summary", stateHandler.RuntimeSummary)
	state.GET("/runtime-items", stateHandler.RuntimeItems)

	// LLM 配置管理路由
	llm := apiV1.Group("/llm")
	llmHandler := v1.NewLLM(s.llmService)
	llm.GET("/list", llmHandler.List)               // 获取LLM配置列表
	llm.GET("/get", llmHandler.Get)                 // 获取LLM配置详情
	llm.GET("/get_default", llmHandler.GetDefault)  // 获取默认LLM配置
	llm.POST("/create", llmHandler.Create)          // 创建LLM配置
	llm.POST("/probe", llmHandler.Probe)            // 检测LLM协议
	llm.POST("/update", llmHandler.Update)          // 更新LLM配置
	llm.POST("/delete", llmHandler.Delete)          // 删除LLM配置
	llm.POST("/set_default", llmHandler.SetDefault) // 设置默认LLM配置

	// 智能工作台（只认 LLM，单模式）
	workspace := apiV1.Group("/workspace")
	workspaceChatHandler := v1.NewWorkspace(s.toolRegistry, s.workspaceChatService)
	workspace.GET("/sessions", workspaceChatHandler.ListSessions)                                   // 获取会话列表
	workspace.POST("/sessions/handoff", workspaceChatHandler.CreateSessionHandoff)                  // 创建阶段交接会话
	workspace.POST("/sessions/interaction/resolve", workspaceChatHandler.ResolvePendingInteraction) // 清除待交互状态
	workspace.POST("/sessions/interaction/event", workspaceChatHandler.RecordInteractionEvent)      // 记录交互卡片事件
	workspace.GET("/sessions/running", workspaceChatHandler.ListRunningSessions)                    // 查询执行中的任务
	workspace.GET("/sessions/finished", workspaceChatHandler.ListFinishedSessions)                  // 查询已结束的任务
	workspace.GET("/sessions/:session_id/sse-status", workspaceChatHandler.GetSessionSSEStatus)     // SSE 存活检测
	workspace.GET("/messages", workspaceChatHandler.ListMessages)                                   // 获取会话消息列表
	workspace.POST("/tools/batch_detail", workspaceChatHandler.BatchToolDetails)                    // 批量获取内置工具展示详情
	workspace.POST("/chat/stream", workspaceChatHandler.ChatStream)
	workspace.POST("/chat/cancel", workspaceChatHandler.CancelChat) // 取消执行中的任务

	serverx.ApplyRouteRegistrars(serverx.ServiceAgentServer, s.httpServer)
}
