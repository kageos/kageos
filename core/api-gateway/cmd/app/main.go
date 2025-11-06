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
	"github.com/ai-agent-os/ai-agent-os/core/api-gateway/server"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"

	// 导入网关 API 以确保 swag 扫描到
	_ "github.com/ai-agent-os/ai-agent-os/core/api-gateway/api/v1"
)

func main() {
	ctx := context.Background()

	// 获取配置
	cfg := config.GetAPIGatewayConfig()

	// 初始化日志系统
	logConfig := logger.Config{
		Level:      cfg.GetLogLevel(),
		Filename:   "./logs/api-gateway.log",
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

	logger.Infof(ctx, "Logger initialized - Service: api-gateway, File: %s", logConfig.Filename)

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

	logger.Infof(ctx, "Shutting down api-gateway...")

	// 优雅关闭服务器
	if err := srv.Stop(ctx); err != nil {
		logger.Errorf(ctx, "Error during shutdown: %v", err)
	}

	logger.Infof(ctx, "Api-gateway stopped")
}

