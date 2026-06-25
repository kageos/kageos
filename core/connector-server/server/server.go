package server

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	v1 "github.com/kageos/kageos/core/connector-server/api/v1"
	"github.com/kageos/kageos/core/connector-server/model"
	"github.com/kageos/kageos/core/connector-server/repository"
	"github.com/kageos/kageos/core/connector-server/service"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/dbx"
	"github.com/kageos/kageos/pkg/logger"
	middleware2 "github.com/kageos/kageos/pkg/middleware"
	"github.com/kageos/kageos/pkg/pprof"
	"github.com/kageos/kageos/pkg/serverx"
	"gorm.io/gorm"
)

type Server struct {
	cfg *config.ConnectorServerConfig

	db         *gorm.DB
	httpServer *gin.Engine

	connectorService *service.ConnectorService

	ctx context.Context
}

func NewServer(cfg *config.ConnectorServerConfig) (*Server, error) {
	ctx := context.Background()
	s := &Server{cfg: cfg, ctx: ctx}
	if err := s.initDatabase(ctx); err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}
	if err := s.initServices(ctx); err != nil {
		return nil, fmt.Errorf("failed to init services: %w", err)
	}
	if err := s.initRouter(ctx); err != nil {
		return nil, fmt.Errorf("failed to init router: %w", err)
	}
	return s, nil
}

func (s *Server) Start(ctx context.Context) error {
	addr := net.JoinHostPort(s.cfg.GetListenHost(), strconv.Itoa(s.cfg.GetPort()))
	logger.Infof(ctx, "[Server] connector-server HTTP starting on %s", addr)
	go func() {
		if err := s.httpServer.Run(addr); err != nil {
			logger.Errorf(ctx, "[Server] connector-server HTTP error: %v", err)
		}
	}()
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Stopping connector-server...")
	if s.db != nil {
		if sqlDB, err := s.db.DB(); err == nil {
			sqlDB.Close()
		}
	}
	logger.Infof(ctx, "[Server] connector-server stopped")
	return nil
}

func (s *Server) initDatabase(ctx context.Context) error {
	dbCfg := s.cfg.GetDB()
	if dbCfg.Type == "" {
		dbCfg.Type = "mysql"
	}
	if dbCfg.Type != "mysql" {
		return fmt.Errorf("unsupported database type: %s", dbCfg.Type)
	}
	db, err := dbx.OpenMySQL(dbCfg, dbx.OpenOptions{
		DefaultMaxLifetime: 5 * time.Minute,
	})
	if err != nil {
		return err
	}
	s.db = db
	if err := model.InitTables(db); err != nil {
		return err
	}
	logger.Infof(ctx, "[Server] connector database initialized")
	return nil
}

func (s *Server) initServices(ctx context.Context) error {
	repo := repository.NewConnectorRepository(s.db)
	tokenSecret := s.cfg.GetOAuth().TokenEncryptionSecret
	if tokenSecret == "" {
		tokenSecret = config.GetGlobalSharedConfig().JWT.Secret
	}
	s.connectorService = service.NewConnectorService(repo, service.WithOAuthConfig(s.cfg.GetOAuth(), tokenSecret))
	if err := s.connectorService.SeedOAuthProviderSettings(ctx); err != nil {
		return err
	}
	logger.Infof(ctx, "[Server] connector services initialized")
	return nil
}

func (s *Server) initRouter(ctx context.Context) error {
	s.httpServer = serverx.NewGin(
		serverx.WithRecovery(),
		serverx.WithMiddleware(middleware2.Cors()),
		serverx.WithRegisteredMiddlewares(serverx.ServiceConnectorServer),
	)
	s.httpServer.GET("/health", s.healthHandler)
	if s.cfg.IsPprofEnabled() {
		pprof.RegisterPprofRoutes(s.httpServer)
	}

	connectorHandler := v1.NewConnectorAPI(s.connectorService)
	s.httpServer.GET("/connector/oauth/callback", connectorHandler.OAuthCallback)
	s.httpServer.GET("/connector/oauth/callback/:provider", connectorHandler.OAuthCallback)

	apiV1 := s.httpServer.Group("/connector/api/v1")
	apiV1.Use(middleware2.JWTAuth())
	apiV1.POST("/oauth/authorize", connectorHandler.StartOAuth)
	apiV1.GET("/oauth/providers", connectorHandler.ListOAuthProviders)
	apiV1.GET("/oauth/providers/:provider", connectorHandler.GetOAuthProvider)
	apiV1.PUT("/oauth/providers/:provider", connectorHandler.UpsertOAuthProvider)
	apiV1.DELETE("/oauth/providers/:provider", connectorHandler.DeleteOAuthProvider)
	apiV1.POST("/connections/:connection_id/refresh", connectorHandler.RefreshOAuthToken)
	apiV1.POST("/connections/:connection_id/revoke", connectorHandler.RevokeConnection)
	apiV1.GET("/connections", connectorHandler.ListConnections)
	apiV1.POST("/connections", connectorHandler.CreateConnection)
	apiV1.DELETE("/connections/:connection_id", connectorHandler.DeleteConnection)
	apiV1.GET("/directory_bindings", connectorHandler.ListDirectoryBindings)
	apiV1.POST("/directory_bindings", connectorHandler.BindDirectory)
	apiV1.DELETE("/directory_bindings", connectorHandler.DeleteDirectoryBinding)
	apiV1.GET("/resolve", connectorHandler.ResolveDirectoryBinding)
	apiV1.POST("/proxy", connectorHandler.Proxy)
	serverx.ApplyRouteRegistrars(serverx.ServiceConnectorServer, s.httpServer)
	logger.Infof(ctx, "[Server] connector routes initialized")
	return nil
}

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Format(time.DateTime),
		"service":   "connector-server",
	})
}
