package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ai-agent-os/ai-agent-os/core/message-server/runner"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stopCh := make(chan struct{})
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		close(stopCh)
		cancel()
	}()

	if err := runner.Main(ctx, stopCh, nil); err != nil {
		fmt.Printf("message-server error: %v\n", err)
		os.Exit(1)
	}
}
