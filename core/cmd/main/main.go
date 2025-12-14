package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	agentServerRunner "github.com/ai-agent-os/ai-agent-os/core/agent-server/runner"
	appServerRunner "github.com/ai-agent-os/ai-agent-os/core/app-server/runner"
	appRuntimeRunner "github.com/ai-agent-os/ai-agent-os/core/app-runtime/runner"
	appStorageRunner "github.com/ai-agent-os/ai-agent-os/core/app-storage/runner"
	apiGatewayRunner "github.com/ai-agent-os/ai-agent-os/core/api-gateway/runner"
	controlServiceRunner "github.com/ai-agent-os/ai-agent-os/core/control-service/runner"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// ServiceMain 服务主函数类型
// ctx: 统一的上下文
// stopCh: 停止信号通道，服务应该监听此通道并在收到信号时优雅关闭
type ServiceMain func(ctx context.Context, stopCh <-chan struct{}) error

// ServiceInfo 服务信息
type ServiceInfo struct {
	Name string      // 服务名称
	Main ServiceMain  // 服务主函数
}

// 服务列表
var services []ServiceInfo

func init() {
	// 注册要启动的服务（按依赖顺序）
	// 1. Control Service（License 服务，其他服务可能依赖）
	services = append(services, ServiceInfo{
		Name: "control-service",
		Main: controlServiceRunner.Main,
	})

	// 2. App Runtime（应用运行时）
	services = append(services, ServiceInfo{
		Name: "app-runtime",
		Main: appRuntimeRunner.Main,
	})

	// 3. App Storage（存储服务）
	services = append(services, ServiceInfo{
		Name: "app-storage",
		Main: appStorageRunner.Main,
	})

	// 4. Agent Server（Agent 服务）
	services = append(services, ServiceInfo{
		Name: "agent-server",
		Main: agentServerRunner.Main,
	})

	// 5. App Server（应用服务）
	services = append(services, ServiceInfo{
		Name: "app-server",
		Main: appServerRunner.Main,
	})

	// 6. API Gateway（API 网关，最后启动，因为依赖其他服务）
	services = append(services, ServiceInfo{
		Name: "api-gateway",
		Main: apiGatewayRunner.Main,
	})
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("========================================")
	fmt.Println("  AI Agent OS - 统一启动入口")
	fmt.Println("========================================")
	fmt.Println("  ⚠️  重要提示：")
	fmt.Println("  - 此入口仅用于开发环境，提升开发效率")
	fmt.Println("  - 生产环境必须使用独立启动方式")
	fmt.Println("  - 每个服务都可以独立启动（见各服务的 cmd/app/main.go）")
	fmt.Println("  - 生产环境请使用容器编排工具（如 Kubernetes）独立部署")
	fmt.Println("========================================")

	// 初始化统一的日志系统（只初始化一次，所有服务共享）
	// 注意：各个服务仍然可以有自己的日志文件（通过各自的配置）
	logConfig := logger.Config{
		Level:      "info",
		Filename:   "./logs/all-services.log",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   true,
		IsDev:      true,
	}

	if err := logger.Init(logConfig); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Infof(ctx, "统一日志系统初始化完成")

	// 创建停止通道
	stopCh := make(chan struct{})

	// 启动所有服务（在独立的 goroutine 中）
	var wg sync.WaitGroup
	errors := make(chan error, len(services))

	fmt.Println("\n[启动服务]")
	for i, svc := range services {
		fmt.Printf("  %d. %s...\n", i+1, svc.Name)
		wg.Add(1)
		go func(info ServiceInfo) {
			defer wg.Done()
			// 为每个服务创建独立的上下文
			svcCtx := context.WithValue(ctx, "service_name", info.Name)
			if err := info.Main(svcCtx, stopCh); err != nil {
				errors <- fmt.Errorf("%s 运行失败: %w", info.Name, err)
			}
		}(svc)
	}

	// 等待一小段时间，确保服务启动
	time.Sleep(1 * time.Second)

	// 检查是否有启动错误
	select {
	case err := <-errors:
		fmt.Printf("\n[错误] %v\n", err)
		logger.Errorf(ctx, "服务运行失败: %v", err)
		// 通知所有服务停止
		close(stopCh)
		wg.Wait()
		os.Exit(1)
	default:
		// 没有错误，继续
	}

	// 统一等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("\n========================================")
	fmt.Println("  所有服务已启动")
	fmt.Println("  按 Ctrl+C 停止所有服务")
	fmt.Println("========================================")

	<-sigChan

	fmt.Println("\n[停止服务]")
	logger.Infof(ctx, "收到停止信号，正在关闭所有服务...")

	// 通知所有服务停止
	close(stopCh)
	cancel()

	// 等待所有服务关闭
	wg.Wait()

	logger.Infof(ctx, "所有服务已停止")
	fmt.Println("所有服务已停止")
}

