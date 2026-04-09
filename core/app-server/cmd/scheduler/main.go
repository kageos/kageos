package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/runner"
)

func main() {
	ctx := context.Background()

	stopCh := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("Received signal, shutting down scheduler...")
		close(stopCh)
	}()

	if err := runner.SchedulerMain(ctx, stopCh, nil); err != nil {
		fmt.Printf("App-scheduler error: %v\n", err)
		os.Exit(1)
	}
}
