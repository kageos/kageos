package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/kageos/kageos/core/timer-scheduler/runner"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stopCh := make(chan struct{})
	readyCh := make(chan struct{}, 1)
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		close(stopCh)
		cancel()
	}()
	if err := runner.Main(ctx, stopCh, readyCh); err != nil {
		panic(err)
	}
}
