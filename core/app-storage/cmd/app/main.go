// @title AI Agent OS Storage API
// @version 1.0
// @description AI Agent OS 存储服务 API 文档

// @host localhost:9092
// @BasePath

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-Token

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/ai-agent-os/ai-agent-os/core/app-storage/docs"
	"github.com/ai-agent-os/ai-agent-os/core/app-storage/runner"
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

	// 调用 Main 函数
	if err := runner.Main(ctx, stopCh); err != nil {
		fmt.Printf("App-storage error: %v\n", err)
		os.Exit(1)
	}
}

