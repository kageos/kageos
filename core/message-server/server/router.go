package server

import (
	middleware2 "github.com/kageos/kageos/pkg/middleware"
	"github.com/kageos/kageos/pkg/pprof"
)

func (s *Server) setupRoutes() {
	s.httpServer.GET("/health", s.healthHandler)
	s.httpServer.GET("/message/health", s.healthHandler)
	if s.cfg.IsPprofEnabled() {
		pprof.RegisterPprofRoutes(s.httpServer)
	}

	message := s.httpServer.Group("/message/api/v1")
	message.Use(middleware2.JWTAuth())
	message.GET("/health", s.messageAPIHealth)
	message.POST("/send", s.sendMessage)
	message.POST("/send/users", s.sendMessageToUsers)
	message.GET("/inbox", s.listInboxMessages)
	message.GET("/inbox/threads", s.listInboxThreads)
	message.GET("/inbox/unread_count", s.getUnreadCount)
	message.GET("/inbox/:id", s.getInboxMessage)
	message.PATCH("/inbox/read_all", s.markAllInboxMessagesRead)
	message.PATCH("/inbox/:id/read", s.markInboxMessageRead)
}
