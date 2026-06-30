package server

import (
	middleware2 "github.com/kageos/kageos/pkg/middleware"
	"github.com/kageos/kageos/pkg/pprof"
	"github.com/kageos/kageos/pkg/serverx"
)

func (s *Server) setupRoutes() {
	s.httpServer.GET("/health", s.healthHandler)
	s.httpServer.GET("/message/health", s.healthHandler)
	if s.cfg.IsPprofEnabled() {
		pprof.RegisterPprofRoutes(s.httpServer)
	}

	publicMessage := s.httpServer.Group("/message/api/v1/public")
	publicMessage.Use(middleware2.JWTAuthOptional())
	publicMessage.GET("/actions/:token", s.getPublicMessageAction)
	publicMessage.POST("/actions/:token/reply", s.submitPublicMessageActionReply)

	message := s.httpServer.Group("/message/api/v1")
	message.Use(middleware2.JWTAuth())
	message.GET("/health", s.messageAPIHealth)
	message.POST("/send", s.sendMessage)
	message.POST("/send/users", s.sendMessageToUsers)
	message.GET("/notification_channels", s.listNotificationChannels)
	message.PUT("/notification_channels/:channel", s.upsertNotificationChannel)
	message.DELETE("/notification_channels/:channel", s.deleteNotificationChannel)
	message.POST("/notification_channels/:channel/test", s.testNotificationChannel)
	message.GET("/notification_routes", s.listNotificationRoutes)
	message.PUT("/notification_routes/:channel", s.upsertNotificationRoute)
	message.DELETE("/notification_routes/:channel", s.deleteNotificationRoute)
	message.POST("/notification_routes/:channel/test", s.testNotificationRoute)
	message.GET("/inbox", s.listInboxMessages)
	message.GET("/inbox/threads", s.listInboxThreads)
	message.GET("/inbox/source_counts", s.listInboxSourceCounts)
	message.GET("/inbox/workspace_counts", s.listInboxWorkspaceCounts)
	message.GET("/inbox/unread_count", s.getUnreadCount)
	message.GET("/inbox/:id", s.getInboxMessage)
	message.PATCH("/inbox/read_all", s.markAllInboxMessagesRead)
	message.PATCH("/inbox/:id/read", s.markInboxMessageRead)

	serverx.ApplyRouteRegistrars(serverx.ServiceMessageServer, s.httpServer)
}
