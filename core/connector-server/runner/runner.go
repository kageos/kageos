package runner

import (
	"context"
	"fmt"

	"github.com/kageos/kageos/core/connector-server/server"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/logger"
)

func Main(ctx context.Context, stopCh <-chan struct{}, readyCh chan<- struct{}) error {
	cfg := config.GetConnectorServerConfig()
	if !logger.IsInitialized() {
		logConfig := logger.Config{
			Level:      cfg.GetLogLevel(),
			Filename:   "./logs/connector-server.log",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   true,
			IsDev:      cfg.IsDebug(),
		}
		if err := logger.Init(logConfig); err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}
	} else {
		logger.Infof(ctx, "Logger already initialized (unified entry), skipping initialization")
	}

	srv, err := server.NewServer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create connector-server: %w", err)
	}
	if err := srv.Start(ctx); err != nil {
		return fmt.Errorf("failed to start connector-server: %w", err)
	}
	logger.Infof(ctx, "connector-server started successfully")

	if readyCh != nil {
		readyCh <- struct{}{}
		logger.Infof(ctx, "connector-server 就绪信号已发送")
	}

	select {
	case <-ctx.Done():
		logger.Infof(ctx, "Context cancelled, shutting down connector-server...")
	case <-stopCh:
		logger.Infof(ctx, "Received stop signal, shutting down connector-server...")
	}
	if err := srv.Stop(ctx); err != nil {
		return fmt.Errorf("error during shutdown: %w", err)
	}
	return nil
}
