package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kageos/kageos/core/connector-server/runner"
)

func main() {
	ctx := context.Background()
	stopCh := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("Received signal, shutting down...")
		close(stopCh)
	}()

	if err := runner.Main(ctx, stopCh, nil); err != nil {
		fmt.Printf("connector-server error: %v\n", err)
		os.Exit(1)
	}
}
