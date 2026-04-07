package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	backupRunner "github.com/ai-agent-os/ai-agent-os/core/backup-service/runner"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	stopCh := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(stopCh)
	}()

	if err := backupRunner.Main(ctx, stopCh, nil); err != nil {
		fmt.Fprintf(os.Stderr, "backup-service exited with error: %v\n", err)
		os.Exit(1)
	}
}
