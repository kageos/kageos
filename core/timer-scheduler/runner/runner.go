package runner

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/timer-scheduler/server"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

func ensureLogger(ctx context.Context, level string, isDev bool) error {
	if logger.IsInitialized() {
		return nil
	}
	return logger.Init(logger.Config{
		Level:      level,
		Filename:   "./logs/timer-scheduler.log",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   true,
		IsDev:      isDev,
	})
}

func Main(ctx context.Context, stopCh <-chan struct{}, readyCh chan<- struct{}) error {
	cfg := config.GetTimerSchedulerConfig()
	if err := ensureLogger(ctx, cfg.GetLogLevel(), cfg.IsDebug()); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	srv, err := server.NewServer(cfg)
	if err != nil {
		return err
	}
	if err := srv.Start(runCtx); err != nil {
		return err
	}
	if readyCh != nil {
		readyCh <- struct{}{}
	}
	select {
	case <-ctx.Done():
	case <-stopCh:
	}
	cancel()
	return srv.Stop(context.Background())
}
