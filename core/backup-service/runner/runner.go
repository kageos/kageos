package runner

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/ai-agent-os/ai-agent-os/core/backup-service/server"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

func Main(ctx context.Context, stopCh <-chan struct{}, readyCh chan<- struct{}) error {
	cfg := config.GetBackupServiceConfig()
	logFile := "./logs/backup-service.log"
	if cfg.Storage.LogsPath != "" {
		logFile = filepath.Join(cfg.Storage.LogsPath, "backup-service.log")
	}

	if !logger.IsInitialized() {
		logConfig := logger.Config{
			Level:      cfg.GetLogLevel(),
			Filename:   logFile,
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   true,
			IsDev:      cfg.IsDebug(),
		}
		if err := logger.Init(logConfig); err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}
	}

	srv, err := server.NewServer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create backup server: %w", err)
	}

	if err := srv.Start(ctx); err != nil {
		return fmt.Errorf("failed to start backup server: %w", err)
	}

	if readyCh != nil {
		readyCh <- struct{}{}
	}

	select {
	case <-ctx.Done():
	case <-stopCh:
	}

	if err := srv.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop backup server: %w", err)
	}

	return nil
}
