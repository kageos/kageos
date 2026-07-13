package server

import (
	middleware2 "github.com/kageos/kageos/pkg/middleware"
	"github.com/kageos/kageos/pkg/pprof"
	"github.com/kageos/kageos/pkg/serverx"
)

func (s *Server) setupRoutes() {
	s.httpServer.GET("/health", s.healthHandler)
	s.httpServer.GET("/message/health", s.healthHandler)
	s.httpServer.GET("/message/api/v1/health", s.messageAPIHealth)
	if s.cfg.IsPprofEnabled() {
		pprof.RegisterPprofRoutes(s.httpServer)
	}

	publicMessage := s.httpServer.Group("/message/api/v1/public")
	publicMessage.Use(middleware2.StrictCredentialAuth())
	publicMessage.GET("/actions/:token", s.getPublicMessageAction)
	publicMessage.POST("/actions/:token/reply", s.submitPublicMessageActionReply)

	message := s.httpServer.Group("/message/api/v1")
	message.Use(middleware2.StrictCredentialAuth())
	message.POST("/send", s.sendMessage)
	message.POST("/send/users", s.sendMessageToUsers)
	message.GET("/notification-channels", s.listNotificationChannels)
	message.PUT("/notification-channels/:channel", s.upsertNotificationChannel)
	message.DELETE("/notification-channels/:channel", s.deleteNotificationChannel)
	message.POST("/notification-channels/:channel/test", s.testNotificationChannel)
	message.GET("/notification-routes", s.listNotificationRoutes)
	message.GET("/notification-routes/summary", s.listNotificationRouteSummary)
	message.PUT("/notification-routes/:channel", s.upsertNotificationRoute)
	message.DELETE("/notification-routes/:channel", s.deleteNotificationRoute)
	message.POST("/notification-routes/:channel/test", s.testNotificationRoute)
	message.GET("/inbox", s.listInboxMessages)
	message.GET("/inbox/threads", s.listInboxThreads)
	message.GET("/inbox/source-counts", s.listInboxSourceCounts)
	message.GET("/inbox/workspace-counts", s.listInboxWorkspaceCounts)
	message.GET("/inbox/unread-count", s.getUnreadCount)
	message.GET("/inbox/:id", s.getInboxMessage)
	message.PATCH("/inbox/read-all", s.markAllInboxMessagesRead)
	message.PATCH("/inbox/read-source", s.markSourceInboxMessagesRead)
	message.PATCH("/inbox/:id/read", s.markInboxMessageRead)

	serverx.ApplyRouteRegistrars(serverx.ServiceMessageServer, s.httpServer)
}
