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
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/server"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

func main() {
	ctx := context.Background()

	// 获取配置
	cfg := config.GetAgentServerConfig()

	// 初始化日志系统
	logConfig := logger.Config{
		Level:      cfg.GetLogLevel(),
		Filename:   "./logs/agent-server.log",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   true,
		IsDev:      cfg.IsDebug(),
	}

	if err := logger.Init(logConfig); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Infof(ctx, "Logger initialized - Service: agent-server, File: %s", logConfig.Filename)

	// 创建并启动服务器
	srv, err := server.NewServer(cfg)
	if err != nil {
		logger.Errorf(ctx, "Failed to create server: %v", err)
		os.Exit(1)
	}

	if err := srv.Start(ctx); err != nil {
		logger.Errorf(ctx, "Failed to start server: %v", err)
		os.Exit(1)
	}

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Infof(ctx, "Shutting down agent-server...")

	// 优雅关闭服务器
	if err := srv.Stop(ctx); err != nil {
		logger.Errorf(ctx, "Error during shutdown: %v", err)
	}

	logger.Infof(ctx, "Agent-server stopped")
}
