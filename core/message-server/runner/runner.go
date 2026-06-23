package runner

import (
	"context"
	"fmt"

	"github.com/kageos/kageos/core/message-server/server"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/logger"
)

func Main(ctx context.Context, stopCh <-chan struct{}, readyCh chan<- struct{}) error {
	cfg := config.GetMessageServerConfig()
	if !logger.IsInitialized() {
		if err := logger.Init(logger.Config{
			Level:      cfg.GetLogLevel(),
			Filename:   "./logs/message-server.log",
			MaxSize:    logger.DefaultMaxSize,
			MaxBackups: logger.DefaultMaxBackups,
			MaxAge:     logger.DefaultMaxAge,
			Compress:   true,
			IsDev:      cfg.IsDebug(),
		}); err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}
	}

	srv, err := server.NewServer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create message-server: %w", err)
	}
	if err := srv.Start(ctx); err != nil {
		return fmt.Errorf("failed to start message-server: %w", err)
	}
	if readyCh != nil {
		readyCh <- struct{}{}
	}

	select {
	case <-ctx.Done():
	case <-stopCh:
	}
	return srv.Stop(context.Background())
}
