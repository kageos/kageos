package runner

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/hr-server/server"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"

	// 导入认证 API 以确保 swag 扫描到（独立启动时）
	_ "github.com/ai-agent-os/ai-agent-os/core/hr-server/api/v1"
)

// Main 服务主函数（支持统一入口调用）
// ctx: 统一的上下文
// stopCh: 停止信号通道，服务应该监听此通道并在收到信号时优雅关闭
func Main(ctx context.Context, stopCh <-chan struct{}) error {
	// 获取配置
	cfg := config.GetHRServerConfig()

	// 初始化日志系统（如果还未初始化）
	if !logger.IsInitialized() {
		logConfig := logger.Config{
			Level:      cfg.GetLogLevel(),
			Filename:   "./logs/hr-server.log",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   true,
			IsDev:      cfg.IsDebug(),
		}

		if err := logger.Init(logConfig); err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}

		logger.Infof(ctx, "Logger initialized - Service: hr-server, File: %s", logConfig.Filename)
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

	logger.Infof(ctx, "HR-server started successfully")

	// 等待停止信号
	select {
	case <-ctx.Done():
		// 上下文被取消
		logger.Infof(ctx, "Context cancelled, shutting down hr-server...")
	case <-stopCh:
		// 收到停止信号
		logger.Infof(ctx, "Received stop signal, shutting down hr-server...")
	}

	// 优雅关闭服务器
	if err := srv.Stop(ctx); err != nil {
		return fmt.Errorf("error during shutdown: %w", err)
	}

	logger.Infof(ctx, "HR-server stopped")
	return nil
}

