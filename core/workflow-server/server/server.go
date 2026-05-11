package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/workflow-server/executor"
	"github.com/ai-agent-os/ai-agent-os/core/workflow-server/model"
	workflowservice "github.com/ai-agent-os/ai-agent-os/core/workflow-server/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/dbx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/serverx"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Server struct {
	cfg        *config.WorkflowServerConfig
	db         *gorm.DB
	service    *workflowservice.Service
	httpServer *gin.Engine
	httpSrv    *http.Server
}

func NewServer(cfg *config.WorkflowServerConfig) (*Server, error) {
	if cfg == nil {
		cfg = &config.WorkflowServerConfig{}
	}
	s := &Server{cfg: cfg}
	if err := s.initDatabase(context.Background()); err != nil {
		return nil, err
	}
	registry := executor.NewRegistry(executor.NewFormSubmitExecutor(nil))
	s.service = workflowservice.NewService(s.db, registry)
	s.httpServer = serverx.NewGin(serverx.WithDebug(cfg.IsDebug()))
	s.setupRoutes()
	return s, nil
}

func (s *Server) Start(ctx context.Context) error {
	addr := net.JoinHostPort(s.cfg.GetListenHost(), strconv.Itoa(s.cfg.GetPort()))
	s.httpSrv = &http.Server{
		Addr:              addr,
		Handler:           s.httpServer,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf(ctx, "[workflow-server] HTTP server error: %v", err)
		}
	}()
	logger.Infof(ctx, "[workflow-server] started on %s", addr)
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
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
	logger.Infof(ctx, "[workflow-server] stopped")
	return nil
}

func (s *Server) initDatabase(ctx context.Context) error {
	dbCfg := s.cfg.GetDB()
	switch dbCfg.Type {
	case "mysql":
		if err := dbx.EnsureMySQLDatabase(dbCfg); err != nil {
			logger.Warnf(ctx, "[workflow-server] ensure database failed: name=%s err=%v", dbCfg.Name, err)
		}
		db, err := dbx.OpenMySQL(dbCfg, dbx.OpenOptions{
			DisableForeignKeyConstraintWhenMigrating: true,
			DefaultMaxLifetime:                       5 * time.Minute,
		})
		if err != nil {
			return fmt.Errorf("failed to connect workflow-server mysql: %w", err)
		}
		if err := model.InitTables(db); err != nil {
			return fmt.Errorf("failed to migrate workflow-server database: %w", err)
		}
		s.db = db
		return nil
	case "sqlite":
		name := dbCfg.Name
		if name == "" {
			name = "data/workflow-server.db"
		}
		db, err := dbx.OpenSQLite(name, dbx.OpenOptions{
			DisableForeignKeyConstraintWhenMigrating: true,
			DefaultMaxLifetime:                       5 * time.Minute,
		})
		if err != nil {
			return err
		}
		if err := model.InitTables(db); err != nil {
			return fmt.Errorf("failed to migrate workflow-server database: %w", err)
		}
		s.db = db
		return nil
	default:
		return fmt.Errorf("unsupported workflow-server database type: %s", dbCfg.Type)
	}
}

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "workflow-server",
	})
}
