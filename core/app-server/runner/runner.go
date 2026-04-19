package runner

import (
	"context"
	"fmt"

	// 注意：不在这里导入 docs，避免统一入口时 swagger 重复注册
	// docs 只在独立启动的 main.go 中导入
	"github.com/ai-agent-os/ai-agent-os/core/app-server/server"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"

	// 导入认证 API 以确保 swag 扫描到（独立启动时）
	// 注意：统一入口不会使用 swagger，所以这里可以不导入
	// 但为了保持独立启动的兼容性，我们保留 API 导入
	_ "github.com/ai-agent-os/ai-agent-os/core/app-server/api/v1"
)

func ensureLogger(ctx context.Context, level, filename string, isDev bool) error {
	if logger.IsInitialized() {
		logger.Infof(ctx, "Logger already initialized (unified entry), skipping initialization")
		return nil
	}

	logConfig := logger.Config{
		Level:      level,
		Filename:   filename,
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   true,
		IsDev:      isDev,
	}

	if err := logger.Init(logConfig); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	logger.Infof(ctx, "Logger initialized - Service: %s", filename)
	return nil
}

// Main 服务主函数（支持统一入口调用）
// ctx: 统一的上下文
// stopCh: 停止信号通道，服务应该监听此通道并在收到信号时优雅关闭
// readyCh: 就绪通道，服务启动完成后应该发送信号到此通道（可选，如果为 nil 则忽略）
func Main(ctx context.Context, stopCh <-chan struct{}, readyCh chan<- struct{}) error {
	// 获取配置
	cfg := config.GetAppServerConfig()

	if err := ensureLogger(ctx, cfg.GetLogLevel(), "./logs/app-server.log", cfg.IsDebug()); err != nil {
		return err
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 创建并启动服务器
	srv, err := server.NewServer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	if err := srv.Start(runCtx); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	logger.Infof(ctx, "App-server started successfully")

	// ⭐ 发送就绪信号（如果提供了 readyCh）
	// 使用阻塞式发送，确保信号被接收（channel 容量为 1，不会阻塞太久）
	if readyCh != nil {
		readyCh <- struct{}{}
		logger.Infof(ctx, "App-server 就绪信号已发送")
	}

	// 等待停止信号
	select {
	case <-ctx.Done():
		// 上下文被取消
		logger.Infof(ctx, "Context cancelled, shutting down app-server...")
	case <-stopCh:
		// 收到停止信号
		logger.Infof(ctx, "Received stop signal, shutting down app-server...")
	}
	cancel()

	// 优雅关闭服务器
	if err := srv.Stop(ctx); err != nil {
		return fmt.Errorf("error during shutdown: %w", err)
	}

	logger.Infof(ctx, "App-server stopped")
	return nil
}

// SchedulerMain 仅启动定时任务调度 worker。
func SchedulerMain(ctx context.Context, stopCh <-chan struct{}, readyCh chan<- struct{}) error {
	cfg := config.GetAppServerConfig()

	if err := ensureLogger(ctx, cfg.GetLogLevel(), "./logs/app-scheduler.log", cfg.IsDebug()); err != nil {
		return err
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	srv, err := server.NewSchedulerServer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create scheduler server: %w", err)
	}

	if err := srv.StartScheduler(runCtx); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	logger.Infof(ctx, "App scheduler started successfully")
	if readyCh != nil {
		readyCh <- struct{}{}
		logger.Infof(ctx, "App scheduler 就绪信号已发送")
	}

	select {
	case <-ctx.Done():
		logger.Infof(ctx, "Context cancelled, shutting down app scheduler...")
	case <-stopCh:
		logger.Infof(ctx, "Received stop signal, shutting down app scheduler...")
	}
	cancel()

	if err := srv.Stop(ctx); err != nil {
		return fmt.Errorf("error during scheduler shutdown: %w", err)
	}

	logger.Infof(ctx, "App scheduler stopped")
	return nil
}
