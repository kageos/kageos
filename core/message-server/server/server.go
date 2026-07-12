package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	msgmodel "github.com/kageos/kageos/core/message-server/model"
	msgrepo "github.com/kageos/kageos/core/message-server/repository"
	"github.com/kageos/kageos/core/message-server/service"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/kageos/kageos/pkg/dbx"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/natsx"
	"github.com/kageos/kageos/pkg/serverx"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

type Server struct {
	cfg        *config.MessageServerConfig
	db         *gorm.DB
	natsConn   *nats.Conn
	httpServer *gin.Engine
	httpSrv    *serverx.HTTPServer

	messageRepo            *msgrepo.MessageRepository
	messageConsumerService *service.MessageConsumerService
	messageCommandHandler  *service.MessageCommandHandler
	workspaceActionRunner  workspaceActionSubmitter
	notificationVault      *service.NotificationSecretVault
	subscriptions          []*nats.Subscription
}

type workspaceActionSubmitter interface {
	Submit(ctx context.Context, req service.WorkspaceActionRequest) (*service.WorkspaceActionSubmitResult, error)
}

func NewServer(cfg *config.MessageServerConfig) (*Server, error) {
	if cfg == nil {
		cfg = &config.MessageServerConfig{}
	}
	s := &Server{
		cfg:           cfg,
		subscriptions: make([]*nats.Subscription, 0),
	}
	if err := s.initDatabase(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}
	if err := s.initNATS(context.Background()); err != nil {
		if cfg.AllowNATSDegradedStartup() {
			logger.Warnf(context.Background(), "[message-server] NATS unavailable: %v, continuing because allow_nats_degraded_startup=true", err)
		} else {
			return nil, fmt.Errorf("failed to initialize required NATS: %w", err)
		}
	}
	if err := s.initServices(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to init services: %w", err)
	}
	s.initRouter()
	return s, nil
}

func (s *Server) Start(ctx context.Context) error {
	logger.Infof(ctx, "[message-server] starting...")
	if err := s.subscribeNATS(ctx); err != nil {
		if s.cfg.AllowNATSDegradedStartup() {
			logger.Warnf(ctx, "[message-server] NATS subscribe failed: %v, continuing because allow_nats_degraded_startup=true", err)
		} else {
			return fmt.Errorf("failed to subscribe required NATS: %w", err)
		}
	}

	addr := net.JoinHostPort(s.cfg.GetListenHost(), strconv.Itoa(s.cfg.GetPort()))
	httpSrv, err := serverx.StartHTTPServer(ctx, addr, s.httpServer)
	if err != nil {
		s.unsubscribeNATS(ctx)
		return fmt.Errorf("failed to start HTTP server on %s: %w", addr, err)
	}
	s.httpSrv = httpSrv
	go func() {
		if err := <-httpSrv.Err(); err != nil {
			logger.Errorf(ctx, "[message-server] HTTP server error: %v", err)
		}
	}()
	logger.Infof(ctx, "[message-server] started on %s", addr)
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	logger.Infof(ctx, "[message-server] stopping...")
	var stopErr error
	if s.httpSrv != nil {
		if err := s.httpSrv.Shutdown(ctx); err != nil {
			logger.Warnf(ctx, "[message-server] HTTP server shutdown failed: %v", err)
			stopErr = err
		}
		s.httpSrv = nil
	}
	s.unsubscribeNATS(ctx)
	if s.natsConn != nil {
		s.natsConn.Close()
		s.natsConn = nil
	}
	if err := closeDB(s.db); err != nil {
		return err
	}
	logger.Infof(ctx, "[message-server] stopped")
	return stopErr
}

func (s *Server) initDatabase(ctx context.Context) error {
	messageDB, err := openMessageDB(s.cfg.GetDB())
	if err != nil {
		return err
	}
	if err := msgmodel.InitModels(messageDB); err != nil {
		_ = closeDB(messageDB)
		return fmt.Errorf("failed to migrate message models: %w", err)
	}
	s.db = messageDB

	logger.Infof(ctx, "[message-server] database connected: message=%s", s.cfg.GetDB().Name)
	return nil
}

func openMessageDB(dbCfg config.DBConfig) (*gorm.DB, error) {
	switch strings.ToLower(strings.TrimSpace(dbCfg.Type)) {
	case "", "mysql":
		if err := dbx.EnsureMySQLDatabase(dbCfg); err != nil {
			return nil, err
		}
		return dbx.OpenMySQL(dbCfg, dbx.OpenOptions{DefaultMaxLifetime: time.Hour})
	case "sqlite":
		return dbx.OpenSQLite(dbCfg.Name, dbx.OpenOptions{})
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbCfg.Type)
	}
}

func (s *Server) initNATS(ctx context.Context) error {
	natsURL := strings.TrimSpace(config.GetGlobalSharedConfig().Nats.URL)
	if natsURL == "" {
		natsURL = "nats://127.0.0.1:4222"
	}
	conn, err := natsx.ConnectNamed(natsURL, "message-server")
	if err != nil {
		return err
	}
	s.natsConn = conn
	logger.Infof(ctx, "[message-server] NATS initialized")
	return nil
}

func (s *Server) initServices(ctx context.Context) error {
	controlPlaneSecret, err := config.GetControlPlaneSecret()
	if err != nil {
		return fmt.Errorf("load control-plane secret: %w", err)
	}
	workspaceActionSigner, err := controlauth.NewSigner(controlPlaneSecret, controlauth.HTTPWorkspaceActionScope)
	if err != nil {
		return fmt.Errorf("initialize workspace-action request signer: %w", err)
	}
	vault, err := service.NewNotificationSecretVault(s.cfg.GetNotificationEncryptionSecret())
	if err != nil {
		return fmt.Errorf("failed to init notification secret vault: %w", err)
	}
	s.notificationVault = vault
	s.messageRepo = msgrepo.NewMessageRepository(s.db)
	targetResolver := service.NewUserNotificationTargetResolver(s.messageRepo, s.notificationVault)
	s.messageConsumerService = service.NewMessageConsumerService(
		s.messageRepo,
		service.WithNotificationCardBaseURL(config.GetPublicSiteBaseURL()),
		service.WithNotificationTargetResolver(targetResolver),
		service.WithChannelProviders(service.NewDefaultNotificationChannelProviders(s.cfg.GetNotificationWebhookTimeout())...),
	)
	s.messageCommandHandler = service.NewMessageCommandHandler(s.messageConsumerService)
	s.workspaceActionRunner = service.NewWorkspaceActionRunner(
		config.GetGlobalSharedConfig().Gateway.GetInternalURL(),
		workspaceActionSigner,
	)
	logger.Infof(ctx, "[message-server] services initialized")
	return nil
}

func (s *Server) initRouter() {
	s.httpServer = serverx.NewGin(
		serverx.WithDebug(s.cfg.IsDebug()),
		serverx.WithRegisteredMiddlewares(serverx.ServiceMessageServer),
	)
	s.setupRoutes()
}

func (s *Server) subscribeNATS(ctx context.Context) error {
	if s.natsConn == nil || s.messageCommandHandler == nil {
		return nil
	}
	return RegisterNATS(ctx, s.natsConn, &s.subscriptions, s.messageCommandHandler)
}

func (s *Server) unsubscribeNATS(ctx context.Context) {
	for _, sub := range s.subscriptions {
		if sub == nil {
			continue
		}
		if err := sub.Unsubscribe(); err != nil {
			logger.Warnf(ctx, "[message-server] unsubscribe %s failed: %v", sub.Subject, err)
		}
	}
	s.subscriptions = s.subscriptions[:0]
}

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "message-server",
	})
}

func closeDB(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil
	}
	return sqlDB.Close()
}
