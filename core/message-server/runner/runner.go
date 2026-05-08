package runner

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/message-server/server"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

func Main(ctx context.Context, stopCh <-chan struct{}, readyCh chan<- struct{}) error {
	cfg := config.GetMessageServerConfig()
	if !logger.IsInitialized() {
		if err := logger.Init(logger.Config{
			Level:      cfg.GetLogLevel(),
			Filename:   "./logs/message-server.log",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     7,
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
