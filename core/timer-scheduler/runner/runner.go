package runner

import (
	"context"
	"fmt"

	"github.com/kageos/kageos/core/timer-scheduler/server"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/logger"
)

func ensureLogger(ctx context.Context, level string, isDev bool) error {
	if logger.IsInitialized() {
		return nil
	}
	return logger.Init(logger.Config{
		Level:      level,
		Filename:   "./logs/timer-scheduler.log",
		MaxSize:    logger.DefaultMaxSize,
		MaxBackups: logger.DefaultMaxBackups,
		MaxAge:     logger.DefaultMaxAge,
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
