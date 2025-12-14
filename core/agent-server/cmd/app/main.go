// @title Agent-Server API
// @version 1.0
// @description Agent-Server 代码生成服务 API 文档
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:9095
// @BasePath

// @securityDefinitions.basic BasicAuth

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-Token

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/ai-agent-os/ai-agent-os/core/agent-server/docs"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/runner"
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
		fmt.Printf("Agent-server error: %v\n", err)
		os.Exit(1)
	}
}
