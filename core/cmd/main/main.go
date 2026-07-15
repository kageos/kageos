package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	agentServerRunner "github.com/kageos/kageos/core/agent-server/runner"
	apiGatewayRunner "github.com/kageos/kageos/core/api-gateway/runner"
	appRuntimeRunner "github.com/kageos/kageos/core/app-runtime/runner"
	appServerRunner "github.com/kageos/kageos/core/app-server/runner"
	appStorageRunner "github.com/kageos/kageos/core/app-storage/runner"
	connectorServerRunner "github.com/kageos/kageos/core/connector-server/runner"
	hrServerRunner "github.com/kageos/kageos/core/hr-server/runner"
	messageServerRunner "github.com/kageos/kageos/core/message-server/runner"
	timerSchedulerRunner "github.com/kageos/kageos/core/timer-scheduler/runner"

	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/infra"
	"github.com/kageos/kageos/pkg/license"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/supervisor"
)

// 服务列表
var services []supervisor.Service

func init() {
	// 注册要启动的核心服务（按依赖顺序）
	// 1. App Runtime（应用运行时）
	services = append(services, supervisor.Service{
		Name:      "app-runtime",
		Main:      appRuntimeRunner.Main,
		DependsOn: nil, // 无依赖
	})

	// 2. App Storage（存储服务）
	services = append(services, supervisor.Service{
		Name:      "app-storage",
		Main:      appStorageRunner.Main,
		DependsOn: nil, // 无依赖
	})

	// 3. HR Server（用户管理、组织架构）
	services = append(services, supervisor.Service{
		Name:      "hr-server",
		Main:      hrServerRunner.Main,
		DependsOn: nil, // 无依赖
	})

	// 4. Agent Server（Agent 服务）
	services = append(services, supervisor.Service{
		Name:      "agent-server",
		Main:      agentServerRunner.Main,
		DependsOn: nil,
	})

	// 5. Connector Server（连接器服务）
	services = append(services, supervisor.Service{
		Name:      "connector-server",
		Main:      connectorServerRunner.Main,
		DependsOn: nil,
	})

	// 6. Timer Scheduler（定时调度服务，独立平台服务）
	services = append(services, supervisor.Service{
		Name:      "timer-scheduler",
		Main:      timerSchedulerRunner.Main,
		DependsOn: nil,
	})

	// 7. Message Server（消息服务，消费 NATS 消息并提供站内信）
	services = append(services, supervisor.Service{
		Name:      "message-server",
		Main:      messageServerRunner.Main,
		DependsOn: []string{"hr-server"},
	})

	// 8. App Server（应用服务，依赖 app-runtime）
	services = append(services, supervisor.Service{
		Name:      "app-server",
		Main:      appServerRunner.Main,
		DependsOn: []string{"app-runtime"},
	})

	apiGatewayDependsOn := []string{
		"app-runtime",
		"app-storage",
		"hr-server",
		"agent-server",
		"connector-server",
		"timer-scheduler",
		"message-server",
		"app-server",
	}

	// 9. API Gateway（API 网关，最后启动，因为依赖其他服务）
	services = append(services, supervisor.Service{
		Name:      "api-gateway",
		Main:      apiGatewayRunner.Main,
		DependsOn: apiGatewayDependsOn,
	})
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("========================================")
	fmt.Println("  Kageos - 统一启动入口")
	fmt.Println("========================================")
	fmt.Println("  说明：")
	fmt.Println("  - 运行模式由 .kageos/kageos.env 决定")
	fmt.Println("  - 开发初始化：kagectl init --dev")
	fmt.Println("  - 正式部署：kagectl init && kagectl up")
	fmt.Println("========================================")

	// 初始化统一的日志系统（只初始化一次，所有服务共享）
	// 注意：各个服务仍然可以有自己的日志文件（通过各自的配置）
	logConfig := logger.Config{
		Level:      "info",
		Filename:   "./logs/all-services.log",
		MaxSize:    logger.DefaultMaxSize,
		MaxBackups: logger.DefaultMaxBackups,
		MaxAge:     logger.DefaultMaxAge,
		Compress:   true,
		IsDev:      config.IsDevMode(),
	}

	if err := logger.Init(logConfig); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Infof(ctx, "统一日志系统初始化完成")
	runtimeInfo := license.Snapshot()
	logger.Infof(ctx, "版本授权初始化完成: %s", license.Summary())
	fmt.Printf("  版本授权: edition=%s, license=%s, effective_mode=%s\n",
		runtimeInfo.Edition, runtimeInfo.LicenseStatus, runtimeInfo.EffectiveMode)

	// 启动预检：确保当前模式下的基础设施可达。
	fmt.Println("\n[启动预检]")
	fmt.Println("  检查基础设施连通性...")
	if err := infra.Preflight(ctx); err != nil {
		fmt.Printf("\n  ⚠️  预检警告: %v\n", err)
		if infra.IsMinIOClockSkewError(err) {
			fmt.Println("  检测到 MinIO 时间偏移，继续启动会导致 app-storage 签名失败，已停止。")
			fmt.Println("  修复后重新执行启动命令即可。")
			logger.Errorf(ctx, "启动预检失败: %v", err)
			os.Exit(1)
		}
		fmt.Println("  部分基础设施可能不可用，服务启动后可能出现连接错误")
		fmt.Println("  如需手动修复，请确保 Podman 已启动且基础设施容器存在")
		logger.Warnf(ctx, "启动预检警告: %v", err)
	} else {
		fmt.Println("  ✅ 预检通过，Podman 环境就绪")
	}

	// 创建停止通道
	stopCh := make(chan struct{})

	fmt.Println("\n[启动服务]")
	for i, svc := range services {
		fmt.Printf("  %d. %s", i+1, svc.Name)
		if len(svc.DependsOn) > 0 {
			fmt.Printf("（依赖: %v）", svc.DependsOn)
		}
		fmt.Println("...")
	}

	group := supervisor.Group{
		Services: services,
		Logger:   supervisorLogger{},
	}
	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- group.Run(ctx, stopCh)
	}()

	// 统一等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("\n========================================")
	fmt.Println("  所有服务已启动")
	fmt.Println("  按 Ctrl+C 停止所有服务")
	fmt.Println("========================================")

	select {
	case err := <-runErrCh:
		if err != nil {
			panicServiceError(ctx, err)
		}
		return
	case <-sigChan:
	}

	fmt.Println("\n[停止服务]")
	logger.Infof(ctx, "收到停止信号，正在关闭所有服务...")

	// 通知所有服务停止
	close(stopCh)

	// 等待所有服务关闭
	if err := <-runErrCh; err != nil {
		panicServiceError(ctx, err)
	}

	logger.Infof(ctx, "所有服务已停止")
	fmt.Println("所有服务已停止")
}

type supervisorLogger struct{}

func (supervisorLogger) Infof(ctx context.Context, format string, args ...interface{}) {
	logger.Infof(ctx, format, args...)
}

func (supervisorLogger) Errorf(ctx context.Context, format string, args ...interface{}) {
	logger.Errorf(ctx, format, args...)
}

func panicServiceError(ctx context.Context, err error) {
	message := fmt.Sprintf("服务启动/运行失败，统一启动入口直接退出: %v", err)
	fmt.Fprintf(os.Stderr, "\n[FATAL] %s\n", message)
	logger.Errorf(ctx, "%s", message)
	panic(message)
}
