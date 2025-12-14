// @title AI Agent OS API Gateway
// @version 1.0
// @description AI Agent OS API 网关服务文档

// @host localhost:9090
// @BasePath /

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/ai-agent-os/ai-agent-os/core/api-gateway/docs"
	"github.com/ai-agent-os/ai-agent-os/core/api-gateway/runner"
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
		fmt.Printf("Api-gateway error: %v\n", err)
		os.Exit(1)
	}
}

