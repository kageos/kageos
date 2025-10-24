package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ai-agent-os/ai-agent-os/core/app-runtime/server"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

func main() {
	ctx := context.Background()

	// 获取配置
	cfg := config.GetAppRuntimeConfig()

	// 初始化日志系统
	logConfig := logger.Config{
		Level:      cfg.Runtime.LogLevel,
		Filename:   "./logs/app-runtime.log",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   true,
		IsDev:      cfg.Runtime.Debug,
	}

	if err := logger.Init(logConfig); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Infof(ctx, "Logger initialized - Service: app-runtime, File: %s", logConfig.Filename)

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

	logger.Infof(ctx, "Shutting down app runtime service...")

	// 优雅关闭服务器
	if err := srv.Stop(ctx); err != nil {
		logger.Errorf(ctx, "Error during shutdown: %v", err)
	}

	logger.Infof(ctx, "App runtime service stopped")
}
