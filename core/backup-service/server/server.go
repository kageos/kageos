package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	backupservice "github.com/ai-agent-os/ai-agent-os/core/backup-service/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/serverx"
	"github.com/gin-gonic/gin"
)

type Server struct {
	cfg          *config.BackupServiceConfig
	controlPlane *backupservice.ControlPlane
	httpServer   *gin.Engine
	httpSrv      *http.Server
}

func NewServer(cfg *config.BackupServiceConfig) (*Server, error) {
	s := &Server{
		cfg: cfg,
	}

	if err := s.preparePaths(); err != nil {
		return nil, err
	}

	controlPlane, err := backupservice.NewControlPlane(cfg)
	if err != nil {
		return nil, fmt.Errorf("create backup control plane: %w", err)
	}
	s.controlPlane = controlPlane

	s.httpServer = serverx.NewGin(
		serverx.WithDebug(cfg.IsDebug()),
		serverx.WithRecovery(),
		serverx.WithLogger(),
	)
	s.setupRoutes()
	s.httpSrv = &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.GetPort()),
		Handler:           s.httpServer,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return s, nil
}

func (s *Server) Start(ctx context.Context) error {
	logger.Infof(ctx, "[backup-service] starting on :%d", s.cfg.GetPort())

	go func() {
		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf(ctx, "[backup-service] http server error: %v", err)
		}
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := s.httpSrv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	return s.controlPlane.Close()
}

func (s *Server) preparePaths() error {
	dirs := []string{
		s.cfg.Storage.NamespacePath,
		s.cfg.Storage.DataPath,
		s.cfg.Storage.LogsPath,
		s.cfg.Storage.MySQLPath,
		s.cfg.Storage.MinIOPath,
		s.cfg.Storage.PodmanStoragePath,
		s.cfg.Repository.RootPath,
		s.cfg.Repository.StatePath,
		s.cfg.Repository.StagingPath,
	}
	for _, dir := range dirs {
		if dir == "" {
			return fmt.Errorf("backup-service repository path is empty")
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}
