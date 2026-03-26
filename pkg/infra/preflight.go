package infra

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// InfraContainers 需要预检的基础设施容器
var InfraContainers = []containerCheck{
	{Name: "mysql8", Label: "MySQL"},
	{Name: "nats-server", Label: "NATS"},
	{Name: "minio", Label: "MinIO"},
}

type containerCheck struct {
	Name  string
	Label string
}

type containerResult struct {
	check   containerCheck
	started bool
	already bool
	err     error
	elapsed time.Duration
}

// Preflight 启动预检：确保 Podman 环境和基础设施容器就绪。
// 在所有服务启动之前调用，适用于开发环境统一入口。
//
// 设计原则：尽可能快地放行，不做多余等待。
// 服务端已有自动重连（NATS MaxReconnects=-1），预检只管"把东西拉起来"。
func Preflight(ctx context.Context) error {
	start := time.Now()
	logger.Infof(ctx, "[Preflight] ========== 启动预检开始 | 平台=%s ==========", runtime.GOOS)

	// 客户主站 Compose：MySQL/NATS/MinIO 由 compose 起在兄弟容器，非本机 podman 名 mysql8 等；entrypoint 已 wait_tcp。
	if os.Getenv("AI_AGENT_OS_SKIP_INFRA_PREFLIGHT") == "1" {
		logger.Infof(ctx, "[Preflight] 跳过 Embedding 基础设施预检 (AI_AGENT_OS_SKIP_INFRA_PREFLIGHT=1)，仅探测 MySQL TCP")
		waitForMySQLTCP(ctx, "mysql:3306", 30*time.Second)
		logger.Infof(ctx, "[Preflight] ========== 启动预检完成 | 总耗时=%s ==========", time.Since(start).Round(time.Millisecond))
		return nil
	}

	// Step 1: Podman Machine（macOS/Windows 需要虚拟机）
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		if err := ensurePodmanMachine(ctx); err != nil {
			return fmt.Errorf("Podman Machine 预检失败: %w", err)
		}
	} else {
		logger.Infof(ctx, "[Preflight] [1/2] 跳过 Podman Machine 检查 (Linux 原生运行)")
	}

	// Step 2: 并行启动所有基础设施容器
	ensureInfraContainers(ctx)

	// Step 3: 等 MySQL 真正就绪（容器 started ≠ 服务 ready，MySQL 内部初始化需要几秒）
	// NATS/MinIO 服务端有自动重连不需要等，MySQL 连不上服务会直接退出
	waitForMySQL(ctx)

	logger.Infof(ctx, "[Preflight] ========== 启动预检完成 | 总耗时=%s ==========", time.Since(start).Round(time.Millisecond))
	return nil
}

// ensurePodmanMachine 确保 Podman Machine 运行中
func ensurePodmanMachine(ctx context.Context) error {
	stepStart := time.Now()

	running, err := isPodmanMachineRunning(ctx)
	if err != nil {
		logger.Warnf(ctx, "[Preflight] [1/2] Podman Machine 状态检查失败: %v，尝试启动...", err)
		running = false
	}

	if running {
		logger.Infof(ctx, "[Preflight] [1/2] ✅ Podman Machine 已在运行 | 耗时=%s", time.Since(stepStart).Round(time.Millisecond))
		return nil
	}

	logger.Infof(ctx, "[Preflight] [1/2] Podman Machine 未运行，正在启动（首次启动需要 15~30 秒）...")
	startTime := time.Now()

	cmd := exec.CommandContext(ctx, "podman", "machine", "start")
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		if strings.Contains(outputStr, "already running") || strings.Contains(outputStr, "is already running") {
			logger.Infof(ctx, "[Preflight] [1/2] ✅ Podman Machine 已在运行")
			return nil
		}
		return fmt.Errorf("podman machine start 失败: %w, 输出: %s", err, outputStr)
	}

	logger.Infof(ctx, "[Preflight] [1/2] ✅ Podman Machine 已启动 | 耗时=%s", time.Since(startTime).Round(time.Millisecond))
	return nil
}

// ensureInfraContainers 并行启动所有基础设施容器
func ensureInfraContainers(ctx context.Context) {
	stepStart := time.Now()
	results := make([]containerResult, len(InfraContainers))
	var wg sync.WaitGroup

	for i, c := range InfraContainers {
		wg.Add(1)
		go func(idx int, check containerCheck) {
			defer wg.Done()
			results[idx] = startOneContainer(ctx, check)
		}(i, c)
	}

	wg.Wait()

	alreadyRunning, started, failed := 0, 0, 0
	for _, r := range results {
		switch {
		case r.already:
			alreadyRunning++
			logger.Infof(ctx, "[Preflight] [2/2] ✅ %s (%s) 已在运行", r.check.Label, r.check.Name)
		case r.started:
			started++
			logger.Infof(ctx, "[Preflight] [2/2] ✅ %s 已启动 | 耗时=%s", r.check.Label, r.elapsed.Round(time.Millisecond))
		case r.err != nil:
			failed++
			logger.Warnf(ctx, "[Preflight] [2/2] ❌ %s 启动失败: %v（容器可能不存在，需手动创建）", r.check.Label, r.err)
		}
	}

	logger.Infof(ctx, "[Preflight] [2/2] 基础设施容器检查完成 | 已运行=%d | 新启动=%d | 失败=%d | 耗时=%s",
		alreadyRunning, started, failed, time.Since(stepStart).Round(time.Millisecond))
}

// startOneContainer 检查并启动单个容器（供并行调用）
func startOneContainer(ctx context.Context, c containerCheck) containerResult {
	running, _ := isContainerRunning(ctx, c.Name)
	if running {
		return containerResult{check: c, already: true}
	}

	t := time.Now()
	cmd := exec.CommandContext(ctx, "podman", "start", c.Name)
	output, err := cmd.CombinedOutput()
	elapsed := time.Since(t)

	if err != nil {
		return containerResult{check: c, err: fmt.Errorf("%v | 输出: %s", err, strings.TrimSpace(string(output))), elapsed: elapsed}
	}
	return containerResult{check: c, started: true, elapsed: elapsed}
}

// isPodmanMachineRunning 检查 Podman Machine 是否运行中
func isPodmanMachineRunning(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "podman", "machine", "list", "--format", "{{.Running}}")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.Contains(strings.TrimSpace(string(output)), "true"), nil
}

// isContainerRunning 检查指定容器是否运行中
func isContainerRunning(ctx context.Context, name string) (bool, error) {
	cmd := exec.CommandContext(ctx, "podman", "inspect", "--format", "{{.State.Running}}", name)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(output)) == "true", nil
}

// waitForMySQL 等待 MySQL 真正可用
// TCP 端口通 ≠ MySQL ready（端口开了但协议层可能还没初始化，连接会 EOF）
// 使用 mysqladmin ping 做应用层健康检查，这是 MySQL 官方推荐的就绪检测方式
func waitForMySQL(ctx context.Context) {
	const (
		maxWait  = 30 * time.Second
		interval = 1 * time.Second
	)

	start := time.Now()
	deadline := start.Add(maxWait)

	logger.Infof(ctx, "[Preflight] [3/3] 等待 MySQL 就绪（mysqladmin ping，最多 %s）...", maxWait)

	for time.Now().Before(deadline) {
		cmd := exec.CommandContext(ctx, "podman", "exec", "mysql8", "mysqladmin", "ping", "-h", "127.0.0.1", "--silent")
		if err := cmd.Run(); err == nil {
			logger.Infof(ctx, "[Preflight] [3/3] ✅ MySQL 已就绪（ping ok）| 等待=%s", time.Since(start).Round(time.Millisecond))
			return
		}
		time.Sleep(interval)
	}

	logger.Warnf(ctx, "[Preflight] [3/3] ⚠ MySQL 在 %s 内未通过 ping 检查，服务启动可能出现连接错误", maxWait)
}

// waitForMySQLTCP 用于客户 Compose 等场景：无 podman exec mysql8，仅确认 TCP 可连。
func waitForMySQLTCP(ctx context.Context, addr string, maxWait time.Duration) {
	deadline := time.Now().Add(maxWait)
	logger.Infof(ctx, "[Preflight] [skip-mode] 等待 MySQL TCP %s（最多 %s）...", addr, maxWait)
	for time.Now().Before(deadline) {
		d := net.Dialer{Timeout: 2 * time.Second}
		c, err := d.DialContext(ctx, "tcp", addr)
		if err == nil {
			_ = c.Close()
			logger.Infof(ctx, "[Preflight] [skip-mode] ✅ MySQL TCP 已连通: %s", addr)
			return
		}
		time.Sleep(time.Second)
	}
	logger.Warnf(ctx, "[Preflight] [skip-mode] ⚠ MySQL TCP %s 在 %s 内未连通", addr, maxWait)
}
