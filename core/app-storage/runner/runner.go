package runner

import (
	"context"
	"fmt"

	// 注意：不在这里导入 docs，避免统一入口时 swagger 重复注册
	// docs 只在独立启动的 main.go 中导入
	"github.com/ai-agent-os/ai-agent-os/core/app-storage/server"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"

	// 导入存储 API 以确保 swag 扫描到（独立启动时）
	_ "github.com/ai-agent-os/ai-agent-os/core/app-storage/api/v1"
)

// Main 服务主函数（支持统一入口调用）
// ctx: 统一的上下文
// stopCh: 停止信号通道，服务应该监听此通道并在收到信号时优雅关闭
// readyCh: 就绪通道，服务启动完成后应该发送信号到此通道（可选，如果为 nil 则忽略）
func Main(ctx context.Context, stopCh <-chan struct{}, readyCh chan<- struct{}) error {
	// 获取配置
	cfg := config.GetAppStorageConfig()

	// 初始化日志系统（如果还未初始化）
	// 注意：统一入口时，日志系统已经在 main.go 中初始化，这里会跳过
	if !logger.IsInitialized() {
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
			return fmt.Errorf("failed to initialize logger: %w", err)
		}

		logger.Infof(ctx, "Logger initialized - Service: app-storage, File: %s", logConfig.Filename)
	} else {
		logger.Infof(ctx, "Logger already initialized (unified entry), skipping initialization")
	}

	// 创建并启动服务器
	srv, err := server.NewServer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	if err := srv.Start(ctx); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	logger.Infof(ctx, "App-storage started successfully")
	
	// ⭐ 发送就绪信号（如果提供了 readyCh）
	// 使用阻塞式发送，确保信号被接收（channel 容量为 1，不会阻塞太久）
	if readyCh != nil {
		readyCh <- struct{}{}
		logger.Infof(ctx, "App-storage 就绪信号已发送")
	}

	// 等待停止信号
	select {
	case <-ctx.Done():
		// 上下文被取消
		logger.Infof(ctx, "Context cancelled, shutting down app-storage...")
	case <-stopCh:
		// 收到停止信号
		logger.Infof(ctx, "Received stop signal, shutting down app-storage...")
	}

	// 优雅关闭服务器
	if err := srv.Stop(ctx); err != nil {
		return fmt.Errorf("error during shutdown: %w", err)
	}

	logger.Infof(ctx, "App-storage stopped")
	return nil
}

