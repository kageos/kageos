package service

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/nats-io/nats.go"
)

// InfraWatchdog 基础设施看门狗
//
// 核心思路：以 NATS 连接状态作为"基础设施是否存活"的探针。
// NATS 跑在 Podman 容器里，NATS 断了 = Podman 大概率挂了。
// 检查 natsConn.IsConnected() 就是读内存布尔值，零开销。
//
// 工作流程：
//  1. 每秒检查一次 NATS 连接状态（极低开销）
//  2. 正常 → 什么都不做
//  3. 断开 → 触发恢复链：Podman Machine → 基础设施容器 → 等待 NATS 自动重连
//  4. 恢复完成后进入冷却期（15 秒），等 NATS 自动重连，不反复触发
type InfraWatchdog struct {
	natsConn         *nats.Conn
	containerService ContainerOperator
	checkInterval    time.Duration
	cooldownPeriod   time.Duration // 恢复后冷却时间，等 NATS 重连
	infraContainers  []string

	// 基础设施恢复后的回调（用于对账应用容器等）
	onRecovered func(ctx context.Context)

	recovering   bool
	recoveringMu sync.Mutex

	lastRecoveryEnd time.Time // 上次恢复结束时间，用于冷却判断
	lastRecoveryMu  sync.Mutex

	totalRecoveries atomic.Int64
}

// NewInfraWatchdog 创建看门狗
func NewInfraWatchdog(natsConn *nats.Conn, containerService ContainerOperator) *InfraWatchdog {
	return &InfraWatchdog{
		natsConn:         natsConn,
		containerService: containerService,
		checkInterval:    1 * time.Second,
		cooldownPeriod:   15 * time.Second,
		infraContainers:  []string{"mysql8", "nats-server", "minio"},
	}
}

// SetOnRecovered 设置基础设施恢复后的回调（用于对账应用容器）
func (w *InfraWatchdog) SetOnRecovered(fn func(ctx context.Context)) {
	w.onRecovered = fn
}

// Start 启动看门狗（阻塞，应在 goroutine 中调用）
func (w *InfraWatchdog) Start(ctx context.Context) {
	logger.Infof(ctx, "[InfraWatchdog] 启动 | 探针=NATS | 轮询=%s | 冷却=%s | 保活容器=%v | 平台=%s",
		w.checkInterval, w.cooldownPeriod, w.infraContainers, runtime.GOOS)

	ticker := time.NewTicker(w.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Infof(ctx, "[InfraWatchdog] 已停止 | 累计恢复次数=%d", w.totalRecoveries.Load())
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *InfraWatchdog) tick(ctx context.Context) {
	if w.natsConn.IsConnected() {
		return
	}

	// 冷却期内不重复触发：恢复完成后等 NATS 自动重连
	w.lastRecoveryMu.Lock()
	if !w.lastRecoveryEnd.IsZero() && time.Since(w.lastRecoveryEnd) < w.cooldownPeriod {
		w.lastRecoveryMu.Unlock()
		return
	}
	w.lastRecoveryMu.Unlock()

	// 防止并发进入恢复
	w.recoveringMu.Lock()
	if w.recovering {
		w.recoveringMu.Unlock()
		return
	}
	w.recovering = true
	w.recoveringMu.Unlock()

	defer func() {
		w.recoveringMu.Lock()
		w.recovering = false
		w.recoveringMu.Unlock()

		// 标记恢复结束时间，开始冷却
		w.lastRecoveryMu.Lock()
		w.lastRecoveryEnd = time.Now()
		w.lastRecoveryMu.Unlock()
	}()

	recoveryStart := time.Now()
	natsStatus := w.natsConn.Status()
	w.totalRecoveries.Add(1)
	recoveryCount := w.totalRecoveries.Load()

	logger.Warnf(ctx, "[InfraWatchdog] ⚡ 检测到 NATS 断开 | NATS状态=%s | 第%d次恢复 | 开始恢复链...",
		natsStatusString(natsStatus), recoveryCount)

	// Step 1: macOS/Windows 检查并恢复 Podman Machine
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		stepStart := time.Now()
		machineRunning := w.isPodmanMachineRunning(ctx)
		logger.Infof(ctx, "[InfraWatchdog] [Step 1/4] 检查 Podman Machine | 运行中=%v | 耗时=%s",
			machineRunning, time.Since(stepStart).Round(time.Millisecond))

		if !machineRunning {
			stepStart = time.Now()
			logger.Infof(ctx, "[InfraWatchdog] [Step 1/4] Podman Machine 未运行，正在启动...")
			if err := w.startPodmanMachine(ctx); err != nil {
				logger.Errorf(ctx, "[InfraWatchdog] [Step 1/4] ❌ Podman Machine 启动失败 | 耗时=%s | 错误=%v",
					time.Since(stepStart).Round(time.Millisecond), err)
				logger.Errorf(ctx, "[InfraWatchdog] 恢复中止 | 总耗时=%s | Machine 起不来，%s 后重试",
					time.Since(recoveryStart).Round(time.Millisecond), w.cooldownPeriod)
				return
			}
			logger.Infof(ctx, "[InfraWatchdog] [Step 1/4] ✅ Podman Machine 已启动 | 耗时=%s",
				time.Since(stepStart).Round(time.Millisecond))
		}
	} else {
		logger.Infof(ctx, "[InfraWatchdog] [Step 1/4] 跳过 Podman Machine 检查 (Linux 无需虚拟机)")
	}

	// Step 2: 确保 Podman API 连接可用
	stepStart := time.Now()
	podmanRunning := w.containerService.IsRunning()
	logger.Infof(ctx, "[InfraWatchdog] [Step 2/4] 检查 Podman API | 连接正常=%v | 耗时=%s",
		podmanRunning, time.Since(stepStart).Round(time.Millisecond))

	if !podmanRunning {
		stepStart = time.Now()
		logger.Infof(ctx, "[InfraWatchdog] [Step 2/4] Podman API 不可用，正在重连...")
		if err := w.containerService.Start(ctx); err != nil {
			logger.Errorf(ctx, "[InfraWatchdog] [Step 2/4] ❌ Podman API 重连失败 | 耗时=%s | 错误=%v",
				time.Since(stepStart).Round(time.Millisecond), err)
			logger.Errorf(ctx, "[InfraWatchdog] 恢复中止 | 总耗时=%s | %s 后重试",
				time.Since(recoveryStart).Round(time.Millisecond), w.cooldownPeriod)
			return
		}
		logger.Infof(ctx, "[InfraWatchdog] [Step 2/4] ✅ Podman API 已重连 | 耗时=%s",
			time.Since(stepStart).Round(time.Millisecond))
	}

	// Step 3: 恢复基础设施容器
	stepStart = time.Now()
	startedCount := 0
	alreadyRunning := 0
	failedCount := 0

	for _, name := range w.infraContainers {
		running, err := w.containerService.IsContainerRunning(ctx, name)
		if err != nil {
			logger.Warnf(ctx, "[InfraWatchdog] [Step 3/4] 无法检查容器 %s 状态: %v", name, err)
			failedCount++
			continue
		}
		if running {
			alreadyRunning++
			continue
		}

		containerStart := time.Now()
		logger.Infof(ctx, "[InfraWatchdog] [Step 3/4] 启动容器 %s ...", name)
		if err := w.containerService.StartContainer(ctx, name); err != nil {
			logger.Warnf(ctx, "[InfraWatchdog] [Step 3/4] ❌ 容器 %s 启动失败 | 耗时=%s | 错误=%v",
				name, time.Since(containerStart).Round(time.Millisecond), err)
			failedCount++
		} else {
			logger.Infof(ctx, "[InfraWatchdog] [Step 3/4] ✅ 容器 %s 已启动 | 耗时=%s",
				name, time.Since(containerStart).Round(time.Millisecond))
			startedCount++
		}
	}

	logger.Infof(ctx, "[InfraWatchdog] [Step 3/4] 基础设施容器恢复完成 | 已启动=%d | 已在运行=%d | 失败=%d | 耗时=%s",
		startedCount, alreadyRunning, failedCount, time.Since(stepStart).Round(time.Millisecond))

	// Step 4: 恢复应用容器（内存中标记为 running 但实际已停止的）
	if w.onRecovered != nil {
		stepStart = time.Now()
		logger.Infof(ctx, "[InfraWatchdog] [Step 4/5] 开始对账应用容器...")
		w.onRecovered(ctx)
		logger.Infof(ctx, "[InfraWatchdog] [Step 4/5] 应用容器对账完成 | 耗时=%s",
			time.Since(stepStart).Round(time.Millisecond))
	}

	// Step 5: 进入冷却，等待 NATS 自动重连
	totalDuration := time.Since(recoveryStart).Round(time.Millisecond)

	logger.Infof(ctx, "[InfraWatchdog] [Step 5/5] 恢复完成 | 耗时=%s | 第%d次恢复 | 进入冷却 %s，等待 NATS 自动重连",
		totalDuration, recoveryCount, w.cooldownPeriod)
}

// isPodmanMachineRunning 检查 Podman Machine 是否在运行
func (w *InfraWatchdog) isPodmanMachineRunning(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "podman", "machine", "list", "--format", "{{.Running}}")
	output, err := cmd.Output()
	if err != nil {
		logger.Debugf(ctx, "[InfraWatchdog] podman machine list 执行失败: %v", err)
		return false
	}
	return strings.Contains(strings.TrimSpace(string(output)), "true")
}

// startPodmanMachine 启动 Podman Machine
func (w *InfraWatchdog) startPodmanMachine(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "podman", "machine", "start")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("podman machine start failed: %w, output: %s", err, string(output))
	}
	time.Sleep(5 * time.Second)
	return nil
}

// natsStatusString 将 NATS 连接状态转为可读字符串
func natsStatusString(status nats.Status) string {
	switch status {
	case nats.CONNECTED:
		return "CONNECTED"
	case nats.DISCONNECTED:
		return "DISCONNECTED"
	case nats.RECONNECTING:
		return "RECONNECTING"
	case nats.CLOSED:
		return "CLOSED"
	case nats.DRAINING_SUBS:
		return "DRAINING_SUBS"
	case nats.DRAINING_PUBS:
		return "DRAINING_PUBS"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", status)
	}
}
