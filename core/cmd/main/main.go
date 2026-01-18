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
	apiGatewayRunner "github.com/ai-agent-os/ai-agent-os/core/api-gateway/runner"
	appRuntimeRunner "github.com/ai-agent-os/ai-agent-os/core/app-runtime/runner"
	appServerRunner "github.com/ai-agent-os/ai-agent-os/core/app-server/runner"
	appStorageRunner "github.com/ai-agent-os/ai-agent-os/core/app-storage/runner"
	controlServiceRunner "github.com/ai-agent-os/ai-agent-os/core/control-service/runner"
	hrServerRunner "github.com/ai-agent-os/ai-agent-os/core/hr-server/runner"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// ServiceMain 服务主函数类型
// ctx: 统一的上下文
// stopCh: 停止信号通道，服务应该监听此通道并在收到信号时优雅关闭
// readyCh: 就绪通道，服务启动完成后应该发送信号到此通道（可选，如果为 nil 则忽略）
type ServiceMain func(ctx context.Context, stopCh <-chan struct{}, readyCh chan<- struct{}) error

// ServiceInfo 服务信息
type ServiceInfo struct {
	Name         string        // 服务名称
	Main         ServiceMain   // 服务主函数
	DependsOn    []string      // 依赖的服务名称列表（这些服务启动完成后才能启动此服务）
	ReadyChannel chan struct{} // 就绪通道（服务启动完成后会发送信号）
}

// 服务列表
var services []*ServiceInfo

func init() {
	// 注册要启动的服务（按依赖顺序）
	// 1. Control Service（License 服务，其他服务可能依赖）
	services = append(services, &ServiceInfo{
		Name:         "control-service",
		Main:         controlServiceRunner.Main,
		DependsOn:    nil, // 无依赖
		ReadyChannel: make(chan struct{}, 1),
	})

	// 2. App Runtime（应用运行时）
	services = append(services, &ServiceInfo{
		Name:         "app-runtime",
		Main:         appRuntimeRunner.Main,
		DependsOn:    nil, // 无依赖
		ReadyChannel: make(chan struct{}, 1),
	})

	// 3. App Storage（存储服务）
	services = append(services, &ServiceInfo{
		Name:         "app-storage",
		Main:         appStorageRunner.Main,
		DependsOn:    nil, // 无依赖
		ReadyChannel: make(chan struct{}, 1),
	})

	// 4. Agent Server（Agent 服务）
	services = append(services, &ServiceInfo{
		Name:         "agent-server",
		Main:         agentServerRunner.Main,
		DependsOn:    nil, // 无依赖
		ReadyChannel: make(chan struct{}, 1),
	})

	// 5. HR Server（HR 服务，用户管理、组织架构）
	services = append(services, &ServiceInfo{
		Name:         "hr-server",
		Main:         hrServerRunner.Main,
		DependsOn:    nil, // 无依赖
		ReadyChannel: make(chan struct{}, 1),
	})

	// 6. App Server（应用服务，依赖 app-runtime）
	services = append(services, &ServiceInfo{
		Name:         "app-server",
		Main:         appServerRunner.Main,
		DependsOn:    []string{"app-runtime"}, // ⭐ 依赖 app-runtime
		ReadyChannel: make(chan struct{}, 1),
	})

	// 7. API Gateway（API 网关，最后启动，因为依赖其他服务）
	services = append(services, &ServiceInfo{
		Name:         "api-gateway",
		Main:         apiGatewayRunner.Main,
		DependsOn:    nil, // 无依赖
		ReadyChannel: make(chan struct{}, 1),
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

	// 启动所有服务
	var wg sync.WaitGroup
	errors := make(chan error, len(services))

	// 创建服务就绪映射（服务名称 -> 就绪通道）
	serviceReadyMap := make(map[string]chan struct{})
	for i := range services {
		serviceReadyMap[services[i].Name] = services[i].ReadyChannel
	}

	fmt.Println("\n[启动服务]")
	wg.Add(len(services))
	// 启动所有服务（根据依赖关系）
	for i, svc := range services {
		fmt.Printf("  %d. %s", i+1, svc.Name)
		if len(svc.DependsOn) > 0 {
			fmt.Printf("（依赖: %v）", svc.DependsOn)
		}
		fmt.Println("...")

		go func(info *ServiceInfo, index int) {
			defer wg.Done()

			// ⭐ 等待依赖的服务启动完成
			if len(info.DependsOn) > 0 {
				logger.Infof(ctx, "[启动] %s 等待依赖服务: %v", info.Name, info.DependsOn)
				for _, depName := range info.DependsOn {
					depReadyCh, exists := serviceReadyMap[depName]
					if !exists {
						errors <- fmt.Errorf("%s 依赖的服务 %s 不存在", info.Name, depName)
						return
					}
					// ⭐ 直接监听依赖服务的 ReadyChannel
					logger.Infof(ctx, "[启动] %s 等待依赖 %s 就绪...", info.Name, depName)
					select {
					case <-depReadyCh:
						logger.Infof(ctx, "[启动] %s 的依赖 %s 已就绪", info.Name, depName)
					case <-time.After(30 * time.Second):
						logger.Errorf(ctx, "[启动] %s 等待依赖 %s 超时", info.Name, depName)
						errors <- fmt.Errorf("%s 等待依赖 %s 启动超时（30秒）", info.Name, depName)
						return
					case <-ctx.Done():
						return
					}
				}
				logger.Infof(ctx, "[启动] %s 所有依赖已就绪，开始启动...", info.Name)
			}

			// ⭐ 直接启动服务（异步），不需要等待启动完成
			// 依赖的服务会监听 ReadyChannel，服务启动完成后会发送信号
			svcCtx := context.WithValue(ctx, "service_name", info.Name)
			go func() {
				logger.Infof(ctx, "[启动] %s 开始执行 Main 函数...", info.Name)
				// 运行 Main 函数（会一直运行，直到收到停止信号）
				// Main 函数内部会调用 srv.Start(ctx)，如果成功返回，应该通过 readyCh 发送就绪信号
				if err := info.Main(svcCtx, stopCh, info.ReadyChannel); err != nil {
					logger.Errorf(ctx, "[启动] %s Main 函数返回错误: %v", info.Name, err)
					errors <- fmt.Errorf("%s 运行失败: %w", info.Name, err)
				}
			}()
		}(svc, i)
	}

	// 不再需要等待，服务启动完成后会通过 ReadyChannel 通知

	// ⭐ 在后台 goroutine 中持续监听错误
	go func() {
		for err := range errors {
			fmt.Printf("\n[错误] %v\n", err)
			logger.Errorf(ctx, "服务运行失败: %v", err)
			// 通知所有服务停止
			close(stopCh)
			wg.Wait()
			os.Exit(1)
		}
	}()

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
