// @title AI Agent OS Storage API
// @version 1.0
// @description AI Agent OS 存储服务 API 文档

// @host localhost:9092
// @BasePath /api/v1

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
	"github.com/ai-agent-os/ai-agent-os/core/app-storage/server"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"

	// 导入存储 API 以确保 swag 扫描到
	_ "github.com/ai-agent-os/ai-agent-os/core/app-storage/api/v1"
)

func main() {
	ctx := context.Background()

	// 获取配置
	cfg := config.GetAppStorageConfig()

	// 初始化日志系统
	logConfig := logger.Config{
		Level:      cfg.GetLogLevel(),
		Filename:   "./logs/app-storage.log",
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

	logger.Infof(ctx, "Logger initialized - Service: app-storage, File: %s", logConfig.Filename)

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

	logger.Infof(ctx, "Shutting down app-storage...")

	// 优雅关闭服务器
	if err := srv.Stop(ctx); err != nil {
		logger.Errorf(ctx, "Error during shutdown: %v", err)
	}

	logger.Infof(ctx, "App-storage stopped")
}

