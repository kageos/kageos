package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos/core/timer-scheduler/model"
	timerservice "github.com/kageos/kageos/core/timer-scheduler/service"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/dbx"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/natsx"
	"github.com/kageos/kageos/pkg/serverx"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

type Server struct {
	cfg        *config.TimerSchedulerConfig
	db         *gorm.DB
	natsConn   *nats.Conn
	service    *timerservice.Service
	publisher  timerservice.OutboxPublisher
	natsSubs   []*nats.Subscription
	httpServer *serverx.HTTPServer
	done       chan struct{}
	cancel     context.CancelFunc
	owner      string
}

func NewServer(cfg *config.TimerSchedulerConfig) (*Server, error) {
	if cfg == nil {
		cfg = &config.TimerSchedulerConfig{}
	}
	db, err := openTimerSchedulerDB(cfg)
	if err != nil {
		return nil, err
	}
	if err := model.InitTables(db); err != nil {
		return nil, fmt.Errorf("failed to migrate timer-scheduler database: %w", err)
	}
	natsConn, err := openTimerSchedulerNATS()
	if err != nil {
		return nil, err
	}
	service := timerservice.NewService(db, timerservice.Options{
		DispatchLeaseDuration:  cfg.GetDispatchLeaseDuration(),
		ExecutionLeaseDuration: cfg.GetExecutionLeaseDuration(),
		QueueAckTimeout:        cfg.GetQueueAckTimeout(),
		MaxDispatchAttempts:    cfg.GetMaxDispatchAttempts(),
		MaxHeartbeatMisses:     cfg.GetMaxHeartbeatMisses(),
		MaxOutboxAttempts:      cfg.GetMaxOutboxAttempts(),
		PayloadLimitBytes:      cfg.GetPayloadLimitBytes(),
	})
	return &Server{
		cfg:       cfg,
		db:        db,
		natsConn:  natsConn,
		service:   service,
		publisher: timerservice.NewNATSOutboxPublisher(natsConn),
		owner:     newOwnerID(),
	}, nil
}

func (s *Server) Start(ctx context.Context) error {
	if s.done != nil {
		return fmt.Errorf("timer-scheduler already started")
	}
	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	natsSubs, err := startTimerNATSControl(s.natsConn, s.service)
	if err != nil {
		cancel()
		return err
	}
	s.natsSubs = natsSubs
	addr := net.JoinHostPort(s.cfg.GetListenHost(), strconv.Itoa(s.cfg.GetPort()))
	httpServer, err := serverx.StartHTTPServer(ctx, addr, NewRouter(s.service))
	if err != nil {
		cancel()
		for _, sub := range natsSubs {
			_ = sub.Unsubscribe()
		}
		s.natsSubs = nil
		return fmt.Errorf("failed to start HTTP server on %s: %w", addr, err)
	}
	s.httpServer = httpServer
	s.done = make(chan struct{})
	go s.runDispatchLoop(runCtx)
	go func() {
		if err := <-httpServer.Err(); err != nil {
			logger.Errorf(runCtx, "[timer-scheduler] HTTP server error: %v", err)
		}
	}()
	logger.Infof(ctx, "[timer-scheduler] started on %s", addr)
	logger.Infof(ctx, "[timer-scheduler] nats execution control started")
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			return err
		}
		s.httpServer = nil
	}
	if s.done != nil {
		<-s.done
		s.done = nil
	}
	for _, sub := range s.natsSubs {
		_ = sub.Unsubscribe()
	}
	s.natsSubs = nil
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err == nil {
			if err := sqlDB.Close(); err != nil {
				return err
			}
		}
	}
	if s.natsConn != nil {
		s.natsConn.Close()
		s.natsConn = nil
	}
	return nil
}

func (s *Server) Service() *timerservice.Service {
	return s.service
}

func (s *Server) runDispatchLoop(ctx context.Context) {
	defer close(s.done)
	interval := s.cfg.GetSchedulerPollInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if _, err := s.service.RecoverStaleExecutions(ctx, s.cfg.GetSchedulerBatchSize()); err != nil {
			logger.Warnf(ctx, "[timer-scheduler] recover stale executions failed: %v", err)
		}
		if _, err := s.service.DispatchDue(ctx, s.owner, s.cfg.GetSchedulerBatchSize()); err != nil {
			logger.Warnf(ctx, "[timer-scheduler] dispatch due failed: %v", err)
		}
		if s.publisher != nil {
			if _, err := s.service.PublishPendingOutbox(ctx, s.publisher, s.cfg.GetSchedulerBatchSize()); err != nil {
				logger.Warnf(ctx, "[timer-scheduler] publish outbox failed: %v", err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func openTimerSchedulerNATS() (*nats.Conn, error) {
	natsURL := strings.TrimSpace(config.GetGlobalSharedConfig().Nats.URL)
	if natsURL == "" {
		return nil, fmt.Errorf("timer-scheduler nats url is required")
	}
	conn, err := natsx.ConnectNamed(natsURL, "timer-scheduler")
	if err != nil {
		return nil, fmt.Errorf("failed to connect timer-scheduler nats: %w", err)
	}
	return conn, nil
}

func openTimerSchedulerDB(cfg *config.TimerSchedulerConfig) (*gorm.DB, error) {
	dbCfg := cfg.GetDB()
	switch dbCfg.Type {
	case "mysql":
		if err := dbx.EnsureMySQLDatabase(dbCfg); err != nil {
			logger.Warnf(context.Background(), "[timer-scheduler] ensure database failed: name=%s err=%v", dbCfg.Name, err)
		}
		db, err := dbx.OpenMySQL(dbCfg, dbx.OpenOptions{
			DisableForeignKeyConstraintWhenMigrating: true,
			DefaultMaxLifetime:                       5 * time.Minute,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to connect timer-scheduler mysql: %w", err)
		}
		return db, nil
	case "sqlite":
		name := dbCfg.Name
		if name == "" {
			name = "data/timer-scheduler.db"
		}
		return dbx.OpenSQLite(name, dbx.OpenOptions{
			DisableForeignKeyConstraintWhenMigrating: true,
			DefaultMaxLifetime:                       5 * time.Minute,
		})
	default:
		return nil, fmt.Errorf("unsupported timer-scheduler database type: %s", dbCfg.Type)
	}
}

func newOwnerID() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "unknown-host"
	}
	return fmt.Sprintf("%s-%d-%d", host, os.Getpid(), time.Now().UnixNano())
}
