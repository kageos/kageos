package server

import middleware2 "github.com/ai-agent-os/ai-agent-os/pkg/middleware"

func (s *Server) setupRoutes() {
	s.httpServer.GET("/health", s.healthHandler)
	s.httpServer.GET("/message/health", s.healthHandler)

	message := s.httpServer.Group("/message/api/v1")
	message.Use(middleware2.JWTAuth())
	message.GET("/health", s.messageAPIHealth)
	message.POST("/send", s.sendMessage)
	message.POST("/send/users", s.sendMessageToUsers)
	message.POST("/send/departments", s.sendMessageToDepartments)
	message.GET("/inbox", s.listInboxMessages)
	message.GET("/inbox/unread_count", s.getUnreadCount)
	message.GET("/inbox/:id", s.getInboxMessage)
	message.PATCH("/inbox/read_all", s.markAllInboxMessagesRead)
	message.PATCH("/inbox/:id/read", s.markInboxMessageRead)
}
