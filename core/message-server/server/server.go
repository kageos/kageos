package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	hrrepo "github.com/ai-agent-os/ai-agent-os/core/hr-server/repository"
	msgmodel "github.com/ai-agent-os/ai-agent-os/core/message-server/model"
	msgrepo "github.com/ai-agent-os/ai-agent-os/core/message-server/repository"
	"github.com/ai-agent-os/ai-agent-os/core/message-server/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/dbx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/natsx"
	"github.com/ai-agent-os/ai-agent-os/pkg/serverx"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

// Server 系统通知/业务消息服务。
type Server struct {
	cfg        *config.MessageServerConfig
	db         *gorm.DB
	natsConn   *nats.Conn
	httpServer *gin.Engine
	httpSrv    *http.Server

	messageRepo            *msgrepo.MessageRepository
	messageConsumerService *service.MessageConsumerService
	messageCommandHandler  *service.MessageCommandHandler
	subscriptions          []*nats.Subscription
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
	s.httpSrv = &http.Server{
		Addr:              addr,
		Handler:           s.httpServer,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf(ctx, "[message-server] HTTP server error: %v", err)
		}
	}()
	logger.Infof(ctx, "[message-server] started on %s", addr)
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	logger.Infof(ctx, "[message-server] stopping...")
	s.unsubscribeNATS(ctx)
	if s.natsConn != nil {
		s.natsConn.Close()
		s.natsConn = nil
	}
	if s.httpSrv != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpSrv.Shutdown(shutdownCtx); err != nil {
			return err
		}
	}
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err == nil {
			if err := sqlDB.Close(); err != nil {
				return err
			}
		}
	}
	logger.Infof(ctx, "[message-server] stopped")
	return nil
}

func (s *Server) initDatabase(ctx context.Context) error {
	dbCfg := s.cfg.GetDB()
	if dbCfg.Type == "mysql" || dbCfg.Type == "" {
		db, err := dbx.OpenMySQL(dbCfg, dbx.OpenOptions{
			DefaultMaxLifetime: time.Hour,
		})
		if err != nil {
			return err
		}
		if err := msgmodel.InitModels(db); err != nil {
			return fmt.Errorf("failed to migrate message models: %w", err)
		}
		s.db = db
		logger.Infof(ctx, "[message-server] database connected: %s", dbCfg.Name)
		return nil
	}
	return fmt.Errorf("unsupported database type: %s", dbCfg.Type)
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
	userRepo := hrrepo.NewUserRepository(s.db)
	s.messageRepo = msgrepo.NewMessageRepository(s.db)
	s.messageConsumerService = service.NewMessageConsumerService(userRepo, s.messageRepo, s.cfg.GetEmail())
	s.messageCommandHandler = service.NewMessageCommandHandler(s.messageConsumerService)
	logger.Infof(ctx, "[message-server] services initialized")
	return nil
}

func (s *Server) initRouter() {
	s.httpServer = serverx.NewGin(serverx.WithDebug(s.cfg.IsDebug()))
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
