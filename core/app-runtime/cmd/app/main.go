package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ai-agent-os/ai-agent-os/core/app-runtime/runner"
)

func main() {
	ctx := context.Background()

	// 创建停止通道（独立启动时使用信号）
	stopCh := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 在 goroutine 中监听信号
	go func() {
		<-sigChan
		fmt.Println("Received signal, shutting down...")
		close(stopCh)
	}()

	// 调用 Main 函数（独立启动时 readyCh 为 nil）
	if err := runner.Main(ctx, stopCh, nil); err != nil {
		fmt.Printf("App-runtime error: %v\n", err)
		os.Exit(1)
	}
}
